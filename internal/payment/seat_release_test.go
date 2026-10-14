package payment

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"erp-azhan/api/internal/testdb"
)

func seatSnapshot(t *testing.T, tx *sql.Tx, b testdb.Booking) (status string, blocked bool, sisa int) {
	t.Helper()
	if err := tx.QueryRow(`SELECT status,is_seat_blocked FROM bookings WHERE id=?`, b.ID).Scan(&status, &blocked); err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRow(`SELECT seat_sisa FROM schedules WHERE id=?`, b.ScheduleID).Scan(&sisa); err != nil {
		t.Fatal(err)
	}
	return
}

// SM-01: booking berbayar yang kursinya dilepas manual lalu naik dp -> lunas
// harus mengambil kursi lagi, bukan hanya menandainya terpegang.
func TestSyncDpToLunasAfterManualReleaseTakesSeatAgain(t *testing.T) {
	countLunasHook(t)
	_, tx := testdb.Tx(t)
	b := testdb.NewBooking(t, tx, testdb.BookingOpts{Status: "dp", TotalHarga: 20000000, MinimalDP: 5000000, Blocked: false})
	testdb.AddPayment(t, tx, b.ID, 6000000, "confirmed")
	testdb.AddPayment(t, tx, b.ID, 14000000, "confirmed")
	_, _, before := seatSnapshot(t, tx, b)

	if err := SyncBookingStatusTx(context.Background(), tx, b.ID); err != nil {
		t.Fatal(err)
	}
	status, blocked, after := seatSnapshot(t, tx, b)
	if status != "lunas" || !blocked || after != before-1 {
		t.Fatalf("status=%s blocked=%v seat_sisa %d->%d, want lunas/true/%d", status, blocked, before, after, before-1)
	}
}

// SM-01: bila kursinya sudah habis terjual, kenaikan status ditolak dan tidak ada yang berubah.
func TestSyncDpToLunasAfterManualReleaseRejectedWhenSeatTaken(t *testing.T) {
	countLunasHook(t)
	_, tx := testdb.Tx(t)
	b := testdb.NewBooking(t, tx, testdb.BookingOpts{Status: "dp", TotalHarga: 20000000, MinimalDP: 5000000, Blocked: false})
	testdb.AddPayment(t, tx, b.ID, 20000000, "confirmed")
	testdb.Exec(t, tx, `UPDATE schedules SET seat_sisa=0 WHERE id=?`, b.ScheduleID)

	err := SyncBookingStatusTx(context.Background(), tx, b.ID)
	if !errors.Is(err, ErrSeatUnavailable) {
		t.Fatalf("err=%v, want ErrSeatUnavailable", err)
	}
}

// Booking yang masih memegang kursi tidak boleh dikurangi dua kali saat dp -> lunas.
func TestSyncDpToLunasWhileBlockedDoesNotTakeSeatTwice(t *testing.T) {
	countLunasHook(t)
	_, tx := testdb.Tx(t)
	b := testdb.NewBooking(t, tx, testdb.BookingOpts{Status: "dp", TotalHarga: 20000000, MinimalDP: 5000000, Blocked: true})
	testdb.AddPayment(t, tx, b.ID, 20000000, "confirmed")
	_, _, before := seatSnapshot(t, tx, b)

	if err := SyncBookingStatusTx(context.Background(), tx, b.ID); err != nil {
		t.Fatal(err)
	}
	status, blocked, after := seatSnapshot(t, tx, b)
	if status != "lunas" || !blocked || after != before {
		t.Fatalf("status=%s blocked=%v seat_sisa %d->%d, want lunas/true/%d", status, blocked, before, after, before)
	}
}
