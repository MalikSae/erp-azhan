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
)

const turnstileVerifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

// turnstileVerifier memverifikasi token Cloudflare Turnstile di server.
// Verifikasi aktif hanya jika TURNSTILE_SECRET_KEY diisi; jika kosong, token
// hanya dicek tidak kosong (mode pengembangan) dan peringatan dicetak saat start.
type turnstileVerifier struct {
	secret    string
	verifyURL string
	client    *http.Client
}

func newTurnstileVerifier() *turnstileVerifier {
	v := &turnstileVerifier{
		secret:    strings.TrimSpace(os.Getenv("TURNSTILE_SECRET_KEY")),
		verifyURL: turnstileVerifyURL,
		client:    &http.Client{Timeout: 5 * time.Second},
	}
	if v.secret == "" {
		log.Println("[WARN] TURNSTILE_SECRET_KEY kosong: verifikasi captcha self-booking NONAKTIF")
	}
	return v
}

func (v *turnstileVerifier) enabled() bool {
	return v.secret != ""
}

// verify mengembalikan true jika token valid. Error berarti Cloudflare tidak
// bisa dihubungi; pemanggil harus menolak request (fail closed).
func (v *turnstileVerifier) verify(ctx context.Context, token, remoteIP string) (bool, error) {
	if !v.enabled() {
		return true, nil
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
