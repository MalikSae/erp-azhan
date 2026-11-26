package akses

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// UserAccess hasil resolusi akses satu pengguna untuk klaim JWT.
type UserAccess struct {
	Jenis       string
	Permissions []string
	PermVersion int
}

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

// GetUserAccess dipanggil saat login/refresh. Super Admin mendapat semua kode
// di tabel permissions; Tim mendapat izin yang dicentang untuknya, disaring ulang
// dengan batas A4/A5 supaya izin di luar batas tidak pernah masuk token.
func (r *Repository) GetUserAccess(ctx context.Context, adminUserID int64) (*UserAccess, error) {
	out := &UserAccess{Permissions: []string{}}
	var super bool
	var brandID sql.NullInt64
	err := r.db.QueryRowContext(ctx,
		`SELECT is_super_admin, perm_version, brand_id FROM admin_users WHERE id = ?`, adminUserID).
		Scan(&super, &out.PermVersion, &brandID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("akses.GetUserAccess: %w", err)
	}

	query := `SELECT p.code FROM user_permissions up JOIN permissions p ON p.id = up.permission_id
		WHERE up.admin_user_id = ? ORDER BY p.code`
	args := []any{adminUserID}
	out.Jenis = Tim
	if super {
		out.Jenis = SuperAdmin
		query, args = `SELECT code FROM permissions ORDER BY code`, nil
	}
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("akses.GetUserAccess izin: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, fmt.Errorf("akses.GetUserAccess scan: %w", err)
		}
		if super || izinTimSah(code, brandID.Valid) {
			out.Permissions = append(out.Permissions, code)
		}
	}
	return out, rows.Err()
}

// izinTimSah: kode boleh dipegang Tim (A4) dan, bila terikat brand, ada di izinBrand (A5).
func izinTimSah(code string, terikatBrand bool) bool {
	if hanyaSuperAdmin[code] {
		return false
	}
	return !terikatBrand || izinBrand[code]
}
