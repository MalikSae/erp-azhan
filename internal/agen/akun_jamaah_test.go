package agen

import (
	"bytes"
	"net/http/httptest"
	"erp-azhan/api/internal/identity"
	"context"
	"database/sql"
	"fmt"
	"testing"

	"erp-azhan/api/internal/komisi"
	"erp-azhan/api/internal/testdb"
)

// Akun jamaah hasil aktivasi (E3): Ibrahim (agen aktif) memesankan Ahmad; akun Ahmad tertaut ke jamaahnya.
type fxJamaahTertaut struct {
	f        fixture
	sched    int64
	ibrahim  int64
	jamaah   int64
	nik      string
	akunUji  int64
	bookings []int64
}

func newFxTertaut(t *testing.T, biaya float64, o testdb.JamaahOpts, uplineAkun int64) *fxJamaahTertaut {
	t.Helper()
	f := setup(t, biaya)
	sched, _ := testdb.Schedule(t, f.tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	x := &fxJamaahTertaut{f: f, sched: sched, nik: "9999999999993001"}
	x.ibrahim = testdb.NewAkun(t, f.tx, f.brandID, testdb.AkunOpts{Nama: "Ibrahim", StatusAgen: "aktif"})
	if o.Direkrut == -1 {
		o.Direkrut = x.ibrahim
	}
	o.Nama, o.NIK = "Ahmad", x.nik
	x.jamaah = testdb.NewJamaah(t, f.tx, f.brandID, o)
	x.akunUji = testdb.NewAkun(t, f.tx, f.brandID, testdb.AkunOpts{Nama: "AkunAhmad", Direkrut: uplineAkun})
	testdb.Exec(t, f.tx, `UPDATE customer_accounts SET jamaah_id=? WHERE id=?`, x.jamaah, x.akunUji)
	return x
}

func (x *fxJamaahTertaut) booking(t *testing.T, status string, pax ...testdb.Pax) int64 {
	t.Helper()
	if len(pax) == 0 {
		pax = []testdb.Pax{{JamaahID: x.jamaah}}
	}
	id, _ := testdb.NewBookingPax(t, x.f.tx, x.sched, status, 1000, pax...)
	x.bookings = append(x.bookings, id)
	return id
}

func (x *fxJamaahTertaut) akunJamaah(t *testing.T) bool {
	t.Helper()
	ok, err := akunJamaah(context.Background(), x.f.tx, x.akunUji)
	must(t, "akunJamaah", err)
	return ok
}

// akunJamaah hanya untuk jamaah tertaut terikat_agen dengan pemilik kunci = upline akun dan booking dp/lunas.
func TestAkunJamaahSempit(t *testing.T) {
	// Terikat, upline akun = pemilik kunci.
	x := newFxTertaut(t, 500_000, testdb.JamaahOpts{Direkrut: -1}, 0)
	testdb.Exec(t, x.f.tx, `UPDATE customer_accounts SET direkrut_oleh_akun_id=? WHERE id=?`, x.ibrahim, x.akunUji)
	if x.akunJamaah(t) {
		t.Fatal("tanpa booking: bukan akun jamaah")
	}
	b := x.booking(t, "baru")
	if x.akunJamaah(t) {
		t.Fatal("booking baru: bukan akun jamaah")
	}
	testdb.Exec(t, x.f.tx, `UPDATE bookings SET status='batal' WHERE id=?`, b)
	if x.akunJamaah(t) {
		t.Fatal("booking batal: bukan akun jamaah")
	}
	testdb.Exec(t, x.f.tx, `UPDATE bookings SET status='draft' WHERE id=?`, b)
	if x.akunJamaah(t) {
		t.Fatal("booking draft: bukan akun jamaah")
	}
	testdb.Exec(t, x.f.tx, `UPDATE bookings SET status='dp' WHERE id=?`, b)
	if !x.akunJamaah(t) {
		t.Fatal("booking dp: akun jamaah")
	}
	testdb.Exec(t, x.f.tx, `UPDATE bookings SET status='lunas' WHERE id=?`, b)
	if !x.akunJamaah(t) {
		t.Fatal("booking lunas: akun jamaah")
	}

	// Pax bayi dan pax batal tidak dihitung.
	testdb.Exec(t, x.f.tx, `UPDATE booking_pax SET pax_type='infant' WHERE booking_id=?`, b)
	if x.akunJamaah(t) {
		t.Fatal("pax bayi: bukan akun jamaah")
	}
	testdb.Exec(t, x.f.tx, `UPDATE booking_pax SET pax_type='reguler', pax_status='batal' WHERE booking_id=?`, b)
	if x.akunJamaah(t) {
		t.Fatal("pax batal: bukan akun jamaah")
	}
	testdb.Exec(t, x.f.tx, `UPDATE booking_pax SET pax_status='aktif' WHERE booking_id=?`, b)

	// Upline akun berbeda dari pemilik kunci, atau belum terisi.
	z := testdb.NewAkun(t, x.f.tx, x.f.brandID, testdb.AkunOpts{Nama: "Z", StatusAgen: "aktif"})
	testdb.Exec(t, x.f.tx, `UPDATE customer_accounts SET direkrut_oleh_akun_id=? WHERE id=?`, z, x.akunUji)
	if x.akunJamaah(t) {
		t.Fatal("upline akun beda dari pemilik kunci: bukan akun jamaah")
	}
	testdb.Exec(t, x.f.tx, `UPDATE customer_accounts SET direkrut_oleh_akun_id=NULL WHERE id=?`, x.akunUji)
	if x.akunJamaah(t) {
		t.Fatal("upline akun kosong: bukan akun jamaah")
	}
	testdb.Exec(t, x.f.tx, `UPDATE customer_accounts SET direkrut_oleh_akun_id=?, upline_akun_id=? WHERE id=?`, x.ibrahim, z, x.akunUji)
	if x.akunJamaah(t) {
		t.Fatal("upline_akun_id terisi beda: bukan akun jamaah")
	}
	testdb.Exec(t, x.f.tx, `UPDATE customer_accounts SET upline_akun_id=? WHERE id=?`, x.ibrahim, x.akunUji)
	if !x.akunJamaah(t) {
		t.Fatal("upline_akun_id = pemilik kunci: akun jamaah")
	}
}

// tanpa_agen (pilihan admin maupun jamaah baru, bekas belum_ditentukan) tetap lewat persetujuan admin, walau booking dp.
func TestAkunJamaahTanpaAgenDanBelumDitentukan(t *testing.T) {
	for _, o := range []testdb.JamaahOpts{{TanpaAgen: true}, {}} {
		t.Run(fmt.Sprintf("tanpa_agen-%v", o.TanpaAgen), func(t *testing.T) { tanpaAgenDanBelumDitentukan(t, o) })
	}
}

func tanpaAgenDanBelumDitentukan(t *testing.T, o testdb.JamaahOpts) {
	{
		x := newFxTertaut(t, 500_000, o, 0)
		x.booking(t, "dp")
		if x.akunJamaah(t) {
			t.Fatalf("jamaah %+v: bukan akun jamaah", o)
		}
		// Upline akun terisi tidak mengubah hasilnya.
		testdb.Exec(t, x.f.tx, `UPDATE customer_accounts SET direkrut_oleh_akun_id=? WHERE id=?`, x.ibrahim, x.akunUji)
		if x.akunJamaah(t) {
			t.Fatalf("jamaah %+v dengan upline akun: bukan akun jamaah", o)
		}
		// Admin menyetujui dengan biaya normal.
		must(t, "ajukan", ajukanTx(context.Background(), x.f.tx, x.akunUji, ajuan))
		var status string
		var otomatis bool
		must(t, "baca", x.f.tx.QueryRow(`SELECT status, disetujui_otomatis FROM pembayaran_pendaftaran_agen WHERE akun_id=?`, x.akunUji).Scan(&status, &otomatis))
		if status != "menunggu_verifikasi" || otomatis {
			t.Fatalf("jamaah %+v: pembayaran %s otomatis=%v, want menunggu_verifikasi", o, status, otomatis)
		}
		if s := x.f.status(t, x.akunUji); s.StatusAgen != "pengajuan" {
			t.Fatalf("jamaah %+v: status %s, want pengajuan", o, s.StatusAgen)
		}
	}
}

// Definisi lama tetap berlaku: booking sendiri dp/lunas.
func TestAkunJamaahBookingSendiriTetapBerlaku(t *testing.T) {
	x := newFxTertaut(t, 500_000, testdb.JamaahOpts{TanpaAgen: true}, 0)
	b := x.booking(t, "dp")
	if x.akunJamaah(t) {
		t.Fatal("tanpa_agen: bukan akun jamaah")
	}
	testdb.Exec(t, x.f.tx, `UPDATE bookings SET customer_account_id=? WHERE id=?`, x.akunUji, b)
	if !x.akunJamaah(t) {
		t.Fatal("booking sendiri dp: akun jamaah")
	}
}

// Akun hasil aktivasi dengan jamaah terikat Ibrahim: ajukan gratis dan aktif, nik_agen dari jamaah, upline Ibrahim.
// Isian NIK pengguna diabaikan.
func TestAjukanAkunTertautOtomatisAktif(t *testing.T) {
	x := newFxTertaut(t, 500_000, testdb.JamaahOpts{Direkrut: -1}, 0)
	x.booking(t, "dp")
	// Aktivasi menautkan lewat IkatAkunDariKunciTx; akun uji belum punya direkrut.
	must(t, "ikat", IkatAkunDariKunciTx(context.Background(), x.f.tx, x.akunUji, x.jamaah))

	req := ajuan
	req.NIK = "1111111111111111" // diabaikan
	must(t, "ajukan", ajukanTx(context.Background(), x.f.tx, x.akunUji, req))

	var status string
	var nik sql.NullString
	var upline, direkrut sql.NullInt64
	must(t, "baca akun", x.f.tx.QueryRow(`SELECT status_agen, nik_agen, upline_akun_id, direkrut_oleh_akun_id FROM customer_accounts WHERE id=?`, x.akunUji).
		Scan(&status, &nik, &upline, &direkrut))
	if status != "aktif" || nik.String != x.nik || upline.Int64 != x.ibrahim || direkrut.Int64 != x.ibrahim {
		t.Fatalf("akun = %s nik=%q upline=%d direkrut=%d, want aktif %s Ibrahim", status, nik.String, upline.Int64, direkrut.Int64, x.nik)
	}
	var bayar string
	var otomatis bool
	must(t, "baca bayar", x.f.tx.QueryRow(`SELECT status, disetujui_otomatis FROM pembayaran_pendaftaran_agen WHERE akun_id=?`, x.akunUji).Scan(&bayar, &otomatis))
	if bayar != "dibebaskan" || !otomatis {
		t.Fatalf("pembayaran %s otomatis=%v, want dibebaskan otomatis", bayar, otomatis)
	}
}

// Akun beda upline dari pemilik kunci: pengajuan tidak otomatis; admin memutuskan dengan biaya normal.
func TestAjukanAkunTertautBedaUplineLewatAdmin(t *testing.T) {
	x := newFxTertaut(t, 500_000, testdb.JamaahOpts{Direkrut: -1}, 0)
	x.booking(t, "dp")
	z := testdb.NewAkun(t, x.f.tx, x.f.brandID, testdb.AkunOpts{Nama: "Z", StatusAgen: "aktif"})
	testdb.Exec(t, x.f.tx, `UPDATE customer_accounts SET direkrut_oleh_akun_id=? WHERE id=?`, z, x.akunUji)
	must(t, "ajukan", ajukanTx(context.Background(), x.f.tx, x.akunUji, ajuan))
	if s := x.f.status(t, x.akunUji); s.StatusAgen != "pengajuan" || s.AkunJamaah {
		t.Fatalf("status %s akunJamaah=%v, want pengajuan dan bukan akun jamaah", s.StatusAgen, s.AkunJamaah)
	}
	// NIK tetap dipaksa dari jamaah tertaut.
	var nik sql.NullString
	must(t, "nik", x.f.tx.QueryRow(`SELECT nik_agen FROM customer_accounts WHERE id=?`, x.akunUji).Scan(&nik))
	if nik.String != x.nik {
		t.Fatalf("nik_agen = %q, want %s", nik.String, x.nik)
	}
}

// NIK jamaah tertaut sudah dipegang akun lain: galat "Hubungi admin"; admin menyelesaikan lewat Kosongkan NIK Agen.
func TestAjukanAkunTertautNIKDipegangAkunLain(t *testing.T) {
	x := newFxTertaut(t, 0, testdb.JamaahOpts{Direkrut: -1}, 0)
	x.booking(t, "dp")
	penyerobot := testdb.NewAkun(t, x.f.tx, x.f.brandID, testdb.AkunOpts{Nama: "Penyerobot", StatusAgen: "pengajuan"})
	testdb.Exec(t, x.f.tx, `UPDATE customer_accounts SET nik_agen=? WHERE id=?`, x.nik, penyerobot)
	_, err := x.f.tx.Exec(`SAVEPOINT sebelum`)
	must(t, "savepoint", err)
	mustErr(t, "NIK dipegang akun lain", ajukanTx(context.Background(), x.f.tx, x.akunUji, ajuan), ErrNIKAgenTidakBisaDipakai)
	if ErrNIKAgenTidakBisaDipakai.Error() != "NIK tidak dapat dipakai. Hubungi admin." {
		t.Fatalf("pesan = %q", ErrNIKAgenTidakBisaDipakai.Error())
	}
	_, err = x.f.tx.Exec(`ROLLBACK TO SAVEPOINT sebelum`)
	must(t, "rollback savepoint", err)

	// Setelah NIK dikosongkan dari akun lain, ajukan berhasil.
	testdb.Exec(t, x.f.tx, `UPDATE customer_accounts SET nik_agen=NULL WHERE id=?`, penyerobot)
	must(t, "ajukan", ajukanTx(context.Background(), x.f.tx, x.akunUji, ajuan))
}

// Akun tidak tertaut: NIK yang sudah tercatat sebagai NIK jamaah (orang lain atau belum ditautkan) ditolak.
func TestAjukanAkunTidakTertautNIKJamaahDitolak(t *testing.T) {
	f := setup(t, 0)
	ctx := context.Background()
	const nikLain = "9999999999993101"
	const nikBebas = "9999999999993102"
	testdb.NewJamaah(t, f.tx, f.brandID, testdb.JamaahOpts{Nama: "Orang Lain", NIK: nikLain})
	akun := testdb.NewAkun(t, f.tx, f.brandID, testdb.AkunOpts{Nama: "Pemohon"})
	req := ajuan
	req.NIK = nikLain
	mustErr(t, "NIK jamaah lain", ajukanTx(ctx, f.tx, akun, req), ErrNIKAgenTidakBisaDipakai)
	if s := f.status(t, akun); s.StatusAgen != "tidak_aktif" {
		t.Fatalf("status %s, want tidak_aktif (tidak ada efek samping)", s.StatusAgen)
	}
	req.NIK = nikBebas
	must(t, "NIK bebas", ajukanTx(ctx, f.tx, akun, req))
}

// Pemilik kunci yang mengajukan sebagai agen untuk jamaahnya sendiri (kunci = akun itu): tidak bentrok, tidak otomatis.
func TestAjukanAkunTertautPemilikKunciSendiri(t *testing.T) {
	x := newFxTertaut(t, 500_000, testdb.JamaahOpts{}, 0)
	testdb.Exec(t, x.f.tx, `UPDATE jamaah SET direkrut_oleh_akun_id=?, kaitan_status='terikat_agen' WHERE id=?`, x.akunUji, x.jamaah)
	x.booking(t, "dp")
	must(t, "ajukan", ajukanTx(context.Background(), x.f.tx, x.akunUji, ajuan))
	if s := x.f.status(t, x.akunUji); s.StatusAgen != "pengajuan" || s.AkunJamaah {
		t.Fatalf("status %s akunJamaah=%v, want pengajuan lewat admin", s.StatusAgen, s.AkunJamaah)
	}
}

// Admin Ganti Upline Ahmad (akun jamaah, agen aktif) ke Z saat booking pendaftarannya (B1, dibuat Ibrahim) belum lunas:
// komisi B1 mengikuti kunci jamaah (Ibrahim), bukan upline akun lama maupun baru. Jamaah Ahmad tidak ikut pindah
// karena booking pertamanya dibuat Ibrahim.
func TestGantiUplineAkunJamaahKomisiMengikutiKunci(t *testing.T) {
	ctx := context.Background()
	f := setup(t, 500_000)
	sched, _ := testdb.Schedule(t, f.tx, testdb.ScheduleOpts{
		KomisiLangsung: func() *float64 { v := 1_000_000.0; return &v }(), BonusPembinaan: func() *float64 { v := 300_000.0; return &v }(),
		BerangkatTanggal: "2099-01-01",
	})
	ibrahim := testdb.NewAkun(t, f.tx, f.brandID, testdb.AkunOpts{Nama: "Ibrahim", StatusAgen: "aktif"})
	z := testdb.NewAkun(t, f.tx, f.brandID, testdb.AkunOpts{Nama: "Z", StatusAgen: "aktif"})
	jamaah := testdb.NewJamaah(t, f.tx, f.brandID, testdb.JamaahOpts{Nama: "Ahmad", Direkrut: ibrahim, NIK: "9999999999993201"})
	ahmad := testdb.NewAkun(t, f.tx, f.brandID, testdb.AkunOpts{Nama: "AkunAhmad", Direkrut: ibrahim})
	testdb.Exec(t, f.tx, `UPDATE customer_accounts SET jamaah_id=? WHERE id=?`, jamaah, ahmad)
	b1, _ := testdb.NewBookingPax(t, f.tx, sched, "dp", 1000, testdb.Pax{JamaahID: jamaah})
	testdb.Exec(t, f.tx, `UPDATE bookings SET customer_account_id=? WHERE id=?`, ibrahim, b1)

	must(t, "ajukan", ajukanTx(ctx, f.tx, ahmad, ajuan))
	if s := f.status(t, ahmad); s.StatusAgen != "aktif" {
		t.Fatalf("Ahmad %s, want aktif otomatis", s.StatusAgen)
	}

	must(t, "ganti upline", gantiAgenAkunTx(ctx, f.tx, nil, ahmad, f.adminID, GantiAgenAkunRequest{AgenAkunID: &z, Alasan: "dipindah admin"}))
	var upline sql.NullInt64
	must(t, "upline", f.tx.QueryRow(`SELECT upline_akun_id FROM customer_accounts WHERE id=?`, ahmad).Scan(&upline))
	if upline.Int64 != z {
		t.Fatalf("upline Ahmad = %d, want Z", upline.Int64)
	}

	must(t, "snapshot", komisi.SnapshotNominal(ctx, f.tx, b1))
	testdb.Exec(t, f.tx, `UPDATE bookings SET status='lunas' WHERE id=?`, b1)
	must(t, "komisi", komisi.ProcessBookingLunas(ctx, f.tx, b1))
	rows, err := f.tx.Query(`SELECT akun_penerima_id, jenis FROM transaksi_komisi WHERE booking_id=? AND dibatalkan_at IS NULL`, b1)
	must(t, "ledger", err)
	defer rows.Close()
	n := 0
	for rows.Next() {
		var penerima int64
		var jenis string
		must(t, "scan", rows.Scan(&penerima, &jenis))
		n++
		if penerima != ibrahim || jenis != "langsung" {
			t.Fatalf("komisi B1 ke %d (%s), want Ibrahim langsung", penerima, jenis)
		}
	}
	if n != 1 {
		t.Fatalf("baris komisi B1 = %d, want 1", n)
	}
}

// Akun tidak tertaut yang memesan untuk dirinya (jamaah buatan akun itu, booking sendiri dp) mengajukan dengan
// NIK-nya sendiri: lolos, bebas biaya, aktif otomatis (definisi lama akun jamaah). Jamaah buatan akun lain,
// atau buatan akun ini tetapi tertaut ke akun lain, tetap ditolak.
func TestAjukanAkunTidakTertautJamaahBuatanSendiri(t *testing.T) {
	f := setup(t, 500_000)
	ctx := context.Background()
	sched, _ := testdb.Schedule(t, f.tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	const nikSendiri, nikOrang, nikTertautLain = "9999999999993201", "9999999999993202", "9999999999993203"

	pemesan := testdb.NewAkun(t, f.tx, f.brandID, testdb.AkunOpts{Nama: "Pemesan"})
	lain := testdb.NewAkun(t, f.tx, f.brandID, testdb.AkunOpts{Nama: "Lain"})
	jSendiri := testdb.NewJamaah(t, f.tx, f.brandID, testdb.JamaahOpts{Nama: "Pemesan", NIK: nikSendiri})
	jOrang := testdb.NewJamaah(t, f.tx, f.brandID, testdb.JamaahOpts{Nama: "Orang", NIK: nikOrang})
	jTertaut := testdb.NewJamaah(t, f.tx, f.brandID, testdb.JamaahOpts{Nama: "Tertaut", NIK: nikTertautLain})
	testdb.Exec(t, f.tx, `UPDATE jamaah SET dibuat_oleh_akun_id=? WHERE id IN (?, ?)`, pemesan, jSendiri, jTertaut)
	testdb.Exec(t, f.tx, `UPDATE jamaah SET dibuat_oleh_akun_id=? WHERE id=?`, lain, jOrang)
	testdb.Exec(t, f.tx, `UPDATE customer_accounts SET jamaah_id=? WHERE id=?`, jTertaut, lain)
	b, _ := testdb.NewBookingPax(t, f.tx, sched, "dp", 1000, testdb.Pax{JamaahID: jSendiri})
	testdb.Exec(t, f.tx, `UPDATE bookings SET customer_account_id=? WHERE id=?`, pemesan, b)

	for _, nik := range []string{nikOrang, nikTertautLain} {
		_, err := f.tx.Exec(`SAVEPOINT s`)
		must(t, "savepoint", err)
		req := ajuan
		req.NIK = nik
		mustErr(t, "NIK "+nik, ajukanTx(ctx, f.tx, pemesan, req), ErrNIKAgenTidakBisaDipakai)
		_, err = f.tx.Exec(`ROLLBACK TO SAVEPOINT s`)
		must(t, "rollback", err)
	}

	req := ajuan
	req.NIK = nikSendiri
	must(t, "ajukan NIK sendiri", ajukanTx(ctx, f.tx, pemesan, req))
	if s := f.status(t, pemesan); s.StatusAgen != "aktif" || !s.AkunJamaah {
		t.Fatalf("status %s akunJamaah=%v, want aktif otomatis", s.StatusAgen, s.AkunJamaah)
	}
}

// Percobaan NIK bentrok dibatasi per akun; percobaan lain tidak dihitung.
func TestAjukanBatasNIKBentrok(t *testing.T) {
	h := NewHandler(nil)
	for i := 0; i < nikGagalPerAkunSehari; i++ {
		if !h.nikGagalAkun.Take("77") {
			t.Fatalf("percobaan %d belum boleh ditolak", i+1)
		}
	}
	body := `{"nik":"9999999999993301","domisili":"Kota Bekasi","jenis_kelamin":"L","tanggal_lahir":"1990-05-12","setuju_syarat_ketentuan":true}`
	req := httptest.NewRequest("POST", "/pengajuan", bytes.NewBufferString(body))
	req = req.WithContext(identity.WithCustomer(req.Context(), 77, 1))
	rec := httptest.NewRecorder()
	h.Ajukan(rec, req)
	if rec.Code != 429 {
		t.Fatalf("kode %d (%s), want 429", rec.Code, rec.Body.String())
	}
}
