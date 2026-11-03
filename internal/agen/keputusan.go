package agen

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

	"erp-azhan/api/internal/identity"
)

// Keputusan admin atas satu pengajuan agen (layar B1, spek bagian 7): satu aksi, satu transaksi.
// Memakai ulang verifikasiPembayaranTx, tolakPembayaranTx, setujuiTx, dan tolakTx.
const (
	AksiSetujui              = "setujui"
	AksiBebaskan             = "bebaskan"
	AksiMintaUnggahUlang     = "minta_unggah_ulang"
	AksiTolak                = "tolak"
	AksiSetujuiBayarMenyusul = "setujui_bayar_menyusul"
)

type KeputusanRequest struct {
	Aksi    string `json:"aksi"`
	Catatan string `json:"catatan"` // wajib untuk semua aksi kecuali setujui; untuk tolak berarti alasan
}

// keputusanTx menjalankan satu aksi. Berlaku untuk akun berstatus 'pengajuan', dan untuk agen aktif
// yang masih punya tagihan tercatat (setujui bayar menyusul): hanya setujui (melunasi) dan bebaskan.
func keputusanTx(ctx context.Context, tx *sql.Tx, akunID int64, brandID *int64, adminID int64, req KeputusanRequest) error {
	req.Aksi = strings.TrimSpace(req.Aksi)
	req.Catatan = strings.TrimSpace(req.Catatan)
	switch req.Aksi {
	case AksiSetujui:
	case AksiBebaskan, AksiMintaUnggahUlang, AksiTolak, AksiSetujuiBayarMenyusul:
		if utf8.RuneCountInString(req.Catatan) < 3 {
			return ErrCatatanWajib
		}
	default:
		return ErrAksiTidakValid
	}

	var status string
	var aBrand int64
	err := tx.QueryRowContext(ctx, `SELECT status_agen, brand_id FROM customer_accounts WHERE id = ? FOR UPDATE`, akunID).Scan(&status, &aBrand)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && brandID != nil && *brandID != aBrand) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("agen.Keputusan kunci akun: %w", err)
	}
	var ppaID int64
	var payStatus, keputusan string
	var nominal float64
	var bukti sql.NullString
	err = tx.QueryRowContext(ctx, `
		SELECT id, status, nominal_tagihan, bukti_transfer_url, keputusan_agen FROM pembayaran_pendaftaran_agen
		WHERE akun_id = ? ORDER BY id DESC LIMIT 1 FOR UPDATE`, akunID).Scan(&ppaID, &payStatus, &nominal, &bukti, &keputusan)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrTidakAdaPengajuan
	}
	if err != nil {
		return fmt.Errorf("agen.Keputusan baca pembayaran: %w", err)
	}
	final := payStatus == "terverifikasi" || payStatus == "dibebaskan"
	pengajuan := status == "pengajuan"
	susulan := status == "aktif" && keputusan == "disetujui" && !final && nominal > 0
	if !pengajuan && !susulan {
		return ErrKeputusanTidakBisa
	}
	catatan := func() error {
		if req.Catatan == "" {
			return nil
		}
		_, err := tx.ExecContext(ctx, `UPDATE pembayaran_pendaftaran_agen SET catatan_admin = ? WHERE id = ?`, req.Catatan, ppaID)
		return err
	}

	switch req.Aksi {
	case AksiSetujui:
		if !final {
			if payStatus != "menunggu_verifikasi" || !bukti.Valid || bukti.String == "" {
				return ErrBuktiBelumAda
			}
			if err := verifikasiPembayaranTx(ctx, tx, ppaID, brandID, adminID); err != nil {
				return err
			}
		}
		if pengajuan {
			if err := setujuiTx(ctx, tx, akunID, brandID, adminID); err != nil {
				return err
			}
		}
		return catatan()
	case AksiBebaskan:
		if final {
			return ErrPembayaranFinal
		}
		if _, err := lockPembayaran(ctx, tx, ppaID, brandID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE pembayaran_pendaftaran_agen SET status = 'dibebaskan', catatan_admin = ?, diverifikasi_oleh = ?, diverifikasi_at = NOW()
			WHERE id = ?`, req.Catatan, adminID, ppaID); err != nil {
			return fmt.Errorf("agen.Keputusan bebaskan: %w", err)
		}
		if pengajuan {
			return setujuiTx(ctx, tx, akunID, brandID, adminID)
		}
		return nil
	case AksiMintaUnggahUlang:
		if err := tolakPembayaranTx(ctx, tx, ppaID, brandID, adminID, req.Catatan); err != nil {
			return err
		}
		return nil // status agen tetap; upload baru mengembalikan tagihan ke antrian (insertUpload)
	case AksiTolak:
		if !pengajuan {
			return ErrKeputusanTidakBisa
		}
		return tolakTx(ctx, tx, akunID, brandID, req.Catatan)
	default: // AksiSetujuiBayarMenyusul
		if !pengajuan || final || nominal <= 0 {
			return ErrKeputusanTidakBisa
		}
		if err := setujuiTx(ctx, tx, akunID, brandID, adminID); err != nil {
			return err
		}
		return catatan()
	}
}

func (r *Repository) Keputusan(ctx context.Context, akunID int64, brandID *int64, adminID int64, req KeputusanRequest) error {
	return r.withTx(ctx, func(tx *sql.Tx) error { return keputusanTx(ctx, tx, akunID, brandID, adminID, req) })
}

// Keputusan POST /api/admin/agen/{akunID}/keputusan {aksi, catatan}
func (h *Handler) Keputusan(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "akunID")
	if !ok {
		return
	}
	var req KeputusanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}
	adminID := identity.GetAdminUserID(r.Context())
	if err := h.repo.Keputusan(r.Context(), id, identity.GetBrandID(r.Context()), adminID, req); err != nil {
		handleError(w, err)
		return
	}
	log.Printf("[AUDIT] agen_keputusan akun_id=%d aksi=%s admin_id=%d", id, strings.TrimSpace(req.Aksi), adminID)
	writeJSON(w, http.StatusOK, map[string]string{"message": "keputusan tersimpan"})
}
