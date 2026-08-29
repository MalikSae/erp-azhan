package identity

import "testing"

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
