package selfbooking

import (
	"context"
	"erp-azhan/api/internal/testdb"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestCheckoutReplayDoesNotReserveTwice(t *testing.T) {
	tx, scheduleID, brandID, agentID := setupAgenBooking(t)
	req := BookingRequest{RequestKey: strings.Repeat("a", 48), TermsVersion: TermsVersion, TermsAccepted: true, ScheduleID: scheduleID, PIC: PICInput{NamaLengkap: "Retry PIC", NoHP: nomorUji(980), JenisKelamin: "L", RoomType: "Quad"}}
	first, err := processBookingTx(context.Background(), tx, brandID, req, inisiator{agenID: agentID})
	if err != nil {
		t.Fatal(err)
	}
	var remaining int
	tx.QueryRow(`SELECT seat_sisa FROM schedules WHERE id=?`, scheduleID).Scan(&remaining)
	second, err := processBookingTx(context.Background(), tx, brandID, req, inisiator{agenID: agentID})
	if err != nil {
		t.Fatal(err)
	}
	var after int
	tx.QueryRow(`SELECT seat_sisa FROM schedules WHERE id=?`, scheduleID).Scan(&after)
	if first.Booking.BookingCode != second.Booking.BookingCode || first.Booking.InvoiceToken != second.Booking.InvoiceToken || remaining != after {
		t.Fatal("retry changed booking or seats")
	}
	if !second.Replayed || first.Replayed {
		t.Fatal("retry must not consume the successful-booking phone quota")
	}
	req.PIC.NamaLengkap = "Changed request"
	if _, err = processBookingTx(context.Background(), tx, brandID, req, inisiator{agenID: agentID}); !errors.Is(err, ErrRequestConflict) {
		t.Fatalf("changed request accepted: %v", err)
	}
}

func TestCheckoutRejectsChangedQuote(t *testing.T) {
	tx, scheduleID, brandID, agentID := setupAgenBooking(t)
	oldPrice := 1.0
	zero := 0.0
	req := BookingRequest{ExpectedTotal: &oldPrice, ExpectedDP: &zero, ScheduleID: scheduleID, PIC: PICInput{NamaLengkap: "Quote PIC", NoHP: nomorUji(981), JenisKelamin: "L", RoomType: "Quad"}}
	if _, err := processBookingTx(context.Background(), tx, brandID, req, inisiator{agenID: agentID}); !errors.Is(err, ErrQuoteChanged) {
		t.Fatalf("stale quote accepted: %v", err)
	}
}

func TestCheckoutSnapshotsDP(t *testing.T) {
	tx, scheduleID, brandID, agentID := setupAgenBooking(t)
	testdb.Exec(t, tx, `UPDATE schedules SET minimal_dp=1000000 WHERE id=?`, scheduleID)
	req := BookingRequest{TermsVersion: TermsVersion, ScheduleID: scheduleID, PIC: PICInput{NamaLengkap: "Snapshot PIC", NoHP: nomorUji(982), JenisKelamin: "L", RoomType: "Quad"}}
	response, err := processBookingTx(context.Background(), tx, brandID, req, inisiator{agenID: agentID})
	if err != nil {
		t.Fatal(err)
	}
	testdb.Exec(t, tx, `UPDATE schedules SET minimal_dp=2000000 WHERE id=?`, scheduleID)
	var snapshot float64
	if err = tx.QueryRow(`SELECT c.dp_per_pax FROM booking_checkout c JOIN bookings b ON b.id=c.booking_id WHERE b.id_booking=?`, response.Booking.BookingCode).Scan(&snapshot); err != nil {
		t.Fatal(err)
	}
	if snapshot != 1000000 {
		t.Fatalf("snapshot changed: %v", snapshot)
	}

}

func TestLateDepartureSnapshotsFullPayment(t *testing.T) {
	tx, scheduleID, brandID, agentID := setupAgenBooking(t)
	testdb.Exec(t, tx, `UPDATE schedules SET berangkat_tanggal=DATE_ADD(CURRENT_DATE,INTERVAL 30 DAY) WHERE id=?`, scheduleID)
	req := BookingRequest{TermsVersion: TermsVersion, ScheduleID: scheduleID, PIC: PICInput{NamaLengkap: "Late PIC", NoHP: nomorUji(983), JenisKelamin: "L", RoomType: "Quad"}}
	response, err := processBookingTx(context.Background(), tx, brandID, req, inisiator{agenID: agentID})
	if err != nil {
		t.Fatal(err)
	}
	if response.Booking.MinimalDP != response.Booking.TotalHarga {
		t.Fatal("late booking does not require full payment")
	}
	var full bool
	var due, expiry time.Time
	if err := tx.QueryRow(`SELECT c.full_payment,c.due_at,c.original_expires_at FROM booking_checkout c JOIN bookings b ON b.id=c.booking_id WHERE b.id_booking=?`, response.Booking.BookingCode).Scan(&full, &due, &expiry); err != nil {
		t.Fatal(err)
	}
	if !full || !due.Equal(expiry) {
		t.Fatal("full-payment deadline differs from hold")
	}
}

func TestCheckoutRequiresConsentAndQuote(t *testing.T) {
	total, dp := 20000000.0, 5000000.0
	valid := BookingRequest{TermsAccepted: true, TermsVersion: TermsVersion, RequestKey: strings.Repeat("a", 48), ExpectedTotal: &total, ExpectedDP: &dp, PIC: PICInput{NamaLengkap: "Test PIC", NoHP: "081234567890", JenisKelamin: "L", NIK: &nikUjiValid}}
	if reason := validateCheckout(&valid, false); reason != "" {
		t.Fatal(reason)
	}
	for _, mutate := range []func(*BookingRequest){
		func(r *BookingRequest) { r.TermsAccepted = false },
		func(r *BookingRequest) { r.TermsVersion = "old" },
		func(r *BookingRequest) { r.ExpectedTotal = nil },
		func(r *BookingRequest) { r.RequestKey = "short" },
	} {
		invalid := valid
		mutate(&invalid)
		if validateCheckout(&invalid, false) == "" {
			t.Fatal("invalid checkout accepted")
		}
	}
}

// Verifier tanpa secret dan tanpa devBypass (dibuat langsung, bukan lewat newTurnstileVerifier) menolak.
func TestCaptchaFailClosedWithoutSecret(t *testing.T) {
	verifier := &turnstileVerifier{}
	if !verifier.required() {
		t.Fatal("verifier tanpa secret tidak boleh dilewati")
	}
	if ok, err := verifier.verify(context.Background(), "", "127.0.0.1"); ok || err == nil {
		t.Fatalf("tanpa secret harus ditolak: ok=%v err=%v", ok, err)
	}
	dev := &turnstileVerifier{devBypass: true}
	if ok, err := dev.verify(context.Background(), "", "127.0.0.1"); !ok || err != nil {
		t.Fatalf("bypass pengembangan: ok=%v err=%v", ok, err)
	}
}
