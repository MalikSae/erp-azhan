package agen

import (
	"context"
	"database/sql"
	"net/url"
	"testing"

	"erp-azhan/api/internal/komisi"
	"erp-azhan/api/internal/testdb"
)

// Code review Agen, 11 November 2026.

// C1: upline yang sedang nonaktif tetap dipakai untuk mengikat jamaah saat akun booking di web.
func TestUplineAkun(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	_, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	aktif := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Aktif", StatusAgen: "aktif"})
	nonaktif := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Nonaktif", StatusAgen: "nonaktif"})
	bukanAgen := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "BukanAgen"})
	uplineAktif := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Z1", Direkrut: aktif})
	uplineNonaktif := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Z2", Direkrut: nonaktif})
	uplineBukanAgen := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Z3", Direkrut: bukanAgen})
	tanpaUpline := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Z4"})
	for _, c := range []struct {
		nama       string
		akun, want int64
	}{
		{"upline aktif", uplineAktif, aktif},
		{"upline nonaktif tetap mengikat", uplineNonaktif, nonaktif},
		{"upline bukan agen", uplineBukanAgen, 0},
		{"tanpa upline", tanpaUpline, 0},
	} {
		got, err := UplineAkun(ctx, tx, c.akun)
		must(t, c.nama, err)
		if got != c.want {
			t.Errorf("%s: UplineAkun = %d, mau %d", c.nama, got, c.want)
		}
	}
}

// C4: hanya akun bukan agen yang mendapat upline dari jamaah dirinya, dan kejadiannya tercatat di log.
func TestIkatAkunDariJamaahHanyaBukanAgen(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	_, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	x := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenX", StatusAgen: "aktif"})
	bukanAgen := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Pemesan"})
	agenTanpaUpline := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenZ", StatusAgen: "aktif"})
	j1 := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Diri Pemesan", Direkrut: x})
	j2 := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Diri Agen Z", Direkrut: x})

	upline := func(akun int64) sql.NullInt64 {
		var v sql.NullInt64
		must(t, "baca upline", tx.QueryRow(`SELECT direkrut_oleh_akun_id FROM customer_accounts WHERE id=?`, akun).Scan(&v))
		return v
	}
	jumlahLog := func(akun int64) int {
		var n int
		must(t, "hitung log", tx.QueryRow(`SELECT COUNT(*) FROM akun_kaitan_log WHERE akun_id=?`, akun).Scan(&n))
		return n
	}

	must(t, "ikat bukan agen", IkatAkunDariJamaah(ctx, tx, bukanAgen, j1))
	if got := upline(bukanAgen); got.Int64 != x {
		t.Fatalf("akun bukan agen: upline = %v, mau %d", got, x)
	}
	if n := jumlahLog(bukanAgen); n != 1 {
		t.Fatalf("log kaitan akun = %d baris, mau 1", n)
	}
	var oleh sql.NullInt64
	var alasan string
	must(t, "baca log", tx.QueryRow(`SELECT diganti_oleh, alasan FROM akun_kaitan_log WHERE akun_id=?`, bukanAgen).Scan(&oleh, &alasan))
	if oleh.Valid || alasan != AlasanIkatAkunDariJamaah {
		t.Fatalf("log: diganti_oleh=%v alasan=%q", oleh, alasan)
	}
	logs, err := listAkunKaitanLog(ctx, tx, nil, bukanAgen)
	if err != nil || len(logs) != 1 || logs[0].DigantiOleh != "Sistem" {
		t.Fatalf("daftar log = %+v, %v", logs, err)
	}

	// Dipanggil lagi: sudah punya upline, tidak ada perubahan dan tidak ada log baru.
	must(t, "ikat ulang", IkatAkunDariJamaah(ctx, tx, bukanAgen, j1))
	if n := jumlahLog(bukanAgen); n != 1 {
		t.Fatalf("log setelah ikat ulang = %d baris, mau 1", n)
	}

	// Akun yang sudah agen tidak disentuh.
	must(t, "ikat agen", IkatAkunDariJamaah(ctx, tx, agenTanpaUpline, j2))
	if got := upline(agenTanpaUpline); got.Valid {
		t.Fatalf("akun agen tidak boleh mendapat upline lewat jalur ini, dapat %d", got.Int64)
	}
	if n := jumlahLog(agenTanpaUpline); n != 0 {
		t.Fatalf("log untuk akun agen = %d baris, mau 0", n)
	}
}

