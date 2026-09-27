package agen

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"erp-azhan/api/internal/testdb"
)

const fotoUji = "/api/admin/media/dokumen-jamaah/11111111-2222-3333-4444-555555555555.jpg"

func bukti(n string) string {
	return "/api/admin/media/dokumen-jamaah/00000000-0000-0000-0000-00000000000" + n + ".jpg"
}

type fixture struct {
	tx      *sql.Tx
	brandID int64
	adminID int64
}

func setup(t *testing.T, biaya float64) fixture {
	t.Helper()
	_, tx := testdb.Tx(t)
	_, brandID := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	testdb.Exec(t, tx, `UPDATE brands SET biaya_pendaftaran_agen=? WHERE id=?`, biaya, brandID)
	var adminID int64
	if err := tx.QueryRow(`SELECT id FROM admin_users ORDER BY id LIMIT 1`).Scan(&adminID); err != nil {
		t.Fatalf("admin: %v", err)
	}
	return fixture{tx: tx, brandID: brandID, adminID: adminID}
}

func (f fixture) status(t *testing.T, jamaahID int64) *StatusPortal {
	t.Helper()
	s, err := getStatusPortal(context.Background(), f.tx, jamaahID)
	if err != nil {
		t.Fatalf("getStatusPortal: %v", err)
	}
	return s
}

func must(t *testing.T, label string, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", label, err)
	}
}

func mustErr(t *testing.T, label string, err, want error) {
	t.Helper()
	if !errors.Is(err, want) {
		t.Fatalf("%s: err=%v, want %v", label, err, want)
	}
}

var ajuan = AjukanRequest{FotoAgenURL: fotoUji, Domisili: "Kota Bekasi", SetujuSyaratKetentuan: true}

// Langkah 17: brand berbayar, upload bukti, approve TANPA menunggu verifikasi,
// verifikasi pembayaran belakangan.
func TestLangkah17ApproveTidakMenungguPembayaran(t *testing.T) {
	f := setup(t, 150_000)
	ctx := context.Background()
	adam := testdb.NewJamaah(t, f.tx, f.brandID, testdb.JamaahOpts{Nama: "Adam", StatusAgen: "aktif"})
	zainab := testdb.NewJamaah(t, f.tx, f.brandID, testdb.JamaahOpts{Nama: "Zainab", Direkrut: adam})

	must(t, "ajukan", ajukanTx(ctx, f.tx, zainab, ajuan))
	s := f.status(t, zainab)
	if s.StatusAgen != "pengajuan" || s.Pembayaran == nil || s.Pembayaran.NominalTagihan != 150_000 || s.Pembayaran.Status != "menunggu_verifikasi" {
		t.Fatalf("setelah ajukan: %+v / %+v", s, s.Pembayaran)
	}
	mustErr(t, "ajukan dua kali", ajukanTx(ctx, f.tx, zainab, ajuan), ErrSudahMengajukan)

	must(t, "upload bukti", uploadBuktiJamaahTx(ctx, f.tx, zainab, bukti("1")))
	must(t, "setujui", setujuiTx(ctx, f.tx, zainab, &f.brandID, f.adminID))

	s = f.status(t, zainab)
	if s.StatusAgen != "aktif" || s.KodeReferral == nil || len(*s.KodeReferral) != 6 {
		t.Fatalf("setelah setujui: status=%s kode=%v", s.StatusAgen, s.KodeReferral)
	}
	if s.Pembayaran.Status != "menunggu_verifikasi" || s.Pembayaran.KeputusanAgen != "disetujui" {
		t.Fatalf("pembayaran harus tetap menunggu: %+v", s.Pembayaran)
	}
	var upline sql.NullInt64
	must(t, "baca upline", f.tx.QueryRow(`SELECT upline_jamaah_id FROM jamaah WHERE id=?`, zainab).Scan(&upline))
	if !upline.Valid || upline.Int64 != adam {
		t.Fatalf("upline=%v, want %d (agen perekrut)", upline, adam)
	}

	must(t, "verifikasi", verifikasiPembayaranTx(ctx, f.tx, s.Pembayaran.ID, &f.brandID, f.adminID))
	if st := f.status(t, zainab).Pembayaran.Status; st != "terverifikasi" {
		t.Fatalf("status pembayaran=%s, want terverifikasi", st)
	}
	mustErr(t, "verifikasi ulang", verifikasiPembayaranTx(ctx, f.tx, s.Pembayaran.ID, &f.brandID, f.adminID), ErrPembayaranFinal)
}

