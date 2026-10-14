package identity

import (
	"context"
	"net/http"
	"regexp"
	"strings"
)

var crmPaymentPath = regexp.MustCompile(`^/api/admin/bookings/[0-9]+/payments$`)

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
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := ValidateAccessClaims(tokenString)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
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

// RequireAdminRole mencegah akun CS mengakses endpoint operasional ERP di luar
// gateway CRM. Wrapper transisi RBAC Fase 1: semua role non-CS lolos; penggantian
// per-route ke RequirePermission dilakukan bertahap di Fase 2.
func RequireAdminRole(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if GetRole(r.Context()) == "cs" {
			writeError(w, http.StatusForbidden, "akun CS hanya dapat mengakses modul CRM")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequirePermission memeriksa klaim perms pada access token (diisi saat
// login/refresh dari tabel RBAC). Tanpa permission yang diminta → 403.
func RequirePermission(perm string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !HasPermission(r.Context(), perm) {
				writeError(w, http.StatusForbidden, "anda tidak memiliki izin untuk aksi ini")
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

// RequireAdminOrCRMAccess keeps CS credentials out of the ERP backoffice while
// allowing only the small ERP contract used by the CRM BFF.
func RequireAdminOrCRMAccess(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if GetRole(r.Context()) != "cs" {
			next.ServeHTTP(w, r)
			return
		}
		allowed := (r.Method == http.MethodGet && r.URL.Path == "/api/admin/my-brand") ||
			(r.Method == http.MethodGet && r.URL.Path == "/api/admin/schedules") ||
			(r.Method == http.MethodPost && r.URL.Path == "/api/admin/crm/deals") ||
			(r.Method == http.MethodGet && crmPaymentPath.MatchString(r.URL.Path))
		if !allowed {
			writeError(w, http.StatusForbidden, "akun CS hanya dapat mengakses modul CRM")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequirePortalAuth middleware memvalidasi portal token di header Authorization
func RequirePortalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		jamaahID, err := ValidatePortalToken(tokenString)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		ctx := context.WithValue(r.Context(), PortalJamaahIDKey, jamaahID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
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

// GetPortalJamaahID mengambil ID jamaah dari context portal
func GetPortalJamaahID(ctx context.Context) int64 {
	if val, ok := ctx.Value(PortalJamaahIDKey).(int64); ok {
		return val
	}
	return 0
}
