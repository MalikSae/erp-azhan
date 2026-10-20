package identity

import (
	"errors"
	"log"
	"net/http"
)

// MeResponse adalah identitas pengguna admin yang sedang masuk, diturunkan dari token yang sudah divalidasi
// ERP. Dipakai CMS (cms.<domain brand>) untuk mengetahui brand, role, dan izin tanpa memegang rahasia JWT.
// BrandID kosong berarti akun Holding (semua brand).
type MeResponse struct {
	UserID      int64    `json:"user_id"`
	DisplayName string   `json:"display_name"`
	Email       string   `json:"email"`
	BrandID     *int64   `json:"brand_id"`
	Role        string   `json:"role"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
}

// Me godoc
// GET /api/admin/me
// Response 200: MeResponse. 401 bila akun sudah dinonaktifkan atau dihapus setelah token terbit.
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	resp := MeResponse{
		UserID:      GetAdminUserID(ctx),
		BrandID:     GetBrandID(ctx),
		Role:        GetRole(ctx),
		Roles:       nonNil(GetRoles(ctx)),
		Permissions: nonNil(GetPerms(ctx)),
	}
	if h.repo != nil {
		user, err := h.repo.GetByID(ctx, resp.UserID)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				writeError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			log.Printf("[ERROR] identity.Me GetByID: %v", err)
			writeError(w, http.StatusInternalServerError, "terjadi kesalahan internal")
			return
		}
		if !user.IsActive {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		resp.DisplayName = user.DisplayName
		resp.Email = user.Email
	}
	writeJSON(w, http.StatusOK, resp)
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
