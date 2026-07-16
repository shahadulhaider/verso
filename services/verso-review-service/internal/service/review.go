package service

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"log/slog"
	"math"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/shahadulhaider/verso/libs/go/envelope"
	"github.com/shahadulhaider/verso/libs/go/outbox"

	"github.com/shahadulhaider/verso/services/verso-review-service/internal/repository"
)

const (
	serviceName       = "verso-review-service"
	reviewPublishedV1 = "verso.review.review-published.v1"
	ratingUpdatedV1   = "verso.review.rating-updated.v1"
	aggregateReview   = "review"
)

type ReviewService struct {
	repo *repository.Repo
	log  *slog.Logger
}

func New(repo *repository.Repo, log *slog.Logger) *ReviewService {
	return &ReviewService{repo: repo, log: log}
}

type CreateReviewRequest struct {
	WorkID           string   `json:"workId"`
	EditionID        *string  `json:"editionId,omitempty"`
	RatingOverall    float64  `json:"ratingOverall"`
	RatingPlot       *float64 `json:"ratingPlot,omitempty"`
	RatingCharacters *float64 `json:"ratingCharacters,omitempty"`
	RatingPacing     *float64 `json:"ratingPacing,omitempty"`
	RatingProse      *float64 `json:"ratingProse,omitempty"`
	Title            *string  `json:"title,omitempty"`
	Body             *string  `json:"body,omitempty"`
	ContainsSpoilers bool     `json:"containsSpoilers"`
}

