package shared

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5/middleware"
)

// Butir 7h: token di query tidak tertulis ke log akses.
func TestRedactingLoggerMenyamarkanToken(t *testing.T) {
	token := "rahasia-verifikasi"
	var buf bytes.Buffer
	mw := RedactingLogger(&middleware.DefaultLogFormatter{Logger: log.New(&buf, "", 0), NoColor: true})
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("token") != token {
			t.Error("handler harus tetap menerima query asli")
		}
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/public/auth/verify-email?brand=1&token="+token, nil)
	h.ServeHTTP(httptest.NewRecorder(), req)
	out := buf.String()
	if strings.Contains(out, token) {
		t.Fatalf("log memuat token: %s", out)
	}
	if !strings.Contains(out, "brand=1") {
		t.Fatalf("log tidak informatif: %s", out)
	}
	if got := RedactURI("/api/schedules?brand=2"); got != "/api/schedules?brand=2" {
		t.Fatalf("URI tanpa token berubah: %s", got)
	}
}

// Butir 7g: Cache-Control publik hanya untuk GET 200.
func TestPublicCacheHanyaUntuk200(t *testing.T) {
	mw := PublicCache(60)
	for _, tc := range []struct {
		method string
		status int
		want   string
	}{
		{http.MethodGet, http.StatusOK, "public, max-age=60"},
		{http.MethodGet, http.StatusBadRequest, ""},
		{http.MethodGet, http.StatusInternalServerError, ""},
		{http.MethodPost, http.StatusOK, ""},
	} {
		h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(tc.status)
			_, _ = w.Write([]byte("{}"))
		}))
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(tc.method, "/api/public/v1/packages?brand=1", nil))
		if got := rec.Header().Get("Cache-Control"); got != tc.want {
			t.Errorf("%s %d: Cache-Control=%q, want %q", tc.method, tc.status, got, tc.want)
		}
	}
	// Tanpa WriteHeader eksplisit (Write langsung) tetap 200 dan ter-cache.
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("[]")) }))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/public/v1/destinations?brand=1", nil))
	if rec.Header().Get("Cache-Control") != "public, max-age=60" {
		t.Fatalf("Write tanpa WriteHeader: %q", rec.Header().Get("Cache-Control"))
	}
}

// Captcha opsional (keputusan 2026-10-28): tanpa TURNSTILE_SECRET_KEY nonaktif di semua APP_ENV; dengan secret aktif.
func TestTurnstileOpsional(t *testing.T) {
	t.Setenv("TURNSTILE_SECRET_KEY", "")
	for _, env := range []string{"", "production", "development"} {
		t.Setenv("APP_ENV", env)
		ts := NewTurnstile()
		ok, err := ts.Verify(t.Context(), "", "127.0.0.1")
		if !ok || err != nil || ts.Enabled() {
			t.Errorf("APP_ENV=%q: ok=%v err=%v enabled=%v, want lolos dan nonaktif", env, ok, err, ts.Enabled())
		}
		if err := ValidateCaptchaConfig(); err != nil {
			t.Fatalf("APP_ENV=%q: startup tidak boleh gagal tanpa secret: %v", env, err)
		}
	}
	t.Setenv("TURNSTILE_SECRET_KEY", "secret-uji")
	if !NewTurnstile().Enabled() {
		t.Fatal("dengan secret captcha harus aktif")
	}
	if err := ValidateCaptchaConfig(); err != nil {
		t.Fatal(err)
	}
}
