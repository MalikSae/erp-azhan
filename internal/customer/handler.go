package customer

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"erp-azhan/api/internal/mailer"
	"erp-azhan/api/internal/shared"
	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

// Batas laju (di memori): daftar per IP, login per IP+email dan per IP, lupa/ kirim ulang per kunci.
const (
	registerIPLimit    = 10 // per jam
	loginKeyLimit      = 5  // gagal per 15 menit per IP+email
	loginIPLimit       = 30 // gagal per 15 menit per IP
	forgotKeyLimit     = 5  // per jam per IP+email
	resendAccountLimit = 3  // per jam per akun
)

type Handler struct {
	repo      *Repository
	mail      mailer.Sender
	captcha   *shared.Turnstile
	regIP     *shared.WindowLimiter
	loginKey  *shared.WindowLimiter
	loginIP   *shared.WindowLimiter
	forgot    *shared.WindowLimiter
	resend    *shared.WindowLimiter
	dummyHash []byte
}

func NewHandler(repo *Repository, m mailer.Sender) *Handler {
	// Hash tiruan agar login untuk email yang tidak ada memakan waktu sama dengan yang ada.
	dummy, _ := bcrypt.GenerateFromPassword([]byte("dummy-password-for-timing"), bcryptCost)
	return &Handler{
		repo: repo, mail: m, captcha: shared.NewTurnstile(),
		regIP:     shared.NewWindowLimiter(registerIPLimit, time.Hour),
		loginKey:  shared.NewWindowLimiter(loginKeyLimit, 15*time.Minute),
		loginIP:   shared.NewWindowLimiter(loginIPLimit, 15*time.Minute),
		forgot:    shared.NewWindowLimiter(forgotKeyLimit, time.Hour),
		resend:    shared.NewWindowLimiter(resendAccountLimit, time.Hour),
		dummyHash: dummy,
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return false
	}
	return true
}

type authResponse struct {
	Account      *Account `json:"account"`
	SessionToken string   `json:"session_token,omitempty"`
}

// siteLink membentuk URL situs brand dari domain di database (bukan dari header request,
// supaya tautan di email tidak bisa dibelokkan). SITE_URL_TEMPLATE, mis. "http://{domain}:3100".
func siteLink(domain, path string, q url.Values) string {
	tpl := strings.TrimSpace(os.Getenv("SITE_URL_TEMPLATE"))
	if tpl == "" {
		tpl = "https://{domain}"
	}
	base := strings.TrimRight(strings.ReplaceAll(tpl, "{domain}", domain), "/")
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	return base + path
}

func (h *Handler) sendAsync(msg mailer.Message) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := h.mail.Send(ctx, msg); err != nil {
			log.Printf("[ERROR] mailer: %v", err)
		}
	}()
}

func (h *Handler) sendVerification(ctx context.Context, acc *Account) {
	name, domain, err := h.repo.BrandSite(ctx, acc.BrandID)
	if err != nil || domain == "" {
		log.Printf("[WARN] customer: domain brand %d tidak ditemukan, email verifikasi dilewati", acc.BrandID)
		return
	}
	token, err := h.repo.IssueToken(ctx, acc.ID, PurposeVerifyMail, VerifyTokenTTL)
	if err != nil {
		log.Printf("[ERROR] customer issue verify token: %v", err)
		return
	}
	link := siteLink(domain, "/verifikasi-email", url.Values{"token": {token}})
	subject, text := mailer.VerifyEmail(name, acc.NamaLengkap, link)
	h.sendAsync(mailer.Message{To: acc.Email, Subject: subject, Text: text})
}

type registerRequest struct {
	BrandID      int64  `json:"brand_id"`
	NamaLengkap  string `json:"nama_lengkap"`
	Email        string `json:"email"`
	NoHP         string `json:"no_hp"`
	Password     string `json:"password"`
	CaptchaToken string `json:"captcha_token"`
}

