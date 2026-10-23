package selfbooking

import (
	"context"
	"database/sql"
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

// Booking publik wajib punya akun: tanpa akun (dan bukan agen) ditolak, dan akun brand lain tidak sah.
func TestAuditSelfBookingTanpaAkun(t *testing.T) {
	tx, scheduleID, brandID, _ := setupAgenBooking(t)
	req := BookingRequest{ScheduleID: scheduleID, PIC: PICInput{NamaLengkap: "Audit Tanpa Akun", JenisKelamin: "L", RoomType: "Quad"}}
	if _, err := processBookingTx(context.Background(), tx, brandID, req, inisiator{}); !errors.Is(err, ErrAkunTidakValid) {
		t.Fatalf("tanpa akun: %v", err)
	}
	var brandLain int64
	if err := tx.QueryRow(`SELECT id FROM brands WHERE id<>? ORDER BY id LIMIT 1`, brandID).Scan(&brandLain); err != nil {
		t.Skip("butuh minimal dua brand untuk menguji akun lintas brand")
	}
	akunLain := akunUji(t, tx, brandLain, "Akun Brand Lain", nomorUji(901))
	if _, err := processBookingTx(context.Background(), tx, brandID, req, inisiator{accountID: akunLain}); !errors.Is(err, ErrAkunTidakValid) {
		t.Fatalf("akun brand lain: %v", err)
	}
}

// K1 (tinjauan keamanan 2026-10-22): nomor HP akun belum terverifikasi, jadi akun TIDAK boleh
// ditautkan ke jamaah lama hanya karena HP (dan nama) cocok. Setiap akun tanpa jamaah mendapat
// jamaah PIC baru; jawaban sama untuk nama cocok maupun beda (tidak ada oracle), dan data jamaah
// lama (termasuk NIK) tidak tersentuh.
func TestAuditSelfBookingNomorMilikOrangLain(t *testing.T) {
	tx, scheduleID, brandID, _ := setupAgenBooking(t)
	ctx := context.Background()
	nomor := nomorUji(902)
	nikLama := "3174010101015555"
	testdb.Exec(t, tx, `INSERT INTO jamaah (brand_id, id_jamaah, kode_jamaah, nama_lengkap, no_hp, nik) VALUES (?,?,?,?,?,?)`,
		brandID, "TST-NOMOR-902", "TN902", "Pemilik Asli", nomor, nikLama)
	var lamaID int64
	if err := tx.QueryRow(`SELECT id FROM jamaah WHERE id_jamaah='TST-NOMOR-902'`).Scan(&lamaID); err != nil {
		t.Fatal(err)
	}

	for i, nama := range []string{"Penyusup", "  pemilik   ASLI "} {
		akun := akunUji(t, tx, brandID, nama, nomor)
		nik := []string{"3174010101014444", "3174010101013333"}[i]
		req := BookingRequest{ScheduleID: scheduleID, PIC: PICInput{NamaLengkap: nama, JenisKelamin: "L", RoomType: "Quad", NIK: &nik}}
		resp, err := processBookingTx(ctx, tx, brandID, req, inisiator{accountID: akun})
		if err != nil {
			t.Fatalf("akun %q: booking harus berhasil tanpa membuka data jamaah lama: %v", nama, err)
		}
		if resp.Jamaah.ID == lamaID {
			t.Fatalf("akun %q tertaut ke jamaah lama %d lewat nomor HP", nama, lamaID)
		}
		var linked int64
		var jamaahNama string
		var hpPIC sql.NullString
		if err = tx.QueryRow(`SELECT a.jamaah_id, j.nama_lengkap, j.no_hp FROM customer_accounts a JOIN jamaah j ON j.id=a.jamaah_id WHERE a.id=?`, akun).Scan(&linked, &jamaahNama, &hpPIC); err != nil {
			t.Fatal(err)
		}
		if linked != resp.Jamaah.ID || normalizeName(jamaahNama) != normalizeName(nama) {
			t.Fatalf("akun %q tertaut ke jamaah %d (%s), want jamaah PIC baru %d", nama, linked, jamaahNama, resp.Jamaah.ID)
		}
		// Satu nomor = satu jamaah per brand (migrasi 060): PIC baru dibuat tanpa nomor yang bentrok.
		if hpPIC.Valid {
			t.Fatalf("PIC baru memakai nomor jamaah lama: %q", hpPIC.String)
		}
	}

	var namaLama, nikTersimpan string
	if err := tx.QueryRow(`SELECT nama_lengkap, nik FROM jamaah WHERE id=?`, lamaID).Scan(&namaLama, &nikTersimpan); err != nil {
		t.Fatal(err)
	}
	if namaLama != "Pemilik Asli" || nikTersimpan != nikLama {
		t.Fatalf("jamaah lama berubah: %q %q", namaLama, nikTersimpan)
	}
}

// NIK PIC yang sudah dimiliki jamaah lain tidak menolak booking (tidak menjadi oracle) dan tidak
// menautkan akun ke jamaah itu; NIK tidak dipasang ke jamaah PIC baru (admin menggabungkan manual).
func TestBookingAkunNIKPICMilikJamaahLain(t *testing.T) {
	tx, scheduleID, brandID, _ := setupAgenBooking(t)
	nik := "3174010101016666"
	testdb.Exec(t, tx, `INSERT INTO jamaah (brand_id, id_jamaah, kode_jamaah, nama_lengkap, nik) VALUES (?,?,?,?,?)`,
		brandID, "TST-NIK-905", "TK905", "Pemilik NIK", nik)
	akun := akunUji(t, tx, brandID, "Orang Lain", nomorUji(905))
	req := BookingRequest{ScheduleID: scheduleID, PIC: PICInput{NamaLengkap: "Orang Lain", JenisKelamin: "L", RoomType: "Quad", NIK: &nik}}
	resp, err := processBookingTx(context.Background(), tx, brandID, req, inisiator{accountID: akun})
	if err != nil {
		t.Fatalf("NIK bentrok harus tetap diproses tanpa membuka data: %v", err)
	}
	var nikPIC sql.NullString
	if err = tx.QueryRow(`SELECT nik FROM jamaah WHERE id=?`, resp.Jamaah.ID).Scan(&nikPIC); err != nil {
		t.Fatal(err)
	}
	if nikPIC.Valid && nikPIC.String != "" {
		t.Fatalf("NIK milik jamaah lain terpasang ke PIC baru: %q", nikPIC.String)
	}
}

// Anggota rombongan tersimpan ke akun tanpa duplikat.
func TestBookingSimpanJamaahTersimpan(t *testing.T) {
	tx, scheduleID, brandID, _ := setupAgenBooking(t)
	akun := akunUji(t, tx, brandID, "Pemesan Simpan", nomorUji(903))
	req := BookingRequest{ScheduleID: scheduleID, PIC: PICInput{NamaLengkap: "Pemesan Simpan", JenisKelamin: "L", RoomType: "Quad"},
		Anggota: []AnggotaInput{{PaxType: "reguler", NamaLengkap: "Anggota Satu", JenisKelamin: "P", RoomType: quad()}}}
	if _, err := processBookingTx(context.Background(), tx, brandID, req, inisiator{accountID: akun}); err != nil {
		t.Fatal(err)
	}
	if err := simpanTravelers(context.Background(), tx, akun, req.Anggota); err != nil { // panggilan ulang: tidak boleh menggandakan
		t.Fatal(err)
	}
	var n int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM customer_travelers WHERE account_id=?`, akun).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("jamaah tersimpan = %d, want 1", n)
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

// Pemesan yang tidak berangkat: jamaah pertama menjadi PIC booking, akun pemesan tidak
// dibuatkan jamaah, dan booking tetap milik akun pemesan.
func TestBookingPemesanTidakIkut(t *testing.T) {
	tx, scheduleID, brandID, _ := setupAgenBooking(t)
	akun := akunUji(t, tx, brandID, "Anak Pemesan", nomorUji(904))
	req := BookingRequest{ScheduleID: scheduleID, PemesanTidakIkut: true,
		PIC:     PICInput{NamaLengkap: "Ibu Berangkat", JenisKelamin: "P", RoomType: "Quad"},
		Anggota: []AnggotaInput{{PaxType: "reguler", NamaLengkap: "Bapak Berangkat", JenisKelamin: "L", RoomType: quad()}}}
	resp, err := processBookingTx(context.Background(), tx, brandID, req, inisiator{accountID: akun})
	if err != nil {
		t.Fatal(err)
	}
	var picNama string
	var owner, akunJamaah sql.NullInt64
	if err = tx.QueryRow(`SELECT j.nama_lengkap, b.customer_account_id FROM bookings b JOIN jamaah j ON j.id=b.pic_jamaah_id WHERE b.id=?`, resp.Booking.BookingID).Scan(&picNama, &owner); err != nil {
		t.Fatal(err)
	}
	if picNama != "Ibu Berangkat" || owner.Int64 != akun {
		t.Fatalf("PIC=%q owner=%d, want Ibu Berangkat / akun %d", picNama, owner.Int64, akun)
	}
	if err = tx.QueryRow(`SELECT jamaah_id FROM customer_accounts WHERE id=?`, akun).Scan(&akunJamaah); err != nil {
		t.Fatal(err)
	}
	if akunJamaah.Valid {
		t.Fatalf("akun pemesan tidak boleh dibuatkan jamaah, dapat %d", akunJamaah.Int64)
	}
	var n int
	if err = tx.QueryRow(`SELECT COUNT(*) FROM booking_pax WHERE booking_id=?`, resp.Booking.BookingID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("pax = %d, want 2", n)
	}
}
