package tabungan

import (
	"strings"
	"testing"
	"time"
)

var sekarang = time.Date(2026, 11, 17, 10, 0, 0, 0, ZonaWIB)

func targetSah() Target {
	id := int64(1)
	return Target{
		Jenis: "paket", ScheduleID: &id, NamaPaket: "Umroh Reguler", BulanBerangkat: "2027-08",
		JumlahDewasa: 2, JumlahBayi: 0, TipeKamar: "Quad", TargetTotal: 60_000_000, DPMinimal: 10_000_000,
		PengingatAktif: true, PengingatTanggal: 5, PengingatJam: 9,
	}
}

func TestValidasiTarget(t *testing.T) {
	cases := []struct {
		nama string
		ubah func(*Target)
		want string
	}{
		{"sah", func(*Target) {}, ""},
		{"perkiraan tanpa jadwal sah", func(x *Target) { x.Jenis = "perkiraan"; x.ScheduleID = nil; x.NamaPaket = "" }, ""},
		{"jenis salah", func(x *Target) { x.Jenis = "lain" }, "Jenis target tidak valid."},
		{"paket tanpa jadwal", func(x *Target) { x.ScheduleID = nil }, "Pilih paket."},
		{"perkiraan dengan jadwal", func(x *Target) { x.Jenis = "perkiraan" }, "Data target tidak valid."},
		{"nama paket 151 huruf", func(x *Target) { x.NamaPaket = strings.Repeat("a", 151) }, "Nama paket maksimal 150 karakter."},
		{"nama paket 150 huruf sah", func(x *Target) { x.NamaPaket = strings.Repeat("a", 150) }, ""},
		{"bulan salah format", func(x *Target) { x.BulanBerangkat = "2027-8" }, "Bulan berangkat tidak valid."},
		{"bulan bukan bulan", func(x *Target) { x.BulanBerangkat = "2027-13" }, "Bulan berangkat tidak valid."},
		{"bulan lalu", func(x *Target) { x.BulanBerangkat = "2026-10" }, "Bulan berangkat tidak boleh lewat."},
		{"bulan lalu sama dengan tersimpan sah", func(x *Target) { x.BulanBerangkat = "2026-09" }, ""},
		{"bulan ini sah", func(x *Target) { x.BulanBerangkat = "2026-11" }, ""},
		{"36 bulan sah", func(x *Target) { x.BulanBerangkat = "2029-11" }, ""},
		{"37 bulan", func(x *Target) { x.BulanBerangkat = "2029-12" }, "Bulan berangkat maksimal 36 bulan lagi."},
		{"dewasa 0", func(x *Target) { x.JumlahDewasa = 0 }, "Jumlah dewasa minimal 1."},
		{"dewasa 10", func(x *Target) { x.JumlahDewasa = 10 }, "Jamaah maksimal 9 orang."},
		{"dewasa 9 sah", func(x *Target) { x.JumlahDewasa = 9 }, ""},
		{"bayi minus", func(x *Target) { x.JumlahBayi = -1 }, "Jumlah bayi tidak valid."},
		{"bayi lebih dari dewasa", func(x *Target) { x.JumlahBayi = 3 }, "Jumlah bayi tidak boleh melebihi dewasa."},
		{"5 dewasa 5 bayi", func(x *Target) { x.JumlahDewasa = 5; x.JumlahBayi = 5 }, "Jamaah maksimal 9 orang."},
		{"5 dewasa 4 bayi sah", func(x *Target) { x.JumlahDewasa = 5; x.JumlahBayi = 4 }, ""},
		{"bayi sama dengan dewasa sah", func(x *Target) { x.JumlahBayi = 2 }, ""},
		{"kamar huruf kecil", func(x *Target) { x.TipeKamar = "quad" }, "Pilih tipe kamar."},
		{"kamar Triple sah", func(x *Target) { x.TipeKamar = "Triple" }, ""},
		{"target kurang", func(x *Target) { x.TargetTotal = 999_999; x.DPMinimal = 0 }, "Target total Rp1.000.000 sampai Rp2.000.000.000."},
		{"target batas bawah sah", func(x *Target) { x.TargetTotal = 1_000_000; x.DPMinimal = 0 }, ""},
		{"target lebih", func(x *Target) { x.TargetTotal = 2_000_000_001 }, "Target total Rp1.000.000 sampai Rp2.000.000.000."},
		{"target batas atas sah", func(x *Target) { x.TargetTotal = 2_000_000_000 }, ""},
		{"dp minus", func(x *Target) { x.DPMinimal = -1 }, "DP tidak valid."},
		{"dp melebihi target", func(x *Target) { x.DPMinimal = 60_000_001 }, "DP tidak boleh melebihi target total."},
		{"dp sama dengan target sah", func(x *Target) { x.DPMinimal = 60_000_000 }, ""},
		{"tanggal pengingat 0", func(x *Target) { x.PengingatTanggal = 0 }, "Tanggal pengingat 1 sampai 28."},
		{"tanggal pengingat 29", func(x *Target) { x.PengingatTanggal = 29 }, "Tanggal pengingat 1 sampai 28."},
		{"jam pengingat 24", func(x *Target) { x.PengingatJam = 24 }, "Jam pengingat 0 sampai 23."},
		{"jam pengingat 0 sah", func(x *Target) { x.PengingatJam = 0 }, ""},
	}
	for _, c := range cases {
		t.Run(c.nama, func(t *testing.T) {
			x := targetSah()
			c.ubah(&x)
			if got := ValidasiTarget(&x, "2026-09", sekarang); got != c.want {
				t.Fatalf("pesan = %q, mau %q", got, c.want)
			}
		})
	}
}

