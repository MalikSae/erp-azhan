package selfbooking

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	"erp-azhan/api/internal/testdb"
)

func quad() *string { s := "Quad"; return &s }

// nomorUji menghasilkan nomor HP yang tidak mungkin bentrok dengan data dev.
func nomorUji(n int) string {
	return fmt.Sprintf("0819%08d", (time.Now().UnixNano()/1000+int64(n))%100000000)
}

func setupAgenBooking(t *testing.T) (*sql.Tx, int64, int64, int64) {
	t.Helper()
	_, tx := testdb.Tx(t)
	sched, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	testdb.Exec(t, tx, `UPDATE schedules SET status='published' WHERE id=?`, sched)
	agenID := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "AgenJalur1", StatusAgen: "aktif"})
	return tx, sched, brand, agenID
}

func kaitan(t *testing.T, tx *sql.Tx, id int64) (string, int64, bool) {
	t.Helper()
	var st string
	var by sql.NullInt64
	var pin sql.NullString
	if err := tx.QueryRow(`SELECT kaitan_status, direkrut_oleh_jamaah_id, portal_pin_hash FROM jamaah WHERE id=?`, id).Scan(&st, &by, &pin); err != nil {
		t.Fatalf("baca jamaah %d: %v", id, err)
	}
	return st, by.Int64, pin.Valid && pin.String != ""
}

// Langkah 2 (Jalur 1): agen membuat booking; jamaah baru terikat ke agen,
// jamaah miliknya dipakai ulang (repeat order), agen bukan pax/PIC.
func TestBookingAgenJalur1(t *testing.T) {
	tx, sched, brand, agenID := setupAgenBooking(t)
	ctx := context.Background()
	milik := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "MilikAgen", Direkrut: agenID})

	req := BookingRequest{
		ScheduleID:   sched,
		KodeReferral: "HARUS-DIABAIKAN",
		PIC:          PICInput{NamaLengkap: "PIC Baru Jalur1", NoHP: nomorUji(1), JenisKelamin: "L", RoomType: "Quad"},
		Anggota: []AnggotaInput{
			{PaxType: "reguler", NamaLengkap: "Anggota Tanpa HP", JenisKelamin: "P", RoomType: quad()},
			{PaxType: "reguler", JamaahID: &milik, RoomType: quad(), NamaLengkap: "MilikAgen", JenisKelamin: "L"},
		},
	}
	resp, err := processBookingTx(ctx, tx, brand, req, inisiator{agenID: agenID})
	if err != nil {
		t.Fatalf("booking agen: %v", err)
	}
	if resp.PortalToken != "" {
		t.Fatal("booking agen tidak boleh menerbitkan token portal")
	}

	var bookingID, pic int64
	var status string
	if err := tx.QueryRow(`SELECT id, pic_jamaah_id, status FROM bookings WHERE id_booking=?`, resp.Booking.BookingCode).Scan(&bookingID, &pic, &status); err != nil {
		t.Fatalf("baca booking: %v", err)
	}
	if status != "baru" || pic == agenID {
		t.Fatalf("booking status=%s pic=%d (agen %d)", status, pic, agenID)
	}
	rows, err := tx.Query(`SELECT jamaah_id FROM booking_pax WHERE booking_id=?`, bookingID)
	if err != nil {
		t.Fatal(err)
	}
	var pax []int64
	for rows.Next() {
		var id int64
		rows.Scan(&id)
		pax = append(pax, id)
	}
	rows.Close()
	if len(pax) != 3 {
		t.Fatalf("pax = %v, want 3", pax)
	}
	for _, id := range pax {
		if id == agenID {
			t.Fatal("agen ikut menjadi pax")
		}
		st, by, pin := kaitan(t, tx, id)
		if st != "terikat_agen" || by != agenID {
			t.Fatalf("pax %d kaitan %s/%d, want terikat_agen/%d", id, st, by, agenID)
		}
		if pin {
			t.Fatalf("pax %d punya PIN; jamaah baru agen harus aktivasi sendiri", id)
		}
	}
}

// D8: nomor milik jamaah lain (termasuk organik belum_ditentukan) ditolak
// dengan pesan umum, dan kaitannya tidak berubah.
func TestBookingAgenTolakNomorMilikLain(t *testing.T) {
	tx, sched, brand, agenID := setupAgenBooking(t)
	ctx := context.Background()
	organik := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Organik"})
	hp := nomorUji(2)
	testdb.Exec(t, tx, `UPDATE jamaah SET no_hp=? WHERE id=?`, hp, organik)

	req := BookingRequest{ScheduleID: sched, PIC: PICInput{NamaLengkap: "Siapa Saja", NoHP: hp, JenisKelamin: "L", RoomType: "Quad"}}
	if _, err := processBookingTx(ctx, tx, brand, req, inisiator{agenID: agenID}); !errors.Is(err, ErrNomorMilikLain) {
		t.Fatalf("err = %v, want ErrNomorMilikLain", err)
	}
	if st, _, _ := kaitan(t, tx, organik); st != "belum_ditentukan" {
		t.Fatalf("jamaah organik berubah menjadi %s", st)
	}
}

