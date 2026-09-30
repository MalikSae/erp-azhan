package selfbooking

import (
	"context"
	"erp-azhan/api/internal/testdb"
	"errors"
	"testing"
	"time"
)

// Regression tests converted from the self-booking audit reproductions.
// All database changes use the existing rollback fixture.
func TestAuditSelfBookingDuplicateMember(t *testing.T) {
	tx, scheduleID, brandID, agentID := setupAgenBooking(t)
	member := testdb.NewJamaah(t, tx, brandID, testdb.JamaahOpts{Nama: "Audit shared member", Direkrut: agentID})
	for i := 0; i < 2; i++ {
		req := BookingRequest{ScheduleID: scheduleID, PIC: PICInput{NamaLengkap: "Audit PIC", NoHP: nomorUji(800 + i), JenisKelamin: "L", RoomType: "Quad"}, Anggota: []AnggotaInput{{JamaahID: &member, NamaLengkap: "Audit shared member", JenisKelamin: "L", PaxType: "reguler", RoomType: quad()}}}
		_, err := processBookingTx(context.Background(), tx, brandID, req, inisiator{agenID: agentID})
		if i == 0 && err != nil {
			t.Fatal(err)
		}
		if i == 1 && !errors.Is(err, ErrDuplicate) {
			t.Fatalf("duplicate member: %v", err)
		}
	}
	var count int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM booking_pax bp JOIN bookings b ON b.id=bp.booking_id WHERE bp.jamaah_id=? AND b.schedule_id=? AND b.status='baru'`, member, scheduleID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("probe changed: count=%d", count)
	}
}

func TestAuditSelfBookingAlphabeticPIN(t *testing.T) {
	tx, scheduleID, brandID, _ := setupAgenBooking(t)
	req := BookingRequest{ScheduleID: scheduleID, PIC: PICInput{NamaLengkap: "Audit PIN", NoHP: nomorUji(900), JenisKelamin: "L", RoomType: "Quad", PortalPIN: "abcdef"}}
	if _, err := processBookingTx(context.Background(), tx, brandID, req, inisiator{}); !errors.Is(err, ErrPinRequired) {
		t.Fatalf("alphabetic PIN: %v", err)
	}
}

func TestAuditSelfBookingExistingAdultAsInfant(t *testing.T) {
	tx, scheduleID, brandID, agentID := setupAgenBooking(t)
	departure := time.Now().AddDate(0, 2, 0).Format("2006-01-02")
	testdb.Exec(t, tx, `UPDATE schedules SET berangkat_tanggal=?,harga_infant=0 WHERE id=?`, departure, scheduleID)
	member := testdb.NewJamaah(t, tx, brandID, testdb.JamaahOpts{Nama: "Audit adult", Direkrut: agentID})
	testdb.Exec(t, tx, `UPDATE jamaah SET tanggal_lahir='1990-01-01' WHERE id=?`, member)
	suppliedDOB := time.Now().AddDate(0, -6, 0).Format("2006-01-02")
	req := BookingRequest{ScheduleID: scheduleID, PIC: PICInput{NamaLengkap: "Audit infant PIC", NoHP: nomorUji(950), JenisKelamin: "L", RoomType: "Quad"}, Anggota: []AnggotaInput{{JamaahID: &member, NamaLengkap: "Audit adult", JenisKelamin: "L", PaxType: "infant", TanggalLahir: &suppliedDOB}}}
	if _, err := processBookingTx(context.Background(), tx, brandID, req, inisiator{agenID: agentID}); !errors.Is(err, ErrUsiaInfant) {
		t.Fatalf("adult as infant: %v", err)
	}
}
