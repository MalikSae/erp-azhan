package selfbooking

import (
	"context"
	"errors"
	"testing"
	"time"

	"erp-azhan/api/internal/testdb"
)

func TestPublicBookingRejectsUnpricedInfant(t *testing.T) {
	_, tx := testdb.Tx(t)
	today := time.Now()
	id, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: today.AddDate(0, 2, 0).Format("2006-01-02")})
	testdb.Exec(t, tx, `UPDATE schedules SET status='published', harga_infant=NULL WHERE id=?`, id)
	dob := today.AddDate(0, -6, 0).Format("2006-01-02")
	req := BookingRequest{ScheduleID: id, PIC: PICInput{NamaLengkap: "PIC Test", NoHP: "6281299998812", JenisKelamin: "P", RoomType: "Quad", PortalPIN: "135790"}, Anggota: []AnggotaInput{{PaxType: "infant", NamaLengkap: "Bayi Test", JenisKelamin: "L", TanggalLahir: &dob}}}
	_, err := processBookingTx(context.Background(), tx, brand, req, inisiator{})
	if !errors.Is(err, ErrInfantUnavailable) {
		t.Fatalf("unpriced infant: %v", err)
	}
}
