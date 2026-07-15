// Package repository provides database access for the identity service.
package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrDuplicateEmail is returned when an account with the same email already exists.
var ErrDuplicateEmail = errors.New("duplicate email")

// ErrNotFound is returned when the requested record does not exist.
var ErrNotFound = errors.New("not found")

// Account represents a row in identity.account.
type Account struct {
	ID            string
	Email         string
	EmailVerified bool
	PasswordHash  *string
	Status        string
	Roles         []string
	DisplayName   string
	MFAEnabled    bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// RefreshToken represents a row in identity.refresh_tokens.
type RefreshToken struct {
	ID        string
	AccountID string
	TokenHash string
	ExpiresAt time.Time
	CreatedAt time.Time
}

// Repo provides database operations for the identity domain.
type Repo struct {
	pool *pgxpool.Pool
}

// New creates a repository backed by the given connection pool.
func New(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

// Pool returns the underlying connection pool (for readiness checks).
func (r *Repo) Pool() *pgxpool.Pool {
	return r.pool
}

// CreateAccount inserts a new account within the given transaction.
func (r *Repo) CreateAccount(ctx context.Context, tx pgx.Tx, a *Account) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO account (id, email, email_verified, password_hash, status, roles, display_name, mfa_enabled, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		a.ID, a.Email, a.EmailVerified, a.PasswordHash, a.Status, a.Roles,
		a.DisplayName, a.MFAEnabled, a.CreatedAt, a.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrDuplicateEmail
		}
		return err
	}
	return nil
}

// GetByEmail retrieves an account by email address.
func (r *Repo) GetByEmail(ctx context.Context, email string) (*Account, error) {
	a := &Account{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, email, email_verified, password_hash, status, roles, display_name, mfa_enabled, created_at, updated_at
		 FROM account WHERE email = $1`,
		email,
	).Scan(&a.ID, &a.Email, &a.EmailVerified, &a.PasswordHash, &a.Status, &a.Roles,
		&a.DisplayName, &a.MFAEnabled, &a.CreatedAt, &a.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return a, err
}

// StoreRefreshToken inserts a refresh token record within the given transaction.
func (r *Repo) StoreRefreshToken(ctx context.Context, tx pgx.Tx, rt *RefreshToken) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO refresh_tokens (id, account_id, token_hash, expires_at, created_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		rt.ID, rt.AccountID, rt.TokenHash, rt.ExpiresAt, rt.CreatedAt,
	)
	return err
}

// GetRefreshToken retrieves a valid (non-expired) refresh token by its hash.
func (r *Repo) GetRefreshToken(ctx context.Context, tokenHash string) (*RefreshToken, error) {
	rt := &RefreshToken{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, account_id, token_hash, expires_at, created_at
		 FROM refresh_tokens WHERE token_hash = $1 AND expires_at > NOW()`,
		tokenHash,
	).Scan(&rt.ID, &rt.AccountID, &rt.TokenHash, &rt.ExpiresAt, &rt.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return rt, err
}

// GetAccountByID retrieves an account by its ULID.
func (r *Repo) GetAccountByID(ctx context.Context, id string) (*Account, error) {
	a := &Account{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, email, email_verified, password_hash, status, roles, display_name, mfa_enabled, created_at, updated_at
		 FROM account WHERE id = $1`,
		id,
	).Scan(&a.ID, &a.Email, &a.EmailVerified, &a.PasswordHash, &a.Status, &a.Roles,
		&a.DisplayName, &a.MFAEnabled, &a.CreatedAt, &a.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return a, err
}

// DeleteRefreshToken removes a refresh token by its hash.
func (r *Repo) DeleteRefreshToken(ctx context.Context, tokenHash string) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM refresh_tokens WHERE token_hash = $1`, tokenHash)
	return err
}

// BeginTx starts a new database transaction.
func (r *Repo) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return r.pool.Begin(ctx)
}

// HashRefreshToken returns a SHA-256 hex digest of the raw token string.
func HashRefreshToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
