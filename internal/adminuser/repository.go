package adminuser

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/go-sql-driver/mysql"

	"erp-azhan/api/internal/identity"
	"erp-azhan/api/internal/rbac"
)

var (
	ErrNotFound             = errors.New("pengguna tidak ditemukan")
	ErrDuplicateEmail       = errors.New("email sudah terdaftar")
	ErrInvalidBrand         = errors.New("brand tidak valid atau tidak ditemukan")
	ErrCannotDeleteSelf     = errors.New("tidak bisa menghapus akun sendiri")
	ErrCannotDeactivateSelf = errors.New("tidak bisa menonaktifkan akun sendiri")
	ErrHasHistory           = errors.New("pengguna ini punya riwayat transaksi/aktivitas — nonaktifkan saja, jangan dihapus")
	ErrInvalidKeanggotaan   = errors.New("keanggotaan unit tidak valid")
)

// Repository mengelola agregat Pengguna: profil + role + keanggotaan unit.
// Semua operasi tulis berjalan dalam SATU transaksi (SPEK §4.2).
type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// ─── Baca ─────────────────────────────────────────────────────────────────────

const selectUser = `
	SELECT u.id, COALESCE(u.display_name, ''), u.email, u.no_hp, u.jabatan,
	       u.brand_id, b.name, b.primary_color, u.is_active, u.created_at
	FROM admin_users u
	LEFT JOIN brands b ON u.brand_id = b.id`

