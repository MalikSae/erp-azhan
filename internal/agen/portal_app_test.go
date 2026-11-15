package agen

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"erp-azhan/api/internal/testdb"
)

func TestStatusBinaan(t *testing.T) {
	cases := []struct {
		status string
		jamaah int
		want   string
	}{
		{"aktif", 0, "rintisan"},
		{"aktif", 3, "aktif"},
		{"nonaktif", 5, "nonaktif"},
	}
	for _, c := range cases {
		if got := statusBinaan(c.status, c.jamaah); got != c.want {
			t.Errorf("statusBinaan(%s,%d) = %s, mau %s", c.status, c.jamaah, got, c.want)
		}
	}
}

func TestPortalAppAgen(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	now := time.Date(2099, 5, 10, 10, 0, 0, 0, zonaWIB)
	jadwal, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-06-01"})

	agen := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenAppQzp", StatusAgen: "aktif"})
	binaanAktif := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "BinaanSatuQzp", StatusAgen: "aktif", Upline: agen, Direkrut: agen})
	binaanRintis := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "BinaanDuaQzp", StatusAgen: "aktif", Upline: agen, Direkrut: agen})
	binaanMati := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "BinaanTigaQzp", StatusAgen: "nonaktif", Upline: agen, Direkrut: agen})
	testdb.Exec(t, tx, `UPDATE customer_accounts SET domisili = 'Kota Depok', disetujui_agen_at = '2099-01-02 09:00:00' WHERE id = ?`, binaanAktif)

	// Jamaah agen: satu booking DP, satu lunas yang sudah berangkat, satu booking baru (tidak dihitung).
	j1 := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "JamaahAppSatuQzp", Direkrut: agen})
	j2 := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "JamaahAppDuaQzp", Direkrut: agen})
	j3 := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "JamaahAppTigaQzp", Direkrut: agen})
	b1, pax1 := testdb.NewBookingPax(t, tx, jadwal, "dp", 1000, testdb.Pax{JamaahID: j1})
	testdb.NewBookingPax(t, tx, jadwal, "baru", 1000, testdb.Pax{JamaahID: j3})
	jadwalLalu := jadwalLainBrand(t, tx, brand, jadwal, "2099-05-01")
	testdb.NewBookingPax(t, tx, jadwalLalu, "lunas", 1000, testdb.Pax{JamaahID: j2})
	// Jamaah binaan aktif: satu booking DP di paket yang belum berangkat.
	jb := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "JamaahBinaanQzp", Direkrut: binaanAktif})
	testdb.NewBookingPax(t, tx, jadwal, "dp", 1000, testdb.Pax{JamaahID: jb})

	data, err := listDataJamaah(ctx, tx, agen, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 2 {
		t.Fatalf("data jamaah: %+v", data)
	}
	status := map[int64]string{}
	for _, d := range data {
		status[d.JamaahID] = d.Status
		if d.Jenis != "umroh" {
			t.Errorf("jenis %s: %s", d.Nama, d.Jenis)
		}
	}
	if status[j1] != "dp" || status[j2] != "berangkat" {
		t.Errorf("status data jamaah: %+v", status)
	}

	binaan, err := listAgenBinaan(ctx, tx, agen, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(binaan) != 3 {
		t.Fatalf("agen binaan: %+v", binaan)
	}
	for _, b := range binaan {
		switch b.AkunID {
		case binaanAktif:
			if b.Status != "aktif" || b.JumlahJamaah != 1 || b.PaketAktif != 1 || b.Kota == nil || *b.Kota != "Kota Depok" ||
				b.Bergabung == nil || *b.Bergabung != "2099-01-02" {
				t.Errorf("binaan aktif: %+v", b)
			}
		case binaanRintis:
			if b.Status != "rintisan" {
				t.Errorf("binaan rintisan: %+v", b)
			}
		case binaanMati:
			if b.Status != "nonaktif" {
				t.Errorf("binaan nonaktif: %+v", b)
			}
		}
	}

	// Mutasi: komisi masuk, koreksi saat batal, pencairan diproses; pencairan ditolak tidak ikut; cashback tidak ikut.
	testdb.Exec(t, tx, `INSERT INTO transaksi_komisi (brand_id, akun_penerima_id, jamaah_sumber_id, booking_id, booking_pax_id, jenis, nominal, created_at)
		VALUES (?,?,?,?,?,'langsung',500000,'2030-05-01 10:00:00')`, brand, agen, j1, b1, pax1[0])
	testdb.Exec(t, tx, `INSERT INTO transaksi_komisi (brand_id, akun_penerima_id, jamaah_sumber_id, booking_id, booking_pax_id, jenis, nominal, created_at, dibatalkan_at)
		VALUES (?,?,?,?,?,'pembinaan',100000,'2030-05-02 10:00:00','2030-05-05 10:00:00')`, brand, agen, j1, b1, pax1[0])
	testdb.Exec(t, tx, `INSERT INTO transaksi_komisi (brand_id, akun_penerima_id, jamaah_sumber_id, booking_id, booking_pax_id, jenis, nominal)
		VALUES (?,?,?,?,?,'cashback',50000)`, brand, agen, j1, b1, pax1[0])
	testdb.Exec(t, tx, `INSERT INTO pengajuan_pencairan (brand_id, akun_agen_id, nominal_diajukan, status, rekening_bank, rekening_nomor, rekening_atas_nama, diajukan_at)
		VALUES (?,?,500000,'pending','BCA','123','Agen','2030-05-06 10:00:00')`, brand, agen)
	testdb.Exec(t, tx, `INSERT INTO pengajuan_pencairan (brand_id, akun_agen_id, nominal_diajukan, status, rekening_bank, rekening_nomor, rekening_atas_nama, diajukan_at)
		VALUES (?,?,700000,'ditolak','BCA','123','Agen','2030-05-07 10:00:00')`, brand, agen)
	mutasi, err := listMutasiAgen(ctx, tx, agen)
	if err != nil {
		t.Fatal(err)
	}
	if len(mutasi) != 4 {
		t.Fatalf("mutasi: %+v", mutasi)
	}
	want := []struct {
		jenis   string
		nominal float64
	}{{"pencairan", -500000}, {"koreksi", -100000}, {"komisi", 100000}, {"komisi", 500000}}
	for i, w := range want {
		if mutasi[i].Jenis != w.jenis || mutasi[i].Nominal != w.nominal {
			t.Errorf("mutasi[%d] = %+v, mau %s %.0f", i, mutasi[i], w.jenis, w.nominal)
		}
	}
	if mutasi[0].Judul != "Pencairan ke BCA" {
		t.Errorf("judul pencairan: %s", mutasi[0].Judul)
	}

	// Klik referral: satu per pengunjung per hari; kode tak dikenal diabaikan.
	var kode string
	if err := tx.QueryRowContext(ctx, `SELECT kode_referral FROM customer_accounts WHERE id = ?`, agen).Scan(&kode); err != nil {
		t.Fatal(err)
	}
	p1, p2 := HashPengunjung("pengunjung-1", "", ""), HashPengunjung("pengunjung-2", "", "")
	for _, p := range []string{p1, p1, p2} {
		if err := catatKlikReferral(ctx, tx, brand, kode, p, now); err != nil {
			t.Fatal(err)
		}
	}
	if err := catatKlikReferral(ctx, tx, brand, "TIDAKADA", p1, now); err != nil {
		t.Errorf("kode tak dikenal: %v", err)
	}
	if err := catatKlikReferral(ctx, tx, brand, kode, p1, now.AddDate(0, 0, 1)); err != nil {
		t.Fatal(err)
	}
	if err := catatKlikReferral(ctx, tx, brand, kode, p1, now.AddDate(0, -1, 0)); err != nil { // bulan lalu tidak dihitung
		t.Fatal(err)
	}
	if n, err := klikBulanIni(ctx, tx, agen, now.AddDate(0, 0, 1)); err != nil || n != 3 {
		t.Errorf("klik bulan ini = %d, mau 3 (err %v)", n, err)
	}
}

// jadwalLainBrand memakai paket lain milik brand yang sama dengan tanggal berangkat tertentu.
func jadwalLainBrand(t *testing.T, tx *sql.Tx, brand, kecuali int64, berangkat string) int64 {
	t.Helper()
	var id int64
	if err := tx.QueryRowContext(context.Background(), `SELECT id FROM schedules WHERE brand_id = ? AND id <> ? ORDER BY id LIMIT 1`, brand, kecuali).Scan(&id); err != nil {
		t.Skipf("butuh dua paket satu brand di DB uji: %v", err)
	}
	testdb.Exec(t, tx, `UPDATE schedules SET berangkat_tanggal = ? WHERE id = ?`, berangkat, id)
	return id
}
