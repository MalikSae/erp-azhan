package identity

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Rotasi refresh token admin (temuan audit keamanan, 12 November 2026): setiap refresh mencabut
// token lama dan menerbitkan yang baru. Lihat migrations/136_rotasi_refresh_token_admin.sql.

// masaTenggangRotasi: token yang baru diganti masih diterima selama ini, supaya dua tab dashboard
// yang refresh hampir bersamaan tidak saling mengeluarkan.
const masaTenggangRotasi = 60 * time.Second

var (
	// ErrRefreshTidakValid: token tidak tercatat, kedaluwarsa, atau dicabut (logout/reset password).
	ErrRefreshTidakValid = errors.New("refresh token tidak valid")
	// ErrRefreshDipakaiUlang: token yang sudah diganti dipakai lagi setelah masa tenggang. Dianggap
	// bocor, sehingga semua refresh token user itu sudah dicabut.
	ErrRefreshDipakaiUlang = errors.New("refresh token dipakai ulang")
)

// RotasiRefreshToken mencabut refresh token lama (jtiLama) dan mencatat penggantinya (jtiBaru).
func (r *Repository) RotasiRefreshToken(ctx context.Context, jtiLama string, adminUserID int64, jtiBaru string, expBaru time.Time) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("identity.RotasiRefreshToken: %w", err)
	}
	defer tx.Rollback()
	errRotasi := rotasiRefreshTx(ctx, tx, jtiLama, adminUserID, jtiBaru, expBaru)
	// Pencabutan semua token saat dipakai ulang harus tersimpan walau rotasinya ditolak.
	if errRotasi != nil && !errors.Is(errRotasi, ErrRefreshDipakaiUlang) {
		return errRotasi
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("identity.RotasiRefreshToken commit: %w", err)
	}
	return errRotasi
}

func rotasiRefreshTx(ctx context.Context, tx *sql.Tx, jtiLama string, adminUserID int64, jtiBaru string, expBaru time.Time) error {
	var dicabut, diganti sql.NullTime
	var masihBerlaku, dalamTenggang bool
	err := tx.QueryRowContext(ctx, `
		SELECT revoked_at, diganti_at, expires_at > NOW(),
		       COALESCE(diganti_at > NOW() - INTERVAL ? SECOND, FALSE)
		FROM admin_refresh_tokens WHERE jti = ? AND admin_user_id = ? FOR UPDATE`,
		int(masaTenggangRotasi.Seconds()), jtiLama, adminUserID).
		Scan(&dicabut, &diganti, &masihBerlaku, &dalamTenggang)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrRefreshTidakValid
	}
	if err != nil {
		return fmt.Errorf("identity.rotasiRefresh baca: %w", err)
	}
	if !masihBerlaku {
		return ErrRefreshTidakValid
	}

	switch {
	case !dicabut.Valid:
		if _, err := tx.ExecContext(ctx,
			`UPDATE admin_refresh_tokens SET revoked_at = NOW(), diganti_at = NOW() WHERE jti = ?`, jtiLama); err != nil {
			return fmt.Errorf("identity.rotasiRefresh cabut: %w", err)
		}
	case diganti.Valid && dalamTenggang:
		// Tab lain baru saja merotasi token ini: terbitkan token baru tanpa mencabut apa pun.
	case diganti.Valid:
		if _, err := tx.ExecContext(ctx,
			`UPDATE admin_refresh_tokens SET revoked_at = NOW() WHERE admin_user_id = ? AND revoked_at IS NULL`,
			adminUserID); err != nil {
			return fmt.Errorf("identity.rotasiRefresh cabut semua: %w", err)
		}
		return ErrRefreshDipakaiUlang
	default:
		return ErrRefreshTidakValid
	}

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO admin_refresh_tokens (jti, admin_user_id, expires_at) VALUES (?, ?, ?)`,
		jtiBaru, adminUserID, expBaru.UTC()); err != nil {
		return fmt.Errorf("identity.rotasiRefresh simpan: %w", err)
	}
	return nil
}
