package shared

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const turnstileVerifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

// ErrCaptchaNotConfigured: TURNSTILE_SECRET_KEY kosong di luar mode pengembangan eksplisit.
// Pemanggil menolak request (fail closed).
var ErrCaptchaNotConfigured = errors.New("captcha belum dikonfigurasi (TURNSTILE_SECRET_KEY kosong)")

// Turnstile memverifikasi token Cloudflare Turnstile di server. Fail closed (tinjauan keamanan
// 2026-10-22): tanpa TURNSTILE_SECRET_KEY verifikasi gagal, kecuali APP_ENV=development dan
// ALLOW_DEV_CAPTCHA=true. Di produksi startup gagal bila secret kosong (ValidateCaptchaConfig).
type Turnstile struct {
	secret    string
	verifyURL string
	client    *http.Client
	devBypass bool
}

func NewTurnstile() *Turnstile {
	t := &Turnstile{
		secret:    strings.TrimSpace(os.Getenv("TURNSTILE_SECRET_KEY")),
		verifyURL: turnstileVerifyURL,
		client:    &http.Client{Timeout: 5 * time.Second},
	}
	if t.secret == "" {
		t.devBypass = DevCaptchaBypass()
	}
	return t
}

// ValidateCaptchaConfig dipanggil saat startup. Produksi tanpa secret: error (startup gagal).
// Di luar produksi tanpa secret: peringatan keras; request bercaptcha ditolak kecuali bypass dev.
func ValidateCaptchaConfig() error {
	if strings.TrimSpace(os.Getenv("TURNSTILE_SECRET_KEY")) != "" {
		return nil
	}
	if IsProduction() {
		return errors.New("TURNSTILE_SECRET_KEY wajib diisi saat APP_ENV=production")
	}
	if DevCaptchaBypass() {
		log.Println("[WARN] TURNSTILE_SECRET_KEY kosong: captcha DILEWATI (APP_ENV=development, ALLOW_DEV_CAPTCHA=true). Jangan dipakai di server publik.")
		return nil
	}
	log.Println("[WARN] TURNSTILE_SECRET_KEY kosong: pendaftaran akun dan booking mandiri DITOLAK. Isi secret, atau untuk pengembangan lokal set APP_ENV=development dan ALLOW_DEV_CAPTCHA=true.")
	return nil
}

// Enabled true bila captcha wajib dikirim klien (semua mode kecuali bypass pengembangan).
func (t *Turnstile) Enabled() bool { return !(t.secret == "" && t.devBypass) }

// Verify mengembalikan true hanya jika token valid, atau bypass pengembangan aktif. Error berarti
// captcha belum dikonfigurasi atau Cloudflare tidak bisa dihubungi; pemanggil harus menolak.
func (t *Turnstile) Verify(ctx context.Context, token, remoteIP string) (bool, error) {
	if t.secret == "" {
		if t.devBypass {
			return true, nil
		}
		return false, ErrCaptchaNotConfigured
	}
	form := url.Values{}
	form.Set("secret", t.secret)
	form.Set("response", token)
	if remoteIP != "" {
		form.Set("remoteip", remoteIP)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.verifyURL, strings.NewReader(form.Encode()))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := t.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("turnstile siteverify: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("turnstile siteverify: status %d", resp.StatusCode)
	}
	var result struct {
		Success    bool     `json:"success"`
		ErrorCodes []string `json:"error-codes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, fmt.Errorf("turnstile siteverify decode: %w", err)
	}
	if !result.Success {
		log.Printf("[AUDIT] captcha_rejected ip=%s codes=%v", remoteIP, result.ErrorCodes)
	}
	return result.Success, nil
}
