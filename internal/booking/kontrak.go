package booking

import (
	"context"
	"database/sql"
)

// Fungsi di file ini dipakai modul lain (schedule, manifest, manasik) supaya aturan booking
// tidak ditulis ulang di luar modul booking.

// SQLStatusAktif: syarat booking aktif (baru, dp, lunas) untuk tabel bookings beralias b.
const SQLStatusAktif = `b.status IN ('baru','dp','lunas')`

type rowQuerier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type execQuerier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// HitungAktif menghitung booking aktif pada paket. Dalam transaksi, kunci baris paket lebih dulu
// supaya booking yang masuk bersamaan ikut terhitung.
func HitungAktif(ctx context.Context, q rowQuerier, scheduleID int64) (int, error) {
	var n int
	err := q.QueryRowContext(ctx, `SELECT COUNT(*) FROM bookings b WHERE b.schedule_id = ? AND `+SQLStatusAktif, scheduleID).Scan(&n)
	return n, err
}

// CatatProgressManasik mengisi atau mengosongkan penanda manasik satu pax, beserta pencatat dan waktunya.
// Pemanggil memastikan pax memang peserta manasik. Mengembalikan false bila pax tidak ada atau nilainya sudah sama.
func CatatProgressManasik(ctx context.Context, q execQuerier, paxID int64, hadir bool, adminID int64) (bool, error) {
	var oleh any
	if adminID > 0 {
		oleh = adminID
	}
	res, err := q.ExecContext(ctx, `
		UPDATE booking_pax SET progress_manasik = ?, progress_manasik_oleh = ?, progress_manasik_at = NOW()
		WHERE id = ? AND pax_type <> 'infant'`, hadir, oleh, paxID)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}
