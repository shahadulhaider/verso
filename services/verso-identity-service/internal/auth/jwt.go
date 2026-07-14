package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

const keyID = "verso-identity-1"

// TokenManager handles RSA key management, JWT signing, and JWKS serving.
type TokenManager struct {
	privateKey *rsa.PrivateKey
	publicSet  jwk.Set
	expiry     time.Duration
}

// NewTokenManager loads an RSA private key from keyPath, generating one if absent.
func NewTokenManager(keyPath string, expiry time.Duration) (*TokenManager, error) {
	privKey, err := loadOrGenerateKey(keyPath)
	if err != nil {
		return nil, fmt.Errorf("key setup: %w", err)
	}

	privJWK, err := jwk.FromRaw(privKey)
	if err != nil {
		return nil, fmt.Errorf("jwk from raw: %w", err)
	}
	_ = privJWK.Set(jwk.KeyIDKey, keyID)
	_ = privJWK.Set(jwk.AlgorithmKey, jwa.RS256)
	_ = privJWK.Set(jwk.KeyUsageKey, "sig")

	pubJWK, err := jwk.PublicKeyOf(privJWK)
	if err != nil {
		return nil, fmt.Errorf("jwk public key: %w", err)
	}

	set := jwk.NewSet()
	_ = set.AddKey(pubJWK)

	return &TokenManager{
		privateKey: privKey,
		publicSet:  set,
		expiry:     expiry,
	}, nil
}

// SignAccessToken creates a signed RS256 JWT with the given claims.
func (tm *TokenManager) SignAccessToken(userID, email string, roles []string) (string, error) {
	now := time.Now()
	token := jwt.New()

	for k, v := range map[string]interface{}{
		jwt.SubjectKey:    userID,
		jwt.IssuedAtKey:   now,
		jwt.ExpirationKey: now.Add(tm.expiry),
		"email":           email,
		"roles":           roles,
	} {
		if err := token.Set(k, v); err != nil {
			return "", fmt.Errorf("set claim %s: %w", k, err)
		}
	}

	signed, err := jwt.Sign(token, jwt.WithKey(jwa.RS256, tm.privateKey))
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return string(signed), nil
}

// VerifyAccessToken parses and validates a signed JWT, returning its claims.
func (tm *TokenManager) VerifyAccessToken(tokenString string) (jwt.Token, error) {
	return jwt.Parse(
		[]byte(tokenString),
		jwt.WithKey(jwa.RS256, &tm.privateKey.PublicKey),
		jwt.WithValidate(true),
	)
}

// WriteJWKS writes the public JWKS as JSON to the response writer.
func (tm *TokenManager) WriteJWKS(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	json.NewEncoder(w).Encode(tm.publicSet)
}

func loadOrGenerateKey(keyPath string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(keyPath)
	if err == nil {
		return parsePrivateKeyPEM(data)
	}
	if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read key file: %w", err)
	}

	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("generate rsa key: %w", err)
	}

	if err := savePrivateKeyPEM(keyPath, privKey); err != nil {
		return nil, fmt.Errorf("save key: %w", err)
	}
	return privKey, nil
}

func parsePrivateKeyPEM(data []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("no PEM block found")
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

func savePrivateKeyPEM(path string, key *rsa.PrivateKey) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	return pem.Encode(f, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
}
