package rbac

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/go-sql-driver/mysql"
)

// SuperAdminSlug adalah role berkemampuan penuh (SPEK-RBAC). Sistem wajib
// selalu punya minimal satu pengguna AKTIF yang memegangnya (R4).
const SuperAdminSlug = "super_admin_grup"

var ErrLastSuperAdmin = errors.New("sistem harus punya minimal satu Super Admin aktif")

// Querier dipenuhi *sql.DB maupun *sql.Tx, sehingga aturan role yang sama
// bisa dijalankan di dalam transaksi milik modul lain (agregat pengguna).
type Querier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func slugArgs(slugs []string) (string, []any) {
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(slugs)), ",")
	args := make([]any, len(slugs))
	for i, s := range slugs {
		args[i] = s
	}
	return placeholders, args
}

// NormalizeSlugs membuang spasi & duplikat sambil mempertahankan urutan.
func NormalizeSlugs(slugs []string) []string {
	seen := make(map[string]bool, len(slugs))
	out := make([]string, 0, len(slugs))
	for _, s := range slugs {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

// ValidateRolesScope memastikan semua role ada dan scope-nya cocok dengan
// akses brand (R3): 'holding' hanya untuk brandID nil, 'brand' hanya untuk
// brandID terisi, 'semua' untuk keduanya. Satu-satunya implementasi aturan ini.
func ValidateRolesScope(ctx context.Context, q Querier, slugs []string, brandID *int64) error {
	if len(slugs) == 0 {
		return ErrNoRole
	}
	placeholders, args := slugArgs(slugs)
	rows, err := q.QueryContext(ctx,
		`SELECT name, scope FROM roles WHERE slug IN (`+placeholders+`)`, args...)
	if err != nil {
		return fmt.Errorf("rbac.ValidateRolesScope: %w", err)
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var name, scope string
		if err := rows.Scan(&name, &scope); err != nil {
			return fmt.Errorf("rbac.ValidateRolesScope scan: %w", err)
		}
		count++
		if scope == "holding" && brandID != nil {
			return fmt.Errorf("%w: role %q hanya untuk akses Holding", ErrScopeMismatch, name)
		}
		if scope == "brand" && brandID == nil {
			return fmt.Errorf("%w: role %q hanya untuk akses Brand", ErrScopeMismatch, name)
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if count != len(slugs) {
		return ErrUnknownRole
	}
	return nil
}

// UserRoleSlugs mengembalikan slug role yang sedang dipegang pengguna.
func UserRoleSlugs(ctx context.Context, q Querier, userID int64) ([]string, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT r.slug FROM user_roles ur JOIN roles r ON r.id = ur.role_id
		WHERE ur.admin_user_id = ? ORDER BY r.slug`, userID)
	if err != nil {
		return nil, fmt.Errorf("rbac.UserRoleSlugs: %w", err)
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// ReplaceUserRoles mengganti seluruh role pengguna dan menaikkan perm_version.
// Kolom lama admin_users.role ikut disinkronkan karena masih menjadi batas
// gateway CRM (RequireAdminOrCRMAccess): 'cs' bila role-nya HANYA Customer
// Service, selain itu 'admin'. Wajib dipanggil di dalam transaksi pemanggil.
func ReplaceUserRoles(ctx context.Context, q Querier, userID int64, slugs []string) error {
	if len(slugs) == 0 {
		return ErrNoRole
	}
	if _, err := q.ExecContext(ctx, `DELETE FROM user_roles WHERE admin_user_id = ?`, userID); err != nil {
		return fmt.Errorf("rbac.ReplaceUserRoles clear: %w", err)
	}
	placeholders, args := slugArgs(slugs)
	if _, err := q.ExecContext(ctx,
		`INSERT INTO user_roles (admin_user_id, role_id)
		 SELECT ?, id FROM roles WHERE slug IN (`+placeholders+`)`,
		append([]any{userID}, args...)...); err != nil {
		return fmt.Errorf("rbac.ReplaceUserRoles insert: %w", err)
	}
	legacy := "admin"
	if len(slugs) == 1 && slugs[0] == "cs" {
		legacy = "cs"
	}
	if _, err := q.ExecContext(ctx,
		`UPDATE admin_users SET role = ?, perm_version = perm_version + 1 WHERE id = ?`,
		legacy, userID); err != nil {
		return fmt.Errorf("rbac.ReplaceUserRoles sync: %w", err)
	}
	return nil
}

// EnsureOtherActiveSuperAdmin memastikan masih ada Super Admin aktif SELAIN
// excludeUserID (R4). Baris Super Admin aktif dikunci (FOR UPDATE) supaya dua
// transaksi yang saling menonaktifkan tidak bisa sama-sama lolos.
func EnsureOtherActiveSuperAdmin(ctx context.Context, q Querier, excludeUserID int64) error {
	rows, err := q.QueryContext(ctx, `
		SELECT u.id FROM admin_users u
		JOIN user_roles ur ON ur.admin_user_id = u.id
		JOIN roles r ON r.id = ur.role_id
		WHERE r.slug = ? AND u.is_active = TRUE
		FOR UPDATE`, SuperAdminSlug)
	if err != nil {
		return fmt.Errorf("rbac.EnsureOtherActiveSuperAdmin: %w", err)
	}
	defer rows.Close()
	others := 0
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return err
		}
		if id != excludeUserID {
			others++
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if others == 0 {
		return ErrLastSuperAdmin
	}
	return nil
}

// HasSlug true bila slug ada di daftar.
func HasSlug(slugs []string, slug string) bool {
	for _, s := range slugs {
		if s == slug {
			return true
		}
	}
	return false
}

var ErrUnitMismatch = errors.New("penugasan tidak sesuai role")

// ValidateUnitsForRoles menegakkan R10: unit di Penugasan harus berada di
// divisi asal salah satu role pengguna (roles.divisi_id; tim mengikuti divisi
// induknya). Super Admin boleh di unit mana pun. Pengguna akses Brand tidak
// masuk struktur Holding, jadi tidak boleh punya penugasan. Satu-satunya
// implementasi aturan ini.
func ValidateUnitsForRoles(ctx context.Context, q Querier, slugs []string, brandID *int64, unitIDs []int64) error {
	if len(unitIDs) == 0 {
		return nil
	}
	if brandID != nil {
		return fmt.Errorf("%w: pengguna akses Brand tidak masuk struktur organisasi Holding", ErrUnitMismatch)
	}
	if HasSlug(slugs, SuperAdminSlug) {
		return nil
	}
	allowed := map[int64]bool{}
	if len(slugs) > 0 {
		placeholders, args := slugArgs(slugs)
		rows, err := q.QueryContext(ctx,
			`SELECT divisi_id FROM roles WHERE slug IN (`+placeholders+`) AND divisi_id IS NOT NULL`, args...)
		if err != nil {
			return fmt.Errorf("rbac.ValidateUnitsForRoles roles: %w", err)
		}
		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return err
			}
			allowed[id] = true
		}
		rows.Close()
	}
	for _, uid := range unitIDs {
		var divisiID int64
		var nama string
		err := q.QueryRowContext(ctx,
			`SELECT COALESCE(parent_id, id), nama FROM org_unit WHERE id = ? FOR SHARE`, uid).Scan(&divisiID, &nama)
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("%w: unit #%d tidak ditemukan", ErrUnitMismatch, uid)
		}
		if err != nil {
			return fmt.Errorf("rbac.ValidateUnitsForRoles unit: %w", err)
		}
		if !allowed[divisiID] {
			return fmt.Errorf("%w: unit %q bukan divisi dari role yang dipilih", ErrUnitMismatch, nama)
		}
	}
	return nil
}

// UserUnitIDs mengembalikan unit tempat pengguna ditugaskan saat ini.
func UserUnitIDs(ctx context.Context, q Querier, userID int64) ([]int64, error) {
	rows, err := q.QueryContext(ctx, `SELECT org_unit_id FROM org_member WHERE admin_user_id = ?`, userID)
	if err != nil {
		return nil, fmt.Errorf("rbac.UserUnitIDs: %w", err)
	}
	defer rows.Close()
	out := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// ErrConcurrentMsg pesan untuk transaksi yang kalah balapan kunci (deadlock /
// lock wait timeout): aturan tetap aman, pengguna cukup mencoba lagi.
const ErrConcurrentMsg = "data sedang diubah bersamaan oleh admin lain — coba lagi"

// IsConcurrencyErr true untuk MySQL 1213 (deadlock) & 1205 (lock wait timeout).
func IsConcurrencyErr(err error) bool {
	var myErr *mysql.MySQLError
	return errors.As(err, &myErr) && (myErr.Number == 1213 || myErr.Number == 1205)
}
