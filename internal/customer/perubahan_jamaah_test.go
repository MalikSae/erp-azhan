package customer

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	"erp-azhan/api/internal/testdb"
)

func TestSamarkanNilai(t *testing.T) {
	for in, want := range map[string]string{"": "", "ab": "••", "A1234567": "•••••567"} {
		if got := samarkanNilai(in); got != want {
			t.Errorf("samarkanNilai(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestKelompokOrang(t *testing.T) {
	semua := []anggotaJamaah{
		{ID: 9, Nama: "Siti  Aminah", Lahir: "1980-01-01"},
		{ID: 8, Nama: "Budi", Lahir: ""},
		{ID: 7, Nama: "siti aminah", Lahir: ""},           // nama sama, tanggal lahir belum diisi: orang yang sama
		{ID: 6, Nama: "Siti Aminah", Lahir: "1999-09-09"}, // tanggal lahir berbeda: orang lain
	}
	g := kelompokOrang(semua)
	if len(g) != 3 {
		t.Fatalf("jumlah orang = %d, want 3: %+v", len(g), g)
	}
	if len(g[0]) != 2 || g[0][0].ID != 9 || g[0][1].ID != 7 {
		t.Fatalf("grup pertama = %+v", g[0])
	}
}

// fixturePerubahan: satu akun dengan dua booking berisi orang yang sama (dua baris jamaah). Mengembalikan akun, baris lama, baris baru, admin.
func fixturePerubahan(t *testing.T, tx *sql.Tx) (akun, lama, baru, admin int64) {
	t.Helper()
	jadwal, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	akun = testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "PemilikQzk"})
	lama = testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "SitiQzk"})
	baru = testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "SitiQzk"})
	b1, _ := testdb.NewBookingPax(t, tx, jadwal, "dp", 1000, testdb.Pax{JamaahID: lama})
	b2, _ := testdb.NewBookingPax(t, tx, jadwal, "batal", 1000, testdb.Pax{JamaahID: baru}) // booking batal tetap ikut
	testdb.Exec(t, tx, `UPDATE bookings SET customer_account_id=? WHERE id IN (?,?)`, akun, b1, b2)
	testdb.Exec(t, tx, `INSERT INTO admin_users (email, password_hash) VALUES (?, 'x')`, "adm-qzk@uji.test")
	if err := tx.QueryRow(`SELECT id FROM admin_users WHERE email='adm-qzk@uji.test'`).Scan(&admin); err != nil {
		t.Fatal(err)
	}
	return
}

func nilaiKolom(t *testing.T, tx *sql.Tx, id int64, kolom string) string {
	t.Helper()
	var v string
	if err := tx.QueryRow(`SELECT COALESCE(CAST(`+kolom+` AS CHAR),'') FROM jamaah WHERE id=?`, id).Scan(&v); err != nil {
		t.Fatal(err)
	}
	return v
}

