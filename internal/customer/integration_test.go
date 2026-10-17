package customer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"erp-azhan/api/internal/mailer"
	"erp-azhan/api/internal/testdb"
)

type captureMail struct {
	mu   sync.Mutex
	msgs []mailer.Message
}

func (c *captureMail) Send(_ context.Context, m mailer.Message) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.msgs = append(c.msgs, m)
	return nil
}

func (c *captureMail) last(to string) (mailer.Message, bool) {
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		c.mu.Lock()
		for i := len(c.msgs) - 1; i >= 0; i-- {
			if c.msgs[i].To == to {
				m := c.msgs[i]
				c.mu.Unlock()
				return m, true
			}
		}
		c.mu.Unlock()
		time.Sleep(20 * time.Millisecond)
	}
	return mailer.Message{}, false
}

func tokenFrom(t *testing.T, text string) string {
	t.Helper()
	i := strings.Index(text, "token=")
	if i < 0 {
		t.Fatalf("token tidak ada di email: %s", text)
	}
	rest := text[i+len("token="):]
	if j := strings.IndexAny(rest, " \n\r"); j >= 0 {
		rest = rest[:j]
	}
	return rest
}

func post(h http.HandlerFunc, body any, token string) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(b))
	req.RemoteAddr = "203.0.113.9:1234"
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h(rec, req)
	return rec
}

func TestAccountLifecycle(t *testing.T) {
	db := testdb.DB(t)
	ctx := context.Background()

	var brandID int64
	if err := db.QueryRowContext(ctx, `SELECT id FROM brands ORDER BY id LIMIT 1`).Scan(&brandID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT IGNORE INTO brand_domains (brand_id, domain) VALUES (?, 'akun-test.azhan.test')`, brandID); err != nil {
		t.Fatal(err)
	}
	email := fmt.Sprintf("uji-%d@example.com", time.Now().UnixNano())
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM customer_accounts WHERE email_normal=?`, email)
		_, _ = db.Exec(`DELETE FROM brand_domains WHERE domain='akun-test.azhan.test'`)
	})

	mail := &captureMail{}
	h := NewHandler(NewRepository(db), mail)
	t.Setenv("SITE_URL_TEMPLATE", "http://{domain}")

	// Daftar
	rec := post(h.Register, map[string]any{"brand_id": brandID, "nama_lengkap": "Budi Uji", "email": email, "no_hp": "081234567890", "password": "rahasia-123"}, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("register: %d %s", rec.Code, rec.Body)
	}
	var reg authResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &reg)
	if reg.SessionToken == "" || reg.Account.EmailVerifiedAt != nil {
		t.Fatalf("register harus memberi sesi dan email belum terverifikasi: %+v", reg)
	}

	// Email sama (huruf besar) ditolak
	if rec = post(h.Register, map[string]any{"brand_id": brandID, "nama_lengkap": "Budi Dua", "email": strings.ToUpper(email), "no_hp": "081234567891", "password": "rahasia-123"}, ""); rec.Code != http.StatusConflict {
		t.Fatalf("email ganda harus 409, dapat %d", rec.Code)
	}

	// Verifikasi email lewat tautan di email
	msg, ok := mail.last(email)
	if !ok {
		t.Fatal("email verifikasi tidak terkirim")
	}
	verifyToken := tokenFrom(t, msg.Text)
	if rec = post(h.VerifyEmail, map[string]string{"token": verifyToken}, ""); rec.Code != http.StatusOK {
		t.Fatalf("verify: %d %s", rec.Code, rec.Body)
	}
	if rec = post(h.VerifyEmail, map[string]string{"token": verifyToken}, ""); rec.Code != http.StatusBadRequest {
		t.Fatal("token verifikasi tidak boleh bisa dipakai dua kali")
	}
	acc, err := h.repo.AccountBySession(ctx, reg.SessionToken)
	if err != nil || acc.EmailVerifiedAt == nil {
		t.Fatalf("akun harus terverifikasi: %v %+v", err, acc)
	}

	// Login: salah vs benar
	if rec = post(h.Login, map[string]any{"brand_id": brandID, "email": email, "password": "salah-salah"}, ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("password salah harus 401, dapat %d", rec.Code)
	}
	rec = post(h.Login, map[string]any{"brand_id": brandID, "email": strings.ToUpper(email), "password": "rahasia-123"}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("login: %d %s", rec.Code, rec.Body)
	}
	var login authResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &login)

	// Lupa password -> reset -> sesi lama mati, password baru berlaku
	if rec = post(h.ForgotPassword, map[string]any{"brand_id": brandID, "email": email}, ""); rec.Code != http.StatusOK {
		t.Fatalf("forgot: %d", rec.Code)
	}
	var resetMsg mailer.Message
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if m, ok := mail.last(email); ok && strings.Contains(m.Subject, "Atur ulang") {
			resetMsg = m
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if resetMsg.Text == "" {
		t.Fatal("email reset tidak terkirim")
	}
	if rec = post(h.ResetPassword, map[string]string{"token": tokenFrom(t, resetMsg.Text), "password": "baru-12345"}, ""); rec.Code != http.StatusOK {
		t.Fatalf("reset: %d %s", rec.Code, rec.Body)
	}
	if _, err = h.repo.AccountBySession(ctx, login.SessionToken); err == nil {
		t.Fatal("sesi lama harus dicabut setelah reset password")
	}
	if rec = post(h.Login, map[string]any{"brand_id": brandID, "email": email, "password": "rahasia-123"}, ""); rec.Code != http.StatusUnauthorized {
		t.Fatal("password lama tidak boleh berlaku")
	}
	if rec = post(h.Login, map[string]any{"brand_id": brandID, "email": email, "password": "baru-12345"}, ""); rec.Code != http.StatusOK {
		t.Fatal("password baru harus berlaku")
	}

	// Forgot untuk email tak dikenal: jawaban sama, tidak membocorkan
	if rec = post(h.ForgotPassword, map[string]any{"brand_id": brandID, "email": "tidak-ada@example.com"}, ""); rec.Code != http.StatusOK {
		t.Fatalf("forgot email tak dikenal harus tetap 200, dapat %d", rec.Code)
	}
}

func TestLoginRateLimit(t *testing.T) {
	db := testdb.DB(t)
	h := NewHandler(NewRepository(db), &captureMail{})
	for i := 0; i < loginKeyLimit; i++ {
		if rec := post(h.Login, map[string]any{"brand_id": 1, "email": "x@example.com", "password": "salah-salah"}, ""); rec.Code != http.StatusUnauthorized {
			t.Fatalf("percobaan %d: %d", i+1, rec.Code)
		}
	}
	if rec := post(h.Login, map[string]any{"brand_id": 1, "email": "x@example.com", "password": "salah-salah"}, ""); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("percobaan ke-%d harus 429, dapat %d", loginKeyLimit+1, rec.Code)
	}
}
