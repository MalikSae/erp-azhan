package adminuser

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

var (
	ErrNotFound         = errors.New("admin user tidak ditemukan")
	ErrDuplicateEmail   = errors.New("email sudah terdaftar")
	ErrInvalidBrand     = errors.New("brand tidak valid atau tidak ditemukan")
	ErrLastSuperAdmin   = errors.New("tidak bisa mengubah atau menghapus Super Admin terakhir, sistem harus punya minimal 1 Super Admin")
	ErrCannotDeleteSelf = errors.New("tidak bisa menghapus akun sendiri")
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) List(ctx context.Context) ([]AdminUser, error) {
	query := `
		SELECT u.id, u.email, u.brand_id, b.name AS brand_name, b.primary_color AS brand_color, u.created_at
		FROM admin_users u
		LEFT JOIN brands b ON u.brand_id = b.id
		ORDER BY u.id ASC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("adminuser.List: %w", err)
	}
	defer rows.Close()

	users := make([]AdminUser, 0)
	for rows.Next() {
		var u AdminUser
		var brandID sql.NullInt64
		var brandName sql.NullString
		var brandColor sql.NullString
		if err := rows.Scan(&u.ID, &u.Email, &brandID, &brandName, &brandColor, &u.CreatedAt); err != nil {
			return nil, fmt.Errorf("adminuser.List scan: %w", err)
		}
		if brandID.Valid {
			bID := uint64(brandID.Int64)
			u.BrandID = &bID
		}
		if brandName.Valid {
			u.BrandName = &brandName.String
		}
		if brandColor.Valid {
			u.BrandColor = &brandColor.String
		}
		users = append(users, u)
	}
	return users, nil
}

func (r *Repository) GetByID(ctx context.Context, id uint64) (*AdminUser, error) {
	query := `
		SELECT u.id, u.email, u.brand_id, b.name AS brand_name, b.primary_color AS brand_color, u.created_at
		FROM admin_users u
		LEFT JOIN brands b ON u.brand_id = b.id
		WHERE u.id = ?
	`
	var u AdminUser
	var brandID sql.NullInt64
	var brandName sql.NullString
	var brandColor sql.NullString
	err := r.db.QueryRowContext(ctx, query, id).Scan(&u.ID, &u.Email, &brandID, &brandName, &brandColor, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("adminuser.GetByID: %w", err)
	}
	if brandID.Valid {
		bID := uint64(brandID.Int64)
		u.BrandID = &bID
	}
	if brandName.Valid {
		u.BrandName = &brandName.String
	}
	if brandColor.Valid {
		u.BrandColor = &brandColor.String
	}
	return &u, nil
}

func (r *Repository) Create(ctx context.Context, email, passwordHash string, brandID *uint64) (*AdminUser, error) {
	// Cek duplikasi email
	var count int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM admin_users WHERE email = ?", email).Scan(&count)
	if err != nil {
		return nil, fmt.Errorf("adminuser.Create check email: %w", err)
	}
	if count > 0 {
		return nil, ErrDuplicateEmail
	}

	// Validasi brand_id jika disediakan
	if brandID != nil {
		var brandCount int
		err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM brands WHERE id = ?", *brandID).Scan(&brandCount)
		if err != nil {
			return nil, fmt.Errorf("adminuser.Create check brand: %w", err)
		}
		if brandCount == 0 {
			return nil, ErrInvalidBrand
		}
	}

	query := "INSERT INTO admin_users (email, password_hash, brand_id) VALUES (?, ?, ?)"
	res, err := r.db.ExecContext(ctx, query, email, passwordHash, brandID)
	if err != nil {
		return nil, fmt.Errorf("adminuser.Create: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("adminuser.Create lastInsertId: %w", err)
	}

	return r.GetByID(ctx, uint64(id))
}

func (r *Repository) Update(ctx context.Context, id uint64, email string, newBrandID *uint64) (*AdminUser, error) {
	// Cek apakah user ada dan dapatkan brand_id saat ini
	var currentBrandID sql.NullInt64
	err := r.db.QueryRowContext(ctx, "SELECT brand_id FROM admin_users WHERE id = ?", id).Scan(&currentBrandID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("adminuser.Update check user: %w", err)
	}

	// Cek duplikasi email pada user lain
	var count int
	err = r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM admin_users WHERE email = ? AND id != ?", email, id).Scan(&count)
	if err != nil {
		return nil, fmt.Errorf("adminuser.Update check email: %w", err)
	}
	if count > 0 {
		return nil, ErrDuplicateEmail
	}

	// Validasi brand_id baru jika tidak null
	if newBrandID != nil {
		var brandCount int
		err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM brands WHERE id = ?", *newBrandID).Scan(&brandCount)
		if err != nil {
			return nil, fmt.Errorf("adminuser.Update check brand: %w", err)
		}
		if brandCount == 0 {
			return nil, ErrInvalidBrand
		}
	}

	// SAFETY CHECK: jika user saat ini adalah Super Admin (brand_id NULL) dan akan diubah menjadi non-NULL (Admin Travel)
	if !currentBrandID.Valid && newBrandID != nil {
		var superAdminCount int
		err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM admin_users WHERE brand_id IS NULL").Scan(&superAdminCount)
		if err != nil {
			return nil, fmt.Errorf("adminuser.Update check super admin count: %w", err)
		}
		if superAdminCount <= 1 {
			return nil, ErrLastSuperAdmin
		}
	}

	query := "UPDATE admin_users SET email = ?, brand_id = ? WHERE id = ?"
	_, err = r.db.ExecContext(ctx, query, email, newBrandID, id)
	if err != nil {
		return nil, fmt.Errorf("adminuser.Update: %w", err)
	}

	return r.GetByID(ctx, id)
}

func (r *Repository) ResetPassword(ctx context.Context, id uint64, passwordHash string) error {
	// Cek user ada
	var exists int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM admin_users WHERE id = ?", id).Scan(&exists)
	if err != nil {
		return fmt.Errorf("adminuser.ResetPassword check user: %w", err)
	}
	if exists == 0 {
		return ErrNotFound
	}

	query := "UPDATE admin_users SET password_hash = ? WHERE id = ?"
	_, err = r.db.ExecContext(ctx, query, passwordHash, id)
	if err != nil {
		return fmt.Errorf("adminuser.ResetPassword: %w", err)
	}
	return nil
}

func (r *Repository) Delete(ctx context.Context, id uint64, currentLoggedInID uint64) error {
	// SAFETY CHECK 1: Tidak bisa menghapus diri sendiri
	if id == currentLoggedInID {
		return ErrCannotDeleteSelf
	}

	// Cek user ada dan ambil brand_id
	var brandID sql.NullInt64
	err := r.db.QueryRowContext(ctx, "SELECT brand_id FROM admin_users WHERE id = ?", id).Scan(&brandID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("adminuser.Delete check user: %w", err)
	}

	// SAFETY CHECK 2: Jika Super Admin (brand_id NULL) dan dia satu-satunya
	if !brandID.Valid {
		var superAdminCount int
		err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM admin_users WHERE brand_id IS NULL").Scan(&superAdminCount)
		if err != nil {
			return fmt.Errorf("adminuser.Delete check super admin count: %w", err)
		}
		if superAdminCount <= 1 {
			return ErrLastSuperAdmin
		}
	}

	query := "DELETE FROM admin_users WHERE id = ?"
	_, err = r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("adminuser.Delete: %w", err)
	}

	return nil
}
