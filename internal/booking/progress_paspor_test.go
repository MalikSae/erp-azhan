package booking

import (
	"context"
	"testing"

	"erp-azhan/api/internal/testdb"
)

// Keputusan 2026-11-25: progress_paspor (daftar dan detail booking) hanya true bila paspor disetujui admin;
// paspor_terunggah tetap menandai berkas yang sudah ada untuk badge "Perlu dicek".
func TestProgressPasporHanyaDisetujui(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	b := testdb.NewBooking(t, tx, testdb.BookingOpts{Status: "dp", TotalHarga: 30_000_000, MinimalDP: 5_000_000, Blocked: true})
	cek := func(label string, wantSetuju, wantUnggah bool) {
		t.Helper()
		setuju, unggah, err := statusPaspor(ctx, tx, b.JamaahID)
		if err != nil {
			t.Fatal(err)
		}
		set, err := pasporDisetujuiSet(ctx, tx, []int64{b.JamaahID})
		if err != nil {
			t.Fatal(err)
		}
		if setuju != wantSetuju || unggah != wantUnggah || set[b.JamaahID] != wantSetuju {
			t.Fatalf("%s: disetujui=%v terunggah=%v set=%v, want disetujui=%v terunggah=%v", label, setuju, unggah, set[b.JamaahID], wantSetuju, wantUnggah)
		}
	}
	cek("tanpa paspor", false, false)
	testdb.Exec(t, tx, `INSERT INTO dokumen_jamaah (jamaah_id, jenis, file_url, status) VALUES (?, 'paspor', '/uploads/uji.pdf', 'submitted')`, b.JamaahID)
	cek("menunggu verifikasi", false, true)
	testdb.Exec(t, tx, `UPDATE dokumen_jamaah SET status='rejected' WHERE jamaah_id=? AND jenis='paspor'`, b.JamaahID)
	cek("ditolak", false, true)
	testdb.Exec(t, tx, `UPDATE dokumen_jamaah SET status='approved' WHERE jamaah_id=? AND jenis='paspor'`, b.JamaahID)
	cek("disetujui", true, true)
}
