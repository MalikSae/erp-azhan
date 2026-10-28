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

// ErrCaptchaNotConfigured: dikembalikan Verify bila captcha dipaksa aktif tanpa secret (struct tanpa devBypass).
var ErrCaptchaNotConfigured = errors.New("captcha belum dikonfigurasi (TURNSTILE_SECRET_KEY kosong)")

// Turnstile memverifikasi token Cloudflare Turnstile di server. Captcha OPSIONAL (keputusan 2026-10-28):
// tanpa TURNSTILE_SECRET_KEY captcha nonaktif dan semua request lolos; isi secret untuk mengaktifkannya.
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
	t.devBypass = t.secret == ""
	return t
}

// ValidateCaptchaConfig dipanggil saat startup. Captcha opsional: tanpa secret hanya mencatat bahwa captcha nonaktif.
func ValidateCaptchaConfig() error {
	if strings.TrimSpace(os.Getenv("TURNSTILE_SECRET_KEY")) == "" {
		log.Println("[INFO] TURNSTILE_SECRET_KEY kosong: captcha NONAKTIF. Pendaftaran akun dan booking mandiri hanya dibatasi rate limit.")
	}
	return nil
}

// Enabled true bila captcha aktif (TURNSTILE_SECRET_KEY terisi).
func (t *Turnstile) Enabled() bool { return !(t.secret == "" && t.devBypass) }

// Verify mengembalikan true bila token valid atau captcha nonaktif. Error berarti Cloudflare tidak bisa
// dihubungi; pemanggil harus menolak.
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
