package payment

import (
	"context"
	"testing"

	"erp-azhan/api/internal/testdb"
)

// Tidak ada booking di luar sistem: konfirmasi pembayaran yang mengunci seat booking ikut
// menyamakan seat_count dengan seat yang diambil, sehingga seat_sisa tetap
// seat_total − seat dipegang booking walau seat_count lama lebih kecil dari pax aktif.
func TestRumusSeatKonfirmasiPembayaranSamakanSeatCount(t *testing.T) {
	countLunasHook(t)
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	sched, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	a := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Bayar A"})
	b := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Bayar B"})
	id, _ := testdb.NewBookingPax(t, tx, sched, "baru", 60_000_000, testdb.Pax{JamaahID: a}, testdb.Pax{JamaahID: b})
	// Data lama: belum memegang seat dan seat_count tercatat 1 padahal 2 pax aktif.
	testdb.Exec(t, tx, `UPDATE bookings SET is_seat_blocked=FALSE, seat_count=1 WHERE id=?`, id)
	testdb.SamakanSisa(t, tx, sched)

	testdb.AddPayment(t, tx, id, 60_000_000, "confirmed")
	if err := SyncBookingStatusTx(ctx, tx, id); err != nil {
		t.Fatalf("sinkron status: %v", err)
	}
	var blocked bool
	var seatCount int
	if err := tx.QueryRow(`SELECT is_seat_blocked, seat_count FROM bookings WHERE id=?`, id).Scan(&blocked, &seatCount); err != nil {
		t.Fatal(err)
	}
	if !blocked || seatCount != 2 {
		t.Fatalf("blocked=%v seat_count=%d, want true dan 2", blocked, seatCount)
	}
	testdb.CekRumusSeat(t, tx, sched, "konfirmasi pembayaran")
}
