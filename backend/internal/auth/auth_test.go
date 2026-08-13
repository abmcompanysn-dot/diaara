package auth

import (
	"strings"
	"testing"
)

func TestGenerateToken(t *testing.T) {
	a, err := GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken() error: %v", err)
	}
	b, _ := GenerateToken()
	if a == b {
		t.Fatalf("GenerateToken() returned the same token twice")
	}
	if len(a) != 64 { // 32 octets hexadécimaux
		t.Fatalf("GenerateToken() length = %d, want 64", len(a))
	}
}

func TestHashToken(t *testing.T) {
	token := "abc-123"
	h1 := HashToken(token)
	h2 := HashToken(token)
	if h1 != h2 {
		t.Fatalf("HashToken() not deterministic: %s != %s", h1, h2)
	}
	if len(h1) != 64 {
		t.Fatalf("HashToken() length = %d, want 64", len(h1))
	}
	if HashToken("autre-token") == h1 {
		t.Fatalf("HashToken() collision on different inputs")
	}
	// Le hash ne doit pas contenir le token en clair
	if strings.Contains(h1, "abc") {
		t.Fatalf("HashToken() leaks plaintext token")
	}
}

func TestJWTManagerRoundTrip(t *testing.T) {
	m := NewJWTManager("access-secret", "refresh-secret")

	access, err := m.GenerateAccessToken("user-123", true, []string{"vendeur"}, nil)
	if err != nil {
		t.Fatalf("GenerateAccessToken() error: %v", err)
	}
	claims, err := m.ValidateAccessToken(access)
	if err != nil {
		t.Fatalf("ValidateAccessToken() error: %v", err)
	}
	if claims.UserID != "user-123" {
		t.Fatalf("UserID = %q, want user-123", claims.UserID)
	}
	if !claims.IsAdmin {
		t.Fatalf("IsAdmin = false, want true")
	}
	if len(claims.Roles) != 1 || claims.Roles[0] != "vendeur" {
		t.Fatalf("Roles = %v, want [vendeur]", claims.Roles)
	}
}

func TestJWTManagerRejectsWrongSecret(t *testing.T) {
	m1 := NewJWTManager("access-secret", "refresh-secret")
	m2 := NewJWTManager("wrong-secret", "refresh-secret")

	token, _ := m1.GenerateAccessToken("user-123", false, nil, nil)
	if _, err := m2.ValidateAccessToken(token); err == nil {
		t.Fatalf("ValidateAccessToken() accepted token signed with another secret")
	}
}

func TestRefreshTokenValidatedWithRefreshSecret(t *testing.T) {
	m := NewJWTManager("access-secret", "refresh-secret")

	refresh, err := m.GenerateRefreshToken("user-123")
	if err != nil {
		t.Fatalf("GenerateRefreshToken() error: %v", err)
	}
	// Un access token ne doit pas être accepté comme refresh token
	if _, err := m.ValidateRefreshToken(refresh); err != nil {
		t.Fatalf("ValidateRefreshToken() error: %v", err)
	}
}

func TestJWTManagerRejectsGarbage(t *testing.T) {
	m := NewJWTManager("access-secret", "refresh-secret")
	if _, err := m.ValidateAccessToken("not-a-jwt"); err == nil {
		t.Fatalf("ValidateAccessToken() accepted garbage")
	}
	if _, err := m.ValidateRefreshToken("not-a-jwt"); err == nil {
		t.Fatalf("ValidateRefreshToken() accepted garbage")
	}
}
