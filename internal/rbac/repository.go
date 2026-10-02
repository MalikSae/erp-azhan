package rbac

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrUserNotFound = errors.New("user tidak ditemukan")
	ErrUnknownRole   = errors.New("role tidak dikenal")
	ErrScopeMismatch = errors.New("kombinasi role dan akses brand tidak sah")
)

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

// GetUserAccess meresolusi role, permission, dan perm_version milik satu user.
// Dipanggil saat login/refresh untuk mengisi klaim JWT. Jika tabel RBAC belum
// termigrasi, pemanggil yang memutuskan fallback-nya (lihat identity.Handler).
func (r *Repository) GetUserAccess(ctx context.Context, adminUserID int64) (*UserAccess, error) {
	access := &UserAccess{Roles: []string{}, Permissions: []string{}}

	err := r.db.QueryRowContext(ctx,
		`SELECT perm_version FROM admin_users WHERE id = ?`, adminUserID).
		Scan(&access.PermVersion)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("rbac.GetUserAccess perm_version: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT r.slug FROM user_roles ur
		JOIN roles r ON r.id = ur.role_id
		WHERE ur.admin_user_id = ? ORDER BY r.slug`, adminUserID)
	if err != nil {
		return nil, fmt.Errorf("rbac.GetUserAccess roles: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var slug string
		if err := rows.Scan(&slug); err != nil {
			return nil, fmt.Errorf("rbac.GetUserAccess roles scan: %w", err)
		}
		access.Roles = append(access.Roles, slug)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	permRows, err := r.db.QueryContext(ctx, `
		SELECT DISTINCT p.code FROM user_roles ur
		JOIN role_permissions rp ON rp.role_id = ur.role_id
		JOIN permissions p ON p.id = rp.permission_id
		WHERE ur.admin_user_id = ? ORDER BY p.code`, adminUserID)
	if err != nil {
		return nil, fmt.Errorf("rbac.GetUserAccess perms: %w", err)
	}
	defer permRows.Close()
	for permRows.Next() {
		var code string
		if err := permRows.Scan(&code); err != nil {
			return nil, fmt.Errorf("rbac.GetUserAccess perms scan: %w", err)
		}
		access.Permissions = append(access.Permissions, code)
	}
	return access, permRows.Err()
}

func (r *Repository) ListPermissions(ctx context.Context) ([]Permission, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, code, description FROM permissions ORDER BY code`)
	if err != nil {
		return nil, fmt.Errorf("rbac.ListPermissions: %w", err)
	}
	defer rows.Close()
	perms := make([]Permission, 0)
	for rows.Next() {
		var p Permission
		if err := rows.Scan(&p.ID, &p.Code, &p.Description); err != nil {
			return nil, fmt.Errorf("rbac.ListPermissions scan: %w", err)
		}
		perms = append(perms, p)
	}
	return perms, rows.Err()
}

func (r *Repository) ListRoles(ctx context.Context) ([]Role, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, slug, name, is_system, scope, created_at FROM roles ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("rbac.ListRoles: %w", err)
	}
	defer rows.Close()
	roles := make([]Role, 0)
	index := make(map[int64]int)
	for rows.Next() {
		var role Role
		if err := rows.Scan(&role.ID, &role.Slug, &role.Name, &role.IsSystem, &role.Scope, &role.CreatedAt); err != nil {
			return nil, fmt.Errorf("rbac.ListRoles scan: %w", err)
		}
		role.Permissions = []string{}
		index[role.ID] = len(roles)
		roles = append(roles, role)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	permRows, err := r.db.QueryContext(ctx, `
		SELECT rp.role_id, p.code FROM role_permissions rp
		JOIN permissions p ON p.id = rp.permission_id ORDER BY p.code`)
	if err != nil {
		return nil, fmt.Errorf("rbac.ListRoles perms: %w", err)
	}
	defer permRows.Close()
	for permRows.Next() {
		var roleID int64
		var code string
		if err := permRows.Scan(&roleID, &code); err != nil {
			return nil, fmt.Errorf("rbac.ListRoles perms scan: %w", err)
		}
		if i, ok := index[roleID]; ok {
			roles[i].Permissions = append(roles[i].Permissions, code)
		}
	}
	return roles, permRows.Err()
}

