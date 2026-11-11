package agen

// Nilai kolom customer_accounts.status_agen.
const (
	StatusAgenTidakAktif = "tidak_aktif" // belum pernah menjadi agen, atau pengajuannya ditolak
	StatusAgenPengajuan  = "pengajuan"
	StatusAgenAktif      = "aktif"
	StatusAgenNonaktif   = "nonaktif"
)

// sudahAgen: akun pernah disetujui menjadi agen (aktif atau nonaktif).
func sudahAgen(status string) bool {
	return status == StatusAgenAktif || status == StatusAgenNonaktif
}

// Nilai kolom pembayaran_pendaftaran_agen.status.
const (
	bayarMenungguVerifikasi = "menunggu_verifikasi"
	bayarTerverifikasi      = "terverifikasi"
	bayarDibebaskan         = "dibebaskan"
)

// pembayaranFinal: biaya pendaftaran sudah beres, baik dibayar dan diverifikasi maupun dibebaskan admin.
func pembayaranFinal(status string) bool {
	return status == bayarTerverifikasi || status == bayarDibebaskan
}

// Hasil keputusan admin atas satu siklus pengajuan agen.
const (
	keputusanDisetujui = "disetujui"
	keputusanDitolak   = "ditolak"
)

// Nilai kolom pengajuan_pencairan.status.
const (
	pencairanPending   = "pending"
	pencairanDisetujui = "disetujui"
	pencairanDitolak   = "ditolak"
)

// Nilai kolom jamaah.kaitan_status.
const (
	kaitanTerikatAgen = "terikat_agen"
	kaitanTanpaAgen   = "tanpa_agen"
)

// Pilihan admin saat menetapkan atau mengganti kaitan jamaah.
const (
	modeKaitanAgen      = "agen"
	modeKaitanTanpaAgen = "tanpa_agen"
)
