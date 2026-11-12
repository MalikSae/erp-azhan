package analitik

import (
	"context"
	"testing"
	"time"

	"erp-azhan/api/internal/agen"
	"erp-azhan/api/internal/testdb"
)

func TestTandaSeat(t *testing.T) {
	kasus := []struct {
		sisaHari, persen int
		mau              string
	}{
		{30, 0, TandaMerah},
		{45, 69, TandaMerah},
		{45, 70, ""},
		{46, 69, ""},
		{46, 39, TandaKuning},
		{90, 39, TandaKuning},
		{90, 40, ""},
		{91, 0, ""},
	}
	for _, k := range kasus {
		if got := tandaSeat(k.sisaHari, k.persen); got != k.mau {
			t.Errorf("tandaSeat(%d, %d) = %q, mau %q", k.sisaHari, k.persen, got, k.mau)
		}
	}
}

func TestKurangSeat(t *testing.T) {
	// Target 70%: 45 seat butuh 32 (31,5 dibulatkan ke atas).
	if got := kurangSeat(45, 0); got != 32 {
		t.Errorf("kurangSeat(45, 0) = %d, mau 32", got)
	}
	if got := kurangSeat(45, 40); got != 0 {
		t.Errorf("kurangSeat(45, 40) = %d, mau 0", got)
	}
}

func TestUrutkanSeatMerahKuningLaluKekurangan(t *testing.T) {
	items := []SeatJadwal{
		{ScheduleID: 1, BerangkatTanggal: "2026-12-20"},
		{ScheduleID: 5, BerangkatTanggal: "2026-12-15"},
		{ScheduleID: 2, BerangkatTanggal: "2027-01-30", Tanda: TandaKuning, KurangSeat: 40},
		{ScheduleID: 3, BerangkatTanggal: "2026-12-14", Tanda: TandaMerah, KurangSeat: 10},
		{ScheduleID: 4, BerangkatTanggal: "2026-12-20", Tanda: TandaMerah, KurangSeat: 30},
	}
	urutkanSeat(items)
	// Merah urut kekurangan terbanyak, lalu kuning, lalu tanpa tanda urut tanggal.
	mau := []int64{4, 3, 2, 5, 1}
	for i, s := range items {
		if s.ScheduleID != mau[i] {
			t.Fatalf("urutan ke-%d = jadwal %d, mau %d", i, s.ScheduleID, mau[i])
		}
	}
}

func TestPilihAgenTerbaik(t *testing.T) {
	var peringkat []agen.PeringkatItem
	// Hanya bonus pembinaan (pax closing 0) tidak masuk daftar.
	peringkat = append(peringkat, agen.PeringkatItem{AkunID: 99, NamaLengkap: "Leader", PaxClosing: 0, TotalKomisi: 9_000_000})
	peringkat = append(peringkat,
		agen.PeringkatItem{AkunID: 1, NamaLengkap: "Budi", PaxClosing: 3, TotalKomisi: 1_000_000},
		agen.PeringkatItem{AkunID: 2, NamaLengkap: "Ani", PaxClosing: 3, TotalKomisi: 1_000_000},
		agen.PeringkatItem{AkunID: 3, NamaLengkap: "Citra", PaxClosing: 3, TotalKomisi: 2_000_000},
		agen.PeringkatItem{AkunID: 4, NamaLengkap: "Dodi", PaxClosing: 5, TotalKomisi: 100},
	)
	for i := 0; i < 12; i++ {
		peringkat = append(peringkat, agen.PeringkatItem{AkunID: int64(100 + i), NamaLengkap: "Z", PaxClosing: 1})
	}
	hasil := pilihAgenTerbaik(peringkat)
	if len(hasil) != jumlahAgenTerbaik {
		t.Fatalf("jumlah agen = %d, mau %d", len(hasil), jumlahAgenTerbaik)
	}
	// Pax closing dulu, lalu komisi, lalu nama.
	mau := []int64{4, 3, 2, 1}
	for i, id := range mau {
		if hasil[i].AkunID != id || hasil[i].Peringkat != i+1 {
			t.Fatalf("peringkat %d = akun %d (#%d), mau akun %d", i+1, hasil[i].AkunID, hasil[i].Peringkat, id)
		}
	}
	for _, a := range hasil {
		if a.AkunID == 99 {
			t.Fatal("agen tanpa pax closing tidak boleh masuk daftar")
		}
	}
}

