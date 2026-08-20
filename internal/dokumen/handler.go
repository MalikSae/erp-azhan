package dokumen

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"erp-azhan/api/internal/identity"
	"github.com/go-chi/chi/v5"
)

// Handler menyimpan dependency untuk semua HTTP handler dokumen.
type Handler struct {
	repo *Repository
}

// NewHandler membuat Handler baru.
func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

var validJenis = map[string]bool{
	"pas_foto":          true,
	"paspor":            true,
	"ktp":               true,
	"kk":                true,
	"buku_nikah":        true,
	"akte_lahir":        true,
	"vaksin_meningitis": true,
}

var validStatuses = map[string]bool{
	"approved": true,
	"rejected": true,
}

// ─── List ─────────────────────────────────────────────────────────────────────

// ListDokumen godoc
// GET /api/admin/jamaah/{jamaah_id}/dokumen
func (h *Handler) ListDokumen(w http.ResponseWriter, r *http.Request) {
	jamaahID, ok := parseID(w, r, "jamaah_id")
	if !ok {
		return
	}

	brandID := identity.GetBrandID(r.Context())

	// Verify jamaah exists and belongs to brand
	exists, err := h.repo.JamaahExistsForBrand(r.Context(), jamaahID, brandID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan internal")
		return
	}
	if !exists {
		writeError(w, http.StatusNotFound, "jamaah_id tidak ditemukan atau bukan milik brand Anda")
		return
	}

	items, err := h.repo.ListByJamaahID(r.Context(), jamaahID, brandID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal mengambil data dokumen")
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// ─── Upsert ───────────────────────────────────────────────────────────────────

// UpsertDokumen godoc
// POST /api/admin/jamaah/{jamaah_id}/dokumen
func (h *Handler) UpsertDokumen(w http.ResponseWriter, r *http.Request) {
	jamaahID, ok := parseID(w, r, "jamaah_id")
	if !ok {
		return
	}

	var req CreateDokumenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}

	if !validJenis[req.Jenis] {
		writeError(w, http.StatusBadRequest, "jenis dokumen tidak valid")
		return
	}

	req.FileURL = strings.TrimSpace(req.FileURL)
	if req.FileURL == "" {
		writeError(w, http.StatusBadRequest, "file_url wajib diisi")
		return
	}

	brandID := identity.GetBrandID(r.Context())

	// Verify jamaah
	exists, err := h.repo.JamaahExistsForBrand(r.Context(), jamaahID, brandID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan internal")
		return
	}
	if !exists {
		writeError(w, http.StatusNotFound, "jamaah_id tidak valid atau bukan milik brand Anda")
		return
	}

	d, err := h.repo.Upsert(r.Context(), jamaahID, &req)
	if err != nil {
		handleRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, d) // 200 OK karena bisa jadi update/replace
}

// ─── Update Status ────────────────────────────────────────────────────────────

// UpdateDokumenStatus godoc
// PUT /api/admin/dokumen/{id}/status
func (h *Handler) UpdateDokumenStatus(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}

	var req UpdateDokumenStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}

	if !validStatuses[req.Status] {
		writeError(w, http.StatusBadRequest, "status tidak valid, harus approved/rejected")
		return
	}

	brandID := identity.GetBrandID(r.Context())

	// Ambil dokumen untuk verifikasi brand
	_, err := h.repo.GetByID(r.Context(), id, brandID)
	if err != nil {
		handleRepoError(w, err)
		return
	}

	updated, err := h.repo.UpdateStatus(r.Context(), id, req.Status)
	if err != nil {
		handleRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

// ─── Error handling ───────────────────────────────────────────────────────────

func handleRepoError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, "data tidak ditemukan")
	default:
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("terjadi kesalahan internal: %v", err))
	}
}

// ─── Shared helpers ───────────────────────────────────────────────────────────

func parseID(w http.ResponseWriter, r *http.Request, param string) (int64, bool) {
	raw := chi.URLParam(r, param)
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, param+" tidak valid")
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
