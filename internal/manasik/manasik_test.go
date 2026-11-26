package manasik

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"erp-azhan/api/internal/testdb"
)

func inputSah() Input {
	return Input{
		Judul: "Manasik Umroh Lengkap", Tanggal: "2099-01-01", JamMulai: "08:00", JamSelesai: "15:00",
		Agenda: []string{"08.00 Pembukaan", "  ", "09.00 Praktik thawaf"}, Ustadz: "Ust. Ahmad",
		Tempat: "Aula Kantor", Alamat: "Jl. Contoh No. 1, Jakarta", GmapsURL: "https://maps.app.goo.gl/abc",
	}
}

func TestValidasiInput(t *testing.T) {
	in := inputSah()
	in.rapikan()
	if msg := in.validasi(); msg != "" {
		t.Fatalf("input sah ditolak: %s", msg)
	}
	if len(in.Agenda) != 2 {
		t.Errorf("butir agenda kosong harus dibuang: %v", in.Agenda)
	}
	for nama, ubah := range map[string]func(*Input){
		"judul kosong":     func(i *Input) { i.Judul = " " },
		"tanggal salah":    func(i *Input) { i.Tanggal = "01-01-2099" },
		"jam salah":        func(i *Input) { i.JamMulai = "8:00" },
		"selesai sebelum":  func(i *Input) { i.JamSelesai = "07:00" },
		"tempat kosong":    func(i *Input) { i.Tempat = "" },
		"alamat kosong":    func(i *Input) { i.Alamat = "" },
		"link bukan https": func(i *Input) { i.GmapsURL = "http://maps.google.com" },
		"link javascript":  func(i *Input) { i.GmapsURL = "javascript:alert(1)" },
		"agenda terlalu banyak": func(i *Input) {
			i.Agenda = make([]string, maksAgenda+1)
			for n := range i.Agenda {
				i.Agenda[n] = "butir"
			}
		},
	} {
		in := inputSah()
		ubah(&in)
		in.rapikan()
		if in.validasi() == "" {
			t.Errorf("%s: harus ditolak", nama)
		}
	}
}

// jadwalLain memakai paket lain milik brand yang sama, lalu mengatur tanggal berangkatnya.
func jadwalLain(t *testing.T, tx *sql.Tx, brand, kecuali int64, berangkat string) int64 {
	t.Helper()
	var id int64
	if err := tx.QueryRowContext(context.Background(), `SELECT id FROM schedules WHERE brand_id = ? AND id <> ? ORDER BY id LIMIT 1`,
		brand, kecuali).Scan(&id); err != nil {
		t.Skipf("butuh dua paket satu brand di DB uji: %v", err)
	}
	testdb.Exec(t, tx, `UPDATE schedules SET berangkat_tanggal = ?, status = 'published' WHERE id = ?`, berangkat, id)
	return id
}

func galatValidasi(err error) bool {
	var v *ErrValidasi
	return errors.As(err, &v)
}

func cariJamaah(d *Detail, jamaahID int64) *JamaahManasik {
	for i := range d.Jamaah {
		if d.Jamaah[i].JamaahID == jamaahID {
			return &d.Jamaah[i]
		}
	}
	return nil
}

