package auth

import (
	"testing"
	"time"
)

func TestHashAndVerifyPassword(t *testing.T) {
	hash, err := HashPassword("correct-horse-battery-staple")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}

	ok, err := VerifyPassword("correct-horse-battery-staple", hash)
	if err != nil {
		t.Fatalf("VerifyPassword: %v", err)
	}
	if !ok {
		t.Fatal("expected correct password to verify")
	}

	ok, err = VerifyPassword("wrong-password", hash)
	if err != nil {
		t.Fatalf("VerifyPassword: %v", err)
	}
	if ok {
		t.Fatal("expected wrong password to fail verification")
	}
}

func TestHashPasswordProducesDifferentSalts(t *testing.T) {
	h1, _ := HashPassword("same-password")
	h2, _ := HashPassword("same-password")
	if h1 == h2 {
		t.Fatal("expected different hashes for the same password due to random salt")
	}
}

func TestGenerateAndParseToken(t *testing.T) {
	secret := "top-secret"
	token, err := GenerateToken("user-123", "user@example.com", secret, time.Hour)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	claims, err := ParseToken(token, secret)
	if err != nil {
		t.Fatalf("ParseToken: %v", err)
	}
	if claims.UserID != "user-123" {
		t.Fatalf("expected UserID user-123, got %s", claims.UserID)
	}
	if claims.Email != "user@example.com" {
		t.Fatalf("expected email user@example.com, got %s", claims.Email)
	}
}

func TestParseTokenRejectsWrongSecret(t *testing.T) {
	token, _ := GenerateToken("user-123", "user@example.com", "secret-a", time.Hour)
	if _, err := ParseToken(token, "secret-b"); err == nil {
		t.Fatal("expected error when parsing with the wrong secret")
	}
}

func TestParseTokenRejectsExpiredToken(t *testing.T) {
	secret := "top-secret"
	token, _ := GenerateToken("user-123", "user@example.com", secret, -time.Hour)
	if _, err := ParseToken(token, secret); err != ErrExpiredToken {
		t.Fatalf("expected ErrExpiredToken, got %v", err)
	}
}
