package customer

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"erp-azhan/api/internal/agen"
	"erp-azhan/api/internal/shared"
	"golang.org/x/crypto/bcrypt"
)

// ErrGoogleLain: akun dengan email itu sudah tertaut ke akun Google lain.
var ErrGoogleLain = errors.New("akun sudah tertaut ke akun Google lain")

// GetByGoogleSub mencari akun brand yang tertaut ke akun Google.
func (r *Repository) GetByGoogleSub(ctx context.Context, brandID int64, sub string) (*Account, error) {
	return scanAccount(r.db.QueryRowContext(ctx,
		`SELECT `+accountCols+` FROM customer_accounts WHERE brand_id=? AND google_sub=?`, brandID, sub))
}

// TautkanGoogle menautkan akun ke akun Google. Akun yang sudah tertaut ke akun Google lain ditolak.
func (r *Repository) TautkanGoogle(ctx context.Context, id int64, sub string) error {
	res, err := r.db.ExecContext(ctx, `UPDATE customer_accounts SET google_sub=? WHERE id=? AND (google_sub IS NULL OR google_sub=?)`, sub, id, sub)
	if err != nil {
		return mapDuplicate(err, "tautkan google")
	}
	if n, _ := res.RowsAffected(); n == 0 {
		var ada string
		if err := r.db.QueryRowContext(ctx, `SELECT COALESCE(google_sub, '') FROM customer_accounts WHERE id=?`, id).Scan(&ada); err != nil {
			return err
		}
		if ada != sub {
			return ErrGoogleLain
		}
	}
	return nil
}

