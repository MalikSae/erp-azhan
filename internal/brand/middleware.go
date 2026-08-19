package brand

import (

	"net/http"

	"erp-azhan/api/internal/identity"
)

// RequireSuperAdmin middleware untuk memastikan hanya Super Admin Grup (brand_id = null)
// yang dapat mengakses endpoint tertentu (misalnya manajemen brand).
func RequireSuperAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		brandID := identity.GetBrandID(r.Context())
		if brandID != nil {
			writeError(w, http.StatusForbidden, "hanya Super Admin Grup yang dapat mengakses ini")
			return
		}
		next.ServeHTTP(w, r)
	})
}
