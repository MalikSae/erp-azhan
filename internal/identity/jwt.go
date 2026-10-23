package identity

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func getJWTSecret() []byte {
	return []byte(os.Getenv("JWT_SECRET"))
}

// jwtIssuer diisi pada setiap token baru. Token lama tanpa "iss" tetap diterima sampai
// kedaluwarsa (access maks. 60 menit, refresh maks. JWT_REFRESH_TTL_DAYS); token dengan "iss"
// lain ditolak. Setelah seluruh token lama habis, "iss" dapat diwajibkan.
const jwtIssuer = "erp-azhan-api"

// Placeholder yang dikenal (dibandingkan setelah dipangkas, tanpa membedakan huruf besar/kecil).
var placeholderJWTSecrets = []string{
	"your_super_secret_jwt_key_here_minimum_32_chars",
	"changeme", "change_me", "change-me", "secret", "jwt_secret", "your_jwt_secret",
	"supersecret", "super_secret", "please_change_me", "replace_me",
}

// ValidateJWTSecret dijalankan SELALU saat startup (semua APP_ENV): JWT_SECRET wajib terisi,
// minimal 32 karakter, dan bukan placeholder/contoh. Tes mengatur secret sendiri lewat t.Setenv.
func ValidateJWTSecret() error {
	s := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if s == "" {
		return errors.New("JWT_SECRET wajib diisi")
	}
	if len(s) < 32 {
		return errors.New("JWT_SECRET harus berisi minimal 32 karakter")
	}
	lower := strings.ToLower(s)
	for _, p := range placeholderJWTSecrets {
		if lower == p {
			return errors.New("JWT_SECRET masih berisi nilai contoh; buat secret acak (mis. openssl rand -hex 32)")
		}
	}
	if strings.HasPrefix(lower, "your_") || strings.HasPrefix(lower, "changeme") || strings.Contains(lower, "your_super_secret") {
		return errors.New("JWT_SECRET masih berisi nilai contoh; buat secret acak (mis. openssl rand -hex 32)")
	}
	return nil
}

// maxAccessTTLMinutes membatasi umur access token: pencabutan akses yang tidak tertangkap cek
// status RequireAuth tetap berakhir paling lambat dalam rentang ini.
const maxAccessTTLMinutes = 60

func getAccessTTL() time.Duration {
	mins, _ := strconv.Atoi(os.Getenv("JWT_ACCESS_TTL_MINUTES"))
	if mins <= 0 {
		mins = 15 // fallback
	}
	if mins > maxAccessTTLMinutes {
		mins = maxAccessTTLMinutes
	}
	return time.Duration(mins) * time.Minute
}

// parseHS256 memverifikasi token hanya dengan algoritma HS256 (alg lain, termasuk "none" dan
// HS384/HS512, ditolak) dan menolak "iss" yang bukan milik API ini.
func parseHS256(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return getJWTSecret(), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return nil, fmt.Errorf("token tidak valid atau kedaluwarsa: %w", err)
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, errors.New("token tidak valid")
	}
	if iss, ada := claims["iss"]; ada && iss != jwtIssuer {
		return nil, errors.New("issuer token tidak valid")
	}
	return claims, nil
}

func getRefreshTTL() time.Duration {
	days, _ := strconv.Atoi(os.Getenv("JWT_REFRESH_TTL_DAYS"))
	if days <= 0 {
		days = 7 // fallback
	}
	return time.Duration(days) * 24 * time.Hour
}

// AccessClaims klaim access token hasil parse. Roles/Perms/PermVersion diisi
// sejak RBAC Fase 1; token lama tanpa klaim tersebut tetap valid (slice kosong).
type AccessClaims struct {
	AdminUserID int64
	BrandID     *int64
	Role        string
	Roles       []string
	Perms       []string
	PermVersion int
}

// GenerateAccessToken membuat token access baru.
// Email dibuat variadic agar pemanggil internal lama tetap kompatibel.
func GenerateAccessToken(adminUserID int64, brandID *int64, role string, emails ...string) (string, error) {
	email := ""
	if len(emails) > 0 {
		email = emails[0]
	}
	return GenerateAccessTokenWithAccess(adminUserID, brandID, role, email, nil, nil, 0)
}

