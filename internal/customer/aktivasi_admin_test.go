package customer

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"erp-azhan/api/internal/testdb"
)

func nikUji(seq int) string {
	return fmt.Sprintf("97%014d", (time.Now().UnixNano()/1000+int64(seq))%100000000000000)
}

func sesiAktif(t *testing.T, tx *sql.Tx, akun int64, tanda string) {
	t.Helper()
	testdb.Exec(t, tx, `INSERT INTO customer_sessions (token_hash, account_id, expires_at) VALUES (?, ?, DATE_ADD(NOW(), INTERVAL 1 DAY))`,
		fmt.Sprintf("%-64s", fmt.Sprintf("%s%d%d", tanda, akun, time.Now().UnixNano()))[:64], akun)
}

func hitung(t *testing.T, tx *sql.Tx, q string, args ...any) int {
	t.Helper()
	var n int
	must(t, "hitung", tx.QueryRow(q, args...).Scan(&n))
	return n
}

func TestLepasTautanAkun(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	_, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	admin := adminId(t, tx)
	kunci := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Pemilik Kunci", StatusAgen: "aktif"})
	jamaah := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Ahmad", Direkrut: kunci, NIK: nikUji(1)})
	akun := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Akun Ahmad", Direkrut: kunci})
	testdb.Exec(t, tx, `UPDATE customer_accounts SET jamaah_id=? WHERE id=?`, jamaah, akun)
	sesiAktif(t, tx, akun, "a")
	sesiAktif(t, tx, akun, "b")

	// Alasan wajib.
	var ev *ErrValidasi
	if err := lepasTautanTx(ctx, tx, &brand, akun, admin, AlasanRequest{Alasan: "abc"}); !errors.As(err, &ev) {
		t.Fatalf("alasan pendek: err = %v, want ErrValidasi", err)
	}
	// Brand lain tidak boleh menyentuh akun ini.
	lain := brand + 100000
	mustErrIs(t, "brand lain", lepasTautanTx(ctx, tx, &lain, akun, admin, AlasanRequest{Alasan: "salah orang"}), ErrNotFound)

	must(t, "lepas", lepasTautanTx(ctx, tx, &brand, akun, admin, AlasanRequest{Alasan: "tertaut ke jamaah yang salah"}))

	var jid sql.NullInt64
	var direkrut sql.NullInt64
	must(t, "baca akun", tx.QueryRow(`SELECT jamaah_id, direkrut_oleh_akun_id FROM customer_accounts WHERE id=?`, akun).Scan(&jid, &direkrut))
	if jid.Valid {
		t.Fatalf("jamaah_id = %v, want NULL", jid)
	}
	if !direkrut.Valid || direkrut.Int64 != kunci {
		t.Fatalf("upline berubah: %v, want tetap %d", direkrut, kunci)
	}
	if n := hitung(t, tx, `SELECT COUNT(*) FROM customer_sessions WHERE account_id=? AND revoked_at IS NULL`, akun); n != 0 {
		t.Fatalf("sesi aktif tersisa = %d, want 0", n)
	}
	if n := hitung(t, tx, `SELECT COUNT(*) FROM akun_data_log WHERE akun_id=? AND kolom='jamaah_id' AND diubah_oleh=? AND alasan='tertaut ke jamaah yang salah'`, akun, admin); n != 1 {
		t.Fatalf("akun_data_log = %d, want 1", n)
	}
	var aksi, catatan string
	var oleh, jamaahLog, akunLog sql.NullInt64
	must(t, "log aktivasi", tx.QueryRow(`SELECT aksi, oleh_admin_id, jamaah_id, akun_id, catatan FROM jamaah_aktivasi_log WHERE akun_id=? AND aksi='lepas_admin'`, akun).
		Scan(&aksi, &oleh, &jamaahLog, &akunLog, &catatan))
	if oleh.Int64 != admin || jamaahLog.Int64 != jamaah || catatan != "tertaut ke jamaah yang salah" {
		t.Fatalf("log = oleh %v jamaah %v catatan %q", oleh, jamaahLog, catatan)
	}

	// Sudah lepas: tidak ada yang dilepas lagi.
	mustErrIs(t, "ulang", lepasTautanTx(ctx, tx, &brand, akun, admin, AlasanRequest{Alasan: "coba lagi saja"}), ErrTidakTertaut)

	// Setelah dilepas, jamaah bebas ditautkan akun lain (UNIQUE tidak menghalangi).
	lain2 := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Akun Lain"})
	if _, err := tx.Exec(`UPDATE customer_accounts SET jamaah_id=? WHERE id=?`, jamaah, lain2); err != nil {
		t.Fatalf("tautkan akun lain: %v", err)
	}
}

