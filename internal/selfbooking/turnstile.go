package selfbooking

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"erp-azhan/api/internal/shared"
)

const turnstileVerifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

// turnstileVerifier memverifikasi token Cloudflare Turnstile di server.
// Captcha OPSIONAL (keputusan 2026-10-28): tanpa TURNSTILE_SECRET_KEY captcha nonaktif dan semua
// request lolos; isi secret untuk mengaktifkannya.
type turnstileVerifier struct {
	secret    string
	verifyURL string
	client    *http.Client
	devBypass bool // true bila secret kosong (captcha nonaktif); tes mengisinya langsung
}

func newTurnstileVerifier() *turnstileVerifier {
	v := &turnstileVerifier{
		secret:    strings.TrimSpace(os.Getenv("TURNSTILE_SECRET_KEY")),
		verifyURL: turnstileVerifyURL,
		client:    &http.Client{Timeout: 5 * time.Second},
	}
	v.devBypass = v.secret == ""
	return v
}

// required: token captcha wajib dikirim (hanya bila captcha aktif).
func (v *turnstileVerifier) required() bool {
	return !(v.secret == "" && v.devBypass)
}

// verify mengembalikan true bila token valid atau captcha nonaktif. Error berarti Cloudflare tidak
// bisa dihubungi; pemanggil harus menolak request.
func (v *turnstileVerifier) verify(ctx context.Context, token, remoteIP string) (bool, error) {
	if v.secret == "" {
		if v.devBypass {
			return true, nil
		}
		return false, shared.ErrCaptchaNotConfigured
	}

	form := url.Values{}
	form.Set("secret", v.secret)
	form.Set("response", token)
	if remoteIP != "" {
		form.Set("remoteip", remoteIP)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, v.verifyURL, strings.NewReader(form.Encode()))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := v.client.Do(req)
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
		log.Printf("[AUDIT] selfbooking captcha_rejected ip=%s codes=%v", remoteIP, result.ErrorCodes)
	}
	return result.Success, nil
}
