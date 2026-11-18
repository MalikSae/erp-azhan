package agen

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"erp-azhan/api/internal/testdb"
)

// IkatAkunDariKunciTx: upline akun = pemilik kunci jamaah yang ditautkan (aktivasi akun jamaah).
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
	t.Run("direkrut sama: tanpa perubahan dan tanpa log", func(t *testing.T) {
		j := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "J3", Direkrut: K})
		akun := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "A3", Direkrut: K})
		if err := IkatAkunDariKunciTx(ctx, tx, akun, j); err != nil {
			t.Fatal(err)
		}
		if d, n := direkrutDan(akun); d.Int64 != K || n != 0 {
			t.Fatalf("direkrut=%v log=%d", d, n)
		}
	})
	t.Run("direkrut berbeda: bentrok, akun tidak berubah", func(t *testing.T) {
		j := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "J4", Direkrut: K})
		akun := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "A4", Direkrut: Z})
		if err := IkatAkunDariKunciTx(ctx, tx, akun, j); !errors.Is(err, ErrUplineBentrok) {
			t.Fatalf("err = %v", err)
		}
		if d, n := direkrutDan(akun); d.Int64 != Z || n != 0 {
			t.Fatalf("direkrut=%v log=%d", d, n)
		}
	})
	t.Run("akun agen aktif: wajib upline sama dengan pemilik kunci", func(t *testing.T) {
		j := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "J5", Direkrut: K})
		beda := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenBeda", StatusAgen: "aktif", Direkrut: Z, Upline: Z})
		if err := IkatAkunDariKunciTx(ctx, tx, beda, j); !errors.Is(err, ErrUplineBentrok) {
			t.Fatalf("beda: %v", err)
		}
		sama := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenSama", StatusAgen: "aktif", Direkrut: K, Upline: K})
		if err := IkatAkunDariKunciTx(ctx, tx, sama, j); err != nil {
			t.Fatalf("sama: %v", err)
		}
		tanpaUpline := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenTanpaUpline", StatusAgen: "aktif"})
		if err := IkatAkunDariKunciTx(ctx, tx, tanpaUpline, j); !errors.Is(err, ErrUplineBentrok) {
			t.Fatalf("tanpa upline: %v", err)
		}
	})
	t.Run("tanpa_agen dan belum_ditentukan: tidak ada upline", func(t *testing.T) {
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
	t.Run("pemilik kunci menautkan jamaah orang lain ditolak", func(t *testing.T) {
		j := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "J7", Direkrut: K})
		if err := IkatAkunDariKunciTx(ctx, tx, K, j); !errors.Is(err, ErrUplineBentrok) {
			t.Fatalf("err = %v", err)
		}
		if d, _ := direkrutDan(K); d.Valid {
			t.Fatalf("direkrut = %v", d)
		}
	})
	t.Run("pemilik kunci menautkan jamaah miliknya sendiri lolos", func(t *testing.T) {
		const nik = "9999999999990088"
		j := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "J7b", Direkrut: K, NIK: nik})
		testdb.Exec(t, tx, `UPDATE customer_accounts SET nik_agen=? WHERE id=?`, nik, K)
		if err := IkatAkunDariKunciTx(ctx, tx, K, j); err != nil {
			t.Fatal(err)
		}
		if d, _ := direkrutDan(K); d.Valid {
			t.Fatalf("direkrut = %v", d)
		}
		testdb.Exec(t, tx, `UPDATE customer_accounts SET nik_agen=NULL WHERE id=?`, K)
	})
	t.Run("pemilik kunci beda brand ditolak", func(t *testing.T) {
		var brandLain int64
		if err := tx.QueryRowContext(ctx, `SELECT id FROM brands WHERE id <> ? ORDER BY id LIMIT 1`, brand).Scan(&brandLain); err != nil {
			t.Skip("hanya satu brand")
		}
		kLain := testdb.NewAkun(t, tx, brandLain, testdb.AkunOpts{Nama: "KunciBrandLain", StatusAgen: "aktif"})
		j := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "J8", Direkrut: kLain})
		akun := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "A8"})
		if err := IkatAkunDariKunciTx(ctx, tx, akun, j); !errors.Is(err, ErrUplineBentrok) {
			t.Fatalf("err = %v", err)
		}
	})
}
