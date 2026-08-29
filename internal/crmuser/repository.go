package crmuser

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/go-sql-driver/mysql"
)

var (
	ErrNotFound       = errors.New("akun CS tidak ditemukan")
	ErrDuplicateEmail = errors.New("email sudah terdaftar")
	ErrInvalidBrand   = errors.New("brand tidak ditemukan")
)

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

func (r *Repository) BrandExists(ctx context.Context, brandID uint64) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM brands WHERE id=?", brandID).Scan(&count)
	return count > 0, err
}

func (r *Repository) List(ctx context.Context, brandID uint64) ([]User, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id,brand_id,email,display_name,role,is_active,created_at
		FROM admin_users WHERE brand_id=? AND role='cs' ORDER BY id`, brandID)
	if err != nil {
		return nil, fmt.Errorf("crmuser.List: %w", err)
	}
	defer rows.Close()
	users := make([]User, 0)
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.BrandID, &user.Email, &user.DisplayName, &user.Role, &user.IsActive, &user.CreatedAt); err != nil {
			return nil, fmt.Errorf("crmuser.List scan: %w", err)
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

func (r *Repository) Get(ctx context.Context, brandID, userID uint64) (*User, error) {
	var user User
	err := r.db.QueryRowContext(ctx, `
		SELECT id,brand_id,email,display_name,role,is_active,created_at
		FROM admin_users WHERE id=? AND brand_id=? AND role='cs'`, userID, brandID).
		Scan(&user.ID, &user.BrandID, &user.Email, &user.DisplayName, &user.Role, &user.IsActive, &user.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("crmuser.Get: %w", err)
	}
	return &user, nil
}

func (r *Repository) Create(ctx context.Context, brandID uint64, email, displayName, passwordHash string) (*User, error) {
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO admin_users (brand_id,email,display_name,role,is_active,password_hash)
		VALUES (?,?,?,'cs',TRUE,?)`, brandID, email, displayName, passwordHash)
	if err != nil {
		if isDuplicate(err) {
			return nil, ErrDuplicateEmail
		}
		return nil, fmt.Errorf("crmuser.Create: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("crmuser.Create last id: %w", err)
	}
	return r.Get(ctx, brandID, uint64(id))
}

func (r *Repository) Update(ctx context.Context, brandID, userID uint64, email, displayName string, active bool) (*User, error) {
	result, err := r.db.ExecContext(ctx, `UPDATE admin_users SET email=?,display_name=?,is_active=?
		WHERE id=? AND brand_id=? AND role='cs'`, email, displayName, active, userID, brandID)
	if err != nil {
		if isDuplicate(err) {
			return nil, ErrDuplicateEmail
		}
		return nil, fmt.Errorf("crmuser.Update: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rows == 0 {
		return nil, ErrNotFound
	}
	return r.Get(ctx, brandID, userID)
}

func (r *Repository) ResetPassword(ctx context.Context, brandID, userID uint64, passwordHash string) error {
	result, err := r.db.ExecContext(ctx, `UPDATE admin_users SET password_hash=? WHERE id=? AND brand_id=? AND role='cs'`, passwordHash, userID, brandID)
	if err != nil {
		return fmt.Errorf("crmuser.ResetPassword: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func isDuplicate(err error) bool {
	var target *mysql.MySQLError
	return errors.As(err, &target) && target.Number == 1062
}
