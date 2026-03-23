package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "test-jwt-secret-at-least-32-bytes-long"

func TestIssueToken_and_ParseAndValidate_roundtrip(t *testing.T) {
	userID := "user-123"
	role := "admin"
	secret := []byte(testSecret)
	expiry := 1 * time.Hour

	token, err := IssueToken(userID, role, secret, expiry)

	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	claims, err := ParseAndValidate(token, secret)

	if err != nil {
		t.Fatalf("ParseAndValidate: %v", err)
	}
	if claims.Sub != userID || claims.Role != role {
		t.Errorf("expected sub=%q role=%q, got sub=%q role=%q", userID, role, claims.Sub, claims.Role)
	}
}

func TestParseAndValidate_wrongSecretReturnsError(t *testing.T) {
	token, _ := IssueToken("user-1", "viewer", []byte(testSecret), time.Hour)

	_, err := ParseAndValidate(token, []byte("wrong-secret"))

	if err == nil {
		t.Fatal("expected error for wrong secret")
	}
}

func TestParseAndValidate_tamperedTokenReturnsError(t *testing.T) {
	token, _ := IssueToken("user-1", "viewer", []byte(testSecret), time.Hour)
	tampered := token[:len(token)-2] + "xx"

	_, err := ParseAndValidate(tampered, []byte(testSecret))

	if err == nil {
		t.Fatal("expected error for tampered token")
	}
}

func TestParseAndValidate_emptyTokenReturnsError(t *testing.T) {
	_, err := ParseAndValidate("", []byte(testSecret))

	if err == nil {
		t.Fatal("expected error for empty token")
	}
}

func TestIssueToken_zeroExpiryUsesDefault(t *testing.T) {
	token, err := IssueToken("u", "admin", []byte(testSecret), 0)

	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}
	claims, err := ParseAndValidate(token, []byte(testSecret))
	if err != nil {
		t.Fatalf("ParseAndValidate: %v", err)
	}
	if claims.Sub != "u" {
		t.Errorf("expected sub=u, got %q", claims.Sub)
	}
}

func TestParseAndValidate_rejectsUnexpectedSigningMethod(t *testing.T) {
	claims := jwtClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "u1",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    tokenIssuer,
		},
		Role: "admin",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS384, claims)
	tokenString, err := token.SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("SignedString: %v", err)
	}

	_, err = ParseAndValidate(tokenString, []byte(testSecret))

	if err == nil {
		t.Fatal("expected error for unexpected signing method")
	}
}

func TestParseAndValidate_rejectsEmptySubject(t *testing.T) {
	token, err := IssueToken("", "admin", []byte(testSecret), time.Hour)
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}

	_, err = ParseAndValidate(token, []byte(testSecret))

	if err == nil {
		t.Fatal("expected error for empty subject claim")
	}
}

func TestParseAndValidate_rejectsEmptyRole(t *testing.T) {
	token, err := IssueToken("user-1", "", []byte(testSecret), time.Hour)
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}

	_, err = ParseAndValidate(token, []byte(testSecret))

	if err == nil {
		t.Fatal("expected error for empty role claim")
	}
}
