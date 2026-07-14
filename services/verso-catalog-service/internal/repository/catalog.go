package repository

import (
	"context"
	"encoding/base64"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")

type Work struct {
	ID                      string     `json:"id"`
	Title                   string     `json:"title"`
	Description             *string    `json:"description,omitempty"`
	OriginalLanguage        *string    `json:"originalLanguage,omitempty"`
	OriginalPublicationYear *int       `json:"originalPublicationYear,omitempty"`
	AvgRating               float64    `json:"avgRating"`
	RatingsCount            int        `json:"ratingsCount"`
	ReviewsCount            int        `json:"reviewsCount"`
	CreatedAt               time.Time  `json:"createdAt"`
	UpdatedAt               time.Time  `json:"updatedAt"`
	DeletedAt               *time.Time `json:"-"`
	Version                 int        `json:"version"`
}

type Edition struct {
	ID              string     `json:"id"`
	WorkID          string     `json:"workId"`
	Title           *string    `json:"title,omitempty"`
	Language        *string    `json:"language,omitempty"`
	Publisher       *string    `json:"publisher,omitempty"`
	PublicationDate *time.Time `json:"publicationDate,omitempty"`
	PageCount       *int       `json:"pageCount,omitempty"`
	WordCount       *int       `json:"wordCount,omitempty"`
	CoverImageURL   *string    `json:"coverImageUrl,omitempty"`
	Description     *string    `json:"description,omitempty"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
	DeletedAt       *time.Time `json:"-"`
	Version         int        `json:"version"`
}

type Format struct {
	ID              string    `json:"id"`
	EditionID       string    `json:"editionId"`
	FormatType      string    `json:"formatType"`
	DurationSeconds *int      `json:"durationSeconds,omitempty"`
	FileSizeBytes   *int64    `json:"fileSizeBytes,omitempty"`
	DRMType         *string   `json:"drmType,omitempty"`
	FileFormat      *string   `json:"fileFormat,omitempty"`
	AssetURL        *string   `json:"assetUrl,omitempty"`
	IsAvailable     bool      `json:"isAvailable"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
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

func (r *Repo) CreateWork(ctx context.Context, tx pgx.Tx, w *Work) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO work (id, title, description, original_language, original_publication_year, avg_rating, ratings_count, reviews_count, created_at, updated_at, version)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		w.ID, w.Title, w.Description, w.OriginalLanguage, w.OriginalPublicationYear,
		w.AvgRating, w.RatingsCount, w.ReviewsCount, w.CreatedAt, w.UpdatedAt, w.Version,
	)
	return err
}

