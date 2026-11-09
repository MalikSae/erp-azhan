package selfbooking

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"erp-azhan/api/internal/komisi"
	"erp-azhan/api/internal/testdb"
)

func quad() *string { s := "Quad"; return &s }

var nikUjiValid = "3174010101010001"

func nikp(s string) *string { return &s }

var seqNIK int64

// nikUji menghasilkan NIK 16 digit yang tidak mungkin bentrok dengan data dev.
func nikUji() string {
	seqNIK++
	return fmt.Sprintf("99%014d", (time.Now().UnixNano()/1000+seqNIK)%100000000000000)
}

// nomorUji menghasilkan nomor HP yang tidak mungkin bentrok dengan data dev.
func nomorUji(n int) string {
	return fmt.Sprintf("0819%08d", (time.Now().UnixNano()/1000+int64(n))%100000000)
}

func setupAgenBooking(t *testing.T) (*sql.Tx, int64, int64, int64) {
	t.Helper()
	_, tx := testdb.Tx(t)
	sched, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	testdb.Exec(t, tx, `UPDATE schedules SET status='published' WHERE id=?`, sched)
	agenID := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenPortal", StatusAgen: "aktif"})
	return tx, sched, brand, agenID
}

// kaitan mengembalikan status, pemilik kunci (akun agen), dan sumber kaitan sebuah jamaah.
func kaitan(t *testing.T, tx *sql.Tx, id int64) (string, int64, string) {
	t.Helper()
	var st string
	var by sql.NullInt64
	var sumber sql.NullString
	if err := tx.QueryRow(`SELECT kaitan_status, direkrut_oleh_akun_id, kaitan_sumber FROM jamaah WHERE id=?`, id).Scan(&st, &by, &sumber); err != nil {
		t.Fatalf("baca jamaah %d: %v", id, err)
	}
	return st, by.Int64, sumber.String
}

func agenReq(sched int64, pic PICInput, anggota ...AnggotaInput) BookingRequest {
	return BookingRequest{ScheduleID: sched, PIC: pic, Anggota: anggota}
}

func inisiatorAgen(agenID int64) inisiator { return inisiator{accountID: agenID, agenID: agenID} }

// Portal agen: jamaah baru terikat ke agen pembuat booking, jamaah miliknya dipakai ulang
// (booking ulang), agen bukan pax/PIC. Pemesan dan agen pembuat sama-sama akun agen.
func TestBookingAgenPortal(t *testing.T) {
	tx, sched, brand, agenID := setupAgenBooking(t)
	ctx := context.Background()
	nikMilik := nikUji()
	milik := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "MilikAgen", Direkrut: agenID, NIK: nikMilik})

	req := agenReq(sched,
		PICInput{NamaLengkap: "PIC Baru Portal", JenisKelamin: "L", RoomType: "Quad", NIK: nikp(nikUji())},
		AnggotaInput{PaxType: "reguler", NamaLengkap: "Anggota Tanpa HP", JenisKelamin: "P", RoomType: quad(), NIK: nikp(nikUji())},
		AnggotaInput{PaxType: "reguler", JamaahID: &milik, RoomType: quad(), NamaLengkap: "MilikAgen (uji)", JenisKelamin: "L", NIK: nikp(nikMilik)},
	)
	resp, err := processBookingTx(ctx, tx, brand, req, inisiatorAgen(agenID))
	if err != nil {
		t.Fatalf("booking agen: %v", err)
	}
	if len(resp.Peringatan) != 0 {
		t.Fatalf("peringatan = %v, want kosong", resp.Peringatan)
	}
	var bookingID, pic int64
	var status string
	var pemesan, pembuat sql.NullInt64
	if err := tx.QueryRow(`SELECT id, pic_jamaah_id, status, customer_account_id, dibuat_oleh_agen_akun_id FROM bookings WHERE id_booking=?`, resp.Booking.BookingCode).
		Scan(&bookingID, &pic, &status, &pemesan, &pembuat); err != nil {
		t.Fatalf("baca booking: %v", err)
	}
	if status != "baru" || pemesan.Int64 != agenID || pembuat.Int64 != agenID {
		t.Fatalf("booking status=%s pemesan=%v pembuat=%v, want baru/%d/%d", status, pemesan, pembuat, agenID, agenID)
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
		st, by, _ := kaitan(t, tx, id)
		if st != "terikat_agen" || by != agenID {
			t.Fatalf("pax %d kaitan %s/%d, want terikat_agen/%d", id, st, by, agenID)
		}
	}
	if pic == milik {
		t.Fatal("PIC seharusnya jamaah baru, bukan jamaah lama")
	}
}