func (s *ReviewService) CreateReview(ctx context.Context, userID string, req CreateReviewRequest) (*repository.Review, error) {
	if req.WorkID == "" {
		return nil, errors.New("workId is required")
	}
	if err := validateRating(req.RatingOverall); err != nil {
		return nil, err
	}
	if err := validateOptionalRatings(req.RatingPlot, req.RatingCharacters, req.RatingPacing, req.RatingProse); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	rev := &repository.Review{
		ID:               ulid.MustNew(ulid.Now(), rand.Reader).String(),
		UserID:           userID,
		WorkID:           req.WorkID,
		EditionID:        req.EditionID,
		RatingOverall:    req.RatingOverall,
		RatingPlot:       req.RatingPlot,
		RatingCharacters: req.RatingCharacters,
		RatingPacing:     req.RatingPacing,
		RatingProse:      req.RatingProse,
		Title:            req.Title,
		Body:             req.Body,
		ContainsSpoilers: req.ContainsSpoilers,
		CreatedAt:        now,
		UpdatedAt:        now,
		Version:          1,
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if err := s.repo.CreateReview(ctx, tx, rev); err != nil {
		return nil, err
	}

	payload, _ := json.Marshal(map[string]any{
		"reviewId":      rev.ID,
		"userId":        userID,
		"workId":        req.WorkID,
		"ratingOverall": req.RatingOverall,
	})
	env := envelope.New(ctx, reviewPublishedV1, serviceName, req.WorkID, payload)
	if err := outbox.InsertEvent(ctx, tx, aggregateReview, rev.ID, env); err != nil {
		return nil, err
	}

	return rev, tx.Commit(ctx)
}

type UpdateReviewRequest struct {
	RatingOverall    *float64 `json:"ratingOverall,omitempty"`
	RatingPlot       *float64 `json:"ratingPlot,omitempty"`
	RatingCharacters *float64 `json:"ratingCharacters,omitempty"`
	RatingPacing     *float64 `json:"ratingPacing,omitempty"`
	RatingProse      *float64 `json:"ratingProse,omitempty"`
	Title            *string  `json:"title,omitempty"`
	Body             *string  `json:"body,omitempty"`
	ContainsSpoilers *bool    `json:"containsSpoilers,omitempty"`
}

func (s *ReviewService) UpdateReview(ctx context.Context, userID, reviewID string, req UpdateReviewRequest) (*repository.Review, error) {
	rev, err := s.repo.GetReviewByID(ctx, reviewID)
	if err != nil {
		return nil, err
	}
	if rev.UserID != userID {
		return nil, repository.ErrForbidden
	}

	oldRating := rev.RatingOverall

	if req.RatingOverall != nil {
		if err := validateRating(*req.RatingOverall); err != nil {
			return nil, err
		}
		rev.RatingOverall = *req.RatingOverall
	}
	if req.RatingPlot != nil {
		if err := validateOptionalRating(*req.RatingPlot); err != nil {
			return nil, err
		}
		rev.RatingPlot = req.RatingPlot
	}
	if req.RatingCharacters != nil {
		if err := validateOptionalRating(*req.RatingCharacters); err != nil {
			return nil, err
		}
		rev.RatingCharacters = req.RatingCharacters
	}
	if req.RatingPacing != nil {
		if err := validateOptionalRating(*req.RatingPacing); err != nil {
			return nil, err
		}
		rev.RatingPacing = req.RatingPacing
	}
	if req.RatingProse != nil {
		if err := validateOptionalRating(*req.RatingProse); err != nil {
			return nil, err
		}
		rev.RatingProse = req.RatingProse
	}
	if req.Title != nil {
		rev.Title = req.Title
	}
	if req.Body != nil {
		rev.Body = req.Body
	}
	if req.ContainsSpoilers != nil {
		rev.ContainsSpoilers = *req.ContainsSpoilers
	}

	rev.Version++
	rev.UpdatedAt = time.Now().UTC()

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if err := s.repo.UpdateReview(ctx, tx, rev); err != nil {
		return nil, err
	}

	if oldRating != rev.RatingOverall {
		payload, _ := json.Marshal(map[string]any{
			"reviewId":  rev.ID,
			"userId":    userID,
			"workId":    rev.WorkID,
			"oldRating": oldRating,
			"newRating": rev.RatingOverall,
		})
		env := envelope.New(ctx, ratingUpdatedV1, serviceName, rev.WorkID, payload)
		if err := outbox.InsertEvent(ctx, tx, aggregateReview, rev.ID, env); err != nil {
			return nil, err
		}
	}

	return rev, tx.Commit(ctx)
}

func (s *ReviewService) DeleteReview(ctx context.Context, userID, reviewID string) error {
	rev, err := s.repo.GetReviewByID(ctx, reviewID)
	if err != nil {
		return err
	}
	if rev.UserID != userID {
		return repository.ErrForbidden
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := s.repo.SoftDeleteReview(ctx, tx, reviewID, time.Now().UTC()); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *ReviewService) GetReview(ctx context.Context, id string) (*repository.Review, error) {
	return s.repo.GetReviewByID(ctx, id)
}

func (s *ReviewService) ListReviews(ctx context.Context, workID, cursor, sort string, limit int) ([]*repository.Review, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return s.repo.ListReviewsByWork(ctx, workID, cursor, sort, limit)
}

func (s *ReviewService) GetAggregateRating(ctx context.Context, workID string) (*repository.AggregateRating, error) {
	return s.repo.GetAggregateRating(ctx, workID)
}

func (s *ReviewService) ListComments(ctx context.Context, reviewID string) ([]*repository.ReviewComment, error) {
	return s.repo.ListCommentsByReview(ctx, reviewID)
}

func validateRating(r float64) error {
	if r < 0.5 || r > 5.0 {
		return errors.New("rating must be between 0.5 and 5.0")
	}
	if math.Mod(r*2, 1) != 0 {
		return errors.New("rating must be in 0.5 increments")
	}
	return nil
}

func validateOptionalRating(r float64) error {
	return validateRating(r)
}

func validateOptionalRatings(ratings ...*float64) error {
	for _, r := range ratings {
		if r != nil {
			if err := validateRating(*r); err != nil {
				return err
			}
		}
	}
	return nil
}
