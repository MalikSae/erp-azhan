package media

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

var validCategoryRegex = regexp.MustCompile(`^[a-zA-Z0-9\-]+$`)

// Handler struct
type Handler struct{}

// NewHandler creates a new media handler
func NewHandler() *Handler {
	return &Handler{}
}

// UploadMedia godoc
// POST /api/admin/media/upload
func (h *Handler) UploadMedia(w http.ResponseWriter, r *http.Request) {
	category := r.FormValue("category")
	if strings.TrimSpace(category) == "" {
		writeError(w, http.StatusBadRequest, "category wajib diisi")
		return
	}
	if !validCategoryRegex.MatchString(category) {
		writeError(w, http.StatusBadRequest, "category tidak valid")
		return
	}

	h.processUpload(w, r, category)
}

// UploadPortalMedia godoc
// POST /api/portal/media/upload
// Category is strictly forced to "dokumen-jamaah" regardless of request payload.
func (h *Handler) UploadPortalMedia(w http.ResponseWriter, r *http.Request) {
	h.processUpload(w, r, "dokumen-jamaah")
}

func (h *Handler) processUpload(w http.ResponseWriter, r *http.Request, category string) {
	// 1. Limit max size to 25MB
	r.Body = http.MaxBytesReader(w, r.Body, 25<<20)

	if err := r.ParseMultipartForm(25 << 20); err != nil {
		writeError(w, http.StatusRequestEntityTooLarge, "ukuran file terlalu besar (maksimal 25MB)")
		return
	}

	// 2. Get the file
	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "file tidak ditemukan dalam request")
		return
	}
	defer file.Close()

	// Read first 512 bytes for magic bytes to detect content type
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		writeError(w, http.StatusInternalServerError, "gagal membaca file")
		return
	}

	contentType := http.DetectContentType(buffer[:n])
	extensions := map[string]string{
		"image/jpeg": ".jpg", "image/png": ".png", "image/webp": ".webp", "image/gif": ".gif",
	}
	ext, isImage := extensions[contentType]
	if contentType == "application/pdf" && category == "dokumen-jamaah" {
		ext = ".pdf"
	}
	if ext == "" {
		writeError(w, http.StatusBadRequest, "file harus berupa JPG, PNG, WEBP, GIF, atau PDF")
		return
	}

	// Rewind the file
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		writeError(w, http.StatusInternalServerError, "gagal memproses file")
		return
	}

	// 3. Generate unique name
	newFileName := uuid.New().String() + ext
	uploadDir := filepath.Join(".", "uploads", category)

	// 4. Create folder if not exists
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		fmt.Printf("Gagal membuat direktori upload: %v\n", err)
		writeError(w, http.StatusInternalServerError, "gagal memproses gambar")
		return
	}

	finalPath := filepath.Join(uploadDir, newFileName)
	destination, err := os.OpenFile(finalPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal menyimpan file")
		return
	}
	if _, err := io.Copy(destination, file); err != nil {
		destination.Close()
		_ = os.Remove(finalPath)
		writeError(w, http.StatusInternalServerError, "gagal menyimpan file")
		return
	}
	if err := destination.Close(); err != nil {
		_ = os.Remove(finalPath)
		writeError(w, http.StatusInternalServerError, "gagal menyimpan file")
		return
	}

	publicURL := fmt.Sprintf("/uploads/%s/%s", category, newFileName)
	response := map[string]string{"url": publicURL}

	// Format asli dipertahankan agar upload tidak bergantung pada binary eksternal.
	_ = isImage

	// 8. Success
	writeJSON(w, http.StatusCreated, response)
}

// ─── Shared helpers ───────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
