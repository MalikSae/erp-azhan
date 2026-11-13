package benefit

import (
	"errors"
	"strings"
	"testing"
)

const ikonContoh = "solar:bed-bold-duotone"

func TestValidasiIsiMenolakTeksKosongDanTerlaluPanjang(t *testing.T) {
	if _, _, err := ValidasiIsi("   ", ikonContoh); err == nil {
		t.Fatal("teks kosong harus ditolak")
	}
	if _, _, err := ValidasiIsi(strings.Repeat("a", MaksTeks+1), ikonContoh); err == nil {
		t.Fatal("teks 31 karakter harus ditolak")
	}
}

func TestValidasiIsiMenerimaTepatTigaPuluhKarakter(t *testing.T) {
	// Batas dihitung per karakter, bukan per byte: huruf beraksen tidak boleh memakan jatah dua.
	teks := strings.Repeat("é", MaksTeks)
	hasil, _, err := ValidasiIsi(teks, ikonContoh)
	if err != nil {
		t.Fatalf("30 karakter harus diterima: %v", err)
	}
	if hasil != teks {
		t.Fatalf("teks berubah: %q", hasil)
	}
}

func TestValidasiIsiMerapikanSpasiTepi(t *testing.T) {
	teks, ikon, err := ValidasiIsi("  Hotel dekat Haram  ", " "+ikonContoh+" ")
	if err != nil {
		t.Fatal(err)
	}
	if teks != "Hotel dekat Haram" || ikon != ikonContoh {
		t.Fatalf("tidak dirapikan: %q %q", teks, ikon)
	}
}

func TestValidasiIsiMenolakIkonKosongDanDiLuarDaftar(t *testing.T) {
	if _, _, err := ValidasiIsi("Hotel dekat Haram", ""); err == nil {
		t.Fatal("ikon kosong harus ditolak")
	}
	if _, _, err := ValidasiIsi("Hotel dekat Haram", "solar:tidak-ada"); err == nil {
		t.Fatal("ikon di luar daftar harus ditolak")
	}
}

func TestDaftarIkonTanpaKembar(t *testing.T) {
	// Daftar ini kontrak dengan app: app hanya membundel ikon yang ada di sini.
	if len(DaftarIkon) != 35 {
		t.Fatalf("daftar ikon berisi %d, seharusnya 35", len(DaftarIkon))
	}
	lihat := map[string]bool{}
	for _, i := range DaftarIkon {
		if i.Key == "" || i.Label == "" {
			t.Fatalf("ikon tanpa kunci atau label: %+v", i)
		}
		if lihat[i.Key] {
			t.Fatalf("kunci ikon kembar: %s", i.Key)
		}
		lihat[i.Key] = true
	}
}

func TestValidasiDaftarID(t *testing.T) {
	kasus := []struct {
		nama string
		ids  []int64
		sah  bool
	}{
		{"kosong boleh untuk draf", nil, true},
		{"empat berbeda", []int64{1, 2, 3, 4}, true},
		{"lima ditolak", []int64{1, 2, 3, 4, 5}, false},
		{"kembar ditolak", []int64{1, 2, 2}, false},
		{"id nol ditolak", []int64{0}, false},
		{"id negatif ditolak", []int64{-3}, false},
	}
	for _, k := range kasus {
		err := ValidasiDaftarID(k.ids)
		if k.sah && err != nil {
			t.Errorf("%s: seharusnya sah, dapat %v", k.nama, err)
		}
		if !k.sah {
			var pe PilihanError
			if !errors.As(err, &pe) {
				t.Errorf("%s: seharusnya PilihanError, dapat %v", k.nama, err)
			}
		}
	}
}

func TestKePublikTidakPernahNil(t *testing.T) {
	// Web dan app membaca array; null membuat keduanya harus menjaga dua bentuk.
	if out := KePublik(nil); out == nil || len(out) != 0 {
		t.Fatalf("seharusnya array kosong, dapat %#v", out)
	}
	out := KePublik([]Ref{{ID: 7, Teks: "Makan 3x sehari", Ikon: "solar:chef-hat-bold-duotone", IsActive: false}})
	if len(out) != 1 || out[0].Teks != "Makan 3x sehari" || out[0].Ikon != "solar:chef-hat-bold-duotone" {
		t.Fatalf("hasil salah: %+v", out)
	}
}
