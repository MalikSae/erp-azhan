package identity

import (
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

func TestAccessTokenCarriesCRMRole(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-minimum-32-characters-long")
	brandID := int64(7)
	token, err := GenerateAccessToken(42, &brandID, "cs")
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}
	userID, parsedBrandID, role, err := ValidateToken(token, "access")
	if err != nil {
		t.Fatalf("ValidateToken: %v", err)
	}
	if userID != 42 || parsedBrandID == nil || *parsedBrandID != 7 || role != "cs" {
		t.Fatalf("claims tidak sesuai: user=%d brand=%v role=%s", userID, parsedBrandID, role)
	}
}

func TestRefreshTokenRoundTrip(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-minimum-32-characters-long")
	token, jti, expiresAt, err := GenerateRefreshToken(42)
	if err != nil {
		t.Fatalf("GenerateRefreshToken: %v", err)
	}
	if jti == "" || expiresAt.IsZero() {
		t.Fatalf("jti/expiresAt kosong: %q %v", jti, expiresAt)
	}
	userID, parsedJTI, err := ParseRefreshToken(token)
	if err != nil {
		t.Fatalf("ParseRefreshToken: %v", err)
	}
	if userID != 42 || parsedJTI != jti {
		t.Fatalf("got user=%d jti=%q, want 42 %q", userID, parsedJTI, jti)
	}

	access, _ := GenerateAccessToken(42, nil, "admin")
	if _, _, err := ParseRefreshToken(access); err == nil {
		t.Fatal("access token diterima sebagai refresh token")
	}
	if _, _, err := ParseRefreshToken(token + "x"); err == nil {
		t.Fatal("refresh token dengan signature rusak diterima")
	}
}

func TestAccessTokenCarriesAccountEmail(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-minimum-32-characters-long")
	brandID := int64(7)
	tokenString, err := GenerateAccessToken(42, &brandID, "admin", "admin2@hana.id")
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		return getJWTSecret(), nil
	})
	if err != nil || !token.Valid {
		t.Fatalf("token tidak valid: %v", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || claims["email"] != "admin2@hana.id" {
		t.Fatalf("email claim tidak sesuai: %v", claims["email"])
	}
}
