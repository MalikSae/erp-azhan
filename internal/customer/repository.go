// Package customer berisi akun pelanggan (email + password) untuk situs booking
// publik: daftar, login, sesi, verifikasi email, dan reset password.
package customer

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

var (
	ErrEmailTaken     = errors.New("email sudah terdaftar")
	ErrPhoneTaken     = errors.New("nomor WhatsApp sudah terdaftar")
	ErrNotFound       = errors.New("akun tidak ditemukan")
	ErrTokenInvalid   = errors.New("tautan tidak valid atau sudah kedaluwarsa")
	ErrSessionInvalid = errors.New("sesi tidak valid")
)

const (
	SessionTTL        = 30 * 24 * time.Hour
	VerifyTokenTTL    = 24 * time.Hour
	ResetTokenTTL     = time.Hour
	PurposeVerifyMail = "verify_email"
	PurposeResetPass  = "reset_password"
)

type Account struct {
	ID              int64      `json:"id"`
	BrandID         int64      `json:"brand_id"`
	NamaLengkap     string     `json:"nama_lengkap"`
	Email           string     `json:"email"`
	NoHP            string     `json:"no_hp"`
	EmailVerifiedAt *time.Time `json:"email_verified_at"`
	JamaahID        *int64     `json:"-"`
	passwordHash    string
}

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

// NormalizeEmail: huruf kecil dan tanpa spasi tepi. Dipakai untuk kunci unik per brand.
func NormalizeEmail(email string) string { return strings.ToLower(strings.TrimSpace(email)) }

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func newRawToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

const accountCols = `id, brand_id, nama_lengkap, email, no_hp, email_verified_at, jamaah_id, password_hash`

func scanAccount(row interface{ Scan(...any) error }) (*Account, error) {
	var a Account
	var verified sql.NullTime
	var jamaah sql.NullInt64
	if err := row.Scan(&a.ID, &a.BrandID, &a.NamaLengkap, &a.Email, &a.NoHP, &verified, &jamaah, &a.passwordHash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if verified.Valid {
		t := verified.Time
		a.EmailVerifiedAt = &t
	}
	if jamaah.Valid {
		a.JamaahID = &jamaah.Int64
	}
	return &a, nil
}

func (r *Repository) Create(ctx context.Context, brandID int64, nama, email, noHP, passwordHash string) (*Account, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO customer_accounts (brand_id, nama_lengkap, email, email_normal, no_hp, password_hash) VALUES (?,?,?,?,?,?)`,
		brandID, nama, strings.TrimSpace(email), NormalizeEmail(email), noHP, passwordHash)
	if err != nil {
		return nil, mapDuplicate(err, "create account")
	}
	id, _ := res.LastInsertId()
	return r.GetByID(ctx, id)
}

// mapDuplicate mengubah pelanggaran kunci unik (1062) menjadi ErrPhoneTaken atau ErrEmailTaken; galat lain dibungkus dengan op.
func mapDuplicate(err error, op string) error {
	var me *mysql.MySQLError
	if errors.As(err, &me) && me.Number == 1062 {
		if strings.Contains(me.Message, "uq_customer_brand_hp") {
			return ErrPhoneTaken
		}
		return ErrEmailTaken
	}
	return fmt.Errorf("%s: %w", op, err)
}

// UpdateProfile mengganti nama, email, dan nomor WhatsApp akun. Email yang berubah membatalkan status terverifikasi.
// email_verified_at harus ditulis SEBELUM email_normal: MySQL mengevaluasi SET dari kiri ke kanan, jadi pembandingnya
// masih membaca email lama.
func (r *Repository) UpdateProfile(ctx context.Context, id int64, nama, email, noHP string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE customer_accounts SET email_verified_at = IF(email_normal <> ?, NULL, email_verified_at),
		 nama_lengkap=?, email=?, email_normal=?, no_hp=? WHERE id=?`,
		NormalizeEmail(email), nama, strings.TrimSpace(email), NormalizeEmail(email), noHP, id)
	if err != nil {
		return mapDuplicate(err, "update profile")
	}
	return nil
}

func (r *Repository) GetByID(ctx context.Context, id int64) (*Account, error) {
	return scanAccount(r.db.QueryRowContext(ctx, `SELECT `+accountCols+` FROM customer_accounts WHERE id=?`, id))
}

func (r *Repository) GetByEmail(ctx context.Context, brandID int64, email string) (*Account, error) {
	return scanAccount(r.db.QueryRowContext(ctx,
		`SELECT `+accountCols+` FROM customer_accounts WHERE brand_id=? AND email_normal=?`, brandID, NormalizeEmail(email)))
}

func (r *Repository) TouchLogin(ctx context.Context, id int64) {
	_, _ = r.db.ExecContext(ctx, `UPDATE customer_accounts SET last_login_at=? WHERE id=?`, time.Now(), id)
}

