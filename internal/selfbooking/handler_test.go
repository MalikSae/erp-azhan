package selfbooking

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const validBookingBody = `{
	"captcha_token": "token-uji",
	"brand_id": 2,
	"schedule_id": 10,
	"pic": {"nama_lengkap": "Uji", "no_hp": "0812-3456-7890", "jenis_kelamin": "L", "room_type": "Quad"}
}`

func newTestHandler(t *testing.T, siteverifyBody string, siteverifyStatus int) *Handler {
	t.Helper()
	h := &Handler{
		captcha:        &turnstileVerifier{},
		phoneCheckByIP: newWindowLimiter(phoneCheckIPLimit, 15*time.Minute),
		bookingByIP:    newWindowLimiter(bookingIPAttemptLimit, time.Hour),
		bookingByPhone: newWindowLimiter(bookingPhoneSuccessLimit, 24*time.Hour),
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

func postBook(h *Handler, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/public/book", strings.NewReader(body))
	req.RemoteAddr = "198.51.100.9:5555"
	rec := httptest.NewRecorder()
	h.CreateBooking(rec, req)
	return rec
}

func TestCreateBookingRejectsInvalidCaptcha(t *testing.T) {
	h := newTestHandler(t, `{"success":false,"error-codes":["invalid-input-response"]}`, http.StatusOK)
	rec := postBook(h, validBookingBody)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "Verifikasi keamanan gagal") {
		t.Fatalf("got %d %s, want 400 captcha error", rec.Code, rec.Body.String())
	}
}

func TestCreateBookingFailsClosedWhenSiteverifyDown(t *testing.T) {
	h := newTestHandler(t, `oops`, http.StatusInternalServerError)
	rec := postBook(h, validBookingBody)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("got %d %s, want 503", rec.Code, rec.Body.String())
	}
}

func TestCreateBookingIPLimit(t *testing.T) {
	h := newTestHandler(t, "", 0)
	for i := 1; i <= bookingIPAttemptLimit; i++ {
		if rec := postBook(h, `bukan-json`); rec.Code != http.StatusBadRequest {
			t.Fatalf("percobaan #%d: got %d, want 400", i, rec.Code)
		}
	}
	rec := postBook(h, `bukan-json`)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("percobaan #%d: got %d %s, want 429", bookingIPAttemptLimit+1, rec.Code, rec.Body.String())
	}
}

func TestCreateBookingPhoneLimitBeforeProcessing(t *testing.T) {
	// Captcha valid; nomor sudah mencapai batas, jadi request harus berhenti
	// sebelum menyentuh repository (repo nil akan panic jika tersentuh).
	h := newTestHandler(t, `{"success":true}`, http.StatusOK)
	key := "brand:2|6281234567890"
	for i := 0; i < bookingPhoneSuccessLimit; i++ {
		h.bookingByPhone.record(key)
	}
	rec := postBook(h, validBookingBody)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("got %d %s, want 429", rec.Code, rec.Body.String())
	}
}

func TestCheckPhoneCountsEveryCheck(t *testing.T) {
	// Semua percobaan dihitung, apa pun hasilnya. Body tanpa brand_id ditolak
	// sebelum menyentuh repository, jadi repo nil aman di sini.
	h := newTestHandler(t, "", 0)
	post := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/public/jamaah/check", strings.NewReader(`{"no_hp":"0812"}`))
		req.RemoteAddr = "198.51.100.9:5555"
		rec := httptest.NewRecorder()
		h.CheckPhone(rec, req)
		return rec
	}
	for i := 1; i <= phoneCheckIPLimit; i++ {
		if rec := post(); rec.Code != http.StatusBadRequest {
			t.Fatalf("percobaan #%d: got %d, want 400", i, rec.Code)
		}
	}
	if rec := post(); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("percobaan #%d: got %d %s, want 429", phoneCheckIPLimit+1, rec.Code, rec.Body.String())
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
