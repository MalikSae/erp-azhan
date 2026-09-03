package portal

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"erp-azhan/api/internal/identity"
	"erp-azhan/api/internal/jamaah"
	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"
)

type CheckActivationTokenRequest struct {
	Token string `json:"token"`
}

type CheckActivationTokenResponse struct {
	Valid       bool   `json:"valid"`
	NamaLengkap string `json:"nama_lengkap,omitempty"`
}

type VerifyActivationDobRequest struct {
	Token        string `json:"token"`
	TanggalLahir string `json:"tanggal_lahir"`
}

type ActivateAccountRequest struct {
	Token        string `json:"token"`
	TanggalLahir string `json:"tanggal_lahir"`
	PortalPIN    string `json:"portal_pin"`
}

// ==========================================
// BAGIAN 2 — POST /api/admin/jamaah/{id}/activation-link
// ==========================================

func (h *Handler) GenerateActivationLink(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	jamaahID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || jamaahID <= 0 {
		writeError(w, http.StatusBadRequest, "id jamaah tidak valid")
		return
	}

	brandID := identity.GetBrandID(r.Context())
	adminUserID := identity.GetAdminUserID(r.Context())
	if adminUserID <= 0 {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// 1. Ambil jamaah, verifikasi masuk scope brand admin yang login. Kalau tidak, 404.
	j, err := h.jamaahRepo.GetByID(r.Context(), jamaahID, brandID)
	if err != nil {
		if errors.Is(err, jamaah.ErrNotFound) {
			writeError(w, http.StatusNotFound, "jamaah tidak ditemukan")
			return
		}
		writeError(w, http.StatusInternalServerError, "gagal mengambil data jamaah")
		return
	}

	// 2. Kalau tanggal_lahir jamaah NULL atau kosong -> 400
	if j.TanggalLahir == nil || strings.TrimSpace(*j.TanggalLahir) == "" {
		writeError(w, http.StatusBadRequest, "Lengkapi tanggal lahir jamaah terlebih dahulu sebelum menerbitkan link aktivasi.")
		return
	}

	// 3. Batalkan token lama yang masih aktif untuk jamaah itu:
	// set expires_at = NOW() pada baris yang used_at IS NULL AND expires_at > NOW().
	_, err = h.db.ExecContext(r.Context(), `
		UPDATE jamaah_activation_tokens 
		SET expires_at = NOW() 
		WHERE jamaah_id = ? AND used_at IS NULL AND expires_at > NOW()
	`, jamaahID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal memperbarui token lama")
		return
	}

	// 4. Buat token acak kriptografis (crypto/rand, minimal 32 byte, base64url)
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		writeError(w, http.StatusInternalServerError, "gagal menghasilkan token aktivasi")
		return
	}
	rawToken := base64.RawURLEncoding.EncodeToString(tokenBytes)

	// Simpan HASH-nya (SHA-256)
	hash := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(hash[:])

	// 5. expires_at = NOW() + 24 jam
	expiresAt := time.Now().Add(24 * time.Hour)

	// Simpan ke database
	_, err = h.db.ExecContext(r.Context(), `
		INSERT INTO jamaah_activation_tokens (jamaah_id, token_hash, expires_at, created_by)
		VALUES (?, ?, ?, ?)
	`, jamaahID, tokenHash, expiresAt, adminUserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal menyimpan token aktivasi")
		return
	}

	// Ambil domain brand dari tabel brands
	var brandDomain sql.NullString
	_ = h.db.QueryRowContext(r.Context(), `SELECT domain FROM brands WHERE id = ?`, j.BrandID).Scan(&brandDomain)

	var activationURL string
	if brandDomain.Valid && strings.TrimSpace(brandDomain.String) != "" {
		domain := strings.TrimSpace(brandDomain.String)
		if strings.HasPrefix(domain, "http://") || strings.HasPrefix(domain, "https://") {
			activationURL = fmt.Sprintf("%s/portal/aktivasi?token=%s", strings.TrimRight(domain, "/"), rawToken)
		} else {
			activationURL = fmt.Sprintf("http://%s/portal/aktivasi?token=%s", domain, rawToken)
		}
	} else {
		activationURL = fmt.Sprintf("/portal/aktivasi?token=%s", rawToken)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"activation_url": activationURL,
		"expires_at":     expiresAt.Format(time.RFC3339),
	})
}

// ==========================================
// BAGIAN 3 — POST /api/public/aktivasi/check
// ==========================================

func (h *Handler) checkTokenRateLimit(ip string) bool {
	h.failedTokenChecksMu.Lock()
	defer h.failedTokenChecksMu.Unlock()

	if h.failedTokenChecks == nil {
		h.failedTokenChecks = make(map[string]*loginAttempt)
		return true
	}

	attempt, exists := h.failedTokenChecks[ip]
	if !exists {
		return true
	}
	if time.Since(attempt.FirstFail) > 15*time.Minute {
		delete(h.failedTokenChecks, ip)
		return true
	}
	return attempt.Count < 10
}

