package jamaah

import "time"

// ─── Response structs ─────────────────────────────────────────────────────────

// Jamaah adalah response lengkap admin.
type Jamaah struct {
	ID                  int64      `json:"id"`
	BrandID             int64      `json:"brand_id"`
	IDJamaah            string     `json:"id_jamaah"`
	KodeJamaah          string     `json:"kode_jamaah"`
	NamaLengkap         string     `json:"nama_lengkap"`
	NamaAyahKandung     *string    `json:"nama_ayah_kandung"`
	NIK                 *string    `json:"nik"`
	TempatLahir         *string    `json:"tempat_lahir"`
	TanggalLahir        *string    `json:"tanggal_lahir"`
	NoPaspor            *string    `json:"no_paspor"`
	TempatPasporKeluar  *string    `json:"tempat_paspor_keluar"`
	TanggalPasporKeluar *string    `json:"tanggal_paspor_keluar"`
	PasporBerlakuSampai *string    `json:"paspor_berlaku_sampai"`
	NoHP                *string    `json:"no_hp"`
	Email               *string    `json:"email"`
	Pekerjaan           *string    `json:"pekerjaan"`
	PendidikanTerakhir  *string    `json:"pendidikan_terakhir"`
	PenjaminKesehatan   *string    `json:"penjamin_kesehatan"`
	NoAsuransiBPJS      *string    `json:"no_asuransi_bpjs"`
	Alamat              *string    `json:"alamat"`
	Kota                *string    `json:"kota"`
	Catatan             *string    `json:"catatan"`
	Status              string     `json:"status"`
	EmergencyNama       *string    `json:"emergency_nama"`
	EmergencyNIK        *string    `json:"emergency_nik"`
	EmergencyHP         *string    `json:"emergency_hp"`
	EmergencyHubungan   *string    `json:"emergency_hubungan"`
	EmergencyAlamat     *string    `json:"emergency_alamat"`
	CreatedAt           time.Time  `json:"created_at"`
}

// JamaahListItem adalah ringkasan untuk endpoint list.
type JamaahListItem struct {
	ID           int64     `json:"id"`
	BrandID      int64     `json:"brand_id"`
	IDJamaah     string    `json:"id_jamaah"`
	KodeJamaah   string    `json:"kode_jamaah"`
	NamaLengkap  string    `json:"nama_lengkap"`
	NIK          *string   `json:"nik"`
	TanggalLahir *string   `json:"tanggal_lahir"`
	NoHP         *string   `json:"no_hp"`
	Email        *string   `json:"email"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}

// ─── Relasi Kekerabatan ────────────────────────────────────────────────────────

// ValidHubunganList adalah daftar ENUM hubungan yang valid di database.
var ValidHubunganList = []string{
	"Pasangan",
	"Orang Tua",
	"Anak",
	"Saudara Kandung",
	"Mahram",
	"Kerabat Lain",
}

// HubunganInverseMap mendefinisikan label kebalikan untuk setiap hubungan.
var HubunganInverseMap = map[string]string{
	"Pasangan":        "Pasangan",
	"Saudara Kandung": "Saudara Kandung",
	"Mahram":          "Mahram",
	"Kerabat Lain":    "Kerabat Lain",
	"Anak":            "Orang Tua",
	"Orang Tua":       "Anak",
}

// JamaahRelasiItem adalah model response untuk relasi kekerabatan jamaah.
type JamaahRelasiItem struct {
	ID             int64     `json:"id"`
	JamaahID       int64     `json:"jamaah_id"`
	RelasiJamaahID int64     `json:"relasi_jamaah_id"`
	NamaRelasi     string    `json:"nama_relasi"`
	IDJamaahRelasi string    `json:"id_jamaah_relasi"`
	NIKRelasi      *string   `json:"nik_relasi"`
	Hubungan       string    `json:"hubungan"`      // Label yang sudah dimapping (asli atau kebalikan)
	HubunganAsli   string    `json:"hubungan_asli"` // Nilai asli yang tersimpan di baris database
	IsAsal         bool      `json:"is_asal"`       // true jika ditambahkan dari sisi jamaah ini
	Keterangan     *string   `json:"keterangan"`
	CreatedAt      time.Time `json:"created_at"`
}

// ─── Request structs ──────────────────────────────────────────────────────────

// CreateJamaahRequest adalah payload untuk POST /api/admin/jamaah.
type CreateJamaahRequest struct {
	BrandID             *int64  `json:"brand_id"`
	NamaLengkap         string  `json:"nama_lengkap"`
	NamaAyahKandung     *string `json:"nama_ayah_kandung"`
	NIK                 *string `json:"nik"`
	TempatLahir         *string `json:"tempat_lahir"`
	TanggalLahir        *string `json:"tanggal_lahir"`
	NoPaspor            *string `json:"no_paspor"`
	TempatPasporKeluar  *string `json:"tempat_paspor_keluar"`
	TanggalPasporKeluar *string `json:"tanggal_paspor_keluar"`
	PasporBerlakuSampai *string `json:"paspor_berlaku_sampai"`
	NoHP                *string `json:"no_hp"`
	Email               *string `json:"email"`
	Pekerjaan           *string `json:"pekerjaan"`
	PendidikanTerakhir  *string `json:"pendidikan_terakhir"`
	PenjaminKesehatan   *string `json:"penjamin_kesehatan"`
	NoAsuransiBPJS      *string `json:"no_asuransi_bpjs"`
	Alamat              *string `json:"alamat"`
	Kota                *string `json:"kota"`
	Catatan             *string `json:"catatan"`
	Status              *string `json:"status"`
	EmergencyNama       *string `json:"emergency_nama"`
	EmergencyNIK        *string `json:"emergency_nik"`
	EmergencyHP         *string `json:"emergency_hp"`
	EmergencyHubungan   *string `json:"emergency_hubungan"`
	EmergencyAlamat     *string `json:"emergency_alamat"`
}

// UpdateJamaahRequest memiliki struktur yang sama dengan Create.
type UpdateJamaahRequest = CreateJamaahRequest

// UpdateCatatanRequest adalah payload untuk update khusus field catatan.
type UpdateCatatanRequest struct {
	Catatan *string `json:"catatan"`
}

// CreateRelasiRequest adalah payload untuk POST /api/admin/jamaah/{id}/relasi.
type CreateRelasiRequest struct {
	RelasiJamaahID int64   `json:"relasi_jamaah_id"`
	Hubungan       string  `json:"hubungan"`
	Keterangan     *string `json:"keterangan"`
}

// UpdateRelasiRequest adalah payload untuk PUT /api/admin/jamaah/{id}/relasi/{relasi_id}.
type UpdateRelasiRequest struct {
	Hubungan   string  `json:"hubungan"`
	Keterangan *string `json:"keterangan"`
}
