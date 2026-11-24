package komisi

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

const (
	JenisLangsung    = "langsung"
	JenisPembinaan   = "pembinaan"
	JenisRepeatOrder = "repeat_order"
	JenisCashback    = "cashback"
)

// KomisiRepeatCashbackAktif menyalakan cabang komisi repeat order (50/50) dan cashback. Dimatikan di tahap awal
// keagenan berbasis akun: booking ulang dihitung seperti booking biasa (komisi langsung dan bonus pembinaan
// penuh, spek 5.1). Logikanya sengaja dipertahankan di bawah flag ini. Berupa variabel agar test dapat
// menyalakannya; jangan diubah di kode produksi. Bila dinyalakan lagi, penerima cashback adalah akun pemesan booking.
var KomisiRepeatCashbackAktif = false

// querier: *sql.Tx atau *sql.DB, untuk pemeriksaan yang juga dipakai ringkasan Detail Booking.
type querier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// paxKomisi adalah satu pax aktif beserta data kaitan agennya.
type paxKomisi struct {
	paxID                 int64
	jamaahID              int64
	penerimaLangsungID    sql.NullInt64 // akun penerima komisi langsung: pemilik kunci jamaah (jamaah.direkrut_oleh_akun_id); pada booking pribadi agen, upline agen itu
	penerimaLangsungAktif bool
	uplineID              sql.NullInt64 // upline akun agen tsb (customer_accounts.upline_akun_id)
	uplineAktif           bool
}