// K16: jamaah yang sudah terkunci agen lain tidak lagi ditolak. Booking jalan dengan peringatan
// generik, kunci tidak berubah, booking tercatat milik agen pembuat, komisi mengalir ke pemilik kunci.
func TestBookingAgenJamaahTerkunciAgenLainJalanDenganPeringatan(t *testing.T) {
	tx, sched, brand, agenY := setupAgenBooking(t)
	ctx := context.Background()
	agenX := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenX", StatusAgen: "aktif"})
	nik := nikUji()
	terkunci := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Jamaah Terkunci", Direkrut: agenX, NIK: nik})
	testdb.Exec(t, tx, `UPDATE schedules SET nominal_komisi_langsung=1000000, nominal_bonus_pembinaan=300000 WHERE id=?`, sched)

	req := agenReq(sched, PICInput{NamaLengkap: "Jamaah Terkunci (uji)", JenisKelamin: "L", RoomType: "Quad", NIK: nikp(nik)})
	resp, err := processBookingTx(ctx, tx, brand, req, inisiatorAgen(agenY))
	if err != nil {
		t.Fatalf("booking agen: %v", err)
	}
	if len(resp.Peringatan) != 1 || !strings.Contains(resp.Peringatan[0], "sudah terikat agen lain") ||
		!strings.Contains(resp.Peringatan[0], "tidak menghasilkan komisi untuk Anda") {
		t.Fatalf("peringatan = %v", resp.Peringatan)
	}
	if strings.Contains(resp.Peringatan[0], "AgenX") || strings.Contains(resp.Peringatan[0], "Terkunci") {
		t.Fatalf("peringatan membuka data pemilik kunci atau jamaah: %q", resp.Peringatan[0])
	}
	if resp.Jamaah.ID != terkunci {
		t.Fatalf("jamaah = %d, want jamaah terkunci %d (NIK dan nama cocok)", resp.Jamaah.ID, terkunci)
	}
	if st, by, _ := kaitan(t, tx, terkunci); st != "terikat_agen" || by != agenX {
		t.Fatalf("kunci berubah: %s/%d, want terikat_agen/%d", st, by, agenX)
	}
	var pembuat sql.NullInt64
	if err := tx.QueryRow(`SELECT dibuat_oleh_agen_akun_id FROM bookings WHERE id_booking=?`, resp.Booking.BookingCode).Scan(&pembuat); err != nil || pembuat.Int64 != agenY {
		t.Fatalf("pembuat booking = %v (err %v), want %d", pembuat, err, agenY)
	}

	// Saat lunas, komisi ke pemilik kunci (X), bukan agen pembuat (Y).
	lunasDanLedger := func() map[int64]float64 {
		testdb.Exec(t, tx, `UPDATE bookings SET status='lunas' WHERE id=?`, resp.Booking.BookingID)
		if err := komisi.SnapshotNominal(ctx, tx, resp.Booking.BookingID); err != nil {
			t.Fatal(err)
		}
		if err := komisi.ProcessBookingLunas(ctx, tx, resp.Booking.BookingID); err != nil {
			t.Fatal(err)
		}
		rows, err := tx.Query(`SELECT akun_penerima_id, SUM(nominal) FROM transaksi_komisi WHERE booking_id=? GROUP BY akun_penerima_id`, resp.Booking.BookingID)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		out := map[int64]float64{}
		for rows.Next() {
			var id int64
			var n float64
			rows.Scan(&id, &n)
			out[id] = n
		}
		return out
	}
	got := lunasDanLedger()
	if got[agenX] != 1_000_000 || got[agenY] != 0 || len(got) != 1 {
		t.Fatalf("ledger = %v, want hanya X (%d) 1.000.000", got, agenX)
	}
}