// Register POST /api/public/auth/register
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	ip := shared.ClientIP(r)
	if !h.regIP.Allow(ip) {
		writeError(w, http.StatusTooManyRequests, "terlalu banyak pendaftaran, coba lagi dalam 1 jam")
		return
	}
	var req registerRequest
	if !decode(w, r, &req) {
		return
	}
	h.regIP.Record(ip)

	if h.captcha.Enabled() {
		ok, err := h.captcha.Verify(r.Context(), strings.TrimSpace(req.CaptchaToken), ip)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "verifikasi keamanan sedang tidak tersedia, coba lagi")
			return
		}
		if !ok {
			writeError(w, http.StatusBadRequest, "verifikasi keamanan gagal, silakan coba lagi")
			return
		}
	}
	if req.BrandID <= 0 {
		writeError(w, http.StatusBadRequest, "brand_id wajib diisi")
		return
	}
	req.NamaLengkap = strings.TrimSpace(req.NamaLengkap)
	req.Email = strings.TrimSpace(req.Email)
	req.NoHP = strings.TrimSpace(req.NoHP)
	switch {
	case !ValidName(req.NamaLengkap):
		writeError(w, http.StatusBadRequest, "nama lengkap wajib diisi (2–150 karakter)")
		return
	case !ValidEmail(req.Email):
		writeError(w, http.StatusBadRequest, "format email tidak valid")
		return
	case !ValidPhone(req.NoHP):
		writeError(w, http.StatusBadRequest, "nomor WhatsApp harus 10–15 digit")
		return
	}
	if ok, msg := ValidPassword(req.Password); !ok {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcryptCost)
	if err != nil {
		log.Printf("[ERROR] customer hash: %v", err)
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan, silakan coba lagi")
		return
	}
	acc, err := h.repo.Create(r.Context(), req.BrandID, req.NamaLengkap, req.Email, req.NoHP, string(hash))
	if errors.Is(err, ErrEmailTaken) {
		writeError(w, http.StatusConflict, "email sudah terdaftar, silakan masuk")
		return
	}
	if err != nil {
		log.Printf("[ERROR] customer create: %v", err)
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan, silakan coba lagi")
		return
	}
	token, err := h.repo.CreateSession(r.Context(), acc.ID)
	if err != nil {
		log.Printf("[ERROR] customer session: %v", err)
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan, silakan coba lagi")
		return
	}
	h.sendVerification(r.Context(), acc)
	writeJSON(w, http.StatusCreated, authResponse{Account: acc, SessionToken: token})
}