// Langkah 18: sudah bayar & terverifikasi, keagenan ditolak -> tidak_aktif,
// siklus tertutup; ajukan ulang membuat siklus baru dari nol.
func TestLangkah18DitolakLaluAjukanUlang(t *testing.T) {
	f := setup(t, 150_000)
	ctx := context.Background()
	yunus := testdb.NewJamaah(t, f.tx, f.brandID, testdb.JamaahOpts{Nama: "Yunus"})

	must(t, "ajukan", ajukanTx(ctx, f.tx, yunus, ajuan))
	must(t, "upload", uploadBuktiJamaahTx(ctx, f.tx, yunus, bukti("1")))
	ppa1 := f.status(t, yunus).Pembayaran.ID
	must(t, "verifikasi", verifikasiPembayaranTx(ctx, f.tx, ppa1, &f.brandID, f.adminID))
	must(t, "tolak keagenan", tolakTx(ctx, f.tx, yunus, &f.brandID, "domisili di luar area layanan"))

	s := f.status(t, yunus)
	if s.StatusAgen != "tidak_aktif" || s.Pembayaran.KeputusanAgen != "ditolak" ||
		s.Pembayaran.AlasanDitolakAgen == nil || *s.Pembayaran.AlasanDitolakAgen != "domisili di luar area layanan" {
		t.Fatalf("setelah tolak: %+v / %+v", s, s.Pembayaran)
	}
	mustErr(t, "upload ke siklus ditolak", uploadBuktiJamaahTx(ctx, f.tx, yunus, bukti("2")), ErrSiklusDitutup)
	mustErr(t, "setujui setelah ditolak", setujuiTx(ctx, f.tx, yunus, &f.brandID, f.adminID), ErrBukanPengajuan)

	must(t, "ajukan ulang", ajukanTx(ctx, f.tx, yunus, ajuan))
	s = f.status(t, yunus)
	if s.Pembayaran.ID == ppa1 || s.Pembayaran.Status != "menunggu_verifikasi" || s.Pembayaran.NominalTagihan != 150_000 {
		t.Fatalf("siklus baru harus terpisah & bayar dari nol: %+v", s.Pembayaran)
	}
	var siklus int
	must(t, "hitung siklus", f.tx.QueryRow(`SELECT COUNT(*) FROM pembayaran_pendaftaran_agen WHERE jamaah_id=?`, yunus).Scan(&siklus))
	if siklus != 2 {
		t.Fatalf("siklus=%d, want 2", siklus)
	}
}

// Siklus yang keagenannya ditolak sebelum pembayaran diverifikasi: upload
// ke siklus lama ditolak dengan pesan siklus tertutup.
func TestUploadKeSiklusDitolak(t *testing.T) {
	f := setup(t, 150_000)
	ctx := context.Background()
	x := testdb.NewJamaah(t, f.tx, f.brandID, testdb.JamaahOpts{Nama: "X"})
	must(t, "ajukan", ajukanTx(ctx, f.tx, x, ajuan))
	must(t, "tolak", tolakTx(ctx, f.tx, x, &f.brandID, "data tidak lengkap"))
	mustErr(t, "upload", uploadBuktiJamaahTx(ctx, f.tx, x, bukti("1")), ErrSiklusDitutup)
}

// Langkah 19: brand tanpa biaya -> pembayaran otomatis terverifikasi tanpa verifikator.
func TestLangkah19BrandTanpaBiaya(t *testing.T) {
	f := setup(t, 0)
	ctx := context.Background()
	bilqis := testdb.NewJamaah(t, f.tx, f.brandID, testdb.JamaahOpts{Nama: "Bilqis"})
	must(t, "ajukan", ajukanTx(ctx, f.tx, bilqis, ajuan))

	var status string
	var nominal float64
	var oleh sql.NullInt64
	must(t, "baca", f.tx.QueryRow(`SELECT status, nominal_tagihan, diverifikasi_oleh FROM pembayaran_pendaftaran_agen WHERE jamaah_id=?`, bilqis).
		Scan(&status, &nominal, &oleh))
	if status != "terverifikasi" || nominal != 0 || oleh.Valid {
		t.Fatalf("status=%s nominal=%.0f oleh=%v, want terverifikasi 0 NULL", status, nominal, oleh)
	}
	must(t, "setujui", setujuiTx(ctx, f.tx, bilqis, &f.brandID, f.adminID))
}

