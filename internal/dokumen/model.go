package dokumen

import "time"

// DokumenJamaah adalah response lengkap.
type DokumenJamaah struct {
	ID        int64     `json:"id"`
	JamaahID  int64     `json:"jamaah_id"`
	Jenis     string    `json:"jenis"`
	FileURL   *string   `json:"file_url"`
	Status    string    `json:"status"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateDokumenRequest adalah payload untuk POST upsert.
type CreateDokumenRequest struct {
	Jenis   string `json:"jenis"`
	FileURL string `json:"file_url"`
}

// UpdateDokumenStatusRequest adalah payload untuk PUT status.
type UpdateDokumenStatusRequest struct {
	Status string `json:"status"`
}