// NIK yang sama dengan nama berbeda ditolak generik, tanpa membuka nama terdaftar.
func TestBookingAgenNIKNamaBedaDitolakGenerik(t *testing.T) {
	tx, sched, brand, agenID := setupAgenBooking(t)
	nik := nikUji()
	testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Nama Terdaftar", NIK: nik})
	req := agenReq(sched, PICInput{NamaLengkap: "Nama Lain Sekali", JenisKelamin: "L", RoomType: "Quad", NIK: nikp(nik)})
	_, err := processBookingTx(context.Background(), tx, brand, req, inisiatorAgen(agenID))
	if !errors.Is(err, ErrNIKAtasNamaLain) {
		t.Fatalf("err = %v, want ErrNIKAtasNamaLain", err)
	}
	if strings.Contains(strings.ToLower(err.Error()), "terdaftar") && strings.Contains(err.Error(), "Nama Terdaftar") {
		t.Fatalf("pesan membuka nama terdaftar: %q", err.Error())
	}
}

// jamaah_id (Jamaah Saya) hanya jalan pintas: harus cocok NIK dan nama, jadi id tebakan tidak membuka apa pun.
func TestBookingAgenJamaahIDHarusCocokNIKDanNama(t *testing.T) {
	tx, sched, brand, agenID := setupAgenBooking(t)
	nik := nikUji()
	orangLain := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Orang Lain", NIK: nik})
	req := agenReq(sched, PICInput{JamaahID: &orangLain, NamaLengkap: "Tebakan", JenisKelamin: "L", RoomType: "Quad", NIK: nikp(nikUji())})
	if _, err := processBookingTx(context.Background(), tx, brand, req, inisiatorAgen(agenID)); !errors.Is(err, ErrJamaahBukanMilik) {
		t.Fatalf("err = %v, want ErrJamaahBukanMilik", err)
	}
}

// K13 (diubah 2026-11-09): agen tidak mendapat komisi dari booking dirinya sendiri. "Saya sendiri" menandai booking pribadi, jadi
// komisi langsung mengalir ke upline agen (A), bukan ke agen (X). A tidak punya upline di sini, jadi tidak ada bonus pembinaan.
func TestBookingAgenSayaSendiriKomisiUntukUpline(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	sched, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	testdb.Exec(t, tx, `UPDATE schedules SET status='published', nominal_komisi_langsung=1000000, nominal_bonus_pembinaan=300000 WHERE id=?`, sched)
	a := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenA", StatusAgen: "aktif"})
	x := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenX", StatusAgen: "aktif", Direkrut: a, Upline: a})
	nik := nikUji()
	testdb.Exec(t, tx, `UPDATE customer_accounts SET nik_agen=? WHERE id=?`, nik, x) // NIK yang diisi X saat mendaftar agen (K24)
	jamaahX := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Agen X", Direkrut: a, NIK: nik})

	req := agenReq(sched, PICInput{NamaLengkap: "Agen X (uji)", JenisKelamin: "L", RoomType: "Quad", NIK: nikp(nik), SayaSendiri: true})
	resp, err := processBookingTx(ctx, tx, brand, req, inisiatorAgen(x))
	if err != nil {
		t.Fatalf("booking: %v", err)
	}
	if len(resp.Peringatan) != 0 {
		t.Fatalf("Saya sendiri tidak boleh diberi peringatan: %v", resp.Peringatan)
	}
	if resp.Jamaah.ID != jamaahX {
		t.Fatalf("jamaah = %d, want %d", resp.Jamaah.ID, jamaahX)
	}
	var penanda sql.NullInt64
	if err := tx.QueryRow(`SELECT bp.saya_sendiri_akun_id FROM booking_pax bp WHERE bp.booking_id=? AND bp.jamaah_id=?`, resp.Booking.BookingID, jamaahX).Scan(&penanda); err != nil || penanda.Int64 != x {
		t.Fatalf("saya_sendiri_akun_id = %v (err %v), want %d", penanda, err, x)
	}
	if st, by, _ := kaitan(t, tx, jamaahX); st != "terikat_agen" || by != a {
		t.Fatalf("kunci jamaah X berubah: %s/%d, want tetap A (%d)", st, by, a)
	}
	testdb.Exec(t, tx, `UPDATE bookings SET status='lunas' WHERE id=?`, resp.Booking.BookingID)
	if err := komisi.SnapshotNominal(ctx, tx, resp.Booking.BookingID); err != nil {
		t.Fatal(err)
	}
	if err := komisi.ProcessBookingLunas(ctx, tx, resp.Booking.BookingID); err != nil {
		t.Fatal(err)
	}
	jumlah := func(akun int64, jenis string) float64 {
		var v float64
		tx.QueryRow(`SELECT COALESCE(SUM(nominal),0) FROM transaksi_komisi WHERE booking_id=? AND akun_penerima_id=? AND jenis=?`, resp.Booking.BookingID, akun, jenis).Scan(&v)
		return v
	}
	if jumlah(x, "langsung") != 0 || jumlah(x, "pembinaan") != 0 {
		t.Fatalf("agen X tidak boleh mendapat komisi dari booking dirinya: langsung=%.0f pembinaan=%.0f", jumlah(x, "langsung"), jumlah(x, "pembinaan"))
	}
	if jumlah(a, "langsung") != 1_000_000 || jumlah(a, "pembinaan") != 0 {
		t.Fatalf("komisi A: langsung=%.0f pembinaan=%.0f, want 1000000 dan 0", jumlah(a, "langsung"), jumlah(a, "pembinaan"))
	}
}