func TestBookingAgenTolakJamaahAgenLain(t *testing.T) {
	tx, sched, brand, agenID := setupAgenBooking(t)
	ctx := context.Background()
	lain := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "AgenLain", StatusAgen: "aktif"})
	milikLain := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "MilikLain", Direkrut: lain})

	req := BookingRequest{ScheduleID: sched, PIC: PICInput{JamaahID: &milikLain, NamaLengkap: "MilikLain", RoomType: "Quad"}}
	if _, err := processBookingTx(ctx, tx, brand, req, inisiator{agenID: agenID}); !errors.Is(err, ErrJamaahBukanMilik) {
		t.Fatalf("err = %v, want ErrJamaahBukanMilik", err)
	}
}

func TestBookingAgenNonaktifDitolak(t *testing.T) {
	tx, sched, brand, agenID := setupAgenBooking(t)
	testdb.Exec(t, tx, `UPDATE jamaah SET status_agen='nonaktif' WHERE id=?`, agenID)
	req := BookingRequest{ScheduleID: sched, PIC: PICInput{NamaLengkap: "X", NoHP: nomorUji(3), JenisKelamin: "L", RoomType: "Quad"}}
	if _, err := processBookingTx(context.Background(), tx, brand, req, inisiator{agenID: agenID}); !errors.Is(err, ErrBukanAgenAktif) {
		t.Fatalf("err = %v, want ErrBukanAgenAktif", err)
	}
}

// Regresi refactor inisiator: booking publik tetap membuat PIN PIC dan
// mengikat pax lewat kode referral (Jalur 2).
func TestBookingPublikReferralSetelahRefactor(t *testing.T) {
	tx, sched, brand, agenID := setupAgenBooking(t)
	testdb.Exec(t, tx, `UPDATE jamaah SET kode_referral='UJIJL2' WHERE id=?`, agenID)
	req := BookingRequest{
		ScheduleID:   sched,
		KodeReferral: "ujijl2",
		PIC:          PICInput{NamaLengkap: "PIC Publik", NoHP: nomorUji(4), JenisKelamin: "P", RoomType: "Quad", PortalPIN: "135790"},
	}
	resp, err := processBookingTx(context.Background(), tx, brand, req, inisiator{})
	if err != nil {
		t.Fatalf("booking publik: %v", err)
	}
	if resp.PortalToken == "" {
		t.Fatal("booking publik PIC baru harus menerbitkan token portal")
	}
	st, by, pin := kaitan(t, tx, resp.Jamaah.ID)
	if st != "terikat_agen" || by != agenID || !pin {
		t.Fatalf("PIC publik = %s/%d pin=%v, want terikat_agen/%d dengan PIN", st, by, pin, agenID)
	}
}

// Tanggal lahir anggota (infant) tersimpan dan usia infant divalidasi
// terhadap tanggal berangkat.
func TestBookingPublikInfantTanggalLahir(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	hariIni := time.Now()
	berangkat := hariIni.AddDate(0, 2, 0).Format("2006-01-02")
	sched, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: berangkat})
	testdb.Exec(t, tx, `UPDATE schedules SET status='published' WHERE id=?`, sched)

	booking := func(tanggalLahir string, n int) (*BookingResponse, error) {
		tl := tanggalLahir
		req := BookingRequest{
			ScheduleID: sched,
			PIC:        PICInput{NamaLengkap: "PIC Infant", NoHP: nomorUji(10 + n), JenisKelamin: "P", RoomType: "Quad", PortalPIN: "135790"},
			Anggota:    []AnggotaInput{{PaxType: "infant", NamaLengkap: "Bayi Uji", JenisKelamin: "L", TanggalLahir: &tl}},
		}
		return processBookingTx(ctx, tx, brand, req, inisiator{})
	}

	lahir := hariIni.AddDate(0, -8, 0).Format("2006-01-02")
	resp, err := booking(lahir, 1)
	if err != nil {
		t.Fatalf("booking infant: %v", err)
	}
	var tersimpan sql.NullString
	if err := tx.QueryRow(`
		SELECT DATE_FORMAT(j.tanggal_lahir, '%Y-%m-%d') FROM booking_pax bp
		JOIN bookings b ON b.id = bp.booking_id JOIN jamaah j ON j.id = bp.jamaah_id
		WHERE b.id_booking = ? AND bp.pax_type = 'infant'`, resp.Booking.BookingCode).Scan(&tersimpan); err != nil {
		t.Fatalf("baca infant: %v", err)
	}
	if tersimpan.String != lahir {
		t.Fatalf("tanggal lahir infant tersimpan %q, want %q", tersimpan.String, lahir)
	}

	if _, err := booking(hariIni.AddDate(-3, 0, 0).Format("2006-01-02"), 2); !errors.Is(err, ErrUsiaInfant) {
		t.Fatalf("infant 3 tahun: %v", err)
	}
	if _, err := booking(hariIni.AddDate(0, 0, 5).Format("2006-01-02"), 3); !errors.Is(err, ErrTanggalLahirTidakValid) {
		t.Fatalf("tanggal lahir masa depan: %v", err)
	}
	if _, err := booking("12-01-2026", 4); !errors.Is(err, ErrTanggalLahirTidakValid) {
		t.Fatalf("format salah: %v", err)
	}
}
