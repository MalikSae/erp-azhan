package selfbooking

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"erp-azhan/api/internal/customer"
)

const validBookingBody = `{
	"captcha_token": "token-uji",
	"schedule_id": 10,
	"pic": {"nama_lengkap": "Uji", "jenis_kelamin": "L", "room_type": "Quad"}
}`

var akunHandlerUji = &customer.Account{ID: 77, BrandID: 2, NamaLengkap: "Uji", Email: "uji@example.com", NoHP: "0812-3456-7890"}

func newTestHandler(t *testing.T, siteverifyBody string, siteverifyStatus int) *Handler {
	t.Helper()
	h := &Handler{
		captcha:          &turnstileVerifier{},
		invoiceMissIP:    newWindowLimiter(invoiceMissIPLimit, 15*time.Minute),
		bookingByIP:      newWindowLimiter(bookingIPAttemptLimit, time.Hour),
		bookingByAccount: newWindowLimiter(bookingAccountSuccessLimit, 24*time.Hour),
		bookingByAgen:    newWindowLimiter(agenBookingLimit, time.Hour),
	}
	if siteverifyBody != "" {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := r.ParseForm(); err != nil || r.PostForm.Get("secret") != "secret-uji" || r.PostForm.Get("response") != "token-uji" {
				t.Errorf("siteverify menerima form tidak sesuai: %v", r.PostForm)
			}
			w.WriteHeader(siteverifyStatus)
			w.Write([]byte(siteverifyBody))
		}))
		t.Cleanup(srv.Close)
		h.captcha = &turnstileVerifier{secret: "secret-uji", verifyURL: srv.URL, client: srv.Client()}
	}
	return h
}

func postBook(h *Handler, body string, akun *customer.Account) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/account/bookings", strings.NewReader(body))
	req.RemoteAddr = "198.51.100.9:5555"
	if akun != nil {
		req = req.WithContext(customer.ContextWithAccount(req.Context(), akun))
	}
	rec := httptest.NewRecorder()
	h.CreateBooking(rec, req)
	return rec
}

func TestCreateBookingRequiresAccount(t *testing.T) {
	h := newTestHandler(t, "", 0)
	if rec := postBook(h, validBookingBody, nil); rec.Code != http.StatusUnauthorized {
		t.Fatalf("tanpa sesi: got %d %s, want 401", rec.Code, rec.Body.String())
	}
}

func TestCreateBookingRejectsInvalidCaptcha(t *testing.T) {
	h := newTestHandler(t, `{"success":false,"error-codes":["invalid-input-response"]}`, http.StatusOK)
	rec := postBook(h, validBookingBody, akunHandlerUji)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "Verifikasi keamanan gagal") {
		t.Fatalf("got %d %s, want 400 captcha error", rec.Code, rec.Body.String())
	}
}

func TestCreateBookingFailsClosedWhenSiteverifyDown(t *testing.T) {
	h := newTestHandler(t, `oops`, http.StatusInternalServerError)
	rec := postBook(h, validBookingBody, akunHandlerUji)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("got %d %s, want 503", rec.Code, rec.Body.String())
	}
}

func TestCreateBookingSkipsCaptchaWhenDisabled(t *testing.T) {
	// Tanpa TURNSTILE_SECRET_KEY, captcha_token tidak diwajibkan: request lolos
	// gate captcha dan berhenti di batas booking per akun (repo nil tidak tersentuh).
	h := newTestHandler(t, "", 0)
	for i := 0; i < bookingAccountSuccessLimit; i++ {
		h.bookingByAccount.record("77")
	}
	body := strings.Replace(validBookingBody, `"captcha_token": "token-uji",`, "", 1)
	rec := postBook(h, body, akunHandlerUji)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("got %d %s, want 429 (lolos gate captcha)", rec.Code, rec.Body.String())
	}
}

func TestCreateBookingIPLimit(t *testing.T) {
	h := newTestHandler(t, "", 0)
	for i := 1; i <= bookingIPAttemptLimit; i++ {
		if rec := postBook(h, `bukan-json`, akunHandlerUji); rec.Code != http.StatusBadRequest {
			t.Fatalf("percobaan #%d: got %d, want 400", i, rec.Code)
		}
	}
	rec := postBook(h, `bukan-json`, akunHandlerUji)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("percobaan #%d: got %d %s, want 429", bookingIPAttemptLimit+1, rec.Code, rec.Body.String())
	}
}

func TestCreateBookingAccountLimitBeforeProcessing(t *testing.T) {
	// Captcha valid; akun sudah mencapai batas, jadi request harus berhenti
	// sebelum menyentuh repository (repo nil akan panic jika tersentuh).
	h := newTestHandler(t, `{"success":true}`, http.StatusOK)
	for i := 0; i < bookingAccountSuccessLimit; i++ {
		h.bookingByAccount.record("77")
	}
	rec := postBook(h, validBookingBody, akunHandlerUji)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("got %d %s, want 429", rec.Code, rec.Body.String())
	}
}

func TestGetPublicInvoiceBlockedAfterMisses(t *testing.T) {
	// IP yang sudah mencapai batas kode salah diblokir sebelum query ke
	// repository (repo nil akan panic jika tersentuh).
	h := newTestHandler(t, "", 0)
	for i := 0; i < invoiceMissIPLimit; i++ {
		h.invoiceMissIP.record("198.51.100.9")
	}
	req := httptest.NewRequest(http.MethodGet, "/api/public/invoice/HNABCD", nil)
	req.RemoteAddr = "198.51.100.9:5555"
	rec := httptest.NewRecorder()
	h.GetPublicInvoice(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("got %d %s, want 429", rec.Code, rec.Body.String())
	}
}

func TestWindowLimiterResetsAfterWindow(t *testing.T) {
	l := newWindowLimiter(2, 50*time.Millisecond)
	l.record("k")
	l.record("k")
	if l.allow("k") {
		t.Fatal("allow() = true setelah mencapai batas")
	}
	time.Sleep(60 * time.Millisecond)
	if !l.allow("k") {
		t.Fatal("allow() = false setelah jendela berakhir")
	}
}