// GetUserRoles mengembalikan slug role milik satu user.
func (r *Repository) GetUserRoles(ctx context.Context, adminUserID int64) ([]string, error) {
	access, err := r.GetUserAccess(ctx, adminUserID)
	if err != nil {
		return nil, err
	}
	return access.Roles, nil
}

// SetUserRoles mengganti seluruh role user (replace set) dalam satu transaksi,
// lalu menaikkan perm_version supaya klaim lama tidak dipakai menerus.
// Pencabutan refresh token dilakukan pemanggil (handler) lewat identity repo.
func (r *Repository) SetUserRoles(ctx context.Context, adminUserID int64, roleSlugs []string) error {
	if len(roleSlugs) == 0 {
		return errors.New("minimal satu role wajib dipilih")
	}

	placeholders := strings.Repeat("?,", len(roleSlugs))
	placeholders = placeholders[:len(placeholders)-1]
	args := make([]any, len(roleSlugs))
	for i, s := range roleSlugs {
		args[i] = s
	}

	// Validasi scope: role 'holding' hanya untuk user akses holding
	// (brand_id NULL), role 'brand' hanya untuk user terikat brand.
	var brandID sql.NullInt64
	err := r.db.QueryRowContext(ctx,
		`SELECT brand_id FROM admin_users WHERE id = ?`, adminUserID).Scan(&brandID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrUserNotFound
	}
	if err != nil {
		return fmt.Errorf("rbac.SetUserRoles user: %w", err)
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT slug, name, scope FROM roles WHERE slug IN (`+placeholders+`)`, args...)
	if err != nil {
		return fmt.Errorf("rbac.SetUserRoles validate: %w", err)
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var slug, name, scope string
		if err := rows.Scan(&slug, &name, &scope); err != nil {
			return fmt.Errorf("rbac.SetUserRoles validate scan: %w", err)
		}
		count++
		if scope == "holding" && brandID.Valid {
			return fmt.Errorf("%w: role %q hanya untuk akses Holding", ErrScopeMismatch, name)
		}
		if scope == "brand" && !brandID.Valid {
			return fmt.Errorf("%w: role %q hanya untuk user yang terikat brand", ErrScopeMismatch, name)
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if count != len(roleSlugs) {
		return ErrUnknownRole
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("rbac.SetUserRoles begin: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx,
		`UPDATE admin_users SET perm_version = perm_version + 1 WHERE id = ?`, adminUserID)
	if err != nil {
		return fmt.Errorf("rbac.SetUserRoles bump: %w", err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return ErrUserNotFound
	}

	if _, err := tx.ExecContext(ctx,
		`DELETE FROM user_roles WHERE admin_user_id = ?`, adminUserID); err != nil {
		return fmt.Errorf("rbac.SetUserRoles clear: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO user_roles (admin_user_id, role_id)
		 SELECT ?, id FROM roles WHERE slug IN (`+placeholders+`)`,
		append([]any{adminUserID}, args...)...); err != nil {
		return fmt.Errorf("rbac.SetUserRoles insert: %w", err)
	}

	return tx.Commit()
}

// AssignRole menambahkan satu role ke user tanpa menghapus role lain.
// Dipakai crmuser.Create untuk meng-assign role 'cs'.
func (r *Repository) AssignRole(ctx context.Context, adminUserID int64, roleSlug string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT IGNORE INTO user_roles (admin_user_id, role_id)
		 SELECT ?, id FROM roles WHERE slug = ?`, adminUserID, roleSlug)
	if err != nil {
		return fmt.Errorf("rbac.AssignRole: %w", err)
	}
	return nil
}
