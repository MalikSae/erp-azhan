package selfbooking

import (
	"context"
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

var waktuVerifikasiUji = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

var akunHandlerUji = &customer.Account{ID: 77, BrandID: 2, NamaLengkap: "Uji", Email: "uji@example.com", NoHP: "0812-3456-7890", EmailVerifiedAt: &waktuVerifikasiUji}

// newTestHandler: tanpa siteverifyBody, captcha memakai bypass pengembangan eksplisit.
func newTestHandler(t *testing.T, siteverifyBody string, siteverifyStatus int) *Handler {
	t.Helper()
	h := &Handler{
		captcha:          &turnstileVerifier{devBypass: true},
		invoiceMissIP:    newWindowLimiter(invoiceMissIPLimit, 15*time.Minute),
		bookingByIP:      newWindowLimiter(bookingIPAttemptLimit, time.Hour),
		bookingByAccount: newWindowLimiter(bookingAccountSuccessLimit, 24*time.Hour),
		bookingByAgen:    newWindowLimiter(agenBookingLimit, time.Hour),
		identitasGagal:   newWindowLimiter(identitasGagalAccountLimit, identitasGagalAccountWindow),
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

// Tanpa TURNSTILE_SECRET_KEY dan tanpa bypass pengembangan, booking DITOLAK (fail closed).
func TestCreateBookingCaptchaFailClosedTanpaSecret(t *testing.T) {
	h := newTestHandler(t, "", 0)
	h.captcha = &turnstileVerifier{}
	for _, body := range []string{validBookingBody, strings.Replace(validBookingBody, `"captcha_token": "token-uji",`, "", 1)} {
		rec := postBook(h, body, akunHandlerUji)
		if rec.Code != http.StatusServiceUnavailable && rec.Code != http.StatusBadRequest {
			t.Fatalf("got %d %s, want penolakan captcha", rec.Code, rec.Body.String())
		}
		if !strings.Contains(strings.ToLower(rec.Body.String()), "verifikasi keamanan") {
			t.Fatalf("bukan penolakan captcha: %s", rec.Body.String())
		}
	}
	t.Setenv("TURNSTILE_SECRET_KEY", "")
	for _, env := range [][2]string{{"", ""}, {"production", "true"}, {"development", ""}} {
		t.Setenv("APP_ENV", env[0])
		t.Setenv("ALLOW_DEV_CAPTCHA", env[1])
		if v := newTurnstileVerifier(); !v.required() {
			t.Fatalf("APP_ENV=%q ALLOW_DEV_CAPTCHA=%q: captcha tidak boleh dilewati", env[0], env[1])
		}
	}
	t.Setenv("APP_ENV", "development")
	t.Setenv("ALLOW_DEV_CAPTCHA", "true")
	if v := newTurnstileVerifier(); v.required() {
		t.Fatal("bypass pengembangan eksplisit harus melewati captcha")
	}
}

// K2: akun yang emailnya belum terverifikasi tidak boleh membuat booking (403 + kode stabil).
func TestCreateBookingWajibEmailTerverifikasi(t *testing.T) {
	h := newTestHandler(t, "", 0)
	dipanggil := false
	h.book = func(context.Context, int64, int64, BookingRequest) (*BookingResponse, error) {
		dipanggil = true
		return &BookingResponse{Status: "success"}, nil
	}
	belum := *akunHandlerUji
	belum.EmailVerifiedAt = nil
	rec := postBook(h, bodyCheckout, &belum)
	if rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), KodeEmailBelumTerverifikasi) || !strings.Contains(rec.Body.String(), "Verifikasi email") {
		t.Fatalf("akun belum terverifikasi: got %d %s, want 403 %s", rec.Code, rec.Body.String(), KodeEmailBelumTerverifikasi)
	}
	if dipanggil {
		t.Fatal("booking diproses untuk akun belum terverifikasi")
	}
	if rec = postBook(h, bodyCheckout, akunHandlerUji); rec.Code != http.StatusOK || !dipanggil {
		t.Fatalf("akun terverifikasi: got %d %s", rec.Code, rec.Body.String())
	}
}

// Butir 1: kegagalan validasi NIK/nomor milik jamaah lain dibatasi per akun.
func TestCreateBookingBatasPercobaanDataJamaah(t *testing.T) {
	h := newTestHandler(t, "", 0)
	n := 0
	h.book = func(context.Context, int64, int64, BookingRequest) (*BookingResponse, error) {
		n++
		return nil, errNIKBeda(2)
	}
	for i := 0; i < identitasGagalAccountLimit; i++ {
		if rec := postBook(h, bodyCheckout, akunHandlerUji); rec.Code != http.StatusBadRequest {
			t.Fatalf("percobaan %d: got %d %s", i+1, rec.Code, rec.Body.String())
		}
	}
	rec := postBook(h, bodyCheckout, akunHandlerUji)
	if rec.Code != http.StatusTooManyRequests || !strings.Contains(rec.Body.String(), KodeIdentitasTerlaluSering) {
		t.Fatalf("got %d %s, want 429", rec.Code, rec.Body.String())
	}
	if n != identitasGagalAccountLimit {
		t.Fatalf("diproses %d kali, want %d", n, identitasGagalAccountLimit)
	}
}

// K2: batas seat aktif dari repository dipetakan ke 429 dengan kode stabil.
func TestCreateBookingBatasSeatAkun(t *testing.T) {
	h := newTestHandler(t, "", 0)
	h.book = func(context.Context, int64, int64, BookingRequest) (*BookingResponse, error) {
		return nil, ErrBatasSeatAkun
	}
	rec := postBook(h, bodyCheckout, akunHandlerUji)
	if rec.Code != http.StatusTooManyRequests || !strings.Contains(rec.Body.String(), KodeBatasSeatAkun) {
		t.Fatalf("got %d %s", rec.Code, rec.Body.String())
	}
}

func TestCreateBookingSkipsCaptchaWhenDevBypass(t *testing.T) {
	// Bypass pengembangan eksplisit: captcha_token tidak diwajibkan; request lolos gate captcha
	// dan berhenti di batas booking per akun (repo nil tidak tersentuh).
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
