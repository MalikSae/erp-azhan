package destination

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-sql-driver/mysql"
)

// Handler menyimpan dependency untuk semua HTTP handler destinasi.
type Handler struct {
	repo *Repository
}

// NewHandler membuat Handler baru dengan repository yang diberikan.
func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

// ListDestinations: GET /api/admin/destinations. Response 200: array Destination ([] jika kosong).
func (h *Handler) ListDestinations(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.List(r.Context(), false)
	if err != nil {
		log.Printf("[ERROR] destination: list: %v", err)
		writeError(w, http.StatusInternalServerError, "gagal mengambil data destinasi")
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// CreateDestination: POST /api/admin/destinations. Response 201: Destination baru.
func (h *Handler) CreateDestination(w http.ResponseWriter, r *http.Request) {
	name, slug, active, ok := decodeSave(w, r)
	if !ok {
		return
	}
	d, err := h.repo.Create(r.Context(), name, slug, active)
	if err != nil {
		handleRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, d)
}

// UpdateDestination: PUT /api/admin/destinations/{id}. Response 200: Destination terupdate.
func (h *Handler) UpdateDestination(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	name, slug, active, ok := decodeSave(w, r)
	if !ok {
		return
	}
	d, err := h.repo.Update(r.Context(), id, name, slug, active)
	if err != nil {
		handleRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, d)
}

// DeleteDestination: DELETE /api/admin/destinations/{id}. Response 200: pesan berhasil.
func (h *Handler) DeleteDestination(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := h.repo.Delete(r.Context(), id); err != nil {
		handleRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "berhasil dihapus"})
}

// ─── Validation ───────────────────────────────────────────────────────────────

func decodeSave(w http.ResponseWriter, r *http.Request) (name, slug string, active bool, ok bool) {
	var req SaveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return "", "", false, false
	}
	name = strings.TrimSpace(req.Name)
	if name == "" {
		writeError(w, http.StatusBadRequest, "nama destinasi wajib diisi")
		return "", "", false, false
	}
	if len([]rune(name)) > 100 {
		writeError(w, http.StatusBadRequest, "nama destinasi maksimal 100 karakter")
		return "", "", false, false
	}
	slug = strings.ToLower(strings.TrimSpace(req.Slug))
	if !ValidSlug(slug) {
		writeError(w, http.StatusBadRequest, "slug destinasi hanya huruf kecil, angka, dan tanda hubung (mis. abu-dhabi)")
		return "", "", false, false
	}
	active = true
	if req.IsActive != nil {
		active = *req.IsActive
	}
	return name, slug, active, true
}

// ─── Error handling ───────────────────────────────────────────────────────────

func handleRepoError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, "data tidak ditemukan")
	case errors.Is(err, ErrDuplicateSlug):
		writeError(w, http.StatusConflict, "Slug destinasi sudah digunakan")
	case errors.Is(err, ErrInUse):
		writeError(w, http.StatusConflict, err.Error())
	default:
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) {
			switch mysqlErr.Number {
			case 1062:
				writeError(w, http.StatusConflict, "Slug destinasi sudah digunakan")
				return
			case 1451:
				writeError(w, http.StatusConflict, "tidak bisa dihapus, masih dipakai oleh data lain")
				return
			}
		}
		log.Printf("[ERROR] destination: %v", err)
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan internal")
	}
}

// ─── Shared helpers ───────────────────────────────────────────────────────────

func parseID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "id tidak valid")
		return 0, false
	}
	return id, true
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