func TestLepasTautanDitolakUntukAgen(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	_, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	admin := adminId(t, tx)
	for i, status := range []string{"aktif", "nonaktif"} {
		jamaah := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Calon Agen", NIK: nikUji(10 + i)})
		akun := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Agen " + status, StatusAgen: status})
		testdb.Exec(t, tx, `UPDATE customer_accounts SET jamaah_id=? WHERE id=?`, jamaah, akun)
		sesiAktif(t, tx, akun, status)
		mustErrIs(t, status, lepasTautanTx(ctx, tx, &brand, akun, admin, AlasanRequest{Alasan: "ingin dipindah"}), ErrAgenAktif)
		var jid sql.NullInt64
		must(t, "baca", tx.QueryRow(`SELECT jamaah_id FROM customer_accounts WHERE id=?`, akun).Scan(&jid))
		if !jid.Valid || jid.Int64 != jamaah {
			t.Fatalf("%s: tautan berubah: %v", status, jid)
		}
		if n := hitung(t, tx, `SELECT COUNT(*) FROM customer_sessions WHERE account_id=? AND revoked_at IS NULL`, akun); n != 1 {
			t.Fatalf("%s: sesi tercabut padahal ditolak", status)
		}
		if n := hitung(t, tx, `SELECT COUNT(*) FROM jamaah_aktivasi_log WHERE akun_id=?`, akun); n != 0 {
			t.Fatalf("%s: log tertulis padahal ditolak", status)
		}
	}
	// Pengajuan agen boleh dilepas.
	jamaah := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Pengaju", NIK: nikUji(20)})
	akun := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Pengaju", StatusAgen: "pengajuan"})
	testdb.Exec(t, tx, `UPDATE customer_accounts SET jamaah_id=? WHERE id=?`, jamaah, akun)
	must(t, "pengajuan", lepasTautanTx(ctx, tx, &brand, akun, admin, AlasanRequest{Alasan: "salah tautan"}))
}

