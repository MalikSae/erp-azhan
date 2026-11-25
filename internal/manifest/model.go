// Package manifest menyusun manifest keberangkatan satu paket (docs/spek/SPEK-MANIFEST-PAKET-2026-11-25.md).
// Fase 1: PNR dan detail penerbangan per paket, pemeriksaan data pax, unduh .xlsx (Blok Seat, Manifest, Visa,
// Asuransi), dan jejak unduhan.
package manifest

import "errors"

var ErrTidakAda = errors.New("Paket tidak ditemukan.")

// ErrValidasi membawa pesan galat untuk pengguna (400).
type ErrValidasi struct{ Pesan string }

func (e *ErrValidasi) Error() string { return e.Pesan }

const (
	maksPNR    = 100
	maksDetail = 2000
)

// Paket adalah header manifest.
type Paket struct {
	ID                int64  `json:"id"`
	BrandID           int64  `json:"brand_id"`
	Nama              string `json:"nama"`
	BerangkatTanggal  string `json:"berangkat_tanggal"`
	PulangTanggal     string `json:"pulang_tanggal"`
	Maskapai          string `json:"maskapai"`
	KodeMaskapai      string `json:"kode_maskapai"`
	PNR               string `json:"pnr"`
	DetailPenerbangan string `json:"detail_penerbangan"`
}

// Pax adalah satu baris manifest.
type Pax struct {
	PaxID         int64  `json:"pax_id"`
	BookingID     int64  `json:"booking_id"`
	KodeBooking   string `json:"kode_booking"`
	StatusBooking string `json:"status_booking"`
	JamaahID      int64  `json:"jamaah_id"`
	Nama          string `json:"nama"`
	NamaPaspor    string `json:"nama_paspor"`
	JenisKelamin  string `json:"jenis_kelamin"`
	NIK           string `json:"nik"`
	TempatLahir   string `json:"tempat_lahir"`
	TanggalLahir  string `json:"tanggal_lahir"`
	NoPaspor      string `json:"no_paspor"`
	PasporTerbit  string `json:"paspor_terbit"`
	PasporHabis   string `json:"paspor_habis"`
	KantorPaspor  string `json:"kantor_paspor"`
	TipePax       string `json:"tipe_pax"`
	TipeKamar     string `json:"tipe_kamar"`
	Usia          *int   `json:"usia"`
	JenisTiket    string `json:"jenis_tiket"`
	// Masalah: daftar kekurangan data pax ini; kosong = lengkap.
	Masalah []string `json:"masalah"`
}

// Unduhan adalah jejak unduhan terakhir.
type Unduhan struct {
	Nama  string `json:"nama"`
	Waktu string `json:"waktu"`
}

// Manifest adalah isi tab Manifest.
type Manifest struct {
	Paket         Paket    `json:"paket"`
	Pax           []Pax    `json:"pax"`
	JumlahMasalah int      `json:"jumlah_masalah"`
	LamaHari      int      `json:"lama_hari"`
	TerakhirUnduh *Unduhan `json:"terakhir_unduh"`
}

// InputInfo adalah isian PUT manifest.
type InputInfo struct {
	PNR               string `json:"pnr"`
	DetailPenerbangan string `json:"detail_penerbangan"`
}
