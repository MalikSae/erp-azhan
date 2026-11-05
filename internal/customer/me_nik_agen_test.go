package customer

import (
	"context"
	"testing"

	"erp-azhan/api/internal/testdb"
)

// /api/account/me memuat NIK agen hanya dalam bentuk tersamar (untuk petunjuk "terisi otomatis" di form booking agen).
func TestMeMemuatNIKAgenTersamar(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	_, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	agen := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenBernik", StatusAgen: "aktif"})
	biasa := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AkunBiasa"})
	testdb.Exec(t, tx, `UPDATE customer_accounts SET nik_agen='9999999999990501', jenis_kelamin='P' WHERE id=?`, agen)

	v := buildMeView(ctx, tx, &Account{ID: agen})
	if v.NIKAgenMasked != MaskNIK("9999999999990501") || v.NIKAgenMasked == "9999999999990501" {
		t.Fatalf("nik_agen_masked = %q", v.NIKAgenMasked)
	}
	if v.JenisKelamin != "P" {
		t.Fatalf("jenis kelamin akun harus terbawa di /me: %q", v.JenisKelamin)
	}
	if v := buildMeView(ctx, tx, &Account{ID: biasa}); v.NIKAgenMasked != "" {
		t.Fatalf("akun tanpa NIK agen harus kosong: %q", v.NIKAgenMasked)
	}
}
