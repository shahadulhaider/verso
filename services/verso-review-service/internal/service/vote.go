package service

import (
	"context"
	"errors"
	"time"

	"github.com/shahadulhaider/verso/services/verso-review-service/internal/repository"
)

type CastVoteRequest struct {
	VoteType string `json:"voteType"`
}

type VoteResult struct {
	Action   string `json:"action"`
	VoteType string `json:"voteType"`
}

func (s *ReviewService) CastVote(ctx context.Context, userID, reviewID string, req CastVoteRequest) (*VoteResult, error) {
	if req.VoteType != "like" && req.VoteType != "helpful" {
		return nil, errors.New("voteType must be 'like' or 'helpful'")
	}

	if _, err := s.repo.GetReviewByID(ctx, reviewID); err != nil {
		return nil, err
	}

	existing, err := s.repo.GetVote(ctx, userID, reviewID)
	if err != nil {
		return nil, err
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var result VoteResult

	if existing == nil {
		vote := &repository.ReviewVote{
			UserID:    userID,
			ReviewID:  reviewID,
			VoteType:  req.VoteType,
			CreatedAt: time.Now().UTC(),
		}
		if err := s.repo.InsertVote(ctx, tx, vote); err != nil {
			return nil, err
		}
		if err := s.repo.IncrementCounter(ctx, tx, reviewID, counterColumn(req.VoteType), 1); err != nil {
			return nil, err
		}
		result = VoteResult{Action: "created", VoteType: req.VoteType}
	} else if existing.VoteType == req.VoteType {
		if err := s.repo.DeleteVote(ctx, tx, userID, reviewID); err != nil {
			return nil, err
		}
		if err := s.repo.IncrementCounter(ctx, tx, reviewID, counterColumn(req.VoteType), -1); err != nil {
			return nil, err
		}
		result = VoteResult{Action: "removed", VoteType: req.VoteType}
	} else {
		if err := s.repo.UpdateVoteType(ctx, tx, userID, reviewID, req.VoteType); err != nil {
			return nil, err
		}
		if err := s.repo.IncrementCounter(ctx, tx, reviewID, counterColumn(existing.VoteType), -1); err != nil {
			return nil, err
		}
		if err := s.repo.IncrementCounter(ctx, tx, reviewID, counterColumn(req.VoteType), 1); err != nil {
			return nil, err
		}
		result = VoteResult{Action: "switched", VoteType: req.VoteType}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &result, nil
}

func counterColumn(voteType string) string {
	if voteType == "helpful" {
		return "helpful_count"
	}
	return "like_count"
}
