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
		adminUserID, brandID, err := ValidateToken(tokenString, "access")
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		ctx := context.WithValue(r.Context(), AdminUserIDKey, adminUserID)
		ctx = context.WithValue(ctx, BrandIDKey, brandID)
		next.ServeHTTP(w, r.WithContext(ctx))
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

// GetPortalJamaahID mengambil ID jamaah dari context portal
func GetPortalJamaahID(ctx context.Context) int64 {
	if val, ok := ctx.Value(PortalJamaahIDKey).(int64); ok {
		return val
	}
	return 0
}