func TestKosongkanNIKAgen(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	_, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	admin := adminId(t, tx)
	nik := nikUji(30)
	jamaah := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Ahmad", NIK: nik})
	pemilik := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Akun Ahmad"})
	testdb.Exec(t, tx, `UPDATE customer_accounts SET jamaah_id=? WHERE id=?`, jamaah, pemilik)
	penyerobot := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Penyerobot", StatusAgen: "pengajuan"})
	testdb.Exec(t, tx, `UPDATE customer_accounts SET nik_agen=? WHERE id=?`, nik, penyerobot)
	alasan := AlasanRequest{Alasan: "NIK milik jamaah tertaut"}

	mustErrIs(t, "tanpa izin", kosongkanNIKAgenTx(ctx, tx, &brand, penyerobot, admin, IzinNIK{}, alasan), ErrDilarang)
	mustErrIs(t, "izin aktif saja untuk pengajuan", kosongkanNIKAgenTx(ctx, tx, &brand, penyerobot, admin, IzinNIK{Aktif: true}, alasan), ErrDilarang)
	var ev *ErrValidasi
	if err := kosongkanNIKAgenTx(ctx, tx, &brand, penyerobot, admin, IzinNIK{Pengajuan: true}, AlasanRequest{Alasan: "x"}); !errors.As(err, &ev) {
		t.Fatalf("alasan pendek: %v", err)
	}
	lain := brand + 100000
	mustErrIs(t, "brand lain", kosongkanNIKAgenTx(ctx, tx, &lain, penyerobot, admin, IzinNIK{Pengajuan: true}, alasan), ErrNotFound)
	// Pemegang yang sah (akun tertaut sendiri) tidak bentrok dengan dirinya.
	testdb.Exec(t, tx, `UPDATE customer_accounts SET nik_agen=NULL WHERE id=?`, penyerobot)
	testdb.Exec(t, tx, `UPDATE customer_accounts SET nik_agen=? WHERE id=?`, nik, pemilik)
	mustErrIs(t, "pemegang sah", kosongkanNIKAgenTx(ctx, tx, &brand, pemilik, admin, IzinNIK{Pengajuan: true, Aktif: true}, alasan), ErrNIKTidakBentrok)
	mustErrIs(t, "nik kosong", kosongkanNIKAgenTx(ctx, tx, &brand, penyerobot, admin, IzinNIK{Pengajuan: true}, alasan), ErrNIKTidakBentrok)

	// UNIQUE (brand, nik_agen): pemilik melepas NIK dulu supaya penyerobot bisa memegangnya, seperti keadaan nyata.
	testdb.Exec(t, tx, `UPDATE customer_accounts SET nik_agen=NULL WHERE id=?`, pemilik)
	testdb.Exec(t, tx, `UPDATE customer_accounts SET nik_agen=? WHERE id=?`, nik, penyerobot)
	must(t, "kosongkan", kosongkanNIKAgenTx(ctx, tx, &brand, penyerobot, admin, IzinNIK{Pengajuan: true}, alasan))
	var n sql.NullString
	must(t, "baca", tx.QueryRow(`SELECT nik_agen FROM customer_accounts WHERE id=?`, penyerobot).Scan(&n))
	if n.Valid {
		t.Fatalf("nik_agen = %q, want NULL", n.String)
	}
	var baru sql.NullString
	var lama string
	must(t, "log", tx.QueryRow(`SELECT nilai_lama, nilai_baru FROM akun_data_log WHERE akun_id=? AND kolom='nik_agen'`, penyerobot).Scan(&lama, &baru))
	if lama != samarkanNIK(nik) || baru.Valid || strings.Contains(lama, nik) {
		t.Fatalf("log = %q -> %v", lama, baru)
	}

	// Agen aktif butuh izin Aktif (agen.kaitan.edit).
	nik2 := nikUji(31)
	j2 := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Zainab", NIK: nik2})
	a2 := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Akun Zainab"})
	testdb.Exec(t, tx, `UPDATE customer_accounts SET jamaah_id=? WHERE id=?`, j2, a2)
	agen := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Agen Serobot", StatusAgen: "aktif"})
	testdb.Exec(t, tx, `UPDATE customer_accounts SET nik_agen=? WHERE id=?`, nik2, agen)
	mustErrIs(t, "agen aktif tanpa izin aktif", kosongkanNIKAgenTx(ctx, tx, &brand, agen, admin, IzinNIK{Pengajuan: true}, alasan), ErrDilarang)
	must(t, "agen aktif", kosongkanNIKAgenTx(ctx, tx, &brand, agen, admin, IzinNIK{Aktif: true}, alasan))
}

