package addon

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-sql-driver/mysql"
)

// Handler menyimpan dependency untuk semua HTTP handler addon.
type Handler struct {
	repo *Repository
}

// NewHandler membuat Handler baru dengan repository yang diberikan.
func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

// ─── List ─────────────────────────────────────────────────────────────────────

// ListAddOns godoc
// GET /api/admin/addons
// Response 200: array AddOn ([] jika kosong)
func (h *Handler) ListAddOns(w http.ResponseWriter, r *http.Request) {
	addons, err := h.repo.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal mengambil data add-on")
		return
	}
	writeJSON(w, http.StatusOK, addons)
}

// ─── Create ───────────────────────────────────────────────────────────────────

// CreateAddOn godoc
// POST /api/admin/addons
// Response 201: AddOn baru
func (h *Handler) CreateAddOn(w http.ResponseWriter, r *http.Request) {
	var req CreateAddOnRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}

	name, ok := validateAddOnInput(w, req.Name)
	if !ok {
		return
	}

	addon, err := h.repo.Create(r.Context(), name)
	if err != nil {
		handleAddOnRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, addon)
}

// ─── Update ───────────────────────────────────────────────────────────────────

// UpdateAddOn godoc
// PUT /api/admin/addons/{id}
// Response 200: AddOn yang sudah diupdate
func (h *Handler) UpdateAddOn(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	var req UpdateAddOnRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}

	name, ok := validateAddOnInput(w, req.Name)
	if !ok {
		return
	}

	addon, err := h.repo.Update(r.Context(), id, name)
	if err != nil {
		handleAddOnRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, addon)
}

// ─── Delete ───────────────────────────────────────────────────────────────────

// DeleteAddOn godoc
// DELETE /api/admin/addons/{id}
// Response 200: {"message": "berhasil dihapus"}
func (h *Handler) DeleteAddOn(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		handleAddOnRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "berhasil dihapus"})
}

// ─── Validation ───────────────────────────────────────────────────────────────

func validateAddOnInput(w http.ResponseWriter, rawName string) (string, bool) {
	// Gate 1: name wajib
	if strings.TrimSpace(rawName) == "" {
		writeError(w, http.StatusBadRequest, "name wajib diisi")
		return "", false
	}

	// Gate 2: normalisasi
	name := strings.ToUpper(strings.TrimSpace(rawName))

	return name, true
}

// ─── Error handling ───────────────────────────────────────────────────────────

func handleAddOnRepoError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, "data tidak ditemukan")
	case errors.Is(err, ErrDuplicate):
		writeError(w, http.StatusConflict, "add-on dengan nama ini sudah ada")
	default:
		// Tangkap MySQL FK constraint violation (error 1451)
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1451 {
			writeError(w, http.StatusConflict, "tidak bisa dihapus, masih dipakai oleh paket lain")
			return
		}
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan internal")
	}
}

// ─── Shared helpers ───────────────────────────────────────────────────────────

func parseID(w http.ResponseWriter, r *http.Request) (uint64, bool) {
	raw := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || id == 0 {
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
