package shared

import "strings"

// Aturan kelengkapan jamaah yang dipakai lebih dari satu modul (portal pelanggan dan portal agen), supaya angkanya tidak berbeda.

// KunciDataWajibJamaah: kolom tabel jamaah yang wajib terisi, sama dengan kunci JSON di customer.dataWajibJamaah.
var KunciDataWajibJamaah = []string{
	"tempat_lahir", "tanggal_lahir", "nama_ayah_kandung", "no_paspor", "paspor_berlaku_sampai",
	"alamat", "kota", "no_hp", "emergency_nama", "emergency_hp",
}

// DataWajibBerlaku: infant tidak wajib kontak darurat dan nomor WhatsApp (kontak ada di pemesan).
func DataWajibBerlaku(paxType, kunci string) bool {
	return !(paxType == "infant" && (strings.HasPrefix(kunci, "emergency_") || kunci == "no_hp"))
}

// DokumenWajibJamaah: jenis dokumen yang harus disetujui admin sebelum jamaah dianggap lengkap.
// Bayi: paspor, KK, pas foto; akta lahir tidak wajib (keputusan 2026-11-26).
func DokumenWajibJamaah(paxType string) []string {
	if paxType == "infant" {
		return []string{"paspor", "kk", "pas_foto"}
	}
	return []string{"paspor", "ktp", "pas_foto"}
}

// LabelDokumen: label jenis dokumen jamaah (sentence case), satu peta untuk portal pelanggan, notifikasi, dan admin.
// Kuncinya sama dengan enum dokumen_jamaah.jenis.
var LabelDokumen = map[string]string{
	"pas_foto":          "Pas foto",
	"paspor":            "Paspor",
	"ktp":               "KTP",
	"kk":                "Kartu keluarga",
	"buku_nikah":        "Buku nikah",
	"akte_lahir":        "Akta lahir",
	"vaksin_meningitis": "Vaksin meningitis",
}

// HitungDokumenWajib: jumlah dokumen wajib (sesuai tipe pax) yang ada di disetujui, dan jumlah dokumen wajib.
func HitungDokumenWajib(paxType string, disetujui map[string]bool) (setuju, total int) {
	wajib := DokumenWajibJamaah(paxType)
	for _, j := range wajib {
		if disetujui[j] {
			setuju++
		}
	}
	return setuju, len(wajib)
}
