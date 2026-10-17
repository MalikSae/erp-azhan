package jamaah

import "time"

// ─── Response structs ─────────────────────────────────────────────────────────

// Jamaah adalah response lengkap admin.
type Jamaah struct {
	ID                   int64     `json:"id"`
	BrandID              int64     `json:"brand_id"`
	IDJamaah             string    `json:"id_jamaah"`
	KodeJamaah           string    `json:"kode_jamaah"`
	NamaLengkap          string    `json:"nama_lengkap"`
	JenisKelamin         *string   `json:"jenis_kelamin"`
	NamaAyahKandung      *string   `json:"nama_ayah_kandung"`
	NIK                  *string   `json:"nik"`
	TempatLahir          *string   `json:"tempat_lahir"`
	TanggalLahir         *string   `json:"tanggal_lahir"`
	NoPaspor             *string   `json:"no_paspor"`
	TempatPasporKeluar   *string   `json:"tempat_paspor_keluar"`
	TanggalPasporKeluar  *string   `json:"tanggal_paspor_keluar"`
	PasporBerlakuSampai  *string   `json:"paspor_berlaku_sampai"`
	NoHP                 *string   `json:"no_hp"`
	Email                *string   `json:"email"`
	Pekerjaan            *string   `json:"pekerjaan"`
	PendidikanTerakhir   *string   `json:"pendidikan_terakhir"`
	PenjaminKesehatan    *string   `json:"penjamin_kesehatan"`
	NoAsuransiBPJS       *string   `json:"no_asuransi_bpjs"`
	Alamat               *string   `json:"alamat"`
	Kota                 *string   `json:"kota"`
	Catatan              *string   `json:"catatan"`
	Status               string    `json:"status"`
	EmergencyNama        *string   `json:"emergency_nama"`
	EmergencyNIK         *string   `json:"emergency_nik"`
	EmergencyHP          *string   `json:"emergency_hp"`
	EmergencyHubungan    *string   `json:"emergency_hubungan"`
	EmergencyAlamat      *string   `json:"emergency_alamat"`
	CreatedAt            time.Time `json:"created_at"`
	PortalAktif          bool      `json:"portal_aktif"`
	KaitanStatus         string    `json:"kaitan_status"`
	DirekrutOlehJamaahID *int64    `json:"direkrut_oleh_jamaah_id"`
	DirekrutOlehNama     *string   `json:"direkrut_oleh_nama"`
	StatusAgen           string    `json:"status_agen"`
}

// JamaahListItem adalah ringkasan untuk endpoint list.
type JamaahListItem struct {
	ID                   int64     `json:"id"`
	BrandID              int64     `json:"brand_id"`
	IDJamaah             string    `json:"id_jamaah"`
	KodeJamaah           string    `json:"kode_jamaah"`
	NamaLengkap          string    `json:"nama_lengkap"`
	JenisKelamin         *string   `json:"jenis_kelamin"`
	NIK                  *string   `json:"nik"`
	TanggalLahir         *string   `json:"tanggal_lahir"`
	NoHP                 *string   `json:"no_hp"`
	Email                *string   `json:"email"`
	Status               string    `json:"status"`
	PctData              int       `json:"pct_data"`
	PctDokumen           int       `json:"pct_dokumen"`
	CreatedAt            time.Time `json:"created_at"`
	KaitanStatus         string    `json:"kaitan_status"`
	DirekrutOlehJamaahID *int64    `json:"direkrut_oleh_jamaah_id"`
	DirekrutOlehNama     *string   `json:"direkrut_oleh_nama"`
	StatusAgen           string    `json:"status_agen"`
	// Jumlah booking yang melibatkan jamaah ini (sebagai PIC atau peserta), semua status.
	TotalBooking int `json:"total_booking"`
}

// ─── Request structs ──────────────────────────────────────────────────────────

// CreateJamaahRequest adalah payload untuk POST /api/admin/jamaah.
type CreateJamaahRequest struct {
	BrandID             *int64  `json:"brand_id"`
	NamaLengkap         string  `json:"nama_lengkap"`
	JenisKelamin        *string `json:"jenis_kelamin"`
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
	// KaitanAgen wajib saat membuat jamaah baru (Jalur 3, agen-azhan.md 3.5).
	// Diabaikan saat update: kaitan hanya bisa diganti Admin Master (C4).
	KaitanAgen *KaitanAgenInput `json:"kaitan_agen"`
}

// KaitanAgenInput: mode "agen" (wajib agen_jamaah_id agen aktif di brand yang
// sama) atau "tanpa_agen".
type KaitanAgenInput struct {
	Mode         string `json:"mode"`
	AgenJamaahID *int64 `json:"agen_jamaah_id"`
}

// UpdateJamaahRequest memiliki struktur yang sama dengan Create.
type UpdateJamaahRequest = CreateJamaahRequest

// UpdateCatatanRequest adalah payload untuk update khusus field catatan.
type UpdateCatatanRequest struct {
	Catatan *string `json:"catatan"`
}
