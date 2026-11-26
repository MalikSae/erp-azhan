package booking

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"erp-azhan/api/internal/payment"
	"erp-azhan/api/internal/testdb"
)

// Audit booking 2026-11-26.

// Bug 1: batal pax pada booking 'baru' yang sudah menerima pembayaran di bawah DP tidak boleh membuat tagihan
// lebih kecil dari uang masuk (dulu booking jadi lunas dengan kelebihan bayar dan komisi tercatat).
func TestBatalPaxDitolakBilaTagihanDiBawahDibayar(t *testing.T) {
	db, tx := testdb.Tx(t)
	repo := NewRepository(db)
	ctx := context.Background()
	sch, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2030-01-01"})
	testdb.Exec(t, tx, `UPDATE schedules SET minimal_dp=50 WHERE id=?`, sch)
	j1 := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "A"})
	j2 := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "B"})
	bid, pax := testdb.NewBookingPax(t, tx, sch, "baru", 100, testdb.Pax{JamaahID: j1}, testdb.Pax{JamaahID: j2})
	testdb.AddPayment(t, tx, bid, 80, "confirmed")
	if err := payment.SyncBookingStatusTx(ctx, tx, bid); err != nil {
		t.Fatal(err)
	}
	if got := testdb.Status(t, tx, bid); got != "baru" {
		t.Fatalf("status awal = %s, want baru (80 di bawah DP wajib 100)", got)
	}

	err := repo.cancelPaxTx(ctx, tx, bid, pax[1], nil, true)
	var bisnis *BusinessError
	if !errors.As(err, &bisnis) {
		t.Fatalf("batal pax: err = %v, want BusinessError tagihan di bawah pembayaran", err)
	}
	t.Logf("galat: %s", bisnis.Message)
}

// Bug 2a: pembayaran terkonfirmasi di bawah DP (status tetap 'baru') menghapus batas hold, dan worker tidak
// melepas seat booking itu.
func TestHoldBookingBerbayarTidakKedaluwarsa(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	b := testdb.NewBooking(t, tx, testdb.BookingOpts{Status: "baru", TotalHarga: 1000, MinimalDP: 500, Blocked: true})
	testdb.Exec(t, tx, `UPDATE bookings SET seat_hold_expires_at=DATE_SUB(UTC_TIMESTAMP(), INTERVAL 1 MINUTE) WHERE id=?`, b.ID)
	testdb.AddPayment(t, tx, b.ID, 200, "confirmed")
	if err := payment.SyncBookingStatusTx(ctx, tx, b.ID); err != nil {
		t.Fatal(err)
	}
	if exp := batasHold(t, tx, b.ID); exp.Valid {
		t.Fatalf("batas hold setelah bayar 200 = %v, want NULL", exp.Time)
	}
	released, err := releaseSingleSeatHoldTx(ctx, tx, b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if released || !seatDipegang(t, tx, b.ID) {
		t.Fatalf("released=%v blocked=%v, want seat tetap dipegang", released, seatDipegang(t, tx, b.ID))
	}
}

// Bug 2b: worker tidak melepas seat booking yang punya pembayaran terkonfirmasi walau batas hold masih tercatat
// (data lama atau jalur lain), dan batasnya dihapus.
func TestWorkerHoldMelewatiBookingBerbayar(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	b := testdb.NewBooking(t, tx, testdb.BookingOpts{Status: "baru", TotalHarga: 1000, MinimalDP: 500, Blocked: true})
	testdb.AddPayment(t, tx, b.ID, 200, "confirmed")
	testdb.Exec(t, tx, `UPDATE bookings SET seat_hold_expires_at=DATE_SUB(UTC_TIMESTAMP(), INTERVAL 1 MINUTE) WHERE id=?`, b.ID)

	released, err := releaseSingleSeatHoldTx(ctx, tx, b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if released || !seatDipegang(t, tx, b.ID) {
		t.Fatalf("released=%v, want seat tetap dipegang", released)
	}
	if exp := batasHold(t, tx, b.ID); exp.Valid {
		t.Fatalf("batas hold = %v, want dihapus", exp.Time)
	}
}

// Bug 2c: hold manual dengan batas waktu pada booking 'baru' yang sudah dibayar sebagian dipasang tanpa batas.
func TestHoldManualBookingBerbayarTanpaBatas(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	b := testdb.NewBooking(t, tx, testdb.BookingOpts{Status: "baru", TotalHarga: 1000, MinimalDP: 500, Blocked: false})
	testdb.AddPayment(t, tx, b.ID, 200, "confirmed")
	batas := time.Now().Add(2 * time.Hour)
	if err := blockSeatTx(ctx, tx, b.ID, &batas, "uji-hold-berbayar"); err != nil {
		t.Fatalf("hold: %v", err)
	}
	if !seatDipegang(t, tx, b.ID) {
		t.Fatal("seat belum dipegang setelah hold")
	}
	if exp := batasHold(t, tx, b.ID); exp.Valid {
		t.Fatalf("batas hold = %v, want NULL untuk booking berbayar", exp.Time)
	}
}

// Bug 7: draft tidak bisa dibatalkan (lewat batal pax maupun ubah status), supaya tetap bisa dihapus.
func TestDraftTidakBisaDibatalkan(t *testing.T) {
	db, tx := testdb.Tx(t)
	repo := NewRepository(db)
	ctx := context.Background()
	sch, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2030-01-01"})
	j := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Draft"})
	bid, pax := testdb.NewBookingPax(t, tx, sch, "draft", 100, testdb.Pax{JamaahID: j})
	testdb.Exec(t, tx, `UPDATE bookings SET is_seat_blocked=FALSE WHERE id=?`, bid)

	if err := repo.cancelPaxTx(ctx, tx, bid, pax[0], nil, true); !errors.Is(err, errDraftTidakDibatalkan) {
		t.Fatalf("batal pax draft: err = %v, want errDraftTidakDibatalkan", err)
	}
	if err := updateBookingStatusTx(ctx, tx, bid, "batal", nil); !errors.Is(err, errDraftTidakDibatalkan) {
		t.Fatalf("status batal draft: err = %v, want errDraftTidakDibatalkan", err)
	}
	if got := testdb.Status(t, tx, bid); got != "draft" {
		t.Fatalf("status = %s, want tetap draft", got)
	}
}

func batasHold(t *testing.T, tx *sql.Tx, bookingID int64) sql.NullTime {
	t.Helper()
	var v sql.NullTime
	if err := tx.QueryRow(`SELECT seat_hold_expires_at FROM bookings WHERE id=?`, bookingID).Scan(&v); err != nil {
		t.Fatal(err)
	}
	return v
}

func seatDipegang(t *testing.T, tx *sql.Tx, bookingID int64) bool {
	t.Helper()
	var v bool
	if err := tx.QueryRow(`SELECT is_seat_blocked FROM bookings WHERE id=?`, bookingID).Scan(&v); err != nil {
		t.Fatal(err)
	}
	return v
}
