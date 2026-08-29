package category

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-sql-driver/mysql"
	"erp-azhan/api/internal/identity"
)

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	s = reg.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

func parseID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	raw := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "ID tidak valid")
		return 0, false
	}
	return id, true
}

func handleRepoError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, "kategori tidak ditemukan")
	case errors.Is(err, ErrSlugDuplicate):
		writeError(w, http.StatusConflict, "slug kategori sudah digunakan")
	default:
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1451 {
			writeError(w, http.StatusConflict, "tidak bisa dihapus, masih dipakai oleh paket umroh")
			return
		}
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("terjadi kesalahan internal: %v", err))
	}
}

// ─── Admin Handlers ──────────────────────────────────────────────────────────

// ListCategories GET /api/admin/categories
func (h *Handler) ListCategories(w http.ResponseWriter, r *http.Request) {
	var brandID *int64
	ctxBrandID := identity.GetBrandID(r.Context())
	if ctxBrandID != nil {
		brandID = ctxBrandID
	} else if bParam := r.URL.Query().Get("brand_id"); bParam != "" {
		if bid, err := strconv.ParseInt(bParam, 10, 64); err == nil && bid > 0 {
			brandID = &bid
		}
	}

	activeOnly := r.URL.Query().Get("active_only") == "true"

	items, err := h.repo.List(r.Context(), brandID, activeOnly)
	if err != nil {
		handleRepoError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, items)
}

// GetCategory GET /api/admin/categories/{id}
func (h *Handler) GetCategory(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	cat, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		handleRepoError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, cat)
}

// CreateCategory POST /api/admin/categories
func (h *Handler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	var req CreateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "nama kategori wajib diisi")
		return
	}

	if strings.TrimSpace(req.Slug) == "" {
		req.Slug = slugify(req.Name)
	} else {
		req.Slug = slugify(req.Slug)
	}

	if exists, err := h.repo.CheckSlugExists(r.Context(), req.Slug, nil); err != nil {
		handleRepoError(w, err)
		return
	} else if exists {
		writeError(w, http.StatusConflict, "slug kategori sudah digunakan")
		return
	}

	// Validasi brand IDs
	if valid, err := h.repo.ValidateBrandIDs(r.Context(), req.BrandIDs); err != nil {
		handleRepoError(w, err)
		return
	} else if !valid {
		writeError(w, http.StatusBadRequest, "satu atau lebih brand_id tidak valid")
		return
	}

	cat, err := h.repo.Create(r.Context(), req)
	if err != nil {
		handleRepoError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, cat)
}

// UpdateCategory PUT /api/admin/categories/{id}
func (h *Handler) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	var req UpdateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "nama kategori wajib diisi")
		return
	}

	if strings.TrimSpace(req.Slug) == "" {
		req.Slug = slugify(req.Name)
	} else {
		req.Slug = slugify(req.Slug)
	}

	if exists, err := h.repo.CheckSlugExists(r.Context(), req.Slug, &id); err != nil {
		handleRepoError(w, err)
		return
	} else if exists {
		writeError(w, http.StatusConflict, "slug kategori sudah digunakan")
		return
	}

	if valid, err := h.repo.ValidateBrandIDs(r.Context(), req.BrandIDs); err != nil {
		handleRepoError(w, err)
		return
	} else if !valid {
		writeError(w, http.StatusBadRequest, "satu atau lebih brand_id tidak valid")
		return
	}

	cat, err := h.repo.Update(r.Context(), id, req)
	if err != nil {
		handleRepoError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, cat)
}

// DeleteCategory DELETE /api/admin/categories/{id}
func (h *Handler) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		if strings.Contains(err.Error(), "tidak bisa dihapus") {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		handleRepoError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "kategori berhasil dihapus"})
}

// ─── Public Handlers ─────────────────────────────────────────────────────────

// ListPublicCategories GET /api/public/categories?brand_id=...
func (h *Handler) ListPublicCategories(w http.ResponseWriter, r *http.Request) {
	var brandID *int64
	if bParam := r.URL.Query().Get("brand_id"); bParam != "" {
		if bid, err := strconv.ParseInt(bParam, 10, 64); err == nil && bid > 0 {
			brandID = &bid
		}
	}

	items, err := h.repo.List(r.Context(), brandID, true)
	if err != nil {
		handleRepoError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, items)
}
