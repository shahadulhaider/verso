package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrAlreadyFollowing = errors.New("already following")
	ErrSelfFollow       = errors.New("cannot follow self")
	ErrBlockedByTarget  = errors.New("blocked by target user")
	ErrAlreadyBlocked   = errors.New("already blocked")
	ErrSelfBlock        = errors.New("cannot block self")
)

type Follow struct {
	FollowerID string
	FollowedID string
	CreatedAt  time.Time
}

type Block struct {
	BlockerID string
	BlockedID string
	CreatedAt time.Time
}

type Repo struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

func (r *Repo) Pool() *pgxpool.Pool {
	return r.pool
}

func (r *Repo) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return r.pool.Begin(ctx)
}

func (r *Repo) CreateFollow(ctx context.Context, tx pgx.Tx, followerID, followedID string) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO follow (follower_id, followed_id, created_at) VALUES ($1, $2, NOW())`,
		followerID, followedID,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrAlreadyFollowing
		}
		return err
	}
	return nil
}

func (r *Repo) DeleteFollow(ctx context.Context, tx pgx.Tx, followerID, followedID string) error {
	_, err := tx.Exec(ctx,
		`DELETE FROM follow WHERE follower_id = $1 AND followed_id = $2`,
		followerID, followedID,
	)
	return err
}

func (r *Repo) DeleteFollowBothDirections(ctx context.Context, tx pgx.Tx, userA, userB string) error {
	_, err := tx.Exec(ctx,
		`DELETE FROM follow WHERE (follower_id = $1 AND followed_id = $2) OR (follower_id = $2 AND followed_id = $1)`,
		userA, userB,
	)
	return err
}

func (r *Repo) IsBlocked(ctx context.Context, blockerID, blockedID string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM block WHERE blocker_id = $1 AND blocked_id = $2)`,
		blockerID, blockedID,
	).Scan(&exists)
	return exists, err
}

func (r *Repo) CreateBlock(ctx context.Context, tx pgx.Tx, blockerID, blockedID string) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO block (blocker_id, blocked_id, created_at) VALUES ($1, $2, NOW())`,
		blockerID, blockedID,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrAlreadyBlocked
		}
		return err
	}
	return nil
}

func (r *Repo) DeleteBlock(ctx context.Context, tx pgx.Tx, blockerID, blockedID string) error {
	_, err := tx.Exec(ctx,
		`DELETE FROM block WHERE blocker_id = $1 AND blocked_id = $2`,
		blockerID, blockedID,
	)
	return err
}

type FollowerRow struct {
	UserID    string
	CreatedAt time.Time
}

func (r *Repo) ListFollowers(ctx context.Context, userID, cursor string, limit int) ([]FollowerRow, error) {
	var rows pgx.Rows
	var err error

	if cursor == "" {
		rows, err = r.pool.Query(ctx,
			`SELECT follower_id, created_at FROM follow
			 WHERE followed_id = $1
			 ORDER BY created_at DESC LIMIT $2`,
			userID, limit,
		)
	} else {
		rows, err = r.pool.Query(ctx,
			`SELECT follower_id, created_at FROM follow
			 WHERE followed_id = $1 AND created_at < $2
			 ORDER BY created_at DESC LIMIT $3`,
			userID, cursor, limit,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []FollowerRow
	for rows.Next() {
		var f FollowerRow
		if err := rows.Scan(&f.UserID, &f.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, f)
	}
	return result, rows.Err()
}

func (r *Repo) ListFollowing(ctx context.Context, userID, cursor string, limit int) ([]FollowerRow, error) {
	var rows pgx.Rows
	var err error

	if cursor == "" {
		rows, err = r.pool.Query(ctx,
			`SELECT followed_id, created_at FROM follow
			 WHERE follower_id = $1
			 ORDER BY created_at DESC LIMIT $2`,
			userID, limit,
		)
	} else {
		rows, err = r.pool.Query(ctx,
			`SELECT followed_id, created_at FROM follow
			 WHERE follower_id = $1 AND created_at < $2
			 ORDER BY created_at DESC LIMIT $3`,
			userID, cursor, limit,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []FollowerRow
	for rows.Next() {
		var f FollowerRow
		if err := rows.Scan(&f.UserID, &f.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, f)
	}
	return result, rows.Err()
}

func (r *Repo) CountFollowers(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM follow WHERE followed_id = $1`, userID,
	).Scan(&count)
	return count, err
}

func (r *Repo) CountFollowing(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM follow WHERE follower_id = $1`, userID,
	).Scan(&count)
	return count, err
}