// CreateGoogle membuat akun dari akun Google, tanpa sandi.
func (r *Repository) CreateGoogle(ctx context.Context, brandID int64, nama, email, noHP, sub, kodeReferral string, bio Biodata) (*Account, error) {
	agenID, err := agen.ResolveKodeReferral(ctx, r.db, brandID, kodeReferral)
	if err != nil {
		return nil, err
	}
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO customer_accounts (brand_id, nama_lengkap, email, email_normal, no_hp, password_hash, google_sub, direkrut_oleh_akun_id,
		 jenis_kelamin, domisili, tanggal_lahir, email_verified_at) VALUES (?,?,?,?,?,NULL,?,?, NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NOW())`,
		brandID, nama, strings.TrimSpace(email), NormalizeEmail(email), noHP, sub, nullableAgen(agenID),
		bio.JenisKelamin, bio.Domisili, bio.TanggalLahir)
	if err != nil {
		return nil, mapDuplicate(err, "create account google")
	}
	id, _ := res.LastInsertId()
	return r.GetByID(ctx, id)
}

// BuatPassword mengisi sandi untuk akun yang belum punya sandi. false bila akun sudah punya sandi.
func (r *Repository) BuatPassword(ctx context.Context, id int64, hash string) (bool, error) {
	res, err := r.db.ExecContext(ctx, `UPDATE customer_accounts SET password_hash=? WHERE id=? AND password_hash IS NULL`, hash, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n == 1, nil
}

type googleRequest struct {
	BrandID int64  `json:"brand_id"`
	IDToken string `json:"id_token"`
}

// googleLengkapi: jawaban untuk akun Google yang belum terdaftar di brand. App membuka Lengkapi Data
// dengan nama dan email ini, lalu mengirim POST /api/public/auth/google/daftar.
type googleLengkapi struct {
	PerluLengkapi bool   `json:"perlu_lengkapi"`
	NamaLengkap   string `json:"nama_lengkap"`
	Email         string `json:"email"`
}

func (h *Handler) verifikasiGoogle(w http.ResponseWriter, r *http.Request, idToken string) (*ProfilGoogle, bool) {
	if strings.TrimSpace(idToken) == "" {
		writeError(w, http.StatusBadRequest, "Id_token wajib diisi.")
		return nil, false
	}
	p, err := h.google.Verifikasi(r.Context(), strings.TrimSpace(idToken))
	if errors.Is(err, ErrGoogleTidakDikonfigurasi) {
		writeError(w, http.StatusServiceUnavailable, "Masuk dengan Google belum tersedia.")
		return nil, false
	}
	if err != nil {
		if !errors.Is(err, ErrTokenGoogle) {
			log.Printf("[ERROR] customer google verifikasi: %v", err)
		}
		writeError(w, http.StatusUnauthorized, "Masuk dengan Google gagal, coba lagi.")
		return nil, false
	}
	return p, true
}

func (h *Handler) sesiBaru(w http.ResponseWriter, r *http.Request, acc *Account, status int) {
	token, err := h.repo.CreateSession(r.Context(), acc.ID)
	if err != nil {
		log.Printf("[ERROR] customer session: %v", err)
		writeError(w, http.StatusInternalServerError, "Permintaan gagal diproses. Coba lagi.")
		return
	}
	h.repo.TouchLogin(r.Context(), acc.ID)
	writeJSON(w, status, authResponse{Account: acc, SessionToken: token})
}

// LoginGoogle POST /api/public/auth/google {brand_id, id_token}.
// Akun yang sudah tertaut, atau akun dengan email yang sama (lalu ditautkan), langsung masuk.
// Akun Google yang belum terdaftar mendapat {perlu_lengkapi: true, nama_lengkap, email}.
func (h *Handler) LoginGoogle(w http.ResponseWriter, r *http.Request) {
	ip := shared.ClientIP(r)
	var req googleRequest
	if !decode(w, r, &req) {
		return
	}
	if !h.loginIP.Take(ip) {
		writeError(w, http.StatusTooManyRequests, "Terlalu banyak percobaan masuk, coba lagi dalam 15 menit.")
		return
	}
	if req.BrandID <= 0 {
		writeError(w, http.StatusBadRequest, "Brand wajib diisi.")
		return
	}
	p, ok := h.verifikasiGoogle(w, r, req.IDToken)
	if !ok {
		return
	}
	h.loginIP.Refund(ip)
	acc, err := h.repo.GetByGoogleSub(r.Context(), req.BrandID, p.Sub)
	if errors.Is(err, ErrNotFound) {
		acc, err = h.repo.GetByEmail(r.Context(), req.BrandID, p.Email)
		if err == nil {
			err = h.repo.TautkanGoogle(r.Context(), acc.ID, p.Sub)
			if errors.Is(err, ErrGoogleLain) {
				writeError(w, http.StatusConflict, "Email ini sudah tertaut ke akun Google lain. Masuk dengan email dan kata sandi.")
				return
			}
		}
	}
	if errors.Is(err, ErrNotFound) {
		writeJSON(w, http.StatusOK, googleLengkapi{PerluLengkapi: true, NamaLengkap: p.Nama, Email: p.Email})
		return
	}
	if err != nil {
		log.Printf("[ERROR] customer google login: %v", err)
		writeError(w, http.StatusInternalServerError, "Permintaan gagal diproses. Coba lagi.")
		return
	}
	h.sesiBaru(w, r, acc, http.StatusOK)
}

type googleDaftarRequest struct {
	BrandID      int64  `json:"brand_id"`
	IDToken      string `json:"id_token"`
	NamaLengkap  string `json:"nama_lengkap"`
	NoHP         string `json:"no_hp"`
	JenisKelamin string `json:"jenis_kelamin"`
	Domisili     string `json:"domisili"`
	TanggalLahir string `json:"tanggal_lahir"`
	KodeReferral string `json:"kode_referral"`
	KodeAgen     string `json:"kode_agen"`
}

// DaftarGoogle POST /api/public/auth/google/daftar: membuat akun dari akun Google (Lengkapi Data).
// Email diambil dari token, bukan dari isian. Bila akun ternyata sudah ada, langsung masuk.
func (h *Handler) DaftarGoogle(w http.ResponseWriter, r *http.Request) {
	ip := shared.ClientIP(r)
	var req googleDaftarRequest
	if !decode(w, r, &req) {
		return
	}
	if !h.regIP.Take(ip) {
		writeError(w, http.StatusTooManyRequests, "Terlalu banyak pendaftaran, coba lagi dalam 1 jam.")
		return
	}
	if req.BrandID <= 0 {
		writeError(w, http.StatusBadRequest, "Brand wajib diisi.")
		return
	}
	p, ok := h.verifikasiGoogle(w, r, req.IDToken)
	if !ok {
		return
	}
	if acc, err := h.repo.GetByGoogleSub(r.Context(), req.BrandID, p.Sub); err == nil {
		h.sesiBaru(w, r, acc, http.StatusOK)
		return
	}
	nama := strings.TrimSpace(req.NamaLengkap)
	if nama == "" {
		nama = p.Nama
	}
	req.NoHP = strings.TrimSpace(req.NoHP)
	if msg := validateProfile(nama, p.Email, req.NoHP); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	bio, msg := biodataDaftar(req.JenisKelamin, req.Domisili, req.TanggalLahir, time.Now())
	if msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	kodeAgen, ok, err := h.repo.KodeAgenDipakai(r.Context(), req.BrandID, req.KodeAgen, req.KodeReferral)
	if err != nil {
		log.Printf("[ERROR] customer kode agen: %v", err)
		writeError(w, http.StatusInternalServerError, "Permintaan gagal diproses. Coba lagi.")
		return
	}
	if !ok {
		writeError(w, http.StatusBadRequest, "Kode agen tidak ditemukan")
		return
	}
	acc, err := h.repo.CreateGoogle(r.Context(), req.BrandID, nama, p.Email, req.NoHP, p.Sub, kodeAgen, bio)
	if errors.Is(err, ErrEmailTaken) {
		writeError(w, http.StatusConflict, "Email ini sudah terdaftar. Masuk dengan Google atau email dan kata sandi.")
		return
	}
	if errors.Is(err, ErrPhoneTaken) {
		writeError(w, http.StatusConflict, "Nomor WhatsApp sudah terdaftar. Gunakan nomor WhatsApp lain.")
		return
	}
	if err != nil {
		log.Printf("[ERROR] customer google create: %v", err)
		writeError(w, http.StatusInternalServerError, "Permintaan gagal diproses. Coba lagi.")
		return
	}
	h.sesiBaru(w, r, acc, http.StatusCreated)
}

type buatPasswordRequest struct {
	PasswordBaru string `json:"password_baru"`
}

// BuatPassword POST /api/account/password/buat: sandi pertama untuk akun yang dibuat lewat Google.
func (h *Handler) BuatPassword(w http.ResponseWriter, r *http.Request) {
	acc := FromContext(r.Context())
	var req buatPasswordRequest
	if !decode(w, r, &req) {
		return
	}
	if acc.PunyaPassword {
		writeError(w, http.StatusConflict, "Akun sudah punya kata sandi.")
		return
	}
	if ok, msg := ValidPassword(req.PasswordBaru); !ok {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.PasswordBaru), bcryptCost)
	if err != nil {
		log.Printf("[ERROR] customer hash: %v", err)
		writeError(w, http.StatusInternalServerError, "Permintaan gagal diproses. Coba lagi.")
		return
	}
	dibuat, err := h.repo.BuatPassword(r.Context(), acc.ID, string(hash))
	if err != nil {
		log.Printf("[ERROR] customer buat password: %v", err)
		writeError(w, http.StatusInternalServerError, "Permintaan gagal diproses. Coba lagi.")
		return
	}
	if !dibuat {
		writeError(w, http.StatusConflict, "Akun sudah punya kata sandi.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
