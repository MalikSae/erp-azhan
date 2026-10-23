package identity

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const secretUji = "test-secret-minimum-32-characters-long"

// Butir 2: JWT_SECRET selalu divalidasi saat start.
func TestValidateJWTSecretMenolakSecretLemah(t *testing.T) {
	for _, s := range []string{"", "   ", "pendek", "your_super_secret_jwt_key_here_minimum_32_chars", "  YOUR_SUPER_SECRET_JWT_KEY_HERE_MINIMUM_32_CHARS ",
		"your_own_random_value_that_is_long_enough_1234", "changemechangemechangemechangeme12"} {
		t.Setenv("JWT_SECRET", s)
		if ValidateJWTSecret() == nil {
			t.Errorf("JWT_SECRET %q seharusnya ditolak", s)
		}
	}
	t.Setenv("JWT_SECRET", "nilai-uji-acak-bukan-rahasia-0123456789-uji")
	if err := ValidateJWTSecret(); err != nil {
		t.Fatalf("secret acak ditolak: %v", err)
	}
}

func signUji(t *testing.T, method jwt.SigningMethod, key any, claims jwt.MapClaims) string {
	t.Helper()
	s, err := jwt.NewWithClaims(method, claims).SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// Butir 2: verifikasi dikunci ke HS256 dan issuer API ini; token lama tanpa iss tetap sah.
func TestValidateTokenHanyaHS256DanIssuer(t *testing.T) {
	t.Setenv("JWT_SECRET", secretUji)
	base := func() jwt.MapClaims {
		return jwt.MapClaims{"sub": 42, "type": "access", "exp": time.Now().Add(time.Minute).Unix(), "iat": time.Now().Unix()}
	}
	baru, err := GenerateAccessToken(42, nil, "admin")
	if err != nil {
		t.Fatal(err)
	}
	if c, _, err := jwt.NewParser().ParseUnverified(baru, jwt.MapClaims{}); err != nil || c.Claims.(jwt.MapClaims)["iss"] != jwtIssuer {
		t.Fatalf("token baru tanpa iss %q: %v", jwtIssuer, err)
	}
	if _, err := ValidateAccessClaims(baru); err != nil {
		t.Fatalf("token baru ditolak: %v", err)
	}
	if _, err := ValidateAccessClaims(signUji(t, jwt.SigningMethodHS256, []byte(secretUji), base())); err != nil {
		t.Fatalf("token lama tanpa iss harus tetap sah sampai kedaluwarsa: %v", err)
	}
	asing := base()
	asing["iss"] = "layanan-lain"
	ditolak := map[string]string{
		"HS512":    signUji(t, jwt.SigningMethodHS512, []byte(secretUji), base()),
		"HS384":    signUji(t, jwt.SigningMethodHS384, []byte(secretUji), base()),
		"none":     signUji(t, jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, base()),
		"iss-lain": signUji(t, jwt.SigningMethodHS256, []byte(secretUji), asing),
	}
	for nama, tok := range ditolak {
		if _, err := ValidateAccessClaims(tok); err == nil {
			t.Errorf("token %s diterima", nama)
		}
		if _, _, _, err := ValidateToken(tok, "access"); err == nil {
			t.Errorf("ValidateToken menerima token %s", nama)
		}
	}
}

func TestAccessTTLDibatasi60Menit(t *testing.T) {
	t.Setenv("JWT_ACCESS_TTL_MINUTES", "1440")
	if got := getAccessTTL(); got != 60*time.Minute {
		t.Fatalf("TTL = %v, want 60m", got)
	}
	t.Setenv("JWT_ACCESS_TTL_MINUTES", "15")
	if got := getAccessTTL(); got != 15*time.Minute {
		t.Fatalf("TTL = %v, want 15m", got)
	}
}

// Butir 7e: RequireAuth menolak akun nonaktif dan token dengan perm_version usang tanpa menunggu
// access token kedaluwarsa; cache tidak memblokir token baru setelah refresh.
func TestRequireAuthCekStatusUser(t *testing.T) {
	t.Setenv("JWT_SECRET", secretUji)
	status := map[int64]struct {
		active bool
		ver    int
	}{1: {true, 3}}
	calls := 0
	SetUserStatusLoader(func(_ context.Context, id int64) (bool, int, error) {
		calls++
		s := status[id]
		return s.active, s.ver, nil
	})
	t.Cleanup(func() { SetUserStatusLoader(nil) })

	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	run := func(pver int) int {
		tok, err := GenerateAccessTokenWithAccess(1, nil, "admin", "", nil, []string{"x"}, pver)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		req.Header.Set("Authorization", "Bearer "+tok)
		rec := httptest.NewRecorder()
		RequireAuth(ok).ServeHTTP(rec, req)
		return rec.Code
	}
	if c := run(3); c != http.StatusOK {
		t.Fatalf("token sah: %d", c)
	}
	if c := run(2); c != http.StatusUnauthorized {
		t.Fatalf("pver usang: %d, want 401", c)
	}
	// Izin berubah (pver 4) lalu klien refresh: cache (pver 3) dimuat ulang, token baru lolos.
	status[1] = struct {
		active bool
		ver    int
	}{true, 4}
	if c := run(4); c != http.StatusOK {
		t.Fatalf("token baru setelah refresh: %d, want 200", c)
	}
	// Akun dinonaktifkan: berlaku setelah cache kedaluwarsa (paksa dengan menghapus cache).
	status[1] = struct {
		active bool
		ver    int
	}{false, 5}
	statusCache.mu.Lock()
	statusCache.entries = map[int64]userStatus{}
	statusCache.mu.Unlock()
	if c := run(4); c != http.StatusUnauthorized {
		t.Fatalf("akun nonaktif: %d, want 401", c)
	}
	if calls == 0 {
		t.Fatal("loader tidak dipanggil")
	}
}

// Butir 7d: batas per IP lintas email, body dibatasi 16 KB, map gagal login dibersihkan.
func TestLoginBatasPerIPDanBody(t *testing.T) {
	h := NewHandler(nil)
	for i := 0; i < loginIPAttemptLimit; i++ {
		h.loginIP.Record("198.51.100.7")
	}
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"email":"lain@hana.id","password":"x"}`))
	req.RemoteAddr = "198.51.100.7:1234"
	rec := httptest.NewRecorder()
	h.Login(rec, req) // repo nil: harus berhenti sebelum database
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("batas per IP: %d, want 429", rec.Code)
	}

	big := `{"email":"a@b.id","password":"` + strings.Repeat("x", authMaxBodyBytes) + `"}`
	req = httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(big))
	req.RemoteAddr = "198.51.100.8:1234"
	rec = httptest.NewRecorder()
	h.Login(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("body > 16 KB: %d, want 400", rec.Code)
	}

	h.failedLogins["lama"] = &loginAttempt{Count: 1, FirstFail: time.Now().Add(-2 * loginAttemptWindow)}
	h.lastPrune = time.Time{}
	h.recordFailedLogin("baru")
	if _, ada := h.failedLogins["lama"]; ada {
		t.Fatal("entri kedaluwarsa tidak dibersihkan")
	}
}
