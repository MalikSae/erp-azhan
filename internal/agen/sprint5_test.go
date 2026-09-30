package agen

import (
	"context"
	"database/sql"
	"testing"

	"erp-azhan/api/internal/komisi"
	"erp-azhan/api/internal/testdb"
)

func adminUji(t *testing.T, tx *sql.Tx) int64 {
	t.Helper()
	var id int64
	must(t, "admin", tx.QueryRow(`SELECT id FROM admin_users ORDER BY id LIMIT 1`).Scan(&id))
	return id
}

func tersedia(t *testing.T, tx *sql.Tx, agenID int64) float64 {
	t.Helper()
	s, err := komisi.HitungSaldo(context.Background(), tx, agenID)
	must(t, "saldo", err)
	return s.Tersedia
}

var rekeningUji = Rekening{Bank: "BSI", Nomor: "7123456789", AtasNama: "Adam Uji"}

// Langkah 8–9: pencairan hanya dari komisi yang sudah berangkat, satu pending
// per agen, disetujui Admin Master dengan bukti transfer, saldo berkurang.
func TestLangkah8dan9Pencairan(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	sched, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{
		KomisiLangsung: rp(1_000_000), BerangkatTanggal: "2000-01-01",
	})
	admin := adminUji(t, tx)
	adam := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Adam", StatusAgen: "aktif"})
	idris := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Idris", Direkrut: adam})
	istri := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Istri", Direkrut: adam})
	b, _ := testdb.NewBookingPax(t, tx, sched, "dp", 2000, testdb.Pax{JamaahID: idris}, testdb.Pax{JamaahID: istri})
	must(t, "snapshot", komisi.SnapshotNominal(ctx, tx, b))
	testdb.Exec(t, tx, `UPDATE bookings SET status='lunas' WHERE id=?`, b)
	must(t, "komisi", komisi.ProcessBookingLunas(ctx, tx, b))
	if s := tersedia(t, tx, adam); s != 2_000_000 {
		t.Fatalf("saldo awal %v, want 2.000.000", s)
	}

	ajukan := func(n float64, rek Rekening) error {
		_, err := ajukanPencairanTx(ctx, tx, adam, AjukanPencairanRequest{Nominal: n, Rekening: rek})
		return err
	}
	if err := ajukan(3_000_000, rekeningUji); err != ErrPencairanMelebihi {
		t.Fatalf("melebihi saldo: %v", err)
	}
	if err := ajukan(400_000, rekeningUji); err != ErrPencairanMinimal {
		t.Fatalf("di bawah minimal: %v", err)
	}
	if err := ajukan(1_000_000, Rekening{Bank: "BSI"}); err != ErrRekeningTidakLengkap {
		t.Fatalf("rekening kosong: %v", err)
	}

	// Langkah 8: ajukan Rp1 juta, lalu ajukan lagi selagi pending -> ditolak.
	id1, err := ajukanPencairanTx(ctx, tx, adam, AjukanPencairanRequest{Nominal: 1_000_000, Rekening: rekeningUji})
	must(t, "ajukan", err)
	if err := ajukan(500_000, rekeningUji); err != ErrPencairanMasihPending {
		t.Fatalf("pengajuan kedua selagi pending: %v", err)
	}
	if s := tersedia(t, tx, adam); s != 1_000_000 {
		t.Fatalf("saldo selama pending %v, want 1.000.000", s)
	}
	p, err := getPencairanPortal(ctx, tx, adam)
	must(t, "portal", err)
	if !p.AdaPending || p.Rekening != rekeningUji || len(p.Riwayat) != 1 {
		t.Fatalf("state A7 = %+v", p)
	}

	// Langkah 9: disetujui dengan bukti -> saldo tetap berkurang, tidak bisa diproses dua kali.
	must(t, "setujui", setujuiPencairanTx(ctx, tx, id1, admin, "/api/admin/media/payment-proofs/uji.pdf"))
	if err := setujuiPencairanTx(ctx, tx, id1, admin, "x"); err != ErrPencairanBukanPending {
		t.Fatalf("setujui ulang: %v", err)
	}
	if s := tersedia(t, tx, adam); s != 1_000_000 {
		t.Fatalf("saldo setelah disetujui %v, want 1.000.000", s)
	}
	if u, err := buktiPencairan(ctx, tx, id1, adam); err != nil || u == "" {
		t.Fatalf("bukti milik agen: %q %v", u, err)
	}
	lain := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "AgenLain", StatusAgen: "aktif"})
	if _, err := buktiPencairan(ctx, tx, id1, lain); err != ErrNotFound {
		t.Fatalf("bukti dibuka agen lain: %v", err)
	}

	// Ditolak -> saldo kembali.
	id2, err := ajukanPencairanTx(ctx, tx, adam, AjukanPencairanRequest{Nominal: 600_000, Rekening: rekeningUji})
	must(t, "ajukan 2", err)
	must(t, "tolak", tolakPencairanTx(ctx, tx, id2, admin, "rekening tidak valid"))
	if s := tersedia(t, tx, adam); s != 1_000_000 {
		t.Fatalf("saldo setelah ditolak %v, want 1.000.000", s)
	}

	// Cek ulang saat disetujui: saldo turun (jadwal diundur) -> ditolak sistem.
	id3, err := ajukanPencairanTx(ctx, tx, adam, AjukanPencairanRequest{Nominal: 1_000_000, Rekening: rekeningUji})
	must(t, "ajukan 3", err)
	testdb.Exec(t, tx, `UPDATE schedules SET berangkat_tanggal='2099-01-01' WHERE id=?`, sched)
	if err := setujuiPencairanTx(ctx, tx, id3, admin, "x"); err != ErrSaldoTidakCukup {
		t.Fatalf("saldo tidak cukup saat disetujui: %v", err)
	}
	must(t, "tolak 3", tolakPencairanTx(ctx, tx, id3, admin, "saldo belum tersedia"))

	// Agen nonaktif tidak bisa mengajukan.
	testdb.Exec(t, tx, `UPDATE schedules SET berangkat_tanggal='2000-01-01' WHERE id=?`, sched)
	testdb.Exec(t, tx, `UPDATE jamaah SET status_agen='nonaktif' WHERE id=?`, adam)
	if err := ajukan(500_000, rekeningUji); err != ErrBukanAgenAktif {
		t.Fatalf("agen nonaktif: %v", err)
	}

	// Antrian C1 lintas brand memuat pengajuan ini dengan nama brand.
	items, err := listPencairanAdmin(ctx, tx, "", &brand)
	must(t, "list C1", err)
	if len(items) != 3 {
		t.Fatalf("antrian C1 berisi %d, want 3", len(items))
	}
}

