package dokumen

import (
	"context"
	"erp-azhan/api/internal/testdb"
	"errors"
	"strings"
	"testing"
)

func TestReplacementCannotBeApprovedWithOldVersion(t *testing.T) {
	db := testdb.Portal(t)
	ctx := context.Background()
	if _, err := db.Exec(`INSERT INTO jamaah VALUES(1,1,'hash')`); err != nil {
		t.Fatal(err)
	}
	repo := NewRepository(db)
	first, err := repo.Upsert(ctx, 1, &CreateDokumenRequest{Jenis: "paspor", FileURL: "/synthetic-a.pdf"})
	if err != nil {
		t.Fatal(err)
	}
	rejected, err := repo.UpdateStatus(ctx, first.ID, "rejected", first.Version, "Halaman identitas terpotong")
	if err != nil {
		t.Fatal(err)
	}
	if rejected.RejectionReason == nil {
		t.Fatal("rejection reason missing")
	}
	if _, err = repo.Upsert(ctx, 1, &CreateDokumenRequest{Jenis: "paspor", FileURL: strings.Repeat("x", 600)}); err == nil {
		t.Fatal("oversized replacement unexpectedly succeeded")
	}
	var partial int
	if err = db.QueryRow(`SELECT COUNT(*) FROM dokumen_jamaah_versions WHERE dokumen_id=?`, first.ID).Scan(&partial); err != nil || partial != 0 {
		t.Fatal("failed replacement left partial history", err)
	}
	unchanged, err := repo.GetByID(ctx, first.ID, nil)
	if err != nil || unchanged.Version != first.Version || unchanged.Status != "rejected" {
		t.Fatal("failed replacement changed original document", err)
	}
	t.Log("failed replacement rolled back: history rows=0; original version=1; status=rejected")
	replacement, err := repo.Upsert(ctx, 1, &CreateDokumenRequest{Jenis: "paspor", FileURL: "/synthetic-b.pdf"})
	if err != nil {
		t.Fatal(err)
	}
	if replacement.Version != 2 || replacement.Status != "submitted" || replacement.RejectionReason != nil {
		t.Fatalf("invalid replacement: %+v", replacement)
	}
	if _, err = repo.UpdateStatus(ctx, first.ID, "approved", first.Version, ""); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("stale approval: %v", err)
	}
	current, err := repo.GetByID(ctx, first.ID, nil)
	if err != nil || current.Status != "submitted" {
		t.Fatalf("new version approved without review: %v", err)
	}
	var history int
	if err = db.QueryRow(`SELECT COUNT(*) FROM dokumen_jamaah_versions WHERE dokumen_id=? AND version=1 AND status='rejected' AND rejection_reason='Halaman identitas terpotong'`, first.ID).Scan(&history); err != nil || history != 1 {
		t.Fatal("prior version not retained", err)
	}
	if _, err = repo.UpdateStatus(ctx, first.ID, "approved", replacement.Version, ""); err != nil {
		t.Fatal(err)
	}
	t.Log("stale version rejected; replacement remains submitted; prior rejection retained; current version approved")
}