// K24: "Saya sendiri" hanya diterima bila NIK jamaah pertama sama dengan NIK yang agen isi saat mendaftar.
// Agen Z tidak bisa mengetik NIK jamaah W (terkunci ke agen lain) lalu mencentang "Saya sendiri".
func TestBookingAgenSayaSendiriDitolakBilaNIKBukanMilikAgen(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	sched, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	testdb.Exec(t, tx, `UPDATE schedules SET status='published', nominal_komisi_langsung=1000000, nominal_bonus_pembinaan=300000 WHERE id=?`, sched)
	x := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenX", StatusAgen: "aktif"})
	z := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenZ", StatusAgen: "aktif"})
	nikW := nikUji()
	testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Jamaah W", Direkrut: x, NIK: nikW})
	testdb.Exec(t, tx, `UPDATE customer_accounts SET nik_agen=? WHERE id=?`, nikUji(), z)

	req := agenReq(sched, PICInput{NamaLengkap: "Jamaah W (uji)", JenisKelamin: "L", RoomType: "Quad", NIK: nikp(nikW), SayaSendiri: true})
	if _, err := processBookingTx(ctx, tx, brand, req, inisiatorAgen(z)); !errors.Is(err, ErrSayaSendiriBukanAnda) {
		t.Fatalf("err = %v, want ErrSayaSendiriBukanAnda", err)
	}

	// Agen tanpa NIK tersimpan juga ditolak (tidak ada pembanding).
	testdb.Exec(t, tx, `UPDATE customer_accounts SET nik_agen=NULL WHERE id=?`, z)
	if _, err := processBookingTx(ctx, tx, brand, req, inisiatorAgen(z)); !errors.Is(err, ErrSayaSendiriBukanAnda) {
		t.Fatalf("tanpa nik_agen: err = %v, want ErrSayaSendiriBukanAnda", err)
	}
}

func TestBookingAgenNonaktifDitolak(t *testing.T) {
	tx, sched, brand, agenID := setupAgenBooking(t)
	testdb.Exec(t, tx, `UPDATE customer_accounts SET status_agen='nonaktif' WHERE id=?`, agenID)
	req := agenReq(sched, PICInput{NamaLengkap: "X", JenisKelamin: "L", RoomType: "Quad", NIK: nikp(nikUji())})
	if _, err := processBookingTx(context.Background(), tx, brand, req, inisiatorAgen(agenID)); !errors.Is(err, ErrBukanAgenAktif) {
		t.Fatalf("err = %v, want ErrBukanAgenAktif", err)
	}
}

