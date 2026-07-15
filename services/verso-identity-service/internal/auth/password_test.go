package auth_test

import (
	"testing"

	"github.com/shahadulhaider/verso/services/verso-identity-service/internal/auth"
)

func TestHashAndVerifyPassword(t *testing.T) {
	password := "secure-p@ssw0rd!"

	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if hash == "" {
		t.Fatal("hash should not be empty")
	}
	if hash == password {
		t.Fatal("hash should differ from plaintext")
	}

	match, err := auth.VerifyPassword(password, hash)
	if err != nil {
		t.Fatalf("VerifyPassword: %v", err)
	}
	if !match {
		t.Fatal("correct password should match")
	}

	match, err = auth.VerifyPassword("wrong-password", hash)
	if err != nil {
		t.Fatalf("VerifyPassword (wrong): %v", err)
	}
	if match {
		t.Fatal("wrong password should not match")
	}
}

func TestHashUniqueness(t *testing.T) {
	h1, _ := auth.HashPassword("same-password")
	h2, _ := auth.HashPassword("same-password")
	if h1 == h2 {
		t.Fatal("two hashes of the same password should differ (unique salt)")
	}
}
