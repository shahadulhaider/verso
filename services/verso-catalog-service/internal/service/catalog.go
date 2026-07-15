package service

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/shahadulhaider/verso/libs/go/envelope"
	"github.com/shahadulhaider/verso/libs/go/outbox"

	"github.com/shahadulhaider/verso/services/verso-catalog-service/internal/repository"
)

const (
	eventWorkCreated      = "verso.catalog.work-created.v1"
	eventEditionPublished = "verso.catalog.edition-published.v1"
	producer              = "verso-catalog-service"
)

type CatalogService struct {
	repo *repository.Repo
	log  *slog.Logger
}

func New(repo *repository.Repo, log *slog.Logger) *CatalogService {
	return &CatalogService{repo: repo, log: log}
}

type CreateWorkRequest struct {
	Title                   string
	Description             *string
	OriginalLanguage        *string
	OriginalPublicationYear *int
}

func (s *CatalogService) CreateWork(ctx context.Context, req CreateWorkRequest) (*repository.Work, error) {
	if req.Title == "" {
		return nil, fmt.Errorf("title is required")
	}

	now := time.Now().UTC()
	workID := ulid.MustNew(ulid.Timestamp(now), rand.Reader).String()

	work := &repository.Work{
		ID:                      workID,
		Title:                   req.Title,
		Description:             req.Description,
		OriginalLanguage:        req.OriginalLanguage,
		OriginalPublicationYear: req.OriginalPublicationYear,
		AvgRating:               0,
		RatingsCount:            0,
		ReviewsCount:            0,
		CreatedAt:               now,
		UpdatedAt:               now,
		Version:                 1,
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := s.repo.CreateWork(ctx, tx, work); err != nil {
		return nil, fmt.Errorf("create work: %w", err)
	}

	payload, _ := json.Marshal(map[string]string{
		"workId":    workID,
		"title":     req.Title,
		"createdAt": now.Format(time.RFC3339),
	})
	env := envelope.New(ctx, eventWorkCreated, producer, workID, payload)
	if err := outbox.InsertEvent(ctx, tx, "work", workID, env); err != nil {
		return nil, fmt.Errorf("insert outbox event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	s.log.InfoContext(ctx, "work created", slog.String("work_id", workID))
	return work, nil
}

func (s *CatalogService) GetWork(ctx context.Context, id string) (*repository.Work, []repository.Edition, error) {
	work, err := s.repo.GetWorkByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	editions, err := s.repo.ListEditionsByWorkID(ctx, id)
	if err != nil {
		return nil, nil, fmt.Errorf("list editions: %w", err)
	}

	return work, editions, nil
}

func (s *CatalogService) ListWorks(ctx context.Context, cursor string, limit int) ([]repository.Work, string, error) {
	return s.repo.ListWorks(ctx, cursor, limit)
}

type CreateEditionRequest struct {
	Title           *string
	Language        *string
	Publisher       *string
	PublicationDate *time.Time
	PageCount       *int
}

func (s *CatalogService) CreateEdition(ctx context.Context, workID string, req CreateEditionRequest) (*repository.Edition, error) {
	exists, err := s.repo.WorkExists(ctx, workID)
	if err != nil {
		return nil, fmt.Errorf("check work: %w", err)
	}
	if !exists {
		return nil, repository.ErrNotFound
	}

	now := time.Now().UTC()
	editionID := ulid.MustNew(ulid.Timestamp(now), rand.Reader).String()

	edition := &repository.Edition{
		ID:              editionID,
		WorkID:          workID,
		Title:           req.Title,
		Language:        req.Language,
		Publisher:       req.Publisher,
		PublicationDate: req.PublicationDate,
		PageCount:       req.PageCount,
		CreatedAt:       now,
		UpdatedAt:       now,
		Version:         1,
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := s.repo.CreateEdition(ctx, tx, edition); err != nil {
		return nil, fmt.Errorf("create edition: %w", err)
	}

	payload, _ := json.Marshal(map[string]string{
		"editionId": editionID,
		"workId":    workID,
		"createdAt": now.Format(time.RFC3339),
	})
	env := envelope.New(ctx, eventEditionPublished, producer, workID, payload)
	if err := outbox.InsertEvent(ctx, tx, "edition", editionID, env); err != nil {
		return nil, fmt.Errorf("insert outbox event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	s.log.InfoContext(ctx, "edition created", slog.String("edition_id", editionID), slog.String("work_id", workID))
	return edition, nil
}

func (s *CatalogService) GetEdition(ctx context.Context, id string) (*repository.Edition, []repository.Format, error) {
	edition, err := s.repo.GetEditionByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	formats, err := s.repo.ListFormatsByEditionID(ctx, id)
	if err != nil {
		return nil, nil, fmt.Errorf("list formats: %w", err)
	}

	return edition, formats, nil
}
