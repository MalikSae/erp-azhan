package portal

import (
	"encoding/json"
	"erp-azhan/api/internal/booking"
	"erp-azhan/api/internal/jamaah"
	"erp-azhan/api/internal/testdb"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProfileNeverExposesInternalNotes(t *testing.T) {
	secret := "INTERNAL SYNTHETIC NOTE"
	passport := "SYNTHETIC-PASSPORT"
	raw, err := json.Marshal(profileView(&jamaah.Jamaah{ID: 1, Catatan: &secret, NoPaspor: &passport}))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), secret) || strings.Contains(string(raw), "catatan") {
		t.Fatal("internal note exposed")
	}
	if !strings.Contains(string(raw), `"no_paspor":"SYNTHETIC-PASSPORT"`) {
		t.Fatal("passport contract missing")
	}
}

func TestBookingViewUsesSnapshotAndPersonalPermissions(t *testing.T) {
	t.Setenv("APP_TIMEZONE", "Asia/Jakarta")
	db := testdb.Portal(t)
	for _, stmt := range []string{
		`CREATE TEMPORARY TABLE bookings(id BIGINT PRIMARY KEY,status VARCHAR(20),is_seat_blocked BOOLEAN,seat_hold_expires_at DATETIME)`,
		`CREATE TEMPORARY TABLE booking_checkout(booking_id BIGINT PRIMARY KEY,due_at DATETIME,dp_per_pax DECIMAL(15,2),full_payment BOOLEAN)`,
		`INSERT INTO bookings VALUES(1,'baru',TRUE,DATE_ADD(NOW(),INTERVAL 1 HOUR))`,
		`INSERT INTO booking_checkout VALUES(1,'2026-10-11 00:00:00',5000000,TRUE)`,
	} {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatal(err)
		}
	}
	owner := int64(1)
	b := &booking.Booking{ID: 1, PicJamaahID: &owner, Pax: []booking.BookingPax{{JamaahID: 1, ProgressVisa: true}, {JamaahID: 2, ProgressVisa: false}}}
	h := &Handler{db: db}
	req := httptest.NewRequest("GET", "/portal/bookings", nil)
	view, err := h.bookingView(req, b, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !view.CanSubmitPayment || !view.CanViewInvoice || !view.FullPayment || view.DPPerPax != 5000000 || view.DueAt == nil || *view.DueAt != "2026-10-10T17:00:00Z" {
		t.Fatalf("wrong snapshot/capabilities: %+v", view)
	}
	view, err = h.bookingView(req, b, 2)
	if err != nil {
		t.Fatal(err)
	}
	if view.CanSubmitPayment || view.CanViewInvoice || view.PersonalPax == nil || view.PersonalPax.JamaahID != 2 || view.PersonalPax.ProgressVisa {
		t.Fatalf("other pax inherited PIC state: %+v", view)
	}
	if _, err = db.Exec(`UPDATE bookings SET seat_hold_expires_at=DATE_SUB(NOW(),INTERVAL 1 SECOND) WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	view, err = h.bookingView(req, b, 1)
	if err != nil || view.ReservationPayable || view.CanSubmitPayment {
		t.Fatal("expired hold remains payable", err)
	}
	t.Log("snapshot due_at=2026-10-10T17:00:00Z (11 Oct 00:00 WIB); non-PIC payment=false invoice=false; expired hold payment=false; personal pax=2 visa=false")
}
