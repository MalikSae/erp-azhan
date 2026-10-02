package rbac

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

// refreshRevoker dipenuhi identity.Repository: pencabutan refresh token tetap
// lewat interface, bukan query langsung ke tabel milik modul identity.
type refreshRevoker interface {
	RevokeAllRefreshTokens(ctx context.Context, adminUserID int64) error
}

type Handler struct {
	repo    *Repository
	revoker refreshRevoker
}

func NewHandler(repo *Repository, revoker refreshRevoker) *Handler {
	return &Handler{repo: repo, revoker: revoker}
}

// ListRoles GET /api/admin/roles
func (h *Handler) ListRoles(w http.ResponseWriter, r *http.Request) {
	roles, err := h.repo.ListRoles(r.Context())
	if err != nil {
		log.Printf("[ERROR] rbac.ListRoles: %v", err)
		writeError(w, http.StatusInternalServerError, "gagal memuat role")
		return
	}
	writeJSON(w, http.StatusOK, roles)
}

// ListPermissions GET /api/admin/permissions
func (h *Handler) ListPermissions(w http.ResponseWriter, r *http.Request) {
	perms, err := h.repo.ListPermissions(r.Context())
	if err != nil {
		log.Printf("[ERROR] rbac.ListPermissions: %v", err)
		writeError(w, http.StatusInternalServerError, "gagal memuat permission")
		return
	}
	writeJSON(w, http.StatusOK, perms)
}

// GetUserRoles GET /api/admin/users/{id}/roles
func (h *Handler) GetUserRoles(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || userID <= 0 {
		writeError(w, http.StatusBadRequest, "id user tidak valid")
		return
	}
	roles, err := h.repo.GetUserRoles(r.Context(), userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		log.Printf("[ERROR] rbac.GetUserRoles: %v", err)
		writeError(w, http.StatusInternalServerError, "gagal memuat role user")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"roles": roles})
}

// SetUserRoles PUT /api/admin/users/{id}/roles
// Mengganti seluruh role user, menaikkan perm_version, dan mencabut semua
// refresh token aktif supaya klaim lama mati bersama umur access token.
func (h *Handler) SetUserRoles(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || userID <= 0 {
		writeError(w, http.StatusBadRequest, "id user tidak valid")
		return
	}

	var req SetUserRolesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}
	slugs := make([]string, 0, len(req.Roles))
	for _, s := range req.Roles {
		if s = strings.TrimSpace(s); s != "" {
			slugs = append(slugs, s)
		}
	}
	if len(slugs) == 0 {
		writeError(w, http.StatusBadRequest, "minimal satu role wajib dipilih")
		return
	}

	if err := h.repo.SetUserRoles(r.Context(), userID, slugs); err != nil {
		switch {
		case errors.Is(err, ErrUserNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, ErrUnknownRole), errors.Is(err, ErrScopeMismatch):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			log.Printf("[ERROR] rbac.SetUserRoles: %v", err)
			writeError(w, http.StatusInternalServerError, "gagal mengubah role user")
		}
		return
	}

	if err := h.revoker.RevokeAllRefreshTokens(r.Context(), userID); err != nil {
		// Role sudah berubah; kegagalan revoke tidak membatalkan, hanya dicatat.
		log.Printf("[ERROR] rbac.SetUserRoles revoke tokens: %v", err)
	}
	log.Printf("[AUDIT] user_roles_changed user_id=%d roles=%s", userID, strings.Join(slugs, ","))

	roles, err := h.repo.GetUserRoles(r.Context(), userID)
	if err != nil {
		roles = slugs
	}
	writeJSON(w, http.StatusOK, map[string]any{"roles": roles})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
