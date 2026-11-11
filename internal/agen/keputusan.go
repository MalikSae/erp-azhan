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
// yang masih punya tagihan tercatat (setujui bayar menyusul): setujui (melunasi), bebaskan, atau minta unggah ulang bila bukti susulan tidak terbaca.
func keputusanTx(ctx context.Context, tx *sql.Tx, akunID int64, brandID *int64, adminID int64, req KeputusanRequest) error {
	req.Aksi = strings.TrimSpace(req.Aksi)
	req.Catatan = strings.TrimSpace(req.Catatan)
	ppaID, err := prosesKeputusanTx(ctx, tx, akunID, brandID, adminID, req)
	if err != nil {
		return err
	}
	return catatRiwayatKeputusan(ctx, tx, ppaID, akunID, adminID, req.Aksi, req.Catatan)
}

// catatRiwayatKeputusan menyimpan satu keputusan beserta catatan dan pelakunya. Kolom catatan_admin di siklus
// tagihan hanya memuat catatan terakhir; riwayat ini tidak pernah tertimpa.
func catatRiwayatKeputusan(ctx context.Context, tx *sql.Tx, ppaID, akunID, adminID int64, aksi, catatan string) error {
	var pelaku, isi any
	if adminID > 0 {
		pelaku = adminID
	}
	if catatan != "" {
		isi = catatan
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO keputusan_agen_log (pembayaran_pendaftaran_agen_id, akun_id, aksi, catatan, admin_id) VALUES (?, ?, ?, ?, ?)`,
		ppaID, akunID, aksi, isi, pelaku); err != nil {
		return fmt.Errorf("agen: catat riwayat keputusan: %w", err)
	}
	return nil
}

// prosesKeputusanTx menjalankan aksinya dan mengembalikan id siklus tagihan yang diputuskan.
func prosesKeputusanTx(ctx context.Context, tx *sql.Tx, akunID int64, brandID *int64, adminID int64, req KeputusanRequest) (int64, error) {
	var ppaID int64
	err := func() error {
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
		final := pembayaranFinal(payStatus)
		pengajuan := status == StatusAgenPengajuan
		// Tagihan agen yang sudah disetujui tetap bisa diselesaikan walau agennya dinonaktifkan: status keagenan
		// dan utang biaya pendaftaran dua hal terpisah.
		susulan := sudahAgen(status) && keputusan == keputusanDisetujui && !final && nominal > 0
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
				if payStatus != bayarMenungguVerifikasi || !bukti.Valid || bukti.String == "" {
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
			return tolakTx(ctx, tx, akunID, brandID, adminID, req.Catatan)
		default: // AksiSetujuiBayarMenyusul
			if !pengajuan || final || nominal <= 0 {
				return ErrKeputusanTidakBisa
			}
			if err := setujuiTx(ctx, tx, akunID, brandID, adminID); err != nil {
				return err
			}
			return catatan()
		}
	}()
	return ppaID, err
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
