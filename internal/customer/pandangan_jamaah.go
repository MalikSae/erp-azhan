package customer

import (
	"context"
	"database/sql"
)

// Peran jamaah (ADR 0005, direvisi 18 November 2026): akun yang tertaut ke jamaah (customer_accounts.jamaah_id,
// lahir dari aktivasi) melihat booking orang lain yang memuat jamaah itu lewat endpoint booking biasa
// (owned() menandai Peran = "jamaah"). Booking tampil utuh; aksi per jamaah dibatasi ke dirinya (bolehJamaah)
// dan pembatalan tetap hak pemesan.

// sqlBookingPeran: booking akun lain (bukan draft) yang memuat pax aktif jamaah tertaut. Booking batal ikut,
// sama dengan Data Jamaah pemesan dan daftar booking (keputusan 18 November 2026: tetap terlihat sebagai riwayat).
// Parameter: id jamaah tertaut, id akun.
const sqlBookingPeran = `
	FROM booking_pax bp
	JOIN bookings b ON b.id = bp.booking_id
	JOIN schedules s ON s.id = b.schedule_id
	JOIN jamaah j ON j.id = bp.jamaah_id
	WHERE bp.jamaah_id = ? AND bp.pax_status = 'aktif'
	  AND NOT (b.customer_account_id <=> ?)
	  AND b.status <> 'draft'`

// jamaahTertaut: id jamaah yang tertaut ke akun; ok=false bila tidak tertaut.
func jamaahTertaut(ctx context.Context, q dbq, akunID int64) (int64, bool, error) {
	var id sql.NullInt64
	err := q.QueryRowContext(ctx, `SELECT jamaah_id FROM customer_accounts WHERE id = ?`, akunID).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return id.Int64, id.Valid, nil
}

// muatAnggotaPeran: baris Data Jamaah dari booking orang lain, hanya baris pax milik jamaah tertaut.
func muatAnggotaPeran(ctx context.Context, q dbq, akunID int64) ([]anggotaJamaah, error) {
	jid, ok, err := jamaahTertaut(ctx, q, akunID)
	if err != nil || !ok {
		return nil, err
	}
	rows, err := q.QueryContext(ctx, `
		SELECT j.id, j.nama_lengkap, COALESCE(DATE_FORMAT(j.tanggal_lahir,'%Y-%m-%d'),''), bp.pax_type,
		       b.id, b.id_booking, b.status, s.jadwal_nama, COALESCE(DATE_FORMAT(s.berangkat_tanggal,'%Y-%m-%d'),''),
		       `+sqlAksesPenuh+sqlBookingPeran+` ORDER BY b.id DESC`, akunID, jid, akunID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []anggotaJamaah
	for rows.Next() {
		var a anggotaJamaah
		if err := rows.Scan(&a.ID, &a.Nama, &a.Lahir, &a.PaxType, &a.BookingID, &a.BookingKode, &a.StatusBooking, &a.Paket, &a.Berangkat, &a.Penuh); err != nil {
			return nil, err
		}
		a.DariPeran = true
		out = append(out, a)
	}
	return out, rows.Err()
}
