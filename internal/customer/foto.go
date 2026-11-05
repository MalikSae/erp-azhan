package customer

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"path"
	"path/filepath"

	"erp-azhan/api/internal/agen"
)

// Foto profil akun: satu foto per akun, tidak peduli sudah menjadi agen atau belum. Berkasnya diunggah lewat
// POST /api/account/media/upload (kategori dokumen-jamaah), lalu URL-nya dikirim ke PUT /api/account/foto.
// Foto privat: pemilik membukanya lewat GET /api/account/foto, admin lewat media terlindungi.

var ErrFotoTidakValid = errors.New("foto tidak valid, unggah ulang (JPG, PNG, WEBP, atau GIF)")

type fotoDB interface {
	adminQuerier
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// simpanFotoAkun menyimpan url foto hasil unggahan akun ini; url kosong menghapus foto.
func simpanFotoAkun(ctx context.Context, db fotoDB, akunID int64, url string) error {
	if url == "" {
		if _, err := db.ExecContext(ctx, `UPDATE customer_accounts SET foto_url = NULL WHERE id = ?`, akunID); err != nil {
			return fmt.Errorf("customer: hapus foto: %w", err)
		}
		return nil
	}
	if !agen.FotoPortalURL.MatchString(url) {
		return ErrFotoTidakValid
	}
	ok, err := agen.MediaMilikAkun(ctx, db, akunID, url)
	if err != nil {
		return err
	}
	if !ok {
		return ErrFotoTidakValid
	}
	if _, err := db.ExecContext(ctx, `UPDATE customer_accounts SET foto_url = ? WHERE id = ?`, url, akunID); err != nil {
		return fmt.Errorf("customer: simpan foto: %w", err)
	}
	return nil
}

// berkasFoto menerjemahkan url foto menjadi lokasi berkas; hanya bentuk url foto portal yang diterima.
func berkasFoto(url string) (string, bool) {
	if !agen.FotoPortalURL.MatchString(url) {
		return "", false
	}
	return "uploads/dokumen-jamaah/" + path.Base(url), true
}

type fotoRequest struct {
	FotoURL string `json:"foto_url"`
}

// UpdateFoto PUT /api/account/foto {foto_url}: kosong = hapus foto.
func (h *Handler) UpdateFoto(w http.ResponseWriter, r *http.Request) {
	acc := FromContext(r.Context())
	var req fotoRequest
	if !decode(w, r, &req) {
		return
	}
	if err := simpanFotoAkun(r.Context(), h.repo.db, acc.ID, req.FotoURL); err != nil {
		if errors.Is(err, ErrFotoTidakValid) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		log.Printf("[ERROR] customer simpan foto: %v", err)
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan, silakan coba lagi")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "foto diperbarui"})
}

// Foto GET /api/account/foto: menyajikan foto milik akun yang login (404 bila belum ada).
func (h *Handler) Foto(w http.ResponseWriter, r *http.Request) {
	acc := FromContext(r.Context())
	if acc.FotoURL == nil {
		http.NotFound(w, r)
		return
	}
	berkas, ok := berkasFoto(*acc.FotoURL)
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeFile(w, r, filepath.FromSlash(berkas))
}