// fixturePindahKomisi: jamaah terikat ke agen lama (ber-upline aktif) dengan komisi langsung dan bonus
// pembinaan aktif. Jadwal berangkat tahun 2000, jadi komisinya sudah tersedia.
func fixturePindahKomisi(t *testing.T, tx *sql.Tx) (brand, uplineLama, lama, jamaah, booking int64) {
	t.Helper()
	ctx := context.Background()
	var sched int64
	sched, brand = testdb.Schedule(t, tx, testdb.ScheduleOpts{KomisiLangsung: rp(500_000), BonusPembinaan: rp(100_000), BerangkatTanggal: "2000-01-01"})
	uplineLama = testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "UplineLama", StatusAgen: "aktif"})
	lama = testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenLama", StatusAgen: "aktif", Direkrut: uplineLama, Upline: uplineLama})
	jamaah = testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "J", Direkrut: lama})
	booking, _ = testdb.NewBookingPax(t, tx, sched, "dp", 1000, testdb.Pax{JamaahID: jamaah})
	must(t, "snapshot", komisi.SnapshotNominal(ctx, tx, booking))
	testdb.Exec(t, tx, `UPDATE bookings SET status='lunas' WHERE id=?`, booking)
	must(t, "komisi", komisi.ProcessBookingLunas(ctx, tx, booking))
	return
}

type barisKomisi struct {
	penerima int64
	batal    bool
	sebab    sql.NullString
}

func bacaKomisi(t *testing.T, tx *sql.Tx, booking int64, jenis string) barisKomisi {
	t.Helper()
	var b barisKomisi
	must(t, "baca komisi "+jenis, tx.QueryRow(`SELECT akun_penerima_id, dibatalkan_at IS NOT NULL, sebab_batal FROM transaksi_komisi WHERE booking_id=? AND jenis=?`, booking, jenis).Scan(&b.penerima, &b.batal, &b.sebab))
	return b
}

// C2: bonus pembinaan hanya pindah ke upline agen baru yang aktif; selain itu ditandai batal, tidak dihapus.
func TestPindahKomisiBonusPembinaan(t *testing.T) {
	ctx := context.Background()

	t.Run("upline agen baru aktif", func(t *testing.T) {
		_, tx := testdb.Tx(t)
		brand, _, _, jamaah, booking := fixturePindahKomisi(t, tx)
		uplineBaru := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "UplineBaru", StatusAgen: "aktif"})
		baru := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenBaru", StatusAgen: "aktif", Upline: uplineBaru})
		dipindah, dibatalkan, err := pindahkanKomisiTx(ctx, tx, jamaah, baru)
		must(t, "pindah", err)
		if dipindah != 2 || dibatalkan != 0 {
			t.Fatalf("dipindah=%d dibatalkan=%d, mau 2 dan 0", dipindah, dibatalkan)
		}
		if l := bacaKomisi(t, tx, booking, "langsung"); l.penerima != baru || l.batal {
			t.Fatalf("komisi langsung = %+v", l)
		}
		if p := bacaKomisi(t, tx, booking, "pembinaan"); p.penerima != uplineBaru || p.batal {
			t.Fatalf("bonus pembinaan = %+v", p)
		}
	})

	for _, kasus := range []struct {
		nama         string
		statusUpline string
	}{{"upline agen baru nonaktif", "nonaktif"}, {"agen baru tanpa upline", ""}} {
		t.Run(kasus.nama, func(t *testing.T) {
			_, tx := testdb.Tx(t)
			brand, uplineLama, _, jamaah, booking := fixturePindahKomisi(t, tx)
			opts := testdb.AkunOpts{Nama: "AgenBaru", StatusAgen: "aktif"}
			if kasus.statusUpline != "" {
				opts.Upline = testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "UplineBaru", StatusAgen: kasus.statusUpline})
			}
			baru := testdb.NewAkun(t, tx, brand, opts)
			dipindah, dibatalkan, err := pindahkanKomisiTx(ctx, tx, jamaah, baru)
			must(t, "pindah", err)
			if dipindah != 1 || dibatalkan != 1 {
				t.Fatalf("dipindah=%d dibatalkan=%d, mau 1 dan 1", dipindah, dibatalkan)
			}
			p := bacaKomisi(t, tx, booking, "pembinaan")
			if !p.batal || p.sebab.String != komisi.SebabGantiKaitan || p.penerima != uplineLama {
				t.Fatalf("bonus pembinaan harus batal dengan sebab ganti kaitan dan tetap tercatat: %+v", p)
			}
		})
	}
}