func TestValidasiCatatan(t *testing.T) {
	cases := []struct {
		nama, tanggal string
		jumlah        int64
		wantTgl, want string
	}{
		{"default hari ini WIB", "", 1000, "2026-11-17", ""},
		{"jumlah kurang", "", 999, "", "Jumlah Rp1.000 sampai Rp1.000.000.000."},
		{"jumlah lebih", "", 1_000_000_001, "", "Jumlah Rp1.000 sampai Rp1.000.000.000."},
		{"jumlah batas atas", "", 1_000_000_000, "2026-11-17", ""},
		{"tanggal hari ini", "2026-11-17", 5000, "2026-11-17", ""},
		{"tanggal besok", "2026-11-18", 5000, "", "Tanggal tidak boleh lewat hari ini."},
		{"tanggal lalu", "2026-01-05", 5000, "2026-01-05", ""},
		{"tanggal awal 2020", "2020-01-01", 5000, "2020-01-01", ""},
		{"sebelum 2020", "2019-12-31", 5000, "", "Tanggal tidak boleh sebelum 2020."},
		{"format salah", "17-11-2026", 5000, "", "Tanggal tidak valid."},
		{"tanggal tidak ada", "2026-02-30", 5000, "", "Tanggal tidak valid."},
	}
	for _, c := range cases {
		t.Run(c.nama, func(t *testing.T) {
			tgl, pesan := ValidasiCatatan(c.jumlah, c.tanggal, sekarang)
			if pesan != c.want || tgl != c.wantTgl {
				t.Fatalf("tanggal=%q pesan=%q, mau %q %q", tgl, pesan, c.wantTgl, c.want)
			}
		})
	}
}

// Pukul 02:00 WIB tanggal 1 sudah bulan baru di WIB walau UTC masih bulan lalu.
func TestHariIniMemakaiWIB(t *testing.T) {
	utc := time.Date(2026, 11, 30, 18, 0, 0, 0, time.UTC) // 1 Desember 01:00 WIB
	tgl, pesan := ValidasiCatatan(1000, "", utc)
	if pesan != "" || tgl != "2026-12-01" {
		t.Fatalf("tanggal=%q pesan=%q", tgl, pesan)
	}
	x := targetSah()
	x.BulanBerangkat = "2026-11"
	if got := ValidasiTarget(&x, "", utc); got != "Bulan berangkat tidak boleh lewat." {
		t.Fatalf("bulan berjalan WIB: %q", got)
	}
}

