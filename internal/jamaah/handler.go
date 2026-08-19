package jamaah

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-sql-driver/mysql"
	"erp-azhan/api/internal/identity"
)

// Handler menyimpan dependency untuk semua HTTP handler jamaah.
type Handler struct {
	repo *Repository
}

// NewHandler membuat Handler baru.
func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

// ─── List ─────────────────────────────────────────────────────────────────────

// ListJamaah godoc
// GET /api/admin/jamaah
func (h *Handler) ListJamaah(w http.ResponseWriter, r *http.Request) {
	brandID := identity.GetBrandID(r.Context())
	items, err := h.repo.List(r.Context(), brandID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal mengambil data jamaah")
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// ─── Get Detail ───────────────────────────────────────────────────────────────

// GetJamaah godoc
// GET /api/admin/jamaah/{id}
func (h *Handler) GetJamaah(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	brandID := identity.GetBrandID(r.Context())
	j, err := h.repo.GetByID(r.Context(), id, brandID)
	if err != nil {
		handleRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, j)
}

// ─── Create ───────────────────────────────────────────────────────────────────

// CreateJamaah godoc
// POST /api/admin/jamaah
func (h *Handler) CreateJamaah(w http.ResponseWriter, r *http.Request) {
	var req CreateJamaahRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}

	// Validasi nama_lengkap wajib
	if strings.TrimSpace(req.NamaLengkap) == "" {
		writeError(w, http.StatusBadRequest, "nama_lengkap wajib diisi")
		return
	}
	req.NamaLengkap = strings.TrimSpace(req.NamaLengkap)

	// Brand resolution — pola sama dengan schedule
	ctxBrandID := identity.GetBrandID(r.Context())
	var finalBrandID int64
	if ctxBrandID != nil {
		finalBrandID = *ctxBrandID
	} else {
		if req.BrandID == nil || *req.BrandID == 0 {
			writeError(w, http.StatusBadRequest, "brand_id wajib diisi oleh Super Admin")
			return
		}
		finalBrandID = *req.BrandID
	}

	j, err := h.repo.Create(r.Context(), finalBrandID, &req)
	if err != nil {
		handleRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, j)
}

// ─── Update ───────────────────────────────────────────────────────────────────

// UpdateJamaah godoc
// PUT /api/admin/jamaah/{id}
func (h *Handler) UpdateJamaah(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	var req UpdateJamaahRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}

	if strings.TrimSpace(req.NamaLengkap) == "" {
		writeError(w, http.StatusBadRequest, "nama_lengkap wajib diisi")
		return
	}
	req.NamaLengkap = strings.TrimSpace(req.NamaLengkap)

	ctxBrandID := identity.GetBrandID(r.Context())

	// Ambil existing untuk tentukan finalBrandID
	existing, err := h.repo.GetByID(r.Context(), id, ctxBrandID)
	if err != nil {
		handleRepoError(w, err)
		return
	}

	var finalBrandID int64
	if ctxBrandID != nil {
		finalBrandID = existing.BrandID
	} else {
		if req.BrandID != nil && *req.BrandID != 0 {
			finalBrandID = *req.BrandID
		} else {
			finalBrandID = existing.BrandID
		}
	}

	j, err := h.repo.Update(r.Context(), id, ctxBrandID, finalBrandID, &req)
	if err != nil {
		handleRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, j)
}

// ─── Delete ───────────────────────────────────────────────────────────────────

// DeleteJamaah godoc
// DELETE /api/admin/jamaah/{id}
func (h *Handler) DeleteJamaah(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	brandID := identity.GetBrandID(r.Context())
	if err := h.repo.Delete(r.Context(), id, brandID); err != nil {
		handleRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "berhasil dihapus"})
}

// ─── Error handling ───────────────────────────────────────────────────────────

func handleRepoError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, "data tidak ditemukan")
	case errors.Is(err, ErrDuplicateNIK):
		writeError(w, http.StatusConflict, "NIK sudah terdaftar")
	case errors.Is(err, ErrKodeBrandNotSet):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1451 {
			writeError(w, http.StatusConflict, "tidak bisa dihapus, masih dipakai oleh data booking")
			return
		}
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("terjadi kesalahan internal: %v", err))
	}
}

// ─── Shared helpers ───────────────────────────────────────────────────────────

func parseID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	raw := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(raw, 10, 64)
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