func TestManasikPaketRSVPDanAbsensi(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	s1, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-02-01"})
	testdb.Exec(t, tx, `UPDATE schedules SET status = 'published' WHERE id = ?`, s1)
	s2 := jadwalLain(t, tx, brand, s1, "2099-02-10")
	// Manasik lain di DB lokal tidak boleh memengaruhi uji ini (dibatalkan bersama transaksi).
	testdb.Exec(t, tx, `DELETE FROM manasik_jadwal WHERE schedule_id IN (?, ?)`, s1, s2)

	akun := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AkunManasikQzp"})
	lainAkun := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "LainManasikQzp"})
	j1 := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "JamaahSatuQzp"})
	j2 := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "JamaahDuaQzp"})
	bayi := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "JamaahBayiQzp"})
	j3 := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "JamaahTigaQzp"})
	asing := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "JamaahAsingQzp"})
	b1, pax1 := testdb.NewBookingPax(t, tx, s1, "dp", 1000, testdb.Pax{JamaahID: j1}, testdb.Pax{JamaahID: j2}, testdb.Pax{JamaahID: bayi})
	testdb.Exec(t, tx, `UPDATE booking_pax SET pax_type = 'infant' WHERE id = ?`, pax1[2])
	b2, _ := testdb.NewBookingPax(t, tx, s2, "dp", 1000, testdb.Pax{JamaahID: j3})
	testdb.Exec(t, tx, `UPDATE bookings SET customer_account_id = ? WHERE id IN (?, ?)`, akun, b1, b2)

	sebelum := time.Date(2098, 12, 1, 10, 0, 0, 0, ZonaWIB)
	hariH := time.Date(2099, 1, 1, 9, 0, 0, 0, ZonaWIB)

	// Belum ada manasik: booking tampil dengan jamaah dewasanya saja, RSVP ditolak.
	v, err := UntukBooking(ctx, tx, akun, b1, sebelum)
	if err != nil || v.Manasik != nil || len(v.Jamaah) != 2 || v.BisaRSVP {
		t.Fatalf("tanpa manasik: %+v err=%v", v, err)
	}
	if err := SimpanRSVP(ctx, tx, akun, b1, j1, true, sebelum); !errors.Is(err, ErrBelumAda) {
		t.Errorf("RSVP tanpa manasik: %v", err)
	}

	// Validasi pembuatan.
	lain := brand + 99999
	if _, err := Buat(ctx, tx, &lain, 0, inputSah(), []int64{s1}, sebelum); !galatValidasi(err) {
		t.Errorf("paket brand lain: %v", err)
	}
	if _, err := Buat(ctx, tx, &brand, 0, inputSah(), nil, sebelum); !galatValidasi(err) {
		t.Errorf("tanpa paket: %v", err)
	}
	telat := inputSah()
	telat.Tanggal = "2099-02-01"
	if _, err := Buat(ctx, tx, &brand, 0, telat, []int64{s1, s2}, sebelum); !galatValidasi(err) {
		t.Errorf("tanggal manasik sama dengan keberangkatan: %v", err)
	}

	id, err := Buat(ctx, tx, &brand, 0, inputSah(), []int64{s1, s2}, sebelum)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Buat(ctx, tx, &brand, 0, inputSah(), []int64{s1}, sebelum); !galatValidasi(err) {
		t.Errorf("paket yang sudah punya manasik: %v", err)
	}
	// Kaitan ganda di manasik_jadwal ditolak, bukan dilewati diam-diam.
	if err := kaitkan(ctx, tx, id, []int64{s1}); !errors.Is(err, ErrPaketSudahBermanasik) {
		t.Errorf("kaitkan paket yang sudah punya manasik: %v", err)
	}
	tersedia, err := PaketTersedia(ctx, tx, &brand, sebelum)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range tersedia {
		if p.ScheduleID == s1 || p.ScheduleID == s2 {
			t.Errorf("paket bermanasik masih tersedia: %+v", p)
		}
	}
	if ring, err := UntukPaket(ctx, tx, s2, &brand); err != nil || ring == nil || ring.ID != id {
		t.Errorf("ringkasan manasik paket: %+v err=%v", ring, err)
	}

	d, err := AmbilDetail(ctx, tx, id, &brand, sebelum)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Paket) != 2 || d.SudahDimulai || d.AbsensiTerbuka || d.AdaAbsensi || d.Manasik.BrandID != brand {
		t.Errorf("detail awal: paket=%d dimulai=%v absensi=%v", len(d.Paket), d.SudahDimulai, d.AbsensiTerbuka)
	}
	if cariJamaah(d, j1) == nil || cariJamaah(d, j3) == nil || cariJamaah(d, bayi) != nil {
		t.Errorf("daftar jamaah harus memuat j1 dan j3 tanpa bayi")
	}

	// RSVP per jamaah dewasa.
	if err := SimpanRSVP(ctx, tx, akun, b1, j1, true, sebelum); err != nil {
		t.Fatal(err)
	}
	if err := SimpanRSVP(ctx, tx, akun, b1, j2, false, sebelum); err != nil {
		t.Fatal(err)
	}
	if err := SimpanRSVP(ctx, tx, akun, b2, j3, true, sebelum); err != nil {
		t.Fatal(err)
	}
	if err := SimpanRSVP(ctx, tx, akun, b1, bayi, true, sebelum); !errors.Is(err, ErrBukanJamaah) {
		t.Errorf("RSVP bayi: %v", err)
	}
	if err := SimpanRSVP(ctx, tx, akun, b1, asing, true, sebelum); !errors.Is(err, ErrBukanJamaah) {
		t.Errorf("jamaah di luar booking: %v", err)
	}
	if err := SimpanRSVP(ctx, tx, lainAkun, b1, j1, true, sebelum); !errors.Is(err, ErrTidakAda) {
		t.Errorf("akun lain: %v", err)
	}
	mulai := time.Date(2099, 1, 1, 8, 0, 0, 0, ZonaWIB)
	if err := SimpanRSVP(ctx, tx, akun, b1, j1, false, mulai); !errors.Is(err, ErrRSVPDitutup) {
		t.Errorf("RSVP setelah mulai: %v", err)
	}
	v, err = UntukBooking(ctx, tx, akun, b1, sebelum)
	if err != nil || v.Manasik == nil || v.Manasik.ID != id || !v.BisaRSVP || len(v.Jamaah) != 2 {
		t.Fatalf("dengan manasik: %+v err=%v", v, err)
	}

	// Ubah: tanggal tidak boleh menyamai keberangkatan paling awal; jawaban tetap tersimpan.
	geser := inputSah()
	geser.Tanggal = "2099-02-05"
	if err := Ubah(ctx, tx, id, &brand, 0, geser); !galatValidasi(err) {
		t.Errorf("ubah tanggal lewat keberangkatan: %v", err)
	}
	ubah := inputSah()
	ubah.Ustadz, ubah.GmapsURL = "", ""
	if err := Ubah(ctx, tx, id, &brand, 0, ubah); err != nil {
		t.Fatal(err)
	}
	if d, _ = AmbilDetail(ctx, tx, id, &brand, sebelum); d.Manasik.Ustadz != nil || cariJamaah(d, j1).Jawaban == nil {
		t.Errorf("setelah ubah: ustadz=%v jawaban=%v", d.Manasik.Ustadz, cariJamaah(d, j1).Jawaban)
	}

	// Lepas paket: jawaban jamaah paket itu dihapus; paket terakhir tidak bisa dilepas.
	if err := LepasPaket(ctx, tx, id, &brand, s2, sebelum); err != nil {
		t.Fatal(err)
	}
	var sisa int
	_ = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM manasik_rsvp WHERE manasik_id = ? AND jamaah_id = ?`, id, j3).Scan(&sisa)
	if sisa != 0 {
		t.Error("jawaban jamaah paket yang dilepas harus dihapus")
	}
	if err := LepasPaket(ctx, tx, id, &brand, s1, sebelum); !galatValidasi(err) {
		t.Errorf("lepas paket terakhir: %v", err)
	}
	if err := TambahPaket(ctx, tx, id, &brand, []int64{s2}, sebelum); err != nil {
		t.Fatal(err)
	}

	// Absensi baru terbuka di hari manasik dan mengisi penanda progres.
	if err := CatatHadir(ctx, tx, id, &brand, 0, pax1[0], true, sebelum); !errors.Is(err, ErrAbsensiBelumDibuka) {
		t.Errorf("absensi sebelum hari-H: %v", err)
	}
	if err := CatatHadir(ctx, tx, id, &brand, 0, pax1[2], true, hariH); !galatValidasi(err) {
		t.Errorf("absensi bayi: %v", err)
	}
	if err := CatatHadir(ctx, tx, id, &brand, 0, pax1[0], true, hariH); err != nil {
		t.Fatal(err)
	}
	if err := CatatHadir(ctx, tx, id, &brand, 0, pax1[0], true, hariH); err != nil { // nilai sama tetap berhasil
		t.Errorf("absensi ulang: %v", err)
	}
	d, _ = AmbilDetail(ctx, tx, id, &brand, hariH)
	if it := cariJamaah(d, j1); it == nil || !it.Hadir || it.DicatatAt == nil || !d.AbsensiTerbuka || !d.SudahDimulai {
		t.Errorf("setelah absensi: %+v", it)
	}
	if d.Rekap.Hadir < 1 || d.Rekap.BerniatHadir+d.Rekap.TidakHadir+d.Rekap.Belum != d.Rekap.Jamaah {
		t.Errorf("rekap tidak konsisten: %+v", d.Rekap)
	}

	// Setelah dimulai paket terkunci, dan manasik dengan absensi tidak bisa dihapus.
	if err := LepasPaket(ctx, tx, id, &brand, s2, hariH); !errors.Is(err, ErrSudahDimulai) {
		t.Errorf("lepas setelah mulai: %v", err)
	}
	if err := TambahPaket(ctx, tx, id, &brand, []int64{s2}, hariH); !errors.Is(err, ErrSudahDimulai) {
		t.Errorf("tambah setelah mulai: %v", err)
	}
	if err := Hapus(ctx, tx, id, &brand, hariH); !errors.Is(err, ErrAdaAbsensi) {
		t.Errorf("hapus dengan absensi: %v", err)
	}
	// Penanda bisa diisi dari Detail Booking sebelum hari manasik: hapus tetap ditolak.
	if err := Hapus(ctx, tx, id, &brand, sebelum); !errors.Is(err, ErrAdaAbsensi) {
		t.Errorf("hapus dengan absensi sebelum hari-H: %v", err)
	}
	if d, _ = AmbilDetail(ctx, tx, id, &brand, sebelum); !d.AdaAbsensi {
		t.Error("ada_absensi harus true")
	}

	// Booking kedaluwarsa atau batal tidak lagi membuka manasik dan keluar dari daftar jamaah.
	testdb.Exec(t, tx, `UPDATE bookings SET status = 'baru', is_seat_blocked = FALSE WHERE id = ?`, b2)
	if _, err := UntukBooking(ctx, tx, akun, b2, sebelum); !errors.Is(err, ErrTidakAda) {
		t.Errorf("booking kedaluwarsa: %v", err)
	}
	if d, _ = AmbilDetail(ctx, tx, id, &brand, sebelum); cariJamaah(d, j3) != nil {
		t.Error("jamaah booking kedaluwarsa masih tercantum")
	}

	if err := CatatHadir(ctx, tx, id, &brand, 0, pax1[0], false, hariH); err != nil {
		t.Fatal(err)
	}
	testdb.Exec(t, tx, `UPDATE booking_pax bp JOIN bookings b ON b.id = bp.booking_id SET bp.progress_manasik = FALSE
		WHERE b.schedule_id IN (?, ?)`, s1, s2)
	if d, _ = AmbilDetail(ctx, tx, id, &brand, hariH); d.AdaAbsensi {
		t.Error("ada_absensi harus false setelah penanda dikosongkan")
	}
	if err := Hapus(ctx, tx, id, &lain, hariH); !errors.Is(err, ErrTidakAda) {
		t.Errorf("hapus lintas brand: %v", err)
	}
	if err := Hapus(ctx, tx, id, &brand, hariH); err != nil {
		t.Fatal(err)
	}
	if _, err := AmbilDetail(ctx, tx, id, &brand, hariH); !errors.Is(err, ErrTidakAda) {
		t.Errorf("setelah hapus: %v", err)
	}
	if ring, _ := UntukPaket(ctx, tx, s1, &brand); ring != nil {
		t.Error("paket masih terkait setelah manasik dihapus")
	}
}

// Pax paket brand lain tidak terbaca dan tidak bisa diabsen lewat manasik, walau paketnya
// tertaut ke manasik itu (data lama sebelum brand paket dikunci).
func TestManasikAbaikanPaketBrandLain(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	s1, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-02-01"})
	testdb.Exec(t, tx, `DELETE FROM manasik_jadwal WHERE schedule_id = ?`, s1)
	var brandLain int64
	if err := tx.QueryRow(`SELECT id FROM brands WHERE id <> ? ORDER BY id LIMIT 1`, brand).Scan(&brandLain); err != nil {
		t.Skipf("butuh dua brand: %v", err)
	}
	sebelum := time.Date(2098, 12, 1, 10, 0, 0, 0, ZonaWIB)
	hariH := time.Date(2099, 1, 1, 9, 0, 0, 0, ZonaWIB)
	id, err := Buat(ctx, tx, &brand, 0, inputSah(), []int64{s1}, sebelum)
	if err != nil {
		t.Fatal(err)
	}
	testdb.Exec(t, tx, `UPDATE schedules SET brand_id = ? WHERE id = ?`, brandLain, s1)
	j := testdb.NewJamaah(t, tx, brandLain, testdb.JamaahOpts{Nama: "JamaahBrandLainQzp"})
	_, pax := testdb.NewBookingPax(t, tx, s1, "dp", 1000, testdb.Pax{JamaahID: j})

	d, err := AmbilDetail(ctx, tx, id, &brand, sebelum)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Jamaah) != 0 || len(d.Paket) != 0 {
		t.Fatalf("pax/paket brand lain terbaca: jamaah=%d paket=%d", len(d.Jamaah), len(d.Paket))
	}
	if err := CatatHadir(ctx, tx, id, &brand, 0, pax[0], true, hariH); !galatValidasi(err) {
		t.Fatalf("absen pax brand lain: err=%v, mau galat validasi", err)
	}
}
