package customer

import (
	"context"
	"testing"

	"erp-azhan/api/internal/testdb"
)

// Anonimkan akun ikut menghapus Tabungan Umroh dan token push (baris akun tidak dihapus, jadi CASCADE tidak jalan).
func TestAnonimkanAkunMenghapusTabungan(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	_, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-03-01"})
	akun := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AkunTabunganQzp"})
	testdb.Exec(t, tx, `INSERT INTO tabungan_target (akun_id, jenis, bulan_berangkat, jumlah_dewasa, tipe_kamar, target_total)
		VALUES (?, 'perkiraan', '2027-08', 1, 'Quad', 20000000)`, akun)
	testdb.Exec(t, tx, `INSERT INTO tabungan_catatan (akun_id, jumlah, tanggal) VALUES (?, 1000000, '2026-11-01')`, akun)
	testdb.Exec(t, tx, `INSERT INTO perangkat_push (akun_id, token, platform) VALUES (?, 'ExponentPushToken[hapusakunqzp]', 'ios')`, akun)

	if err := anonimkanAkun(ctx, tx, akun); err != nil {
		t.Fatal(err)
	}
	for _, tabel := range []string{"tabungan_target", "tabungan_catatan", "perangkat_push"} {
		var n int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM `+tabel+` WHERE akun_id = ?`, akun).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 0 {
			t.Errorf("%s: %d baris tersisa setelah anonim", tabel, n)
		}
	}
}
