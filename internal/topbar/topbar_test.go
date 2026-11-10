package topbar

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"erp-azhan/api/internal/testdb"
)

var semuaIzin = []string{IzinPaket, IzinBooking, IzinJamaah, IzinPerubahanJamaah, IzinPencairan, IzinPengajuanAgen, IzinPembayaran}

// brandLain mengembalikan brand selain brand fixture; test batas brand dilewati bila hanya ada satu brand.
func brandLain(t *testing.T, tx *sql.Tx, selain int64) int64 {
	t.Helper()
	var id int64
	err := tx.QueryRowContext(context.Background(), `SELECT id FROM brands WHERE id != ? ORDER BY id LIMIT 1`, selain).Scan(&id)
	if err == sql.ErrNoRows {
		t.Skip("butuh minimal dua brand")
	}
	if err != nil {
		t.Fatalf("brand lain: %v", err)
	}
	return id
}

// kataUnik menghasilkan kata kunci yang tidak ada di data dev.
func kataUnik() string { return fmt.Sprintf("Zqx%d", time.Now().UnixNano()%1_000_000_000) }

// dataCari membuat satu paket, satu booking, dan satu jamaah brand fixture yang memuat kata kunci.
type dataCari struct {
	kata                          string
	brandID                       int64
	scheduleID, bookingID, jamaah int64
}

func siapkanDataCari(t *testing.T, tx *sql.Tx) dataCari {
	t.Helper()
	d := dataCari{kata: kataUnik()}
	d.scheduleID, d.brandID = testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2030-01-15"})
	testdb.Exec(t, tx, `UPDATE schedules SET jadwal_nama = ? WHERE id = ?`, "Umroh "+d.kata, d.scheduleID)
	d.jamaah = testdb.NewJamaah(t, tx, d.brandID, testdb.JamaahOpts{Nama: "Pemesan " + d.kata})
	d.bookingID, _ = testdb.NewBookingPax(t, tx, d.scheduleID, "baru", 1000, testdb.Pax{JamaahID: d.jamaah})
	return d
}

func TestCariMenemukanTiapJenis(t *testing.T) {
	_, tx := testdb.Tx(t)
	d := siapkanDataCari(t, tx)

	h, err := Cari(context.Background(), tx, Akses{Izin: semuaIzin}, "  "+d.kata+" ")
	if err != nil {
		t.Fatal(err)
	}
	if len(h.Paket) != 1 || h.Paket[0].ID != d.scheduleID || h.Paket[0].BerangkatTanggal != "2030-01-15" || h.Paket[0].Brand == "" {
		t.Errorf("paket = %+v", h.Paket)
	}
	// Booking ditemukan lewat nama pemesan.
	if len(h.Booking) != 1 || h.Booking[0].ID != d.bookingID || h.Booking[0].Status != "baru" {
		t.Errorf("booking = %+v", h.Booking)
	}
	if len(h.Jamaah) != 1 || h.Jamaah[0].ID != d.jamaah {
		t.Errorf("jamaah = %+v", h.Jamaah)
	}
}

func TestCariBookingLewatKodeDanJamaahLewatNIK(t *testing.T) {
	_, tx := testdb.Tx(t)
	d := siapkanDataCari(t, tx)
	var kode string
	if err := tx.QueryRowContext(context.Background(), `SELECT id_booking FROM bookings WHERE id = ?`, d.bookingID).Scan(&kode); err != nil {
		t.Fatal(err)
	}
	nik := "99" + fmt.Sprintf("%014d", time.Now().UnixNano()%100_000_000_000_000)
	testdb.Exec(t, tx, `UPDATE jamaah SET nik = ? WHERE id = ?`, nik, d.jamaah)

	h, err := Cari(context.Background(), tx, Akses{Izin: semuaIzin}, kode)
	if err != nil {
		t.Fatal(err)
	}
	// Booking fixture adalah yang terbaru, jadi berada di urutan pertama.
	if len(h.Booking) == 0 || h.Booking[0].ID != d.bookingID || h.Booking[0].IDBooking != kode {
		t.Errorf("booking lewat kode %q = %+v", kode, h.Booking)
	}
	h, err = Cari(context.Background(), tx, Akses{Izin: semuaIzin}, nik)
	if err != nil {
		t.Fatal(err)
	}
	if len(h.Jamaah) != 1 || h.Jamaah[0].ID != d.jamaah {
		t.Errorf("jamaah lewat NIK = %+v", h.Jamaah)
	}
}

func TestCariMengikutiIzin(t *testing.T) {
	_, tx := testdb.Tx(t)
	d := siapkanDataCari(t, tx)

	kasus := []struct {
		izin                  []string
		paket, booking, jamaa int
	}{
		{nil, 0, 0, 0},
		{[]string{IzinPaket}, 1, 0, 0},
		{[]string{IzinBooking}, 0, 1, 0},
		{[]string{IzinJamaah}, 0, 0, 1},
		// Izin memproses antrean tidak membuka pencarian.
		{[]string{IzinPembayaran, IzinPengajuanAgen}, 0, 0, 0},
	}
	for _, k := range kasus {
		h, err := Cari(context.Background(), tx, Akses{Izin: k.izin}, d.kata)
		if err != nil {
			t.Fatal(err)
		}
		if len(h.Paket) != k.paket || len(h.Booking) != k.booking || len(h.Jamaah) != k.jamaa {
			t.Errorf("izin %v: paket=%d booking=%d jamaah=%d", k.izin, len(h.Paket), len(h.Booking), len(h.Jamaah))
		}
	}
}

