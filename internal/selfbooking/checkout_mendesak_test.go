package selfbooking

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"erp-azhan/api/internal/shared"
	"erp-azhan/api/internal/testdb"
)

// Booking yang dibuat tepat di batas pendaftaran online (berangkat 30 hari lagi) wajib lunas: tenggat lunasnya jatuh
// hari ini, sedangkan DP dibayar dalam 24 jam, jadi DP yang diverifikasi besok langsung dibatalkan otomatis
// (keputusan 2026-11-22, membalik aturan 2026-11-17 yang tetap mengizinkan DP).
func TestBookingTenggatHariIniWajibLunas(t *testing.T) {
	tx, scheduleID, brandID, agentID := setupAgenBooking(t)
	testdb.Exec(t, tx, `UPDATE schedules SET berangkat_tanggal=DATE_ADD(CURRENT_DATE,INTERVAL 30 DAY) WHERE id=?`, scheduleID)
	testdb.Exec(t, tx, `UPDATE brands SET batas_pelunasan_hari=30 WHERE id=?`, brandID)
	req := BookingRequest{TermsVersion: TermsVersion, ScheduleID: scheduleID, PemesanTidakIkut: true, PIC: PICInput{NamaLengkap: "Mendesak PIC", NoHP: nomorUji(984), JenisKelamin: "L", RoomType: "Quad"}}
	response, err := processBookingTx(context.Background(), tx, brandID, req, inisiatorAgen(agentID))
	if err != nil {
		t.Fatal(err)
	}
	if response.Booking.MinimalDP != response.Booking.TotalHarga {
		t.Fatalf("minimal bayar %v, mau total %v (wajib lunas)", response.Booking.MinimalDP, response.Booking.TotalHarga)
	}
	full, due := checkoutBooking(t, tx, response.Booking.BookingCode)
	if !full {
		t.Error("full_payment mati untuk booking yang tenggatnya hari ini")
	}
	hariIni := time.Now().In(shared.LoadConfig().Location()).Format("2006-01-02")
	if got := due.Format("2006-01-02"); got != hariIni {
		t.Errorf("tenggat lunas = %s, mau hari ini %s", got, hariIni)
	}
}

// Sehari sebelumnya (berangkat 31 hari lagi) tenggatnya besok: DP tetap boleh.
func TestBookingTenggatBesokTetapBolehDP(t *testing.T) {
	tx, scheduleID, brandID, agentID := setupAgenBooking(t)
	testdb.Exec(t, tx, `UPDATE schedules SET berangkat_tanggal=DATE_ADD(CURRENT_DATE,INTERVAL 31 DAY) WHERE id=?`, scheduleID)
	testdb.Exec(t, tx, `UPDATE brands SET batas_pelunasan_hari=30 WHERE id=?`, brandID)
	req := BookingRequest{TermsVersion: TermsVersion, ScheduleID: scheduleID, PemesanTidakIkut: true, PIC: PICInput{NamaLengkap: "Besok PIC", NoHP: nomorUji(985), JenisKelamin: "L", RoomType: "Quad"}}
	response, err := processBookingTx(context.Background(), tx, brandID, req, inisiatorAgen(agentID))
	if err != nil {
		t.Fatal(err)
	}
	if response.Booking.MinimalDP >= response.Booking.TotalHarga {
		t.Fatalf("booking H-31 dipaksa lunas: minimal %v dari total %v", response.Booking.MinimalDP, response.Booking.TotalHarga)
	}
	if full, _ := checkoutBooking(t, tx, response.Booking.BookingCode); full {
		t.Error("full_payment menyala untuk booking yang tenggatnya besok")
	}
}

func checkoutBooking(t *testing.T, tx *sql.Tx, kode string) (bool, time.Time) {
	t.Helper()
	var full bool
	var due time.Time
	if err := tx.QueryRow(`SELECT c.full_payment, c.due_at FROM booking_checkout c JOIN bookings b ON b.id=c.booking_id WHERE b.id_booking=?`, kode).Scan(&full, &due); err != nil {
		t.Fatal(err)
	}
	return full, due
}