// Simulasi: jadwal berangkat 40 hari lagi (masih bisa dijual online, rentang merah H-30 sampai H-45). Booking DP berisi Budi (terikat agen Rudi), Siti (belum
// ditentukan, booking web tanpa agen), Ahmad (tanpa agen dipilih admin), dan bayi Rina (infant).
// Booking baru (belum bayar) berisi Dewi. Pax DP lama 40 hari lalu: Yusuf, terikat Rudi.
//   - Pax per channel 30 hari: agen +1 (Budi), tanpa agen +3 (Siti, Ahmad, Rina); periode lalu agen +1.
//   - Seat terisi: +3 (infant tidak memakai seat), Dewi hanya hold.
func TestRingkasMarketing(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()

	sched, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	var hariIni time.Time
	if err := tx.QueryRowContext(ctx, `SELECT CURDATE()`).Scan(&hariIni); err != nil {
		t.Fatal(err)
	}
	testdb.Exec(t, tx, `UPDATE schedules SET status='published', berangkat_tanggal=? WHERE id=?`,
		hariIni.AddDate(0, 0, 40).Format("2006-01-02"), sched)

	awal, err := ringkasTanpaAgen(ctx, tx, &brand)
	if err != nil {
		t.Fatal(err)
	}
	seatAwal := cariSeat(t, awal, sched)

	rudi := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Rudi", StatusAgen: "aktif"})
	budi := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Budi", Direkrut: rudi})
	siti := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Siti"})
	ahmad := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Ahmad", TanpaAgen: true})
	rina := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Rina"})
	dewi := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Dewi"})
	yusuf := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Yusuf", Direkrut: rudi})

	_, paxDP := testdb.NewBookingPax(t, tx, sched, "dp", 4_000_000,
		testdb.Pax{JamaahID: budi}, testdb.Pax{JamaahID: siti}, testdb.Pax{JamaahID: ahmad}, testdb.Pax{JamaahID: rina})
	testdb.Exec(t, tx, `UPDATE booking_pax SET pax_type='infant', room_type=NULL WHERE id=?`, paxDP[3])
	testdb.NewBookingPax(t, tx, sched, "baru", 1_000_000, testdb.Pax{JamaahID: dewi})
	_, paxLama := testdb.NewBookingPax(t, tx, sched, "lunas", 1_000_000, testdb.Pax{JamaahID: yusuf})
	testdb.Exec(t, tx, `UPDATE booking_pax SET created_at = NOW() - INTERVAL 40 DAY WHERE id=?`, paxLama[0])

	// Kuota dibuat supaya terisi tepat 50% dan satu seat dipakai hold (Dewi).
	terisi := seatAwal.Terisi + 4 // Budi, Siti, Ahmad, Yusuf
	testdb.Exec(t, tx, `UPDATE schedules SET seat_total=?, seat_sisa=? WHERE id=?`, terisi*2, terisi*2-terisi-1, sched)

	akhir, err := ringkasTanpaAgen(ctx, tx, &brand)
	if err != nil {
		t.Fatal(err)
	}
	pa, pk := awal.PaxChannel, akhir.PaxChannel
	if d := pk.Agen - pa.Agen; d != 1 {
		t.Errorf("pax dari agen bertambah %d, mau 1 (Budi)", d)
	}
	if d := pk.TanpaAgen - pa.TanpaAgen; d != 3 {
		t.Errorf("pax tanpa agen bertambah %d, mau 3 (Siti, Ahmad, Rina)", d)
	}
	if d := pk.AgenSebelumnya - pa.AgenSebelumnya; d != 1 {
		t.Errorf("pax agen periode lalu bertambah %d, mau 1 (Yusuf)", d)
	}
	if len(pk.PerBrand) != 1 || pk.PerBrand[0].BrandID != brand ||
		pk.PerBrand[0].Agen != pk.Agen || pk.PerBrand[0].TanpaAgen != pk.TanpaAgen {
		t.Errorf("rincian per brand tidak cocok dengan total: %+v", pk.PerBrand)
	}

	s := cariSeat(t, akhir, sched)
	if s.Terisi != terisi {
		t.Errorf("seat terisi = %d, mau %d (infant dan booking baru tidak dihitung)", s.Terisi, terisi)
	}
	if s.Hold != 1 || s.Persen != 50 || s.SisaHari != 40 || s.Tanda != TandaMerah {
		t.Errorf("seat jadwal = %+v, mau hold 1, 50%%, 40 hari, merah", s)
	}
	if mau := kurangSeat(terisi*2, terisi); s.KurangSeat != mau || mau == 0 {
		t.Errorf("kurang seat = %d, mau %d", s.KurangSeat, mau)
	}

	// Grafik harian: pax hari ini bertambah sesuai channel.
	hariIniStr := hariIni.Format("2006-01-02")
	ha, hk := cariHarian(awal, hariIniStr), cariHarian(akhir, hariIniStr)
	if hk.Agen-ha.Agen != 1 || hk.TanpaAgen-ha.TanpaAgen != 3 {
		t.Errorf("harian hari ini = %+v (awal %+v), mau agen +1, tanpa agen +3", hk, ha)
	}

	// Jadwal di bawah batas jual online (H-30) tidak tampil.
	testdb.Exec(t, tx, "UPDATE schedules SET berangkat_tanggal=? WHERE id=?", hariIni.AddDate(0, 0, 20).Format("2006-01-02"), sched)
	dekat, err := ringkasTanpaAgen(ctx, tx, &brand)
	if err != nil {
		t.Fatal(err)
	}
	for _, j := range dekat.SeatTerisi {
		if j.ScheduleID == sched {
			t.Errorf("jadwal H-20 masih tampil, padahal booking online sudah ditutup")
		}
	}

	// Scope brand: brand lain tidak melihat jadwal maupun pax brand ini.
	lain := brand + 1_000_000
	r, err := ringkasTanpaAgen(ctx, tx, &lain)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.SeatTerisi) != 0 || r.PaxChannel.Agen+r.PaxChannel.TanpaAgen != 0 {
		t.Errorf("brand lain melihat data: %+v", r)
	}
}