func scanUser(scan func(dest ...any) error) (*AdminUser, error) {
	var u AdminUser
	var noHP, jabatan, brandName, brandColor sql.NullString
	var brandID sql.NullInt64
	if err := scan(&u.ID, &u.DisplayName, &u.Email, &noHP, &jabatan,
		&brandID, &brandName, &brandColor, &u.IsActive, &u.CreatedAt); err != nil {
		return nil, err
	}
	if noHP.Valid {
		u.NoHP = &noHP.String
	}
	if jabatan.Valid {
		u.Jabatan = &jabatan.String
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
	u.Roles = []string{}
	u.Keanggotaan = []Keanggotaan{}
	return &u, nil
}

// attachRelations mengisi Roles & Keanggotaan untuk sekumpulan pengguna dengan
// dua query (bukan N+1).
func (r *Repository) attachRelations(ctx context.Context, users []*AdminUser) error {
	if len(users) == 0 {
		return nil
	}
	byID := make(map[uint64]*AdminUser, len(users))
	for _, u := range users {
		byID[u.ID] = u
	}

	roleRows, err := r.db.QueryContext(ctx, `
		SELECT ur.admin_user_id, ro.slug FROM user_roles ur
		JOIN roles ro ON ro.id = ur.role_id ORDER BY ro.slug`)
	if err != nil {
		return fmt.Errorf("adminuser.attachRelations roles: %w", err)
	}
	defer roleRows.Close()
	for roleRows.Next() {
		var uid uint64
		var slug string
		if err := roleRows.Scan(&uid, &slug); err != nil {
			return err
		}
		if u, ok := byID[uid]; ok {
			u.Roles = append(u.Roles, slug)
		}
	}
	if err := roleRows.Err(); err != nil {
		return err
	}

	memRows, err := r.db.QueryContext(ctx, `
		SELECT m.admin_user_id, m.org_unit_id, o.nama, o.jenis, m.posisi
		FROM org_member m JOIN org_unit o ON o.id = m.org_unit_id
		ORDER BY o.jenis, o.urutan, o.nama`)
	if err != nil {
		return fmt.Errorf("adminuser.attachRelations keanggotaan: %w", err)
	}
	defer memRows.Close()
	for memRows.Next() {
		var uid uint64
		var k Keanggotaan
		if err := memRows.Scan(&uid, &k.OrgUnitID, &k.UnitNama, &k.Jenis, &k.Posisi); err != nil {
			return err
		}
		if u, ok := byID[uid]; ok {
			u.Keanggotaan = append(u.Keanggotaan, k)
		}
	}
	return memRows.Err()
}

func (r *Repository) List(ctx context.Context) ([]AdminUser, error) {
	rows, err := r.db.QueryContext(ctx, selectUser+` ORDER BY u.is_active DESC, u.display_name, u.id`)
	if err != nil {
		return nil, fmt.Errorf("adminuser.List: %w", err)
	}
	defer rows.Close()

	ptrs := make([]*AdminUser, 0)
	for rows.Next() {
		u, err := scanUser(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("adminuser.List scan: %w", err)
		}
		ptrs = append(ptrs, u)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := r.attachRelations(ctx, ptrs); err != nil {
		return nil, err
	}
	users := make([]AdminUser, len(ptrs))
	for i, u := range ptrs {
		users[i] = *u
	}
	return users, nil
}

func (r *Repository) GetByID(ctx context.Context, id uint64) (*AdminUser, error) {
	u, err := scanUser(r.db.QueryRowContext(ctx, selectUser+` WHERE u.id = ?`, id).Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("adminuser.GetByID: %w", err)
	}
	if err := r.attachRelations(ctx, []*AdminUser{u}); err != nil {
		return nil, err
	}
	return u, nil
}

// ─── Helper di dalam transaksi ───────────────────────────────────────────────

func toInt64Ptr(v *uint64) *int64 {
	if v == nil {
		return nil
	}
	x := int64(*v)
	return &x
}

func validateBrand(ctx context.Context, tx *sql.Tx, brandID *uint64) error {
	if brandID == nil {
		return nil
	}
	var n int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM brands WHERE id = ?`, *brandID).Scan(&n); err != nil {
		return fmt.Errorf("adminuser.validateBrand: %w", err)
	}
	if n == 0 {
		return ErrInvalidBrand
	}
	return nil
}

func checkEmailFree(ctx context.Context, tx *sql.Tx, email string, excludeID uint64) error {
	var n int
	if err := tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM admin_users WHERE email = ? AND id != ?`, email, excludeID).Scan(&n); err != nil {
		return fmt.Errorf("adminuser.checkEmailFree: %w", err)
	}
	if n > 0 {
		return ErrDuplicateEmail
	}
	return nil
}

// replaceKeanggotaan mengganti seluruh keanggotaan pengguna. Unit wajib ada &
// aktif; posisi wajib valid; satu unit tidak boleh muncul dua kali.
// Jumlah pemimpin per unit & jumlah unit per orang sengaja TIDAK dibatasi (K2, K3).
func replaceKeanggotaan(ctx context.Context, tx *sql.Tx, userID uint64, items []KeanggotaanInput) error {
	// Keanggotaan yang SUDAH ada boleh dipertahankan walau unitnya kini
	// nonaktif (supaya pengguna tetap bisa diedit); keanggotaan BARU wajib ke
	// unit aktif (R9).
	existing := map[int64]bool{}
	exRows, err := tx.QueryContext(ctx, `SELECT org_unit_id FROM org_member WHERE admin_user_id = ?`, userID)
	if err != nil {
		return fmt.Errorf("adminuser.replaceKeanggotaan existing: %w", err)
	}
	for exRows.Next() {
		var uid int64
		if err := exRows.Scan(&uid); err != nil {
			exRows.Close()
			return err
		}
		existing[uid] = true
	}
	exRows.Close()

	seen := make(map[int64]bool, len(items))
	for _, it := range items {
		if it.Posisi != "pemimpin" && it.Posisi != "anggota" {
			return fmt.Errorf("%w: posisi harus pemimpin atau anggota", ErrInvalidKeanggotaan)
		}
		if seen[it.OrgUnitID] {
			return fmt.Errorf("%w: unit yang sama dipilih dua kali", ErrInvalidKeanggotaan)
		}
		seen[it.OrgUnitID] = true
		var nama string
		var active bool
		err := tx.QueryRowContext(ctx,
			`SELECT nama, is_active FROM org_unit WHERE id = ? FOR SHARE`, it.OrgUnitID).Scan(&nama, &active)
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("%w: unit #%d tidak ditemukan", ErrInvalidKeanggotaan, it.OrgUnitID)
		}
		if err != nil {
			return fmt.Errorf("adminuser.replaceKeanggotaan cek unit: %w", err)
		}
		if !active && !existing[it.OrgUnitID] {
			return fmt.Errorf("%w: unit %q sudah nonaktif", ErrInvalidKeanggotaan, nama)
		}
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM org_member WHERE admin_user_id = ?`, userID); err != nil {
		return fmt.Errorf("adminuser.replaceKeanggotaan clear: %w", err)
	}
	for _, it := range items {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO org_member (admin_user_id, org_unit_id, posisi) VALUES (?, ?, ?)`,
			userID, it.OrgUnitID, it.Posisi); err != nil {
			return fmt.Errorf("adminuser.replaceKeanggotaan insert: %w", err)
		}
	}
	return nil
}

