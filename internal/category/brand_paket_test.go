package category

import (
	"context"
	"testing"

	"erp-azhan/api/internal/testdb"
)

// Audit menu Paket S-08: brand tidak boleh dilepas dari kategori selama paket brand itu masih memakainya.
func TestPaketDiLuarBrand(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	sched, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})

	res, err := tx.Exec(`INSERT INTO package_categories (name, slug, is_active) VALUES ('Kategori Uji S-08', 'kategori-uji-s-08', TRUE)`)
	if err != nil {
		t.Fatalf("buat kategori: %v", err)
	}
	kat, _ := res.LastInsertId()
	testdb.Exec(t, tx, `UPDATE schedules SET category_id=? WHERE id=?`, kat, sched)

	cek := func(label string, brandIDs []int64, mau int) {
		t.Helper()
		n, err := paketDiLuarBrandTx(ctx, tx, kat, brandIDs)
		if err != nil {
			t.Fatalf("%s: %v", label, err)
		}
		if n != mau {
			t.Fatalf("%s: dapat %d, mau %d", label, n, mau)
		}
	}
	cek("semua brand", nil, 0)
	cek("brand paket tetap ada", []int64{brand}, 0)
	cek("brand paket tetap ada bersama brand lain", []int64{brand, brand + 999_999}, 0)
	cek("brand paket dilepas", []int64{brand + 999_999}, 1)
}
