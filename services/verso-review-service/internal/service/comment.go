package service

import (
	"context"
	"crypto/rand"
	"errors"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/shahadulhaider/verso/services/verso-review-service/internal/repository"
)

type AddCommentRequest struct {
	Body            string  `json:"body"`
	ParentCommentID *string `json:"parentCommentId,omitempty"`
}

func (s *ReviewService) AddComment(ctx context.Context, userID, reviewID string, req AddCommentRequest) (*repository.ReviewComment, error) {
	if req.Body == "" {
		return nil, errors.New("body is required")
	}

	if _, err := s.repo.GetReviewByID(ctx, reviewID); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	comment := &repository.ReviewComment{
		ID:              ulid.MustNew(ulid.Now(), rand.Reader).String(),
		ReviewID:        reviewID,
		UserID:          userID,
		ParentCommentID: req.ParentCommentID,
		Body:            req.Body,
		CreatedAt:       now,
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if err := s.repo.CreateComment(ctx, tx, comment); err != nil {
		return nil, err
	}
	if err := s.repo.IncrementCounter(ctx, tx, reviewID, "comment_count", 1); err != nil {
		return nil, err
	}

	return comment, tx.Commit(ctx)
}