// K14: NIK 16 digit wajib untuk pax reguler di portal agen; bayi opsional.
func TestValidasiPortalAgenWajibNIK(t *testing.T) {
	base := func() BookingRequest {
		total, dp := 1000.0, 0.0
		return BookingRequest{
			RequestKey: strings.Repeat("a", 32), TermsVersion: TermsVersion, TermsAccepted: true, ExpectedTotal: &total, ExpectedDP: &dp,
			ScheduleID: 1,
			PIC:        PICInput{NamaLengkap: "Agen Baru", JenisKelamin: "L", RoomType: "Quad", NIK: nikp(nikUjiValid)},
		}
	}
	req := base()
	if msg := validateCheckout(&req, true); msg != "" {
		t.Fatalf("request valid ditolak: %q", msg)
	}
	req = base()
	req.PIC.NIK = nil
	if msg := validateCheckout(&req, true); !strings.Contains(msg, "NIK jamaah 1") {
		t.Fatalf("PIC tanpa NIK: %q", msg)
	}
	req = base()
	req.Anggota = []AnggotaInput{{PaxType: "reguler", NamaLengkap: "Anggota", JenisKelamin: "P", RoomType: quad()}}
	if msg := validateCheckout(&req, true); !strings.Contains(msg, "NIK jamaah 2") {
		t.Fatalf("anggota reguler tanpa NIK: %q", msg)
	}
	req = base()
	tl := "2026-06-01"
	req.Anggota = []AnggotaInput{{PaxType: "infant", NamaLengkap: "Bayi", JenisKelamin: "P", TanggalLahir: &tl}}
	if msg := validateCheckout(&req, true); msg != "" {
		t.Fatalf("bayi tanpa NIK harus lolos: %q", msg)
	}
}

// Booking publik: akun yang terikat ke agen (saat dibuat) menentukan agen penjaga jamaah baru. Jamaah yang sudah
// terkunci agen lain tidak berubah. Akun tanpa agen membuat jamaah tetap belum_ditentukan.
func TestBookingPublikMengikutiAgenPengikatAkun(t *testing.T) {
	tx, sched, brand, _ := setupAgenBooking(t)
	ctx := context.Background()
	w := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenW", StatusAgen: "aktif"})
	x := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenX", StatusAgen: "aktif"})
	zTerikat := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Z", Direkrut: x}) // terikat ke X; W dan Q tidak berperan
	zTanpaAgen := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "ZTanpa"})
	nikTerkunci := nikUji()
	terkunciW := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Milik W", Direkrut: w, NIK: nikTerkunci})

	req := BookingRequest{
		ScheduleID: sched,
		PIC:        PICInput{NamaLengkap: "PIC Z", JenisKelamin: "P", RoomType: "Quad", NIK: nikp(nikUji())},
		Anggota: []AnggotaInput{
			{PaxType: "reguler", NamaLengkap: "Anggota Baru", JenisKelamin: "L", RoomType: quad(), NIK: nikp(nikUji())},
			{PaxType: "reguler", NamaLengkap: "Milik W (uji)", JenisKelamin: "L", RoomType: quad(), NIK: nikp(nikTerkunci)},
		},
	}
	resp, err := processBookingTx(ctx, tx, brand, req, inisiator{accountID: zTerikat})
	if err != nil {
		t.Fatalf("booking publik: %v", err)
	}
	rows, err := tx.Query(`SELECT jamaah_id FROM booking_pax WHERE booking_id=? ORDER BY id`, resp.Booking.BookingID)
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
		t.Fatalf("pax = %v", pax)
	}
	for _, id := range pax[:2] {
		if st, by, sumber := kaitan(t, tx, id); st != "terikat_agen" || by != x || sumber != "jalur2" {
			t.Fatalf("pax baru %d = %s/%d/%s, want terikat_agen/%d/jalur2 (agen pengikat akun)", id, st, by, sumber, x)
		}
	}
	if pax[2] != terkunciW {
		t.Fatalf("pax ketiga = %d, want jamaah terkunci %d", pax[2], terkunciW)
	}
	if st, by, _ := kaitan(t, tx, terkunciW); st != "terikat_agen" || by != w {
		t.Fatalf("kunci seumur hidup berubah: %s/%d, want tetap W (%d)", st, by, w)
	}

	// Akun tanpa agen: jamaah barunya tetap belum_ditentukan.
	req2 := BookingRequest{
		ScheduleID: sched,
		PIC:        PICInput{NamaLengkap: "PIC Tanpa Agen", JenisKelamin: "P", RoomType: "Quad", NIK: nikp(nikUji())},
	}
	resp2, err := processBookingTx(ctx, tx, brand, req2, inisiator{accountID: zTanpaAgen})
	if err != nil {
		t.Fatalf("booking tanpa agen: %v", err)
	}
	if st, _, _ := kaitan(t, tx, resp2.Jamaah.ID); st != "belum_ditentukan" {
		t.Fatalf("jamaah akun tanpa agen = %s, want belum_ditentukan", st)
	}
	var pembuat sql.NullInt64
	var pemilik sql.NullInt64
	if err := tx.QueryRow(`SELECT customer_account_id, dibuat_oleh_agen_akun_id FROM bookings WHERE id_booking=?`, resp.Booking.BookingCode).Scan(&pemilik, &pembuat); err != nil {
		t.Fatal(err)
	}
	if pemilik.Int64 != zTerikat || pembuat.Valid {
		t.Fatalf("booking publik: pemilik=%v pembuat=%v, want %d/NULL", pemilik, pembuat, zTerikat)
	}
}

