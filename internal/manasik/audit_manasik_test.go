package manasik

import (
	"context"
	"errors"
	"testing"
	"time"

	"erp-azhan/api/internal/testdb"
)

// Audit Manasik 2026-11-26: M1, M2 (kunci 00.00 hari-H, tanggal minimal hari ini), M4 (paket arsip), M5 (jamaah
// dihitung sekali), M6 (peserta mulai DP), M9 (notifikasi kotak masuk saat dibuat dan tambah paket).
func TestPerbaikanAuditManasik(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	s1, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-02-01"})
	testdb.Exec(t, tx, `UPDATE schedules SET status = 'published' WHERE id = ?`, s1)
	s2 := jadwalLain(t, tx, brand, s1, "2099-02-10")
	testdb.Exec(t, tx, `DELETE FROM manasik_jadwal WHERE schedule_id IN (?, ?)`, s1, s2)

	pemesan := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "PemesanAuditMnsQzp"})
	tertaut := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "TertautAuditMnsQzp"})
	pemesanBaru := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "BaruAuditMnsQzp"})
	pemesanS2 := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "S2AuditMnsQzp"})
	ganda := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "GandaAuditMnsQzp"})
	jBaru := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "BaruJamaahMnsQzp"})
	jS2 := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "S2JamaahMnsQzp"})
	testdb.Exec(t, tx, `UPDATE customer_accounts SET jamaah_id = ? WHERE id = ?`, ganda, tertaut)

	// Jamaah yang sama di dua booking dp paket yang sama; satu booking baru (hold aktif) belum DP.
	bA, paxA := testdb.NewBookingPax(t, tx, s1, "dp", 1000, testdb.Pax{JamaahID: ganda})
	bB, paxB := testdb.NewBookingPax(t, tx, s1, "lunas", 1000, testdb.Pax{JamaahID: ganda})
	bBaru, _ := testdb.NewBookingPax(t, tx, s1, "baru", 1000, testdb.Pax{JamaahID: jBaru})
	bS2, _ := testdb.NewBookingPax(t, tx, s2, "dp", 1000, testdb.Pax{JamaahID: jS2})
	testdb.Exec(t, tx, `UPDATE bookings SET customer_account_id = ? WHERE id IN (?, ?)`, pemesan, bA, bB)
	testdb.Exec(t, tx, `UPDATE bookings SET customer_account_id = ?, seat_hold_expires_at = NULL WHERE id = ?`, pemesanBaru, bBaru)
	testdb.Exec(t, tx, `UPDATE bookings SET customer_account_id = ? WHERE id = ?`, pemesanS2, bS2)

	sekarang := time.Date(2098, 12, 1, 10, 0, 0, 0, ZonaWIB)
	hariHPagi := time.Date(2099, 1, 1, 7, 0, 0, 0, ZonaWIB) // sebelum jam mulai 08.00

	// Tanggal manasik baru minimal hari ini.
	lampau := inputSah()
	lampau.Tanggal = "2098-11-30"
	if _, err := Buat(ctx, tx, &brand, 0, lampau, []int64{s1}, sekarang); !galatValidasi(err) {
		t.Errorf("buat tanggal lampau: %v", err)
	}
	// M4: paket arsip ditolak lewat API.
	testdb.Exec(t, tx, `UPDATE schedules SET status = 'archived' WHERE id = ?`, s2)
	if _, err := Buat(ctx, tx, &brand, 0, inputSah(), []int64{s1, s2}, sekarang); !galatValidasi(err) {
		t.Errorf("paket arsip: %v", err)
	}
	testdb.Exec(t, tx, `UPDATE schedules SET status = 'published' WHERE id = ?`, s2)

	id, err := Buat(ctx, tx, &brand, 0, inputSah(), []int64{s1}, sekarang)
	if err != nil {
		t.Fatal(err)
	}

	// M9: notifikasi kotak masuk ke pemesan dan akun tertaut, sekali per akun; booking belum DP tidak.
	jumlahNotif := func(akun int64, kunci string) int {
		t.Helper()
		var n int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM notifikasi WHERE akun_id = ? AND kunci = ? AND jenis = 'manasik'`,
			akun, kunci).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	kunci := "manasik:" + itoa(id)
	if jumlahNotif(pemesan, kunci) != 1 || jumlahNotif(tertaut, kunci) != 1 || jumlahNotif(pemesanBaru, kunci) != 0 || jumlahNotif(pemesanS2, kunci) != 0 {
		t.Errorf("notifikasi buat: pemesan %d, tertaut %d, belum DP %d, paket lain %d",
			jumlahNotif(pemesan, kunci), jumlahNotif(tertaut, kunci), jumlahNotif(pemesanBaru, kunci), jumlahNotif(pemesanS2, kunci))
	}
	var refID int64
	_ = tx.QueryRowContext(ctx, `SELECT ref_id FROM notifikasi WHERE akun_id = ? AND kunci = ?`, pemesan, kunci).Scan(&refID)
	if refID != bA && refID != bB {
		t.Errorf("ref_id notifikasi pemesan = %d, harus booking pemesan", refID)
	}

	// Ubah tidak mengirim ulang; tambah paket hanya ke jamaah paket baru.
	if err := Ubah(ctx, tx, id, &brand, 0, inputSah(), sekarang); err != nil {
		t.Fatal(err)
	}
	if err := TambahPaket(ctx, tx, id, &brand, []int64{s2}, sekarang); err != nil {
		t.Fatal(err)
	}
	if jumlahNotif(pemesan, kunci) != 1 || jumlahNotif(pemesanS2, kunci) != 1 {
		t.Errorf("notifikasi tambah paket: pemesan %d, paket baru %d", jumlahNotif(pemesan, kunci), jumlahNotif(pemesanS2, kunci))
	}

	// M5: jamaah dengan dua booking dihitung sekali di paket, daftar, dan rekap.
	d, err := AmbilDetail(ctx, tx, id, &brand, sekarang)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range d.Paket {
		if p.ScheduleID == s1 && p.JumlahJamaah != 1 {
			t.Errorf("jumlah jamaah paket s1 = %d, harus 1", p.JumlahJamaah)
		}
	}
	if d.Rekap.Jamaah != 2 || len(d.Jamaah) != 2 {
		t.Errorf("rekap jamaah = %d, daftar = %d, harus 2", d.Rekap.Jamaah, len(d.Jamaah))
	}
	// M6: booking belum DP tidak ikut daftar, tidak bisa RSVP, dan portal menampilkan tampilan kosong.
	if cariJamaah(d, jBaru) != nil {
		t.Error("jamaah booking belum DP masih tercantum")
	}
	v, err := UntukBooking(ctx, tx, pemesanBaru, bBaru, sekarang)
	if err != nil || v.Manasik != nil || v.Jamaah == nil || len(v.Jamaah) != 0 || v.BisaRSVP {
		t.Errorf("portal booking belum DP: %+v err=%v", v, err)
	}
	if err := SimpanRSVP(ctx, tx, pemesanBaru, bBaru, jBaru, true, sekarang); !errors.Is(err, ErrBelumDP) {
		t.Errorf("RSVP belum DP: %v", err)
	}
	daftar, err := Daftar(ctx, tx, &brand, false, sekarang)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range daftar {
		if r.ID == id && r.JumlahJamaah != 2 {
			t.Errorf("daftar: jumlah jamaah = %d, harus 2", r.JumlahJamaah)
		}
	}

	// M1: mulai 00.00 hari manasik, paket terkunci walau jam mulai belum tiba.
	if err := LepasPaket(ctx, tx, id, &brand, s2, hariHPagi); !errors.Is(err, ErrSudahDimulai) {
		t.Errorf("lepas di hari-H sebelum jam mulai: %v", err)
	}
	if err := TambahPaket(ctx, tx, id, &brand, []int64{s2}, hariHPagi); !errors.Is(err, ErrSudahDimulai) {
		t.Errorf("tambah di hari-H sebelum jam mulai: %v", err)
	}
	if d, _ := AmbilDetail(ctx, tx, id, &brand, hariHPagi); !d.SudahDimulai {
		t.Error("sudah_dimulai harus true sejak 00.00 hari manasik")
	}

	// M2: di hari-H tanggal dan jam terkunci; judul, tempat, alamat, agenda tetap bisa diubah.
	geser := inputSah()
	geser.Tanggal = "2099-01-05"
	if err := Ubah(ctx, tx, id, &brand, 0, geser, hariHPagi); !errors.Is(err, ErrJadwalTerkunci) {
		t.Errorf("ubah tanggal di hari-H: %v", err)
	}
	jamLain := inputSah()
	jamLain.JamMulai = "09:00"
	if err := Ubah(ctx, tx, id, &brand, 0, jamLain, time.Date(2099, 1, 3, 9, 0, 0, 0, ZonaWIB)); !errors.Is(err, ErrJadwalTerkunci) {
		t.Errorf("ubah jam setelah selesai: %v", err)
	}
	isi := inputSah()
	isi.Judul, isi.Tempat, isi.Alamat, isi.Agenda = "Manasik Revisi", "Aula Baru", "Jl. Baru No. 2", []string{"08.00 Pembukaan"}
	if err := Ubah(ctx, tx, id, &brand, 0, isi, hariHPagi); err != nil {
		t.Errorf("ubah isi di hari-H: %v", err)
	}
	// Sebelum hari-H, tanggal tidak boleh dipindah ke masa lalu.
	mundur := inputSah()
	mundur.Tanggal = "2098-11-15"
	if err := Ubah(ctx, tx, id, &brand, 0, mundur, sekarang); !galatValidasi(err) {
		t.Errorf("ubah ke tanggal lampau: %v", err)
	}

	// M5: absensi per orang mengisi penanda semua pax jamaah itu di paket manasik.
	if err := CatatHadir(ctx, tx, id, &brand, 0, paxA[0], true, hariHPagi); err != nil {
		t.Fatal(err)
	}
	var hadirB bool
	if err := tx.QueryRowContext(ctx, `SELECT progress_manasik FROM booking_pax WHERE id = ?`, paxB[0]).Scan(&hadirB); err != nil {
		t.Fatal(err)
	}
	if !hadirB {
		t.Error("absensi harus mengisi penanda booking kedua jamaah yang sama")
	}
	if d, _ := AmbilDetail(ctx, tx, id, &brand, hariHPagi); d.Rekap.Hadir != 1 {
		t.Errorf("rekap hadir = %d, harus 1", d.Rekap.Hadir)
	}
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

// M11: tanggal berangkat paket bermanasik tidak boleh sama dengan atau sebelum tanggal manasik.
func TestBentrokBerangkat(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	s1, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-02-01"})
	testdb.Exec(t, tx, `UPDATE schedules SET status = 'published' WHERE id = ?`, s1)
	testdb.Exec(t, tx, `DELETE FROM manasik_jadwal WHERE schedule_id = ?`, s1)
	if tgl, err := BentrokBerangkat(ctx, tx, s1, "2098-01-01"); err != nil || tgl != "" {
		t.Errorf("paket tanpa manasik: %q %v", tgl, err)
	}
	if _, err := Buat(ctx, tx, &brand, 0, inputSah(), []int64{s1}, time.Date(2098, 12, 1, 10, 0, 0, 0, ZonaWIB)); err != nil {
		t.Fatal(err)
	}
	for berangkat, bentrok := range map[string]bool{"2098-12-31": true, "2099-01-01": true, "2099-01-02": false} {
		tgl, err := BentrokBerangkat(ctx, tx, s1, berangkat)
		if err != nil || (tgl != "") != bentrok {
			t.Errorf("berangkat %s: tanggal manasik %q err=%v, bentrok harus %v", berangkat, tgl, err, bentrok)
		}
	}
}
