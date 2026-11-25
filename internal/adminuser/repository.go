package adminuser

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/go-sql-driver/mysql"

	"erp-azhan/api/internal/akses"
	"erp-azhan/api/internal/identity"
)

var (
	ErrNotFound             = errors.New("pengguna tidak ditemukan")
	ErrDuplicateEmail       = errors.New("email sudah terdaftar")
	ErrInvalidBrand         = errors.New("brand tidak valid atau tidak ditemukan")
	ErrCannotDeleteSelf     = errors.New("tidak bisa menghapus akun sendiri")
	ErrCannotDeactivateSelf = errors.New("tidak bisa menonaktifkan akun sendiri")
	ErrHasHistory           = errors.New("pengguna ini punya riwayat transaksi/aktivitas — nonaktifkan saja, jangan dihapus")
)

// Repository mengelola agregat Pengguna: profil + jenis akses + izin.
// Semua operasi tulis berjalan dalam SATU transaksi.
type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// ─── Baca ─────────────────────────────────────────────────────────────────────

const selectUser = `
	SELECT u.id, COALESCE(u.display_name, ''), u.email, u.no_hp,
	       u.brand_id, b.name, b.primary_color, u.is_active, u.is_super_admin, u.created_at
	FROM admin_users u
	LEFT JOIN brands b ON u.brand_id = b.id`