func TestPerubahanJamaahAlurLengkap(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	akun, lama, baru, admin := fixturePerubahan(t, tx)
	testdb.Exec(t, tx, `UPDATE jamaah SET tempat_lahir='Bandung' WHERE id=?`, lama)

	grup, err := kelompokDari(ctx, tx, akun, lama)
	if err != nil || len(grup) != 2 || grup[0].ID != baru {
		t.Fatalf("grup = %+v, err = %v (wakil harus baris terbaru %d)", grup, err, baru)
	}
	if _, err := kelompokDari(ctx, tx, akun+99999, lama); !errors.Is(err, errJamaahTidakAda) {
		t.Fatalf("akun lain harus ditolak: %v", err)
	}

	// Isian kosong (alamat) langsung tersimpan di kedua baris; isian terisi (tempat_lahir) hanya menjadi pengajuan.
	h, err := simpanPerubahanPelanggan(ctx, tx, akun, grup, map[string]string{"alamat": "Jl. Mawar 1", "tempat_lahir": "Cirebon"})
	if err != nil {
		t.Fatal(err)
	}
	if h.Langsung != 1 || h.Pengajuan != 1 || h.PengajuanID == 0 {
		t.Fatalf("hasil = %+v", h)
	}
	for _, id := range []int64{lama, baru} {
		if got := nilaiKolom(t, tx, id, "alamat"); got != "Jl. Mawar 1" {
			t.Errorf("alamat baris %d = %q", id, got)
		}
	}
	if got := nilaiKolom(t, tx, lama, "tempat_lahir"); got != "Bandung" {
		t.Errorf("tempat_lahir berubah sebelum disetujui: %q", got)
	}
	pj, err := pengajuanUntukPelanggan(ctx, tx, akun, idGrup(grup))
	if err != nil || pj == nil || pj.Status != "menunggu" || len(pj.Perubahan) != 1 || pj.Perubahan[0].Lama != "Bandung" || pj.Perubahan[0].Baru != "Cirebon" {
		t.Fatalf("pengajuan = %+v, err = %v", pj, err)
	}

	// Simpan lagi sebelum diputuskan: pengajuan lama digantikan, hanya satu yang menunggu.
	h2, err := simpanPerubahanPelanggan(ctx, tx, akun, grup, map[string]string{"tempat_lahir": "Garut"})
	if err != nil || h2.Pengajuan != 1 {
		t.Fatalf("simpan kedua: %+v %v", h2, err)
	}
	var menunggu, diganti int
	tx.QueryRow(`SELECT SUM(status='menunggu'), SUM(status='diganti') FROM jamaah_perubahan WHERE akun_id=? AND sumber='pelanggan'`, akun).Scan(&menunggu, &diganti)
	if menunggu != 1 || diganti != 1 {
		t.Fatalf("menunggu=%d diganti=%d, want 1 dan 1", menunggu, diganti)
	}

	// Menyimpan nilai yang sama tidak menghasilkan apa pun.
	if _, err := simpanPerubahanPelanggan(ctx, tx, akun, grup, map[string]string{"alamat": "Jl. Mawar 1"}); !errors.Is(err, ErrTidakBerubah) {
		t.Fatalf("err = %v, want ErrTidakBerubah", err)
	}

	// Disetujui: nilai baru diterapkan ke kedua baris, pengajuan tercatat beserta pemutusnya.
	if _, err := setujuiTx(ctx, tx, h2.PengajuanID, admin, nil); err != nil {
		t.Fatal(err)
	}
	for _, id := range []int64{lama, baru} {
		if got := nilaiKolom(t, tx, id, "tempat_lahir"); got != "Garut" {
			t.Errorf("tempat_lahir baris %d = %q setelah disetujui", id, got)
		}
	}
	var status string
	var oleh sql.NullInt64
	tx.QueryRow(`SELECT status, diputus_oleh FROM jamaah_perubahan WHERE id=?`, h2.PengajuanID).Scan(&status, &oleh)
	if status != "disetujui" || oleh.Int64 != admin {
		t.Fatalf("status=%s oleh=%v", status, oleh)
	}
	if _, err := setujuiTx(ctx, tx, h2.PengajuanID, admin, nil); !errors.Is(err, errPerubahanSelesai) {
		t.Fatalf("setujui dua kali harus ditolak: %v", err)
	}
}

func TestPerubahanJamaahDitolakDanValidasi(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	akun, lama, _, admin := fixturePerubahan(t, tx)
	testdb.Exec(t, tx, `UPDATE jamaah SET kota='Bogor', no_paspor='A1234567' WHERE id=?`, lama)
	grup, _ := kelompokDari(ctx, tx, akun, lama)

	h, err := simpanPerubahanPelanggan(ctx, tx, akun, grup, map[string]string{"kota": "Depok", "no_paspor": "B7654321"})
	if err != nil || h.Pengajuan != 2 {
		t.Fatalf("hasil = %+v, err = %v", h, err)
	}
	if _, err := tolakTx(ctx, tx, h.PengajuanID, admin, nil, "Foto paspor belum cocok"); err != nil {
		t.Fatal(err)
	}
	if got := nilaiKolom(t, tx, lama, "kota"); got != "Bogor" {
		t.Errorf("kota berubah walau ditolak: %q", got)
	}
	pj, _ := pengajuanUntukPelanggan(ctx, tx, akun, idGrup(grup))
	if pj == nil || pj.Status != "ditolak" || pj.Catatan != "Foto paspor belum cocok" {
		t.Fatalf("pelanggan harus melihat alasan penolakan: %+v", pj)
	}

	// Riwayat yang sudah diputuskan menyamarkan nomor paspor; yang menunggu tampil utuh untuk diperiksa admin.
	diputus, _ := muatItem(ctx, tx, h.PengajuanID, true)
	for _, it := range diputus {
		if it.Kolom == "no_paspor" && (strings.Contains(it.Baru, "B7654") || strings.Contains(it.Lama, "A1234")) {
			t.Errorf("paspor tidak tersamar: %+v", it)
		}
	}
	utuh, _ := muatItem(ctx, tx, h.PengajuanID, false)
	for _, it := range utuh {
		if it.Kolom == "no_paspor" && it.Baru != "B7654321" {
			t.Errorf("paspor harus utuh saat diperiksa: %+v", it)
		}
	}

	for nama, perubahan := range map[string]map[string]string{
		"isian tidak dikenal":   {"nik": "3201010101010001"},
		"nama tidak boleh":      {"nama_lengkap": "Lain"},
		"jenis kelamin salah":   {"jenis_kelamin": "X"},
		"paspor format salah":   {"no_paspor": "!!"},
		"tanggal bukan tanggal": {"tanggal_lahir": "31-12-1990"},
	} {
		_, err := simpanPerubahanPelanggan(ctx, tx, akun, grup, perubahan)
		var val *ErrValidasi
		if !errors.As(err, &val) {
			t.Errorf("%s: err = %v, want ErrValidasi", nama, err)
		}
	}
}
