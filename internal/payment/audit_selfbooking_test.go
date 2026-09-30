package payment

import (
	"context"
	"database/sql"
	"erp-azhan/api/internal/testdb"
	"errors"
	"testing"
	"time"
)

// Regression tests converted from audit probes; fixtures always roll back.
func TestPendingReviewExtendsOnceAndNeverRevivesExpiredHold(t *testing.T) {
	_, tx := testdb.Tx(t)
	b := testdb.NewBooking(t, tx, testdb.BookingOpts{Status: "baru", TotalHarga: 20000000, MinimalDP: 5000000, Blocked: true})
	testdb.Exec(t, tx, `UPDATE bookings SET seat_hold_expires_at=DATE_ADD(NOW(),INTERVAL 12 HOUR) WHERE id=?`, b.ID)
	testdb.Exec(t, tx, `INSERT INTO booking_checkout(booking_id,dp_per_pax,due_at,original_expires_at,invoice_token,terms_version) SELECT id,5000000,seat_hold_expires_at,seat_hold_expires_at,REPEAT('a',64),'test' FROM bookings WHERE id=?`, b.ID)
	var original, first, second time.Time
	if err := tx.QueryRow(`SELECT seat_hold_expires_at FROM bookings WHERE id=?`, b.ID).Scan(&original); err != nil {
		t.Fatal(err)
	}
	if err := extendPendingReviewTx(context.Background(), tx, b.ID); err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRow(`SELECT seat_hold_expires_at FROM bookings WHERE id=?`, b.ID).Scan(&first); err != nil {
		t.Fatal(err)
	}
	if first.Sub(original) != 24*time.Hour {
		t.Fatalf("extension = %s", first.Sub(original))
	}
	if err := extendPendingReviewTx(context.Background(), tx, b.ID); err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRow(`SELECT seat_hold_expires_at FROM bookings WHERE id=?`, b.ID).Scan(&second); err != nil {
		t.Fatal(err)
	}
	if !second.Equal(first) {
		t.Fatal("second upload extended hold again")
	}
	testdb.Exec(t, tx, `UPDATE booking_checkout SET review_extended=FALSE,original_expires_at=DATE_SUB(NOW(),INTERVAL 1 HOUR) WHERE booking_id=?`, b.ID)
	if err := extendPendingReviewTx(context.Background(), tx, b.ID); err != nil {
		t.Fatal(err)
	}
	var extended bool
	if err := tx.QueryRow(`SELECT review_extended FROM booking_checkout WHERE booking_id=?`, b.ID).Scan(&extended); err != nil {
		t.Fatal(err)
	}
	if extended {
		t.Fatal("late upload revived expired hold")
	}
}

func TestConfirmationRejectsCombinedOverpayment(t *testing.T) {
	_, tx := testdb.Tx(t)
	b := testdb.NewBooking(t, tx, testdb.BookingOpts{Status: "dp", TotalHarga: 20000000, MinimalDP: 5000000, Blocked: true})
	testdb.Exec(t, tx, `INSERT INTO payments(booking_id,jumlah,metode,tanggal,status) VALUES (?,15000000,'transfer',CURRENT_DATE,'confirmed')`, b.ID)
	ctx := context.Background()
	if err := checkConfirmationBalanceTx(ctx, tx, b.ID, 0, 10000000, 20000000); !errors.Is(err, ErrOverpayment) {
		t.Fatalf("overpayment accepted: %v", err)
	}
	if err := checkConfirmationBalanceTx(ctx, tx, b.ID, 0, 5000000, 20000000); err != nil {
		t.Fatalf("exact balance rejected: %v", err)
	}
}

func TestAuditSelfBookingZeroDPOverrideIgnored(t *testing.T) {
	db, tx := testdb.Tx(t)
	b := testdb.NewBooking(t, tx, testdb.BookingOpts{Status: "baru", TotalHarga: 20000000, MinimalDP: 0, Blocked: true})
	testdb.Exec(t, tx, `UPDATE brands br JOIN schedules s ON s.brand_id=br.id SET br.minimal_dp=5000000 WHERE s.id=?`, b.ScheduleID)
	testdb.Exec(t, tx, `INSERT INTO payments(booking_id,jumlah,metode,tanggal,status) VALUES (?,1000000,'transfer',CURRENT_DATE,'confirmed')`, b.ID)
	if err := NewRepository(db).syncBookingStatusTx(context.Background(), tx, b.ID); err != nil {
		t.Fatal(err)
	}
	var status string
	if err := tx.QueryRow(`SELECT status FROM bookings WHERE id=?`, b.ID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "dp" {
		t.Fatalf("probe changed: %s", status)
	}
}

func TestAuditSelfBookingDPRemainsAfterPaymentRemoved(t *testing.T) {
	db, tx := testdb.Tx(t)
	b := testdb.NewBooking(t, tx, testdb.BookingOpts{Status: "dp", TotalHarga: 20000000, MinimalDP: 5000000, Blocked: true})
	if err := NewRepository(db).syncBookingStatusTx(context.Background(), tx, b.ID); err != nil {
		t.Fatal(err)
	}
	var status string
	if err := tx.QueryRow(`SELECT status FROM bookings WHERE id=?`, b.ID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "baru" {
		t.Fatalf("probe changed: %s", status)
	}
	var expiry sql.NullTime
	if err := tx.QueryRow(`SELECT seat_hold_expires_at FROM bookings WHERE id=?`, b.ID).Scan(&expiry); err != nil || !expiry.Valid {
		t.Fatalf("missing renewed hold: %v", err)
	}
}

// JB-03: booking yang sudah punya pembayaran terkonfirmasi lalu turun ke 'baru'
// tidak boleh mendapat hold 24 jam, agar worker tidak melepas kursinya.
func TestSyncPaidDowngradeKeepsSeatWithoutExpiry(t *testing.T) {
	db, tx := testdb.Tx(t)
	b := testdb.NewBooking(t, tx, testdb.BookingOpts{Status: "dp", TotalHarga: 20000000, MinimalDP: 5000000, Blocked: true})
	testdb.AddPayment(t, tx, b.ID, 6000000, "confirmed")
	// Minimal DP naik di atas jumlah yang dibayar -> status turun ke baru.
	testdb.Exec(t, tx, `UPDATE schedules SET minimal_dp=8000000 WHERE id=?`, b.ScheduleID)
	if err := NewRepository(db).syncBookingStatusTx(context.Background(), tx, b.ID); err != nil {
		t.Fatal(err)
	}
	var status string
	var blocked bool
	var expiry sql.NullTime
	if err := tx.QueryRow(`SELECT status,is_seat_blocked,seat_hold_expires_at FROM bookings WHERE id=?`, b.ID).Scan(&status, &blocked, &expiry); err != nil {
		t.Fatal(err)
	}
	if status != "baru" || !blocked || expiry.Valid {
		t.Fatalf("status=%s blocked=%v expiry=%v, want baru/true/NULL", status, blocked, expiry)
	}
}