func (r *Repo) GetWorkByID(ctx context.Context, id string) (*Work, error) {
	w := &Work{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, title, description, original_language, original_publication_year,
		        avg_rating, ratings_count, reviews_count, created_at, updated_at, deleted_at, version
		 FROM work WHERE id = $1 AND deleted_at IS NULL`, id,
	).Scan(&w.ID, &w.Title, &w.Description, &w.OriginalLanguage, &w.OriginalPublicationYear,
		&w.AvgRating, &w.RatingsCount, &w.ReviewsCount, &w.CreatedAt, &w.UpdatedAt, &w.DeletedAt, &w.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return w, err
}

func (r *Repo) ListWorks(ctx context.Context, cursor string, limit int) ([]Work, string, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	var rows pgx.Rows
	var err error
	if cursor != "" {
		decoded, decErr := base64.URLEncoding.DecodeString(cursor)
		if decErr != nil {
			return nil, "", decErr
		}
		cursorID := string(decoded)
		rows, err = r.pool.Query(ctx,
			`SELECT id, title, description, original_language, original_publication_year,
			        avg_rating, ratings_count, reviews_count, created_at, updated_at, deleted_at, version
			 FROM work WHERE deleted_at IS NULL AND id > $1 ORDER BY id ASC LIMIT $2`,
			cursorID, limit+1)
	} else {
		rows, err = r.pool.Query(ctx,
			`SELECT id, title, description, original_language, original_publication_year,
			        avg_rating, ratings_count, reviews_count, created_at, updated_at, deleted_at, version
			 FROM work WHERE deleted_at IS NULL ORDER BY id ASC LIMIT $1`,
			limit+1)
	}
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	var works []Work
	for rows.Next() {
		var w Work
		if err := rows.Scan(&w.ID, &w.Title, &w.Description, &w.OriginalLanguage, &w.OriginalPublicationYear,
			&w.AvgRating, &w.RatingsCount, &w.ReviewsCount, &w.CreatedAt, &w.UpdatedAt, &w.DeletedAt, &w.Version); err != nil {
			return nil, "", err
		}
		works = append(works, w)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}

	var nextCursor string
	if len(works) > limit {
		works = works[:limit]
		nextCursor = base64.URLEncoding.EncodeToString([]byte(works[limit-1].ID))
	}
	return works, nextCursor, nil
}

func (r *Repo) ListEditionsByWorkID(ctx context.Context, workID string) ([]Edition, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, work_id, title, language, publisher, publication_date, page_count, word_count,
		        cover_image_url, description, created_at, updated_at, deleted_at, version
		 FROM edition WHERE work_id = $1 AND deleted_at IS NULL ORDER BY created_at ASC`, workID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var editions []Edition
	for rows.Next() {
		var e Edition
		if err := rows.Scan(&e.ID, &e.WorkID, &e.Title, &e.Language, &e.Publisher, &e.PublicationDate,
			&e.PageCount, &e.WordCount, &e.CoverImageURL, &e.Description,
			&e.CreatedAt, &e.UpdatedAt, &e.DeletedAt, &e.Version); err != nil {
			return nil, err
		}
		editions = append(editions, e)
	}
	return editions, rows.Err()
}

func (r *Repo) CreateEdition(ctx context.Context, tx pgx.Tx, e *Edition) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO edition (id, work_id, title, language, publisher, publication_date, page_count, word_count, cover_image_url, description, created_at, updated_at, version)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		e.ID, e.WorkID, e.Title, e.Language, e.Publisher, e.PublicationDate,
		e.PageCount, e.WordCount, e.CoverImageURL, e.Description,
		e.CreatedAt, e.UpdatedAt, e.Version,
	)
	return err
}

func (r *Repo) GetEditionByID(ctx context.Context, id string) (*Edition, error) {
	e := &Edition{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, work_id, title, language, publisher, publication_date, page_count, word_count,
		        cover_image_url, description, created_at, updated_at, deleted_at, version
		 FROM edition WHERE id = $1 AND deleted_at IS NULL`, id,
	).Scan(&e.ID, &e.WorkID, &e.Title, &e.Language, &e.Publisher, &e.PublicationDate,
		&e.PageCount, &e.WordCount, &e.CoverImageURL, &e.Description,
		&e.CreatedAt, &e.UpdatedAt, &e.DeletedAt, &e.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return e, err
}

func (r *Repo) ListFormatsByEditionID(ctx context.Context, editionID string) ([]Format, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, edition_id, format_type, duration_seconds, file_size_bytes, drm_type,
		        file_format, asset_url, is_available, created_at, updated_at
		 FROM format WHERE edition_id = $1 ORDER BY created_at ASC`, editionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var formats []Format
	for rows.Next() {
		var f Format
		if err := rows.Scan(&f.ID, &f.EditionID, &f.FormatType, &f.DurationSeconds, &f.FileSizeBytes,
			&f.DRMType, &f.FileFormat, &f.AssetURL, &f.IsAvailable, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, err
		}
		formats = append(formats, f)
	}
	return formats, rows.Err()
}

func (r *Repo) WorkExists(ctx context.Context, id string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM work WHERE id = $1 AND deleted_at IS NULL)`, id,
	).Scan(&exists)
	return exists, err
}

func EncodeCursor(id string) string {
	return base64.URLEncoding.EncodeToString([]byte(id))
}

func DecodeCursor(cursor string) (string, error) {
	decoded, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}
