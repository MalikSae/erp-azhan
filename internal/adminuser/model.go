package adminuser

import "time"

// AdminUser merepresentasikan data admin_users yang aman untuk dikembalikan ke client (tanpa password_hash).
type AdminUser struct {
	ID        uint64    `json:"id"`
	Email     string    `json:"email"`
	BrandID    *uint64   `json:"brand_id"`
	BrandName  *string   `json:"brand_name"`
	BrandColor *string   `json:"brand_color"`
	CreatedAt  time.Time `json:"created_at"`
}

// CreateAdminUserRequest payload untuk POST /api/admin/users.
type CreateAdminUserRequest struct {
	Email    string  `json:"email"`
	Password string  `json:"password"`
	BrandID  *uint64 `json:"brand_id"`
}

// UpdateAdminUserRequest payload untuk PUT /api/admin/users/{id}.
type UpdateAdminUserRequest struct {
	Email   string  `json:"email"`
	BrandID *uint64 `json:"brand_id"`
}

// ResetPasswordRequest payload untuk PUT /api/admin/users/{id}/password.
type ResetPasswordRequest struct {
	Password string `json:"password"`
}
