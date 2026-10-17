package shared

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

// Turnstile memverifikasi token Cloudflare Turnstile di server. Opsional: aktif
// hanya jika TURNSTILE_SECRET_KEY diisi; bila kosong verifikasi dilewati dan
// endpoint tetap dibatasi rate limit.
type Turnstile struct {
	secret    string
	verifyURL string
	client    *http.Client
}

func NewTurnstile() *Turnstile {
	t := &Turnstile{
		secret:    strings.TrimSpace(os.Getenv("TURNSTILE_SECRET_KEY")),
		verifyURL: turnstileVerifyURL,
		client:    &http.Client{Timeout: 5 * time.Second},
	}
	if t.secret == "" {
		log.Println("[INFO] TURNSTILE_SECRET_KEY kosong: verifikasi Turnstile nonaktif (rate limit tetap berlaku)")
	}
	return t
}

func (t *Turnstile) Enabled() bool { return t.secret != "" }

// Verify mengembalikan true jika token valid atau Turnstile nonaktif. Error berarti
// Cloudflare tidak bisa dihubungi saat aktif; pemanggil harus menolak (fail closed).
func (t *Turnstile) Verify(ctx context.Context, token, remoteIP string) (bool, error) {
	if !t.Enabled() {
		return true, nil
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
