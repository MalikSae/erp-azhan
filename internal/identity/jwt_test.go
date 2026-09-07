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