func unitIDs(items []KeanggotaanInput) []int64 {
	out := make([]int64, 0, len(items))
	for _, it := range items {
		out = append(out, it.OrgUnitID)
	}
	return out
}

func isMySQLErr(err error, code uint16) bool {
	var myErr *mysql.MySQLError
	return errors.As(err, &myErr) && myErr.Number == code
}

func sameSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	m := make(map[string]bool, len(a))
	for _, s := range a {
		m[s] = true
	}
	for _, s := range b {
		if !m[s] {
			return false
		}
	}
	return true
}

// ─── Tulis (agregat, atomik) ─────────────────────────────────────────────────

// Create membuat pengguna + role + keanggotaan dalam satu transaksi (R1–R3).
func (r *Repository) Create(ctx context.Context, in UpsertInput, passwordHash string) (uint64, error) {
	roles := rbac.NormalizeSlugs(in.Roles)

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("adminuser.Create begin: %w", err)
	}
	defer tx.Rollback()

	if err := checkEmailFree(ctx, tx, in.Email, 0); err != nil {
		return 0, err
	}
	if err := validateBrand(ctx, tx, in.BrandID); err != nil {
		return 0, err
	}
	if err := rbac.ValidateRolesScope(ctx, tx, roles, toInt64Ptr(in.BrandID)); err != nil {
		return 0, err
	}
	if err := rbac.ValidateUnitsForRoles(ctx, tx, roles, toInt64Ptr(in.BrandID), unitIDs(in.Keanggotaan)); err != nil {
		return 0, err
	}

	res, err := tx.ExecContext(ctx, `
		INSERT INTO admin_users (email, display_name, no_hp, jabatan, password_hash, brand_id, is_active)
		VALUES (?, ?, ?, ?, ?, ?, TRUE)`,
		in.Email, in.DisplayName, in.NoHP, in.Jabatan, passwordHash, in.BrandID)
	if err != nil {
		if isMySQLErr(err, 1062) {
			return 0, ErrDuplicateEmail
		}
		return 0, fmt.Errorf("adminuser.Create insert: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("adminuser.Create id: %w", err)
	}

	if err := rbac.ReplaceUserRoles(ctx, tx, id, roles); err != nil {
		return 0, err
	}
	if err := replaceKeanggotaan(ctx, tx, uint64(id), in.Keanggotaan); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("adminuser.Create commit: %w", err)
	}
	return uint64(id), nil
}

