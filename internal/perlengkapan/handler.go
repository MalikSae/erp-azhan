package perlengkapan

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *Handler) sendError(w http.ResponseWriter, status int, message string) {
	h.sendJSON(w, status, map[string]string{"error": message})
}

// ─── Item Handlers ────────────────────────────────────────────────────────────

// ListItems: GET /api/admin/perlengkapan-items
func (h *Handler) ListItems(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.ListItems(r.Context())
	if err != nil {
		h.sendError(w, http.StatusInternalServerError, "Gagal memuat item perlengkapan")
		return
	}
	h.sendJSON(w, http.StatusOK, items)
}

// CreateItem: POST /api/admin/perlengkapan-items
func (h *Handler) CreateItem(w http.ResponseWriter, r *http.Request) {
	var req CreateItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "Payload request tidak valid")
		return
	}

	nama := strings.TrimSpace(req.Nama)
	if nama == "" {
		h.sendError(w, http.StatusBadRequest, "Nama item perlengkapan wajib diisi")
		return
	}

	qtyPerSet := 0
	if req.QtyPerSet != nil && *req.QtyPerSet > 0 {
		qtyPerSet = *req.QtyPerSet
	}

	item, err := h.repo.CreateItem(r.Context(), nama, qtyPerSet)
	if err != nil {
		if errors.Is(err, ErrDuplicate) {
			h.sendError(w, http.StatusConflict, "Nama item perlengkapan sudah terdaftar")
			return
		}
		h.sendError(w, http.StatusInternalServerError, "Gagal menambahkan item perlengkapan")
		return
	}

	h.sendJSON(w, http.StatusCreated, item)
}

// UpdateItem: PUT /api/admin/perlengkapan-items/{id}
func (h *Handler) UpdateItem(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "ID item tidak valid")
		return
	}

	var req UpdateItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "Payload request tidak valid")
		return
	}

	nama := strings.TrimSpace(req.Nama)
	if nama == "" {
		h.sendError(w, http.StatusBadRequest, "Nama item perlengkapan wajib diisi")
		return
	}

	qtyPerSet := 0
	if req.QtyPerSet != nil && *req.QtyPerSet > 0 {
		qtyPerSet = *req.QtyPerSet
	}

	item, err := h.repo.UpdateItem(r.Context(), id, nama, qtyPerSet)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			h.sendError(w, http.StatusNotFound, "Item perlengkapan tidak ditemukan")
			return
		}
		if errors.Is(err, ErrDuplicate) {
			h.sendError(w, http.StatusConflict, "Nama item perlengkapan sudah terdaftar")
			return
		}
		h.sendError(w, http.StatusInternalServerError, "Gagal mengubah item perlengkapan")
		return
	}

	h.sendJSON(w, http.StatusOK, item)
}

// DeleteItem: DELETE /api/admin/perlengkapan-items/{id}
func (h *Handler) DeleteItem(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "ID item tidak valid")
		return
	}

	if err := h.repo.DeleteItem(r.Context(), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			h.sendError(w, http.StatusNotFound, "Item perlengkapan tidak ditemukan")
			return
		}
		if errors.Is(err, ErrInUse) {
			h.sendError(w, http.StatusConflict, "tidak bisa dihapus, item masih dipakai di template set atau masih memiliki stok")
			return
		}
		h.sendError(w, http.StatusInternalServerError, "Gagal menghapus item perlengkapan")
		return
	}

	h.sendJSON(w, http.StatusOK, map[string]string{"message": "Item perlengkapan berhasil dihapus"})
}

// ─── Stok Handlers ────────────────────────────────────────────────────────────

// ListStok: GET /api/admin/perlengkapan-stok?brand_id={id}
func (h *Handler) ListStok(w http.ResponseWriter, r *http.Request) {
	brandIDStr := r.URL.Query().Get("brand_id")
	if brandIDStr == "" {
		items, err := h.repo.ListStokAll(r.Context())
		if err != nil {
			h.sendError(w, http.StatusInternalServerError, "Gagal memuat stok perlengkapan seluruh brand")
			return
		}
		h.sendJSON(w, http.StatusOK, items)
		return
	}

	brandID, err := strconv.ParseUint(brandIDStr, 10, 64)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "brand_id tidak valid")
		return
	}

	items, err := h.repo.ListStokByBrand(r.Context(), brandID)
	if err != nil {
		h.sendError(w, http.StatusInternalServerError, "Gagal memuat stok perlengkapan")
		return
	}

	h.sendJSON(w, http.StatusOK, items)
}

// UpdateStok: PUT /api/admin/perlengkapan-stok/{item_id}?brand_id={id}
func (h *Handler) UpdateStok(w http.ResponseWriter, r *http.Request) {
	itemIDStr := chi.URLParam(r, "item_id")
	itemID, err := strconv.ParseUint(itemIDStr, 10, 64)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "item_id tidak valid")
		return
	}

	brandIDStr := r.URL.Query().Get("brand_id")
	if brandIDStr == "" {
		h.sendError(w, http.StatusBadRequest, "Parameter brand_id wajib diisi")
		return
	}

	brandID, err := strconv.ParseUint(brandIDStr, 10, 64)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "brand_id tidak valid")
		return
	}

	var req UpdateStokRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "Payload request tidak valid")
		return
	}

	if req.StokTersedia == nil || *req.StokTersedia < 0 {
		h.sendError(w, http.StatusBadRequest, "stok_tersedia wajib bernilai 0 atau lebih")
		return
	}

	item, err := h.repo.UpdateStok(r.Context(), brandID, itemID, *req.StokTersedia)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			h.sendError(w, http.StatusNotFound, "Item perlengkapan tidak ditemukan")
			return
		}
		h.sendError(w, http.StatusInternalServerError, "Gagal memperbarui stok perlengkapan")
		return
	}

	h.sendJSON(w, http.StatusOK, item)
}

// ─── Set Template Handlers ───────────────────────────────────────────────────

// GetSetTemplate: GET /api/admin/perlengkapan-set-template
func (h *Handler) GetSetTemplate(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.GetSetTemplate(r.Context())
	if err != nil {
		h.sendError(w, http.StatusInternalServerError, "Gagal memuat template set perlengkapan")
		return
	}
	h.sendJSON(w, http.StatusOK, items)
}

// UpdateSetTemplate: PUT /api/admin/perlengkapan-set-template
func (h *Handler) UpdateSetTemplate(w http.ResponseWriter, r *http.Request) {
	var inputs []SetTemplateItemInput
	if err := json.NewDecoder(r.Body).Decode(&inputs); err != nil {
		h.sendError(w, http.StatusBadRequest, "Payload request tidak valid (harus array)")
		return
	}

	items, err := h.repo.UpdateSetTemplate(r.Context(), inputs)
	if err != nil {
		if errors.Is(err, ErrInvalidInput) {
			h.sendError(w, http.StatusBadRequest, err.Error())
			return
		}
		h.sendError(w, http.StatusInternalServerError, "Gagal memperbarui template set perlengkapan")
		return
	}

	h.sendJSON(w, http.StatusOK, items)
}
