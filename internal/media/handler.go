package media

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
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
	if !strings.HasPrefix(contentType, "image/") {
		writeError(w, http.StatusBadRequest, "file harus berupa gambar")
		return
	}

	// Rewind the file
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		writeError(w, http.StatusInternalServerError, "gagal memproses file")
		return
	}

	// 3. Generate unique name
	newFileName := uuid.New().String() + ".webp"
	uploadDir := filepath.Join(".", "uploads", category)

	// 4. Create folder if not exists
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		fmt.Printf("Gagal membuat direktori upload: %v\n", err)
		writeError(w, http.StatusInternalServerError, "gagal memproses gambar")
		return
	}

	// 5. Save original file to a temp file
	tempFile, err := os.CreateTemp("", "upload-*.tmp")
	if err != nil {
		fmt.Printf("Gagal membuat temp file: %v\n", err)
		writeError(w, http.StatusInternalServerError, "gagal memproses gambar")
		return
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)

	if _, err := io.Copy(tempFile, file); err != nil {
		tempFile.Close()
		fmt.Printf("Gagal menulis ke temp file: %v\n", err)
		writeError(w, http.StatusInternalServerError, "gagal memproses gambar")
		return
	}
	tempFile.Close()

	finalPath := filepath.Join(uploadDir, newFileName)

	maxWidth := r.FormValue("max_width")
	if strings.TrimSpace(maxWidth) == "" {
		maxWidth = "512"
	}

	generateThumbnail := r.FormValue("generate_thumbnail") == "true"

	// 6. Execute cwebp
	cmd := exec.Command("cwebp", "-q", "80", "-resize", maxWidth, "0", tempPath, "-o", finalPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("cwebp gagal: %v\nOutput: %s\n", err, string(output))
		writeError(w, http.StatusInternalServerError, "gagal memproses gambar")
		return
	}

	publicURL := fmt.Sprintf("/uploads/%s/%s", category, newFileName)
	response := map[string]string{"url": publicURL}

	// 7. Generate thumbnail if requested
	if generateThumbnail {
		thumbFileName := strings.TrimSuffix(newFileName, ".webp") + "-thumb.webp"
		thumbFinalPath := filepath.Join(uploadDir, thumbFileName)
		
		thumbCmd := exec.Command("cwebp", "-q", "75", "-resize", "300", "0", tempPath, "-o", thumbFinalPath)
		thumbOutput, thumbErr := thumbCmd.CombinedOutput()
		if thumbErr != nil {
			fmt.Printf("cwebp thumb gagal: %v\nOutput: %s\n", thumbErr, string(thumbOutput))
			// Non-fatal error, return main image only
		} else {
			response["thumb_url"] = fmt.Sprintf("/uploads/%s/%s", category, thumbFileName)
		}
	}

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
