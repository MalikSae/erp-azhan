package identity

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type portalAccountReader interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func portalPINBinding(hash string) string {
	mac := hmac.New(sha256.New, getJWTSecret())
	mac.Write([]byte("portal-pin:" + hash))
	return hex.EncodeToString(mac.Sum(nil))
}

// GenerateVerifiedPortalToken binds login to the exact credential verified by
// bcrypt. A concurrent PIN reset must invalidate this token, not rebind it.
func GenerateVerifiedPortalToken(id int64, verifiedHash string) (string, error) {
	if verifiedHash == "" {
		return "", errors.New("PIN belum aktif")
	}
	return generateBoundPortalToken(id, portalPINBinding(verifiedHash))
}

// GenerateAccountPortalToken binds the session to the current credential.
func GenerateAccountPortalToken(ctx context.Context, q portalAccountReader, id int64) (string, error) {
	var hash string
	if err := q.QueryRowContext(ctx, `SELECT COALESCE(portal_pin_hash,'') FROM jamaah WHERE id=?`, id).Scan(&hash); err != nil {
		return "", err
	}
	return GenerateVerifiedPortalToken(id, hash)
}

// ValidateAccountPortalToken also protects optional authentication on public booking.
func ValidateAccountPortalToken(ctx context.Context, db *sql.DB, raw string) (int64, error) {
	id, err := ValidatePortalToken(raw)
	if err != nil {
		return 0, err
	}
	claims := jwt.MapClaims{}
	if _, _, err = jwt.NewParser().ParseUnverified(raw, claims); err != nil {
		return 0, err
	}
	binding, _ := claims["pin_binding"].(string)
	jti, _ := claims["jti"].(string)
	var hash string
	var revoked bool
	if err = db.QueryRowContext(ctx, `SELECT COALESCE(portal_pin_hash,''), EXISTS(SELECT 1 FROM portal_revoked_tokens WHERE jti=?) FROM jamaah WHERE id=?`, jti, id).Scan(&hash, &revoked); err != nil {
		return 0, err
	}
	if revoked || hash == "" || jti == "" || binding == "" || !hmac.Equal([]byte(binding), []byte(portalPINBinding(hash))) {
		return 0, errors.New("sesi tidak valid")
	}
	return id, nil
}

// RequirePortalSession runs after RequirePortalAuth. Old unbound tokens fail closed.
func RequirePortalSession(db *sql.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			claims := jwt.MapClaims{}
			// Signature/type/expiry were already checked by RequirePortalAuth.
			if _, _, err := jwt.NewParser().ParseUnverified(raw, claims); err != nil {
				writeError(w, 401, "sesi tidak valid")
				return
			}
			binding, _ := claims["pin_binding"].(string)
			jti, _ := claims["jti"].(string)
			var hash string
			var revoked bool
			err := db.QueryRowContext(r.Context(), `SELECT COALESCE(portal_pin_hash,''), EXISTS(SELECT 1 FROM portal_revoked_tokens WHERE jti=?) FROM jamaah WHERE id=?`, jti, GetPortalJamaahID(r.Context())).Scan(&hash, &revoked)
			if err != nil && err != sql.ErrNoRows {
				writeError(w, 503, "sesi belum dapat diperiksa, coba lagi")
				return
			}
			if err != nil || revoked || hash == "" || jti == "" || binding == "" || !hmac.Equal([]byte(binding), []byte(portalPINBinding(hash))) {
				writeError(w, 401, "sesi telah berakhir, silakan masuk kembali")
				return
			}
			w.Header().Set("Cache-Control", "private, no-store")
			next.ServeHTTP(w, r)
		})
	}
}

func PortalLogout(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := jwt.MapClaims{}
		raw := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if _, _, err := jwt.NewParser().ParseUnverified(raw, claims); err != nil {
			writeError(w, 401, "sesi tidak valid")
			return
		}
		jti, _ := claims["jti"].(string)
		exp, err := claims.GetExpirationTime()
		if jti == "" || err != nil || exp == nil {
			writeError(w, 401, "sesi tidak valid")
			return
		}
		if _, err = db.ExecContext(r.Context(), `INSERT IGNORE INTO portal_revoked_tokens(jti,expires_at) VALUES(?,?)`, jti, exp.Time); err != nil {
			writeError(w, 503, "gagal mengakhiri sesi, coba lagi")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
