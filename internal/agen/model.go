// Package agen mengelola alur menjadi Agen Umroh ("Syiar"): pengajuan,
// kelengkapan data, pembayaran pendaftaran, dan keputusan keagenan
// (agen-azhan.md Bagian 3.6, 7.1, 7.9–7.11).
package agen

import (
	"errors"
	"time"
)

var (
	ErrNotFound = errors.New("data tidak ditemukan")
	// ErrMoUWajib: brand punya draft MoU agen dan pemohon belum menyetujuinya.
	ErrMoUWajib                = errors.New("anda wajib membaca dan menyetujui MoU Agen")
	ErrSudahMengajukan         = errors.New("anda sudah terdaftar atau sedang mengajukan sebagai agen")
	ErrBukanPengajuan          = errors.New("akun ini tidak sedang mengajukan keagenan")
	ErrStatusTidakValid        = errors.New("status agen hanya bisa diubah antara aktif dan nonaktif")
	ErrPembayaranFinal         = errors.New("pembayaran pendaftaran sudah terverifikasi atau dibebaskan")
	ErrPembayaranBukanMenunggu = errors.New("hanya pembayaran berstatus menunggu verifikasi yang bisa ditolak")
	ErrSiklusDitutup           = errors.New("pengajuan keagenan ini sudah ditolak, silakan ajukan ulang dari awal")
	ErrTidakAdaPengajuan       = errors.New("belum ada pengajuan agen")
	ErrAksiTidakValid          = errors.New("aksi keputusan tidak dikenal")
	ErrCatatanWajib            = errors.New("catatan wajib diisi untuk aksi ini")
	ErrKeputusanTidakBisa      = errors.New("aksi ini tidak bisa dilakukan pada status pengajuan saat ini")
	ErrNIKAgenTidakValid       = errors.New("NIK harus 16 digit angka")
	ErrNIKAgenTidakBisaDipakai = errors.New("NIK ini tidak dapat dipakai untuk pendaftaran. Periksa kembali atau hubungi admin")
	ErrAlasanStatusWajib       = errors.New("alasan perubahan status wajib diisi (5 sampai 500 karakter)")
	ErrStatusSama              = errors.New("status agen sudah seperti itu")
	ErrBuktiBelumAda           = errors.New("bukti transfer belum diunggah, pilih bebaskan biaya, setujui bayar menyusul, atau minta unggah ulang")
)

// Pembayaran pendaftaran agen satu siklus pengajuan (7.10) beserta riwayat
// upload bukti transfernya (7.11).
type Pembayaran struct {
	ID                int64      `json:"id"`
	NominalTagihan    float64    `json:"nominal_tagihan"`
	Status            string     `json:"status"`
	BuktiTransferURL  *string    `json:"bukti_transfer_url"`
	CatatanPenolakan  *string    `json:"catatan_penolakan"`
	DiverifikasiAt    *time.Time `json:"diverifikasi_at"`
	KeputusanAgen     string     `json:"keputusan_agen"`
	KeputusanAgenAt   *time.Time `json:"keputusan_agen_at"`
	AlasanDitolakAgen *string    `json:"alasan_ditolak_agen"`
	CatatanAdmin      *string    `json:"catatan_admin"`
	CreatedAt         time.Time  `json:"created_at"`
	Uploads           []Upload   `json:"uploads"`
	// RiwayatKeputusan hanya diisi untuk layar admin (antrean pengajuan); portal pemohon tidak menerimanya.
	RiwayatKeputusan []KeputusanLog `json:"riwayat_keputusan,omitempty"`
}

// KeputusanLog adalah satu keputusan admin atas siklus pengajuan (keputusan_agen_log).
type KeputusanLog struct {
	Aksi      string    `json:"aksi"`
	Catatan   *string   `json:"catatan"`
	OlehNama  *string   `json:"oleh_nama"`
	CreatedAt time.Time `json:"created_at"`
}

