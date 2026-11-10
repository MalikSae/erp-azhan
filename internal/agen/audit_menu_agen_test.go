package agen

import (
	"context"
	"testing"

	"erp-azhan/api/internal/testdb"
)

// Audit menu Agen 10 November 2026: perbaikan di layar Pengajuan Agen dan status agen.

// S-06: catatan internal admin tidak ikut ke tampilan portal pemohon.
func TestCatatanAdminTidakKePortal(t *testing.T) {
	f := setup(t, 150_000)
	id := f.pengajuanBaru(t, "CatatanInternal", false)
	must(t, "bayar menyusul", f.keputusan(id, AksiSetujuiBayarMenyusul, "keluarga pemilik travel"))
	if got := f.catatanAdmin(t, id); got != "keluarga pemilik travel" {
		t.Fatalf("catatan admin = %q", got)
	}
	if s := f.status(t, id); s.Pembayaran == nil || s.Pembayaran.CatatanAdmin != nil {
		t.Fatalf("portal tidak boleh menerima catatan admin: %+v", s.Pembayaran)
	}
}

// S-05, R-06: tiap keputusan tersimpan di riwayat dengan catatan dan pelakunya, tidak saling menimpa.
func TestRiwayatKeputusanTidakTertimpa(t *testing.T) {
	f := setup(t, 150_000)
	ctx := context.Background()
	id := f.pengajuanBaru(t, "RiwayatKeputusan", false)
	must(t, "bayar menyusul", f.keputusan(id, AksiSetujuiBayarMenyusul, "bayar setelah kopdar"))
	must(t, "bebaskan", f.keputusan(id, AksiBebaskan, "dibebaskan pemilik"))

	p, err := latestPembayaran(ctx, f.tx, id)
	must(t, "siklus", err)
	riwayat, err := listRiwayatKeputusan(ctx, f.tx, p.ID)
	must(t, "riwayat", err)
	if len(riwayat) != 2 {
		t.Fatalf("riwayat = %d baris, want 2: %+v", len(riwayat), riwayat)
	}
	// Terbaru dulu.
	if riwayat[0].Aksi != AksiBebaskan || riwayat[0].Catatan == nil || *riwayat[0].Catatan != "dibebaskan pemilik" ||
		riwayat[1].Aksi != AksiSetujuiBayarMenyusul || riwayat[1].Catatan == nil || *riwayat[1].Catatan != "bayar setelah kopdar" {
		t.Fatalf("isi riwayat: %+v", riwayat)
	}
	if riwayat[0].OlehNama == nil {
		t.Error("pelaku keputusan harus tercatat")
	}

	// Penolakan juga tercatat beserta pelakunya.
	ditolak := f.pengajuanBaru(t, "RiwayatTolak", false)
	must(t, "tolak", f.keputusan(ditolak, AksiTolak, "data tidak sesuai"))
	p2, err := latestPembayaran(ctx, f.tx, ditolak)
	must(t, "siklus tolak", err)
	r2, err := listRiwayatKeputusan(ctx, f.tx, p2.ID)
	must(t, "riwayat tolak", err)
	if len(r2) != 1 || r2[0].Aksi != AksiTolak || r2[0].OlehNama == nil {
		t.Fatalf("riwayat tolak: %+v", r2)
	}
	// Keputusan yang gagal tidak meninggalkan baris riwayat.
	mustErr(t, "aksi sesudah tolak", f.keputusan(ditolak, AksiSetujui, ""), ErrKeputusanTidakBisa)
	if r3, _ := listRiwayatKeputusan(ctx, f.tx, p2.ID); len(r3) != 1 {
		t.Fatalf("keputusan gagal ikut tercatat: %+v", r3)
	}
}