func TestHitungRingkasan(t *testing.T) {
	cases := []struct {
		nama            string
		ubah            func(*Target)
		tercatat, bulan int64
		want            Ringkasan
	}{
		{
			"awal: pembulatan ke atas kelipatan 1000",
			func(x *Target) { x.TargetTotal = 12_000_000; x.DPMinimal = 3_000_000 },
			0, 0,
			Ringkasan{Sisa: 12_000_000, SisaBulan: 9, PerBulan: 1_334_000, PerBulanIni: 1_334_000, KurangBulanIni: 1_334_000},
		},
		{
			"habis dibagi tetap kelipatan 1000",
			func(x *Target) { x.TargetTotal = 9_000_000; x.DPMinimal = 0 },
			0, 0,
			Ringkasan{Sisa: 9_000_000, SisaBulan: 9, PerBulan: 1_000_000, PerBulanIni: 1_000_000, KurangBulanIni: 1_000_000, DPTercapai: true},
		},
		{
			"sebagian, DP tercapai",
			func(x *Target) { x.TargetTotal = 20_000_000; x.DPMinimal = 5_000_000 },
			5_000_000, 2_000_000,
			Ringkasan{Tercatat: 5_000_000, Sisa: 15_000_000, Persen: 25, SisaBulan: 9, PerBulan: 1_667_000, TercatatBulanIni: 2_000_000, PerBulanIni: 1_889_000, DPTercapai: true},
		},
		{
			"berangkat bulan ini: sisa bulan minimal 1",
			func(x *Target) { x.BulanBerangkat = "2026-11"; x.TargetTotal = 10_000_000; x.DPMinimal = 0 },
			4_000_000, 0,
			Ringkasan{Tercatat: 4_000_000, Sisa: 6_000_000, Persen: 40, SisaBulan: 1, PerBulan: 6_000_000, PerBulanIni: 6_000_000, KurangBulanIni: 6_000_000, DPTercapai: true},
		},
		{
			"berangkat bulan depan: sisa bulan 1",
			func(x *Target) { x.BulanBerangkat = "2026-12"; x.TargetTotal = 10_000_000; x.DPMinimal = 0 },
			0, 0,
			Ringkasan{Sisa: 10_000_000, SisaBulan: 1, PerBulan: 10_000_000, PerBulanIni: 10_000_000, KurangBulanIni: 10_000_000, DPTercapai: true},
		},
		{
			"bulan berangkat sudah lewat: sisa bulan tetap 1",
			func(x *Target) { x.BulanBerangkat = "2026-01"; x.TargetTotal = 10_000_000; x.DPMinimal = 0 },
			1_000_000, 0,
			Ringkasan{Tercatat: 1_000_000, Sisa: 9_000_000, Persen: 10, SisaBulan: 1, PerBulan: 9_000_000, PerBulanIni: 9_000_000, KurangBulanIni: 9_000_000, DPTercapai: true, Lewat: true},
		},
		{
			"persen dibulatkan ke bawah",
			func(x *Target) { x.TargetTotal = 3_000_000; x.DPMinimal = 0; x.BulanBerangkat = "2026-12" },
			1_000_000, 0,
			Ringkasan{Tercatat: 1_000_000, Sisa: 2_000_000, Persen: 33, SisaBulan: 1, PerBulan: 2_000_000, PerBulanIni: 2_000_000, KurangBulanIni: 2_000_000, DPTercapai: true},
		},
		{
			"tercapai tepat: sisa 0, per bulan 0",
			func(x *Target) { x.TargetTotal = 10_000_000; x.DPMinimal = 2_000_000 },
			10_000_000, 1_000_000,
			Ringkasan{Tercatat: 10_000_000, Persen: 100, SisaBulan: 9, TercatatBulanIni: 1_000_000, PerBulanIni: 112_000, DPTercapai: true, Tercapai: true},
		},
		{
			"melebihi target: persen maksimal 100, sisa tidak minus",
			func(x *Target) { x.TargetTotal = 10_000_000; x.DPMinimal = 2_000_000 },
			15_000_000, 0,
			Ringkasan{Tercatat: 15_000_000, Persen: 100, SisaBulan: 9, DPTercapai: true, Tercapai: true},
		},
		{
			"DP belum tercapai",
			func(x *Target) { x.TargetTotal = 10_000_000; x.DPMinimal = 5_000_000 },
			4_999_000, 0,
			Ringkasan{Tercatat: 4_999_000, Sisa: 5_001_000, Persen: 49, SisaBulan: 9, PerBulan: 556_000, PerBulanIni: 556_000, KurangBulanIni: 556_000},
		},
	}
	for _, c := range cases {
		t.Run(c.nama, func(t *testing.T) {
			x := targetSah()
			x.Jenis, x.ScheduleID = "perkiraan", nil
			c.want.PaketTayang = true
			c.ubah(&x)
			got := Hitung(x, Jadwal{}, c.tercatat, c.bulan, sekarang)
			if got != c.want {
				t.Fatalf("got  %+v\nwant %+v", got, c.want)
			}
		})
	}
}

