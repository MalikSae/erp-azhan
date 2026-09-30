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

// ValidateJWTSecret enforces a cryptographically meaningful secret in
// production. It is intentionally separate from token generation so startup
// can fail fast before serving requests.
func ValidateJWTSecret() error {
	if len(strings.TrimSpace(os.Getenv("JWT_SECRET"))) < 32 {
		return errors.New("JWT_SECRET harus berisi minimal 32 karakter")
	}
	return nil
}

func getAccessTTL() time.Duration {
	mins, _ := strconv.Atoi(os.Getenv("JWT_ACCESS_TTL_MINUTES"))
	if mins <= 0 {
		mins = 15 // fallback
	}
	return time.Duration(mins) * time.Minute
}

func getRefreshTTL() time.Duration {
	days, _ := strconv.Atoi(os.Getenv("JWT_REFRESH_TTL_DAYS"))
	if days <= 0 {
		days = 7 // fallback
	}
	return time.Duration(days) * 24 * time.Hour
}

// GenerateAccessToken membuat token access baru.
// Email dibuat variadic agar pemanggil internal lama tetap kompatibel.
func GenerateAccessToken(adminUserID int64, brandID *int64, role string, emails ...string) (string, error) {
	now := time.Now()
	ttl := getAccessTTL()
	claims := jwt.MapClaims{
		"sub":      adminUserID,
		"brand_id": brandID,
		"role":     role,
		"type":     "access",
		"jti":      uuid.New().String(),
		"exp":      now.Add(ttl).Unix(),
		"iat":      now.Unix(),
	}
	if len(emails) > 0 && emails[0] != "" {
		claims["email"] = emails[0]
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
	claims := jwt.MapClaims{}
	if _, _, err := jwt.NewParser().ParseUnverified(tokenString, claims); err != nil {
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
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("metode signing tidak valid: %v", token.Header["alg"])
		}
		return getJWTSecret(), nil
	})

	if err != nil {
		return 0, nil, "", fmt.Errorf("token tidak valid atau kedaluwarsa: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return 0, nil, "", errors.New("token tidak valid")
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

// GeneratePortalToken membuat token autentikasi khusus Portal Jamaah (24 jam).
func GeneratePortalToken(jamaahID int64) (string, error) {
	return generateBoundPortalToken(jamaahID, "")
}

func generateBoundPortalToken(jamaahID int64, binding string) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":         jamaahID,
		"type":        "portal",
		"jti":         uuid.New().String(),
		"exp":         now.Add(24 * time.Hour).Unix(),
		"iat":         now.Unix(),
		"pin_binding": binding,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(getJWTSecret())
}

// ValidatePortalToken memvalidasi portal token dan mengembalikan jamaahID.
func ValidatePortalToken(tokenString string) (int64, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("metode signing tidak valid: %v", token.Header["alg"])
		}
		return getJWTSecret(), nil
	})

	if err != nil {
		return 0, fmt.Errorf("token tidak valid atau kedaluwarsa: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return 0, errors.New("token tidak valid")
	}

	tokenType, ok := claims["type"].(string)
	if !ok || tokenType != "portal" {
		return 0, errors.New("tipe token salah")
	}

	subFloat, ok := claims["sub"].(float64)
	if !ok {
		return 0, errors.New("sub claim tidak valid")
	}

	return int64(subFloat), nil
}
