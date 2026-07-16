package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

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

func (r *Repo) CreateReview(ctx context.Context, tx pgx.Tx, rev *Review) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO review (id, user_id, work_id, edition_id, rating_overall,
		 rating_plot, rating_characters, rating_pacing, rating_prose,
		 title, body, contains_spoilers, like_count, comment_count, helpful_count,
		 is_featured, created_at, updated_at, deleted_at, version)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20)`,
		rev.ID, rev.UserID, rev.WorkID, rev.EditionID, rev.RatingOverall,
		rev.RatingPlot, rev.RatingCharacters, rev.RatingPacing, rev.RatingProse,
		rev.Title, rev.Body, rev.ContainsSpoilers, rev.LikeCount, rev.CommentCount,
		rev.HelpfulCount, rev.IsFeatured, rev.CreatedAt, rev.UpdatedAt, rev.DeletedAt, rev.Version,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrDuplicateReview
		}
		return err
	}
	return nil
}

func (r *Repo) GetReviewByID(ctx context.Context, id string) (*Review, error) {
	rev := &Review{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, work_id, edition_id, rating_overall,
		 rating_plot, rating_characters, rating_pacing, rating_prose,
		 title, body, contains_spoilers, like_count, comment_count, helpful_count,
		 is_featured, created_at, updated_at, deleted_at, version
		 FROM review WHERE id = $1 AND deleted_at IS NULL`, id,
	).Scan(&rev.ID, &rev.UserID, &rev.WorkID, &rev.EditionID, &rev.RatingOverall,
		&rev.RatingPlot, &rev.RatingCharacters, &rev.RatingPacing, &rev.RatingProse,
		&rev.Title, &rev.Body, &rev.ContainsSpoilers, &rev.LikeCount, &rev.CommentCount,
		&rev.HelpfulCount, &rev.IsFeatured, &rev.CreatedAt, &rev.UpdatedAt, &rev.DeletedAt, &rev.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return rev, err
}

