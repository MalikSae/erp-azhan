package agen

import (
	"context"
	"database/sql"
	"testing"

	"erp-azhan/api/internal/testdb"
)

func (f fixture) keputusan(akun int64, aksi, catatan string) error {
	return keputusanTx(context.Background(), f.tx, akun, &f.brandID, f.adminID, KeputusanRequest{Aksi: aksi, Catatan: catatan})
}

// catatanAdmin membaca catatan internal dari sisi admin. Tampilan portal (f.status) tidak memuatnya.
func (f fixture) catatanAdmin(t *testing.T, akun int64) string {
	t.Helper()
	p, err := latestPembayaran(context.Background(), f.tx, akun)
	if err != nil || p == nil {
		t.Fatalf("latestPembayaran: %v", err)
	}
	if p.CatatanAdmin == nil {
		return ""
	}
	return *p.CatatanAdmin
}

func (f fixture) pengajuanBaru(t *testing.T, nama string, denganBukti bool) int64 {
	t.Helper()
	ctx := context.Background()
	id := testdb.NewAkun(t, f.tx, f.brandID, testdb.AkunOpts{Nama: nama})
	must(t, "ajukan", ajukanTx(ctx, f.tx, id, ajuan))
	if denganBukti {
		must(t, "bukti", uploadBuktiAkunTx(ctx, f.tx, id, bukti("1")))
	}
	return id
}

func TestKeputusanSetujuiButuhBukti(t *testing.T) {
	f := setup(t, 150_000)
	tanpa := f.pengajuanBaru(t, "TanpaBukti", false)
	mustErr(t, "tanpa bukti", f.keputusan(tanpa, AksiSetujui, ""), ErrBuktiBelumAda)
	if s := f.status(t, tanpa); s.StatusAgen != "pengajuan" {
		t.Fatalf("status = %s, want tetap pengajuan", s.StatusAgen)
	}

	dengan := f.pengajuanBaru(t, "DenganBukti", true)
	must(t, "setujui", f.keputusan(dengan, AksiSetujui, ""))
	s := f.status(t, dengan)
	if s.StatusAgen != "aktif" || s.KodeReferral == nil || s.Pembayaran.Status != "terverifikasi" || s.Pembayaran.KeputusanAgen != "disetujui" {
		t.Fatalf("setelah setujui: %+v / %+v", s, s.Pembayaran)
	}
	mustErr(t, "keputusan ulang", f.keputusan(dengan, AksiSetujui, ""), ErrKeputusanTidakBisa)
}

func TestKeputusanSetujuiBiayaNol(t *testing.T) {
	f := setup(t, 0)
	id := f.pengajuanBaru(t, "Gratis", false)
	must(t, "setujui", f.keputusan(id, AksiSetujui, ""))
	if s := f.status(t, id); s.StatusAgen != "aktif" || s.Pembayaran.Status != "terverifikasi" {
		t.Fatalf("biaya Rp0: %+v / %+v", s, s.Pembayaran)
	}
}

func TestKeputusanBebaskanBiaya(t *testing.T) {
	f := setup(t, 150_000)
	id := f.pengajuanBaru(t, "Bebas", false)
	mustErr(t, "tanpa catatan", f.keputusan(id, AksiBebaskan, " "), ErrCatatanWajib)
	must(t, "bebaskan", f.keputusan(id, AksiBebaskan, "kerabat pemilik, biaya dibebaskan"))
	s := f.status(t, id)
	if s.StatusAgen != "aktif" || s.Pembayaran.Status != "dibebaskan" ||
		f.catatanAdmin(t, id) != "kerabat pemilik, biaya dibebaskan" || s.Pembayaran.DiverifikasiAt == nil {
		t.Fatalf("setelah bebaskan: %+v / %+v", s, s.Pembayaran)
	}
	if row := findPengajuan(t, f, id); row != nil {
		t.Fatal("tagihan dibebaskan harus keluar dari antrian B1")
	}
	mustErr(t, "verifikasi tagihan dibebaskan", verifikasiPembayaranTx(context.Background(), f.tx, s.Pembayaran.ID, &f.brandID, f.adminID), ErrPembayaranFinal)
}

func TestKeputusanMintaUnggahUlang(t *testing.T) {
	f := setup(t, 150_000)
	id := f.pengajuanBaru(t, "Buram", true)
	mustErr(t, "tanpa catatan", f.keputusan(id, AksiMintaUnggahUlang, ""), ErrCatatanWajib)
	must(t, "minta ulang", f.keputusan(id, AksiMintaUnggahUlang, "bukti buram"))
	s := f.status(t, id)
	if s.StatusAgen != "pengajuan" || s.Pembayaran.Status != "ditolak" || *s.Pembayaran.CatatanPenolakan != "bukti buram" {
		t.Fatalf("setelah minta ulang: %+v / %+v", s, s.Pembayaran)
	}
	must(t, "unggah ulang", uploadBuktiAkunTx(context.Background(), f.tx, id, bukti("2")))
	if st := f.status(t, id).Pembayaran.Status; st != "menunggu_verifikasi" {
		t.Fatalf("setelah unggah ulang = %s, want menunggu_verifikasi", st)
	}
	must(t, "setujui", f.keputusan(id, AksiSetujui, ""))
}

