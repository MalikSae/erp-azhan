package benefit

import (
	"context"
	"testing"

	"erp-azhan/api/internal/testdb"
)

// Form paket: teks yang sama dengan benefit aktif (tanpa beda huruf besar-kecil) memakai benefit itu;
// benefit nonaktif tidak ikut ditawarkan.
func TestAktifBerteks(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	var brand int64
	if err := tx.QueryRow(`SELECT id FROM brands ORDER BY id LIMIT 1`).Scan(&brand); err != nil {
		t.Fatal(err)
	}
	aktif, err := create(ctx, tx, brand, "Uji Kembar Qzp", ikonContoh)
	if err != nil {
		t.Fatal(err)
	}
	b, err := aktifBerteks(ctx, tx, brand, "  uji KEMBAR qzp ")
	if err != nil || b == nil || b.ID != aktif.ID {
		t.Fatalf("benefit aktif tidak ditemukan: %+v err=%v", b, err)
	}
	testdb.Exec(t, tx, `UPDATE paket_benefits SET is_active=0 WHERE id=?`, aktif.ID)
	if b, err := aktifBerteks(ctx, tx, brand, "Uji Kembar Qzp"); err != nil || b != nil {
		t.Fatalf("benefit nonaktif tidak boleh dipakai: %+v err=%v", b, err)
	}
	if b, err := aktifBerteks(ctx, tx, brand+100000, "Uji Kembar Qzp"); err != nil || b != nil {
		t.Fatalf("brand lain: %+v err=%v", b, err)
	}
}
