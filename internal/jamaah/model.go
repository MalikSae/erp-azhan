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
	PasporBerlakuSampai *string    `json:"paspor_berlaku_sampai"`
	NoHP                *string    `json:"no_hp"`
	Email               *string    `json:"email"`
	Pekerjaan           *string    `json:"pekerjaan"`
	PendidikanTerakhir  *string    `json:"pendidikan_terakhir"`
	PenjaminKesehatan   *string    `json:"penjamin_kesehatan"`
	NoAsuransiBPJS      *string    `json:"no_asuransi_bpjs"`
	Alamat              *string    `json:"alamat"`
	EmergencyNama       *string    `json:"emergency_nama"`
	EmergencyNIK        *string    `json:"emergency_nik"`
	EmergencyHP         *string    `json:"emergency_hp"`
	EmergencyHubungan   *string    `json:"emergency_hubungan"`
	EmergencyAlamat     *string    `json:"emergency_alamat"`
	CreatedAt           time.Time  `json:"created_at"`
}

// JamaahListItem adalah ringkasan untuk endpoint list.
type JamaahListItem struct {
	ID          int64     `json:"id"`
	BrandID     int64     `json:"brand_id"`
	IDJamaah    string    `json:"id_jamaah"`
	KodeJamaah  string    `json:"kode_jamaah"`
	NamaLengkap string    `json:"nama_lengkap"`
	NIK         *string   `json:"nik"`
	NoHP        *string   `json:"no_hp"`
	Email       *string   `json:"email"`
	CreatedAt   time.Time `json:"created_at"`
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
	PasporBerlakuSampai *string `json:"paspor_berlaku_sampai"`
	NoHP                *string `json:"no_hp"`
	Email               *string `json:"email"`
	Pekerjaan           *string `json:"pekerjaan"`
	PendidikanTerakhir  *string `json:"pendidikan_terakhir"`
	PenjaminKesehatan   *string `json:"penjamin_kesehatan"`
	NoAsuransiBPJS      *string `json:"no_asuransi_bpjs"`
	Alamat              *string `json:"alamat"`
	EmergencyNama       *string `json:"emergency_nama"`
	EmergencyNIK        *string `json:"emergency_nik"`
	EmergencyHP         *string `json:"emergency_hp"`
	EmergencyHubungan   *string `json:"emergency_hubungan"`
	EmergencyAlamat     *string `json:"emergency_alamat"`
}

// UpdateJamaahRequest memiliki struktur yang sama dengan Create.
type UpdateJamaahRequest = CreateJamaahRequest