// Update mengubah profil, akses brand, role, dan keanggotaan dalam satu
// transaksi. R3 dicek terhadap akses brand BARU (menutup celah ubah akses
// tanpa cek role). R4 dijaga bila Super Admin dicabut. Perubahan role atau
// akses brand menaikkan perm_version dan mencabut sesi.
func (r *Repository) Update(ctx context.Context, id uint64, in UpsertInput) error {
	roles := rbac.NormalizeSlugs(in.Roles)

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("adminuser.Update begin: %w", err)
	}
	defer tx.Rollback()

	var oldBrand sql.NullInt64
	var isActive bool
	err = tx.QueryRowContext(ctx,
		`SELECT brand_id, is_active FROM admin_users WHERE id = ? FOR UPDATE`, id).Scan(&oldBrand, &isActive)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("adminuser.Update lock: %w", err)
	}

	if err := checkEmailFree(ctx, tx, in.Email, id); err != nil {
		return err
	}
	if err := validateBrand(ctx, tx, in.BrandID); err != nil {
		return err
	}
	if err := rbac.ValidateRolesScope(ctx, tx, roles, toInt64Ptr(in.BrandID)); err != nil {
		return err
	}
	if err := rbac.ValidateUnitsForRoles(ctx, tx, roles, toInt64Ptr(in.BrandID), unitIDs(in.Keanggotaan)); err != nil {
		return err
	}

	current, err := rbac.UserRoleSlugs(ctx, tx, int64(id))
	if err != nil {
		return err
	}
	if isActive && rbac.HasSlug(current, rbac.SuperAdminSlug) && !rbac.HasSlug(roles, rbac.SuperAdminSlug) {
		if err := rbac.EnsureOtherActiveSuperAdmin(ctx, tx, int64(id)); err != nil {
			return err
		}
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE admin_users SET email = ?, display_name = ?, no_hp = ?, jabatan = ?, brand_id = ?
		WHERE id = ?`,
		in.Email, in.DisplayName, in.NoHP, in.Jabatan, in.BrandID, id); err != nil {
		if isMySQLErr(err, 1062) {
			return ErrDuplicateEmail
		}
		return fmt.Errorf("adminuser.Update profil: %w", err)
	}

	roleChanged := !sameSet(current, roles)
	brandChanged := oldBrand.Valid != (in.BrandID != nil) ||
		(oldBrand.Valid && in.BrandID != nil && uint64(oldBrand.Int64) != *in.BrandID)

	if roleChanged {
		if err := rbac.ReplaceUserRoles(ctx, tx, int64(id), roles); err != nil {
			return err
		}
	} else if brandChanged {
		// brand_id ada di klaim JWT — klaim lama harus mati.
		if _, err := tx.ExecContext(ctx,
			`UPDATE admin_users SET perm_version = perm_version + 1 WHERE id = ?`, id); err != nil {
			return fmt.Errorf("adminuser.Update bump: %w", err)
		}
	}
	if roleChanged || brandChanged {
		if err := identity.RevokeAllRefreshTokensTx(ctx, tx, int64(id)); err != nil {
			return err
		}
	}

	if err := replaceKeanggotaan(ctx, tx, id, in.Keanggotaan); err != nil {
		return err
	}
	return tx.Commit()
}

// SetStatus mengaktifkan/menonaktifkan pengguna (offboarding). Nonaktif:
// tidak boleh diri sendiri (R5), tidak boleh Super Admin aktif terakhir (R4),
// dan sesi dicabut + perm_version naik (R6) — semua dalam satu transaksi.
func (r *Repository) SetStatus(ctx context.Context, id uint64, active bool, actorID uint64) error {
	if !active && id == actorID {
		return ErrCannotDeactivateSelf
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("adminuser.SetStatus begin: %w", err)
	}
	defer tx.Rollback()

	var isActive bool
	err = tx.QueryRowContext(ctx,
		`SELECT is_active FROM admin_users WHERE id = ? FOR UPDATE`, id).Scan(&isActive)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("adminuser.SetStatus lock: %w", err)
	}
	if isActive == active {
		return tx.Commit() // tidak ada perubahan
	}

	if !active {
		current, err := rbac.UserRoleSlugs(ctx, tx, int64(id))
		if err != nil {
			return err
		}
		if rbac.HasSlug(current, rbac.SuperAdminSlug) {
			if err := rbac.EnsureOtherActiveSuperAdmin(ctx, tx, int64(id)); err != nil {
				return err
			}
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE admin_users SET is_active = FALSE, perm_version = perm_version + 1 WHERE id = ?`, id); err != nil {
			return fmt.Errorf("adminuser.SetStatus nonaktif: %w", err)
		}
		if err := identity.RevokeAllRefreshTokensTx(ctx, tx, int64(id)); err != nil {
			return err
		}
	} else {
		if _, err := tx.ExecContext(ctx,
			`UPDATE admin_users SET is_active = TRUE WHERE id = ?`, id); err != nil {
			return fmt.Errorf("adminuser.SetStatus aktif: %w", err)
		}
	}
	return tx.Commit()
}

