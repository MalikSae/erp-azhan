package adminuser

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"

	"erp-azhan/api/internal/identity"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) sendJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (h *Handler) sendError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// ListUsers: GET /api/admin/users
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.repo.List(r.Context())
	if err != nil {
		h.sendError(w, http.StatusInternalServerError, "Gagal mengambil data user")
		return
	}
	h.sendJSON(w, http.StatusOK, users)
}

// CreateUser: POST /api/admin/users
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req CreateAdminUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "Payload request tidak valid")
		return
	}

	email := strings.TrimSpace(req.Email)
	if email == "" || !emailRegex.MatchString(email) {
		h.sendError(w, http.StatusBadRequest, "Format email tidak valid")
		return
	}

	password := strings.TrimSpace(req.Password)
	if len(password) < 8 {
		h.sendError(w, http.StatusBadRequest, "Password minimal 8 karakter")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		h.sendError(w, http.StatusInternalServerError, "Gagal memproses password")
		return
	}

	user, err := h.repo.Create(r.Context(), email, string(hash), req.BrandID)
	if err != nil {
		if errors.Is(err, ErrDuplicateEmail) {
			h.sendError(w, http.StatusConflict, "Email sudah terdaftar")
			return
		}
		if errors.Is(err, ErrInvalidBrand) {
			h.sendError(w, http.StatusBadRequest, "Brand yang dipilih tidak valid atau tidak ditemukan")
			return
		}
		h.sendError(w, http.StatusInternalServerError, "Gagal menambahkan user baru")
		return
	}

	h.sendJSON(w, http.StatusCreated, user)
}

// UpdateUser: PUT /api/admin/users/{id}
func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "ID user tidak valid")
		return
	}

	var req UpdateAdminUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "Payload request tidak valid")
		return
	}

	email := strings.TrimSpace(req.Email)
	if email == "" || !emailRegex.MatchString(email) {
		h.sendError(w, http.StatusBadRequest, "Format email tidak valid")
		return
	}

	user, err := h.repo.Update(r.Context(), id, email, req.BrandID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			h.sendError(w, http.StatusNotFound, "User tidak ditemukan")
			return
		}
		if errors.Is(err, ErrDuplicateEmail) {
			h.sendError(w, http.StatusConflict, "Email sudah terdaftar")
			return
		}
		if errors.Is(err, ErrInvalidBrand) {
			h.sendError(w, http.StatusBadRequest, "Brand yang dipilih tidak valid atau tidak ditemukan")
			return
		}
		if errors.Is(err, ErrLastSuperAdmin) {
			h.sendError(w, http.StatusBadRequest, "tidak bisa mengubah role Super Admin terakhir, sistem harus punya minimal 1 Super Admin")
			return
		}
		h.sendError(w, http.StatusInternalServerError, "Gagal memperbarui user")
		return
	}

	h.sendJSON(w, http.StatusOK, user)
}

// ResetPassword: PUT /api/admin/users/{id}/password
func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "ID user tidak valid")
		return
	}

	var req ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "Payload request tidak valid")
		return
	}

	password := strings.TrimSpace(req.Password)
	if len(password) < 8 {
		h.sendError(w, http.StatusBadRequest, "Password minimal 8 karakter")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		h.sendError(w, http.StatusInternalServerError, "Gagal memproses password")
		return
	}

	if err := h.repo.ResetPassword(r.Context(), id, string(hash)); err != nil {
		if errors.Is(err, ErrNotFound) {
			h.sendError(w, http.StatusNotFound, "User tidak ditemukan")
			return
		}
		h.sendError(w, http.StatusInternalServerError, "Gagal mereset password")
		return
	}

	h.sendJSON(w, http.StatusOK, map[string]string{"message": "password berhasil diubah"})
}

// DeleteUser: DELETE /api/admin/users/{id}
func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "ID user tidak valid")
		return
	}

	currentAdminID := uint64(identity.GetAdminUserID(r.Context()))

	if err := h.repo.Delete(r.Context(), id, currentAdminID); err != nil {
		if errors.Is(err, ErrCannotDeleteSelf) {
			h.sendError(w, http.StatusBadRequest, "tidak bisa menghapus akun sendiri")
			return
		}
		if errors.Is(err, ErrLastSuperAdmin) {
			h.sendError(w, http.StatusBadRequest, "tidak bisa menghapus Super Admin terakhir")
			return
		}
		if errors.Is(err, ErrNotFound) {
			h.sendError(w, http.StatusNotFound, "User tidak ditemukan")
			return
		}
		h.sendError(w, http.StatusInternalServerError, "Gagal menghapus user")
		return
	}

	h.sendJSON(w, http.StatusOK, map[string]string{"message": "user berhasil dihapus"})
}
