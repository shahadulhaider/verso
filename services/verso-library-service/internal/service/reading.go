package service

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/shahadulhaider/verso/libs/go/envelope"
	"github.com/shahadulhaider/verso/libs/go/outbox"

	"github.com/shahadulhaider/verso/services/verso-library-service/internal/repository"
)

const (
	progressUpdatedEvt   = "verso.library.reading-progress-updated.v1"
	aggregateProgress    = "reading_progress"
)

type LogSessionRequest struct {
	FormatID       string  `json:"formatId"`
	WorkID         string  `json:"workId"`
	ProgressBefore float64 `json:"progressBefore"`
	ProgressAfter  float64 `json:"progressAfter"`
	PagesRead      *int    `json:"pagesRead,omitempty"`
	DeviceType     *string `json:"deviceType,omitempty"`
}

func (s *LibraryService) LogReadingSession(ctx context.Context, userID string, req LogSessionRequest) (*repository.ReadingSession, error) {
	if req.FormatID == "" || req.WorkID == "" {
		return nil, errors.New("formatId and workId are required")
	}
	if req.ProgressAfter < req.ProgressBefore {
		return nil, errors.New("progressAfter must be >= progressBefore")
	}
	if req.ProgressAfter > 100 || req.ProgressBefore < 0 {
		return nil, errors.New("progress must be between 0 and 100")
	}

	now := time.Now().UTC()
	session := &repository.ReadingSession{
		ID:             ulid.MustNew(ulid.Now(), rand.Reader).String(),
		UserID:         userID,
		FormatID:       req.FormatID,
		WorkID:         req.WorkID,
		StartedAt:      now,
		ProgressBefore: req.ProgressBefore,
		ProgressAfter:  req.ProgressAfter,
		PagesRead:      req.PagesRead,
		DeviceType:     req.DeviceType,
		CreatedAt:      now,
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if err := s.repo.CreateReadingSession(ctx, tx, session); err != nil {
		return nil, err
	}
	return session, tx.Commit(ctx)
}

func (s *LibraryService) GetReadingProgress(ctx context.Context, userID, workID string) (*repository.ReadingProgress, error) {
	return s.repo.GetReadingProgress(ctx, userID, workID)
}

type UpdateProgressRequest struct {
	ProgressPercent *float64 `json:"progressPercent,omitempty"`
	CurrentPage     *int     `json:"currentPage,omitempty"`
	Status          *string  `json:"status,omitempty"`
	FormatID        string   `json:"formatId"`
}

var validStatuses = map[string]bool{
	"not_started": true,
	"reading":     true,
	"completed":   true,
	"dnf":         true,
}

func (s *LibraryService) UpdateReadingProgress(ctx context.Context, userID, workID string, req UpdateProgressRequest) (*repository.ReadingProgress, error) {
	if req.FormatID == "" {
		return nil, errors.New("formatId is required")
	}
	if req.Status != nil && !validStatuses[*req.Status] {
		return nil, errors.New("invalid status")
	}
	if req.ProgressPercent != nil && (*req.ProgressPercent < 0 || *req.ProgressPercent > 100) {
		return nil, errors.New("progressPercent must be between 0 and 100")
	}

	now := time.Now().UTC()

	existing, err := s.repo.GetReadingProgress(ctx, userID, workID)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}

	progress := &repository.ReadingProgress{
		UserID:          userID,
		WorkID:          workID,
		CurrentFormatID: req.FormatID,
		UpdatedAt:       now,
	}

	if existing != nil {
		progress.ProgressPercent = existing.ProgressPercent
		progress.CurrentPage = existing.CurrentPage
		progress.Status = existing.Status
		progress.StartedAt = existing.StartedAt
		progress.CompletedAt = existing.CompletedAt
		progress.ReadCount = existing.ReadCount
	} else {
		progress.Status = "not_started"
	}

	if req.ProgressPercent != nil {
		progress.ProgressPercent = *req.ProgressPercent
	}
	if req.CurrentPage != nil {
		progress.CurrentPage = req.CurrentPage
	}
	if req.Status != nil {
		progress.Status = *req.Status
		if *req.Status == "reading" && progress.StartedAt == nil {
			progress.StartedAt = &now
		}
		if *req.Status == "completed" {
			progress.CompletedAt = &now
			if existing != nil && existing.Status == "completed" {
				progress.ReadCount = existing.ReadCount + 1
			} else if existing == nil {
				progress.ReadCount = 1
			} else {
				progress.ReadCount = existing.ReadCount + 1
			}
		}
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if err := s.repo.UpsertReadingProgress(ctx, tx, progress); err != nil {
		return nil, err
	}

	payload, _ := json.Marshal(map[string]any{
		"userId":          userID,
		"workId":          workID,
		"progressPercent": progress.ProgressPercent,
		"status":          progress.Status,
	})
	env := envelope.New(ctx, progressUpdatedEvt, serviceName, userID, payload)
	if err := outbox.InsertEvent(ctx, tx, aggregateProgress, userID+":"+workID, env); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return progress, nil
}
