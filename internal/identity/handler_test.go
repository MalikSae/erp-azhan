package identity

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestGetClientIPUsesForwardedClientBehindLoopbackProxy(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	req.RemoteAddr = "127.0.0.1:43210"
	req.Header.Set("CF-Connecting-IP", "203.0.113.10")
	req.Header.Set("X-Forwarded-For", "198.51.100.20, 172.68.1.1")

	if got := getClientIP(req); got != "203.0.113.10" {
		t.Fatalf("getClientIP() = %q, want %q", got, "203.0.113.10")
	}
}

func TestGetClientIPIgnoresSpoofedHeadersFromDirectClient(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	req.RemoteAddr = "198.51.100.8:43210"
	req.Header.Set("CF-Connecting-IP", "203.0.113.10")

	if got := getClientIP(req); got != "198.51.100.8" {
		t.Fatalf("getClientIP() = %q, want direct client IP", got)
	}
}

func TestLoginRateLimitReturnsServerCountdown(t *testing.T) {
	handler := NewHandler(nil)
	key := "203.0.113.10|admin@hana.id"
	handler.failedLogins[key] = &loginAttempt{
		Count:     loginAttemptLimit,
		FirstFail: time.Now().Add(-time.Minute),
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/auth/login",
		strings.NewReader(`{"email":"admin@hana.id","password":"invalid-password"}`),
	)
	req.RemoteAddr = "127.0.0.1:43210"
	req.Header.Set("CF-Connecting-IP", "203.0.113.10")
	recorder := httptest.NewRecorder()

	handler.Login(recorder, req)

	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusTooManyRequests)
	}
	if recorder.Header().Get("Retry-After") == "" {
		t.Fatal("Retry-After header is empty")
	}

	var body struct {
		RetryAfterSeconds int `json:"retry_after_seconds"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.RetryAfterSeconds <= 0 || body.RetryAfterSeconds > int(loginAttemptWindow.Seconds()) {
		t.Fatalf("retry_after_seconds = %d, want within login window", body.RetryAfterSeconds)
	}
}
