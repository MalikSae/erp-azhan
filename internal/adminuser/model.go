package adminuser

import "time"

// Keanggotaan unit organisasi milik seorang pengguna (penugasan; bukan izin).
type Keanggotaan struct {
	OrgUnitID int64  `json:"org_unit_id"`
	UnitNama  string `json:"unit_nama"`
	Jenis     string `json:"jenis"`  // 'divisi' | 'tim'
	Posisi    string `json:"posisi"` // 'pemimpin' | 'anggota'
}

// AdminUser = Pengguna (SPEK-PENGGUNA-ORGANISASI §2). Data aman untuk client
// (tanpa password_hash). Jabatan hanya label tampilan, tidak dipakai logika.
type AdminUser struct {
	ID          uint64        `json:"id"`
	DisplayName string        `json:"display_name"`
	Email       string        `json:"email"`
	NoHP        *string       `json:"no_hp"`
	Jabatan     *string       `json:"jabatan"`
	BrandID     *uint64       `json:"brand_id"`
	BrandName   *string       `json:"brand_name"`
	BrandColor  *string       `json:"brand_color"`
	IsActive    bool          `json:"is_active"`
	Roles       []string      `json:"roles"`
	Keanggotaan []Keanggotaan `json:"keanggotaan"`
	CreatedAt   time.Time     `json:"created_at"`
}

// KeanggotaanInput satu baris penugasan di form pengguna.
type KeanggotaanInput struct {
	OrgUnitID int64  `json:"org_unit_id"`
	Posisi    string `json:"posisi"`
}

// UpsertInput field yang sama untuk create & update (tanpa password).
type UpsertInput struct {
	DisplayName string             `json:"display_name"`
	Email       string             `json:"email"`
	NoHP        *string            `json:"no_hp"`
	Jabatan     *string            `json:"jabatan"`
	BrandID     *uint64            `json:"brand_id"` // nil = akses Holding
	Roles       []string           `json:"roles"`
	Keanggotaan []KeanggotaanInput `json:"keanggotaan"`
}

// CreateAdminUserRequest payload POST /api/admin/users.
type CreateAdminUserRequest struct {
	UpsertInput
	Password string `json:"password"`
}

// UpdateAdminUserRequest payload PUT /api/admin/users/{id}.
type UpdateAdminUserRequest struct {
	UpsertInput
}

// SetStatusRequest payload PUT /api/admin/users/{id}/status.
type SetStatusRequest struct {
	IsActive *bool `json:"is_active"`
}

// ResetPasswordRequest payload untuk PUT /api/admin/users/{id}/password.
type ResetPasswordRequest struct {
	Password string `json:"password"`
}

// ChangeOwnPasswordRequest payload untuk akun Admin Travel.
type ChangeOwnPasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}
