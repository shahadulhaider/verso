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

func (r *Repo) CreateShelf(ctx context.Context, tx pgx.Tx, s *Shelf) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO shelf (id, user_id, name, slug, shelf_type, is_system, is_private, display_order, item_count, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		s.ID, s.UserID, s.Name, s.Slug, s.ShelfType, s.IsSystem, s.IsPrivate,
		s.DisplayOrder, s.ItemCount, s.CreatedAt, s.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrDuplicateShelf
		}
		return err
	}
	return nil
}

func (r *Repo) ListShelves(ctx context.Context, userID string) ([]*Shelf, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, name, slug, shelf_type, is_system, is_private, display_order, item_count, created_at, updated_at
		 FROM shelf WHERE user_id = $1 ORDER BY display_order ASC, created_at ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectShelves(rows)
}

func (r *Repo) GetShelfByID(ctx context.Context, shelfID string) (*Shelf, error) {
	s := &Shelf{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, name, slug, shelf_type, is_system, is_private, display_order, item_count, created_at, updated_at
		 FROM shelf WHERE id = $1`, shelfID,
	).Scan(&s.ID, &s.UserID, &s.Name, &s.Slug, &s.ShelfType, &s.IsSystem, &s.IsPrivate,
		&s.DisplayOrder, &s.ItemCount, &s.CreatedAt, &s.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return s, err
}

func (r *Repo) GetSystemShelfForWork(ctx context.Context, tx pgx.Tx, userID, workID string) (*ShelfItem, *Shelf, error) {
	item := &ShelfItem{}
	shelf := &Shelf{}
	err := tx.QueryRow(ctx,
		`SELECT si.id, si.shelf_id, si.user_id, si.work_id, si.edition_id, si.date_added,
		        s.id, s.user_id, s.name, s.slug, s.shelf_type, s.is_system, s.is_private, s.display_order, s.item_count, s.created_at, s.updated_at
		 FROM shelf_item si JOIN shelf s ON s.id = si.shelf_id
		 WHERE si.user_id = $1 AND si.work_id = $2 AND s.is_system = TRUE`, userID, workID,
	).Scan(&item.ID, &item.ShelfID, &item.UserID, &item.WorkID, &item.EditionID, &item.DateAdded,
		&shelf.ID, &shelf.UserID, &shelf.Name, &shelf.Slug, &shelf.ShelfType, &shelf.IsSystem,
		&shelf.IsPrivate, &shelf.DisplayOrder, &shelf.ItemCount, &shelf.CreatedAt, &shelf.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	return item, shelf, nil
}

func (r *Repo) CreateShelfItem(ctx context.Context, tx pgx.Tx, item *ShelfItem) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO shelf_item (id, shelf_id, user_id, work_id, edition_id, date_added, date_started, date_finished, display_order, notes)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		item.ID, item.ShelfID, item.UserID, item.WorkID, item.EditionID,
		item.DateAdded, item.DateStarted, item.DateFinished, item.DisplayOrder, item.Notes,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrDuplicateItem
		}
		return err
	}
	return nil
}

func (r *Repo) DeleteShelfItem(ctx context.Context, tx pgx.Tx, itemID string) (*ShelfItem, error) {
	item := &ShelfItem{}
	err := tx.QueryRow(ctx,
		`DELETE FROM shelf_item WHERE id = $1
		 RETURNING id, shelf_id, user_id, work_id, edition_id, date_added`,
		itemID,
	).Scan(&item.ID, &item.ShelfID, &item.UserID, &item.WorkID, &item.EditionID, &item.DateAdded)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return item, err
}

func (r *Repo) IncrementItemCount(ctx context.Context, tx pgx.Tx, shelfID string, delta int) error {
	_, err := tx.Exec(ctx,
		`UPDATE shelf SET item_count = item_count + $1, updated_at = NOW() WHERE id = $2`,
		delta, shelfID)
	return err
}

func (r *Repo) ListShelfItems(ctx context.Context, shelfID, cursor string, limit int) ([]*ShelfItem, error) {
	var rows pgx.Rows
	var err error
	if cursor != "" {
		cursorTime, parseErr := time.Parse(time.RFC3339Nano, cursor)
		if parseErr != nil {
			return nil, parseErr
		}
		rows, err = r.pool.Query(ctx,
			`SELECT id, shelf_id, user_id, work_id, edition_id, date_added, date_started, date_finished, display_order, notes
			 FROM shelf_item WHERE shelf_id = $1 AND date_added < $2 ORDER BY date_added DESC LIMIT $3`,
			shelfID, cursorTime, limit)
	} else {
		rows, err = r.pool.Query(ctx,
			`SELECT id, shelf_id, user_id, work_id, edition_id, date_added, date_started, date_finished, display_order, notes
			 FROM shelf_item WHERE shelf_id = $1 ORDER BY date_added DESC LIMIT $2`,
			shelfID, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectShelfItems(rows)
}

func (r *Repo) CreateReadingSession(ctx context.Context, tx pgx.Tx, s *ReadingSession) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO reading_session (id, user_id, format_id, work_id, started_at, ended_at, duration_seconds, progress_before, progress_after, pages_read, device_type, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		s.ID, s.UserID, s.FormatID, s.WorkID, s.StartedAt, s.EndedAt,
		s.DurationSeconds, s.ProgressBefore, s.ProgressAfter, s.PagesRead, s.DeviceType, s.CreatedAt,
	)
	return err
}

func (r *Repo) GetReadingProgress(ctx context.Context, userID, workID string) (*ReadingProgress, error) {
	p := &ReadingProgress{}
	err := r.pool.QueryRow(ctx,
		`SELECT user_id, work_id, current_format_id, progress_percent, current_page, status, started_at, completed_at, read_count, updated_at
		 FROM reading_progress WHERE user_id = $1 AND work_id = $2`, userID, workID,
	).Scan(&p.UserID, &p.WorkID, &p.CurrentFormatID, &p.ProgressPercent, &p.CurrentPage,
		&p.Status, &p.StartedAt, &p.CompletedAt, &p.ReadCount, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return p, err
}

func (r *Repo) UpsertReadingProgress(ctx context.Context, tx pgx.Tx, p *ReadingProgress) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO reading_progress (user_id, work_id, current_format_id, progress_percent, current_page, status, started_at, completed_at, read_count, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		 ON CONFLICT (user_id, work_id) DO UPDATE SET
			current_format_id = EXCLUDED.current_format_id,
			progress_percent = EXCLUDED.progress_percent,
			current_page = EXCLUDED.current_page,
			status = EXCLUDED.status,
			started_at = COALESCE(reading_progress.started_at, EXCLUDED.started_at),
			completed_at = EXCLUDED.completed_at,
			read_count = EXCLUDED.read_count,
			updated_at = EXCLUDED.updated_at`,
		p.UserID, p.WorkID, p.CurrentFormatID, p.ProgressPercent, p.CurrentPage,
		p.Status, p.StartedAt, p.CompletedAt, p.ReadCount, p.UpdatedAt,
	)
	return err
}

func (r *Repo) ShelfExists(ctx context.Context, shelfID string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM shelf WHERE id = $1)`, shelfID).Scan(&exists)
	return exists, err
}

func collectShelves(rows pgx.Rows) ([]*Shelf, error) {
	var shelves []*Shelf
	for rows.Next() {
		s := &Shelf{}
		if err := rows.Scan(&s.ID, &s.UserID, &s.Name, &s.Slug, &s.ShelfType, &s.IsSystem,
			&s.IsPrivate, &s.DisplayOrder, &s.ItemCount, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		s.ID = strings.TrimSpace(s.ID)
		shelves = append(shelves, s)
	}
	return shelves, rows.Err()
}

func collectShelfItems(rows pgx.Rows) ([]*ShelfItem, error) {
	var items []*ShelfItem
	for rows.Next() {
		i := &ShelfItem{}
		if err := rows.Scan(&i.ID, &i.ShelfID, &i.UserID, &i.WorkID, &i.EditionID,
			&i.DateAdded, &i.DateStarted, &i.DateFinished, &i.DisplayOrder, &i.Notes); err != nil {
			return nil, err
		}
		i.ID = strings.TrimSpace(i.ID)
		items = append(items, i)
	}
	return items, rows.Err()
}