func (r *Repo) UpdateReview(ctx context.Context, tx pgx.Tx, rev *Review) error {
	tag, err := tx.Exec(ctx,
		`UPDATE review SET rating_overall=$1, rating_plot=$2, rating_characters=$3,
		 rating_pacing=$4, rating_prose=$5, title=$6, body=$7, contains_spoilers=$8,
		 updated_at=$9, version=$10
		 WHERE id=$11 AND version=$12-1 AND deleted_at IS NULL`,
		rev.RatingOverall, rev.RatingPlot, rev.RatingCharacters,
		rev.RatingPacing, rev.RatingProse, rev.Title, rev.Body, rev.ContainsSpoilers,
		rev.UpdatedAt, rev.Version, rev.ID, rev.Version,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repo) SoftDeleteReview(ctx context.Context, tx pgx.Tx, id string, now time.Time) error {
	tag, err := tx.Exec(ctx,
		`UPDATE review SET deleted_at=$1, updated_at=$1 WHERE id=$2 AND deleted_at IS NULL`,
		now, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repo) ListReviewsByWork(ctx context.Context, workID, cursor, sort string, limit int) ([]*Review, error) {
	orderCol := "created_at"
	if sort == "helpful" {
		orderCol = "helpful_count"
	}

	var rows pgx.Rows
	var err error
	if cursor != "" {
		cursorTime, parseErr := time.Parse(time.RFC3339Nano, cursor)
		if parseErr != nil {
			return nil, parseErr
		}
		rows, err = r.pool.Query(ctx,
			`SELECT id, user_id, work_id, edition_id, rating_overall,
			 rating_plot, rating_characters, rating_pacing, rating_prose,
			 title, body, contains_spoilers, like_count, comment_count, helpful_count,
			 is_featured, created_at, updated_at, deleted_at, version
			 FROM review WHERE work_id=$1 AND deleted_at IS NULL AND created_at < $2
			 ORDER BY `+orderCol+` DESC LIMIT $3`, workID, cursorTime, limit)
	} else {
		rows, err = r.pool.Query(ctx,
			`SELECT id, user_id, work_id, edition_id, rating_overall,
			 rating_plot, rating_characters, rating_pacing, rating_prose,
			 title, body, contains_spoilers, like_count, comment_count, helpful_count,
			 is_featured, created_at, updated_at, deleted_at, version
			 FROM review WHERE work_id=$1 AND deleted_at IS NULL
			 ORDER BY `+orderCol+` DESC LIMIT $2`, workID, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectReviews(rows)
}

func (r *Repo) GetAggregateRating(ctx context.Context, workID string) (*AggregateRating, error) {
	agg := &AggregateRating{}
	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(AVG(rating_overall),0), COUNT(*), COUNT(*)
		 FROM review WHERE work_id=$1 AND deleted_at IS NULL`, workID,
	).Scan(&agg.AverageRating, &agg.RatingsCount, &agg.ReviewsCount)
	if err != nil {
		return nil, err
	}

	r.pool.QueryRow(ctx,
		`SELECT AVG(rating_plot), AVG(rating_characters), AVG(rating_pacing), AVG(rating_prose)
		 FROM review WHERE work_id=$1 AND deleted_at IS NULL`, workID,
	).Scan(&agg.AxisRatings.Plot, &agg.AxisRatings.Characters,
		&agg.AxisRatings.Pacing, &agg.AxisRatings.Prose)

	return agg, nil
}

func (r *Repo) GetVote(ctx context.Context, userID, reviewID string) (*ReviewVote, error) {
	v := &ReviewVote{}
	err := r.pool.QueryRow(ctx,
		`SELECT user_id, review_id, vote_type, created_at
		 FROM review_vote WHERE user_id=$1 AND review_id=$2`, userID, reviewID,
	).Scan(&v.UserID, &v.ReviewID, &v.VoteType, &v.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return v, err
}

func (r *Repo) InsertVote(ctx context.Context, tx pgx.Tx, v *ReviewVote) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO review_vote (user_id, review_id, vote_type, created_at)
		 VALUES ($1,$2,$3,$4)`, v.UserID, v.ReviewID, v.VoteType, v.CreatedAt)
	return err
}

func (r *Repo) UpdateVoteType(ctx context.Context, tx pgx.Tx, userID, reviewID, voteType string) error {
	_, err := tx.Exec(ctx,
		`UPDATE review_vote SET vote_type=$1 WHERE user_id=$2 AND review_id=$3`,
		voteType, userID, reviewID)
	return err
}

func (r *Repo) DeleteVote(ctx context.Context, tx pgx.Tx, userID, reviewID string) error {
	_, err := tx.Exec(ctx,
		`DELETE FROM review_vote WHERE user_id=$1 AND review_id=$2`, userID, reviewID)
	return err
}

func (r *Repo) IncrementCounter(ctx context.Context, tx pgx.Tx, reviewID, column string, delta int) error {
	_, err := tx.Exec(ctx,
		`UPDATE review SET `+column+` = `+column+` + $1, updated_at = NOW() WHERE id = $2`,
		delta, reviewID)
	return err
}

func (r *Repo) CreateComment(ctx context.Context, tx pgx.Tx, c *ReviewComment) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO review_comment (id, review_id, user_id, parent_comment_id, body, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6)`,
		c.ID, c.ReviewID, c.UserID, c.ParentCommentID, c.Body, c.CreatedAt)
	return err
}

func (r *Repo) ListCommentsByReview(ctx context.Context, reviewID string) ([]*ReviewComment, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, review_id, user_id, parent_comment_id, body, created_at, deleted_at
		 FROM review_comment WHERE review_id=$1 AND deleted_at IS NULL
		 ORDER BY created_at ASC`, reviewID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []*ReviewComment
	for rows.Next() {
		c := &ReviewComment{}
		if err := rows.Scan(&c.ID, &c.ReviewID, &c.UserID, &c.ParentCommentID,
			&c.Body, &c.CreatedAt, &c.DeletedAt); err != nil {
			return nil, err
		}
		c.ID = strings.TrimSpace(c.ID)
		comments = append(comments, c)
	}
	return comments, rows.Err()
}

func collectReviews(rows pgx.Rows) ([]*Review, error) {
	var reviews []*Review
	for rows.Next() {
		rev := &Review{}
		if err := rows.Scan(&rev.ID, &rev.UserID, &rev.WorkID, &rev.EditionID,
			&rev.RatingOverall, &rev.RatingPlot, &rev.RatingCharacters,
			&rev.RatingPacing, &rev.RatingProse, &rev.Title, &rev.Body,
			&rev.ContainsSpoilers, &rev.LikeCount, &rev.CommentCount,
			&rev.HelpfulCount, &rev.IsFeatured, &rev.CreatedAt, &rev.UpdatedAt,
			&rev.DeletedAt, &rev.Version); err != nil {
			return nil, err
		}
		rev.ID = strings.TrimSpace(rev.ID)
		reviews = append(reviews, rev)
	}
	return reviews, rows.Err()
}