func kaitanSumber(t *testing.T, tx *sql.Tx, id int64) (string, int64, string) {
	t.Helper()
	var st string
	var by sql.NullInt64
	var sumber sql.NullString
	must(t, "baca", tx.QueryRow(`SELECT kaitan_status, direkrut_oleh_jamaah_id, kaitan_sumber FROM jamaah WHERE id=?`, id).Scan(&st, &by, &sumber))
	return st, by.Int64, sumber.String
}

// Langkah 15–16: ganti kaitan Jalur 3 oleh Admin Master, dengan gate komisi.
func TestLangkah15dan16GantiKaitan(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	sched, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{KomisiLangsung: rp(1_000_000), BerangkatTanggal: "2099-01-01"})
	admin := adminUji(t, tx)
	nuh := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Nuh", StatusAgen: "aktif"})
	hud := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Hud", StatusAgen: "aktif"})
	ganti := func(id int64, mode string, agenID int64) error {
		req := GantiKaitanRequest{Mode: mode, Alasan: "salah pilih saat input"}
		if agenID > 0 {
			req.AgenJamaahID = &agenID
		}
		return gantiKaitanTx(ctx, tx, id, admin, req)
	}

	// Langkah 15: Utsman salah dikaitkan ke Nuh, belum pernah lunas -> ganti.
	utsman := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Utsman"})
	companion := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Companion"})
	must(t, "jalur 3", TetapkanKaitanJalur3(ctx, tx, utsman, "agen", nuh))
	must(t, "ikut PIC", IkatRombongan(ctx, tx, utsman, []int64{companion}, 0))
	if st, _, sumber := kaitanSumber(t, tx, utsman); st != "terikat_agen" || sumber != SumberJalur3 {
		t.Fatalf("utsman = %s/%s", st, sumber)
	}
	must(t, "ganti tanpa agen", ganti(utsman, "tanpa_agen", 0))
	if st, by, sumber := kaitanSumber(t, tx, utsman); st != "tanpa_agen" || by != 0 || sumber != SumberJalur3 {
		t.Fatalf("setelah ganti = %s/%d/%s", st, by, sumber)
	}
	// Tidak cascading ke companion yang sudah auto-bind ke Nuh.
	if st, by, sumber := kaitanSumber(t, tx, companion); st != "terikat_agen" || by != nuh || sumber != SumberIkutPIC {
		t.Fatalf("companion berubah: %s/%d/%s", st, by, sumber)
	}
	if err := ganti(utsman, "tanpa_agen", 0); err != ErrKaitanTidakBerubah {
		t.Fatalf("ganti tanpa perubahan: %v", err)
	}
	if err := ganti(utsman, "agen", utsman); err != ErrAgenTidakValid {
		t.Fatalf("ikat ke diri sendiri: %v", err)
	}
	must(t, "ganti ke Hud", ganti(utsman, "agen", hud))
	logs, err := listKaitanLog(ctx, tx, utsman)
	must(t, "log", err)
	if len(logs) != 2 || logs[0].StatusBaru != "terikat_agen" || logs[1].StatusLama != "terikat_agen" || logs[1].Alasan == "" {
		t.Fatalf("log kaitan = %+v", logs)
	}

	// Langkah 16: Utsman sudah pernah lunas (Hud dapat komisi) -> ditolak.
	b, _ := testdb.NewBookingPax(t, tx, sched, "dp", 1000, testdb.Pax{JamaahID: utsman})
	// Booking belum lunas dihitung untuk peringatan UI (audit KA-03).
	if h, err := cariJamaahKaitan(ctx, tx, &brand, "Utsman (uji)"); err != nil || len(h) != 1 || h[0].BookingBelumLunas != 1 || !h[0].BisaDiganti {
		t.Fatalf("booking belum lunas = %+v, %v", h, err)
	}
	must(t, "snapshot", komisi.SnapshotNominal(ctx, tx, b))
	testdb.Exec(t, tx, `UPDATE bookings SET status='lunas' WHERE id=?`, b)
	must(t, "komisi", komisi.ProcessBookingLunas(ctx, tx, b))
	if err := ganti(utsman, "tanpa_agen", 0); err != ErrKaitanSudahKomisi {
		t.Fatalf("langkah 16: %v", err)
	}
	if h, err := cariJamaahKaitan(ctx, tx, &brand, "Utsman (uji)"); err != nil || len(h) != 1 || h[0].BookingBelumLunas != 0 || h[0].BisaDiganti {
		t.Fatalf("setelah lunas = %+v, %v", h, err)
	}
	// Log jamaah yang tidak ada: ErrNotFound, bukan daftar kosong (audit KA-04).
	if _, err := listKaitanLog(ctx, tx, utsman+999_999); err != ErrNotFound {
		t.Fatalf("log jamaah tidak ada: %v", err)
	}

	// Kaitan Jalur 2 (referral) dan ikut PIC tidak bisa diganti lewat C4.
	referral := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Referral"})
	must(t, "referral", IkatJamaah(ctx, tx, referral, nuh, SumberReferral))
	if err := ganti(referral, "tanpa_agen", 0); err != ErrKaitanBukanJalur3 {
		t.Fatalf("jalur 2: %v", err)
	}
	if err := ganti(companion, "tanpa_agen", 0); err != ErrKaitanBukanJalur3 {
		t.Fatalf("ikut PIC: %v", err)
	}

	// Jamaah Jalur 3 yang kemudian jadi agen tidak bisa diganti (upline terkait).
	calon := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Calon"})
	must(t, "jalur 3 calon", TetapkanKaitanJalur3(ctx, tx, calon, "agen", nuh))
	testdb.Exec(t, tx, `UPDATE jamaah SET status_agen='pengajuan' WHERE id=?`, calon)
	if err := ganti(calon, "tanpa_agen", 0); err != ErrKaitanJamaahAgen {
		t.Fatalf("jamaah pengajuan agen: %v", err)
	}

	// Agen tujuan harus agen aktif di brand yang sama.
	salah := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Salah"})
	must(t, "jalur 3 salah", TetapkanKaitanJalur3(ctx, tx, salah, "tanpa_agen", 0))
	biasa := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Biasa"})
	if err := ganti(salah, "agen", biasa); err != ErrAgenTidakValid {
		t.Fatalf("agen tujuan bukan agen: %v", err)
	}

	// Pencarian C4 menandai mana yang bisa diganti.
	hasil, err := cariJamaahKaitan(ctx, tx, &brand, "(uji)")
	must(t, "cari", err)
	bisa := map[int64]bool{}
	for _, h := range hasil {
		bisa[h.JamaahID] = h.BisaDiganti
	}
	if bisa[utsman] || bisa[referral] || !bisa[salah] {
		t.Fatalf("flag bisa diganti = %v", bisa)
	}
}
