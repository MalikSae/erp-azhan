package schedule

import (
	"context"
	"database/sql"
	"testing"

	"erp-azhan/api/internal/testdb"
)

// Klasifikasi paket disimpan dalam transaksi yang sama dengan penyimpanan paket dan dibaca kembali bersama
// tier dari kategori. Memakai data dev dan memulihkan semua yang diubah lewat t.Cleanup.
func TestKlasifikasiDisimpanDanDibaca(t *testing.T) {
	db := testdb.DB(t)
	testdb.Serialize(t, db, "schedule-klasifikasi")
	ctx := context.Background()
	repo := NewRepository(db)

	var schedID int64
	var catID sql.NullInt64
	var oldJourney string
	if err := db.QueryRowContext(ctx, `
		SELECT id, category_id, journey_type FROM schedules ORDER BY id LIMIT 1`).
		Scan(&schedID, &catID, &oldJourney); err != nil {
		t.Skipf("tidak ada paket untuk fixture: %v", err)
	}
	var oldTier sql.NullString
	if catID.Valid {
		_ = db.QueryRowContext(ctx, `SELECT seo_tier FROM package_categories WHERE id=?`, catID.Int64).Scan(&oldTier)
	}

	t.Cleanup(func() {
		_, _ = db.Exec(`UPDATE schedules SET journey_type=? WHERE id=?`, oldJourney, schedID)
		if catID.Valid {
			_, _ = db.Exec(`UPDATE package_categories SET seo_tier=? WHERE id=?`, oldTier, catID.Int64)
		}
	})

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := setKlasifikasiTx(ctx, tx, schedID, &KlasifikasiInput{JourneyType: JourneyUmroh}); err != nil {
		_ = tx.Rollback()
		t.Fatalf("setKlasifikasiTx: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if catID.Valid {
		if _, err := db.ExecContext(ctx, `UPDATE package_categories SET seo_tier='reguler' WHERE id=?`, catID.Int64); err != nil {
			t.Fatalf("set seo_tier: %v", err)
		}
	}

	got, err := repo.loadKlasifikasi(ctx, []int64{schedID})
	if err != nil {
		t.Fatalf("loadKlasifikasi: %v", err)
	}
	k, ok := got[schedID]
	if !ok {
		t.Fatal("klasifikasi paket tidak ditemukan")
	}
	if k.JourneyType != JourneyUmroh {
		t.Errorf("journey salah: %+v", k)
	}
	if catID.Valid && (k.Tier == nil || k.Tier.SEO == nil || *k.Tier.SEO != "reguler") {
		t.Errorf("tier salah: %+v", k.Tier)
	}
}