// GenerateAccessTokenWithAccess membuat access token lengkap dengan klaim RBAC
// (roles, perms, pver). Dipakai login/refresh setelah resolusi rbac.GetUserAccess.
func GenerateAccessTokenWithAccess(adminUserID int64, brandID *int64, role, email string, roles, perms []string, permVersion int) (string, error) {
	now := time.Now()
	ttl := getAccessTTL()
	claims := jwt.MapClaims{
		"sub":      adminUserID,
		"brand_id": brandID,
		"role":     role,
		"type":     "access",
		"iss":      jwtIssuer,
		"jti":      uuid.New().String(),
		"exp":      now.Add(ttl).Unix(),
		"iat":      now.Unix(),
	}
	if email != "" {
		claims["email"] = email
	}
	if len(roles) > 0 {
		claims["roles"] = roles
	}
	if len(perms) > 0 {
		claims["perms"] = perms
	}
	if permVersion > 0 {
		claims["pver"] = permVersion
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(getJWTSecret())
}

// GenerateRefreshToken membuat token refresh baru. jti dan expiresAt
// dikembalikan agar pemanggil bisa mencatatnya di admin_refresh_tokens.
func GenerateRefreshToken(adminUserID int64) (token string, jti string, expiresAt time.Time, err error) {
	now := time.Now()
	jti = uuid.New().String()
	expiresAt = now.Add(getRefreshTTL())
	claims := jwt.MapClaims{
		"sub":  adminUserID,
		"type": "refresh",
		"iss":  jwtIssuer,
		"jti":  jti,
		"exp":  expiresAt.Unix(),
		"iat":  now.Unix(),
	}
	token, err = jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(getJWTSecret())
	return token, jti, expiresAt, err
}

// ParseRefreshToken memvalidasi signature, exp, dan type refresh token, lalu
// mengembalikan admin user ID dan jti-nya.
func ParseRefreshToken(tokenString string) (int64, string, error) {
	adminUserID, _, _, err := ValidateToken(tokenString, "refresh")
	if err != nil {
		return 0, "", err
	}
	claims, err := parseHS256(tokenString)
	if err != nil {
		return 0, "", fmt.Errorf("refresh token tidak valid: %w", err)
	}
	jti, _ := claims["jti"].(string)
	if jti == "" {
		return 0, "", errors.New("jti claim tidak valid")
	}
	return adminUserID, jti, nil
}

// ValidateToken memvalidasi signature, exp, dan type token.
func ValidateToken(tokenString string, expectedType string) (int64, *int64, string, error) {
	claims, err := parseHS256(tokenString)
	if err != nil {
		return 0, nil, "", err
	}

	tokenType, ok := claims["type"].(string)
	if !ok || tokenType != expectedType {
		return 0, nil, "", errors.New("tipe token salah")
	}

	subFloat, ok := claims["sub"].(float64)
	if !ok {
		return 0, nil, "", errors.New("sub claim tidak valid")
	}

	var parsedBrandID *int64
	if brandIDClaim, ok := claims["brand_id"]; ok && brandIDClaim != nil {
		if brandIDFloat, ok := brandIDClaim.(float64); ok {
			bid := int64(brandIDFloat)
			parsedBrandID = &bid
		}
	}

	role, _ := claims["role"].(string)
	return int64(subFloat), parsedBrandID, role, nil
}

// ValidateAccessClaims memvalidasi access token dan mengembalikan seluruh
// klaimnya, termasuk klaim RBAC. Token lama tanpa roles/perms tetap valid.
func ValidateAccessClaims(tokenString string) (*AccessClaims, error) {
	claims, err := parseHS256(tokenString)
	if err != nil {
		return nil, err
	}
	if tokenType, ok := claims["type"].(string); !ok || tokenType != "access" {
		return nil, errors.New("tipe token salah")
	}
	subFloat, ok := claims["sub"].(float64)
	if !ok {
		return nil, errors.New("sub claim tidak valid")
	}

	out := &AccessClaims{AdminUserID: int64(subFloat)}
	if brandIDClaim, ok := claims["brand_id"]; ok && brandIDClaim != nil {
		if brandIDFloat, ok := brandIDClaim.(float64); ok {
			bid := int64(brandIDFloat)
			out.BrandID = &bid
		}
	}
	out.Role, _ = claims["role"].(string)
	out.Roles = claimStringSlice(claims["roles"])
	out.Perms = claimStringSlice(claims["perms"])
	if pver, ok := claims["pver"].(float64); ok {
		out.PermVersion = int(pver)
	}
	return out, nil
}

func claimStringSlice(v any) []string {
	raw, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}