// CreateSession mengembalikan token mentah untuk cookie; hanya hash-nya disimpan.
func (r *Repository) CreateSession(ctx context.Context, accountID int64) (string, error) {
	raw, err := newRawToken()
	if err != nil {
		return "", err
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO customer_sessions (token_hash, account_id, expires_at) VALUES (?,?,?)`,
		hashToken(raw), accountID, time.Now().Add(SessionTTL))
	if err != nil {
		return "", fmt.Errorf("create session: %w", err)
	}
	return raw, nil
}

func (r *Repository) AccountBySession(ctx context.Context, raw string) (*Account, error) {
	if raw == "" {
		return nil, ErrSessionInvalid
	}
	a, err := scanAccount(r.db.QueryRowContext(ctx, `
		SELECT a.id, a.brand_id, a.nama_lengkap, a.email, a.no_hp, a.email_verified_at, a.jamaah_id, a.password_hash
		FROM customer_sessions s JOIN customer_accounts a ON a.id = s.account_id
		WHERE s.token_hash=? AND s.revoked_at IS NULL AND s.expires_at > ?`, hashToken(raw), time.Now()))
	if errors.Is(err, ErrNotFound) {
		return nil, ErrSessionInvalid
	}
	return a, err
}

func (r *Repository) RevokeSession(ctx context.Context, raw string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE customer_sessions SET revoked_at=? WHERE token_hash=? AND revoked_at IS NULL`, time.Now(), hashToken(raw))
	return err
}

func (r *Repository) RevokeAllSessions(ctx context.Context, accountID int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE customer_sessions SET revoked_at=? WHERE account_id=? AND revoked_at IS NULL`, time.Now(), accountID)
	return err
}

// InvalidateTokens menandai semua token aktif akun untuk tujuan tertentu sebagai terpakai (mis. tautan reset lama).
func (r *Repository) InvalidateTokens(ctx context.Context, accountID int64, purpose string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE customer_tokens SET used_at=? WHERE account_id=? AND purpose=? AND used_at IS NULL`, time.Now(), accountID, purpose)
	return err
}

// RevokeOtherSessions mencabut semua sesi akun kecuali sesi yang sedang dipakai (token mentah keep).
func (r *Repository) RevokeOtherSessions(ctx context.Context, accountID int64, keep string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE customer_sessions SET revoked_at=? WHERE account_id=? AND token_hash<>? AND revoked_at IS NULL`,
		time.Now(), accountID, hashToken(keep))
	return err
}

// IssueToken membuat token sekali pakai; token lama untuk tujuan yang sama dinonaktifkan.
func (r *Repository) IssueToken(ctx context.Context, accountID int64, purpose string, ttl time.Duration) (string, error) {
	raw, err := newRawToken()
	if err != nil {
		return "", err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `UPDATE customer_tokens SET used_at=? WHERE account_id=? AND purpose=? AND used_at IS NULL`, time.Now(), accountID, purpose); err != nil {
		return "", err
	}
	if _, err = tx.ExecContext(ctx,
		`INSERT INTO customer_tokens (token_hash, account_id, purpose, expires_at) VALUES (?,?,?,?)`,
		hashToken(raw), accountID, purpose, time.Now().Add(ttl)); err != nil {
		return "", err
	}
	return raw, tx.Commit()
}

// ConsumeToken menandai token terpakai secara atomik dan mengembalikan pemiliknya.
func (r *Repository) ConsumeToken(ctx context.Context, raw, purpose string) (int64, error) {
	if raw == "" {
		return 0, ErrTokenInvalid
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var accountID int64
	err = tx.QueryRowContext(ctx,
		`SELECT account_id FROM customer_tokens WHERE token_hash=? AND purpose=? AND used_at IS NULL AND expires_at > ? FOR UPDATE`,
		hashToken(raw), purpose, time.Now()).Scan(&accountID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrTokenInvalid
	}
	if err != nil {
		return 0, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE customer_tokens SET used_at=? WHERE token_hash=?`, time.Now(), hashToken(raw)); err != nil {
		return 0, err
	}
	return accountID, tx.Commit()
}

func (r *Repository) MarkEmailVerified(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE customer_accounts SET email_verified_at=COALESCE(email_verified_at, ?) WHERE id=?`, time.Now(), id)
	return err
}

func (r *Repository) SetPassword(ctx context.Context, id int64, hash string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE customer_accounts SET password_hash=? WHERE id=?`, hash, id)
	return err
}

// BrandSite mengembalikan nama dan domain utama brand untuk membuat tautan di email.
func (r *Repository) BrandSite(ctx context.Context, brandID int64) (name, domain string, err error) {
	// Domain brand disimpan di brands.domain (dipakai ResolveDomain); brand_domains hanya cadangan lama.
	var primary sql.NullString
	err = r.db.QueryRowContext(ctx, `SELECT name, domain FROM brands WHERE id=?`, brandID).Scan(&name, &primary)
	if err != nil {
		return "", "", err
	}
	domain = strings.TrimSpace(primary.String)
	if domain == "" {
		_ = r.db.QueryRowContext(ctx, `SELECT domain FROM brand_domains WHERE brand_id=? ORDER BY is_primary DESC, id LIMIT 1`, brandID).Scan(&domain)
	}
	return name, domain, nil
}
