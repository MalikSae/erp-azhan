package selfbooking

import (
	"context"
	"testing"
	"time"

	"erp-azhan/api/internal/shared"
	"erp-azhan/api/internal/testdb"
)

// Booking yang dibuat tepat di batas pendaftaran online (berangkat 30 hari lagi) tetap boleh DP dan bayar bertahap;
// tenggat lunasnya jatuh hari ini (hari itu masih termasuk), bukan 24 jam hold seperti aturan lama "wajib lunas".
func TestBookingMendesakTetapBolehDPDanTenggatHariIni(t *testing.T) {
	tx, scheduleID, brandID, agentID := setupAgenBooking(t)
	testdb.Exec(t, tx, `UPDATE schedules SET berangkat_tanggal=DATE_ADD(CURRENT_DATE,INTERVAL 30 DAY) WHERE id=?`, scheduleID)
	req := BookingRequest{TermsVersion: TermsVersion, ScheduleID: scheduleID, PIC: PICInput{NamaLengkap: "Mendesak PIC", NoHP: nomorUji(984), JenisKelamin: "L", RoomType: "Quad"}}
	response, err := processBookingTx(context.Background(), tx, brandID, req, inisiatorAgen(agentID))
	if err != nil {
		t.Fatal(err)
	}
	if response.Booking.MinimalDP >= response.Booking.TotalHarga {
		t.Fatalf("booking mendesak dipaksa lunas: minimal DP %v dari total %v", response.Booking.MinimalDP, response.Booking.TotalHarga)
	}
	var full bool
	var due time.Time
	if err := tx.QueryRow(`SELECT c.full_payment, c.due_at FROM booking_checkout c JOIN bookings b ON b.id=c.booking_id WHERE b.id_booking=?`, response.Booking.BookingCode).Scan(&full, &due); err != nil {
		t.Fatal(err)
	}
	if full {
		t.Error("full_payment menyala untuk booking mendesak")
	}
	now := time.Now().In(shared.LoadConfig().Location())
	hariIni := now.Format("2006-01-02")
	if got := due.Format("2006-01-02"); got != hariIni {
		t.Errorf("tenggat lunas = %s, mau hari ini %s", got, hariIni)
	}
}
