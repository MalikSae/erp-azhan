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
		adminUserID, brandID, role, err := ValidateToken(tokenString, "access")
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		ctx := context.WithValue(r.Context(), AdminUserIDKey, adminUserID)
		ctx = context.WithValue(ctx, BrandIDKey, brandID)
		ctx = context.WithValue(ctx, RoleKey, role)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireAdminRole mencegah akun CS mengakses endpoint operasional ERP di luar gateway CRM.
func RequireAdminRole(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if GetRole(r.Context()) != "admin" {
			writeError(w, http.StatusForbidden, "akun CS hanya dapat mengakses modul CRM")
			return
		}
		next.ServeHTTP(w, r)
	})
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