func TestListAgenDariTautan(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	sched, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	kunci := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Kunci", StatusAgen: "aktif"})
	jamaah := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Ahmad", Direkrut: kunci, NIK: nikUji(40)})
	testdb.NewBookingPax(t, tx, sched, "dp", 1000, testdb.Pax{JamaahID: jamaah})
	agen := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Ahmad Agen", StatusAgen: "aktif", Direkrut: kunci, Upline: kunci})
	testdb.Exec(t, tx, `UPDATE customer_accounts SET jamaah_id=? WHERE id=?`, jamaah, agen)
	// Bukan agen (tidak_aktif) dan agen tanpa tautan tidak masuk daftar.
	biasa := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Biasa", NIK: nikUji(41)})
	akunBiasa := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Akun Biasa"})
	testdb.Exec(t, tx, `UPDATE customer_accounts SET jamaah_id=? WHERE id=?`, biasa, akunBiasa)
	testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Agen Tanpa Tautan", StatusAgen: "aktif"})

	h, err := listAgenDariTautan(ctx, tx, &brand, 1, 50)
	must(t, "list", err)
	var ada *AgenTautanBaris
	for i := range h.Items {
		if h.Items[i].AkunID == akunBiasa {
			t.Fatalf("akun biasa ikut daftar")
		}
		if h.Items[i].AkunID == agen {
			ada = &h.Items[i]
		}
	}
	if ada == nil {
		t.Fatalf("agen dari tautan tidak ada di daftar (%d baris)", len(h.Items))
	}
	if ada.JamaahID != jamaah || ada.PemilikKunciID == nil || *ada.PemilikKunciID != kunci || !ada.UplineSamaKunci || !ada.BookingPendukungAktif || ada.KaitanStatus != "terikat_agen" {
		t.Fatalf("baris = %+v", *ada)
	}
	// Booking pendukung batal: penanda mati.
	testdb.Exec(t, tx, `UPDATE bookings SET status='batal' WHERE id IN (SELECT booking_id FROM booking_pax WHERE jamaah_id=?)`, jamaah)
	h, err = listAgenDariTautan(ctx, tx, &brand, 1, 50)
	must(t, "list 2", err)
	for _, b := range h.Items {
		if b.AkunID == agen && b.BookingPendukungAktif {
			t.Fatalf("booking_pendukung_aktif = true setelah booking batal")
		}
	}
}

func TestMeJamaahTertautTanpaKodeBooking(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	sched, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	jamaah := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Ahmad", NIK: nikUji(50)})
	bid, _ := testdb.NewBookingPax(t, tx, sched, "dp", 1000, testdb.Pax{JamaahID: jamaah})
	var kode string
	must(t, "kode", tx.QueryRow(`SELECT id_booking FROM bookings WHERE id=?`, bid).Scan(&kode))
	batal, _ := testdb.NewBookingPax(t, tx, sched, "batal", 1000, testdb.Pax{JamaahID: jamaah})
	_ = batal
	akun := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Akun Ahmad"})
	testdb.Exec(t, tx, `UPDATE customer_accounts SET jamaah_id=? WHERE id=?`, jamaah, akun)

	a := &Account{ID: akun, JamaahID: &jamaah}
	v := buildMeView(ctx, tx, a)
	if !v.JamaahTertaut || len(v.PerjalananTertaut) != 1 {
		t.Fatalf("view = tertaut %v perjalanan %+v, want 1 perjalanan (booking batal tidak tampil)", v.JamaahTertaut, v.PerjalananTertaut)
	}
	p := v.PerjalananTertaut[0]
	if p.Paket == "" || p.BerangkatTanggal != "2099-01-01" || p.Status != "dp" {
		t.Fatalf("perjalanan = %+v", p)
	}
	raw, err := json.Marshal(map[string]any{"account": v})
	must(t, "json", err)
	for _, dilarang := range []string{kode, "id_booking", "kode_booking", "total", "harga"} {
		if strings.Contains(string(raw), dilarang) {
			t.Fatalf("body memuat %q: %s", dilarang, raw)
		}
	}

	// Akun tanpa tautan: false dan daftar kosong (bukan null).
	biasa := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Biasa"})
	ab := &Account{ID: biasa}
	vb := buildMeView(ctx, tx, ab)
	rawB, _ := json.Marshal(vb)
	if vb.JamaahTertaut || !strings.Contains(string(rawB), `"jamaah_tertaut":false`) || !strings.Contains(string(rawB), `"perjalanan_tertaut":[]`) {
		t.Fatalf("akun biasa: %s", rawB)
	}
}

