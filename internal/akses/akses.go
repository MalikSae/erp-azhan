// Package akses menegakkan aturan akses pengguna dashboard
// (docs/spek/SPEK-AKSES-PENGGUNA-2026-11-25.md): dua jenis akses, Super Admin
// dan Tim. Super Admin memegang semua izin; izin Tim disimpan per pengguna di
// user_permissions.
package akses

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/go-sql-driver/mysql"
)

const (
	SuperAdmin = "super_admin"
	Tim        = "tim"
)

var (
	ErrUserNotFound    = errors.New("Pengguna tidak ditemukan.")
	ErrJenisTidakSah   = errors.New("Pilih jenis akses.")
	ErrSuperAdminBrand = errors.New("Super Admin hanya untuk akses Holding.")
	ErrTanpaIzin       = errors.New("Pilih minimal satu izin.")
	ErrIzinTidakSah    = errors.New("Ada izin yang tidak dapat diberikan.")
	ErrLastSuperAdmin  = errors.New("Harus ada minimal satu Super Admin aktif.")
)

// hanyaSuperAdmin: izin yang tidak bisa diberikan ke Tim (A4).
var hanyaSuperAdmin = map[string]bool{
	"user.admin":      true,
	"brand.edit":      true,
	"crmdeal.process": true,
}

// izinBrand: izin yang boleh diberikan ke pengguna terikat brand (A5). Sama
// dengan kolom Brand di spek §4 dan penanda `brand` di
// frontend/master-dashboard/src/constants/izinAkses.js; ubah ketiganya bersamaan.
var izinBrand = map[string]bool{
	"jamaah.view": true, "jamaah.edit": true, "dokumen.view": true, "dokumen.edit": true,
	"booking.view": true, "booking.edit": true,
	"payment.view": true, "payment.edit": true, "refund.view": true, "refund.edit": true,
	"akun.view": true, "akun.edit": true,
	"agen.view": true, "agen.edit": true, "komisi.view": true,
	"schedule.view":   true,
	"manifest.manage": true,
	"manasik.view":    true, "manasik.edit": true,
	"masterdata.view":   true,
	"perlengkapan.view": true, "perlengkapan.edit": true,
	"bankaccount.view": true, "bankaccount.edit": true,
	"brand.view":        true,
	"pengumuman.manage": true,
	"banner.manage":     true,
	"payment.verify":    true, "payment.void": true,
	"agen.approve":             true,
	"jamaah.perubahan.approve": true,
}

// Querier dipenuhi *sql.DB maupun *sql.Tx, sehingga aturan yang sama bisa
// dijalankan di dalam transaksi milik modul lain (agregat pengguna).
type Querier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Normalize membuang spasi dan duplikat, lalu mengurutkan kode izin.
func Normalize(codes []string) []string {
	seen := make(map[string]bool, len(codes))
	out := make([]string, 0, len(codes))
	for _, c := range codes {
		c = strings.TrimSpace(c)
		if c == "" || seen[c] {
			continue
		}
		seen[c] = true
		out = append(out, c)
	}
	sort.Strings(out)
	return out
}

// ValidasiJenis memeriksa jenis akses terhadap akses brand (A1, A2).
func ValidasiJenis(jenis string, brandID *int64) error {
	switch jenis {
	case SuperAdmin:
		if brandID != nil {
			return ErrSuperAdminBrand
		}
		return nil
	case Tim:
		return nil
	}
	return ErrJenisTidakSah
}