// StatusLog adalah satu perubahan status agen antara aktif dan nonaktif (agen_status_log).
type StatusLog struct {
	StatusLama string    `json:"status_lama"`
	StatusBaru string    `json:"status_baru"`
	Alasan     string    `json:"alasan"`
	OlehNama   *string   `json:"oleh_nama"`
	DiubahAt   time.Time `json:"diubah_at"`
}

type Upload struct {
	ID               int64     `json:"id"`
	BuktiTransferURL string    `json:"bukti_transfer_url"`
	DiuploadOlehTipe string    `json:"diupload_oleh_tipe"`
	CreatedAt        time.Time `json:"created_at"`
}

// StatusPortal adalah state Syiar milik akun yang login (screen A1–A4).
type StatusPortal struct {
	StatusAgen           string  `json:"status_agen"`
	KodeReferral         *string `json:"kode_referral"`
	FotoURL              *string `json:"foto_url"`
	Domisili             *string `json:"domisili"`
	BrandName            string  `json:"brand_name"`
	BiayaPendaftaranAgen float64 `json:"biaya_pendaftaran_agen"`
	NoWAAdminTravel      *string `json:"no_wa_admin_travel"`
	// AkunJamaah: akun punya booking minimal DP; pengajuannya disetujui otomatis tanpa biaya (keputusan 14 November 2026).
	AkunJamaah bool        `json:"akun_jamaah"`
	Pembayaran *Pembayaran `json:"pembayaran"` // siklus terakhir, nil jika belum pernah
}

// Pengajuan adalah satu baris antrian Persetujuan Agen (screen B1).
type Pengajuan struct {
	AkunID       int64       `json:"akun_id"`
	BrandID      int64       `json:"brand_id"`
	NamaLengkap  string      `json:"nama_lengkap"`
	NoHP         *string     `json:"no_hp"`
	StatusAgen   string      `json:"status_agen"` // 'pengajuan', atau 'aktif'/'nonaktif' bila tinggal pembayaran
	FotoURL      *string     `json:"foto_url"`
	Domisili     *string     `json:"domisili"`
	JenisKelamin *string     `json:"jenis_kelamin"`
	TanggalLahir *string     `json:"tanggal_lahir"` // YYYY-MM-DD
	SetujuSKAt   *time.Time  `json:"menyetujui_syarat_ketentuan_agen_at"`
	DiajukanAt   *time.Time  `json:"diajukan_agen_at"`
	DirekrutOleh *string     `json:"direkrut_oleh_nama"`
	Pembayaran   *Pembayaran `json:"pembayaran"`
	// NIKMasked dan PeringatanNIK untuk admin: NIK pendaftar sudah tercatat atas nama lain, atau jamaahnya terkunci ke agen lain.
	NIKMasked     string   `json:"nik_masked"`
	PeringatanNIK []string `json:"peringatan_nik"`
}

type AjukanRequest struct {
	NIK                   string `json:"nik"`
	Domisili              string `json:"domisili"`
	JenisKelamin          string `json:"jenis_kelamin"` // L atau P
	TanggalLahir          string `json:"tanggal_lahir"` // YYYY-MM-DD, minimal 17 tahun
	SetujuSyaratKetentuan bool   `json:"setuju_syarat_ketentuan"`
	// SetujuMoU wajib bila brand punya draft MoU agen (app, keputusan 14 November 2026); salinan MoU disimpan.
	SetujuMoU bool `json:"setuju_mou"`
	// FotoURL opsional: foto yang diunggah di form app (lewat /api/account/media/upload) menjadi foto profil akun.
	FotoURL string `json:"foto_url"`
}

type BuktiRequest struct {
	BuktiTransferURL string `json:"bukti_transfer_url"`
}

type AlasanRequest struct {
	Alasan string `json:"alasan"`
}

type StatusRequest struct {
	Status string `json:"status"`
	Alasan string `json:"alasan"`
}
