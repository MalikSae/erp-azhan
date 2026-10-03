package adminuser

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"

	"erp-azhan/api/internal/identity"
	"erp-azhan/api/internal/rbac"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// TokenRevoker mencabut semua refresh token milik seorang user (diimplementasi
// oleh identity.Repository), dipakai setelah password berubah.
type TokenRevoker interface {
	RevokeAllRefreshTokens(ctx context.Context, adminUserID int64) error
}

type Handler struct {
	repo    *Repository
	revoker TokenRevoker
}

func NewHandler(repo *Repository, revoker TokenRevoker) *Handler {
	return &Handler{repo: repo, revoker: revoker}
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

func parseID(r *http.Request) (uint64, bool) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	return id, err == nil && id > 0
}

// writeRepoError memetakan error domain ke HTTP (satu tempat untuk semua aksi).
func (h *Handler) writeRepoError(w http.ResponseWriter, err error, action string) {
	switch {
	case errors.Is(err, ErrNotFound), errors.Is(err, rbac.ErrUserNotFound):
		h.sendError(w, http.StatusNotFound, ErrNotFound.Error())
	case errors.Is(err, ErrDuplicateEmail):
		h.sendError(w, http.StatusConflict, err.Error())
	case errors.Is(err, ErrInvalidBrand), errors.Is(err, ErrInvalidKeanggotaan),
		errors.Is(err, rbac.ErrNoRole), errors.Is(err, rbac.ErrUnknownRole), errors.Is(err, rbac.ErrScopeMismatch),
		errors.Is(err, rbac.ErrUnitMismatch):
		h.sendError(w, http.StatusBadRequest, err.Error())
	case rbac.IsConcurrencyErr(err):
		h.sendError(w, http.StatusConflict, rbac.ErrConcurrentMsg)
	case errors.Is(err, rbac.ErrLastSuperAdmin), errors.Is(err, ErrCannotDeleteSelf),
		errors.Is(err, ErrCannotDeactivateSelf), errors.Is(err, ErrHasHistory):
		h.sendError(w, http.StatusConflict, err.Error())
	default:
		log.Printf("[ERROR] adminuser.%s: %v", action, err)
		h.sendError(w, http.StatusInternalServerError, "terjadi kesalahan internal")
	}
}

// validateUpsert merapikan & memvalidasi field bersama create/update.
func validateUpsert(in *UpsertInput) string {
	in.DisplayName = strings.TrimSpace(in.DisplayName)
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	in.NoHP = normalizeOptional(in.NoHP)
	in.Jabatan = normalizeOptional(in.Jabatan)
	in.Roles = rbac.NormalizeSlugs(in.Roles)
	if l := len([]rune(in.DisplayName)); l < 2 || l > 120 {
		return "nama wajib 2–120 karakter"
	}
	if !emailRegex.MatchString(in.Email) {
		return "format email tidak valid"
	}
	if in.NoHP != nil && len(*in.NoHP) > 32 {
		return "no. HP maksimal 32 karakter"
	}
	if in.Jabatan != nil && len([]rune(*in.Jabatan)) > 100 {
		return "jabatan maksimal 100 karakter"
	}
	if len(in.Roles) == 0 {
		return rbac.ErrNoRole.Error()
	}
	if in.Keanggotaan == nil {
		in.Keanggotaan = []KeanggotaanInput{}
	}
	return ""
}

// ListUsers: GET /api/admin/users
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.repo.List(r.Context())
	if err != nil {
		h.writeRepoError(w, err, "ListUsers")
		return
	}
	h.sendJSON(w, http.StatusOK, users)
}

// GetUser: GET /api/admin/users/{id}
func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r)
	if !ok {
		h.sendError(w, http.StatusBadRequest, "ID pengguna tidak valid")
		return
	}
	user, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		h.writeRepoError(w, err, "GetUser")
		return
	}
	h.sendJSON(w, http.StatusOK, user)
}

// CreateUser: POST /api/admin/users — profil + role + keanggotaan, atomik.
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req CreateAdminUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "payload tidak valid")
		return
	}
	if msg := validateUpsert(&req.UpsertInput); msg != "" {
		h.sendError(w, http.StatusBadRequest, msg)
		return
	}
	password := strings.TrimSpace(req.Password)
	if len(password) < 8 {
		h.sendError(w, http.StatusBadRequest, "password minimal 8 karakter")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		h.sendError(w, http.StatusInternalServerError, "gagal memproses password")
		return
	}

	id, err := h.repo.Create(r.Context(), req.UpsertInput, string(hash))
	if err != nil {
		h.writeRepoError(w, err, "CreateUser")
		return
	}
	log.Printf("[AUDIT] user_created user_id=%d by=%d roles=%s", id, identity.GetAdminUserID(r.Context()), strings.Join(req.Roles, ","))
	user, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		h.writeRepoError(w, err, "CreateUser")
		return
	}
	h.sendJSON(w, http.StatusCreated, user)
}

