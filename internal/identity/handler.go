package identity

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"erp-azhan/api/internal/shared"
	"golang.org/x/crypto/bcrypt"
)

type loginAttempt struct {
	Count     int
	FirstFail time.Time
}

const (
	loginAttemptLimit  = 10
	loginAttemptWindow = 15 * time.Minute
	// Gagal login per IP lintas email (menahan password spraying dari satu IP).
	loginIPAttemptLimit = 30
	// Body /api/auth/* hanya JSON kecil (email, password, refresh token).
	authMaxBodyBytes = 16 << 10
)

// UserAccess hasil resolusi RBAC untuk satu user (lihat internal/rbac).
type UserAccess struct {
	Roles       []string
	Permissions []string
	PermVersion int
}

// AccessLoader dipenuhi rbac.Repository lewat adaptor di main.go. Gagal/nil
// loader tidak memblokir login: klaim RBAC dikosongkan (fallback transisi).
type AccessLoader func(ctx context.Context, adminUserID int64) (*UserAccess, error)

type Handler struct {
	repo          *Repository
	loadAccess    AccessLoader
	failedLogins  map[string]*loginAttempt
	failedLoginMu sync.Mutex
	lastPrune     time.Time
	loginIP       *shared.WindowLimiter
	// dummyHash dibandingkan untuk email tak dikenal / akun nonaktif agar waktu jawab sama
	// dengan akun aktif (tanpa timing oracle keberadaan email admin).
	dummyHash []byte
}

func NewHandler(repo *Repository, loaders ...AccessLoader) *Handler {
	dummy, _ := bcrypt.GenerateFromPassword([]byte("dummy-password-for-timing"), bcrypt.DefaultCost)
	h := &Handler{
		repo:         repo,
		failedLogins: make(map[string]*loginAttempt),
		loginIP:      shared.NewWindowLimiter(loginIPAttemptLimit, loginAttemptWindow),
		dummyHash:    dummy,
	}
	if len(loaders) > 0 {
		h.loadAccess = loaders[0]
	}
	return h
}

// resolveAccess memuat roles/perms user; error hanya dicatat supaya login tetap
// berjalan bila tabel RBAC belum termigrasi di sebuah environment.
func (h *Handler) resolveAccess(ctx context.Context, adminUserID int64) *UserAccess {
	if h.loadAccess == nil {
		return &UserAccess{}
	}
	access, err := h.loadAccess(ctx, adminUserID)
	if err != nil || access == nil {
		log.Printf("[WARN] identity.resolveAccess user_id=%d: %v", adminUserID, err)
		return &UserAccess{}
	}
	return access
}

// ─── Rate Limiting ────────────────────────────────────────────────────────────

func (h *Handler) checkRateLimit(key string) (bool, time.Duration) {
	h.failedLoginMu.Lock()
	defer h.failedLoginMu.Unlock()

	attempt, exists := h.failedLogins[key]
	if !exists {
		return true, 0
	}

	if time.Since(attempt.FirstFail) >= loginAttemptWindow {
		// Reset setelah 15 menit
		delete(h.failedLogins, key)
		return true, 0
	}

	if attempt.Count < loginAttemptLimit {
		return true, 0
	}

	return false, time.Until(attempt.FirstFail.Add(loginAttemptWindow))
}

func (h *Handler) recordFailedLogin(key string) {
	h.failedLoginMu.Lock()
	defer h.failedLoginMu.Unlock()

	// Bersihkan entri kedaluwarsa berkala agar map tidak tumbuh tanpa batas.
	if now := time.Now(); now.Sub(h.lastPrune) >= time.Minute {
		h.lastPrune = now
		for k, a := range h.failedLogins {
			if now.Sub(a.FirstFail) >= loginAttemptWindow {
				delete(h.failedLogins, k)
			}
		}
	}

	attempt, exists := h.failedLogins[key]
	if !exists || time.Since(attempt.FirstFail) >= loginAttemptWindow {
		h.failedLogins[key] = &loginAttempt{
			Count:     1,
			FirstFail: time.Now(),
		}
		return
	}
	attempt.Count++
}

