package jamaah

import (
	"context"
	"testing"

	"erp-azhan/api/internal/testdb"
)

// S-02: persentase dokumen dihitung dari semua 7 jenis dan tidak pernah melewati 100.
func TestPctDokumen(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	_, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	id := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "DokumenQzk"})
	pct := func() int {
		t.Helper()
		var v int
		if err := tx.QueryRowContext(ctx, `SELECT `+pctDokumenSQL+` FROM jamaah WHERE id=?`, id).Scan(&v); err != nil {
			t.Fatal(err)
		}
		return v
	}
	tambah := func(jenis, status string) {
		testdb.Exec(t, tx, `INSERT INTO dokumen_jamaah (jamaah_id, jenis, file_url, status) VALUES (?,?,?,?)`, id, jenis, "/uploads/uji.pdf", status)
	}
	if got := pct(); got != 0 {
		t.Fatalf("tanpa dokumen = %d, want 0", got)
	}
	tambah("paspor", "rejected") // ditolak tidak dihitung
	if got := pct(); got != 0 {
		t.Fatalf("ditolak = %d, want 0", got)
	}
	tambah("pas_foto", "submitted")
	if got := pct(); got != 14 {
		t.Fatalf("1 dari 7 = %d, want 14", got)
	}
	testdb.Exec(t, tx, `UPDATE dokumen_jamaah SET status='approved' WHERE jamaah_id=? AND jenis='paspor'`, id)
	for _, jenis := range []string{"ktp", "kk", "akte_lahir", "vaksin_meningitis"} {
		tambah(jenis, "approved")
	}
	if got := pct(); got != 86 {
		t.Fatalf("6 dari 7 = %d, want 86 (dulu 100 karena dibagi 6)", got)
	}
	tambah("buku_nikah", "approved")
	if got := pct(); got != 100 {
		t.Fatalf("7 dari 7 = %d, want 100 (dulu 117)", got)
	}
}

// Daftar jamaah harus bisa dijalankan terhadap skema nyata (kueri hanya membaca).
func TestListJamaahSmoke(t *testing.T) {
	db := testdb.DB(t)
	items, err := NewRepository(db).List(context.Background(), nil, "", ListOpts{Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range items {
		if it.PctDokumen < 0 || it.PctDokumen > 100 {
			t.Errorf("jamaah %d: pct_dokumen = %d", it.ID, it.PctDokumen)
		}
	}
}
