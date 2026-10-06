package handler

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/ddmas26/inventory/internal/storage"
)

// maxUploadSize is the maximum accepted image size (5 MB).
const maxUploadSize = 5 << 20

// allowedImageTypes maps a sniffed MIME type to the file extension we store it as.
var allowedImageTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/gif":  ".gif",
	"image/webp": ".webp",
}

// UploadImage handles POST /api/uploads/image.
// It accepts a multipart/form-data body with a single "file" field containing an
// image and returns the URL the image is served from. Images go to the configured
// S3/MinIO bucket, or to the local upload directory when no bucket is set.
func (h *Handler) UploadImage(w http.ResponseWriter, r *http.Request) {
	if !h.requireAuth(w, r) {
		return
	}

	// Cap the request body to avoid memory exhaustion.
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)

	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		respondError(w, http.StatusBadRequest, "invalid upload or file exceeds 5MB limit")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		respondError(w, http.StatusBadRequest, "missing 'file' field")
		return
	}
	defer file.Close()

	// Sniff the real content type instead of trusting the client.
	head := make([]byte, 512)
	n, err := file.Read(head)
	if err != nil && err != io.EOF {
		respondError(w, http.StatusInternalServerError, "failed to read uploaded file")
		return
	}

	contentType := http.DetectContentType(head[:n])

	ext, ok := allowedImageTypes[contentType]
	if !ok {
		respondError(w, http.StatusBadRequest, "unsupported image type (use JPEG, PNG, GIF or WEBP)")
		return
	}

	// Rewind so the full file content is read below.
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to process uploaded file")
		return
	}

	if h.Storage.Enabled() {
		key := h.Storage.NewKey(ext)

		if err := h.Storage.Upload(context.Background(), key, file, contentType, header.Size); err != nil {
			log.Printf("uploads: store image failed (key=%q, size=%d): %v", key, header.Size, err)
			respondError(w, http.StatusInternalServerError, "failed to store image")
			return
		}

		respond(w, http.StatusCreated, map[string]string{
			// This URL is what the client displays and sends back; it is
			// canonicalised to the object key when the product is saved.
			"url":      h.Storage.URL(key),
			"key":      key,
			"filename": header.Filename,
		})
		return
	}

	// No bucket configured — fall back to the local upload directory.
	if err := os.MkdirAll(h.UploadDir, 0o755); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to prepare upload directory")
		return
	}

	filename := storage.RandomHex(16) + ext
	dstPath := filepath.Join(h.UploadDir, filename)

	dst, err := os.Create(dstPath)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to save uploaded file")
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		_ = os.Remove(dstPath)
		respondError(w, http.StatusInternalServerError, "failed to save uploaded file")
		return
	}

	respond(w, http.StatusCreated, map[string]string{
		"url":      "/uploads/" + filename,
		"filename": header.Filename,
	})
}
