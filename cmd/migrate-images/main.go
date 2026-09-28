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
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/ddmas26/inventory/internal/config"
	"github.com/ddmas26/inventory/internal/database"
	"github.com/ddmas26/inventory/internal/storage"
)

// legacyPrefix is the reference prefix used by images stored on local disk.
const legacyPrefix = "/uploads/"

func main() {
	dryRun := flag.Bool("dry-run", false, "report what would be migrated without changing anything")
	flag.Parse()

	cfg := config.Load()

	if !cfg.Storage.Enabled() {
		log.Fatal("S3_BUCKET is not set — configure the object store before migrating")
	}

	ctx := context.Background()

	db, err := database.Connect(cfg.DB)
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	store, err := storage.New(ctx, cfg.Storage)
	if err != nil {
		log.Fatalf("Storage initialisation failed: %v", err)
	}

	repo := database.NewRepository(db)

	images, err := repo.ListImagesByRefPrefix(legacyPrefix)
	if err != nil {
		log.Fatalf("Failed to list local images: %v", err)
	}

	log.Printf("Found %d image(s) stored on local disk (upload dir %q)", len(images), cfg.UploadDir)
	if *dryRun {
		log.Println("Dry run — no changes will be made")
	}

	var migrated, skipped, failed int

	for _, img := range images {
		filename := strings.TrimPrefix(img.URL, legacyPrefix)
		if filename == "" || strings.ContainsAny(filename, "/\\") {
			log.Printf("  skip %s: unexpected local reference", img.URL)
			skipped++
			continue
		}

		srcPath := filepath.Join(cfg.UploadDir, filename)
		file, err := os.Open(srcPath)
		if err != nil {
			log.Printf("  skip %s: %v", img.URL, err)
			skipped++
			continue
		}

		contentType, size := detectContentType(file)
		key := store.NewKey(filepath.Ext(filename))

		if *dryRun {
			file.Close()
			log.Printf("  would upload %s -> %s (%s, %d bytes)", srcPath, key, contentType, size)
			migrated++
			continue
		}

		err = store.Upload(ctx, key, file, contentType, size)
		file.Close()
		if err != nil {
			log.Printf("  FAILED %s: %v", srcPath, err)
			failed++
			continue
		}

		if err := repo.UpdateImageRef(img.ID, key); err != nil {
			log.Printf("  FAILED to update %s: %v", img.URL, err)
			failed++
			continue
		}

		log.Printf("  %s -> %s", img.URL, key)
		migrated++
	}

	log.Printf("Done: %d migrated, %d skipped, %d failed", migrated, skipped, failed)
	if failed > 0 {
		os.Exit(1)
	}
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
