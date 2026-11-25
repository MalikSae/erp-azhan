package adminuser

import "time"

// AdminUser = Pengguna dashboard (SPEK-AKSES-PENGGUNA-2026-11-25). Data aman untuk
// client (tanpa password_hash). Akses: "super_admin" atau "tim"; Permissions berisi
// izin yang dicentang untuk Tim (kosong untuk Super Admin, yang memegang semuanya).
type AdminUser struct {
	ID          uint64    `json:"id"`
	DisplayName string    `json:"display_name"`
	Email       string    `json:"email"`
	NoHP        *string   `json:"no_hp"`
	BrandID     *uint64   `json:"brand_id"`
	BrandName   *string   `json:"brand_name"`
	BrandColor  *string   `json:"brand_color"`
	IsActive    bool      `json:"is_active"`
	Akses       string    `json:"akses"`
	Permissions []string  `json:"permissions"`
	CreatedAt   time.Time `json:"created_at"`
}

// UpsertInput field yang sama untuk create & update (tanpa password).
type UpsertInput struct {
	DisplayName string   `json:"display_name"`
	Email       string   `json:"email"`
	NoHP        *string  `json:"no_hp"`
	BrandID     *uint64  `json:"brand_id"` // nil = akses Holding
	Akses       string   `json:"akses"`
	Permissions []string `json:"permissions"`
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

// ChangeOwnPasswordRequest payload PUT /api/admin/account/password.
type ChangeOwnPasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}
