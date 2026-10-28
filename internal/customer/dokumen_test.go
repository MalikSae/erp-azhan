package customer

import "testing"

func TestDokumenWajibPerTipe(t *testing.T) {
	reg := dokumenWajib("reguler")
	if len(reg) != 3 || reg[0] != "paspor" || reg[1] != "ktp" || reg[2] != "pas_foto" {
		t.Fatalf("wajib reguler: %v", reg)
	}
	inf := dokumenWajib("infant")
	if len(inf) != 3 || inf[1] != "akte_lahir" {
		t.Fatalf("wajib infant: %v", inf)
	}
	for _, tipe := range []string{"reguler", "infant"} {
		for _, j := range append(dokumenWajib(tipe), dokumenTambahan(tipe)...) {
			if labelDokumen[j] == "" {
				t.Errorf("%s: jenis %q tanpa label", tipe, j)
			}
		}
	}
}

func TestSusunDokumenGabungkanStatus(t *testing.T) {
	alasan := "Foto buram"
	got := susunDokumen("reguler", map[string]dokumenItem{
		"paspor": {Status: "approved"},
		"ktp":    {Status: "rejected", RejectionReason: &alasan},
	})
	if len(got) != 6 {
		t.Fatalf("jumlah: %d", len(got))
	}
	if got[0].Jenis != "paspor" || got[0].Status != "approved" || !got[0].Wajib {
		t.Errorf("paspor: %+v", got[0])
	}
	if got[1].Status != "rejected" || got[1].RejectionReason == nil || *got[1].RejectionReason != alasan {
		t.Errorf("ktp: %+v", got[1])
	}
	if got[2].Status != "belum_upload" || got[2].Label != "Pas foto" {
		t.Errorf("pas_foto: %+v", got[2])
	}
	if got[3].Wajib {
		t.Errorf("kk harus opsional: %+v", got[3])
	}
}
