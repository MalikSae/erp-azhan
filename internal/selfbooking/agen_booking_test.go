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

var nikUjiValid = "3174010101010001"

func nikp(s string) *string { return &s }

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
	var hasAkun bool
	if err := tx.QueryRow(`SELECT kaitan_status, direkrut_oleh_jamaah_id, EXISTS(SELECT 1 FROM customer_accounts WHERE jamaah_id=jamaah.id) FROM jamaah WHERE id=?`, id).Scan(&st, &by, &hasAkun); err != nil {
		t.Fatalf("baca jamaah %d: %v", id, err)
	}
	return st, by.Int64, hasAkun
}

// akunUji membuat akun pelanggan di transaksi uji. nomor dipakai sebagai no_hp akun,
// yang menjadi nomor PIC saat booking.
func akunUji(t *testing.T, tx *sql.Tx, brandID int64, nama, nomor string) int64 {
	t.Helper()
	email := fmt.Sprintf("uji-%d@example.com", time.Now().UnixNano())
	res, err := tx.Exec(`INSERT INTO customer_accounts (brand_id, nama_lengkap, email, email_normal, no_hp, password_hash) VALUES (?,?,?,?,?,'x')`,
		brandID, nama, email, email, nomor)
	if err != nil {
		t.Fatalf("buat akun uji: %v", err)
	}
	id, _ := res.LastInsertId()
	return id
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
		st, by, akun := kaitan(t, tx, id)
		if st != "terikat_agen" || by != agenID {
			t.Fatalf("pax %d kaitan %s/%d, want terikat_agen/%d", id, st, by, agenID)
		}
		if akun {
			t.Fatalf("pax %d punya akun; jamaah baru agen harus mendaftar sendiri", id)
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

// Booking publik atas nama akun pelanggan: PIC dibuat dari akun, terikat ke agen lewat
// kode referral (Jalur 2), dan booking tercatat milik akun itu.
func TestBookingPublikReferralDenganAkun(t *testing.T) {
	tx, sched, brand, agenID := setupAgenBooking(t)
	testdb.Exec(t, tx, `UPDATE jamaah SET kode_referral='UJIJL2' WHERE id=?`, agenID)
	akun := akunUji(t, tx, brand, "PIC Publik", nomorUji(4))
	req := BookingRequest{
		ScheduleID:   sched,
		KodeReferral: "ujijl2",
		PIC:          PICInput{NamaLengkap: "PIC Publik", JenisKelamin: "P", RoomType: "Quad"},
	}
	resp, err := processBookingTx(context.Background(), tx, brand, req, inisiator{accountID: akun})
	if err != nil {
		t.Fatalf("booking publik: %v", err)
	}
	st, by, punyaAkun := kaitan(t, tx, resp.Jamaah.ID)
	if st != "terikat_agen" || by != agenID || !punyaAkun {
		t.Fatalf("PIC publik = %s/%d akun=%v, want terikat_agen/%d dengan akun", st, by, punyaAkun, agenID)
	}
	var owner sql.NullInt64
	if err := tx.QueryRow(`SELECT customer_account_id FROM bookings WHERE id_booking=?`, resp.Booking.BookingCode).Scan(&owner); err != nil {
		t.Fatal(err)
	}
	if !owner.Valid || owner.Int64 != akun {
		t.Fatalf("pemilik booking = %v, want akun %d", owner, akun)
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
	testdb.Exec(t, tx, `UPDATE schedules SET status='published', harga_infant=0 WHERE id=?`, sched)

	booking := func(tanggalLahir string, n int) (*BookingResponse, error) {
		tl := tanggalLahir
		akun := akunUji(t, tx, brand, "PIC Infant", nomorUji(10+n))
		req := BookingRequest{
			ScheduleID: sched,
			PIC:        PICInput{NamaLengkap: "PIC Infant", JenisKelamin: "P", RoomType: "Quad"},
			Anggota:    []AnggotaInput{{PaxType: "infant", NamaLengkap: "Bayi Uji", JenisKelamin: "L", TanggalLahir: &tl}},
		}
		return processBookingTx(ctx, tx, brand, req, inisiator{accountID: akun})
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