// ProcessBookingLunas mencatat komisi untuk satu booking yang baru mencapai
// 'lunas' (agen-azhan.md Bagian 4, 7.4). Aman dipanggil berulang.
func ProcessBookingLunas(ctx context.Context, tx *sql.Tx, bookingID int64) error {
	// Kunci baris booking dan tandai waktu lunas pertama (sekali saja).
	var brandID, jadwalID int64
	var langsung, pembinaan sql.NullFloat64
	var akunPemesan sql.NullInt64
	var lunasSebelumnya sql.NullTime
	err := tx.QueryRowContext(ctx, `
		SELECT s.brand_id, b.schedule_id, b.snapshot_komisi_langsung, b.snapshot_bonus_pembinaan, b.customer_account_id, b.pertama_lunas_at
		FROM bookings b JOIN schedules s ON s.id = b.schedule_id
		WHERE b.id = ? FOR UPDATE`, bookingID).Scan(&brandID, &jadwalID, &langsung, &pembinaan, &akunPemesan, &lunasSebelumnya)
	if err != nil {
		return fmt.Errorf("komisi: baca booking %d: %w", bookingID, err)
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE bookings SET pertama_lunas_at = NOW() WHERE id = ? AND pertama_lunas_at IS NULL`, bookingID); err != nil {
		return fmt.Errorf("komisi: tandai lunas pertama: %w", err)
	}

	// Snapshot NULL = booking tidak ikut program Syiar (D9).
	if !langsung.Valid && !pembinaan.Valid {
		return nil
	}

	paxes, err := loadPax(ctx, tx, bookingID)
	if err != nil {
		return err
	}

	for _, p := range paxes {
		// Tanpa agen: tidak ada komisi apa pun,
		// termasuk cashback (agen-azhan.md §6 langkah 11).
		if !p.penerimaLangsungID.Valid {
			continue
		}
		// Pengaitan admin berlaku ke depan (keputusan 2026-11-24): booking yang sudah pernah lunas sebelum jamaah
		// tanpa agen dikaitkan tetap tanpa komisi, juga saat turun lalu lunas lagi.
		if lunasSebelumnya.Valid {
			setelah, err := dikaitkanSetelah(ctx, tx, p.jamaahID, lunasSebelumnya.Time)
			if err != nil {
				return err
			}
			if setelah {
				continue
			}
		}
		repeat := false
		if KomisiRepeatCashbackAktif {
			var err error
			if repeat, err = isRepeatOrder(ctx, tx, bookingID, p.jamaahID); err != nil {
				return err
			}
		}

		if !repeat {
			// Pax baru: komisi langsung ke agen, bonus pembinaan ke upline (1 tier).
			// Satu komisi langsung dan satu bonus pembinaan per jamaah per keberangkatan (ADR 0004).
			if p.penerimaLangsungAktif && positive(langsung) {
				ada, err := sudahDibayarKeberangkatan(ctx, tx, p.jamaahID, jadwalID, bookingID, JenisLangsung)
				if err != nil {
					return err
				}
				if !ada {
					if err := insertKomisi(ctx, tx, brandID, p.penerimaLangsungID.Int64, p, bookingID, JenisLangsung, langsung.Float64); err != nil {
						return err
					}
				}
			}
			if p.uplineID.Valid && p.uplineAktif && positive(pembinaan) {
				ada, err := sudahDibayarKeberangkatan(ctx, tx, p.jamaahID, jadwalID, bookingID, JenisPembinaan)
				if err != nil {
					return err
				}
				if !ada {
					if err := insertKomisi(ctx, tx, brandID, p.uplineID.Int64, p, bookingID, JenisPembinaan, pembinaan.Float64); err != nil {
						return err
					}
				}
			}
			continue
		}

		// Repeat order: 50% komisi langsung ke agen, 50% cashback ke akun pemesan booking.
		if !positive(langsung) {
			continue
		}
		setengah := float64(int64(langsung.Float64) / 2)
		if p.penerimaLangsungAktif && setengah > 0 {
			if err := insertKomisi(ctx, tx, brandID, p.penerimaLangsungID.Int64, p, bookingID, JenisRepeatOrder, setengah); err != nil {
				return err
			}
		}
		// Cashback tidak bergantung status agen (L1): penerimanya akun pemesan booking.
		if setengah > 0 && akunPemesan.Valid {
			if err := insertKomisi(ctx, tx, brandID, akunPemesan.Int64, p, bookingID, JenisCashback, setengah); err != nil {
				return err
			}
		}
	}
	return nil
}

// dikaitkanSetelah: jamaah tanpa agen dikaitkan ke agen oleh admin setelah waktu t.
func dikaitkanSetelah(ctx context.Context, tx querier, jamaahID int64, t time.Time) (bool, error) {
	var ada bool
	err := tx.QueryRowContext(ctx, `
		SELECT EXISTS (SELECT 1 FROM jamaah_kaitan_log WHERE jamaah_id = ? AND kaitan_status_lama = 'tanpa_agen'
		               AND kaitan_status_baru = 'terikat_agen' AND diganti_at > ?)`, jamaahID, t).Scan(&ada)
	if err != nil {
		return false, fmt.Errorf("komisi: cek waktu pengaitan: %w", err)
	}
	return ada, nil
}

// sudahDibayarKeberangkatan: komisi aktif jenis itu untuk jamaah yang sama di booking lain pada jadwal yang sama
// (mis. booking sebelumnya batal setelah lunas; komisi lunas tidak ditarik saat batal).
func sudahDibayarKeberangkatan(ctx context.Context, tx querier, jamaahID, jadwalID, bookingID int64, jenis string) (bool, error) {
	var ada bool
	err := tx.QueryRowContext(ctx, `
		SELECT EXISTS (
		    SELECT 1 FROM transaksi_komisi tk JOIN bookings b2 ON b2.id = tk.booking_id
		    WHERE tk.jamaah_sumber_id = ? AND tk.jenis = ? AND tk.dibatalkan_at IS NULL
		      AND tk.booking_id <> ? AND b2.schedule_id = ?)`, jamaahID, jenis, bookingID, jadwalID).Scan(&ada)
	if err != nil {
		return false, fmt.Errorf("komisi: cek komisi keberangkatan: %w", err)
	}
	return ada, nil
}

func positive(v sql.NullFloat64) bool { return v.Valid && v.Float64 > 0 }

// Booking pribadi agen: agen tidak mendapat komisi dari booking dirinya sendiri. Pax dianggap "diri agen" bila ditandai "Saya sendiri"
// (booking_pax.saya_sendiri_akun_id) atau jamaahnya tertaut ke akun (customer_accounts.jamaah_id), jalur booking apa pun.
// NIK yang sama dengan nik_agen bukan lagi penentu (docs/adr/0004-diri-agen-dan-akun-jamaah.md). Penerima komisi langsung untuk pax diri:
//   - jamaah terikat_agen dengan pemilik kunci bukan akun itu sendiri: pemilik kunci (jamaah.direkrut_oleh_akun_id);
//   - selain itu, akun agen aktif atau nonaktif: upline agen (upline_akun_id), termasuk jamaah diri yang tanpa_agen
//     (sejak migrasi 158 status itu juga mencakup bekas belum_ditentukan);
//   - pax bukan diri yang tanpa_agen: tidak ada komisi.
//
// Bonus pembinaan ke upline penerima (satu tingkat). Penerima nonaktif: komisi langsung hangus, tanpa backfill saat
// diaktifkan lagi; bonus pembinaan tetap ke upline aktif di atasnya. Pax bukan diri: pemilik kunci jamaah.

// joinPenerimaSQL menurunkan penerima komisi per pax (alias bp = booking_pax, j = jamaah): diri = akun milik pax,
// a = penerima komisi langsung menurut kaitan saat ini, u = upline a (penerima bonus pembinaan). Satu sumber untuk
// pencatatan komisi (loadPax) dan ringkasan Detail Booking (RingkasanBooking).
const joinPenerimaSQL = `
		LEFT JOIN customer_accounts diri ON diri.id = COALESCE(bp.saya_sendiri_akun_id,
		     (SELECT ca.id FROM customer_accounts ca WHERE ca.jamaah_id = j.id LIMIT 1))
		LEFT JOIN customer_accounts a ON a.id = CASE
		     WHEN diri.id IS NULL THEN IF(j.kaitan_status = 'tanpa_agen', NULL, j.direkrut_oleh_akun_id)
		     WHEN j.kaitan_status = 'terikat_agen' AND j.direkrut_oleh_akun_id <> diri.id THEN j.direkrut_oleh_akun_id
		     WHEN diri.status_agen IN ('aktif','nonaktif') THEN diri.upline_akun_id
		     ELSE NULL END
		LEFT JOIN customer_accounts u ON u.id = a.upline_akun_id`

// loadPax mengunci baris jamaah pax (FOR SHARE) agar kaitan agen yang dibaca
// tetap berlaku sampai komisi tercatat: Ganti Kaitan Agen (C4) yang datang
// bersamaan menunggu, lalu melihat komisi dan ditolak (audit KA-01).
func loadPax(ctx context.Context, tx *sql.Tx, bookingID int64) ([]paxKomisi, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT bp.id, bp.jamaah_id, a.id,
		       COALESCE(a.status_agen = 'aktif', FALSE),
		       a.upline_akun_id,
		       COALESCE(u.status_agen = 'aktif', FALSE)
		FROM booking_pax bp
		JOIN jamaah j ON j.id = bp.jamaah_id
		`+joinPenerimaSQL+`
		WHERE bp.booking_id = ? AND bp.pax_status = 'aktif' AND bp.pax_type <> 'infant'
		ORDER BY bp.id
		FOR SHARE OF j`, bookingID)
	if err != nil {
		return nil, fmt.Errorf("komisi: baca pax: %w", err)
	}
	defer rows.Close()
	var out []paxKomisi
	for rows.Next() {
		var p paxKomisi
		if err := rows.Scan(&p.paxID, &p.jamaahID, &p.penerimaLangsungID, &p.penerimaLangsungAktif, &p.uplineID, &p.uplineAktif); err != nil {
			return nil, fmt.Errorf("komisi: scan pax: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// isRepeatOrder: jamaah sudah punya pax aktif di booking lain yang lunas lebih
// dulu dari booking ini (keputusan D5). Urutan memakai pertama_lunas_at, lalu id
// sebagai pemecah seri, sehingga klasifikasi stabil saat dipanggil ulang.
func isRepeatOrder(ctx context.Context, tx *sql.Tx, bookingID, jamaahID int64) (bool, error) {
	var n int
	err := tx.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM booking_pax bp2
		JOIN bookings b2 ON b2.id = bp2.booking_id
		JOIN bookings b ON b.id = ?
		WHERE bp2.jamaah_id = ? AND bp2.pax_status = 'aktif'
		  AND b2.id <> b.id
		  AND b2.pertama_lunas_at IS NOT NULL
		  AND (b2.pertama_lunas_at < b.pertama_lunas_at
		       OR (b2.pertama_lunas_at = b.pertama_lunas_at AND b2.id < b.id))`,
		bookingID, jamaahID).Scan(&n)
	if err != nil {
		return false, fmt.Errorf("komisi: cek repeat order: %w", err)
	}
	return n > 0, nil
}

func insertKomisi(ctx context.Context, tx *sql.Tx, brandID, penerimaID int64, p paxKomisi, bookingID int64, jenis string, nominal float64) error {
	// ON DUPLICATE KEY (bukan INSERT IGNORE) supaya error lain seperti FK tetap muncul.
	_, err := tx.ExecContext(ctx, `
		INSERT INTO transaksi_komisi
		  (brand_id, akun_penerima_id, jamaah_sumber_id, booking_id, booking_pax_id, jenis, nominal)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE id = id`,
		brandID, penerimaID, p.jamaahID, bookingID, p.paxID, jenis, nominal)
	if err != nil {
		return fmt.Errorf("komisi: catat %s: %w", jenis, err)
	}
	return nil
}