// Jatah bulan ini dihitung dari sisa di awal bulan, jadi catatan bulan ini tidak menurunkan jatahnya.
func TestPerBulanIni(t *testing.T) {
	x := targetSah()
	x.Jenis, x.ScheduleID = "perkiraan", nil
	x.BulanBerangkat, x.TargetTotal, x.DPMinimal = "2027-11", 12_000_000, 0
	r := Hitung(x, Jadwal{}, 500_000, 500_000, sekarang)
	if r.SisaBulan != 12 || r.PerBulan != 959_000 || r.PerBulanIni != 1_000_000 || r.KurangBulanIni != 500_000 {
		t.Fatalf("got %+v", r)
	}
	// Catatan bulan lalu ikut mengurangi sisa awal bulan.
	r = Hitung(x, Jadwal{}, 1_400_000, 200_000, sekarang)
	if r.PerBulanIni != 900_000 || r.KurangBulanIni != 700_000 {
		t.Fatalf("dengan catatan lama: %+v", r)
	}
	// Bulan ini sudah melebihi jatah: kurang 0.
	r = Hitung(x, Jadwal{}, 3_000_000, 3_000_000, sekarang)
	if r.PerBulanIni != 1_000_000 || r.KurangBulanIni != 0 {
		t.Fatalf("melebihi jatah: %+v", r)
	}
}

func TestHitungPaket(t *testing.T) {
	berangkat := func(tgl, status string, hari int) Jadwal {
		return Jadwal{Ada: true, BerangkatTanggal: tgl, Status: status, BatasPelunasanHari: hari}
	}
	cases := []struct {
		nama      string
		bulan     string
		j         Jadwal
		sisaBulan int
		batas     string
		tayang    bool
		lewat     bool
	}{
		{"sampai bulan batas pelunasan", "2027-08", berangkat("2027-08-20", "published", 30), 8, "2027-07-21", true, false},
		{"batas pelunasan 0 hari: bulan berangkat", "2027-08", berangkat("2027-08-20", "published", 0), 9, "2027-08-20", true, false},
		{"batas pelunasan lewat: minimal 1", "2026-12", berangkat("2026-12-05", "published", 30), 1, "2026-11-05", true, false},
		{"draft tidak tayang", "2027-08", berangkat("2027-08-20", "draft", 30), 8, "2027-07-21", false, false},
		{"berangkat kemarin tidak tayang", "2026-11", berangkat("2026-11-16", "published", 30), 1, "2026-10-17", false, false},
		{"berangkat hari ini tayang", "2026-11", berangkat("2026-11-17", "published", 30), 1, "2026-10-18", true, false},
		{"bulan lewat", "2026-10", berangkat("2026-10-20", "published", 30), 1, "2026-09-20", false, true},
		{"jadwal tidak ada", "2027-08", Jadwal{}, 9, "", false, false},
	}
	for _, c := range cases {
		t.Run(c.nama, func(t *testing.T) {
			x := targetSah()
			x.BulanBerangkat, x.TargetTotal = c.bulan, 16_000_000
			r := Hitung(x, c.j, 0, 0, sekarang)
			batas := ""
			if r.BatasPelunasan != nil {
				batas = *r.BatasPelunasan
			}
			if r.SisaBulan != c.sisaBulan || batas != c.batas || r.PaketTayang != c.tayang || r.Lewat != c.lewat {
				t.Fatalf("sisa_bulan=%d batas=%q tayang=%v lewat=%v", r.SisaBulan, batas, r.PaketTayang, r.Lewat)
			}
			if want := bagiPerBulan(16_000_000, c.sisaBulan); r.PerBulan != want || r.PerBulanIni != want {
				t.Fatalf("per_bulan=%d per_bulan_ini=%d, mau %d", r.PerBulan, r.PerBulanIni, want)
			}
		})
	}
}

func TestTokenValid(t *testing.T) {
	ok := []string{"ExponentPushToken[abc-DEF_123]", "ExpoPushToken[xyz]"}
	salah := []string{"", "abc", "ExponentPushToken[]", "ExponentPushToken[a b]", "ExponentPushToken[abc]x",
		"xExponentPushToken[abc]", "ExponentPushToken[abc", "FcmToken[abc]", "ExponentPushToken[" + strings.Repeat("a", 300) + "]"}
	for _, s := range ok {
		if !TokenValid(s) {
			t.Errorf("harus sah: %q", s)
		}
	}
	for _, s := range salah {
		if TokenValid(s) {
			t.Errorf("harus ditolak: %q", s)
		}
	}
}
