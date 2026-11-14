package selfbooking

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"erp-azhan/api/internal/customer"
)

// Booking dari app tidak memakai captcha: dengan verifier yang selalu menolak, endpoint web gagal di gerbang
// captcha, sedangkan endpoint app lolos dan baru ditolak validasi berikutnya (body uji tidak lengkap).
func TestCreateBookingAppTanpaCaptcha(t *testing.T) {
	h := newTestHandler(t, `{"success":false,"error-codes":["invalid-input-response"]}`, http.StatusOK)
	body := strings.Replace(validBookingBody, `"captcha_token": "token-uji",`, "", 1)

	if rec := postBook(h, body, akunHandlerUji); !strings.Contains(rec.Body.String(), "Verifikasi keamanan") {
		t.Fatalf("booking web tanpa token harus ditolak captcha: %d %s", rec.Code, rec.Body.String())
	}

	req := httptest.NewRequest(http.MethodPost, "/api/account/bookings-app", strings.NewReader(body))
	req.RemoteAddr = "198.51.100.10:5555"
	req = req.WithContext(customer.ContextWithAccount(req.Context(), akunHandlerUji))
	rec := httptest.NewRecorder()
	h.CreateBookingApp(rec, req)
	if strings.Contains(rec.Body.String(), "Verifikasi keamanan") {
		t.Fatalf("booking app tidak boleh memeriksa captcha: %d %s", rec.Code, rec.Body.String())
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("body tidak lengkap harus ditolak validasi: %d %s", rec.Code, rec.Body.String())
	}
}

func TestCreateBookingAppButuhSesi(t *testing.T) {
	h := newTestHandler(t, "", 0)
	req := httptest.NewRequest(http.MethodPost, "/api/account/bookings-app", strings.NewReader(validBookingBody))
	rec := httptest.NewRecorder()
	h.CreateBookingApp(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("tanpa sesi: got %d %s, want 401", rec.Code, rec.Body.String())
	}
}
