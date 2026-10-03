package orgunit

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"erp-azhan/api/internal/identity"
	"erp-azhan/api/internal/rbac"
)

type Handler struct{ repo *Repository }

func NewHandler(repo *Repository) *Handler { return &Handler{repo: repo} }

func parseID(r *http.Request) (uint64, bool) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	return id, err == nil && id > 0
}

func validateNama(nama string) (string, string) {
	nama = strings.TrimSpace(nama)
	if l := len([]rune(nama)); l < 2 || l > 120 {
		return nama, "nama unit wajib 2–120 karakter"
	}
	return nama, ""
}

func (h *Handler) writeRepoError(w http.ResponseWriter, err error, action string) {
	switch {
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrInvalidJenis), errors.Is(err, ErrInvalidParent):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrParentInactive), errors.Is(err, ErrDuplicateNama),
		errors.Is(err, ErrHasChildren), errors.Is(err, ErrChildActive), errors.Is(err, ErrHasMembers),
		errors.Is(err, ErrIsRoleDivisi), errors.Is(err, ErrMoveHasMembers):
		writeError(w, http.StatusConflict, err.Error())
	case rbac.IsConcurrencyErr(err):
		writeError(w, http.StatusConflict, rbac.ErrConcurrentMsg)
	default:
		log.Printf("[ERROR] orgunit.%s: %v", action, err)
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan internal")
	}
}

// List GET /api/admin/org-units — pohon divisi -> tim.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	tree, err := h.repo.Tree(r.Context())
	if err != nil {
		h.writeRepoError(w, err, "List")
		return
	}
	writeJSON(w, http.StatusOK, tree)
}

// Create POST /api/admin/org-units
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "payload tidak valid")
		return
	}
	var msg string
	if req.Nama, msg = validateNama(req.Nama); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	id, err := h.repo.Create(r.Context(), req)
	if err != nil {
		h.writeRepoError(w, err, "Create")
		return
	}
	log.Printf("[AUDIT] org_unit_created id=%d jenis=%s by=%d", id, req.Jenis, identity.GetAdminUserID(r.Context()))
	unit, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		h.writeRepoError(w, err, "Create")
		return
	}
	writeJSON(w, http.StatusCreated, unit)
}

// Update PUT /api/admin/org-units/{id}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "ID unit tidak valid")
		return
	}
	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "payload tidak valid")
		return
	}
	var msg string
	if req.Nama, msg = validateNama(req.Nama); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	if err := h.repo.Update(r.Context(), id, req); err != nil {
		h.writeRepoError(w, err, "Update")
		return
	}
	unit, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		h.writeRepoError(w, err, "Update")
		return
	}
	writeJSON(w, http.StatusOK, unit)
}

// SetStatus PUT /api/admin/org-units/{id}/status
func (h *Handler) SetStatus(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "ID unit tidak valid")
		return
	}
	var req SetStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.IsActive == nil {
		writeError(w, http.StatusBadRequest, "is_active wajib diisi")
		return
	}
	if err := h.repo.SetStatus(r.Context(), id, *req.IsActive); err != nil {
		h.writeRepoError(w, err, "SetStatus")
		return
	}
	log.Printf("[AUDIT] org_unit_status id=%d active=%t by=%d", id, *req.IsActive, identity.GetAdminUserID(r.Context()))
	unit, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		h.writeRepoError(w, err, "SetStatus")
		return
	}
	writeJSON(w, http.StatusOK, unit)
}

// Delete DELETE /api/admin/org-units/{id} — hanya unit kosong (R9).
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "ID unit tidak valid")
		return
	}
	if err := h.repo.Delete(r.Context(), id); err != nil {
		h.writeRepoError(w, err, "Delete")
		return
	}
	log.Printf("[AUDIT] org_unit_deleted id=%d by=%d", id, identity.GetAdminUserID(r.Context()))
	writeJSON(w, http.StatusOK, map[string]string{"message": "unit dihapus"})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
