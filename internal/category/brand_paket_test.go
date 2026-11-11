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

// Code review Paket dan master, 11 November 2026: is_active yang tidak dikirim tidak mengubah status kategori.
func TestIsActiveKosongTidakMengaktifkanUlang(t *testing.T) {
	_, tx := testdb.Tx(t)
	res, err := tx.Exec(`INSERT INTO package_categories (name, slug, is_active) VALUES ('Kategori Uji Aktif', 'kategori-uji-aktif', FALSE)`)
	if err != nil {
		t.Fatalf("buat kategori: %v", err)
	}
	kat, _ := res.LastInsertId()
	simpan := func(ubah, nilai bool) bool {
		t.Helper()
		// Pernyataan yang sama dengan Repository.Update untuk kolom is_active.
		testdb.Exec(t, tx, `UPDATE package_categories SET is_active = IF(?, ?, is_active) WHERE id = ?`, ubah, nilai, kat)
		var aktif bool
		if err := tx.QueryRow(`SELECT is_active FROM package_categories WHERE id=?`, kat).Scan(&aktif); err != nil {
			t.Fatal(err)
		}
		return aktif
	}
	if simpan(false, false) {
		t.Fatal("tanpa is_active, kategori nonaktif harus tetap nonaktif")
	}
	if !simpan(true, true) {
		t.Fatal("is_active=true harus mengaktifkan")
	}
	if !simpan(false, false) {
		t.Fatal("tanpa is_active, kategori aktif harus tetap aktif")
	}
}