func cariSeat(t *testing.T, r *MarketingRingkas, scheduleID int64) SeatJadwal {
	t.Helper()
	for _, s := range r.SeatTerisi {
		if s.ScheduleID == scheduleID {
			return s
		}
	}
	t.Fatalf("jadwal %d tidak ada di seat terisi", scheduleID)
	return SeatJadwal{}
}

func cariHarian(r *MarketingRingkas, tanggal string) PaxHarian {
	for _, h := range r.PaxChannel.Harian {
		if h.Tanggal == tanggal {
			return h
		}
	}
	return PaxHarian{Tanggal: tanggal}
}

// Aktivasi: agen aktif dihitung per brand; agen closing hanya yang aktif dan punya pax closing.
func TestAktivasiAgen(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	_, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})

	awal, err := aktivasiAgen(ctx, tx, &brand, nil)
	if err != nil {
		t.Fatal(err)
	}
	testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Agen Baru", StatusAgen: "aktif"})
	testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Agen Lama", StatusAgen: "nonaktif"})

	peringkat := []agen.PeringkatItem{
		{AkunID: 1, StatusAgen: "aktif", PaxClosing: 3},
		{AkunID: 2, StatusAgen: "nonaktif", PaxClosing: 5},
		{AkunID: 3, StatusAgen: "aktif", PaxClosing: 0},
	}
	akhir, err := aktivasiAgen(ctx, tx, &brand, peringkat)
	if err != nil {
		t.Fatal(err)
	}
	if akhir.AgenAktif-awal.AgenAktif != 1 {
		t.Errorf("agen aktif bertambah %d, mau 1", akhir.AgenAktif-awal.AgenAktif)
	}
	if akhir.AgenClosing != 1 {
		t.Errorf("agen closing = %d, mau 1", akhir.AgenClosing)
	}
}
