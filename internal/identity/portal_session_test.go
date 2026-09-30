package identity

import (
	"context"
	"erp-azhan/api/internal/testdb"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPortalSessionResetAndLogout(t *testing.T) {
	db := testdb.Portal(t)
	t.Setenv("JWT_SECRET", "portal-test-secret-with-at-least-32-characters")
	if _, err := db.Exec(`INSERT INTO jamaah VALUES(1,1,'hash-before')`); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	old, err := GenerateAccountPortalToken(ctx, db, 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = ValidateAccountPortalToken(ctx, db, old); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`UPDATE jamaah SET portal_pin_hash='hash-after' WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	protected := RequirePortalAuth(RequirePortalSession(db)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })))
	check := func(token string, want int) {
		t.Helper()
		req := httptest.NewRequest("GET", "/portal/me", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		protected.ServeHTTP(w, req)
		t.Logf("GET protected: %d %s", w.Code, w.Body.String())
		if w.Code != want {
			t.Fatalf("got %d want %d", w.Code, want)
		}
	}
	check(old, 401)
	concurrent, err := GenerateVerifiedPortalToken(1, "hash-before")
	if err != nil {
		t.Fatal(err)
	}
	check(concurrent, 401)
	if _, err = ValidateAccountPortalToken(ctx, db, old); err == nil {
		t.Fatal("public booking accepted pre-reset token")
	}
	fresh, err := GenerateAccountPortalToken(ctx, db, 1)
	if err != nil {
		t.Fatal(err)
	}
	check(fresh, 204)
	req := httptest.NewRequest("POST", "/portal/logout", nil)
	req.Header.Set("Authorization", "Bearer "+fresh)
	w := httptest.NewRecorder()
	RequirePortalAuth(RequirePortalSession(db)(PortalLogout(db))).ServeHTTP(w, req)
	if w.Code != 204 {
		t.Fatalf("logout %d %s", w.Code, w.Body.String())
	}
	check(fresh, 401)
	if _, err = ValidateAccountPortalToken(ctx, db, fresh); err == nil {
		t.Fatal("public booking accepted revoked token")
	}
	legacy, err := GeneratePortalToken(1)
	if err != nil {
		t.Fatal(err)
	}
	check(legacy, 401)
}