type loginRequest struct {
	BrandID  int64  `json:"brand_id"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Login POST /api/public/auth/login
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	ip := shared.ClientIP(r)
	var req loginRequest
	if !decode(w, r, &req) {
		return
	}
	key := ip + "|" + strconv.FormatInt(req.BrandID, 10) + "|" + NormalizeEmail(req.Email)
	if !h.loginKey.Allow(key) || !h.loginIP.Allow(ip) {
		writeError(w, http.StatusTooManyRequests, "terlalu banyak percobaan masuk, coba lagi dalam 15 menit")
		return
	}
	fail := func() {
		h.loginKey.Record(key)
		h.loginIP.Record(ip)
		writeError(w, http.StatusUnauthorized, "email atau password salah")
	}
	if req.BrandID <= 0 || len(req.Password) > 72 {
		fail()
		return
	}
	acc, err := h.repo.GetByEmail(r.Context(), req.BrandID, req.Email)
	hash := h.dummyHash
	if err == nil {
		hash = []byte(acc.passwordHash)
	}
	cmp := bcrypt.CompareHashAndPassword(hash, []byte(req.Password))
	if err != nil || cmp != nil {
		if err != nil && !errors.Is(err, ErrNotFound) {
			log.Printf("[ERROR] customer login lookup: %v", err)
		}
		fail()
		return
	}
	token, err := h.repo.CreateSession(r.Context(), acc.ID)
	if err != nil {
		log.Printf("[ERROR] customer session: %v", err)
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan, silakan coba lagi")
		return
	}
	h.loginKey.Reset(key)
	h.repo.TouchLogin(r.Context(), acc.ID)
	writeJSON(w, http.StatusOK, authResponse{Account: acc, SessionToken: token})
}

// Logout POST /api/account/logout
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	_ = h.repo.RevokeSession(r.Context(), TokenFromContext(r.Context()))
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Me GET /api/account/me
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	a := FromContext(r.Context())
	// Jenis kelamin diambil dari data jamaah akun (bila sudah pernah memesan) untuk mengisi checkout otomatis.
	view := struct {
		*Account
		JenisKelamin string `json:"jenis_kelamin"`
		NIK          string `json:"nik"`
	}{Account: a}
	if a != nil && a.JamaahID != nil {
		var jk, nik sql.NullString
		if err := h.repo.db.QueryRowContext(r.Context(), `SELECT jenis_kelamin, nik FROM jamaah WHERE id=?`, *a.JamaahID).Scan(&jk, &nik); err == nil {
			view.JenisKelamin = jk.String
			view.NIK = nik.String
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"account": view})
}

type forgotRequest struct {
	BrandID int64  `json:"brand_id"`
	Email   string `json:"email"`
}

// ForgotPassword POST /api/public/auth/forgot-password. Jawaban selalu sama agar
// tidak membocorkan email yang terdaftar.
func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	ip := shared.ClientIP(r)
	var req forgotRequest
	if !decode(w, r, &req) {
		return
	}
	generic := map[string]string{"status": "ok", "message": "Jika email terdaftar, tautan atur ulang password sudah dikirim."}
	key := ip + "|" + strconv.FormatInt(req.BrandID, 10) + "|" + NormalizeEmail(req.Email)
	if !h.forgot.Allow(key) {
		writeError(w, http.StatusTooManyRequests, "terlalu banyak permintaan, coba lagi dalam 1 jam")
		return
	}
	h.forgot.Record(key)
	acc, err := h.repo.GetByEmail(r.Context(), req.BrandID, req.Email)
	if err != nil {
		writeJSON(w, http.StatusOK, generic)
		return
	}
	name, domain, err := h.repo.BrandSite(r.Context(), acc.BrandID)
	if err != nil || domain == "" {
		log.Printf("[WARN] customer: domain brand %d tidak ditemukan, email reset dilewati", acc.BrandID)
		writeJSON(w, http.StatusOK, generic)
		return
	}
	token, err := h.repo.IssueToken(r.Context(), acc.ID, PurposeResetPass, ResetTokenTTL)
	if err != nil {
		log.Printf("[ERROR] customer reset token: %v", err)
		writeJSON(w, http.StatusOK, generic)
		return
	}
	link := siteLink(domain, "/atur-ulang-password", url.Values{"token": {token}})
	subject, text := mailer.ResetPassword(name, acc.NamaLengkap, link)
	h.sendAsync(mailer.Message{To: acc.Email, Subject: subject, Text: text})
	writeJSON(w, http.StatusOK, generic)
}

type resetRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

// ResetPassword POST /api/public/auth/reset-password. Semua sesi lama dicabut.
func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	ip := shared.ClientIP(r)
	var req resetRequest
	if !decode(w, r, &req) {
		return
	}
	if !h.loginIP.Allow(ip) {
		writeError(w, http.StatusTooManyRequests, "terlalu banyak percobaan, coba lagi dalam 15 menit")
		return
	}
	if ok, msg := ValidPassword(req.Password); !ok {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcryptCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan, silakan coba lagi")
		return
	}
	accountID, err := h.repo.ConsumeToken(r.Context(), strings.TrimSpace(req.Token), PurposeResetPass)
	if errors.Is(err, ErrTokenInvalid) {
		h.loginIP.Record(ip)
		writeError(w, http.StatusBadRequest, ErrTokenInvalid.Error())
		return
	}
	if err != nil {
		log.Printf("[ERROR] customer consume reset: %v", err)
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan, silakan coba lagi")
		return
	}
	if err = h.repo.SetPassword(r.Context(), accountID, string(hash)); err != nil {
		log.Printf("[ERROR] customer set password: %v", err)
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan, silakan coba lagi")
		return
	}
	_ = h.repo.RevokeAllSessions(r.Context(), accountID)
	_ = h.repo.MarkEmailVerified(r.Context(), accountID) // tautan sampai ke kotak masuk = email terbukti milik pemilik
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type verifyRequest struct {
	Token string `json:"token"`
}

// VerifyEmail POST /api/public/auth/verify-email
func (h *Handler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	ip := shared.ClientIP(r)
	var req verifyRequest
	if !decode(w, r, &req) {
		return
	}
	if !h.loginIP.Allow(ip) {
		writeError(w, http.StatusTooManyRequests, "terlalu banyak percobaan, coba lagi dalam 15 menit")
		return
	}
	accountID, err := h.repo.ConsumeToken(r.Context(), strings.TrimSpace(req.Token), PurposeVerifyMail)
	if errors.Is(err, ErrTokenInvalid) {
		h.loginIP.Record(ip)
		writeError(w, http.StatusBadRequest, ErrTokenInvalid.Error())
		return
	}
	if err != nil {
		log.Printf("[ERROR] customer consume verify: %v", err)
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan, silakan coba lagi")
		return
	}
	if err = h.repo.MarkEmailVerified(r.Context(), accountID); err != nil {
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan, silakan coba lagi")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ResendVerification POST /api/account/resend-verification
func (h *Handler) ResendVerification(w http.ResponseWriter, r *http.Request) {
	acc := FromContext(r.Context())
	if acc.EmailVerifiedAt != nil {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}
	key := strconv.FormatInt(acc.ID, 10)
	if !h.resend.Allow(key) {
		writeError(w, http.StatusTooManyRequests, "terlalu banyak permintaan, coba lagi dalam 1 jam")
		return
	}
	h.resend.Record(key)
	h.sendVerification(r.Context(), acc)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
