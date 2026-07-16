package repository

import (
	"errors"
	"time"
)

var (
	ErrNotFound       = errors.New("not found")
	ErrDuplicateItem  = errors.New("duplicate shelf item")
	ErrDuplicateShelf = errors.New("duplicate shelf slug")
)

type Shelf struct {
	ID           string
	UserID       string
	Name         string
	Slug         string
	ShelfType    string
	IsSystem     bool
	IsPrivate    bool
	DisplayOrder int
	ItemCount    int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type ShelfItem struct {
	ID           string
	ShelfID      string
	UserID       string
	WorkID       string
	EditionID    *string
	DateAdded    time.Time
	DateStarted  *time.Time
	DateFinished *time.Time
	DisplayOrder *int
	Notes        *string
}

type ReadingSession struct {
	ID              string
	UserID          string
	FormatID        string
	WorkID          string
	StartedAt       time.Time
	EndedAt         *time.Time
	DurationSeconds *int
	ProgressBefore  float64
	ProgressAfter   float64
	PagesRead       *int
	DeviceType      *string
	CreatedAt       time.Time
}

type ReadingProgress struct {
	UserID          string
	WorkID          string
	CurrentFormatID string
	ProgressPercent float64
	CurrentPage     *int
	Status          string
	StartedAt       *time.Time
	CompletedAt     *time.Time
	ReadCount       int
	UpdatedAt       time.Time
}
