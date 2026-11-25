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
	rejected, err := repo.UpdateStatus(ctx, first.ID, "rejected", first.Version, "Halaman identitas terpotong", 0)
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
	if _, err = repo.UpdateStatus(ctx, first.ID, "approved", first.Version, "", 0); !errors.Is(err, ErrVersionConflict) {
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
	if _, err = repo.UpdateStatus(ctx, first.ID, "approved", replacement.Version, "", 0); err != nil {
		t.Fatal(err)
	}
	t.Log("stale version rejected; replacement remains submitted; prior rejection retained; current version approved")
}

func TestJejakVerifikasiDanBatalkanPersetujuan(t *testing.T) {
	db := testdb.Portal(t)
	ctx := context.Background()
	if _, err := db.Exec(`INSERT INTO jamaah VALUES(1,1,'hash')`); err != nil {
		t.Fatal(err)
	}
	var adminID int64
	var adminNama string
	if err := db.QueryRow(`SELECT id, COALESCE(NULLIF(display_name,''), email) FROM admin_users ORDER BY id LIMIT 1`).Scan(&adminID, &adminNama); err != nil {
		t.Skip("butuh minimal satu admin_users:", err)
	}
	repo := NewRepository(db)
	first, err := repo.Upsert(ctx, 1, &CreateDokumenRequest{Jenis: "kk", FileURL: "/synthetic-kk-a.pdf"})
	if err != nil {
		t.Fatal(err)
	}
	if first.VerifiedBy != nil || first.VerifiedAt != nil {
		t.Fatalf("dokumen baru sudah punya peninjau: %+v", first)
	}
	if _, err = repo.UpdateStatus(ctx, first.ID, "submitted", first.Version, "", adminID); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("batalkan persetujuan dari submitted: %v", err)
	}
	approved, err := repo.UpdateStatus(ctx, first.ID, "approved", first.Version, "", adminID)
	if err != nil {
		t.Fatal(err)
	}
	if approved.VerifiedBy == nil || *approved.VerifiedBy != adminID || approved.VerifiedAt == nil || approved.VerifiedByNama == nil || *approved.VerifiedByNama != adminNama {
		t.Fatalf("peninjau tidak tercatat: %+v", approved)
	}
	t.Logf("approved: status=%s verified_by=%d verified_by_nama=%q verified_at=%v", approved.Status, *approved.VerifiedBy, *approved.VerifiedByNama, *approved.VerifiedAt)
	batal, err := repo.UpdateStatus(ctx, first.ID, "submitted", approved.Version, "", adminID)
	if err != nil {
		t.Fatal(err)
	}
	if batal.Status != "submitted" || batal.VerifiedBy == nil {
		t.Fatalf("batalkan persetujuan: %+v", batal)
	}
	rejected, err := repo.UpdateStatus(ctx, first.ID, "rejected", batal.Version, "Buram", adminID)
	if err != nil {
		t.Fatal(err)
	}
	replacement, err := repo.Upsert(ctx, 1, &CreateDokumenRequest{Jenis: "kk", FileURL: "/synthetic-kk-b.pdf"})
	if err != nil {
		t.Fatal(err)
	}
	if replacement.VerifiedBy != nil || replacement.VerifiedAt != nil || replacement.Status != "submitted" || replacement.Version != 2 {
		t.Fatalf("pengganti mewarisi peninjau: %+v", replacement)
	}
	versi, err := repo.ListVersi(ctx, first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(versi) != 1 || versi[0].Version != 1 || versi[0].Status != "rejected" || versi[0].RejectionReason == nil || *versi[0].RejectionReason != "Buram" ||
		versi[0].VerifiedByNama == nil || *versi[0].VerifiedByNama != adminNama || versi[0].VerifiedAt == nil || !versi[0].VerifiedAt.Equal(*rejected.VerifiedAt) {
		t.Fatalf("riwayat versi salah: %+v", versi)
	}
	t.Logf("riwayat: version=%d status=%s alasan=%q oleh=%q pada=%v", versi[0].Version, versi[0].Status, *versi[0].RejectionReason, *versi[0].VerifiedByNama, *versi[0].VerifiedAt)
	kosong, err := repo.ListVersi(ctx, 999999)
	if err != nil || kosong == nil || len(kosong) != 0 {
		t.Fatalf("riwayat kosong harus []: %v %v", kosong, err)
	}
}