// C5: saldo yang sudah negatif sebelum pemindahan tidak menghalangi; yang ditolak hanya pemindahan yang
// menurunkan saldo tersedia sampai di bawah nol.
func TestPindahKomisiDanSaldoNegatif(t *testing.T) {
	ctx := context.Background()
	cairkan := func(t *testing.T, tx *sql.Tx, brand, agen int64, nominal float64) {
		testdb.Exec(t, tx, `INSERT INTO pengajuan_pencairan (brand_id, akun_agen_id, nominal_diajukan, status, rekening_bank, rekening_nomor, rekening_atas_nama)
			VALUES (?,?,?,'disetujui','BSI','123','Agen')`, brand, agen, nominal)
	}

	t.Run("komisi yang sudah dicairkan tidak boleh dipindah", func(t *testing.T) {
		_, tx := testdb.Tx(t)
		brand, _, lama, jamaah, _ := fixturePindahKomisi(t, tx)
		cairkan(t, tx, brand, lama, 500_000)
		baru := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenBaru", StatusAgen: "aktif"})
		_, _, err := pindahkanKomisiTx(ctx, tx, jamaah, baru)
		mustErr(t, "sudah dicairkan", err, ErrKomisiSudahDicairkan)
	})

	t.Run("saldo negatif yang tidak berubah tidak menghalangi", func(t *testing.T) {
		_, tx := testdb.Tx(t)
		brand, _, lama, jamaah, booking := fixturePindahKomisi(t, tx)
		// Saldo agen lama sudah negatif karena pencairan lama yang lebih besar dari komisinya.
		cairkan(t, tx, brand, lama, 700_000)
		// Komisi jamaah ini belum tersedia (jadwalnya diundur ke masa depan), jadi tidak ikut saldo tersedia.
		testdb.Exec(t, tx, `UPDATE schedules s JOIN bookings b ON b.schedule_id=s.id SET s.berangkat_tanggal='2099-01-01' WHERE b.id=?`, booking)
		sebelum, err := komisi.HitungSaldo(ctx, tx, lama)
		must(t, "saldo", err)
		if sebelum.Tersedia >= 0 {
			t.Fatalf("prasyarat: saldo tersedia harus negatif, dapat %v", sebelum.Tersedia)
		}
		baru := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenBaru", StatusAgen: "aktif"})
		dipindah, _, err := pindahkanKomisiTx(ctx, tx, jamaah, baru)
		must(t, "pindah komisi tertahan", err)
		if dipindah != 1 {
			t.Fatalf("dipindah = %d, mau 1 (komisi langsung)", dipindah)
		}
	})
}

// C3: admin brand tidak bisa menyentuh pencairan atau kaitan jamaah brand lain.
func TestScopeBrandPencairanDanKaitan(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	_, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	agen := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Agen", StatusAgen: "aktif"})
	j := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "J"})
	res, err := tx.Exec(`INSERT INTO pengajuan_pencairan (brand_id, akun_agen_id, nominal_diajukan, status, rekening_bank, rekening_nomor, rekening_atas_nama)
		VALUES (?,?,500000,'pending','BSI','123','Agen')`, brand, agen)
	must(t, "pengajuan", err)
	id, _ := res.LastInsertId()
	lain := brand + 999_999

	must(t, "pencairan brand sendiri", brandPencairanTx(ctx, tx, id, &brand))
	must(t, "pencairan holding", brandPencairanTx(ctx, tx, id, nil))
	mustErr(t, "pencairan brand lain", brandPencairanTx(ctx, tx, id, &lain), ErrNotFound)
	mustErr(t, "pencairan tidak ada", brandPencairanTx(ctx, tx, id+999_999, &brand), ErrNotFound)

	must(t, "jamaah brand sendiri", brandJamaah(ctx, tx, j, &brand))
	must(t, "jamaah holding", brandJamaah(ctx, tx, j, nil))
	mustErr(t, "jamaah brand lain", brandJamaah(ctx, tx, j, &lain), ErrNotFound)
}

// Parameter angka pada filter komisi yang rusak ditolak, tidak diam-diam menjadi 0.
func TestAngkaQuery(t *testing.T) {
	for isi, mau := range map[string]int64{"": 0, "0": 0, "25": 25, " 7 ": 7} {
		got, err := angkaQuery(url.Values{"limit": {isi}}, "limit")
		if err != nil || got != mau {
			t.Errorf("angkaQuery(%q) = %d, %v; mau %d", isi, got, err, mau)
		}
	}
	for _, isi := range []string{"abc", "-1", "1.5", "10x"} {
		if _, err := angkaQuery(url.Values{"limit": {isi}}, "limit"); err == nil {
			t.Errorf("angkaQuery(%q) harus ditolak", isi)
		}
	}
}

// Mode kaitan yang tidak dikenal memakai sentinel yang dipetakan ke 400, bukan error polos yang menjadi 500.
func TestTetapkanKaitanModeTidakSah(t *testing.T) {
	_, tx := testdb.Tx(t)
	_, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	j := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "J"})
	mustErr(t, "mode asing", TetapkanKaitanJalur3(context.Background(), tx, j, "lainnya", 0), ErrKaitanModeTidakSah)
}

func TestStatusAgenHelper(t *testing.T) {
	for status, mau := range map[string]bool{StatusAgenTidakAktif: false, StatusAgenPengajuan: false, StatusAgenAktif: true, StatusAgenNonaktif: true} {
		if sudahAgen(status) != mau {
			t.Errorf("sudahAgen(%q) harus %v", status, mau)
		}
	}
	for status, mau := range map[string]bool{bayarMenungguVerifikasi: false, bayarTerverifikasi: true, bayarDibebaskan: true, "ditolak": false} {
		if pembayaranFinal(status) != mau {
			t.Errorf("pembayaranFinal(%q) harus %v", status, mau)
		}
	}
}
