package auth

import (
	"testing"
)

func TestHashPassword_returnsNonEmptyHash(t *testing.T) {
	password := "secret"

	hash, err := HashPassword(password)

	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if hash == "" || hash == password {
		t.Errorf("expected non-empty hash different from password, got %q", hash)
	}
}

func TestHashPassword_differentCallsProduceDifferentHashes(t *testing.T) {
	password := "secret"

	hash1, _ := HashPassword(password)
	hash2, _ := HashPassword(password)

	if hash1 == hash2 {
		t.Error("expected different hashes per call (salt), got same")
	}
}

func TestPasswordMatches_matchingPasswordReturnsTrue(t *testing.T) {
	password := "secret"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}

	result := PasswordMatches(hash, password)

	if !result {
		t.Error("expected true for matching password")
	}
}

func TestPasswordMatches_wrongPasswordReturnsFalse(t *testing.T) {
	hash, _ := HashPassword("secret")

	result := PasswordMatches(hash, "wrong")

	if result {
		t.Error("expected false for wrong password")
	}
}

func TestPasswordMatches_emptyHashReturnsFalse(t *testing.T) {
	result := PasswordMatches("", "anything")

	if result {
		t.Error("expected false for empty hash")
	}
}
