package identity

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAccessTokenCarriesRBACClaims(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-minimum-32-characters-long")
	token, err := GenerateAccessTokenWithAccess(42, nil, "admin", "a@azhan.id",
		[]string{"super_admin_grup"}, []string{"user.admin", "booking.view"}, 3)
	if err != nil {
		t.Fatalf("GenerateAccessTokenWithAccess: %v", err)
	}
	claims, err := ValidateAccessClaims(token)
	if err != nil {
		t.Fatalf("ValidateAccessClaims: %v", err)
	}
	if claims.AdminUserID != 42 || claims.PermVersion != 3 {
		t.Fatalf("sub/pver tidak sesuai: %+v", claims)
	}
	if len(claims.Roles) != 1 || claims.Roles[0] != "super_admin_grup" {
		t.Fatalf("roles tidak sesuai: %v", claims.Roles)
	}
	if len(claims.Perms) != 2 || claims.Perms[0] != "user.admin" {
		t.Fatalf("perms tidak sesuai: %v", claims.Perms)
	}
}

func TestLegacyAccessTokenStillValid(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-minimum-32-characters-long")
	// Token lama tanpa klaim RBAC (jalur GenerateAccessToken) tetap harus valid.
	token, err := GenerateAccessToken(7, nil, "admin", "x@azhan.id")
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}
	claims, err := ValidateAccessClaims(token)
	if err != nil {
		t.Fatalf("ValidateAccessClaims token lama: %v", err)
	}
	if claims.Roles != nil || claims.Perms != nil || claims.PermVersion != 0 {
		t.Fatalf("token lama harus tanpa klaim RBAC: %+v", claims)
	}
}

func TestRequirePermission(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-minimum-32-characters-long")
	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	run := func(perms []string, wantStatus int) {
		t.Helper()
		token, err := GenerateAccessTokenWithAccess(42, nil, "admin", "", nil, perms, 1)
		if err != nil {
			t.Fatalf("generate: %v", err)
		}
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		RequireAuth(RequirePermission("user.admin")(okHandler)).ServeHTTP(rec, req)
		if rec.Code != wantStatus {
			t.Fatalf("perms=%v: status %d, want %d", perms, rec.Code, wantStatus)
		}
	}

	run([]string{"user.admin", "booking.view"}, http.StatusOK)
	run([]string{"booking.view"}, http.StatusForbidden)
	run(nil, http.StatusForbidden)
}

func TestRequireAdminRoleBlocksOnlyCS(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-minimum-32-characters-long")
	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	run := func(role string, wantStatus int) {
		t.Helper()
		token, err := GenerateAccessToken(42, nil, role)
		if err != nil {
			t.Fatalf("generate: %v", err)
		}
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		RequireAuth(RequireAdminRole(okHandler)).ServeHTTP(rec, req)
		if rec.Code != wantStatus {
			t.Fatalf("role=%s: status %d, want %d", role, rec.Code, wantStatus)
		}
	}
	run("admin", http.StatusOK)
	run("cs", http.StatusForbidden)
}
