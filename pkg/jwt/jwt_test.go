package jwt

import (
	"testing"
	"time"
)

func TestGenerateAndParseToken(t *testing.T) {
	secret := "my-super-secret-key-1234567890"
	userID := uint64(42)
	tokenType := "access"
	expiry := 10 * time.Minute

	token, jti, err := GenerateToken(userID, tokenType, expiry, secret)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if token == "" {
		t.Fatal("Expected token string to be non-empty")
	}

	if jti == "" {
		t.Fatal("Expected JTI to be non-empty")
	}

	claims, err := ParseToken(token, secret)
	if err != nil {
		t.Fatalf("Expected no error parsing token, got %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("Expected UserID %d, got %d", userID, claims.UserID)
	}

	if claims.Type != tokenType {
		t.Errorf("Expected Type %s, got %s", tokenType, claims.Type)
	}

	if claims.ID != jti {
		t.Errorf("Expected JTI %s, got %s", jti, claims.ID)
	}
}

func TestExpiredToken(t *testing.T) {
	secret := "my-super-secret-key-1234567890"
	userID := uint64(42)
	tokenType := "access"
	expiry := -1 * time.Minute // expired in past

	token, _, err := GenerateToken(userID, tokenType, expiry, secret)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	_, err = ParseToken(token, secret)
	if err == nil {
		t.Fatal("Expected error parsing expired token, got nil")
	}

	if err != ErrExpiredToken {
		t.Errorf("Expected ErrExpiredToken, got %v", err)
	}
}

func TestTamperedToken(t *testing.T) {
	secret := "my-super-secret-key-1234567890"
	userID := uint64(42)
	tokenType := "access"
	expiry := 10 * time.Minute

	token, _, err := GenerateToken(userID, tokenType, expiry, secret)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Tamper with the token by changing the last character of signature
	tamperedToken := token[:len(token)-1] + "A"
	if tamperedToken == token {
		tamperedToken = token + "A"
	}

	_, err = ParseToken(tamperedToken, secret)
	if err == nil {
		t.Fatal("Expected error parsing tampered token, got nil")
	}

	if err != ErrInvalidToken {
		t.Errorf("Expected ErrInvalidToken, got %v", err)
	}
}

func TestInvalidSignature(t *testing.T) {
	secret := "my-super-secret-key-1234567890"
	wrongSecret := "wrong-secret-key-0987654321"
	userID := uint64(42)
	tokenType := "access"
	expiry := 10 * time.Minute

	token, _, err := GenerateToken(userID, tokenType, expiry, secret)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	_, err = ParseToken(token, wrongSecret)
	if err == nil {
		t.Fatal("Expected error parsing token with wrong secret, got nil")
	}

	if err != ErrInvalidToken {
		t.Errorf("Expected ErrInvalidToken, got %v", err)
	}
}
