package identity

import "time"

// AdminUser merepresentasikan record di tabel admin_users.
type AdminUser struct {
	ID           int64     `json:"id"`
	BrandID      *int64    `json:"brand_id"`
	Email        string    `json:"email"`
	DisplayName  string    `json:"display_name"`
	Role         string    `json:"role"`
	IsActive     bool      `json:"is_active"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

// LoginRequest payload dari client
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RefreshRequest payload dari client
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// TokenResponse kembalian sukses login/refresh
type TokenResponse struct {
	UserID       int64  `json:"user_id"`
	DisplayName  string `json:"display_name"`
	Role         string `json:"role"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"` // tidak selalu ada (misal di refresh token response kalau mau disembunyikan, tapi req minta diabaikan)
	ExpiresIn    int    `json:"expires_in"`              // dalam detik
	TokenType    string `json:"token_type"`              // selalu "Bearer"
}
