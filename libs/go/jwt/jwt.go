// Package jwt provides JWKS-based JWT validation and HTTP middleware.
package jwt

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

type claimsKey struct{}

// Claims holds the validated JWT claims injected into context by the middleware.
type Claims struct {
	UserID    string
	Roles     []string
	Subject   string
	Issuer    string
	ExpiresAt time.Time
}

// Option configures a JWKSValidator.
type Option func(*JWKSValidator)

// WithRefreshInterval sets how often the JWKS cache is refreshed.
func WithRefreshInterval(d time.Duration) Option {
	return func(v *JWKSValidator) { v.refreshInterval = d }
}

// JWKSValidator validates JWTs against a remote JWKS endpoint.
type JWKSValidator struct {
	jwksURL         string
	cache           *jwk.Cache
	refreshInterval time.Duration
}

// NewJWKSValidator creates a validator that fetches and caches keys from jwksURL.
func NewJWKSValidator(jwksURL string, opts ...Option) *JWKSValidator {
	v := &JWKSValidator{
		jwksURL:         jwksURL,
		refreshInterval: 15 * time.Minute,
	}
	for _, o := range opts {
		o(v)
	}
	return v
}

// Start initializes the JWKS cache. Must be called before Validate.
func (v *JWKSValidator) Start(ctx context.Context) error {
	cache := jwk.NewCache(ctx)
	if err := cache.Register(v.jwksURL, jwk.WithMinRefreshInterval(v.refreshInterval)); err != nil {
		return fmt.Errorf("jwks register: %w", err)
	}
	// Fetch once eagerly to fail fast on bad URL.
	if _, err := cache.Refresh(ctx, v.jwksURL); err != nil {
		return fmt.Errorf("jwks initial fetch: %w", err)
	}
	v.cache = cache
	return nil
}

// Validate parses and verifies a JWT string against the cached JWKS.
func (v *JWKSValidator) Validate(tokenString string) (*Claims, error) {
	keyset, err := v.cache.Get(context.Background(), v.jwksURL)
	if err != nil {
		return nil, fmt.Errorf("jwks get: %w", err)
	}

	token, err := jwt.Parse([]byte(tokenString), jwt.WithKeySet(keyset))
	if err != nil {
		return nil, fmt.Errorf("jwt parse: %w", err)
	}

	claims := &Claims{
		Subject:   token.Subject(),
		Issuer:    token.Issuer(),
		ExpiresAt: token.Expiration(),
	}

	if uid, ok := token.Get("user_id"); ok {
		claims.UserID, _ = uid.(string)
	}
	if claims.UserID == "" {
		claims.UserID = token.Subject()
	}

	if roles, ok := token.Get("roles"); ok {
		if rs, ok := roles.([]interface{}); ok {
			for _, r := range rs {
				if s, ok := r.(string); ok {
					claims.Roles = append(claims.Roles, s)
				}
			}
		}
	}

	return claims, nil
}

// ClaimsFromContext extracts validated Claims from the request context.
func ClaimsFromContext(ctx context.Context) (*Claims, bool) {
	c, ok := ctx.Value(claimsKey{}).(*Claims)
	return c, ok
}

// NewContext returns a context carrying the given Claims (useful in tests).
func NewContext(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, claimsKey{}, claims)
}

// Middleware returns an HTTP middleware that validates Bearer tokens via the JWKSValidator.
func Middleware(v *JWKSValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if !strings.HasPrefix(auth, "Bearer ") {
				http.Error(w, `{"error":"missing bearer token"}`, http.StatusUnauthorized)
				return
			}

			claims, err := v.Validate(strings.TrimPrefix(auth, "Bearer "))
			if err != nil {
				http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), claimsKey{}, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