func TestCariDibatasiBrand(t *testing.T) {
	_, tx := testdb.Tx(t)
	d := siapkanDataCari(t, tx)
	lain := brandLain(t, tx, d.brandID)
	jamaahLain := testdb.NewJamaah(t, tx, lain, testdb.JamaahOpts{Nama: "Tetangga " + d.kata})

	// Admin brand fixture: melihat datanya sendiri, bukan jamaah brand lain.
	h, err := Cari(context.Background(), tx, Akses{BrandID: &d.brandID, Izin: semuaIzin}, d.kata)
	if err != nil {
		t.Fatal(err)
	}
	if len(h.Paket) != 1 || len(h.Booking) != 1 || len(h.Jamaah) != 1 || h.Jamaah[0].ID != d.jamaah {
		t.Errorf("brand sendiri: %+v", h)
	}

	// Admin brand lain: tidak melihat paket, booking, dan jamaah brand fixture.
	h, err = Cari(context.Background(), tx, Akses{BrandID: &lain, Izin: semuaIzin}, d.kata)
	if err != nil {
		t.Fatal(err)
	}
	if len(h.Paket) != 0 || len(h.Booking) != 0 || len(h.Jamaah) != 1 || h.Jamaah[0].ID != jamaahLain {
		t.Errorf("brand lain: %+v", h)
	}

	// Lintas brand (holding): kedua jamaah terlihat.
	h, err = Cari(context.Background(), tx, Akses{Izin: semuaIzin}, d.kata)
	if err != nil {
		t.Fatal(err)
	}
	if len(h.Jamaah) != 2 {
		t.Errorf("holding: jamaah = %+v", h.Jamaah)
	}
}

func TestCariKataPendekDanKarakterKhusus(t *testing.T) {
	_, tx := testdb.Tx(t)
	d := siapkanDataCari(t, tx)

	for _, kata := range []string{"", " ", "Z", " Z "} {
		h, err := Cari(context.Background(), tx, Akses{Izin: semuaIzin}, kata)
		if err != nil {
			t.Fatal(err)
		}
		if len(h.Paket)+len(h.Booking)+len(h.Jamaah) != 0 {
			t.Errorf("kata %q seharusnya kosong: %+v", kata, h)
		}
		if h.Paket == nil || h.Booking == nil || h.Jamaah == nil {
			t.Errorf("kata %q: daftar harus [] bukan null", kata)
		}
	}

	// % dan _ dicari apa adanya, bukan sebagai wildcard.
	for _, kata := range []string{"%%", "__", d.kata[:3] + "%" + d.kata[len(d.kata)-2:]} {
		h, err := Cari(context.Background(), tx, Akses{BrandID: &d.brandID, Izin: []string{IzinPaket}}, kata)
		if err != nil {
			t.Fatal(err)
		}
		for _, p := range h.Paket {
			if p.ID == d.scheduleID {
				t.Errorf("kata %q tidak boleh cocok sebagai wildcard", kata)
			}
		}
	}
}

func TestCariMembatasiJumlahHasil(t *testing.T) {
	_, tx := testdb.Tx(t)
	d := siapkanDataCari(t, tx)
	for i := 0; i < BatasHasil+2; i++ {
		testdb.NewJamaah(t, tx, d.brandID, testdb.JamaahOpts{Nama: fmt.Sprintf("Rombongan %s %d", d.kata, i)})
	}
	h, err := Cari(context.Background(), tx, Akses{Izin: semuaIzin}, d.kata)
	if err != nil {
		t.Fatal(err)
	}
	if len(h.Jamaah) != BatasHasil {
		t.Errorf("jamaah = %d, mau %d", len(h.Jamaah), BatasHasil)
	}
}

func jumlahAntrean(t *testing.T, tx *sql.Tx, a Akses) map[string]int {
	t.Helper()
	n, err := HitungNotifikasi(context.Background(), tx, a)
	if err != nil {
		t.Fatal(err)
	}
	m, total := map[string]int{}, 0
	for _, x := range n.Antrean {
		m[x.Jenis] = x.Jumlah
		total += x.Jumlah
	}
	if total != n.Total {
		t.Errorf("total = %d, jumlah antrean = %d", n.Total, total)
	}
	return m
}

