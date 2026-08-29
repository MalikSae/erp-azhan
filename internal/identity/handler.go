package identity

import (
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type loginAttempt struct {
	Count     int
	FirstFail time.Time
}

type Handler struct {
	repo          *Repository
	failedLogins  map[string]*loginAttempt
	failedLoginMu sync.Mutex
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{
		repo:         repo,
		failedLogins: make(map[string]*loginAttempt),
	}
}

// ─── Rate Limiting ────────────────────────────────────────────────────────────

func (h *Handler) checkRateLimit(ip string) bool {
	h.failedLoginMu.Lock()
	defer h.failedLoginMu.Unlock()

	attempt, exists := h.failedLogins[ip]
	if !exists {
		return true
	}

	if time.Since(attempt.FirstFail) > 15*time.Minute {
		// Reset setelah 15 menit
		delete(h.failedLogins, ip)
		return true
	}

	return attempt.Count < 10
}

func (h *Handler) recordFailedLogin(ip string) {
	h.failedLoginMu.Lock()
	defer h.failedLoginMu.Unlock()

	attempt, exists := h.failedLogins[ip]
	if !exists || time.Since(attempt.FirstFail) > 15*time.Minute {
		h.failedLogins[ip] = &loginAttempt{
			Count:     1,
			FirstFail: time.Now(),
		}
		return
	}
	attempt.Count++
}

func (h *Handler) recordSuccessLogin(ip string) {
	h.failedLoginMu.Lock()
	defer h.failedLoginMu.Unlock()
	delete(h.failedLogins, ip)
}

func getClientIP(r *http.Request) string {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// ─── Login ────────────────────────────────────────────────────────────────────

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	ip := getClientIP(r)

	// Cek rate limit sebelum parse atau query DB
	if !h.checkRateLimit(ip) {
		writeError(w, http.StatusTooManyRequests, "terlalu banyak percobaan, coba lagi nanti")
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}

	// Email akun disimpan dalam bentuk lowercase. Normalisasi juga saat login
	// supaya kredensial tetap bekerja walau pengguna mengetik huruf kapital.
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "email dan password wajib diisi")
		return
	}

	user, err := h.repo.GetByEmail(r.Context(), req.Email)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			h.recordFailedLogin(ip)
			writeError(w, http.StatusUnauthorized, "email atau password salah")
			return
		}
		log.Printf("[ERROR] identity.Login GetByEmail: %v", err)
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan internal")
		return
	}
	if !user.IsActive {
		h.recordFailedLogin(ip)
		writeError(w, http.StatusUnauthorized, "email atau password salah")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		h.recordFailedLogin(ip)
		writeError(w, http.StatusUnauthorized, "email atau password salah")
		return
	}

	// Sukses
	h.recordSuccessLogin(ip)

	accessToken, err := GenerateAccessToken(user.ID, user.BrandID, user.Role)
	if err != nil {
		log.Printf("[ERROR] identity.Login GenerateAccessToken: %v", err)
		writeError(w, http.StatusInternalServerError, "gagal membuat access token")
		return
	}

	refreshToken, err := GenerateRefreshToken(user.ID)
	if err != nil {
		log.Printf("[ERROR] identity.Login GenerateRefreshToken: %v", err)
		writeError(w, http.StatusInternalServerError, "gagal membuat refresh token")
		return
	}

	ttl := int(getAccessTTL().Seconds())

	writeJSON(w, http.StatusOK, TokenResponse{
		UserID:       user.ID,
		DisplayName:  user.DisplayName,
		Role:         user.Role,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    ttl,
		TokenType:    "Bearer",
	})
}

// ─── Refresh ──────────────────────────────────────────────────────────────────

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}

	if strings.TrimSpace(req.RefreshToken) == "" {
		writeError(w, http.StatusBadRequest, "refresh_token wajib diisi")
		return
	}

	adminUserID, _, _, err := ValidateToken(req.RefreshToken, "refresh")
	if err != nil {
		writeError(w, http.StatusUnauthorized, "refresh token tidak valid")
		return
	}

	user, err := h.repo.GetByID(r.Context(), adminUserID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusUnauthorized, "user tidak ditemukan")
			return
		}
		log.Printf("[ERROR] identity.Refresh GetByID: %v", err)
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan internal")
		return
	}

	if !user.IsActive {
		writeError(w, http.StatusUnauthorized, "user tidak ditemukan")
		return
	}

	accessToken, err := GenerateAccessToken(user.ID, user.BrandID, user.Role)
	if err != nil {
		log.Printf("[ERROR] identity.Refresh GenerateAccessToken: %v", err)
		writeError(w, http.StatusInternalServerError, "gagal membuat access token")
		return
	}

	ttl := int(getAccessTTL().Seconds())

	writeJSON(w, http.StatusOK, TokenResponse{
		UserID:      user.ID,
		DisplayName: user.DisplayName,
		Role:        user.Role,
		AccessToken: accessToken,
		ExpiresIn:   ttl,
		TokenType:   "Bearer",
	})
}

// ─── Logout ───────────────────────────────────────────────────────────────────

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	// Limitasi sementara: karena token stateless, server tidak menyimpan token blacklist
	// Invalidasi sesungguhnya adalah tanggung jawab client (hapus token dari storage)
	writeJSON(w, http.StatusOK, map[string]string{"message": "logout berhasil"})
}

// ─── Shared ───────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
