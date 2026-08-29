package crmuser

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"erp-azhan/api/internal/identity"
	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"
)

var emailPattern = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

type Handler struct{ repo *Repository }

func NewHandler(repo *Repository) *Handler { return &Handler{repo: repo} }

func (h *Handler) brandID(w http.ResponseWriter, r *http.Request) (uint64, bool) {
	if current := identity.GetBrandID(r.Context()); current != nil {
		return uint64(*current), true
	}
	value, err := strconv.ParseUint(r.Header.Get("X-CRM-Brand-ID"), 10, 64)
	if err != nil || value == 0 {
		h.error(w, http.StatusBadRequest, "brand aktif wajib dipilih")
		return 0, false
	}
	exists, err := h.repo.BrandExists(r.Context(), value)
	if err != nil {
		h.error(w, http.StatusInternalServerError, "gagal memvalidasi brand")
		return 0, false
	}
	if !exists {
		h.error(w, http.StatusNotFound, "brand tidak ditemukan")
		return 0, false
	}
	return value, true
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	brandID, ok := h.brandID(w, r)
	if !ok {
		return
	}
	users, err := h.repo.List(r.Context(), brandID)
	if err != nil {
		h.error(w, 500, "gagal mengambil akun CS")
		return
	}
	h.json(w, 200, users)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	brandID, ok := h.brandID(w, r)
	if !ok {
		return
	}
	var req CreateRequest
	if json.NewDecoder(r.Body).Decode(&req) != nil {
		h.error(w, 400, "payload tidak valid")
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.DisplayName = strings.TrimSpace(req.DisplayName)
	if !emailPattern.MatchString(req.Email) {
		h.error(w, 400, "format email tidak valid")
		return
	}
	if len(req.DisplayName) < 2 || len(req.DisplayName) > 120 {
		h.error(w, 400, "nama CS wajib 2–120 karakter")
		return
	}
	if len(req.Password) < 8 {
		h.error(w, 400, "password minimal 8 karakter")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		h.error(w, 500, "gagal memproses password")
		return
	}
	user, err := h.repo.Create(r.Context(), brandID, req.Email, req.DisplayName, string(hash))
	if errors.Is(err, ErrDuplicateEmail) {
		h.error(w, 409, "email sudah terdaftar")
		return
	}
	if err != nil {
		h.error(w, 500, "gagal membuat akun CS")
		return
	}
	h.json(w, 201, user)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	brandID, ok := h.brandID(w, r)
	if !ok {
		return
	}
	userID, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		h.error(w, 400, "ID user tidak valid")
		return
	}
	var req UpdateRequest
	if json.NewDecoder(r.Body).Decode(&req) != nil {
		h.error(w, 400, "payload tidak valid")
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.DisplayName = strings.TrimSpace(req.DisplayName)
	if !emailPattern.MatchString(req.Email) {
		h.error(w, 400, "format email tidak valid")
		return
	}
	if len(req.DisplayName) < 2 || len(req.DisplayName) > 120 {
		h.error(w, 400, "nama CS wajib 2–120 karakter")
		return
	}
	user, err := h.repo.Update(r.Context(), brandID, userID, req.Email, req.DisplayName, req.IsActive)
	if errors.Is(err, ErrNotFound) {
		h.error(w, 404, "akun CS tidak ditemukan")
		return
	}
	if errors.Is(err, ErrDuplicateEmail) {
		h.error(w, 409, "email sudah terdaftar")
		return
	}
	if err != nil {
		h.error(w, 500, "gagal memperbarui akun CS")
		return
	}
	h.json(w, 200, user)
}

func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	brandID, ok := h.brandID(w, r)
	if !ok {
		return
	}
	userID, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		h.error(w, 400, "ID user tidak valid")
		return
	}
	var req ResetPasswordRequest
	if json.NewDecoder(r.Body).Decode(&req) != nil || len(req.Password) < 8 {
		h.error(w, 400, "password minimal 8 karakter")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		h.error(w, 500, "gagal memproses password")
		return
	}
	err = h.repo.ResetPassword(r.Context(), brandID, userID, string(hash))
	if errors.Is(err, ErrNotFound) {
		h.error(w, 404, "akun CS tidak ditemukan")
		return
	}
	if err != nil {
		h.error(w, 500, "gagal mereset password")
		return
	}
	h.json(w, 200, map[string]string{"message": "password berhasil diubah"})
}

func (h *Handler) json(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
func (h *Handler) error(w http.ResponseWriter, status int, message string) {
	h.json(w, status, map[string]string{"error": message})
}
