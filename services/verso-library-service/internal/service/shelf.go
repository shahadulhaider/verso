package service

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/shahadulhaider/verso/libs/go/envelope"
	"github.com/shahadulhaider/verso/libs/go/outbox"

	"github.com/shahadulhaider/verso/services/verso-library-service/internal/repository"
)

const (
	serviceName        = "verso-library-service"
	shelfItemAddedEvt  = "verso.library.shelf-item-added.v1"
	aggregateShelfItem = "shelf_item"
)

var slugRegex = regexp.MustCompile(`[^a-z0-9]+`)

type LibraryService struct {
	repo *repository.Repo
	log  *slog.Logger
}

func New(repo *repository.Repo, log *slog.Logger) *LibraryService {
	return &LibraryService{repo: repo, log: log}
}

func (s *LibraryService) ListShelves(ctx context.Context, userID string) ([]*repository.Shelf, error) {
	return s.repo.ListShelves(ctx, userID)
}

type CreateShelfRequest struct {
	Name      string `json:"name"`
	IsPrivate bool   `json:"isPrivate"`
}

func (s *LibraryService) CreateShelf(ctx context.Context, userID string, req CreateShelfRequest) (*repository.Shelf, error) {
	if req.Name == "" {
		return nil, errors.New("name is required")
	}

	now := time.Now().UTC()
	shelf := &repository.Shelf{
		ID:           ulid.MustNew(ulid.Now(), rand.Reader).String(),
		UserID:       userID,
		Name:         req.Name,
		Slug:         generateSlug(req.Name),
		ShelfType:    "custom",
		IsSystem:     false,
		IsPrivate:    req.IsPrivate,
		DisplayOrder: 100,
		ItemCount:    0,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if err := s.repo.CreateShelf(ctx, tx, shelf); err != nil {
		return nil, err
	}
	return shelf, tx.Commit(ctx)
}

type AddShelfItemRequest struct {
	WorkID    string  `json:"workId"`
	EditionID *string `json:"editionId,omitempty"`
}

func (s *LibraryService) AddItemToShelf(ctx context.Context, userID, shelfID string, req AddShelfItemRequest) (*repository.ShelfItem, error) {
	if req.WorkID == "" {
		return nil, errors.New("workId is required")
	}

	shelf, err := s.repo.GetShelfByID(ctx, shelfID)
	if err != nil {
		return nil, err
	}
	if shelf.UserID != userID {
		return nil, repository.ErrNotFound
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// system shelf move logic: if adding to a system shelf, remove from any other system shelf
	if shelf.IsSystem {
		existingItem, existingShelf, err := s.repo.GetSystemShelfForWork(ctx, tx, userID, req.WorkID)
		if err != nil {
			return nil, err
		}
		if existingItem != nil && existingShelf.ID != shelfID {
			if _, err := s.repo.DeleteShelfItem(ctx, tx, existingItem.ID); err != nil {
				return nil, err
			}
			if err := s.repo.IncrementItemCount(ctx, tx, existingShelf.ID, -1); err != nil {
				return nil, err
			}
		} else if existingItem != nil && existingShelf.ID == shelfID {
			return nil, repository.ErrDuplicateItem
		}
	}

	now := time.Now().UTC()
	item := &repository.ShelfItem{
		ID:        ulid.MustNew(ulid.Now(), rand.Reader).String(),
		ShelfID:   shelfID,
		UserID:    userID,
		WorkID:    req.WorkID,
		EditionID: req.EditionID,
		DateAdded: now,
	}

	if err := s.repo.CreateShelfItem(ctx, tx, item); err != nil {
		return nil, err
	}
	if err := s.repo.IncrementItemCount(ctx, tx, shelfID, 1); err != nil {
		return nil, err
	}

	payload, _ := json.Marshal(map[string]any{
		"userId":    userID,
		"shelfId":   shelfID,
		"workId":    req.WorkID,
		"editionId": req.EditionID,
		"shelfType": shelf.ShelfType,
	})
	env := envelope.New(ctx, shelfItemAddedEvt, serviceName, userID, payload)
	if err := outbox.InsertEvent(ctx, tx, aggregateShelfItem, item.ID, env); err != nil {
		return nil, err
	}

	return item, tx.Commit(ctx)
}

func (s *LibraryService) RemoveItemFromShelf(ctx context.Context, userID, shelfID, itemID string) error {
	shelf, err := s.repo.GetShelfByID(ctx, shelfID)
	if err != nil {
		return err
	}
	if shelf.UserID != userID {
		return repository.ErrNotFound
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	deleted, err := s.repo.DeleteShelfItem(ctx, tx, itemID)
	if err != nil {
		return err
	}
	if deleted.ShelfID != shelfID {
		return repository.ErrNotFound
	}

	if err := s.repo.IncrementItemCount(ctx, tx, shelfID, -1); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *LibraryService) ListShelfItems(ctx context.Context, userID, shelfID, cursor string, limit int) ([]*repository.ShelfItem, error) {
	shelf, err := s.repo.GetShelfByID(ctx, shelfID)
	if err != nil {
		return nil, err
	}
	if shelf.UserID != userID {
		return nil, repository.ErrNotFound
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return s.repo.ListShelfItems(ctx, shelfID, cursor, limit)
}

func (s *LibraryService) CreateDefaultShelves(ctx context.Context, userID string) error {
	now := time.Now().UTC()
	defaults := []struct {
		name         string
		slug         string
		shelfType    string
		displayOrder int
	}{
		{"Want to Read", "want-to-read", "want_to_read", 0},
		{"Reading", "reading", "reading", 1},
		{"Read", "read", "read", 2},
		{"Did Not Finish", "dnf", "dnf", 3},
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, d := range defaults {
		shelf := &repository.Shelf{
			ID:           ulid.MustNew(ulid.Now(), rand.Reader).String(),
			UserID:       userID,
			Name:         d.name,
			Slug:         d.slug,
			ShelfType:    d.shelfType,
			IsSystem:     true,
			IsPrivate:    false,
			DisplayOrder: d.displayOrder,
			ItemCount:    0,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if err := s.repo.CreateShelf(ctx, tx, shelf); err != nil {
			if errors.Is(err, repository.ErrDuplicateShelf) {
				continue
			}
			return err
		}
	}
	return tx.Commit(ctx)
}

func generateSlug(name string) string {
	slug := strings.ToLower(strings.TrimSpace(name))
	slug = slugRegex.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if len(slug) > 100 {
		slug = slug[:100]
	}
	return slug
}