func TestKeputusanTolak(t *testing.T) {
	f := setup(t, 150_000)
	id := f.pengajuanBaru(t, "Ditolak", true)
	mustErr(t, "tanpa alasan", f.keputusan(id, AksiTolak, ""), ErrCatatanWajib)
	must(t, "tolak", f.keputusan(id, AksiTolak, "domisili di luar area"))
	s := f.status(t, id)
	if s.StatusAgen != "tidak_aktif" || s.Pembayaran.KeputusanAgen != "ditolak" || *s.Pembayaran.AlasanDitolakAgen != "domisili di luar area" {
		t.Fatalf("setelah tolak: %+v / %+v", s, s.Pembayaran)
	}
	mustErr(t, "aksi sesudah tolak", f.keputusan(id, AksiSetujui, ""), ErrKeputusanTidakBisa)
}

// K21: setujui, bayar menyusul. Tagihan tetap tercatat dan dapat dilunasi atau dibebaskan kemudian.
func TestKeputusanSetujuiBayarMenyusul(t *testing.T) {
	f := setup(t, 150_000)
	ctx := context.Background()
	id := f.pengajuanBaru(t, "Menyusul", false)
	mustErr(t, "tanpa catatan", f.keputusan(id, AksiSetujuiBayarMenyusul, ""), ErrCatatanWajib)
	must(t, "bayar menyusul", f.keputusan(id, AksiSetujuiBayarMenyusul, "bayar setelah kopdar"))
	s := f.status(t, id)
	if s.StatusAgen != "aktif" || s.Pembayaran.Status != "menunggu_verifikasi" || f.catatanAdmin(t, id) != "bayar setelah kopdar" {
		t.Fatalf("setelah bayar menyusul: %+v / %+v", s, s.Pembayaran)
	}
	if row := findPengajuan(t, f, id); row == nil {
		t.Fatal("tagihan belum lunas harus tetap di antrian B1")
	}
	mustErr(t, "setujui tanpa bukti", f.keputusan(id, AksiSetujui, ""), ErrBuktiBelumAda)
	must(t, "bukti", uploadBuktiAkunTx(ctx, f.tx, id, bukti("3")))
	must(t, "lunas", f.keputusan(id, AksiSetujui, ""))
	if st := f.status(t, id).Pembayaran.Status; st != "terverifikasi" {
		t.Fatalf("setelah dilunasi = %s", st)
	}

	// Jalur lain: dibebaskan kemudian. Tolak/menyusul lagi tidak berlaku untuk agen aktif.
	id2 := f.pengajuanBaru(t, "MenyusulBebas", false)
	must(t, "menyusul 2", f.keputusan(id2, AksiSetujuiBayarMenyusul, "menyusul"))
	mustErr(t, "tolak agen aktif", f.keputusan(id2, AksiTolak, "x x x"), ErrKeputusanTidakBisa)
	must(t, "bebaskan kemudian", f.keputusan(id2, AksiBebaskan, "dibebaskan pemilik"))
	if st := f.status(t, id2).Pembayaran.Status; st != "dibebaskan" {
		t.Fatalf("setelah bebaskan kemudian = %s", st)
	}

}

// Biaya Rp0 tidak bisa bayar menyusul (tidak ada tagihan).
func TestKeputusanBayarMenyusulBiayaNol(t *testing.T) {
	f := setup(t, 0)
	nol := f.pengajuanBaru(t, "Nol", false)
	mustErr(t, "menyusul biaya nol", f.keputusan(nol, AksiSetujuiBayarMenyusul, "tidak perlu"), ErrKeputusanTidakBisa)
}

func TestKeputusanAksiTidakDikenalDanBrandScope(t *testing.T) {
	f := setup(t, 150_000)
	id := f.pengajuanBaru(t, "Scope", true)
	mustErr(t, "aksi asing", f.keputusan(id, "hapus", "x x x"), ErrAksiTidakValid)
	other := f.brandID + 999_999
	mustErr(t, "brand lain", keputusanTx(context.Background(), f.tx, id, &other, f.adminID, KeputusanRequest{Aksi: AksiSetujui}), ErrNotFound)
	var n int
	must(t, "status", f.tx.QueryRow(`SELECT COUNT(*) FROM customer_accounts WHERE id=? AND status_agen='pengajuan'`, id).Scan(&n))
	if n != 1 {
		t.Fatal("status berubah walau aksi ditolak")
	}
	_ = sql.ErrNoRows
}
