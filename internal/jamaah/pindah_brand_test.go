package jamaah

import (
	"context"
	"testing"

	"erp-azhan/api/internal/testdb"
)

// Pindah brand jamaah ditolak selama dokumen atau pengajuan perubahan data masih menunggu diperiksa
// admin brand lama (audit akses 2026-11-26, T7).
func TestPindahBrandTungguPemeriksaan(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	_, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	cek := func(id int64) string {
		t.Helper()
		msg, err := pesanMenungguPeriksa(ctx, tx, id)
		if err != nil {
			t.Fatal(err)
		}
		return msg
	}
	bersih := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "PindahBersihQzk"})
	if msg := cek(bersih); msg != "" {
		t.Fatalf("tanpa yang menunggu ditolak: %q", msg)
	}
	dok := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "PindahDokumenQzk"})
	testdb.Exec(t, tx, `INSERT INTO dokumen_jamaah (jamaah_id, jenis, file_url, status) VALUES (?, 'paspor', '/uploads/uji.pdf', 'submitted')`, dok)
	if msg := cek(dok); msg == "" {
		t.Fatal("dokumen menunggu: pindah brand tidak ditolak")
	}
	ubah := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "PindahUbahQzk"})
	testdb.Exec(t, tx, `INSERT INTO jamaah_perubahan (jamaah_id, sumber, status) VALUES (?, 'pelanggan', 'menunggu')`, ubah)
	if msg := cek(ubah); msg == "" {
		t.Fatal("perubahan menunggu: pindah brand tidak ditolak")
	}
}