func (h *Handler) recordFailedTokenCheck(ip string) {
	h.failedTokenChecksMu.Lock()
	defer h.failedTokenChecksMu.Unlock()

	if h.failedTokenChecks == nil {
		h.failedTokenChecks = make(map[string]*loginAttempt)
	}

	attempt, exists := h.failedTokenChecks[ip]
	if !exists || time.Since(attempt.FirstFail) > 15*time.Minute {
		h.failedTokenChecks[ip] = &loginAttempt{
			Count:     1,
			FirstFail: time.Now(),
		}
		return
	}
	attempt.Count++
}

func (h *Handler) resetFailedTokenCheck(ip string) {
	h.failedTokenChecksMu.Lock()
	defer h.failedTokenChecksMu.Unlock()
	if h.failedTokenChecks != nil {
		delete(h.failedTokenChecks, ip)
	}
}

func (h *Handler) CheckActivationToken(w http.ResponseWriter, r *http.Request) {
	clientIP := getClientIP(r)
	if !h.checkTokenRateLimit(clientIP) {
		writeError(w, http.StatusTooManyRequests, "terlalu banyak percobaan, coba lagi dalam 15 menit")
		return
	}

	var req CheckActivationTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.recordFailedTokenCheck(clientIP)
		writeJSON(w, http.StatusOK, CheckActivationTokenResponse{Valid: false})
		return
	}

	req.Token = strings.TrimSpace(req.Token)
	if req.Token == "" {
		h.recordFailedTokenCheck(clientIP)
		writeJSON(w, http.StatusOK, CheckActivationTokenResponse{Valid: false})
		return
	}

	hash := sha256.Sum256([]byte(req.Token))
	tokenHash := hex.EncodeToString(hash[:])

	var (
		tokenID     int64
		namaLengkap string
	)
	query := `
		SELECT t.id, j.nama_lengkap
		FROM jamaah_activation_tokens t
		JOIN jamaah j ON t.jamaah_id = j.id
		WHERE t.token_hash = ? AND t.used_at IS NULL AND t.expires_at > NOW()
	`
	err := h.db.QueryRowContext(r.Context(), query, tokenHash).Scan(&tokenID, &namaLengkap)
	if err != nil {
		h.recordFailedTokenCheck(clientIP)
		writeJSON(w, http.StatusOK, CheckActivationTokenResponse{Valid: false})
		return
	}

	h.resetFailedTokenCheck(clientIP)
	writeJSON(w, http.StatusOK, CheckActivationTokenResponse{
		Valid:       true,
		NamaLengkap: namaLengkap,
	})
}

// ==========================================
// BAGIAN 4 — POST /api/public/aktivasi
// ==========================================

func (h *Handler) checkDobRateLimit(ip string) bool {
	h.failedDobAttemptsMu.Lock()
	defer h.failedDobAttemptsMu.Unlock()

	if h.failedDobAttempts == nil {
		h.failedDobAttempts = make(map[string]*loginAttempt)
		return true
	}

	attempt, exists := h.failedDobAttempts[ip]
	if !exists {
		return true
	}
	if time.Since(attempt.FirstFail) > 15*time.Minute {
		delete(h.failedDobAttempts, ip)
		return true
	}
	return attempt.Count < 5
}

func (h *Handler) recordFailedDob(ip string) {
	h.failedDobAttemptsMu.Lock()
	defer h.failedDobAttemptsMu.Unlock()

	if h.failedDobAttempts == nil {
		h.failedDobAttempts = make(map[string]*loginAttempt)
	}

	attempt, exists := h.failedDobAttempts[ip]
	if !exists || time.Since(attempt.FirstFail) > 15*time.Minute {
		h.failedDobAttempts[ip] = &loginAttempt{
			Count:     1,
			FirstFail: time.Now(),
		}
		return
	}
	attempt.Count++
}

func (h *Handler) resetFailedDob(ip string) {
	h.failedDobAttemptsMu.Lock()
	defer h.failedDobAttemptsMu.Unlock()
	if h.failedDobAttempts != nil {
		delete(h.failedDobAttempts, ip)
	}
}

