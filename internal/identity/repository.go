package identity

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

var ErrNotFound = errors.New("admin user not found")

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetByEmail(ctx context.Context, email string) (*AdminUser, error) {
	const q = `SELECT id, brand_id, email, display_name, role, is_active, password_hash, created_at FROM admin_users WHERE email = ?`
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

func (r *Repository) GetByID(ctx context.Context, id int64) (*AdminUser, error) {
	const q = `SELECT id, brand_id, email, display_name, role, is_active, password_hash, created_at FROM admin_users WHERE id = ?`
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
