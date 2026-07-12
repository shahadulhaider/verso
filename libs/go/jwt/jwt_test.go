package jwt_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	libjwt "github.com/lestrrat-go/jwx/v2/jwt"

	versojwt "github.com/shahadulhaider/verso/libs/go/jwt"
)

// setupJWKS creates a test RSA key pair, signs a JWT, and serves the JWKS over HTTP.
func setupJWKS(t *testing.T, expired bool) (tokenStr string, jwksServer *httptest.Server) {
	t.Helper()

	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}

	jwkKey, err := jwk.FromRaw(privKey)
	if err != nil {
		t.Fatalf("jwk from raw: %v", err)
	}
	_ = jwkKey.Set(jwk.KeyIDKey, "test-key-1")
	_ = jwkKey.Set(jwk.AlgorithmKey, jwa.RS256)

	// Build a JWT
	expiry := time.Now().Add(time.Hour)
	if expired {
		expiry = time.Now().Add(-time.Hour)
	}
	token, err := libjwt.NewBuilder().
		Subject("user-123").
		Issuer("verso-identity-service").
		Expiration(expiry).
		Claim("user_id", "user-123").
		Claim("roles", []string{"reader", "admin"}).
		Build()
	if err != nil {
		t.Fatalf("build jwt: %v", err)
	}

	signed, err := libjwt.Sign(token, libjwt.WithKey(jwa.RS256, jwkKey))
	if err != nil {
		t.Fatalf("sign jwt: %v", err)
	}

	// Serve JWKS (public key set)
	pubKey, err := jwkKey.PublicKey()
	if err != nil {
		t.Fatalf("public key: %v", err)
	}
	set := jwk.NewSet()
	_ = set.AddKey(pubKey)

	jwksServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(set)
	}))

	return string(signed), jwksServer
}

func TestValidate_ValidToken(t *testing.T) {
	tokenStr, server := setupJWKS(t, false)
	defer server.Close()

	v := versojwt.NewJWKSValidator(server.URL, versojwt.WithRefreshInterval(time.Second))
	if err := v.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}

	claims, err := v.Validate(tokenStr)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if claims.UserID != "user-123" {
		t.Errorf("user_id = %q, want user-123", claims.UserID)
	}
	if claims.Issuer != "verso-identity-service" {
		t.Errorf("issuer = %q, want verso-identity-service", claims.Issuer)
	}
	if claims.Subject != "user-123" {
		t.Errorf("subject = %q, want user-123", claims.Subject)
	}
	if len(claims.Roles) != 2 {
		t.Errorf("roles length = %d, want 2", len(claims.Roles))
	}
}

func TestValidate_ExpiredToken(t *testing.T) {
	tokenStr, server := setupJWKS(t, true)
	defer server.Close()

	v := versojwt.NewJWKSValidator(server.URL, versojwt.WithRefreshInterval(time.Second))
	if err := v.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}

	_, err := v.Validate(tokenStr)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestMiddleware_NoBearer(t *testing.T) {
	tokenStr, server := setupJWKS(t, false)
	defer server.Close()
	_ = tokenStr

	v := versojwt.NewJWKSValidator(server.URL)
	if err := v.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}

	handler := versojwt.Middleware(v)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestMiddleware_ValidBearer(t *testing.T) {
	tokenStr, server := setupJWKS(t, false)
	defer server.Close()

	v := versojwt.NewJWKSValidator(server.URL, versojwt.WithRefreshInterval(time.Second))
	if err := v.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}

	var gotClaims *versojwt.Claims
	handler := versojwt.Middleware(v)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, ok := versojwt.ClaimsFromContext(r.Context())
		if !ok {
			t.Error("claims not in context")
		}
		gotClaims = c
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if gotClaims == nil || gotClaims.UserID != "user-123" {
		t.Error("expected claims with user_id=user-123 in context")
	}
}
