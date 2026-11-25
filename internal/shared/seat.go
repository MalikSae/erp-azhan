package shared

import (
	"context"
	"database/sql"
	"log"
	"time"
)

// capSeatRestore menghitung sisa kursi setelah n kursi dikembalikan, dibatasi
// kapasitas. capped true bila hasilnya terpotong, tanda sisa sudah lebih besar
// dari seharusnya (SM-04).
func capSeatRestore(total, sisa, n int) (next int, capped bool) {
	next = sisa + n
	if next > total {
		return total, true
	}
	return next, false
}

// RestoreSeats mengembalikan n kursi ke paket dalam transaksi pemanggil. Hasil
// dibatasi seat_total; bila terpotong, dicatat peringatan agar selisih tidak
// tertutup diam-diam seperti pada LEAST() sebelumnya.
func RestoreSeats(ctx context.Context, tx *sql.Tx, scheduleID int64, n int) error {
	if n <= 0 {
		return nil
	}
	var total, sisa int
	if err := tx.QueryRowContext(ctx, `SELECT seat_total, seat_sisa FROM schedules WHERE id=? FOR UPDATE`, scheduleID).Scan(&total, &sisa); err != nil {
		return err
	}
	next, capped := capSeatRestore(total, sisa, n)
	if capped {
		log.Printf("[WARN] seat_restore_capped schedule=%d seat_sisa=%d restore=%d seat_total=%d (sisa sudah melebihi seharusnya, periksa konsistensi kursi)", scheduleID, sisa, n, total)
	}
	_, err := tx.ExecContext(ctx, `UPDATE schedules SET seat_sisa=? WHERE id=?`, next, scheduleID)
	return err
}

// SeatDipegangBooking: jumlah seat yang dipegang booking paket (is_seat_blocked, bukan batal).
// Tidak ada booking di luar sistem, jadi sisa seat yang sah selalu seat_total dikurangi angka ini.
// Satu-satunya tempat rumus ini ditulis di Go; modul schedule memakainya lewat fungsi ini.
func SeatDipegangBooking(ctx context.Context, q rowQuerier, scheduleID int64) (int, error) {
	var n int
	err := q.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(seat_count),0) FROM bookings WHERE schedule_id=? AND is_seat_blocked=1 AND status <> 'batal'`,
		scheduleID).Scan(&n)
	return n, err
}

// SimpanSeatCountTx menyamakan bookings.seat_count dengan jumlah kursi yang baru diambil booking,
// supaya seat_sisa tetap seat_total − seat dipegang booking.
func SimpanSeatCountTx(ctx context.Context, tx *sql.Tx, bookingID int64, n int) error {
	_, err := tx.ExecContext(ctx, `UPDATE bookings SET seat_count=? WHERE id=?`, n, bookingID)
	return err
}

// TodayWIB mengembalikan tanggal hari ini (YYYY-MM-DD) pada zona bisnis WIB.
func TodayWIB() string {
	return time.Now().In(time.FixedZone("WIB", 7*60*60)).Format("2006-01-02")
}

// ScheduleClosedReason mengembalikan alasan bila paket tidak boleh menerima booking
// baru dari admin, atau "" bila boleh (SM-05): paket yang sudah diarsipkan,
// atau tanggal berangkatnya sudah lewat. Paket draft tetap boleh dipesan supaya
// pra-pendaftaran sebelum terbit bisa dicatat. berangkat berformat YYYY-MM-DD.
func ScheduleClosedReason(status, berangkat, today string) string {
	if status == "archived" {
		return "paket sudah diarsipkan dan tidak menerima booking baru"
	}
	if berangkat != "" && berangkat < today {
		return "tanggal keberangkatan paket sudah lewat"
	}
	return ""
}
