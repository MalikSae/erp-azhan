package identity

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func getJWTSecret() []byte {
	return []byte(os.Getenv("JWT_SECRET"))
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

// GenerateAccessToken membuat token access baru
func GenerateAccessToken(adminUserID int64, brandID *int64, role string) (string, error) {
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
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(getJWTSecret())
}

// GenerateRefreshToken membuat token refresh baru
func GenerateRefreshToken(adminUserID int64) (string, error) {
	now := time.Now()
	ttl := getRefreshTTL()
	claims := jwt.MapClaims{
		"sub":  adminUserID,
		"type": "refresh",
		"jti":  uuid.New().String(),
		"exp":  now.Add(ttl).Unix(),
		"iat":  now.Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(getJWTSecret())
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
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":  jamaahID,
		"type": "portal",
		"jti":  uuid.New().String(),
		"exp":  now.Add(24 * time.Hour).Unix(),
		"iat":  now.Unix(),
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
