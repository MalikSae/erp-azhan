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

// Keputusan 2026-11-25: hanya dokumen wajib berstatus approved yang dihitung lengkap.
func TestHitungWajibDisetujuiHanyaApproved(t *testing.T) {
	items := susunDokumen("reguler", map[string]dokumenItem{
		"paspor":            {Status: "approved"},
		"ktp":               {Status: "submitted"},
		"pas_foto":          {Status: "rejected"},
		"kk":                {Status: "approved"}, // tambahan, tidak dihitung
		"vaksin_meningitis": {Status: "approved"},
	})
	setuju, total := hitungWajibDisetujui(items)
	if setuju != 1 || total != 3 {
		t.Fatalf("wajib disetujui = %d/%d, want 1/3", setuju, total)
	}
	items = susunDokumen("reguler", map[string]dokumenItem{
		"paspor": {Status: "approved"}, "ktp": {Status: "approved"}, "pas_foto": {Status: "approved"},
	})
	if setuju, total = hitungWajibDisetujui(items); setuju != 3 || total != 3 {
		t.Fatalf("semua disetujui = %d/%d, want 3/3", setuju, total)
	}
}

// Keputusan 2026-11-25: unggah dokumen oleh pelanggan dibuka setelah booking minimal DP.
func TestBolehUnggahDokumenMinimalDP(t *testing.T) {
	for status, want := range map[string]bool{
		"draft": false, "baru": false, "batal": false, "": false,
		"dp": true, "lunas": true, "dokumen_lengkap": true, "siap_berangkat": true,
	} {
		if got := bolehUnggahDokumen(status); got != want {
			t.Errorf("bolehUnggahDokumen(%q) = %v, want %v", status, got, want)
		}
	}
}

// Keputusan 2026-11-25: akses terbatas boleh mengganti dokumen ditolak; menunggu verifikasi hanya akses penuh;
// disetujui tidak bisa diganti siapa pun.
func TestPutuskanGantiDokumen(t *testing.T) {
	cases := []struct {
		status string
		penuh  bool
		want   keputusanGanti
	}{
		{"", false, gantiBoleh},
		{"", true, gantiBoleh},
		{"rejected", false, gantiBoleh},
		{"rejected", true, gantiBoleh},
		{"submitted", true, gantiBoleh},
		{"submitted", false, gantiMenungguVerifikasi},
		{"approved", true, gantiSudahDisetujui},
		{"approved", false, gantiSudahDisetujui},
	}
	for _, c := range cases {
		if got := putuskanGantiDokumen(c.status, c.penuh); got != c.want {
			t.Errorf("putuskanGantiDokumen(%q, penuh=%v) = %v, want %v", c.status, c.penuh, got, c.want)
		}
	}
}