func TestNotifikasiMenghitungTiapAntreanPerBrand(t *testing.T) {
	_, tx := testdb.Tx(t)
	scheduleID, brandID := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2030-01-15"})
	lain := brandLain(t, tx, brandID)

	holding := Akses{Izin: semuaIzin}
	sendiri := Akses{BrandID: &brandID, Izin: semuaIzin}
	tetangga := Akses{BrandID: &lain, Izin: semuaIzin}
	awalHolding, awalSendiri, awalTetangga := jumlahAntrean(t, tx, holding), jumlahAntrean(t, tx, sendiri), jumlahAntrean(t, tx, tetangga)

	// Satu butir baru di tiap antrean brand fixture, ditambah butir yang sudah diputuskan (tidak dihitung).
	jamaahID := testdb.NewJamaah(t, tx, brandID, testdb.JamaahOpts{Nama: "Antre"})
	testdb.Exec(t, tx, `INSERT INTO jamaah_perubahan (jamaah_id, sumber, status) VALUES (?,'pelanggan','menunggu')`, jamaahID)
	testdb.Exec(t, tx, `INSERT INTO jamaah_perubahan (jamaah_id, sumber, status) VALUES (?,'pelanggan','disetujui')`, jamaahID)

	agen := testdb.NewAkun(t, tx, brandID, testdb.AkunOpts{Nama: "Agen Antre", StatusAgen: "aktif"})
	for _, status := range []string{"pending", "disetujui"} {
		testdb.Exec(t, tx, `INSERT INTO pengajuan_pencairan (brand_id, akun_agen_id, nominal_diajukan, status, rekening_bank, rekening_nomor, rekening_atas_nama)
			VALUES (?,?,100000,?,'BCA','123','Agen Antre')`, brandID, agen, status)
	}

	testdb.NewAkun(t, tx, brandID, testdb.AkunOpts{Nama: "Calon Agen", StatusAgen: "pengajuan"})

	bookingID, _ := testdb.NewBookingPax(t, tx, scheduleID, "baru", 1000, testdb.Pax{JamaahID: jamaahID})
	testdb.AddPayment(t, tx, bookingID, 500, "pending")
	testdb.AddPayment(t, tx, bookingID, 500, "confirmed")

	semua := []string{JenisPerubahanJamaah, JenisPencairan, JenisPengajuanAgen, JenisPembayaran}
	akhirHolding, akhirSendiri, akhirTetangga := jumlahAntrean(t, tx, holding), jumlahAntrean(t, tx, sendiri), jumlahAntrean(t, tx, tetangga)
	for _, jenis := range semua {
		if akhirHolding[jenis] != awalHolding[jenis]+1 {
			t.Errorf("holding %s: %d -> %d, mau +1", jenis, awalHolding[jenis], akhirHolding[jenis])
		}
		if akhirSendiri[jenis] != awalSendiri[jenis]+1 {
			t.Errorf("brand sendiri %s: %d -> %d, mau +1", jenis, awalSendiri[jenis], akhirSendiri[jenis])
		}
		if akhirTetangga[jenis] != awalTetangga[jenis] {
			t.Errorf("brand lain %s: %d -> %d, mau tetap", jenis, awalTetangga[jenis], akhirTetangga[jenis])
		}
	}
}

func TestNotifikasiPembayaranBookingDraftTidakDihitung(t *testing.T) {
	_, tx := testdb.Tx(t)
	scheduleID, brandID := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2030-01-15"})
	a := Akses{BrandID: &brandID, Izin: []string{IzinPembayaran}}
	awal := jumlahAntrean(t, tx, a)[JenisPembayaran]

	jamaahID := testdb.NewJamaah(t, tx, brandID, testdb.JamaahOpts{Nama: "Draf"})
	bookingID, _ := testdb.NewBookingPax(t, tx, scheduleID, "draft", 1000, testdb.Pax{JamaahID: jamaahID})
	testdb.AddPayment(t, tx, bookingID, 500, "pending")

	if akhir := jumlahAntrean(t, tx, a)[JenisPembayaran]; akhir != awal {
		t.Errorf("pembayaran booking draft ikut terhitung: %d -> %d", awal, akhir)
	}
}

func TestNotifikasiMengikutiIzin(t *testing.T) {
	_, tx := testdb.Tx(t)

	n, err := HitungNotifikasi(context.Background(), tx, Akses{})
	if err != nil {
		t.Fatal(err)
	}
	if n.Antrean == nil || len(n.Antrean) != 0 || n.Total != 0 {
		t.Errorf("tanpa izin = %+v", n)
	}

	// Izin melihat data tidak membuka antrean; tiap izin proses membuka tepat satu antrean.
	n, _ = HitungNotifikasi(context.Background(), tx, Akses{Izin: []string{IzinPaket, IzinBooking, IzinJamaah}})
	if len(n.Antrean) != 0 {
		t.Errorf("izin lihat saja = %+v", n)
	}
	pasangan := map[string]string{
		IzinPerubahanJamaah: JenisPerubahanJamaah,
		IzinPencairan:       JenisPencairan,
		IzinPengajuanAgen:   JenisPengajuanAgen,
		IzinPembayaran:      JenisPembayaran,
	}
	for izin, jenis := range pasangan {
		n, err := HitungNotifikasi(context.Background(), tx, Akses{Izin: []string{izin}})
		if err != nil {
			t.Fatal(err)
		}
		if len(n.Antrean) != 1 || n.Antrean[0].Jenis != jenis {
			t.Errorf("izin %s = %+v", izin, n.Antrean)
		}
	}
}
