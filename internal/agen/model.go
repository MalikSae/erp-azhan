// Package agen mengelola alur menjadi Agen Umroh ("Syiar"): pengajuan,
// kelengkapan data, pembayaran pendaftaran, dan keputusan keagenan
// (agen-azhan.md Bagian 3.6, 7.1, 7.9–7.11).
package agen

import (
	"errors"
	"time"
)

var (
	ErrNotFound                = errors.New("data tidak ditemukan")
	ErrSudahMengajukan         = errors.New("anda sudah terdaftar atau sedang mengajukan sebagai agen")
	ErrBukanPengajuan          = errors.New("jamaah ini tidak sedang mengajukan keagenan")
	ErrStatusTidakValid        = errors.New("status agen hanya bisa diubah antara aktif dan nonaktif")
	ErrPembayaranFinal         = errors.New("pembayaran pendaftaran sudah terverifikasi")
	ErrPembayaranBukanMenunggu = errors.New("hanya pembayaran berstatus menunggu verifikasi yang bisa ditolak")
	ErrSiklusDitutup           = errors.New("pengajuan keagenan ini sudah ditolak, silakan ajukan ulang dari awal")
	ErrTidakAdaPengajuan       = errors.New("belum ada pengajuan agen")
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
	CreatedAt         time.Time  `json:"created_at"`
	Uploads           []Upload   `json:"uploads"`
}

type Upload struct {
	ID               int64     `json:"id"`
	BuktiTransferURL string    `json:"bukti_transfer_url"`
	DiuploadOlehTipe string    `json:"diupload_oleh_tipe"`
	CreatedAt        time.Time `json:"created_at"`
}

// StatusPortal adalah state Syiar milik jamaah yang login (screen A1–A4).
type StatusPortal struct {
	StatusAgen           string      `json:"status_agen"`
	KodeReferral         *string     `json:"kode_referral"`
	FotoAgenURL          *string     `json:"foto_agen_url"`
	Domisili             *string     `json:"domisili"`
	BrandName            string      `json:"brand_name"`
	BiayaPendaftaranAgen float64     `json:"biaya_pendaftaran_agen"`
	NoWAAdminTravel      *string     `json:"no_wa_admin_travel"`
	Pembayaran           *Pembayaran `json:"pembayaran"` // siklus terakhir, nil jika belum pernah
}

// Pengajuan adalah satu baris antrian Persetujuan Agen (screen B1).
type Pengajuan struct {
	JamaahID     int64       `json:"jamaah_id"`
	IDJamaah     string      `json:"id_jamaah"`
	BrandID      int64       `json:"brand_id"`
	NamaLengkap  string      `json:"nama_lengkap"`
	NoHP         *string     `json:"no_hp"`
	StatusAgen   string      `json:"status_agen"` // 'pengajuan', atau 'aktif'/'nonaktif' bila tinggal pembayaran
	FotoAgenURL  *string     `json:"foto_agen_url"`
	Domisili     *string     `json:"domisili"`
	SetujuSKAt   *time.Time  `json:"menyetujui_syarat_ketentuan_agen_at"`
	DiajukanAt   *time.Time  `json:"diajukan_agen_at"`
	DirekrutOleh *string     `json:"direkrut_oleh_nama"`
	Pembayaran   *Pembayaran `json:"pembayaran"`
}

type AjukanRequest struct {
	FotoAgenURL           string `json:"foto_agen_url"`
	Domisili              string `json:"domisili"`
	SetujuSyaratKetentuan bool   `json:"setuju_syarat_ketentuan"`
}

type BuktiRequest struct {
	BuktiTransferURL string `json:"bukti_transfer_url"`
}

type AlasanRequest struct {
	Alasan string `json:"alasan"`
}

type StatusRequest struct {
	Status string `json:"status"`
}
