package manasik

import (
	"context"
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

func TestManasikDanRSVP(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	jadwal, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-02-01"})
	akun := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AkunManasikQzp"})
	lainAkun := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "LainManasikQzp"})
	j1 := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "JamaahSatuQzp"})
	j2 := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "JamaahDuaQzp"})
	asing := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "JamaahAsingQzp"})
	b, _ := testdb.NewBookingPax(t, tx, jadwal, "dp", 1000, testdb.Pax{JamaahID: j1}, testdb.Pax{JamaahID: j2})
	testdb.Exec(t, tx, `UPDATE bookings SET customer_account_id=? WHERE id=?`, akun, b)
	sebelum := time.Date(2098, 12, 31, 12, 0, 0, 0, ZonaWIB)

	// Belum ada manasik: booking tetap tampil dengan jamaahnya, RSVP ditolak.
	v, err := UntukBooking(ctx, tx, akun, b, sebelum)
	if err != nil || v.Manasik != nil || len(v.Jamaah) != 2 || v.BisaRSVP {
		t.Fatalf("tanpa manasik: %+v err=%v", v, err)
	}
	if err := SimpanRSVP(ctx, tx, akun, b, j1, true, sebelum); !errors.Is(err, ErrBelumAda) {
		t.Errorf("RSVP tanpa manasik: %v", err)
	}

	// Brand lain tidak bisa mengisi manasik jadwal ini.
	lain := brand + 99999
	if _, err := Simpan(ctx, tx, jadwal, &lain, 0, inputSah()); !errors.Is(err, ErrJadwalTidakAda) {
		t.Errorf("simpan lintas brand: %v", err)
	}
	m, err := Simpan(ctx, tx, jadwal, &brand, 0, inputSah())
	if err != nil {
		t.Fatal(err)
	}
	if m.JamMulai != "08:00" || len(m.Agenda) != 2 || m.Ustadz == nil || m.GmapsURL == nil {
		t.Errorf("manasik tersimpan: %+v", m)
	}
	// Simpan kedua memperbarui baris yang sama (satu manasik per jadwal).
	ubah := inputSah()
	ubah.Ustadz, ubah.GmapsURL = "", ""
	m2, err := Simpan(ctx, tx, jadwal, &brand, 0, ubah)
	if err != nil || m2.ID != m.ID || m2.Ustadz != nil || m2.GmapsURL != nil {
		t.Fatalf("perbarui manasik: %+v err=%v", m2, err)
	}

	// RSVP: akun pemilik booking mengisi per jamaah.
	if err := SimpanRSVP(ctx, tx, akun, b, j1, true, sebelum); err != nil {
		t.Fatal(err)
	}
	if err := SimpanRSVP(ctx, tx, akun, b, j2, false, sebelum); err != nil {
		t.Fatal(err)
	}
	if err := SimpanRSVP(ctx, tx, akun, b, j2, true, sebelum); err != nil { // jawaban bisa diubah
		t.Fatal(err)
	}
	if err := SimpanRSVP(ctx, tx, akun, b, asing, true, sebelum); !errors.Is(err, ErrBukanJamaah) {
		t.Errorf("jamaah di luar booking: %v", err)
	}
	if err := SimpanRSVP(ctx, tx, lainAkun, b, j1, true, sebelum); !errors.Is(err, ErrTidakAda) {
		t.Errorf("akun lain: %v", err)
	}
	mulai := time.Date(2099, 1, 1, 8, 0, 0, 0, ZonaWIB)
	if err := SimpanRSVP(ctx, tx, akun, b, j1, false, mulai); !errors.Is(err, ErrRSVPDitutup) {
		t.Errorf("RSVP setelah mulai: %v", err)
	}

	v, err = UntukBooking(ctx, tx, akun, b, sebelum)
	if err != nil || v.Manasik == nil || !v.BisaRSVP || len(v.Jamaah) != 2 {
		t.Fatalf("dengan manasik: %+v err=%v", v, err)
	}
	for _, j := range v.Jamaah {
		if j.Hadir == nil || !*j.Hadir {
			t.Errorf("jawaban %s: %v", j.Nama, j.Hadir)
		}
	}
	if v, _ = UntukBooking(ctx, tx, akun, b, mulai); v.BisaRSVP {
		t.Error("RSVP harus ditutup saat manasik dimulai")
	}

	rek, err := RekapJadwal(ctx, tx, m2)
	// The test schedule can be a shared fixture with other bookings: check only this booking's jamaah.
	if err != nil {
		t.Fatal(err)
	}
	if rek.Hadir+rek.TidakHadir+rek.Belum != len(rek.Jamaah) || rek.Hadir < 2 {
		t.Errorf("rekap tidak konsisten: %+v", rek)
	}
	dilihat := 0
	for _, j := range rek.Jamaah {
		if j.JamaahID == j1 || j.JamaahID == j2 {
			dilihat++
			if j.Hadir == nil || !*j.Hadir || j.BookingID != b {
				t.Errorf("rekap jamaah %s: %+v", j.Nama, j)
			}
		}
	}
	if dilihat != 2 {
		t.Errorf("rekap memuat %d dari 2 jamaah booking uji", dilihat)
	}

	// Booking "baru" yang hold seat-nya lewat (kedaluwarsa tanpa DP) tidak lagi bisa membuka manasik.
	testdb.Exec(t, tx, `UPDATE bookings SET status='baru', is_seat_blocked=FALSE WHERE id=?`, b)
	if _, err := UntukBooking(ctx, tx, akun, b, sebelum); !errors.Is(err, ErrTidakAda) {
		t.Errorf("booking kedaluwarsa: %v", err)
	}
	if rek, _ := RekapJadwal(ctx, tx, m2); rek != nil {
		for _, j := range rek.Jamaah {
			if j.BookingID == b {
				t.Errorf("rekap memuat jamaah booking kedaluwarsa: %+v", j)
			}
		}
	}

	// Booking batal tidak lagi bisa membuka manasik.
	testdb.Exec(t, tx, `UPDATE bookings SET status='batal' WHERE id=?`, b)
	if _, err := UntukBooking(ctx, tx, akun, b, sebelum); !errors.Is(err, ErrTidakAda) {
		t.Errorf("booking batal: %v", err)
	}
	if err := Hapus(ctx, tx, jadwal, &brand); err != nil {
		t.Fatal(err)
	}
	if err := Hapus(ctx, tx, jadwal, &brand); !errors.Is(err, ErrTidakAda) {
		t.Errorf("hapus kedua: %v", err)
	}
}
