package schedule

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"erp-azhan/api/internal/testdb"
)

// Klasifikasi paket disimpan dalam transaksi yang sama dengan penyimpanan paket, dibaca kembali bersama
// tier dari kategori, dan menghitung destinasi yang sedang dijual. Memakai data dev dan memulihkan
// semua yang diubah lewat t.Cleanup.
func TestKlasifikasiDisimpanDanDibaca(t *testing.T) {
	db := testdb.DB(t)
	testdb.Serialize(t, db, "schedule-klasifikasi")
	ctx := context.Background()
	repo := NewRepository(db)

	var schedID, brandID int64
	var catID sql.NullInt64
	var oldJourney string
	var oldProgram sql.NullString
	var oldDest sql.NullInt64
	var oldStatus, oldBerangkat string
	if err := db.QueryRowContext(ctx, `
		SELECT id, brand_id, category_id, journey_type, program_slug, destination_id, status,
			DATE_FORMAT(berangkat_tanggal, '%Y-%m-%d')
		FROM schedules ORDER BY id LIMIT 1`).
		Scan(&schedID, &brandID, &catID, &oldJourney, &oldProgram, &oldDest, &oldStatus, &oldBerangkat); err != nil {
		t.Skipf("tidak ada paket untuk fixture: %v", err)
	}

	slug := fmt.Sprintf("uji-%d", time.Now().UnixNano()%1_000_000_000)
	res, err := db.ExecContext(ctx, `INSERT INTO destinations (name, slug) VALUES (?, ?)`, "Destinasi Uji", slug)
	if err != nil {
		t.Fatalf("buat destinasi: %v", err)
	}
	destID, _ := res.LastInsertId()
	var oldTier sql.NullString
	if catID.Valid {
		_ = db.QueryRowContext(ctx, `SELECT seo_tier FROM package_categories WHERE id=?`, catID.Int64).Scan(&oldTier)
	}

	t.Cleanup(func() {
		_, _ = db.Exec(`UPDATE schedules SET journey_type=?, program_slug=?, destination_id=?, status=?, berangkat_tanggal=? WHERE id=?`,
			oldJourney, oldProgram, oldDest, oldStatus, oldBerangkat, schedID)
		if catID.Valid {
			_, _ = db.Exec(`UPDATE package_categories SET seo_tier=? WHERE id=?`, oldTier, catID.Int64)
		}
		_, _ = db.Exec(`DELETE FROM destinations WHERE id=?`, destID)
	})

	// Simpan klasifikasi lewat jalur yang sama dengan Create/Update paket.
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	prog := "plus"
	if err := setKlasifikasiTx(ctx, tx, schedID, &KlasifikasiInput{JourneyType: JourneyUmroh, ProgramSlug: &prog, DestinationID: &destID}); err != nil {
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
	if k.JourneyType != JourneyUmroh || k.ProgramSlug == nil || *k.ProgramSlug != "plus" {
		t.Errorf("journey/program salah: %+v", k)
	}
	if k.Destination == nil || k.Destination.ID != destID || k.Destination.Slug != slug {
		t.Errorf("destinasi salah: %+v", k.Destination)
	}
	if catID.Valid && (k.Tier == nil || k.Tier.SEO == nil || *k.Tier.SEO != "reguler") {
		t.Errorf("tier salah: %+v", k.Tier)
	}

	// fetchFull menempelkan klasifikasi ke Schedule admin.
	s, err := repo.GetByID(ctx, schedID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if s.DestinationID == nil || *s.DestinationID != destID || s.Destination == nil || s.ProgramSlug == nil {
		t.Errorf("Schedule admin tidak memuat klasifikasi: dest=%v prog=%v", s.DestinationID, s.ProgramSlug)
	}

	// Destinasi dihitung hanya bila paketnya sedang dijual (published dan berangkat >= batas).
	const jauh = "2099-12-31"
	if _, err := db.ExecContext(ctx, `UPDATE schedules SET status='published', berangkat_tanggal=? WHERE id=?`, jauh, schedID); err != nil {
		t.Fatal(err)
	}
	hitung := func() int {
		items, err := repo.ListPublicDestinations(ctx, brandID, batasPublikWIB())
		if err != nil {
			t.Fatalf("ListPublicDestinations: %v", err)
		}
		for _, d := range items {
			if d.ID == destID {
				return d.PackageCount
			}
		}
		return 0
	}
	if n := hitung(); n != 1 {
		t.Errorf("paket dijual: package_count = %d, want 1", n)
	}
	if _, err := db.ExecContext(ctx, `UPDATE schedules SET status='draft' WHERE id=?`, schedID); err != nil {
		t.Fatal(err)
	}
	if n := hitung(); n != 0 {
		t.Errorf("paket draft tidak boleh dihitung: package_count = %d", n)
	}

	// Destinasi yang masih dipakai paket tidak boleh dihapus (FK).
	if _, err := db.ExecContext(ctx, `DELETE FROM destinations WHERE id=?`, destID); err == nil {
		t.Error("destinasi yang dipakai paket seharusnya tidak bisa dihapus")
	}

	// Destinasi nonaktif ditolak untuk klasifikasi baru.
	if _, err := db.ExecContext(ctx, `UPDATE destinations SET is_active=FALSE WHERE id=?`, destID); err != nil {
		t.Fatal(err)
	}
	if aktif, err := repo.DestinationActive(ctx, destID); err != nil || aktif {
		t.Errorf("DestinationActive(nonaktif) = %v, %v; want false, nil", aktif, err)
	}
}