// Langkah 20: pembayaran ditolak, keagenan tetap disetujui, revisi upload
// mengembalikan status ke menunggu, riwayat upload tidak tertimpa.
func TestLangkah20RevisiPembayaranDitolak(t *testing.T) {
	f := setup(t, 150_000)
	ctx := context.Background()
	maryam := testdb.NewJamaah(t, f.tx, f.brandID, testdb.JamaahOpts{Nama: "Maryam"})
	must(t, "ajukan", ajukanTx(ctx, f.tx, maryam, ajuan))
	must(t, "upload 1", uploadBuktiJamaahTx(ctx, f.tx, maryam, bukti("1")))
	ppa := f.status(t, maryam).Pembayaran.ID

	must(t, "tolak pembayaran", tolakPembayaranTx(ctx, f.tx, ppa, &f.brandID, f.adminID, "nominal kurang Rp10.000"))
	must(t, "setujui walau ditolak", setujuiTx(ctx, f.tx, maryam, &f.brandID, f.adminID))
	s := f.status(t, maryam)
	if s.StatusAgen != "aktif" || s.Pembayaran.Status != "ditolak" || *s.Pembayaran.CatatanPenolakan != "nominal kurang Rp10.000" {
		t.Fatalf("setelah setujui: %s / %+v", s.StatusAgen, s.Pembayaran)
	}

	must(t, "upload revisi", uploadBuktiJamaahTx(ctx, f.tx, maryam, bukti("2")))
	s = f.status(t, maryam)
	if s.Pembayaran.Status != "menunggu_verifikasi" || len(s.Pembayaran.Uploads) != 2 ||
		s.Pembayaran.Uploads[0].BuktiTransferURL != bukti("1") || *s.Pembayaran.BuktiTransferURL != bukti("2") {
		t.Fatalf("revisi: %+v", s.Pembayaran)
	}
	must(t, "verifikasi", verifikasiPembayaranTx(ctx, f.tx, ppa, &f.brandID, f.adminID))
	mustErr(t, "tolak setelah terverifikasi", tolakPembayaranTx(ctx, f.tx, ppa, &f.brandID, f.adminID, "salah"), ErrPembayaranBukanMenunggu)
}

func TestAdminUploadDanBrandScope(t *testing.T) {
	f := setup(t, 150_000)
	ctx := context.Background()
	x := testdb.NewJamaah(t, f.tx, f.brandID, testdb.JamaahOpts{Nama: "X"})
	must(t, "ajukan", ajukanTx(ctx, f.tx, x, ajuan))
	ppa := f.status(t, x).Pembayaran.ID

	// Admin brand lain tidak bisa menyentuh pengajuan ini.
	other := f.brandID + 999_999
	mustErr(t, "setujui brand lain", setujuiTx(ctx, f.tx, x, &other, f.adminID), ErrNotFound)
	mustErr(t, "verifikasi brand lain", verifikasiPembayaranTx(ctx, f.tx, ppa, &other, f.adminID), ErrNotFound)

	// Admin Travel mengunggah bukti atas nama pemohon (konfirmasi via WA).
	must(t, "upload admin", uploadBuktiAdminTx(ctx, f.tx, ppa, &f.brandID, f.adminID,
		"/api/admin/media/payment-proofs/99999999-2222-3333-4444-555555555555.pdf"))
	if u := f.status(t, x).Pembayaran.Uploads; len(u) != 1 || u[0].DiuploadOlehTipe != "admin" {
		t.Fatalf("upload admin: %+v", u)
	}

	// Antrian B1 memuat pengajuan ini beserta pembayarannya.
	items, err := listPengajuan(ctx, f.tx, &f.brandID)
	must(t, "list", err)
	found := false
	for _, it := range items {
		if it.JamaahID == x && it.Pembayaran != nil && it.Pembayaran.ID == ppa {
			found = true
		}
	}
	if !found {
		t.Fatal("pengajuan tidak muncul di antrian B1")
	}

	// Toggle status hanya untuk agen yang pernah aktif.
	mustErr(t, "nonaktifkan pengajuan", ubahStatusTx(ctx, f.tx, x, &f.brandID, "nonaktif"), ErrStatusTidakValid)
	must(t, "setujui", setujuiTx(ctx, f.tx, x, &f.brandID, f.adminID))
	must(t, "nonaktifkan", ubahStatusTx(ctx, f.tx, x, &f.brandID, "nonaktif"))
	must(t, "aktifkan", ubahStatusTx(ctx, f.tx, x, &f.brandID, "aktif"))
	mustErr(t, "status lain", ubahStatusTx(ctx, f.tx, x, &f.brandID, "pengajuan"), ErrStatusTidakValid)
}
