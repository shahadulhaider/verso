package auth_test

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/shahadulhaider/verso/services/verso-identity-service/internal/auth"
)

func newTestTokenManager(t *testing.T) *auth.TokenManager {
	t.Helper()
	keyPath := filepath.Join(t.TempDir(), "test-key.pem")
	tm, err := auth.NewTokenManager(keyPath, time.Hour)
	if err != nil {
		t.Fatalf("NewTokenManager: %v", err)
	}
	return tm
}

func TestSignAndVerifyAccessToken(t *testing.T) {
	tm := newTestTokenManager(t)

	tokenStr, err := tm.SignAccessToken("USER123", "test@example.com", []string{"reader", "author"})
	if err != nil {
		t.Fatalf("SignAccessToken: %v", err)
	}
	if tokenStr == "" {
		t.Fatal("token should not be empty")
	}

	parsed, err := tm.VerifyAccessToken(tokenStr)
	if err != nil {
		t.Fatalf("VerifyAccessToken: %v", err)
	}

	if parsed.Subject() != "USER123" {
		t.Errorf("subject: got %q, want %q", parsed.Subject(), "USER123")
	}

	emailVal, ok := parsed.Get("email")
	if !ok {
		t.Fatal("email claim missing")
	}
	if emailVal.(string) != "test@example.com" {
		t.Errorf("email: got %q, want %q", emailVal, "test@example.com")
	}
}

func TestTokenExpiry(t *testing.T) {
	keyPath := filepath.Join(t.TempDir(), "test-key.pem")
	tm, err := auth.NewTokenManager(keyPath, -time.Hour) // already expired
	if err != nil {
		t.Fatalf("NewTokenManager: %v", err)
	}

	tokenStr, err := tm.SignAccessToken("USER123", "test@example.com", nil)
	if err != nil {
		t.Fatalf("SignAccessToken: %v", err)
	}

	_, err = tm.VerifyAccessToken(tokenStr)
	if err == nil {
		t.Fatal("expired token should fail verification")
	}
}

func TestKeyPersistence(t *testing.T) {
	keyPath := filepath.Join(t.TempDir(), "persist-key.pem")

	tm1, err := auth.NewTokenManager(keyPath, time.Hour)
	if err != nil {
		t.Fatalf("first init: %v", err)
	}

	token1, err := tm1.SignAccessToken("USER1", "a@b.com", nil)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	// Second manager loads the same key
	tm2, err := auth.NewTokenManager(keyPath, time.Hour)
	if err != nil {
		t.Fatalf("second init: %v", err)
	}

	// Token signed by first manager should verify with second
	_, err = tm2.VerifyAccessToken(token1)
	if err != nil {
		t.Fatalf("cross-verify: %v", err)
	}

	// Key file should exist
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		t.Fatal("key file should persist on disk")
	}
}

func TestJWKSOutput(t *testing.T) {
	tm := newTestTokenManager(t)

	w := httptest.NewRecorder()
	tm.WriteJWKS(w)

	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("content-type: got %q", ct)
	}

	var jwks struct {
		Keys []json.RawMessage `json:"keys"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &jwks); err != nil {
		t.Fatalf("unmarshal JWKS: %v", err)
	}
	if len(jwks.Keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(jwks.Keys))
	}

	// Verify key contains expected fields
	var keyData map[string]interface{}
	json.Unmarshal(jwks.Keys[0], &keyData)

	if keyData["kty"] != "RSA" {
		t.Errorf("kty: got %v, want RSA", keyData["kty"])
	}
	if keyData["use"] != "sig" {
		t.Errorf("use: got %v, want sig", keyData["use"])
	}
	if keyData["alg"] != "RS256" {
		t.Errorf("alg: got %v, want RS256", keyData["alg"])
	}
	if _, ok := keyData["n"]; !ok {
		t.Error("missing RSA modulus 'n'")
	}
	if _, ok := keyData["d"]; ok {
		t.Error("JWKS should NOT contain private key component 'd'")
	}
}
