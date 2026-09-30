package komisi

import (
	"context"
	"database/sql"
	"fmt"
)

const (
	JenisLangsung    = "langsung"
	JenisPembinaan   = "pembinaan"
	JenisRepeatOrder = "repeat_order"
	JenisCashback    = "cashback"
)

// paxKomisi adalah satu pax aktif beserta data kaitan agennya.
type paxKomisi struct {
	paxID      int64
	jamaahID   int64
	agenID     sql.NullInt64 // jamaah.direkrut_oleh_jamaah_id
	agenAktif  bool
	uplineID   sql.NullInt64 // upline dari agen tsb
	uplineAktf bool
}

// ProcessBookingLunas mencatat komisi untuk satu booking yang baru mencapai
// 'lunas' (agen-azhan.md Bagian 4, 7.4). Aman dipanggil berulang.
func ProcessBookingLunas(ctx context.Context, tx *sql.Tx, bookingID int64) error {
	// Kunci baris booking dan tandai waktu lunas pertama (sekali saja).
	var brandID int64
	var langsung, pembinaan sql.NullFloat64
	err := tx.QueryRowContext(ctx, `
		SELECT s.brand_id, b.snapshot_komisi_langsung, b.snapshot_bonus_pembinaan
		FROM bookings b JOIN schedules s ON s.id = b.schedule_id
		WHERE b.id = ? FOR UPDATE`, bookingID).Scan(&brandID, &langsung, &pembinaan)
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
		// Tanpa agen (belum_ditentukan / tanpa_agen): tidak ada komisi apa pun,
		// termasuk cashback (agen-azhan.md §6 langkah 11).
		if !p.agenID.Valid {
			continue
		}
		repeat, err := isRepeatOrder(ctx, tx, bookingID, p.jamaahID)
		if err != nil {
			return err
		}

		if !repeat {
			// Pax baru: komisi langsung ke agen, bonus pembinaan ke upline (1 tier).
			if p.agenAktif && positive(langsung) {
				if err := insertKomisi(ctx, tx, brandID, p.agenID.Int64, p, bookingID, JenisLangsung, langsung.Float64); err != nil {
					return err
				}
			}
			if p.uplineID.Valid && p.uplineAktf && positive(pembinaan) {
				if err := insertKomisi(ctx, tx, brandID, p.uplineID.Int64, p, bookingID, JenisPembinaan, pembinaan.Float64); err != nil {
					return err
				}
			}
			continue
		}

		// Repeat order: 50% komisi langsung ke agen, 50% cashback ke jamaah.
		if !positive(langsung) {
			continue
		}
		setengah := float64(int64(langsung.Float64) / 2)
		if p.agenAktif && setengah > 0 {
			if err := insertKomisi(ctx, tx, brandID, p.agenID.Int64, p, bookingID, JenisRepeatOrder, setengah); err != nil {
				return err
			}
		}
		// Cashback tidak bergantung status agen (L1): penerimanya jamaah itu sendiri.
		if setengah > 0 {
			if err := insertKomisi(ctx, tx, brandID, p.jamaahID, p, bookingID, JenisCashback, setengah); err != nil {
				return err
			}
		}
	}
	return nil
}

func positive(v sql.NullFloat64) bool { return v.Valid && v.Float64 > 0 }

// loadPax mengunci baris jamaah pax (FOR SHARE) agar kaitan agen yang dibaca
// tetap berlaku sampai komisi tercatat: Ganti Kaitan Agen (C4) yang datang
// bersamaan menunggu, lalu melihat komisi dan ditolak (audit KA-01).
func loadPax(ctx context.Context, tx *sql.Tx, bookingID int64) ([]paxKomisi, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT bp.id, bp.jamaah_id, j.direkrut_oleh_jamaah_id,
		       COALESCE(a.status_agen = 'aktif', FALSE),
		       a.upline_jamaah_id,
		       COALESCE(u.status_agen = 'aktif', FALSE)
		FROM booking_pax bp
		JOIN jamaah j ON j.id = bp.jamaah_id
		LEFT JOIN jamaah a ON a.id = j.direkrut_oleh_jamaah_id
		LEFT JOIN jamaah u ON u.id = a.upline_jamaah_id
		WHERE bp.booking_id = ? AND bp.pax_status = 'aktif'
		ORDER BY bp.id
		FOR SHARE OF j`, bookingID)
	if err != nil {
		return nil, fmt.Errorf("komisi: baca pax: %w", err)
	}
	defer rows.Close()
	var out []paxKomisi
	for rows.Next() {
		var p paxKomisi
		if err := rows.Scan(&p.paxID, &p.jamaahID, &p.agenID, &p.agenAktif, &p.uplineID, &p.uplineAktf); err != nil {
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
		  (brand_id, jamaah_penerima_id, jamaah_sumber_id, booking_id, booking_pax_id, jenis, nominal)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE id = id`,
		brandID, penerimaID, p.jamaahID, bookingID, p.paxID, jenis, nominal)
	if err != nil {
		return fmt.Errorf("komisi: catat %s: %w", jenis, err)
	}
	return nil
}
