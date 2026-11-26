package notifikasi

import (
	"context"
	"strconv"
	"testing"

	"erp-azhan/api/internal/testdb"
)

// Audit Manasik M9: pengingat H-1 manasik sekali per akun per manasik, ke pemesan dan akun tertaut jamaah dewasa
// booking dp/lunas. Booking belum DP dan bayi tidak.
func TestManasikBesok(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	jadwal, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-02-01"})
	testdb.Exec(t, tx, `DELETE FROM manasik_jadwal WHERE schedule_id = ?`, jadwal)

	pemesan := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "PemesanBesokQzp"})
	tertaut := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "TertautBesokQzp"})
	belumDP := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "BelumDPBesokQzp"})
	tertautBayi := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "BayiBesokQzp"})
	j := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "JamaahBesokQzp"})
	bayi := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "BayiJamaahBesokQzp"})
	jBaru := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "BaruBesokQzp"})
	testdb.Exec(t, tx, `UPDATE customer_accounts SET jamaah_id = ? WHERE id = ?`, j, tertaut)
	testdb.Exec(t, tx, `UPDATE customer_accounts SET jamaah_id = ? WHERE id = ?`, bayi, tertautBayi)
	b, pax := testdb.NewBookingPax(t, tx, jadwal, "dp", 1000, testdb.Pax{JamaahID: j}, testdb.Pax{JamaahID: bayi})
	testdb.Exec(t, tx, `UPDATE booking_pax SET pax_type = 'infant' WHERE id = ?`, pax[1])
	bBaru, _ := testdb.NewBookingPax(t, tx, jadwal, "baru", 1000, testdb.Pax{JamaahID: jBaru})
	testdb.Exec(t, tx, `UPDATE bookings SET customer_account_id = ? WHERE id = ?`, pemesan, b)
	testdb.Exec(t, tx, `UPDATE bookings SET customer_account_id = ? WHERE id = ?`, belumDP, bBaru)

	res, err := tx.ExecContext(ctx, `INSERT INTO manasik (brand_id, judul, tanggal, jam_mulai, jam_selesai, tempat, alamat)
		VALUES (?, 'Manasik Besok Uji', '2099-01-15', '08:00', '12:00', 'Aula Uji', 'Jl. Uji')`, brand)
	if err != nil {
		t.Fatal(err)
	}
	mid, _ := res.LastInsertId()
	testdb.Exec(t, tx, `INSERT INTO manasik_jadwal (schedule_id, manasik_id) VALUES (?, ?)`, jadwal, mid)

	kunci := "manasik-besok:" + strconv.FormatInt(mid, 10)
	jumlah := func(akun int64) int {
		t.Helper()
		var n int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM notifikasi WHERE akun_id = ? AND kunci = ? AND jenis = ?`,
			akun, kunci, JenisManasikBesok).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	// Dua hari sebelum: belum dikirim.
	if err := manasikBesok(ctx, tx, "2099-01-13"); err != nil {
		t.Fatal(err)
	}
	if jumlah(pemesan) != 0 {
		t.Error("pengingat terkirim terlalu awal")
	}
	// H-1, dua putaran loop: sekali per akun.
	for i := 0; i < 2; i++ {
		if err := manasikBesok(ctx, tx, "2099-01-14"); err != nil {
			t.Fatal(err)
		}
	}
	if jumlah(pemesan) != 1 || jumlah(tertaut) != 1 || jumlah(belumDP) != 0 || jumlah(tertautBayi) != 0 {
		t.Errorf("pengingat H-1: pemesan %d, tertaut %d, belum DP %d, bayi %d",
			jumlah(pemesan), jumlah(tertaut), jumlah(belumDP), jumlah(tertautBayi))
	}
	var isi string
	var ref int64
	if err := tx.QueryRowContext(ctx, `SELECT isi, ref_id FROM notifikasi WHERE akun_id = ? AND kunci = ?`, pemesan, kunci).Scan(&isi, &ref); err != nil {
		t.Fatal(err)
	}
	if isi != "Manasik Besok Uji · 08.00 · Aula Uji" || ref != b {
		t.Errorf("isi %q ref %d", isi, ref)
	}
}


