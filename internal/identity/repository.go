package identity

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var ErrNotFound = errors.New("admin user not found")

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetByEmail(ctx context.Context, email string) (*AdminUser, error) {
	const q = `SELECT id, brand_id, email, COALESCE(display_name, ''), role, is_active, password_hash, created_at FROM admin_users WHERE email = ?`
	var user AdminUser
	err := r.db.QueryRowContext(ctx, q, email).Scan(&user.ID, &user.BrandID, &user.Email, &user.DisplayName, &user.Role, &user.IsActive, &user.PasswordHash, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("identity.GetByEmail: %w", err)
	}
	return &user, nil
}

// SaveRefreshToken mencatat refresh token yang baru diterbitkan.
func (r *Repository) SaveRefreshToken(ctx context.Context, jti string, adminUserID int64, expiresAt time.Time) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO admin_refresh_tokens (jti, admin_user_id, expires_at) VALUES (?, ?, ?)`,
		jti, adminUserID, expiresAt.UTC())
	if err != nil {
		return fmt.Errorf("identity.SaveRefreshToken: %w", err)
	}
	return nil
}

// IsRefreshTokenActive true jika jti tercatat untuk user tsb, belum dicabut,
// dan belum kedaluwarsa.
func (r *Repository) IsRefreshTokenActive(ctx context.Context, jti string, adminUserID int64) (bool, error) {
	var n int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM admin_refresh_tokens
		 WHERE jti = ? AND admin_user_id = ? AND revoked_at IS NULL AND expires_at > UTC_TIMESTAMP()`,
		jti, adminUserID).Scan(&n)
	if err != nil {
		return false, fmt.Errorf("identity.IsRefreshTokenActive: %w", err)
	}
	return n > 0, nil
}

// RevokeRefreshToken mencabut satu refresh token (logout).
func (r *Repository) RevokeRefreshToken(ctx context.Context, jti string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE admin_refresh_tokens SET revoked_at = UTC_TIMESTAMP() WHERE jti = ? AND revoked_at IS NULL`, jti)
	if err != nil {
		return fmt.Errorf("identity.RevokeRefreshToken: %w", err)
	}
	return nil
}

// RevokeAllRefreshTokens mencabut semua refresh token aktif milik user
// (reset/ganti password, akun dinonaktifkan). Dipanggil modul lain lewat
// interface, bukan query langsung ke tabel ini.
func (r *Repository) RevokeAllRefreshTokens(ctx context.Context, adminUserID int64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE admin_refresh_tokens SET revoked_at = UTC_TIMESTAMP() WHERE admin_user_id = ? AND revoked_at IS NULL`, adminUserID)
	if err != nil {
		return fmt.Errorf("identity.RevokeAllRefreshTokens: %w", err)
	}
	return nil
}

func (r *Repository) GetByID(ctx context.Context, id int64) (*AdminUser, error) {
	const q = `SELECT id, brand_id, email, COALESCE(display_name, ''), role, is_active, password_hash, created_at FROM admin_users WHERE id = ?`
	var user AdminUser
	err := r.db.QueryRowContext(ctx, q, id).Scan(&user.ID, &user.BrandID, &user.Email, &user.DisplayName, &user.Role, &user.IsActive, &user.PasswordHash, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("identity.GetByID: %w", err)
	}
	return &user, nil
}
