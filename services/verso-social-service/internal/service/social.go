package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/shahadulhaider/verso/libs/go/envelope"
	"github.com/shahadulhaider/verso/libs/go/outbox"

	"github.com/shahadulhaider/verso/services/verso-social-service/internal/repository"
)

const (
	svcName            = "verso-social-service"
	userFollowedEvent  = "verso.social.user-followed.v1"
	aggregateTypeFollow = "follow"
)

type SocialService struct {
	repo *repository.Repo
	log  *slog.Logger
}

func New(repo *repository.Repo, log *slog.Logger) *SocialService {
	return &SocialService{repo: repo, log: log}
}

func (s *SocialService) Follow(ctx context.Context, followerID, followedID string) error {
	followerID = strings.TrimSpace(followerID)
	followedID = strings.TrimSpace(followedID)

	if followerID == followedID {
		return repository.ErrSelfFollow
	}

	blocked, err := s.repo.IsBlocked(ctx, followedID, followerID)
	if err != nil {
		return err
	}
	if blocked {
		return repository.ErrBlockedByTarget
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := s.repo.CreateFollow(ctx, tx, followerID, followedID); err != nil {
		return err
	}

	if err := s.publishUserFollowed(ctx, tx, followerID, followedID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (s *SocialService) Unfollow(ctx context.Context, followerID, followedID string) error {
	followerID = strings.TrimSpace(followerID)
	followedID = strings.TrimSpace(followedID)

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := s.repo.DeleteFollow(ctx, tx, followerID, followedID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (s *SocialService) Block(ctx context.Context, blockerID, blockedID string) error {
	blockerID = strings.TrimSpace(blockerID)
	blockedID = strings.TrimSpace(blockedID)

	if blockerID == blockedID {
		return repository.ErrSelfBlock
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := s.repo.CreateBlock(ctx, tx, blockerID, blockedID); err != nil {
		return err
	}

	if err := s.repo.DeleteFollowBothDirections(ctx, tx, blockerID, blockedID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (s *SocialService) Unblock(ctx context.Context, blockerID, blockedID string) error {
	blockerID = strings.TrimSpace(blockerID)
	blockedID = strings.TrimSpace(blockedID)

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := s.repo.DeleteBlock(ctx, tx, blockerID, blockedID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

type FollowListResult struct {
	UserIDs    []string `json:"userIds"`
	Count      int      `json:"count"`
	NextCursor string   `json:"nextCursor,omitempty"`
}

func (s *SocialService) Followers(ctx context.Context, userID, cursor string, limit int) (*FollowListResult, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	rows, err := s.repo.ListFollowers(ctx, userID, cursor, limit+1)
	if err != nil {
		return nil, err
	}

	result := &FollowListResult{UserIDs: make([]string, 0)}
	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}

	for _, r := range rows {
		result.UserIDs = append(result.UserIDs, strings.TrimSpace(r.UserID))
	}

	count, err := s.repo.CountFollowers(ctx, userID)
	if err != nil {
		return nil, err
	}
	result.Count = count

	if hasMore && len(rows) > 0 {
		result.NextCursor = rows[len(rows)-1].CreatedAt.Format("2006-01-02T15:04:05.999999Z")
	}

	return result, nil
}

func (s *SocialService) Following(ctx context.Context, userID, cursor string, limit int) (*FollowListResult, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	rows, err := s.repo.ListFollowing(ctx, userID, cursor, limit+1)
	if err != nil {
		return nil, err
	}

	result := &FollowListResult{UserIDs: make([]string, 0)}
	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}

	for _, r := range rows {
		result.UserIDs = append(result.UserIDs, strings.TrimSpace(r.UserID))
	}

	count, err := s.repo.CountFollowing(ctx, userID)
	if err != nil {
		return nil, err
	}
	result.Count = count

	if hasMore && len(rows) > 0 {
		result.NextCursor = rows[len(rows)-1].CreatedAt.Format("2006-01-02T15:04:05.999999Z")
	}

	return result, nil
}

type CountsResult struct {
	FollowersCount int `json:"followersCount"`
	FollowingCount int `json:"followingCount"`
}

func (s *SocialService) Counts(ctx context.Context, userID string) (*CountsResult, error) {
	followers, err := s.repo.CountFollowers(ctx, userID)
	if err != nil {
		return nil, err
	}

	following, err := s.repo.CountFollowing(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &CountsResult{
		FollowersCount: followers,
		FollowingCount: following,
	}, nil
}

func (s *SocialService) publishUserFollowed(ctx context.Context, tx pgx.Tx, followerID, followedID string) error {
	payload, err := json.Marshal(map[string]string{
		"followerId": followerID,
		"followedId": followedID,
	})
	if err != nil {
		return err
	}

	env := envelope.New(ctx, userFollowedEvent, svcName, followedID, payload)
	return outbox.InsertEvent(ctx, tx, aggregateTypeFollow, followedID, env)
}
