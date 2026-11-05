package customer

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
)

// Kartu "Konsultan Anda" / "Leader Anda" di situs: data ringkas agen yang mengajak akun ini (upline). Hanya agen yang masih aktif;
// akun tanpa upline atau dengan upline nonaktif tidak mendapat data. Foto upline privat, hanya disajikan ke akun yang diajaknya.

type UplineView struct {
	NamaLengkap string `json:"nama_lengkap"`
	WhatsApp    string `json:"whatsapp"` // angka saja berawalan 62, siap dipakai di wa.me
	AdaFoto     bool   `json:"ada_foto"`
	fotoURL     string
}

// cariUpline mengembalikan upline aktif akun, atau nil bila tidak ada.
func cariUpline(ctx context.Context, q rowQuerier, akunID int64) (*UplineView, error) {
	var u UplineView
	var hp string
	var foto sql.NullString
	err := q.QueryRowContext(ctx, `
		SELECT up.nama_lengkap, up.no_hp, up.foto_url
		FROM customer_accounts a
		JOIN customer_accounts up ON up.id = a.direkrut_oleh_akun_id AND up.brand_id = a.brand_id
		WHERE a.id = ? AND up.status_agen = 'aktif'`, akunID).Scan(&u.NamaLengkap, &hp, &foto)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("customer: cari upline: %w", err)
	}
	u.WhatsApp = normalisasiHP(hp)
	if foto.Valid && foto.String != "" {
		u.AdaFoto, u.fotoURL = true, foto.String
	}
	return &u, nil
}

// Upline GET /api/account/upline: {"upline": {...} | null}.
func (h *Handler) Upline(w http.ResponseWriter, r *http.Request) {
	acc := FromContext(r.Context())
	u, err := cariUpline(r.Context(), h.repo.db, acc.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan, silakan coba lagi")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"upline": u})
}

// UplineFoto GET /api/account/upline/foto: foto upline aktif milik akun yang login (404 bila tidak ada).
func (h *Handler) UplineFoto(w http.ResponseWriter, r *http.Request) {
	acc := FromContext(r.Context())
	u, err := cariUpline(r.Context(), h.repo.db, acc.ID)
	if err != nil || u == nil || !u.AdaFoto {
		http.NotFound(w, r)
		return
	}
	berkas, ok := berkasFoto(u.fotoURL)
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeFile(w, r, filepath.FromSlash(berkas))
}
