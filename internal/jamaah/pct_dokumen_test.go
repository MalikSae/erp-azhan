package jamaah

import (
	"context"
	"testing"

	"erp-azhan/api/internal/testdb"
)

func TestKelengkapanDokumenHitung(t *testing.T) {
	cases := []struct {
		tipe, csv            string
		setuju, total, pct int
	}{
		{"reguler", "", 0, 3, 0},
		{"reguler", "pas_foto", 1, 3, 33},
		{"reguler", "paspor,pas_foto", 2, 3, 67},
		{"reguler", "paspor,ktp,pas_foto", 3, 3, 100},
		{"reguler", "kk,buku_nikah,vaksin_meningitis,akte_lahir", 0, 3, 0}, // tambahan tidak dihitung
		{"infant", "paspor,akte_lahir,pas_foto", 2, 3, 67},                // akta lahir tidak wajib lagi
		{"infant", "paspor,kk,pas_foto", 3, 3, 100},
	}
	for _, c := range cases {
		s, tot, p := kelengkapanDokumen(c.tipe, c.csv)
		if s != c.setuju || tot != c.total || p != c.pct {
			t.Errorf("%s %q = %d/%d %d%%, want %d/%d %d%%", c.tipe, c.csv, s, tot, p, c.setuju, c.total, c.pct)
		}
	}
}

// D1: kelengkapan dokumen di admin memakai dokumen wajib sesuai tipe pax (booking aktif terakhir), bukan 7 jenis.
func TestKelengkapanDokumenAdmin(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	sched, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	id := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "DokumenQzk"})
	baca := func() (tipe string, setuju, total, pct int) {
		t.Helper()
		var csv string
		if err := tx.QueryRowContext(ctx, `SELECT `+tipeDokumenSQL+`, `+dokumenDisetujuiSQL+` FROM jamaah WHERE id=?`, id).Scan(&tipe, &csv); err != nil {
			t.Fatal(err)
		}
		setuju, total, pct = kelengkapanDokumen(tipe, csv)
		return
	}
	tambah := func(jenis, status string) {
		testdb.Exec(t, tx, `INSERT INTO dokumen_jamaah (jamaah_id, jenis, file_url, status) VALUES (?,?,?,?)`, id, jenis, "/uploads/uji.pdf", status)
	}
	if tipe, s, tot, p := baca(); tipe != "reguler" || s != 0 || tot != 3 || p != 0 {
		t.Fatalf("tanpa booking = %s %d/%d %d", tipe, s, tot, p)
	}
	tambah("paspor", "approved")
	tambah("ktp", "submitted") // menunggu verifikasi tidak dihitung
	tambah("pas_foto", "approved")
	tambah("kk", "approved")
	if _, s, _, p := baca(); s != 2 || p != 67 {
		t.Fatalf("reguler 2 wajib disetujui = %d %d%%", s, p)
	}
	testdb.Exec(t, tx, `UPDATE dokumen_jamaah SET status='approved' WHERE jamaah_id=? AND jenis='ktp'`, id)
	if _, s, tot, p := baca(); s != 3 || tot != 3 || p != 100 {
		t.Fatalf("reguler lengkap tanpa buku nikah = %d/%d %d%% (dulu 57%% dari 7 jenis)", s, tot, p)
	}

	// Booking aktif terakhir sebagai bayi: wajib paspor, KK, pas foto (KTP tidak dihitung).
	b, pax := testdb.NewBookingPax(t, tx, sched, "dp", 1000, testdb.Pax{JamaahID: id})
	testdb.Exec(t, tx, `UPDATE booking_pax SET pax_type='infant' WHERE id=?`, pax[0])
	testdb.Exec(t, tx, `DELETE FROM dokumen_jamaah WHERE jamaah_id=? AND jenis='kk'`, id)
	tambah("akte_lahir", "approved")
	if tipe, s, _, p := baca(); tipe != "infant" || s != 2 || p != 67 {
		t.Fatalf("bayi tanpa KK = %s %d %d%%", tipe, s, p)
	}
	tambah("kk", "approved")
	if _, s, _, p := baca(); s != 3 || p != 100 {
		t.Fatalf("bayi lengkap = %d %d%%", s, p)
	}
	// Booking batal tidak dipakai: kembali ke daftar reguler.
	testdb.Exec(t, tx, `UPDATE bookings SET status='batal' WHERE id=?`, b)
	if tipe, _, _, _ := baca(); tipe != "reguler" {
		t.Fatalf("booking batal: tipe=%s, want reguler", tipe)
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
		if it.PctDokumen < 0 || it.PctDokumen > 100 || it.DokumenWajibTotal != 3 {
			t.Errorf("jamaah %d: pct_dokumen = %d, wajib = %d/%d", it.ID, it.PctDokumen, it.DokumenWajibDisetujui, it.DokumenWajibTotal)
		}
	}
}
