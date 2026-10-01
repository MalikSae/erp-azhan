package selfbooking

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"unicode/utf8"

	"erp-azhan/api/internal/agen"
	"erp-azhan/api/internal/shared"
	"golang.org/x/crypto/bcrypt"
)

// Pendaftaran akun untuk calon agen yang belum punya akun jamaah (screen A0,
// agen-azhan.md 3.6.2). MVP tanpa OTP (keputusan D3): captcha + rate limit,
// dan keagenan tetap wajib disetujui Admin Travel.

var ErrNomorTerdaftar = errors.New("nomor ini sudah terdaftar, silakan masuk ke portal lalu ajukan dari menu Syiar")

const daftarAgenIPLimit = 10 // percobaan POST /api/public/agen/daftar per IP per jam

type DaftarAgenRequest struct {
	BrandID      int64  `json:"brand_id"`
	NamaLengkap  string `json:"nama_lengkap"`
	NoHP         string `json:"no_hp"`
	PortalPIN    string `json:"portal_pin"`
	CaptchaToken string `json:"captcha_token"`
	// Diisi route handler microsite dari cookie referral, bukan input pengguna.
	KodeReferral string `json:"kode_referral"`
}

// DaftarAgen POST /api/public/agen/daftar
func (h *Handler) DaftarAgen(w http.ResponseWriter, r *http.Request) {
	clientIP := getClientIP(r)
	if !h.daftarByIP.allow(clientIP) {
		writeError(w, http.StatusTooManyRequests, "terlalu banyak percobaan pendaftaran, coba lagi dalam 1 jam")
		return
	}
	h.daftarByIP.record(clientIP)

	var req DaftarAgenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}

	// Gate 1: captcha (hanya bila Turnstile aktif, lihat turnstile.go)
	if h.captcha.enabled() {
		token := strings.TrimSpace(req.CaptchaToken)
		if token == "" {
			writeError(w, http.StatusBadRequest, "Verifikasi keamanan gagal, silakan coba lagi")
			return
		}
		valid, err := h.captcha.verify(r.Context(), token, clientIP)
		if err != nil {
			log.Printf("[ERROR] daftar agen captcha: %v", err)
			writeError(w, http.StatusServiceUnavailable, "verifikasi keamanan sedang tidak tersedia, silakan coba lagi")
			return
		}
		if !valid {
			writeError(w, http.StatusBadRequest, "Verifikasi keamanan gagal, silakan coba lagi")
			return
		}
	}

	// Gate 2–5: data akun
	if req.BrandID <= 0 {
		writeError(w, http.StatusBadRequest, "brand_id wajib diisi")
		return
	}
	req.NamaLengkap = strings.TrimSpace(req.NamaLengkap)
	if n := utf8.RuneCountInString(req.NamaLengkap); n < 2 || n > 150 {
		writeError(w, http.StatusBadRequest, "nama lengkap wajib diisi (2–150 karakter)")
		return
	}
	canonical, _ := shared.PhoneVariants(req.NoHP)
	if len(canonical) < 9 || len(canonical) > 15 {
		writeError(w, http.StatusBadRequest, "nomor WhatsApp tidak valid")
		return
	}
	req.PortalPIN = strings.TrimSpace(req.PortalPIN)
	if len(req.PortalPIN) != 6 || strings.Trim(req.PortalPIN, "0123456789") != "" {
		writeError(w, http.StatusBadRequest, "PIN portal harus 6 digit angka")
		return
	}

	if err := h.repo.DaftarAkunAgen(r.Context(), req); err != nil {
		switch {
		case errors.Is(err, ErrNomorTerdaftar):
			writeError(w, http.StatusConflict, err.Error())
		case errors.Is(err, shared.ErrBrandCodeMissing), errors.Is(err, ErrNotFound):
			writeError(w, http.StatusBadRequest, "brand tidak valid")
		default:
			log.Printf("[ERROR] daftar agen: %v", err)
			writeError(w, http.StatusInternalServerError, "terjadi kesalahan, silakan coba lagi")
		}
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"message": "akun berhasil dibuat"})
}

// DaftarAkunAgen membuat akun jamaah baru ber-PIN dan menerapkan kaitan agen
// dari kode referral (bila valid). Status agen tetap 'tidak_aktif' sampai
// pemohon mengirim Lengkapi Data Agen (3.6.3).
func (r *Repository) DaftarAkunAgen(ctx context.Context, req DaftarAgenRequest) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("daftar agen begin: %w", err)
	}
	defer tx.Rollback()

	var brandExists int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM brands WHERE id = ?`, req.BrandID).Scan(&brandExists); err != nil {
		return fmt.Errorf("daftar agen cek brand: %w", err)
	}
	if brandExists == 0 {
		return ErrNotFound
	}

	canonical, local := shared.PhoneVariants(req.NoHP)
	var existing int64
	err = tx.QueryRowContext(ctx, `
		SELECT id FROM jamaah WHERE brand_id = ? AND REGEXP_REPLACE(COALESCE(no_hp,''),'[^0-9]','') IN (?,?)
		LIMIT 1 FOR UPDATE`, req.BrandID, canonical, local).Scan(&existing)
	if err == nil {
		return ErrNomorTerdaftar
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("daftar agen cek nomor: %w", err)
	}

	jamaahID, err := shared.ResolveJamaah(ctx, tx, req.BrandID, shared.JamaahInput{
		NamaLengkap: req.NamaLengkap,
		NoHP:        req.NoHP,
	})
	if err != nil {
		return err
	}
	pinHash, err := bcrypt.GenerateFromPassword([]byte(req.PortalPIN), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("daftar agen hash pin: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE jamaah SET portal_pin_hash = ? WHERE id = ?`, string(pinHash), jamaahID); err != nil {
		return fmt.Errorf("daftar agen simpan pin: %w", err)
	}

	agenReferral, err := agen.ResolveKodeReferral(ctx, tx, req.BrandID, req.KodeReferral)
	if err != nil {
		return err
	}
	if err := agen.IkatJamaah(ctx, tx, jamaahID, agenReferral, agen.SumberReferral); err != nil {
		return err
	}
	return tx.Commit()
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
