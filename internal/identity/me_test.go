package identity

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func meRequest(ctx context.Context) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/api/admin/me", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	NewHandler(nil).Me(rec, req)
	return rec
}

// /api/admin/me mengembalikan identitas dari klaim token yang sudah divalidasi middleware.
func TestMeMengembalikanKlaimToken(t *testing.T) {
	brand := int64(2)
	ctx := context.WithValue(context.Background(), AdminUserIDKey, int64(7))
	ctx = context.WithValue(ctx, BrandIDKey, &brand)
	ctx = context.WithValue(ctx, RoleKey, "admin")
	ctx = context.WithValue(ctx, RolesKey, []string{"marketing_member"})
	ctx = context.WithValue(ctx, PermsKey, []string{"cms.view", "cms.edit"})

	rec := meRequest(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var got MeResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.UserID != 7 || got.BrandID == nil || *got.BrandID != 2 || got.Role != "admin" {
		t.Fatalf("identitas salah: %+v", got)
	}
	if len(got.Roles) != 1 || got.Roles[0] != "marketing_member" || len(got.Permissions) != 2 {
		t.Fatalf("role/izin salah: %+v", got)
	}
}

// Akun Holding tidak terikat brand dan token lama tanpa klaim RBAC tetap menghasilkan JSON bentuk tetap.
func TestMeHoldingDanKlaimKosong(t *testing.T) {
	ctx := context.WithValue(context.Background(), AdminUserIDKey, int64(1))
	ctx = context.WithValue(ctx, BrandIDKey, (*int64)(nil))
	ctx = context.WithValue(ctx, RoleKey, "admin")

	rec := meRequest(ctx)
	var raw map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatal(err)
	}
	if v, ok := raw["brand_id"]; !ok || v != nil {
		t.Errorf("brand_id harus ada dan null untuk Holding, dapat %v (ada=%v)", v, ok)
	}
	for _, k := range []string{"roles", "permissions"} {
		arr, ok := raw[k].([]any)
		if !ok || len(arr) != 0 {
			t.Errorf("%s harus berupa array kosong, dapat %v", k, raw[k])
		}
	}
}