func TestBolehTulisLogGagalBatas(t *testing.T) {
	_, tx := testdb.Tx(t)
	_, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	h := strings.Repeat("c", 64)
	ctx := context.Background()
	for i := 0; i < maxLogGagalPerKode-1; i++ {
		must(t, "log", catatLogAktivasi(ctx, tx, entriLogAktivasi{BrandID: brand, Aksi: "cek_gagal", KodeHash: h, Alasan: "nik_salah"}))
	}
	if !bolehTulisLogGagal(ctx, tx, h) {
		t.Fatal("di bawah batas: harus boleh")
	}
	must(t, "log terakhir", catatLogAktivasi(ctx, tx, entriLogAktivasi{BrandID: brand, Aksi: "cek_gagal", KodeHash: h, Alasan: "nik_salah"}))
	if bolehTulisLogGagal(ctx, tx, h) {
		t.Fatal("di batas: harus ditolak")
	}
	// Kode lain tidak terpengaruh; aktivasi sukses tidak dihitung.
	if !bolehTulisLogGagal(ctx, tx, strings.Repeat("d", 64)) {
		t.Fatal("kode lain harus boleh")
	}
}

func TestBersihkanAktivasi(t *testing.T) {
	db := testdb.DB(t)
	ctx := context.Background()
	var brand int64
	must(t, "brand", db.QueryRow(`SELECT brand_id FROM schedules ORDER BY id LIMIT 1`).Scan(&brand))
	tanda := fmt.Sprintf("%x", time.Now().UnixNano())
	hash := func(s string) string { return strings.Repeat("e", 64-len(tanda)-len(s)) + tanda + s }
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM jamaah_aktivasi_hitung WHERE kunci_hash LIKE ?`, "%"+tanda+"%")
		_, _ = db.Exec(`DELETE FROM jamaah_aktivasi_log WHERE kode_hash LIKE ?`, "%"+tanda+"%")
	})
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := db.Exec(q, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`INSERT INTO jamaah_aktivasi_hitung (kunci_hash, jenis, jendela, jumlah, kedaluwarsa_at) VALUES (?, 'ip', 1, 3, DATE_SUB(NOW(), INTERVAL 1 HOUR))`, hash("h1"))
	exec(`INSERT INTO jamaah_aktivasi_hitung (kunci_hash, jenis, jendela, jumlah, kedaluwarsa_at) VALUES (?, 'ip', 2, 3, DATE_ADD(NOW(), INTERVAL 1 HOUR))`, hash("h2"))
	ins := func(aksi, kode string, hari int) {
		exec(`INSERT INTO jamaah_aktivasi_log (brand_id, aksi, kode_hash, alasan, dibuat_pada) VALUES (?, ?, ?, 'uji', DATE_SUB(NOW(), INTERVAL ? DAY))`, brand, aksi, hash(kode), hari)
	}
	ins("cek_gagal", "l1", 100)    // gagal lama: dibuang
	ins("cek_gagal", "l2", 10)     // gagal baru: tetap
	ins("aktivasi", "l3", 100)     // jejak 100 hari: tetap
	ins("lepas_admin", "l4", 800)  // jejak lewat 2 tahun: dibuang
	ins("kode_dikunci", "l5", 120) // dibuang

	h, err := BersihkanAktivasi(ctx, db)
	must(t, "bersih", err)
	if h.Hitung < 1 || h.LogGagal < 2 || h.LogJejak < 1 {
		t.Fatalf("hasil = %+v", h)
	}
	ada := func(tabel, kolom, kode string) bool {
		var n int
		must(t, "ada", db.QueryRow(`SELECT COUNT(*) FROM `+tabel+` WHERE `+kolom+` = ?`, hash(kode)).Scan(&n))
		return n > 0
	}
	for kode, want := range map[string]bool{"l1": false, "l2": true, "l3": true, "l4": false, "l5": false} {
		if got := ada("jamaah_aktivasi_log", "kode_hash", kode); got != want {
			t.Errorf("log %s ada = %v, want %v", kode, got, want)
		}
	}
	if ada("jamaah_aktivasi_hitung", "kunci_hash", "h1") || !ada("jamaah_aktivasi_hitung", "kunci_hash", "h2") {
		t.Errorf("penghitung: kedaluwarsa harus dibuang, yang berlaku tetap")
	}
}

func TestDetailAkunJamaahTertautDanRiwayat(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	_, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	admin := adminId(t, tx)
	kunci := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Pemilik Kunci", StatusAgen: "aktif"})
	nik := nikUji(30)
	jamaah := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Ahmad", Direkrut: kunci, NIK: nik})
	akun := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Akun Ahmad", Direkrut: kunci})
	testdb.Exec(t, tx, `UPDATE customer_accounts SET jamaah_id=? WHERE id=?`, jamaah, akun)
	// Akun lain memegang NIK jamaah tertaut sebagai NIK agen.
	squat := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Pemegang NIK", StatusAgen: "pengajuan"})
	testdb.Exec(t, tx, `UPDATE customer_accounts SET nik_agen=? WHERE id=?`, nik, squat)
	must(t, "log tolak", catatLogAktivasi(ctx, tx, entriLogAktivasi{BrandID: brand, Aksi: "tolak", JamaahID: jamaah, AkunID: akun, Alasan: "upline_beda", KodeHash: strings.Repeat("a", 64)}))

	d, err := getAkunDetail(ctx, tx, &brand, akun)
	must(t, "detail", err)
	if d.Jamaah == nil || !strings.HasPrefix(d.Jamaah.Nama, "Ahmad") || d.Jamaah.PemilikKunciID == nil || *d.Jamaah.PemilikKunciID != kunci || !strings.HasPrefix(*d.Jamaah.PemilikKunciNama, "Pemilik Kunci") {
		t.Fatalf("jamaah tertaut = %+v", d.Jamaah)
	}
	if len(d.RiwayatAktivasi) != 1 || d.RiwayatAktivasi[0].Aksi != "tolak" || d.RiwayatAktivasi[0].Alasan != "upline_beda" {
		t.Fatalf("riwayat = %+v", d.RiwayatAktivasi)
	}
	ds, err := getAkunDetail(ctx, tx, &brand, squat)
	must(t, "detail squat", err)
	if !ds.NIKAgenBentrok {
		t.Fatal("nik_agen_bentrok = false, want true")
	}
	if d.NIKAgenBentrok {
		t.Fatal("akun tanpa NIK agen ditandai bentrok")
	}

	must(t, "lepas", lepasTautanTx(ctx, tx, &brand, akun, admin, AlasanRequest{Alasan: "tertaut ke orang salah"}))
	d, err = getAkunDetail(ctx, tx, &brand, akun)
	must(t, "detail lepas", err)
	if d.Jamaah != nil || len(d.RiwayatAktivasi) != 2 || d.RiwayatAktivasi[0].Aksi != "lepas_admin" || d.RiwayatAktivasi[0].Catatan != "tertaut ke orang salah" || d.RiwayatAktivasi[0].Admin == nil {
		t.Fatalf("setelah lepas: jamaah=%v riwayat=%+v", d.Jamaah, d.RiwayatAktivasi)
	}

	hasil, err := listAkun(ctx, tx, &brand, FilterAkun{Cari: "Akun Ahmad"})
	must(t, "list", err)
	if len(hasil.Items) != 1 || hasil.Items[0].JamaahID != nil {
		t.Fatalf("list setelah lepas = %+v", hasil.Items)
	}
}