// Delete menghapus permanen HANYA pengguna tanpa riwayat (R7). Riwayat
// dideteksi dari FK (MySQL 1451) — mencakup otomatis tabel baru di masa depan —
// ditambah payments.verified_by yang ber-ON DELETE SET NULL (akan menghapus
// jejak verifikasi bila tidak dicegah).
func (r *Repository) Delete(ctx context.Context, id uint64, actorID uint64) error {
	if id == actorID {
		return ErrCannotDeleteSelf
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("adminuser.Delete begin: %w", err)
	}
	defer tx.Rollback()

	var isActive bool
	err = tx.QueryRowContext(ctx,
		`SELECT is_active FROM admin_users WHERE id = ? FOR UPDATE`, id).Scan(&isActive)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("adminuser.Delete lock: %w", err)
	}

	current, err := rbac.UserRoleSlugs(ctx, tx, int64(id))
	if err != nil {
		return err
	}
	if isActive && rbac.HasSlug(current, rbac.SuperAdminSlug) {
		if err := rbac.EnsureOtherActiveSuperAdmin(ctx, tx, int64(id)); err != nil {
			return err
		}
	}

	var verified int
	if err := tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM payments WHERE verified_by = ?`, id).Scan(&verified); err != nil {
		return fmt.Errorf("adminuser.Delete cek verifikasi: %w", err)
	}
	if verified > 0 {
		return ErrHasHistory
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM admin_users WHERE id = ?`, id); err != nil {
		if isMySQLErr(err, 1451) {
			return ErrHasHistory
		}
		return fmt.Errorf("adminuser.Delete: %w", err)
	}
	return tx.Commit()
}

// ─── Password ────────────────────────────────────────────────────────────────

func (r *Repository) ResetPassword(ctx context.Context, id uint64, passwordHash string) error {
	res, err := r.db.ExecContext(ctx, `UPDATE admin_users SET password_hash = ? WHERE id = ?`, passwordHash, id)
	if err != nil {
		return fmt.Errorf("adminuser.ResetPassword: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		// RowsAffected 0 juga terjadi bila hash identik; pastikan pengguna ada.
		var exists int
		if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM admin_users WHERE id = ?`, id).Scan(&exists); err != nil {
			return fmt.Errorf("adminuser.ResetPassword cek: %w", err)
		}
		if exists == 0 {
			return ErrNotFound
		}
	}
	return nil
}

func (r *Repository) GetPasswordHash(ctx context.Context, id uint64) (string, error) {
	var passwordHash string
	err := r.db.QueryRowContext(ctx, "SELECT password_hash FROM admin_users WHERE id = ?", id).Scan(&passwordHash)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("adminuser.GetPasswordHash: %w", err)
	}
	return passwordHash, nil
}

// normalizeOptional mengubah string kosong menjadi nil (kolom opsional).
func normalizeOptional(s *string) *string {
	if s == nil {
		return nil
	}
	v := strings.TrimSpace(*s)
	if v == "" {
		return nil
	}
	return &v
}
