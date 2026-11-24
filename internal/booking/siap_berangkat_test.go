package booking

import (
	"context"
	"testing"

	"erp-azhan/api/internal/testdb"
)

// S-03: kelengkapan pax untuk daftar booking mengikuti tahapan umroh (keputusan 2026-11-24): perlengkapan diserahkan
// (bukan untuk infant), paspor disetujui, SISKOPATUH, dan visa. Vaksin dan manasik tidak lagi dihitung.
func TestPaxBelumLengkapSet(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	b := testdb.NewBooking(t, tx, testdb.BookingOpts{Status: "dp", TotalHarga: 30_000_000, MinimalDP: 5_000_000, Blocked: true})
	belum := func() bool {
		t.Helper()
		set, err := paxBelumLengkapSet(ctx, tx, []int64{b.ID})
		if err != nil {
			t.Fatal(err)
		}
		return set[b.ID]
	}
	if !belum() {
		t.Fatal("pax tanpa progress harus terhitung belum lengkap")
	}
	testdb.Exec(t, tx, `UPDATE booking_pax SET progress_visa=TRUE, progress_siskopatuh=TRUE, perlengkapan_status='sudah_diberikan' WHERE id=?`, b.PaxID)
	if !belum() {
		t.Fatal("tanpa dokumen paspor harus tetap belum lengkap")
	}
	testdb.Exec(t, tx, `INSERT INTO dokumen_jamaah (jamaah_id, jenis, file_url, status) VALUES (?, 'paspor', '/uploads/uji.pdf', 'submitted')`, b.JamaahID)
	if !belum() {
		t.Fatal("paspor yang belum disetujui harus tetap belum lengkap")
	}
	testdb.Exec(t, tx, `UPDATE dokumen_jamaah SET status='approved' WHERE jamaah_id=? AND jenis='paspor'`, b.JamaahID)
	if belum() {
		t.Fatal("perlengkapan, paspor disetujui, SISKOPATUH, dan visa harus terhitung lengkap tanpa vaksin dan manasik")
	}
	testdb.Exec(t, tx, `UPDATE booking_pax SET perlengkapan_status='belum_diberikan' WHERE id=?`, b.PaxID)
	if !belum() {
		t.Fatal("reguler tanpa perlengkapan harus belum lengkap")
	}
	testdb.Exec(t, tx, `UPDATE booking_pax SET pax_type='infant', counts_for_seat=FALSE, room_type=NULL WHERE id=?`, b.PaxID)
	if belum() {
		t.Fatal("infant tanpa perlengkapan harus terhitung lengkap")
	}
	// Pax batal tidak dihitung.
	testdb.Exec(t, tx, `UPDATE booking_pax SET progress_visa=FALSE, pax_status='batal' WHERE id=?`, b.PaxID)
	if belum() {
		t.Fatal("pax batal tidak boleh membuat booking belum lengkap")
	}
	if set, err := paxBelumLengkapSet(ctx, tx, nil); err != nil || len(set) != 0 {
		t.Fatalf("daftar kosong: %v %v", set, err)
	}
}

// Daftar booking harus bisa dijalankan terhadap skema nyata (kueri hanya membaca), dan siap_berangkat
// di daftar harus sama dengan yang dihitung halaman detail.
func TestListBookingSiapBerangkatSamaDenganDetail(t *testing.T) {
	db := testdb.DB(t)
	repo := NewRepository(db)
	ctx := context.Background()
	items, err := repo.List(ctx, nil, nil, "non_draft", ListOpts{Limit: 40})
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range items {
		detail, err := repo.GetByID(ctx, it.ID, nil)
		if err != nil {
			t.Fatalf("booking %d: %v", it.ID, err)
		}
		if detail.SiapBerangkat != it.SiapBerangkat {
			t.Errorf("booking %d: daftar=%v detail=%v", it.ID, it.SiapBerangkat, detail.SiapBerangkat)
		}
	}
	t.Logf("dibandingkan %d booking", len(items))
}
