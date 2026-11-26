package dokumen

import (
	"context"
	"errors"
	"testing"

	"erp-azhan/api/internal/testdb"
)

// D2: disetujui/ditolak hanya dari menunggu verifikasi (submitted). Koreksi lewat batalkan persetujuan lalu tolak.
func TestStatusTinjauHanyaDariSubmitted(t *testing.T) {
	db := testdb.Portal(t)
	ctx := context.Background()
	if _, err := db.Exec(`INSERT INTO jamaah VALUES(1,1,'hash')`); err != nil {
		t.Fatal(err)
	}
	repo := NewRepository(db)
	d, err := repo.Upsert(ctx, 1, &CreateDokumenRequest{Jenis: "ktp", FileURL: "/synthetic-ktp.pdf"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repo.UpdateStatus(ctx, d.ID, "rejected", d.Version, "Buram", 0); err != nil {
		t.Fatal(err)
	}
	// Ditolak -> disetujui tanpa berkas baru ditolak.
	if _, err = repo.UpdateStatus(ctx, d.ID, "approved", d.Version, "", 0); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("ditolak -> disetujui: err=%v, want ErrVersionConflict", err)
	}
	// Ditolak -> ditolak lagi juga ditolak.
	if _, err = repo.UpdateStatus(ctx, d.ID, "rejected", d.Version, "Lain", 0); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("ditolak -> ditolak: err=%v, want ErrVersionConflict", err)
	}
	cur, err := repo.GetByID(ctx, d.ID, nil)
	if err != nil || cur.Status != "rejected" || cur.RejectionReason == nil || *cur.RejectionReason != "Buram" {
		t.Fatalf("status berubah: %+v err=%v", cur, err)
	}

	// Disetujui -> ditolak langsung ditolak; harus batalkan dulu.
	d2, err := repo.Upsert(ctx, 1, &CreateDokumenRequest{Jenis: "ktp", FileURL: "/synthetic-ktp-2.pdf"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repo.UpdateStatus(ctx, d2.ID, "approved", d2.Version, "", 0); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.UpdateStatus(ctx, d2.ID, "rejected", d2.Version, "Salah", 0); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("disetujui -> ditolak: err=%v, want ErrVersionConflict", err)
	}
	if _, err = repo.UpdateStatus(ctx, d2.ID, "submitted", d2.Version, "", 0); err != nil {
		t.Fatalf("batalkan persetujuan: %v", err)
	}
	ditolak, err := repo.UpdateStatus(ctx, d2.ID, "rejected", d2.Version, "Salah", 0)
	if err != nil || ditolak.Status != "rejected" {
		t.Fatalf("tolak setelah batalkan: %+v err=%v", ditolak, err)
	}
	t.Logf("ditolak->disetujui: ErrVersionConflict; disetujui->ditolak: ErrVersionConflict; batalkan lalu tolak: status=%s", ditolak.Status)
}
