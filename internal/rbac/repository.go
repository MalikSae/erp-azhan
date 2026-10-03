package rbac

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"erp-azhan/api/internal/identity"
)

var (
	ErrUserNotFound  = errors.New("pengguna tidak ditemukan")
	ErrUnknownRole   = errors.New("role tidak dikenal")
	ErrNoRole        = errors.New("minimal satu role wajib dipilih")
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
		`SELECT r.id, r.slug, r.name, r.is_system, r.scope, r.divisi_id, u.nama, r.created_at
		 FROM roles r LEFT JOIN org_unit u ON u.id = r.divisi_id ORDER BY r.id`)
	if err != nil {
		return nil, fmt.Errorf("rbac.ListRoles: %w", err)
	}
	defer rows.Close()
	roles := make([]Role, 0)
	index := make(map[int64]int)
	for rows.Next() {
		var role Role
		var divisiID sql.NullInt64
		var divisiNama sql.NullString
		if err := rows.Scan(&role.ID, &role.Slug, &role.Name, &role.IsSystem, &role.Scope, &divisiID, &divisiNama, &role.CreatedAt); err != nil {
			return nil, fmt.Errorf("rbac.ListRoles scan: %w", err)
		}
		if divisiID.Valid {
			role.DivisiID = &divisiID.Int64
			role.DivisiNama = &divisiNama.String
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

// SetUserRoles mengganti seluruh role user dalam SATU transaksi: validasi
// scope (R3), penjaga Super Admin terakhir (R4), ganti role + sinkron kolom
// lama + perm_version, dan cabut semua refresh token.
func (r *Repository) SetUserRoles(ctx context.Context, adminUserID int64, roleSlugs []string) error {
	roleSlugs = NormalizeSlugs(roleSlugs)
	if len(roleSlugs) == 0 {
		return ErrNoRole
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("rbac.SetUserRoles begin: %w", err)
	}
	defer tx.Rollback()

	var brandID sql.NullInt64
	var isActive bool
	err = tx.QueryRowContext(ctx,
		`SELECT brand_id, is_active FROM admin_users WHERE id = ? FOR UPDATE`, adminUserID).
		Scan(&brandID, &isActive)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrUserNotFound
	}
	if err != nil {
		return fmt.Errorf("rbac.SetUserRoles user: %w", err)
	}
	var bid *int64
	if brandID.Valid {
		bid = &brandID.Int64
	}

	if err := ValidateRolesScope(ctx, tx, roleSlugs, bid); err != nil {
		return err
	}
	// R10: penugasan yang sudah ada harus tetap sesuai role baru.
	units, err := UserUnitIDs(ctx, tx, adminUserID)
	if err != nil {
		return err
	}
	if err := ValidateUnitsForRoles(ctx, tx, roleSlugs, bid, units); err != nil {
		return err
	}

	current, err := UserRoleSlugs(ctx, tx, adminUserID)
	if err != nil {
		return err
	}
	if isActive && HasSlug(current, SuperAdminSlug) && !HasSlug(roleSlugs, SuperAdminSlug) {
		if err := EnsureOtherActiveSuperAdmin(ctx, tx, adminUserID); err != nil {
			return err
		}
	}

	if err := ReplaceUserRoles(ctx, tx, adminUserID, roleSlugs); err != nil {
		return err
	}
	if err := identity.RevokeAllRefreshTokensTx(ctx, tx, adminUserID); err != nil {
		return err
	}
	return tx.Commit()
}
