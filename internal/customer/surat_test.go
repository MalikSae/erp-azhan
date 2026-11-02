package customer

import "testing"

func dataLengkap() *dataJamaah {
	return &dataJamaah{JamaahID: 7, NamaLengkap: "Syahid Malik", PaxType: "reguler", TempatLahir: "Jakarta", TanggalLahir: "1990-01-12"}
}

func pesananDP() *orderSummary {
	return &orderSummary{Tab: "sebagian", MinimalDP: 10000000, Terbayar: 10500000}
}

func itemJenis(items []suratItem, jenis string) *suratItem {
	for i := range items {
		if items[i].Jenis == jenis {
			return &items[i]
		}
	}
	return nil
}

func TestEvaluasiSurat_JamaahLengkapSetelahDP(t *testing.T) {
	items := evaluasiSurat(pesananDP(), dataLengkap(), true, nil)
	if len(items) != 2 {
		t.Fatalf("tanpa paspor harus mendapat 2 surat, dapat %d", len(items))
	}
	for _, it := range items {
		if !it.Bisa || it.Alasan != "" || it.Slug == "" {
			t.Errorf("%s harus bisa diunduh: %+v", it.Jenis, it)
		}
	}
}

func TestEvaluasiSurat_RekomPasporHanyaBilaBelumPunyaPaspor(t *testing.T) {
	d := dataLengkap()
	d.NoPaspor = "A1234567"
	items := evaluasiSurat(pesananDP(), d, true, nil)
	if len(items) != 1 || items[0].Jenis != "keterangan_umroh" {
		t.Fatalf("jamaah berpaspor hanya mendapat surat keterangan: %+v", items)
	}
}

func TestEvaluasiSurat_BayiTidakMendapatSurat(t *testing.T) {
	d := dataLengkap()
	d.PaxType = "infant"
	if items := evaluasiSurat(pesananDP(), d, true, nil); len(items) != 0 {
		t.Fatalf("bayi tidak mendapat surat: %+v", items)
	}
}

func TestEvaluasiSurat_SyaratUnduh(t *testing.T) {
	sebelumDP := &orderSummary{Tab: "sebagian", MinimalDP: 10000000, Terbayar: 7500000}
	if it := evaluasiSurat(sebelumDP, dataLengkap(), true, nil)[0]; it.Bisa || it.Alasan != "Tersedia setelah DP terkonfirmasi." {
		t.Errorf("sebelum DP: %+v", it)
	}
	if it := evaluasiSurat(pesananDP(), dataLengkap(), false, nil)[0]; it.Bisa || it.Alasan != "Surat belum tersedia. Hubungi kami." {
		t.Errorf("pengaturan brand belum lengkap: %+v", it)
	}
	kurang := dataLengkap()
	kurang.TempatLahir, kurang.TanggalLahir = "", ""
	if it := evaluasiSurat(pesananDP(), kurang, true, nil)[0]; it.Bisa || it.Alasan != "Lengkapi tempat lahir dan tanggal lahir terlebih dahulu." {
		t.Errorf("data kurang: %+v", it)
	}
	batal := &orderSummary{Tab: "batal", MinimalDP: 10000000, Terbayar: 10500000}
	if it := evaluasiSurat(batal, dataLengkap(), true, nil)[0]; it.Bisa {
		t.Errorf("booking batal tidak boleh bisa unduh: %+v", it)
	}
}

func TestEvaluasiSurat_NomorTerbitDitampilkan(t *testing.T) {
	items := evaluasiSurat(pesananDP(), dataLengkap(), true, map[string]string{"7|keterangan_umroh": "100808/HAW/SKU/XI/2026"})
	if it := itemJenis(items, "keterangan_umroh"); it == nil || it.Nomor != "100808/HAW/SKU/XI/2026" {
		t.Fatalf("nomor harus tampil: %+v", items)
	}
}

func TestDPTerpenuhi(t *testing.T) {
	if !dpTerpenuhi(&orderSummary{Tab: "lunas"}) {
		t.Error("lunas = DP terpenuhi")
	}
	if dpTerpenuhi(&orderSummary{Tab: "menunggu", MinimalDP: 10000000, Terbayar: 9999999}) {
		t.Error("kurang dari DP minimal")
	}
	if !dpTerpenuhi(&orderSummary{Tab: "sebagian", MinimalDP: 10000000, Terbayar: 10000000}) {
		t.Error("tepat DP minimal")
	}
}
