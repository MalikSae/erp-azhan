package perlengkapan

import (
	"context"
	"errors"
	"testing"

	"erp-azhan/api/internal/testdb"
)

// Review Inventory 2026-11-25: set perlengkapan kategori mengikuti aturan brand modul kategori (MP-01, S-08).
func TestCekAksesKategori(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()

	var brandA, brandB int64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM brands ORDER BY id LIMIT 1`).Scan(&brandA); err != nil {
		t.Skipf("tidak ada brand di database uji: %v", err)
	}
	if err := tx.QueryRowContext(ctx, `SELECT id FROM brands WHERE id <> ? ORDER BY id LIMIT 1`, brandA).Scan(&brandB); err != nil {
		t.Skipf("butuh dua brand di database uji: %v", err)
	}
	buat := func(nama string, brands ...int64) int64 {
		res, err := tx.ExecContext(ctx, `INSERT INTO package_categories (name, slug) VALUES (?, ?)`, nama, nama)
		if err != nil {
			t.Fatalf("buat kategori: %v", err)
		}
		id, _ := res.LastInsertId()
		for _, b := range brands {
			testdb.Exec(t, tx, `INSERT INTO category_brands (category_id, brand_id) VALUES (?, ?)`, id, b)
		}
		return id
	}
	khususA := buat("uji-set-khusus-a", brandA)
	bersama := buat("uji-set-bersama", brandA, brandB)
	khususB := buat("uji-set-khusus-b", brandB)

	cases := []struct {
		nama   string
		cat    int64
		brand  *int64
		ubah   bool
		mauErr error
	}{
		{"holding ubah kategori bersama", bersama, nil, true, nil},
		{"brand lihat kategori bersama", bersama, &brandA, false, nil},
		{"brand ubah kategori khusus sendiri", khususA, &brandA, true, nil},
		{"brand ubah kategori bersama", bersama, &brandA, true, ErrKategoriBrandLain},
		{"brand lihat kategori brand lain", khususB, &brandA, false, ErrNotFound},
		{"brand ubah kategori brand lain", khususB, &brandA, true, ErrNotFound},
		{"kategori tidak ada", -1, nil, false, ErrNotFound},
	}
	for _, c := range cases {
		err := cekAksesKategori(ctx, tx, c.cat, c.brand, c.ubah)
		if !errors.Is(err, c.mauErr) || (c.mauErr == nil && err != nil) {
			t.Errorf("%s: err=%v, mau %v", c.nama, err, c.mauErr)
		}
	}
}