// Jamaah diri agen (NIK sama dengan NIK agen) dikunci ke upline aktif agen, bukan ke agen sendiri. Agen tanpa upline aktif: tidak
// dikunci ke siapa pun. Jamaah lain tetap dikunci ke agen pembuat booking.
func TestBookingAgenJamaahDiriDikunciKeUpline(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	sched, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	testdb.Exec(t, tx, `UPDATE schedules SET status='published', nominal_komisi_langsung=1000000, nominal_bonus_pembinaan=300000 WHERE id=?`, sched)
	a := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenA", StatusAgen: "aktif"})
	x := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenX", StatusAgen: "aktif", Direkrut: a, Upline: a})
	tanpa := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenTanpa", StatusAgen: "aktif"})
	nikX, nikTanpa := nikUji(), nikUji()
	testdb.Exec(t, tx, `UPDATE customer_accounts SET nik_agen=? WHERE id=?`, nikX, x)
	testdb.Exec(t, tx, `UPDATE customer_accounts SET nik_agen=? WHERE id=?`, nikTanpa, tanpa)

	req := agenReq(sched, PICInput{NamaLengkap: "Agen X (uji)", JenisKelamin: "L", RoomType: "Quad", NIK: nikp(nikX), SayaSendiri: true})
	req.Anggota = []AnggotaInput{{PaxType: "reguler", NamaLengkap: "Keluarga X", JenisKelamin: "P", RoomType: quad(), NIK: nikp(nikUji())}}
	resp, err := processBookingTx(ctx, tx, brand, req, inisiatorAgen(x))
	if err != nil {
		t.Fatalf("booking X: %v", err)
	}
	if st, by, _ := kaitan(t, tx, resp.Jamaah.ID); st != "terikat_agen" || by != a {
		t.Fatalf("jamaah diri X = %s/%d, want terikat_agen/%d (upline)", st, by, a)
	}
	var keluarga int64
	if err := tx.QueryRow(`SELECT jamaah_id FROM booking_pax WHERE booking_id=? ORDER BY id DESC LIMIT 1`, resp.Booking.BookingID).Scan(&keluarga); err != nil {
		t.Fatal(err)
	}
	if st, by, _ := kaitan(t, tx, keluarga); st != "terikat_agen" || by != x {
		t.Fatalf("jamaah keluarga X = %s/%d, want terikat_agen/%d", st, by, x)
	}

	req2 := agenReq(sched, PICInput{NamaLengkap: "Agen Tanpa (uji)", JenisKelamin: "L", RoomType: "Quad", NIK: nikp(nikTanpa), SayaSendiri: true})
	resp2, err := processBookingTx(ctx, tx, brand, req2, inisiatorAgen(tanpa))
	if err != nil {
		t.Fatalf("booking tanpa upline: %v", err)
	}
	if st, _, _ := kaitan(t, tx, resp2.Jamaah.ID); st != "belum_ditentukan" {
		t.Fatalf("jamaah diri agen tanpa upline = %s, want belum_ditentukan", st)
	}
}

