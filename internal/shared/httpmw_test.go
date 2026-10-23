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

// Butir 7h: token invoice di path dan token di query tidak tertulis ke log akses.
func TestRedactingLoggerMenyamarkanToken(t *testing.T) {
	token := strings.Repeat("ab", 32)
	var buf bytes.Buffer
	mw := RedactingLogger(&middleware.DefaultLogFormatter{Logger: log.New(&buf, "", 0), NoColor: true})
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, token) {
			t.Error("handler harus tetap menerima path asli")
		}
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/public/invoice/"+token+"?brand=1&token=rahasia-verifikasi", nil)
	h.ServeHTTP(httptest.NewRecorder(), req)
	out := buf.String()
	if strings.Contains(out, token) || strings.Contains(out, "rahasia-verifikasi") {
		t.Fatalf("log memuat token: %s", out)
	}
	if !strings.Contains(out, "/api/public/invoice/[disamarkan]") || !strings.Contains(out, "brand=1") {
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

// Butir 4: bypass captcha hanya untuk APP_ENV=development + ALLOW_DEV_CAPTCHA=true.
func TestTurnstileFailClosed(t *testing.T) {
	t.Setenv("TURNSTILE_SECRET_KEY", "")
	cases := []struct {
		env, allow string
		bypass     bool
	}{{"", "", false}, {"", "true", false}, {"production", "true", false}, {"development", "", false}, {"development", "true", true}, {"Development", "TRUE", true}}
	for _, c := range cases {
		t.Setenv("APP_ENV", c.env)
		t.Setenv("ALLOW_DEV_CAPTCHA", c.allow)
		ts := NewTurnstile()
		ok, err := ts.Verify(t.Context(), "", "127.0.0.1")
		if c.bypass != (ok && err == nil) || ts.Enabled() == c.bypass {
			t.Errorf("APP_ENV=%q ALLOW=%q: ok=%v err=%v enabled=%v, want bypass=%v", c.env, c.allow, ok, err, ts.Enabled(), c.bypass)
		}
	}
	t.Setenv("APP_ENV", "production")
	if ValidateCaptchaConfig() == nil {
		t.Fatal("produksi tanpa TURNSTILE_SECRET_KEY harus gagal start")
	}
	t.Setenv("TURNSTILE_SECRET_KEY", "secret-uji")
	if err := ValidateCaptchaConfig(); err != nil {
		t.Fatal(err)
	}
}