// S-02: penolakan melepas NIK agen supaya pemilik NIK asli bisa mengajukan, dan pelepasannya tercatat.
func TestTolakMelepasNIK(t *testing.T) {
	f := setup(t, 0)
	ctx := context.Background()
	denganNIK := ajuan
	denganNIK.NIK = "9999999999990725"
	salah := testdb.NewAkun(t, f.tx, f.brandID, testdb.AkunOpts{Nama: "SalahNIK"})
	must(t, "ajukan", ajukanTx(ctx, f.tx, salah, denganNIK))
	var nik *string
	if err := f.tx.QueryRowContext(ctx, `SELECT nik_agen FROM customer_accounts WHERE id = ?`, salah).Scan(&nik); err != nil || nik == nil {
		t.Fatalf("NIK pengajuan harus terisi: %v %v", nik, err)
	}
	must(t, "tolak", f.keputusan(salah, AksiTolak, "NIK milik orang lain"))
	if err := f.tx.QueryRowContext(ctx, `SELECT nik_agen FROM customer_accounts WHERE id = ?`, salah).Scan(&nik); err != nil || nik != nil {
		t.Fatalf("NIK harus kosong setelah ditolak: %v %v", nik, err)
	}
	var logBaris int
	if err := f.tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM akun_data_log WHERE akun_id = ? AND kolom = 'nik_agen' AND nilai_baru IS NULL`, salah).Scan(&logBaris); err != nil || logBaris != 1 {
		t.Fatalf("log pelepasan NIK = %d (%v), want 1", logBaris, err)
	}
	// Pemilik NIK asli kini bisa mengajukan dengan NIK yang sama.
	pemilik := testdb.NewAkun(t, f.tx, f.brandID, testdb.AkunOpts{Nama: "PemilikNIK"})
	must(t, "ajukan pemilik NIK", ajukanTx(ctx, f.tx, pemilik, denganNIK))
	if s := f.status(t, pemilik); s.StatusAgen != "pengajuan" {
		t.Fatalf("pemilik NIK asli harus bisa mengajukan: %+v", s)
	}
}

// S-03: tagihan agen yang sudah disetujui bisa diselesaikan walau agennya dinonaktifkan.
// S-01: bukti susulan bisa diminta unggah ulang.
func TestTagihanAgenNonaktifDanUnggahUlangSusulan(t *testing.T) {
	f := setup(t, 150_000)
	ctx := context.Background()
	id := f.pengajuanBaru(t, "SusulanNonaktif", false)
	must(t, "bayar menyusul", f.keputusan(id, AksiSetujuiBayarMenyusul, "bayar akhir bulan"))
	must(t, "bukti susulan", uploadBuktiAkunTx(ctx, f.tx, id, bukti("7")))

	// S-01: bukti susulan salah, minta unggah ulang. Status agen tidak berubah.
	must(t, "minta unggah ulang", f.keputusan(id, AksiMintaUnggahUlang, "foto buram"))
	if s := f.status(t, id); s.StatusAgen != "aktif" || s.Pembayaran.Status != "ditolak" {
		t.Fatalf("setelah minta unggah ulang: %+v / %+v", s, s.Pembayaran)
	}

	// S-03: agen dinonaktifkan, tagihannya tetap bisa diputuskan.
	must(t, "nonaktifkan", ubahStatusTx(ctx, f.tx, id, &f.brandID, f.adminID, "nonaktif", "melanggar ketentuan"))
	mustErr(t, "tolak agen nonaktif", f.keputusan(id, AksiTolak, "tidak berlaku"), ErrKeputusanTidakBisa)
	must(t, "bukti baru", uploadBuktiAkunTx(ctx, f.tx, id, bukti("8")))
	must(t, "lunasi saat nonaktif", f.keputusan(id, AksiSetujui, ""))
	s := f.status(t, id)
	if s.StatusAgen != "nonaktif" || s.Pembayaran.Status != "terverifikasi" {
		t.Fatalf("pelunasan tidak boleh mengubah status agen: %+v / %+v", s, s.Pembayaran)
	}
	if row := findPengajuan(t, f, id); row != nil {
		t.Fatal("tagihan lunas harus keluar dari antrean")
	}
}

// R-02: nonaktifkan dan aktifkan agen wajib beralasan dan tercatat.
func TestUbahStatusAgenBeralasanDanTercatat(t *testing.T) {
	f := setup(t, 0)
	ctx := context.Background()
	id := f.pengajuanBaru(t, "StatusLog", false)
	must(t, "setujui", f.keputusan(id, AksiSetujui, ""))

	mustErr(t, "tanpa alasan", ubahStatusTx(ctx, f.tx, id, &f.brandID, f.adminID, "nonaktif", "  "), ErrAlasanStatusWajib)
	mustErr(t, "status sama", ubahStatusTx(ctx, f.tx, id, &f.brandID, f.adminID, "aktif", "tidak berubah"), ErrStatusSama)
	must(t, "nonaktifkan", ubahStatusTx(ctx, f.tx, id, &f.brandID, f.adminID, "nonaktif", "melanggar ketentuan"))
	must(t, "aktifkan", ubahStatusTx(ctx, f.tx, id, &f.brandID, f.adminID, "aktif", "sudah diklarifikasi"))

	riwayat, err := listRiwayatStatus(ctx, f.tx, id)
	must(t, "riwayat", err)
	if len(riwayat) != 2 || riwayat[0].StatusBaru != "aktif" || riwayat[0].Alasan != "sudah diklarifikasi" ||
		riwayat[1].StatusLama != "aktif" || riwayat[1].StatusBaru != "nonaktif" || riwayat[1].OlehNama == nil {
		t.Fatalf("riwayat status: %+v", riwayat)
	}
}