// Pemesan ikut berangkat dengan NIK yang cocok dengan jamaah yang sudah dikunci agen X: akun tanpa pengikat menjadi downline X,
// sehingga jamaah baru yang ia pesankan juga terkunci ke X. Akun yang sudah punya pengikat tidak berubah, dan pemesan yang tidak ikut
// berangkat tidak mengikat akunnya.
func TestBookingPublikMengikatAkunKeAgenPemilikKunciJamaah(t *testing.T) {
	tx, sched, brand, _ := setupAgenBooking(t)
	ctx := context.Background()
	x := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenX", StatusAgen: "aktif"})
	y := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenY", StatusAgen: "aktif"})
	pengikat := func(akun int64) int64 {
		var v sql.NullInt64
		if err := tx.QueryRow(`SELECT direkrut_oleh_akun_id FROM customer_accounts WHERE id=?`, akun).Scan(&v); err != nil {
			t.Fatal(err)
		}
		return v.Int64
	}

	// Z dibookingkan X sebelumnya (jamaah terkunci ke X); lalu Z membuat akun tanpa link referral dan memesan sendiri.
	nikZ := nikUji()
	testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Z Terkunci", Direkrut: x, NIK: nikZ})
	akunZ := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AkunZ"})
	resp, err := processBookingTx(ctx, tx, brand, BookingRequest{
		ScheduleID: sched,
		PIC:        PICInput{NamaLengkap: "Z Terkunci", JenisKelamin: "L", RoomType: "Quad", NIK: nikp(nikZ)},
		Anggota:    []AnggotaInput{{PaxType: "reguler", NamaLengkap: "Anak Z", JenisKelamin: "L", RoomType: quad(), NIK: nikp(nikUji())}},
	}, inisiator{accountID: akunZ})
	if err != nil {
		t.Fatalf("booking Z: %v", err)
	}
	if got := pengikat(akunZ); got != x {
		t.Fatalf("pengikat akun Z = %d, want X %d", got, x)
	}
	var anak int64
	if err := tx.QueryRow(`SELECT jamaah_id FROM booking_pax WHERE booking_id=? ORDER BY id DESC LIMIT 1`, resp.Booking.BookingID).Scan(&anak); err != nil {
		t.Fatal(err)
	}
	if st, by, _ := kaitan(t, tx, anak); st != "terikat_agen" || by != x {
		t.Fatalf("jamaah baru anak Z = %s/%d, want terikat_agen/%d", st, by, x)
	}

	// Akun yang sudah punya pengikat Y tidak dipindah ke X walau NIK-nya milik jamaah terkunci X.
	nikV := nikUji()
	testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "V Terkunci", Direkrut: x, NIK: nikV})
	akunV := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AkunV", Direkrut: y})
	if _, err := processBookingTx(ctx, tx, brand, BookingRequest{
		ScheduleID: sched,
		PIC:        PICInput{NamaLengkap: "V Terkunci", JenisKelamin: "L", RoomType: "Quad", NIK: nikp(nikV)},
	}, inisiator{accountID: akunV}); err != nil {
		t.Fatalf("booking V: %v", err)
	}
	if got := pengikat(akunV); got != y {
		t.Fatalf("pengikat akun V = %d, want tetap Y %d", got, y)
	}

	// Pemesan tidak ikut berangkat: akun tidak diikat walau salah satu penumpang terkunci ke X.
	nikU := nikUji()
	testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "U Terkunci", Direkrut: x, NIK: nikU})
	akunPemesan := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Pemesan"})
	if _, err := processBookingTx(ctx, tx, brand, BookingRequest{
		ScheduleID:       sched,
		PemesanTidakIkut: true,
		PIC:              PICInput{NamaLengkap: "U Terkunci", JenisKelamin: "L", RoomType: "Quad", NIK: nikp(nikU)},
	}, inisiator{accountID: akunPemesan}); err != nil {
		t.Fatalf("booking pemesan tidak ikut: %v", err)
	}
	if got := pengikat(akunPemesan); got != 0 {
		t.Fatalf("pengikat akun pemesan = %d, want 0 (tidak ikut berangkat)", got)
	}
}