func (h *Handler) recordSuccessLogin(key string) {
	h.failedLoginMu.Lock()
	defer h.failedLoginMu.Unlock()
	delete(h.failedLogins, key)
}

func getClientIP(r *http.Request) string {
	return shared.ClientIP(r)
}

func loginRateLimitKey(r *http.Request, email string) string {
	return getClientIP(r) + "|" + email
}

func writeRateLimitError(w http.ResponseWriter, retryAfter time.Duration) {
	retryAfterSeconds := int(math.Ceil(retryAfter.Seconds()))
	if retryAfterSeconds < 1 {
		retryAfterSeconds = 1
	}

	w.Header().Set("Retry-After", strconv.Itoa(retryAfterSeconds))
	writeJSON(w, http.StatusTooManyRequests, map[string]any{
		"error":               "terlalu banyak percobaan, coba lagi setelah waktu tunggu selesai",
		"retry_after_seconds": retryAfterSeconds,
	})
}

// ─── Login ────────────────────────────────────────────────────────────────────

// decodeAuthBody membaca body JSON /api/auth/* dengan batas ukuran authMaxBodyBytes.
func decodeAuthBody(w http.ResponseWriter, r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(w, r.Body, authMaxBodyBytes)
	return json.NewDecoder(r.Body).Decode(v)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := decodeAuthBody(w, r, &req); err != nil {
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

	// Pisahkan limit berdasarkan client dan akun. Ini mencegah kegagalan satu
	// pengguna di balik Nginx/Cloudflare memblokir seluruh admin lainnya.
	rateLimitKey := loginRateLimitKey(r, req.Email)
	clientIP := getClientIP(r)
	if allowed, retryAfter := h.checkRateLimit(rateLimitKey); !allowed {
		log.Printf("[AUDIT] login_rate_limited ip=%s email=%s", clientIP, maskEmail(req.Email))
		writeRateLimitError(w, retryAfter)
		return
	}
	if h.loginIP != nil && !h.loginIP.Allow(clientIP) {
		log.Printf("[AUDIT] login_rate_limited ip=%s scope=ip", clientIP)
		writeRateLimitError(w, loginAttemptWindow)
		return
	}
	fail := func(reason string) {
		h.recordFailedLogin(rateLimitKey)
		if h.loginIP != nil {
			h.loginIP.Record(clientIP)
		}
		log.Printf("[AUDIT] login_failed ip=%s email=%s reason=%s", clientIP, maskEmail(req.Email), reason)
		writeError(w, http.StatusUnauthorized, "email atau password salah")
	}
	if len(req.Password) > 72 { // batas bcrypt; jangan biarkan input panjang membuang CPU
		fail("bad_password")
		return
	}

	user, err := h.repo.GetByEmail(r.Context(), req.Email)
	if err != nil && !errors.Is(err, ErrNotFound) {
		log.Printf("[ERROR] identity.Login GetByEmail: %v", err)
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan internal")
		return
	}
	// Selalu jalankan bcrypt (hash tiruan untuk email tak dikenal/nonaktif) agar waktu jawab sama.
	hash := h.dummyHash
	if user != nil && user.IsActive {
		hash = []byte(user.PasswordHash)
	}
	cmpErr := bcrypt.CompareHashAndPassword(hash, []byte(req.Password))
	switch {
	case user == nil:
		fail("unknown_account")
		return
	case !user.IsActive:
		fail("inactive")
		return
	case cmpErr != nil:
		fail("bad_password")
		return
	}

	// Sukses
	h.recordSuccessLogin(rateLimitKey)
	log.Printf("[AUDIT] login_success ip=%s user_id=%d role=%s", getClientIP(r), user.ID, user.Role)

	access := h.resolveAccess(r.Context(), user.ID)
	accessToken, err := GenerateAccessTokenWithAccess(user.ID, user.BrandID, user.Role, user.Email,
		access.Roles, access.Permissions, access.PermVersion)
	if err != nil {
		log.Printf("[ERROR] identity.Login GenerateAccessToken: %v", err)
		writeError(w, http.StatusInternalServerError, "gagal membuat access token")
		return
	}

	refreshToken, refreshJTI, refreshExp, err := GenerateRefreshToken(user.ID)
	if err != nil {
		log.Printf("[ERROR] identity.Login GenerateRefreshToken: %v", err)
		writeError(w, http.StatusInternalServerError, "gagal membuat refresh token")
		return
	}
	if err := h.repo.SaveRefreshToken(r.Context(), refreshJTI, user.ID, refreshExp); err != nil {
		log.Printf("[ERROR] identity.Login SaveRefreshToken: %v", err)
		writeError(w, http.StatusInternalServerError, "gagal membuat refresh token")
		return
	}

	ttl := int(getAccessTTL().Seconds())

	writeJSON(w, http.StatusOK, TokenResponse{
		UserID:       user.ID,
		DisplayName:  user.DisplayName,
		Email:        user.Email,
		Role:         user.Role,
		Roles:        access.Roles,
		Permissions:  access.Permissions,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    ttl,
		TokenType:    "Bearer",
	})
}

func maskEmail(email string) string {
	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 || parts[0] == "" {
		return "***"
	}
	local := parts[0]
	if len(local) > 2 {
		local = local[:1] + "***" + local[len(local)-1:]
	} else {
		local = "***"
	}
	return local + "@" + parts[1]
}

// ─── Refresh ──────────────────────────────────────────────────────────────────

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := decodeAuthBody(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}

	if strings.TrimSpace(req.RefreshToken) == "" {
		writeError(w, http.StatusBadRequest, "refresh_token wajib diisi")
		return
	}

	adminUserID, jti, err := ParseRefreshToken(req.RefreshToken)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "refresh token tidak valid")
		return
	}

	active, err := h.repo.IsRefreshTokenActive(r.Context(), jti, adminUserID)
	if err != nil {
		log.Printf("[ERROR] identity.Refresh IsRefreshTokenActive: %v", err)
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan internal")
		return
	}
	if !active {
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

	access := h.resolveAccess(r.Context(), user.ID)
	accessToken, err := GenerateAccessTokenWithAccess(user.ID, user.BrandID, user.Role, user.Email,
		access.Roles, access.Permissions, access.PermVersion)
	if err != nil {
		log.Printf("[ERROR] identity.Refresh GenerateAccessToken: %v", err)
		writeError(w, http.StatusInternalServerError, "gagal membuat access token")
		return
	}

	ttl := int(getAccessTTL().Seconds())

	writeJSON(w, http.StatusOK, TokenResponse{
		UserID:      user.ID,
		DisplayName: user.DisplayName,
		Email:       user.Email,
		Role:        user.Role,
		Roles:       access.Roles,
		Permissions: access.Permissions,
		AccessToken: accessToken,
		ExpiresIn:   ttl,
		TokenType:   "Bearer",
	})
}

// ─── Logout ───────────────────────────────────────────────────────────────────

// Logout mencabut refresh token yang dikirim. Selalu 200 agar client tetap
// bisa membersihkan sesi lokal walau token sudah kedaluwarsa/tidak valid.
// Access token yang sudah terbit tetap berlaku sampai habis (maks. 15 menit).
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	_ = decodeAuthBody(w, r, &req)

	if token := strings.TrimSpace(req.RefreshToken); token != "" {
		if _, jti, err := ParseRefreshToken(token); err == nil {
			if err := h.repo.RevokeRefreshToken(r.Context(), jti); err != nil {
				log.Printf("[ERROR] identity.Logout RevokeRefreshToken: %v", err)
				writeError(w, http.StatusInternalServerError, "gagal logout, silakan coba lagi")
				return
			}
		}
	}

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
