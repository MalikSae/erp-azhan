package agen

import (
	"context"
	"database/sql"
	"testing"

	"erp-azhan/api/internal/testdb"
)

// IkatAkunDariKunciTx: upline akun baru = pemilik kunci jamaah yang ditautkan (aktivasi akun jamaah).
func TestIkatAkunDariKunciTx(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	_, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	K := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "KunciAktif", StatusAgen: "aktif"})
	Kn := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "KunciNonaktif", StatusAgen: "nonaktif"})
	Z := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenLain", StatusAgen: "aktif"})

	direkrutDan := func(akun int64) (direkrut sql.NullInt64, log int) {
		t.Helper()
		if err := tx.QueryRowContext(ctx, `SELECT direkrut_oleh_akun_id FROM customer_accounts WHERE id=?`, akun).Scan(&direkrut); err != nil {
			t.Fatal(err)
		}
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM akun_kaitan_log WHERE akun_id=? AND alasan=?`, akun, AlasanIkatAkunDariKunci).Scan(&log); err != nil {
			t.Fatal(err)
		}
		return
	}

	t.Run("direkrut kosong diisi dan dicatat tanpa admin", func(t *testing.T) {
		j := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "J1", Direkrut: K})
		akun := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "A1"})
		if err := IkatAkunDariKunciTx(ctx, tx, akun, j); err != nil {
			t.Fatal(err)
		}
		if d, n := direkrutDan(akun); d.Int64 != K || n != 1 {
			t.Fatalf("direkrut=%v log=%d", d, n)
		}
		var oleh sql.NullInt64
		_ = tx.QueryRowContext(ctx, `SELECT diganti_oleh FROM akun_kaitan_log WHERE akun_id=?`, akun).Scan(&oleh)
		if oleh.Valid {
			t.Fatal("log harus oleh sistem")
		}
		var up sql.NullInt64
		_ = tx.QueryRowContext(ctx, `SELECT upline_akun_id FROM customer_accounts WHERE id=?`, akun).Scan(&up)
		if up.Valid {
			t.Fatal("upline_akun_id tidak boleh diisi")
		}
	})
	t.Run("pemilik kunci nonaktif tetap mengikat", func(t *testing.T) {
		j := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "J2", Direkrut: Kn})
		akun := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "A2"})
		if err := IkatAkunDariKunciTx(ctx, tx, akun, j); err != nil {
			t.Fatal(err)
		}
		if d, _ := direkrutDan(akun); d.Int64 != Kn {
			t.Fatalf("direkrut=%v", d)
		}
	})
	t.Run("direkrut sudah terisi: tidak berubah dan tanpa log", func(t *testing.T) {
		j := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "J3", Direkrut: K})
		akun := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "A3", Direkrut: Z})
		if err := IkatAkunDariKunciTx(ctx, tx, akun, j); err != nil {
			t.Fatal(err)
		}
		if d, n := direkrutDan(akun); d.Int64 != Z || n != 0 {
			t.Fatalf("direkrut=%v log=%d", d, n)
		}
	})
	t.Run("pemilik kunci bukan agen: tidak mengikat", func(t *testing.T) {
		biasa := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "KunciBiasa"})
		j := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "J4", Direkrut: biasa})
		akun := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "A4"})
		if err := IkatAkunDariKunciTx(ctx, tx, akun, j); err != nil {
			t.Fatal(err)
		}
		if d, n := direkrutDan(akun); d.Valid || n != 0 {
			t.Fatalf("direkrut=%v log=%d", d, n)
		}
	})
	t.Run("tanpa_agen admin dan jamaah baru: tidak ada upline", func(t *testing.T) {
		akun := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "A6"})
		for _, o := range []testdb.JamaahOpts{{Nama: "J6a", TanpaAgen: true}, {Nama: "J6b"}} {
			j := testdb.NewJamaah(t, tx, brand, o)
			if err := IkatAkunDariKunciTx(ctx, tx, akun, j); err != nil {
				t.Fatal(err)
			}
		}
		if d, n := direkrutDan(akun); d.Valid || n != 0 {
			t.Fatalf("direkrut=%v log=%d", d, n)
		}
	})
	t.Run("pemilik kunci beda brand: tidak mengikat", func(t *testing.T) {
		var brandLain int64
		if err := tx.QueryRowContext(ctx, `SELECT id FROM brands WHERE id <> ? ORDER BY id LIMIT 1`, brand).Scan(&brandLain); err != nil {
			t.Skip("hanya satu brand")
		}
		kLain := testdb.NewAkun(t, tx, brandLain, testdb.AkunOpts{Nama: "KunciBrandLain", StatusAgen: "aktif"})
		j := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "J8", Direkrut: kLain})
		akun := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "A8"})
		if err := IkatAkunDariKunciTx(ctx, tx, akun, j); err != nil {
			t.Fatal(err)
		}
		if d, n := direkrutDan(akun); d.Valid || n != 0 {
			t.Fatalf("direkrut=%v log=%d", d, n)
		}
	})
}