// PIC yang ikut berangkat dicocokkan lewat NIK dan nama seperti penumpang lain: dua akun yang memasukkan NIK
// yang sama memakai satu baris jamaah; NIK sama dengan nama berbeda ditolak generik.
func TestBookingPublikPICCocokNIKDanNama(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	sched, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	testdb.Exec(t, tx, `UPDATE schedules SET status='published' WHERE id=?`, sched)
	sched2, _ := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-02-01"})
	_ = sched2
	akun1 := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Akun1"})
	akun2 := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Akun2"})
	nik := nikUji()
	pic := func() PICInput {
		return PICInput{NamaLengkap: "Budi Santoso", NoHP: nomorUji(21), JenisKelamin: "L", RoomType: "Quad", NIK: nikp(nik)}
	}

	r1, err := processBookingTx(ctx, tx, brand, BookingRequest{ScheduleID: sched, PIC: pic()}, inisiator{accountID: akun1})
	if err != nil {
		t.Fatalf("akun 1: %v", err)
	}
	// Akun lain, nama sama (beda kapitalisasi dan spasi): jamaah yang sama dipakai ulang, tidak ada duplikat.
	p2 := pic()
	p2.NamaLengkap = "  budi   SANTOSO "
	testdb.Exec(t, tx, `UPDATE bookings SET status='batal' WHERE id=?`, r1.Booking.BookingID)
	r2, err := processBookingTx(ctx, tx, brand, BookingRequest{ScheduleID: sched, PIC: p2}, inisiator{accountID: akun2})
	if err != nil {
		t.Fatalf("akun 2: %v", err)
	}
	if r1.Jamaah.ID != r2.Jamaah.ID {
		t.Fatalf("jamaah akun 1 = %d, akun 2 = %d, want sama (NIK sama)", r1.Jamaah.ID, r2.Jamaah.ID)
	}
	var n int
	tx.QueryRow(`SELECT COUNT(*) FROM jamaah WHERE brand_id=? AND nik=?`, brand, nik).Scan(&n)
	if n != 1 {
		t.Fatalf("baris jamaah dengan NIK itu = %d, want 1", n)
	}

	// Nama berbeda untuk NIK yang sama: ditolak dengan pesan generik.
	p3 := pic()
	p3.NamaLengkap = "Orang Lain Sama Sekali"
	_, err = processBookingTx(ctx, tx, brand, BookingRequest{ScheduleID: sched, PIC: p3}, inisiator{accountID: akun1})
	if !errors.Is(err, ErrNIKAtasNamaLain) {
		t.Fatalf("err = %v, want ErrNIKAtasNamaLain", err)
	}
	if strings.Contains(err.Error(), "Budi") {
		t.Fatalf("pesan membuka nama terdaftar: %q", err.Error())
	}
}

// Bayi tanpa NIK dicocokkan lewat nama dan tanggal lahir; nomor WA bukan kunci identitas jamaah.
func TestAnggotaTanpaNIKCocokNamaDanTanggalLahir(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	_, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	tl := "2026-03-04"
	hp := nomorUji(31)
	bayi := AnggotaInput{PaxType: "infant", NamaLengkap: "Bayi Sama", JenisKelamin: "L", TanggalLahir: &tl, NoHP: &hp}
	id1, err := resolveAnggotaPublik(ctx, tx, brand, bayi)
	if err != nil {
		t.Fatal(err)
	}
	id2, err := resolveAnggotaPublik(ctx, tx, brand, bayi)
	if err != nil || id1 != id2 {
		t.Fatalf("bayi sama: id1=%d id2=%d err=%v, want dipakai ulang", id1, id2, err)
	}
	// Nama beda dengan nomor WA yang sama (keluarga berbagi nomor): jamaah baru, tanpa penolakan.
	saudara := AnggotaInput{PaxType: "infant", NamaLengkap: "Bayi Saudara", JenisKelamin: "P", TanggalLahir: &tl, NoHP: &hp}
	id3, err := resolveAnggotaPublik(ctx, tx, brand, saudara)
	if err != nil || id3 == id1 {
		t.Fatalf("saudara: id=%d err=%v, want jamaah baru", id3, err)
	}
	var n int
	tx.QueryRow(`SELECT COUNT(*) FROM jamaah WHERE brand_id=? AND no_hp=?`, brand, hp).Scan(&n)
	if n != 2 {
		t.Fatalf("jamaah dengan nomor yang sama = %d, want 2 (nomor bukan kunci identitas)", n)
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
		akun := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "PIC Infant"})
		req := BookingRequest{
			ScheduleID: sched,
			PIC:        PICInput{NamaLengkap: "PIC Infant", JenisKelamin: "P", RoomType: "Quad", NIK: nikp(nikUji())},
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

// akunUji membuat akun pelanggan di transaksi uji dengan nomor WA tertentu.
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