func scanUser(scan func(dest ...any) error) (*AdminUser, error) {
	var u AdminUser
	var noHP, brandName, brandColor sql.NullString
	var brandID sql.NullInt64
	var super bool
	if err := scan(&u.ID, &u.DisplayName, &u.Email, &noHP,
		&brandID, &brandName, &brandColor, &u.IsActive, &super, &u.CreatedAt); err != nil {
		return nil, err
	}
	if noHP.Valid {
		u.NoHP = &noHP.String
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
	u.Akses = akses.Tim
	if super {
		u.Akses = akses.SuperAdmin
	}
	u.Permissions = []string{}
	return &u, nil
}

// attachIzin mengisi Permissions pengguna Tim dengan satu query (bukan N+1).
func (r *Repository) attachIzin(ctx context.Context, users []*AdminUser) error {
	if len(users) == 0 {
		return nil
	}
	byID := make(map[uint64]*AdminUser, len(users))
	for _, u := range users {
		byID[u.ID] = u
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT up.admin_user_id, p.code FROM user_permissions up
		JOIN permissions p ON p.id = up.permission_id ORDER BY p.code`)
	if err != nil {
		return fmt.Errorf("adminuser.attachIzin: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var uid uint64
		var code string
		if err := rows.Scan(&uid, &code); err != nil {
			return err
		}
		if u, ok := byID[uid]; ok && u.Akses == akses.Tim {
			u.Permissions = append(u.Permissions, code)
		}
	}
	return rows.Err()
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
	if err := r.attachIzin(ctx, ptrs); err != nil {
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
	if err := r.attachIzin(ctx, []*AdminUser{u}); err != nil {
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

func isMySQLErr(err error, code uint16) bool {
	var myErr *mysql.MySQLError
	return errors.As(err, &myErr) && myErr.Number == code
}

// izinDisimpan mengembalikan izin yang akan disimpan: kosong untuk Super Admin.
func izinDisimpan(in UpsertInput) []string {
	if in.Akses == akses.SuperAdmin {
		return []string{}
	}
	return akses.Normalize(in.Permissions)
}

// validasiAkses menegakkan A1–A5 terhadap akses brand yang akan disimpan.
func validasiAkses(ctx context.Context, tx *sql.Tx, in UpsertInput, izin []string) error {
	brandID := toInt64Ptr(in.BrandID)
	if err := akses.ValidasiJenis(in.Akses, brandID); err != nil {
		return err
	}
	if in.Akses == akses.SuperAdmin {
		return nil
	}
	return akses.ValidasiIzinTim(ctx, tx, izin, brandID)
}

// ─── Tulis (agregat, atomik) ─────────────────────────────────────────────────

// Create membuat pengguna beserta jenis akses dan izinnya dalam satu transaksi.
func (r *Repository) Create(ctx context.Context, in UpsertInput, passwordHash string) (uint64, error) {
	izin := izinDisimpan(in)

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
	if err := validasiAkses(ctx, tx, in, izin); err != nil {
		return 0, err
	}

	res, err := tx.ExecContext(ctx, `
		INSERT INTO admin_users (email, display_name, no_hp, password_hash, brand_id, is_active)
		VALUES (?, ?, ?, ?, ?, TRUE)`,
		in.Email, in.DisplayName, in.NoHP, passwordHash, in.BrandID)
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

	if err := akses.Ganti(ctx, tx, id, in.Akses, izin); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("adminuser.Create commit: %w", err)
	}
	return uint64(id), nil
}

// Update mengubah profil, akses brand, jenis akses, dan izin dalam satu
// transaksi. Izin divalidasi terhadap akses brand BARU. A6 dijaga bila Super
// Admin diturunkan. Perubahan akses atau akses brand menaikkan perm_version dan
// mencabut sesi (A7).
func (r *Repository) Update(ctx context.Context, id uint64, in UpsertInput) error {
	izin := izinDisimpan(in)

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
	if err := validasiAkses(ctx, tx, in, izin); err != nil {
		return err
	}

	jenisLama, izinLama, err := akses.Pengguna(ctx, tx, int64(id))
	if err != nil {
		return err
	}
	if isActive && jenisLama == akses.SuperAdmin && in.Akses != akses.SuperAdmin {
		if err := akses.EnsureOtherActiveSuperAdmin(ctx, tx, int64(id)); err != nil {
			return err
		}
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE admin_users SET email = ?, display_name = ?, no_hp = ?, brand_id = ?
		WHERE id = ?`,
		in.Email, in.DisplayName, in.NoHP, in.BrandID, id); err != nil {
		if isMySQLErr(err, 1062) {
			return ErrDuplicateEmail
		}
		return fmt.Errorf("adminuser.Update profil: %w", err)
	}

	aksesBerubah := jenisLama != in.Akses || !akses.SamaIzin(izinLama, izin)
	brandChanged := oldBrand.Valid != (in.BrandID != nil) ||
		(oldBrand.Valid && in.BrandID != nil && uint64(oldBrand.Int64) != *in.BrandID)

	if aksesBerubah {
		if err := akses.Ganti(ctx, tx, int64(id), in.Akses, izin); err != nil {
			return err
		}
	} else if brandChanged {
		// brand_id ada di klaim JWT — klaim lama harus mati.
		if _, err := tx.ExecContext(ctx,
			`UPDATE admin_users SET perm_version = perm_version + 1 WHERE id = ?`, id); err != nil {
			return fmt.Errorf("adminuser.Update bump: %w", err)
		}
	}
	if aksesBerubah || brandChanged {
		if err := identity.RevokeAllRefreshTokensTx(ctx, tx, int64(id)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// SetStatus mengaktifkan/menonaktifkan pengguna (offboarding). Nonaktif:
// tidak boleh diri sendiri, tidak boleh Super Admin aktif terakhir (A6),
// dan sesi dicabut + perm_version naik — semua dalam satu transaksi.
func (r *Repository) SetStatus(ctx context.Context, id uint64, active bool, actorID uint64) error {
	if !active && id == actorID {
		return ErrCannotDeactivateSelf
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("adminuser.SetStatus begin: %w", err)
	}
	defer tx.Rollback()

	var isActive, super bool
	err = tx.QueryRowContext(ctx,
		`SELECT is_active, is_super_admin FROM admin_users WHERE id = ? FOR UPDATE`, id).Scan(&isActive, &super)
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
		if super {
			if err := akses.EnsureOtherActiveSuperAdmin(ctx, tx, int64(id)); err != nil {
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

// Delete menghapus permanen HANYA pengguna tanpa riwayat. Riwayat
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

	var isActive, super bool
	err = tx.QueryRowContext(ctx,
		`SELECT is_active, is_super_admin FROM admin_users WHERE id = ? FOR UPDATE`, id).Scan(&isActive, &super)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("adminuser.Delete lock: %w", err)
	}

	if isActive && super {
		if err := akses.EnsureOtherActiveSuperAdmin(ctx, tx, int64(id)); err != nil {
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

// ResetPassword mengganti password dan menaikkan perm_version. Kenaikan versi membuat access token lama
// ditolak pada permintaan berikutnya (identity.StatusChecker), jadi token yang sudah bocor tidak bertahan
// sampai kedaluwarsa. Refresh token dicabut pemanggil.
func (r *Repository) ResetPassword(ctx context.Context, id uint64, passwordHash string) error {
	res, err := r.db.ExecContext(ctx, `UPDATE admin_users SET password_hash = ?, perm_version = perm_version + 1 WHERE id = ?`, passwordHash, id)
	if err != nil {
		return fmt.Errorf("adminuser.ResetPassword: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("adminuser.ResetPassword baris: %w", err)
	}
	// perm_version selalu berubah, jadi nol baris berarti pengguna tidak ada.
	if n == 0 {
		return ErrNotFound
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
