package identity

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const (
	AdminUserIDKey    contextKey = "adminUserID"
	BrandIDKey        contextKey = "brandID"
	RoleKey           contextKey = "role"
	RolesKey          contextKey = "roles"
	PermsKey          contextKey = "perms"
	PortalJamaahIDKey contextKey = "portalJamaahID"
)

// RequireAuth middleware memvalidasi access token di header Authorization
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			writeError(w, http.StatusUnauthorized, "Unauthorized.")
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := ValidateAccessClaims(tokenString)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "Unauthorized.")
			return
		}
		// Cek ringan status user (cache 30 detik): akun nonaktif atau izin yang berubah tidak
		// menunggu access token kedaluwarsa. 401 membuat klien me-refresh token.
		ok, err := statusCache.allow(r.Context(), claims)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "Layanan sementara tidak tersedia, coba lagi.")
			return
		}
		if !ok {
			writeError(w, http.StatusUnauthorized, "Unauthorized.")
			return
		}

		ctx := context.WithValue(r.Context(), AdminUserIDKey, claims.AdminUserID)
		ctx = context.WithValue(ctx, BrandIDKey, claims.BrandID)
		ctx = context.WithValue(ctx, RoleKey, claims.Role)
		ctx = context.WithValue(ctx, RolesKey, claims.Roles)
		ctx = context.WithValue(ctx, PermsKey, claims.Perms)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequirePermission memeriksa klaim perms pada access token (diisi saat
// login/refresh dari internal/akses). Tanpa permission yang diminta → 403.
func RequirePermission(perm string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !HasPermission(r.Context(), perm) {
				writeError(w, http.StatusForbidden, "Anda tidak memiliki izin untuk aksi ini.")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireHoldingPermission seperti RequirePermission, tetapi juga menolak token akun brand.
// Dipakai untuk izin tingkat holding (user.admin: kelola pengguna) yang handlernya tidak memfilter
// brand. Izin itu hanya dimiliki Super Admin; penjaga ini lapis kedua bila token brand keliru
// membawanya.
func RequireHoldingPermission(perm string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !HasPermission(r.Context(), perm) || GetBrandID(r.Context()) != nil {
				writeError(w, http.StatusForbidden, "Anda tidak memiliki izin untuk aksi ini.")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// HasPermission true bila klaim perms pada context memuat permission tsb.
func HasPermission(ctx context.Context, perm string) bool {
	for _, p := range GetPerms(ctx) {
		if p == perm {
			return true
		}
	}
	return false
}

// GetPerms mengambil daftar permission dari context.
func GetPerms(ctx context.Context) []string {
	if val, ok := ctx.Value(PermsKey).([]string); ok {
		return val
	}
	return nil
}

// GetRoles mengambil daftar slug role dari context.
func GetRoles(ctx context.Context) []string {
	if val, ok := ctx.Value(RolesKey).([]string); ok {
		return val
	}
	return nil
}

// GetAdminUserID mengambil ID admin dari context
func GetAdminUserID(ctx context.Context) int64 {
	if val, ok := ctx.Value(AdminUserIDKey).(int64); ok {
		return val
	}
	return 0
}

// GetBrandID mengambil brand ID dari context. Return nil jika Super Admin.
func GetBrandID(ctx context.Context) *int64 {
	if val, ok := ctx.Value(BrandIDKey).(*int64); ok {
		return val
	}
	return nil
}

func GetRole(ctx context.Context) string {
	if val, ok := ctx.Value(RoleKey).(string); ok {
		return val
	}
	return ""
}

// GetPortalJamaahID: hanya sisa media portal jamaah lama yang masih membacanya; selalu 0 selama portal PIN
// tidak dipasang. Handler agen Syiar kini memakai akun: lihat GetCustomerAccountID (customer_ctx.go).
func GetPortalJamaahID(ctx context.Context) int64 {
	if val, ok := ctx.Value(PortalJamaahIDKey).(int64); ok {
		return val
	}
	return 0
}
