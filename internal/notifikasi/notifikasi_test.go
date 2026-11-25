package notifikasi

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"erp-azhan/api/internal/testdb"
)

func TestRupiahDanGabung(t *testing.T) {
	if got := Rupiah(5000000); got != "Rp5.000.000" {
		t.Errorf("Rupiah: %q", got)
	}
	if got := Rupiah(500); got != "Rp500" {
		t.Errorf("Rupiah kecil: %q", got)
	}
	if got := Gabung("Rp5.000.000", " ", "Umroh"); got != "Rp5.000.000 · Umroh" {
		t.Errorf("Gabung: %q", got)
	}
	if got := TanggalTeks("2027-03-12"); got != "12 Mar 2027" {
		t.Errorf("TanggalTeks: %q", got)
	}
}

func daftarJenis(t *testing.T, q Querier, akun int64) map[string][]Item {
	t.Helper()
	items, err := Daftar(context.Background(), q, akun, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	out := map[string][]Item{}
	for _, it := range items {
		out[it.Jenis] = append(out[it.Jenis], it)
	}
	return out
}

func TestKotakMasukDanEvent(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	now := time.Now()
	tglBerangkat := now.In(ZonaWIB).AddDate(0, 0, 7).Format("2006-01-02")
	jadwal, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: tglBerangkat})

	agen := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenNotifQzp", StatusAgen: "aktif"})
	pemilik := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "PemilikNotifQzp"})
	j := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "JamaahNotifQzp", Direkrut: agen})
	b, _ := testdb.NewBookingPax(t, tx, jadwal, "dp", 10000000, testdb.Pax{JamaahID: j})
	testdb.Exec(t, tx, `UPDATE bookings SET customer_account_id = ? WHERE id = ?`, pemilik, b)

	// Kirim dengan kunci yang sama hanya sekali.
	for i := 0; i < 2; i++ {
		if err := Kirim(ctx, tx, pemilik, Notif{Jenis: JenisPengumuman, Judul: "Uji", Kunci: "uji:1"}); err != nil {
			t.Fatal(err)
		}
	}
	if n := len(daftarJenis(t, tx, pemilik)[JenisPengumuman]); n != 1 {
		t.Errorf("kunci dobel: %d", n)
	}
	// Akun 0 diabaikan tanpa galat.
	if err := Kirim(ctx, tx, 0, Notif{Jenis: JenisPengumuman, Judul: "x"}); err != nil {
		t.Errorf("akun 0: %v", err)
	}

	Pembayaran(ctx, tx, b, true, 5000000)
	StatusBooking(ctx, tx, b, "baru", "dp")
	StatusBooking(ctx, tx, b, "baru", "dp") // referral tidak dobel
	StatusBooking(ctx, tx, b, "dp", "lunas")

	m := daftarJenis(t, tx, pemilik)
	if len(m[JenisPembayaranDiterima]) != 1 || m[JenisPembayaranDiterima][0].Isi == "" {
		t.Errorf("pembayaran: %+v", m[JenisPembayaranDiterima])
	}
	if len(m[JenisBookingLunas]) != 1 {
		t.Errorf("lunas: %+v", m[JenisBookingLunas])
	}
	ma := daftarJenis(t, tx, agen)
	if len(ma[JenisJamaahReferral]) != 1 {
		t.Errorf("jamaah referral agen: %+v", ma[JenisJamaahReferral])
	}

	// Dokumen: pemilik booking aktif menerima.
	Dokumen(ctx, tx, j, "paspor", false)
	if d := daftarJenis(t, tx, pemilik)[JenisDokumenDitolak]; len(d) != 1 || d[0].Isi == "" {
		t.Errorf("dokumen ditolak: %+v", d)
	}

	// Pengingat berangkat H-7 sekali walau putaran dua kali.
	for i := 0; i < 2; i++ {
		if err := berangkat(ctx, tx, now.In(ZonaWIB).Format("2006-01-02")); err != nil {
			t.Fatal(err)
		}
	}
	if d := daftarJenis(t, tx, pemilik)[JenisBerangkat]; len(d) != 1 || d[0].Judul != "Berangkat 7 hari lagi" {
		t.Errorf("berangkat H-7: %+v", d)
	}

	// Hold seat: booking baru dengan hold berakhir 2 jam lagi.
	b2, _ := testdb.NewBookingPax(t, tx, jadwal, "baru", 10000000, testdb.Pax{JamaahID: j})
	testdb.Exec(t, tx, `UPDATE bookings SET customer_account_id = ?, seat_hold_expires_at = ? WHERE id = ?`, pemilik, now.Add(2*time.Hour), b2)
	for i := 0; i < 2; i++ {
		if err := holdSeat(ctx, tx, now); err != nil {
			t.Fatal(err)
		}
	}
	if d := daftarJenis(t, tx, pemilik)[JenisHoldSeat]; len(d) != 1 {
		t.Errorf("hold seat: %+v", d)
	}

	// Baca satu, lalu semua.
	belum, _ := JumlahBelumDibaca(ctx, tx, pemilik, now)
	if belum < 5 {
		t.Fatalf("belum dibaca: %d", belum)
	}
	satu := daftarJenis(t, tx, pemilik)[JenisBookingLunas][0]
	if err := TandaiDibaca(ctx, tx, pemilik, satu.ID); err != nil {
		t.Fatal(err)
	}
	if err := TandaiDibaca(ctx, tx, agen, satu.ID); err != ErrTidakAda {
		t.Errorf("baca milik akun lain: %v", err)
	}
	if sisa, _ := JumlahBelumDibaca(ctx, tx, pemilik, now); sisa != belum-1 {
		t.Errorf("setelah baca satu: %d", sisa)
	}
	if err := TandaiSemuaDibaca(ctx, tx, pemilik); err != nil {
		t.Fatal(err)
	}
	if sisa, _ := JumlahBelumDibaca(ctx, tx, pemilik, now); sisa != 0 {
		t.Errorf("setelah baca semua: %d", sisa)
	}

	// Notifikasi lebih dari 90 hari tidak tampil dan dibersihkan.
	testdb.Exec(t, tx, `UPDATE notifikasi SET dibuat_at = ? WHERE id = ?`, now.AddDate(0, 0, -91), satu.ID)
	if err := Bersihkan(ctx, tx, now); err != nil {
		t.Fatal(err)
	}
	var ada int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM notifikasi WHERE id = ?`, satu.ID).Scan(&ada); err != nil || ada != 0 {
		t.Errorf("bersihkan: %d %v", ada, err)
	}
}

func TestPengumumanKeKotakMasuk(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	now := time.Now()
	hari := now.In(ZonaWIB).Format("2006-01-02")
	jadwal, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2030-01-10"})
	akunJadwal := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AkunJadwalQzp"})
	akunLain := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AkunLainQzp"})
	j := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "JamaahPengumumanQzp"})
	b, _ := testdb.NewBookingPax(t, tx, jadwal, "dp", 1000, testdb.Pax{JamaahID: j})
	testdb.Exec(t, tx, `UPDATE bookings SET customer_account_id = ? WHERE id = ?`, akunJadwal, b)

	res, err := tx.ExecContext(ctx, `INSERT INTO pengumuman (brand_id, schedule_id, judul, isi, mulai_tayang, akhir_tayang)
		VALUES (?, ?, 'Manasik Qzp', 'Isi panjang\npengumuman', ?, ?)`, brand, jadwal, hari, hari)
	if err != nil {
		t.Fatal(err)
	}
	pid, _ := res.LastInsertId()
	for i := 0; i < 2; i++ {
		if err := KirimPengumuman(ctx, tx, now); err != nil {
			t.Fatal(err)
		}
	}
	if d := daftarJenis(t, tx, akunJadwal)[JenisPengumuman]; len(d) != 1 || d[0].Judul != "Manasik Qzp" || d[0].Isi != "Isi panjang pengumuman" {
		t.Errorf("pengumuman jadwal: %+v", d)
	}
	if d := daftarJenis(t, tx, akunLain)[JenisPengumuman]; len(d) != 0 {
		t.Errorf("akun di luar jadwal menerima: %+v", d)
	}
	// Judul terbaru tampil setelah diubah.
	testdb.Exec(t, tx, `UPDATE pengumuman SET judul = 'Manasik Diubah Qzp' WHERE id = ?`, pid)
	if d := daftarJenis(t, tx, akunJadwal)[JenisPengumuman]; len(d) != 1 || d[0].Judul != "Manasik Diubah Qzp" {
		t.Errorf("judul terbaru: %+v", d)
	}
	if p, err := AmbilPengumuman(ctx, tx, akunJadwal, pid); err != nil || p.Isi == "" {
		t.Errorf("detail pengumuman: %+v %v", p, err)
	}
	if _, err := AmbilPengumuman(ctx, tx, akunLain, pid); err != ErrTidakAda {
		t.Errorf("detail untuk akun lain: %v", err)
	}
	// Dihapus: notifikasi ikut hilang.
	testdb.Exec(t, tx, `DELETE FROM pengumuman WHERE id = ?`, pid)
	if d := daftarJenis(t, tx, akunJadwal)[JenisPengumuman]; len(d) != 0 {
		t.Errorf("notifikasi pengumuman terhapus masih tampil: %+v", d)
	}
}

var _ Querier = (*sql.Tx)(nil)

func TestDokumenAkunTertaut(t *testing.T) {
	_, tx := testdb.Tx(t)
	jadwal, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-03-01"})
	ibrahim := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Ibrahim"})
	tertaut := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Ahmad"})
	lain := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Lain"})
	j := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Ahmad"})
	testdb.Exec(t, tx, `UPDATE jamaah SET dibuat_oleh_akun_id=? WHERE id=?`, ibrahim, j)
	testdb.Exec(t, tx, `UPDATE customer_accounts SET jamaah_id=? WHERE id=?`, j, tertaut)
	b, _ := testdb.NewBookingPax(t, tx, jadwal, "dp", 1000000, testdb.Pax{JamaahID: j})
	testdb.Exec(t, tx, `UPDATE bookings SET customer_account_id=? WHERE id=?`, ibrahim, b)

	Dokumen(context.Background(), tx, j, "ktp", true)
	if d := daftarJenis(t, tx, tertaut)[JenisDokumenDiverifikasi]; len(d) != 1 || d[0].Isi == "" {
		t.Errorf("akun tertaut: %+v", d)
	}
	if d := daftarJenis(t, tx, ibrahim)[JenisDokumenDiverifikasi]; len(d) != 1 {
		t.Errorf("pemilik tetap menerima satu: %+v", d)
	}
	if d := daftarJenis(t, tx, lain)[JenisDokumenDiverifikasi]; len(d) != 0 {
		t.Errorf("akun tak tertaut tak berubah: %+v", d)
	}

	// Akun tertaut = pemilik: tidak dobel.
	testdb.Exec(t, tx, `UPDATE customer_accounts SET jamaah_id=NULL WHERE id=?`, tertaut)
	testdb.Exec(t, tx, `UPDATE customer_accounts SET jamaah_id=? WHERE id=?`, j, ibrahim)
	testdb.Exec(t, tx, `DELETE FROM notifikasi WHERE akun_id IN (?,?)`, ibrahim, tertaut)
	Dokumen(context.Background(), tx, j, "paspor", false)
	if d := daftarJenis(t, tx, ibrahim)[JenisDokumenDitolak]; len(d) != 1 {
		t.Errorf("tidak boleh dobel: %+v", d)
	}
	if d := daftarJenis(t, tx, tertaut)[JenisDokumenDitolak]; len(d) != 0 {
		t.Errorf("lepas tautan: %+v", d)
	}

	// Booking batal: akun tertaut tak menerima.
	testdb.Exec(t, tx, `UPDATE customer_accounts SET jamaah_id=NULL WHERE id=?`, ibrahim)
	testdb.Exec(t, tx, `UPDATE customer_accounts SET jamaah_id=? WHERE id=?`, j, tertaut)
	testdb.Exec(t, tx, `UPDATE bookings SET status='batal' WHERE id=?`, b)
	testdb.Exec(t, tx, `DELETE FROM notifikasi WHERE akun_id IN (?,?)`, ibrahim, tertaut)
	Dokumen(context.Background(), tx, j, "ktp", true)
	if d := daftarJenis(t, tx, tertaut)[JenisDokumenDiverifikasi]; len(d) != 0 {
		t.Errorf("booking batal: %+v", d)
	}

	// Booking baru yang tahannya lewat: tak tampil di kartu, tak menerima.
	testdb.Exec(t, tx, `UPDATE bookings SET status='baru', is_seat_blocked=1, seat_hold_expires_at=DATE_SUB(NOW(), INTERVAL 1 HOUR) WHERE id=?`, b)
	Dokumen(context.Background(), tx, j, "ktp", true)
	if d := daftarJenis(t, tx, tertaut)[JenisDokumenDiverifikasi]; len(d) != 0 {
		t.Errorf("hold kedaluwarsa: %+v", d)
	}
}

// Komisi tersedia (H+3) dikabarkan juga bila booking dibatalkan setelah lunas (K20: komisinya tetap di saldo).
// Komisi yang sudah dibatalkan (booking turun dari lunas) tidak dikabarkan.
func TestKomisiTersediaBookingBatalSetelahLunas(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	hari := time.Now().In(ZonaWIB).Format("2006-01-02")
	berangkat := time.Now().In(ZonaWIB).AddDate(0, 0, -3).Format("2006-01-02")
	jadwal, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: berangkat})
	agen := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenKomisiTersediaQzp", StatusAgen: "aktif"})

	komisi := func(status string, batal bool) {
		j := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "JamaahKomisi" + status + "Qzp", Direkrut: agen})
		b, pax := testdb.NewBookingPax(t, tx, jadwal, status, 1000, testdb.Pax{JamaahID: j})
		dibatalkan := any(nil)
		if batal {
			dibatalkan = time.Now()
		}
		testdb.Exec(t, tx, `INSERT INTO transaksi_komisi (brand_id, akun_penerima_id, jamaah_sumber_id, booking_id, booking_pax_id, jenis, nominal, dibatalkan_at)
			VALUES (?,?,?,?,?,'langsung',500000,?)`, brand, agen, j, b, pax[0], dibatalkan)
	}
	komisi("lunas", false)
	komisi("batal", false) // dibatalkan setelah lunas: komisi tetap
	komisi("dp", true)     // turun dari lunas: komisi batal

	for i := 0; i < 2; i++ {
		if err := komisiTersedia(ctx, tx, hari); err != nil {
			t.Fatal(err)
		}
	}
	if d := daftarJenis(t, tx, agen)[JenisKomisiTersedia]; len(d) != 2 {
		t.Errorf("komisi tersedia: ingin 2 (lunas dan batal setelah lunas), dapat %d: %+v", len(d), d)
	}
}

// Pengingat hold 3 jam juga ke agen perekrut akun pemesan (prospek agen), tidak untuk booking buatan agen itu sendiri.
func TestHoldSeatAgenPerekrut(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	now := time.Now()
	jadwal, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-06-01"})
	agen := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenHoldQzp", StatusAgen: "aktif"})
	pemesan := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "PemesanHoldQzp", Direkrut: agen})
	booking := func(buatanAgen any) {
		j := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "JamaahHoldQzp"})
		b, _ := testdb.NewBookingPax(t, tx, jadwal, "baru", 1000, testdb.Pax{JamaahID: j})
		testdb.Exec(t, tx, `UPDATE bookings SET customer_account_id=?, seat_hold_expires_at=?, dibuat_oleh_agen_akun_id=? WHERE id=?`, pemesan, now.Add(2*time.Hour), buatanAgen, b)
	}
	booking(nil)
	booking(agen)
	for i := 0; i < 2; i++ {
		if err := holdSeat(ctx, tx, now); err != nil {
			t.Fatal(err)
		}
	}
	if d := daftarJenis(t, tx, agen)[JenisProspekHold]; len(d) != 1 || d[0].Judul != "Hold berakhir 3 jam lagi" {
		t.Errorf("hold agen: %+v", d)
	}
	if d := daftarJenis(t, tx, pemesan)[JenisHoldSeat]; len(d) != 2 {
		t.Errorf("hold pemesan tetap: %+v", d)
	}
}
