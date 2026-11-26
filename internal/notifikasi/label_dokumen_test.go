package notifikasi

import "testing"

// D4: semua jenis dokumen, termasuk vaksin meningitis, punya label di notifikasi (bukan kunci mentah).
func TestLabelDokumenLengkap(t *testing.T) {
	for jenis, want := range map[string]string{
		"pas_foto": "Pas foto", "paspor": "Paspor", "ktp": "KTP", "kk": "Kartu keluarga",
		"buku_nikah": "Buku nikah", "akte_lahir": "Akta lahir", "vaksin_meningitis": "Vaksin meningitis",
	} {
		if got := labelDokumen[jenis]; got != want {
			t.Errorf("label %s = %q, want %q", jenis, got, want)
		}
	}
}
