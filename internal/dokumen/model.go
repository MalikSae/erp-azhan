package dokumen

import "time"

// DokumenJamaah adalah response lengkap.
type DokumenJamaah struct {
	Version         int64      `json:"version"`
	RejectionReason *string    `json:"rejection_reason"`
	ID              int64      `json:"id"`
	JamaahID        int64      `json:"jamaah_id"`
	Jenis           string     `json:"jenis"`
	FileURL         *string    `json:"file_url"`
	Status          string     `json:"status"`
	UpdatedAt       time.Time  `json:"updated_at"`
	VerifiedBy      *int64     `json:"verified_by"`
	VerifiedByNama  *string    `json:"verified_by_nama"`
	VerifiedAt      *time.Time `json:"verified_at"`
}

// VersiDokumen adalah satu versi lama dokumen (sebelum diganti), untuk riwayat.
type VersiDokumen struct {
	Version         int64      `json:"version"`
	Status          string     `json:"status"`
	RejectionReason *string    `json:"rejection_reason"`
	FileURL         *string    `json:"file_url"`
	VerifiedByNama  *string    `json:"verified_by_nama"`
	VerifiedAt      *time.Time `json:"verified_at"`
	RecordedAt      time.Time  `json:"recorded_at"`
}

// CreateDokumenRequest adalah payload untuk POST upsert.
type CreateDokumenRequest struct {
	Jenis   string `json:"jenis"`
	FileURL string `json:"file_url"`
}

// UpdateDokumenStatusRequest adalah payload untuk PUT status.
type UpdateDokumenStatusRequest struct {
	Status          string `json:"status"`
	Version         int64  `json:"version"`
	RejectionReason string `json:"rejection_reason"`
}
