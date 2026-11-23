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
func DokumenWajibJamaah(paxType string) []string {
	if paxType == "infant" {
		return []string{"paspor", "akte_lahir", "pas_foto"}
	}
	return []string{"paspor", "ktp", "pas_foto"}
}
