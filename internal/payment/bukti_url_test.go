package payment

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"erp-azhan/api/internal/identity"
	"erp-azhan/api/internal/testdb"
	"github.com/go-chi/chi/v5"
)

// Pembayaran admin: bukti_url wajib unggahan admin resmi. URL luar (yang akan dimuat dashboard dengan token admin)
// dan path buatan ditolak sebelum booking diperiksa atau pembayaran dibuat.
func TestCreatePaymentRejectsUnofficialBuktiURL(t *testing.T) {
	db := testdb.Portal(t)
	t.Chdir(t.TempDir())
	h := NewHandler(NewRepository(db))
	brand := int64(10)

	for _, tc := range []struct {
		nama  string
		bukti string
		brand *int64
	}{
		{"URL luar, admin brand", "https://evil.example/x.pdf", &brand},
		{"URL luar, super admin", "https://evil.example/x.pdf", nil},
		{"skema protocol-relative", "//evil.example/x.pdf", nil},
		{"path buatan", "/api/admin/media/payment-proofs/../../etc/passwd", nil},
		{"kategori lain", "/api/admin/media/dokumen-jamaah/11111111-1111-1111-1111-111111111111.pdf", nil},
		{"unggahan tidak tercatat", "/api/admin/media/payment-proofs/55555555-5555-5555-5555-555555555555.pdf", nil},
	} {
		body := `{"jumlah":1000,"metode":"transfer","bank_account_id":1,"bukti_url":"` + tc.bukti + `"}`
		req := httptest.NewRequest(http.MethodPost, "/api/admin/bookings/1/payments", strings.NewReader(body))
		rc := chi.NewRouteContext()
		rc.URLParams.Add("booking_id", "1")
		ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rc)
		ctx = context.WithValue(ctx, identity.AdminUserIDKey, int64(7))
		ctx = context.WithValue(ctx, identity.BrandIDKey, tc.brand)
		rec := httptest.NewRecorder()
		h.CreatePayment(rec, req.WithContext(ctx))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("%s: status %d, harus 400 (body: %s)", tc.nama, rec.Code, rec.Body.String())
		}
	}
}
