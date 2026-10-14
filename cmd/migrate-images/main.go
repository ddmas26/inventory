// Command migrate-images copies product images that still live on the local disk
// into the configured S3/MinIO bucket and rewrites their database references to
// object keys.
//
// Safe to run repeatedly: rows that already point at an object key (or an
// external URL) are left untouched.
//
// Usage:
//
//	go run ./cmd/migrate-images            # migrate
//	go run ./cmd/migrate-images -dry-run   # report only, change nothing
package main

import (
	"context"
	"flag"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/ddmas26/inventory/internal/config"
	"github.com/ddmas26/inventory/internal/database"
	"github.com/ddmas26/inventory/internal/logging"
	"github.com/ddmas26/inventory/internal/storage"
)

// legacyPrefix is the reference prefix used by images stored on local disk.
const legacyPrefix = "/uploads/"

func main() {
	dryRun := flag.Bool("dry-run", false, "report what would be migrated without changing anything")
	flag.Parse()

	logging.Init()
	cfg := config.Load()

	if !cfg.Storage.Enabled() {
		fatalMsg("S3_BUCKET is not set — configure the object store before migrating")
	}

	ctx := context.Background()

	db, err := database.Connect(cfg.DB)
	if err != nil {
		fatal("database connection failed", err)
	}

	store, err := storage.New(ctx, cfg.Storage)
	if err != nil {
		fatal("storage initialisation failed", err)
	}

	repo := database.NewRepository(db)

	images, err := repo.ListImagesByRefPrefix(legacyPrefix)
	if err != nil {
		fatal("failed to list local images", err)
	}

	slog.Info("found local images", "count", len(images), "upload_dir", cfg.UploadDir)
	if *dryRun {
		slog.Info("dry run, no changes will be made")
	}

	var migrated, skipped, failed int

	for _, img := range images {
		filename := strings.TrimPrefix(img.URL, legacyPrefix)
		if filename == "" || strings.ContainsAny(filename, "/\\") {
			slog.Warn("skipping image: unexpected local reference", "url", img.URL)
			skipped++
			continue
		}

		srcPath := filepath.Join(cfg.UploadDir, filename)
		file, err := os.Open(srcPath)
		if err != nil {
			slog.Warn("skipping image", "url", img.URL, "error", err)
			skipped++
			continue
		}

		contentType, size := detectContentType(file)
		key := store.NewKey(filepath.Ext(filename))

		if *dryRun {
			file.Close()
			slog.Info("would upload", "src", srcPath, "key", key, "content_type", contentType, "bytes", size)
			migrated++
			continue
		}

		err = store.Upload(ctx, key, file, contentType, size)
		file.Close()
		if err != nil {
			slog.Error("upload failed", "src", srcPath, "error", err)
			failed++
			continue
		}

		if err := repo.UpdateImageRef(img.ID, key); err != nil {
			slog.Error("update image ref failed", "url", img.URL, "error", err)
			failed++
			continue
		}

		slog.Info("migrated image", "from", img.URL, "to", key)
		migrated++
	}

	slog.Info("migration complete", "migrated", migrated, "skipped", skipped, "failed", failed)
	if failed > 0 {
		os.Exit(1)
	}
}

// fatal logs a structured error and exits.
func fatal(msg string, err error) {
	slog.Error(msg, "error", err)
	os.Exit(1)
}

// fatalMsg logs a structured error message and exits.
func fatalMsg(msg string) {
	slog.Error(msg)
	os.Exit(1)
}

// detectContentType sniffs an image's MIME type from its contents (falling back
// to the file extension) and reports the file size, rewinding the reader.
func detectContentType(file *os.File) (string, int64) {
	head := make([]byte, 512)
	n, err := file.Read(head)
	if err != nil && err != io.EOF {
		return "application/octet-stream", 0
	}

	_, _ = file.Seek(0, io.SeekStart)

	contentType := http.DetectContentType(head[:n])
	if contentType == "application/octet-stream" {
		if byExt := mime.TypeByExtension(filepath.Ext(file.Name())); byExt != "" {
			contentType = byExt
		}
	}

	info, err := file.Stat()
	size := int64(0)
	if err == nil {
		size = info.Size()
	}

	return contentType, size
}
