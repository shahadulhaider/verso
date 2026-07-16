// Package repository provides database access for the profile service.
package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Sentinel errors for the profile repository.
var (
	ErrNotFound          = errors.New("not found")
	ErrDuplicateUsername = errors.New("duplicate username")
	ErrDuplicateProfile  = errors.New("duplicate profile")
)

// Profile represents a row in profile.user_profile.
type Profile struct {
	ID                string
	Username          string
	DisplayName       string
	Bio               *string
	AvatarURL         *string
	Location          *string
	WebsiteURL        *string
	IsAuthor          bool
	IsPublisher       bool
	IsVerifiedCritic  bool
	PrivacyLevel      string
	ReadingGoalAnnual *int
	PreferredLanguage string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// Repo provides database operations for the profile domain.
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

// BeginTx starts a new database transaction.
func (r *Repo) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return r.pool.Begin(ctx)
}

// Create inserts a new profile within the given transaction.
func (r *Repo) Create(ctx context.Context, tx pgx.Tx, p *Profile) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO user_profile (id, username, display_name, bio, avatar_url, location, website_url,
			is_author, is_publisher, is_verified_critic, privacy_level, reading_goal_annual,
			preferred_language, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
		p.ID, p.Username, p.DisplayName, p.Bio, p.AvatarURL, p.Location, p.WebsiteURL,
		p.IsAuthor, p.IsPublisher, p.IsVerifiedCritic, p.PrivacyLevel,
		p.ReadingGoalAnnual, p.PreferredLanguage, p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			if pgErr.ConstraintName == "user_profile_pkey" {
				return ErrDuplicateProfile
			}
			return ErrDuplicateUsername
		}
		return err
	}
	return nil
}

// GetByID retrieves a profile by its user ID.
func (r *Repo) GetByID(ctx context.Context, id string) (*Profile, error) {
	p := &Profile{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, username, display_name, bio, avatar_url, location, website_url,
			is_author, is_publisher, is_verified_critic, privacy_level,
			reading_goal_annual, preferred_language, created_at, updated_at
		 FROM user_profile WHERE id = $1`, id,
	).Scan(&p.ID, &p.Username, &p.DisplayName, &p.Bio, &p.AvatarURL, &p.Location,
		&p.WebsiteURL, &p.IsAuthor, &p.IsPublisher, &p.IsVerifiedCritic,
		&p.PrivacyLevel, &p.ReadingGoalAnnual, &p.PreferredLanguage,
		&p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return p, err
}

// UpdateFields performs a partial update on the profile within a transaction.
// Only non-nil fields in the map are updated. Returns the updated profile.
func (r *Repo) UpdateFields(ctx context.Context, tx pgx.Tx, id string, fields map[string]any) (*Profile, error) {
	if len(fields) == 0 {
		return nil, errors.New("no fields to update")
	}

	query := "UPDATE user_profile SET updated_at = NOW()"
	args := []any{}
	argIdx := 1

	for col, val := range fields {
		query += fmt.Sprintf(", %s = $%d", col, argIdx)
		args = append(args, val)
		argIdx++
	}

	query += fmt.Sprintf(" WHERE id = $%d", argIdx)
	args = append(args, id)
	query += ` RETURNING id, username, display_name, bio, avatar_url, location, website_url,
		is_author, is_publisher, is_verified_critic, privacy_level,
		reading_goal_annual, preferred_language, created_at, updated_at`

	p := &Profile{}
	err := tx.QueryRow(ctx, query, args...).Scan(
		&p.ID, &p.Username, &p.DisplayName, &p.Bio, &p.AvatarURL, &p.Location,
		&p.WebsiteURL, &p.IsAuthor, &p.IsPublisher, &p.IsVerifiedCritic,
		&p.PrivacyLevel, &p.ReadingGoalAnnual, &p.PreferredLanguage,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrDuplicateUsername
		}
		return nil, err
	}
	return p, nil
}

// Exists checks if a profile with the given ID exists.
func (r *Repo) Exists(ctx context.Context, id string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM user_profile WHERE id = $1)`, id,
	).Scan(&exists)
	return exists, err
}