// ValidasiIzinTim memeriksa izin pengguna Tim (A3–A5): minimal satu, bukan izin
// khusus Super Admin, dalam batas izin brand bila pengguna terikat brand, dan
// semua kode ada di tabel permissions. Berhenti di pelanggaran pertama.
func ValidasiIzinTim(ctx context.Context, q Querier, codes []string, brandID *int64) error {
	if len(codes) == 0 {
		return ErrTanpaIzin
	}
	for _, c := range codes {
		if hanyaSuperAdmin[c] {
			return ErrIzinTidakSah
		}
		if brandID != nil && !izinBrand[c] {
			return ErrIzinTidakSah
		}
	}
	placeholders, args := inArgs(codes)
	var n int
	if err := q.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM permissions WHERE code IN (`+placeholders+`)`, args...).Scan(&n); err != nil {
		return fmt.Errorf("akses.ValidasiIzinTim: %w", err)
	}
	if n != len(codes) {
		return ErrIzinTidakSah
	}
	return nil
}

func inArgs(codes []string) (string, []any) {
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(codes)), ",")
	args := make([]any, len(codes))
	for i, c := range codes {
		args[i] = c
	}
	return placeholders, args
}

// Pengguna mengembalikan jenis akses dan izin tersimpan seorang pengguna.
// Untuk Super Admin, izin tersimpan selalu kosong (ia memegang semuanya).
func Pengguna(ctx context.Context, q Querier, userID int64) (jenis string, codes []string, err error) {
	var super bool
	err = q.QueryRowContext(ctx, `SELECT is_super_admin FROM admin_users WHERE id = ?`, userID).Scan(&super)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil, ErrUserNotFound
	}
	if err != nil {
		return "", nil, fmt.Errorf("akses.Pengguna: %w", err)
	}
	if super {
		return SuperAdmin, []string{}, nil
	}
	rows, err := q.QueryContext(ctx, `
		SELECT p.code FROM user_permissions up JOIN permissions p ON p.id = up.permission_id
		WHERE up.admin_user_id = ? ORDER BY p.code`, userID)
	if err != nil {
		return "", nil, fmt.Errorf("akses.Pengguna izin: %w", err)
	}
	defer rows.Close()
	codes = []string{}
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return "", nil, err
		}
		codes = append(codes, c)
	}
	return Tim, codes, rows.Err()
}

// Ganti menyimpan jenis akses dan izin pengguna (replace-all) serta menaikkan
// perm_version. Wajib dipanggil di dalam transaksi pemanggil, setelah validasi.
func Ganti(ctx context.Context, q Querier, userID int64, jenis string, codes []string) error {
	if _, err := q.ExecContext(ctx, `DELETE FROM user_permissions WHERE admin_user_id = ?`, userID); err != nil {
		return fmt.Errorf("akses.Ganti hapus: %w", err)
	}
	if jenis == Tim && len(codes) > 0 {
		placeholders, args := inArgs(codes)
		if _, err := q.ExecContext(ctx,
			`INSERT INTO user_permissions (admin_user_id, permission_id)
			 SELECT ?, id FROM permissions WHERE code IN (`+placeholders+`)`,
			append([]any{userID}, args...)...); err != nil {
			return fmt.Errorf("akses.Ganti simpan: %w", err)
		}
	}
	if _, err := q.ExecContext(ctx,
		`UPDATE admin_users SET is_super_admin = ?, perm_version = perm_version + 1 WHERE id = ?`,
		jenis == SuperAdmin, userID); err != nil {
		return fmt.Errorf("akses.Ganti jenis: %w", err)
	}
	return nil
}

// EnsureOtherActiveSuperAdmin memastikan masih ada Super Admin aktif SELAIN
// excludeUserID (A6). Baris Super Admin aktif dikunci (FOR UPDATE) supaya dua
// transaksi yang saling menonaktifkan tidak bisa sama-sama lolos.
func EnsureOtherActiveSuperAdmin(ctx context.Context, q Querier, excludeUserID int64) error {
	rows, err := q.QueryContext(ctx,
		`SELECT id FROM admin_users WHERE is_super_admin = TRUE AND is_active = TRUE FOR UPDATE`)
	if err != nil {
		return fmt.Errorf("akses.EnsureOtherActiveSuperAdmin: %w", err)
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

// SamaIzin true bila dua daftar kode (sudah dinormalisasi) sama persis.
func SamaIzin(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// ErrConcurrentMsg pesan untuk transaksi yang kalah balapan kunci (deadlock /
// lock wait timeout): aturan tetap aman, pengguna cukup mencoba lagi.
const ErrConcurrentMsg = "Data sedang diubah admin lain. Coba lagi."

// IsConcurrencyErr true untuk MySQL 1213 (deadlock) & 1205 (lock wait timeout).
func IsConcurrencyErr(err error) bool {
	var myErr *mysql.MySQLError
	return errors.As(err, &myErr) && (myErr.Number == 1213 || myErr.Number == 1205)
}