func isOnlyDigits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func (h *Handler) VerifyActivationDob(w http.ResponseWriter, r *http.Request) {
	clientIP := getClientIP(r)
	if !h.checkDobRateLimit(clientIP) {
		writeError(w, http.StatusTooManyRequests, "terlalu banyak percobaan, coba lagi dalam 15 menit")
		return
	}

	var req VerifyActivationDobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}

	req.Token = strings.TrimSpace(req.Token)
	req.TanggalLahir = strings.TrimSpace(req.TanggalLahir)

	if req.Token == "" {
		writeError(w, http.StatusBadRequest, "Link aktivasi tidak valid atau sudah kedaluwarsa.")
		return
	}

	// 1. Hash token, cari baris dengan syarat yang SAMA seperti endpoint aktivasi:
	// ditemukan, used_at IS NULL, expires_at > NOW(). Tidak valid -> 400 "Link aktivasi tidak valid atau sudah kedaluwarsa."
	hash := sha256.Sum256([]byte(req.Token))
	tokenHash := hex.EncodeToString(hash[:])

	var (
		tokenID      int64
		tanggalLahir sql.NullString
	)
	query := `
		SELECT t.id, DATE_FORMAT(j.tanggal_lahir, '%Y-%m-%d')
		FROM jamaah_activation_tokens t
		JOIN jamaah j ON t.jamaah_id = j.id
		WHERE t.token_hash = ? AND t.used_at IS NULL AND t.expires_at > NOW()
	`
	err := h.db.QueryRowContext(r.Context(), query, tokenHash).Scan(&tokenID, &tanggalLahir)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Link aktivasi tidak valid atau sudah kedaluwarsa.")
		return
	}

	// 2. Cocokkan tanggal_lahir dengan tanggal_lahir jamaah, pakai perbandingan yang SAMA PERSIS dengan ActivateAccount (DATE_FORMAT '%Y-%m-%d').
	// Tidak cocok -> 400 "Tanggal lahir tidak sesuai."
	dbDob := ""
	if tanggalLahir.Valid {
		dbDob = strings.TrimSpace(tanggalLahir.String)
	}

	if dbDob == "" || req.TanggalLahir != dbDob {
		h.recordFailedDob(clientIP)
		writeError(w, http.StatusBadRequest, "Tanggal lahir tidak sesuai.")
		return
	}

	// 3. Cocok -> 200 {"valid": true}
	h.resetFailedDob(clientIP)
	writeJSON(w, http.StatusOK, map[string]bool{"valid": true})
}

func (h *Handler) ActivateAccount(w http.ResponseWriter, r *http.Request) {
	clientIP := getClientIP(r)
	if !h.checkDobRateLimit(clientIP) {
		writeError(w, http.StatusTooManyRequests, "terlalu banyak percobaan, coba lagi dalam 15 menit")
		return
	}

	var req ActivateAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}

	req.Token = strings.TrimSpace(req.Token)
	req.TanggalLahir = strings.TrimSpace(req.TanggalLahir)
	req.PortalPIN = strings.TrimSpace(req.PortalPIN)

	if req.Token == "" {
		writeError(w, http.StatusBadRequest, "Link aktivasi tidak valid atau sudah kedaluwarsa.")
		return
	}

	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal memulai transaksi")
		return
	}
	defer tx.Rollback()

	// 1. Hash token, cari baris (FOR UPDATE). Tidak valid -> 400 pesan seragam
	hash := sha256.Sum256([]byte(req.Token))
	tokenHash := hex.EncodeToString(hash[:])

	var (
		tokenID      int64
		jamaahID     int64
		tanggalLahir sql.NullString
	)
	query := `
		SELECT t.id, t.jamaah_id, DATE_FORMAT(j.tanggal_lahir, '%Y-%m-%d')
		FROM jamaah_activation_tokens t
		JOIN jamaah j ON t.jamaah_id = j.id
		WHERE t.token_hash = ? AND t.used_at IS NULL AND t.expires_at > NOW()
		FOR UPDATE
	`
	err = tx.QueryRowContext(r.Context(), query, tokenHash).Scan(&tokenID, &jamaahID, &tanggalLahir)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Link aktivasi tidak valid atau sudah kedaluwarsa.")
		return
	}

	// 2. Cocokkan tanggal_lahir dari request dengan tanggal_lahir jamaah.
	// Tidak cocok -> 400 "Tanggal lahir tidak sesuai."
	dbDob := ""
	if tanggalLahir.Valid {
		dbDob = strings.TrimSpace(tanggalLahir.String)
	}

	if dbDob == "" || req.TanggalLahir != dbDob {
		h.recordFailedDob(clientIP)
		writeError(w, http.StatusBadRequest, "Tanggal lahir tidak sesuai.")
		return
	}

	// 3. Validasi portal_pin: wajib tepat 6 digit angka. Tolak "123456" secara eksplisit -> 400
	if req.PortalPIN == "123456" {
		writeError(w, http.StatusBadRequest, "PIN terlalu mudah ditebak, gunakan kombinasi lain.")
		return
	}
	if len(req.PortalPIN) != 6 || !isOnlyDigits(req.PortalPIN) {
		writeError(w, http.StatusBadRequest, "PIN portal harus 6 digit angka")
		return
	}

	// 4. Hash PIN dengan bcrypt, simpan ke jamaah.portal_pin_hash.
	pinHash, err := bcrypt.GenerateFromPassword([]byte(req.PortalPIN), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal memproses PIN")
		return
	}

	_, err = tx.ExecContext(r.Context(), `UPDATE jamaah SET portal_pin_hash = ? WHERE id = ?`, string(pinHash), jamaahID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal memperbarui PIN jamaah")
		return
	}

	// 5. Set used_at = NOW() pada baris token.
	_, err = tx.ExecContext(r.Context(), `UPDATE jamaah_activation_tokens SET used_at = NOW() WHERE id = ?`, tokenID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal menandai token sudah digunakan")
		return
	}

	// 6. Commit
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "gagal menyimpan perubahan")
		return
	}

	h.resetFailedDob(clientIP)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