// UpdateUser: PUT /api/admin/users/{id} — atomik; R3/R4 ditegakkan.
func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r)
	if !ok {
		h.sendError(w, http.StatusBadRequest, "ID pengguna tidak valid")
		return
	}
	var req UpdateAdminUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "payload tidak valid")
		return
	}
	if msg := validateUpsert(&req.UpsertInput); msg != "" {
		h.sendError(w, http.StatusBadRequest, msg)
		return
	}
	if err := h.repo.Update(r.Context(), id, req.UpsertInput); err != nil {
		h.writeRepoError(w, err, "UpdateUser")
		return
	}
	log.Printf("[AUDIT] user_updated user_id=%d by=%d", id, identity.GetAdminUserID(r.Context()))
	user, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		h.writeRepoError(w, err, "UpdateUser")
		return
	}
	h.sendJSON(w, http.StatusOK, user)
}

// SetStatus: PUT /api/admin/users/{id}/status — offboarding/aktivasi ulang.
func (h *Handler) SetStatus(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r)
	if !ok {
		h.sendError(w, http.StatusBadRequest, "ID pengguna tidak valid")
		return
	}
	var req SetStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.IsActive == nil {
		h.sendError(w, http.StatusBadRequest, "is_active wajib diisi")
		return
	}
	actor := uint64(identity.GetAdminUserID(r.Context()))
	if err := h.repo.SetStatus(r.Context(), id, *req.IsActive, actor); err != nil {
		h.writeRepoError(w, err, "SetStatus")
		return
	}
	log.Printf("[AUDIT] user_status_changed user_id=%d active=%t by=%d", id, *req.IsActive, actor)
	user, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		h.writeRepoError(w, err, "SetStatus")
		return
	}
	h.sendJSON(w, http.StatusOK, user)
}

// DeleteUser: DELETE /api/admin/users/{id} — hanya pengguna tanpa riwayat (R7).
func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r)
	if !ok {
		h.sendError(w, http.StatusBadRequest, "ID pengguna tidak valid")
		return
	}
	actor := uint64(identity.GetAdminUserID(r.Context()))
	if err := h.repo.Delete(r.Context(), id, actor); err != nil {
		h.writeRepoError(w, err, "DeleteUser")
		return
	}
	log.Printf("[AUDIT] user_deleted user_id=%d by=%d", id, actor)
	h.sendJSON(w, http.StatusOK, map[string]string{"message": "pengguna dihapus"})
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
	if err := h.revoker.RevokeAllRefreshTokens(r.Context(), int64(id)); err != nil {
		log.Printf("[ERROR] adminuser.ResetPassword RevokeAllRefreshTokens user_id=%d: %v", id, err)
		h.sendError(w, http.StatusInternalServerError, "Password tersimpan, tetapi gagal mengakhiri sesi lama. Silakan reset ulang")
		return
	}

	h.sendJSON(w, http.StatusOK, map[string]string{"message": "password berhasil diubah"})
}

// ChangeOwnPassword: PUT /api/admin/account/password
// Hanya Admin Travel yang boleh mengubah password akunnya sendiri.
func (h *Handler) ChangeOwnPassword(w http.ResponseWriter, r *http.Request) {
	if identity.GetRole(r.Context()) != "admin" || identity.GetBrandID(r.Context()) == nil {
		h.sendError(w, http.StatusForbidden, "hanya Admin Travel yang dapat mengubah password akunnya sendiri")
		return
	}

	var req ChangeOwnPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "Payload request tidak valid")
		return
	}
	currentPassword := strings.TrimSpace(req.CurrentPassword)
	newPassword := strings.TrimSpace(req.NewPassword)
	if currentPassword == "" || newPassword == "" {
		h.sendError(w, http.StatusBadRequest, "Password saat ini dan password baru wajib diisi")
		return
	}
	if len(newPassword) < 8 {
		h.sendError(w, http.StatusBadRequest, "Password baru minimal 8 karakter")
		return
	}
	if currentPassword == newPassword {
		h.sendError(w, http.StatusBadRequest, "Password baru harus berbeda dari password saat ini")
		return
	}

	userID := uint64(identity.GetAdminUserID(r.Context()))
	passwordHash, err := h.repo.GetPasswordHash(r.Context(), userID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			h.sendError(w, http.StatusNotFound, "Akun tidak ditemukan")
			return
		}
		h.sendError(w, http.StatusInternalServerError, "Gagal memverifikasi password saat ini")
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(currentPassword)); err != nil {
		h.sendError(w, http.StatusUnauthorized, "Password saat ini salah")
		return
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		h.sendError(w, http.StatusInternalServerError, "Gagal memproses password baru")
		return
	}
	if err := h.repo.ResetPassword(r.Context(), userID, string(newHash)); err != nil {
		h.sendError(w, http.StatusInternalServerError, "Gagal menyimpan password baru")
		return
	}
	// Semua sesi akun ini diakhiri, termasuk sesi saat ini: setelah access token
	// habis (maks. 15 menit) pengguna login ulang dengan password baru.
	if err := h.revoker.RevokeAllRefreshTokens(r.Context(), int64(userID)); err != nil {
		log.Printf("[ERROR] adminuser.ChangeOwnPassword RevokeAllRefreshTokens user_id=%d: %v", userID, err)
		h.sendError(w, http.StatusInternalServerError, "Password tersimpan, tetapi gagal mengakhiri sesi lama. Silakan ganti ulang")
		return
	}

	h.sendJSON(w, http.StatusOK, map[string]string{"message": "password berhasil diubah"})
}
