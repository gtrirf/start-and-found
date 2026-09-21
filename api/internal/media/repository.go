package media

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/gtrirf/start-and-found/api/internal/platform/apierr"
	"github.com/gtrirf/start-and-found/api/internal/platform/database"
)

const selectMedia = `
SELECT id, owner_user_id, kind, mime_type, filename, size_bytes, width, height, storage_key, status, created_at, updated_at
FROM media`

// Repository persists media metadata.
type Repository struct {
	q database.Querier
}

// NewRepository builds a repository on top of the given querier.
func NewRepository(q database.Querier) *Repository {
	return &Repository{q: q}
}

// Create stores the metadata of a pending upload.
func (r *Repository) Create(ctx context.Context, item *Media) error {
	const query = `
INSERT INTO media (id, owner_user_id, kind, mime_type, filename, size_bytes, storage_key, status)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING created_at, updated_at`

	err := r.q.QueryRow(ctx, query,
		item.ID, item.OwnerUserID, item.Kind, item.MimeType, item.Filename, item.SizeBytes, item.StorageKey, item.Status,
	).Scan(&item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create media: %w", err)
	}
	return nil
}

// ByID loads media metadata.
func (r *Repository) ByID(ctx context.Context, id uuid.UUID) (Media, error) {
	item, err := scanMedia(r.q.QueryRow(ctx, selectMedia+` WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Media{}, apierr.NotFound("media not found")
	}
	if err != nil {
		return Media{}, fmt.Errorf("load media: %w", err)
	}
	return item, nil
}

// MarkUploaded records the stored size and flips the row to "uploaded".
func (r *Repository) MarkUploaded(ctx context.Context, id uuid.UUID, sizeBytes int64) (Media, error) {
	const query = `
UPDATE media
SET status = $2, size_bytes = $3
WHERE id = $1
RETURNING id, owner_user_id, kind, mime_type, filename, size_bytes, width, height, storage_key, status, created_at, updated_at`

	item, err := scanMedia(r.q.QueryRow(ctx, query, id, StatusUploaded, sizeBytes))
	if errors.Is(err, pgx.ErrNoRows) {
		return Media{}, apierr.NotFound("media not found")
	}
	if err != nil {
		return Media{}, fmt.Errorf("mark media uploaded: %w", err)
	}
	return item, nil
}

// Delete removes media metadata.
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	if _, err := r.q.Exec(ctx, `DELETE FROM media WHERE id = $1`, id); err != nil {
		return fmt.Errorf("delete media: %w", err)
	}
	return nil
}

func scanMedia(row scanner) (Media, error) {
	var item Media
	err := row.Scan(
		&item.ID,
		&item.OwnerUserID,
		&item.Kind,
		&item.MimeType,
		&item.Filename,
		&item.SizeBytes,
		&item.Width,
		&item.Height,
		&item.StorageKey,
		&item.Status,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	return item, err
}

type scanner interface {
	Scan(dest ...any) error
}
