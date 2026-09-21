package publishers

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/gtrirf/start-and-found/api/internal/platform/apierr"
	"github.com/gtrirf/start-and-found/api/internal/platform/database"
)

// selectPublisher renders handles directly in SQL so every caller sees the same
// canonical form regardless of the case stored in the database.
const selectPublisher = `
SELECT p.id,
       p.kind,
       p.user_id,
       p.project_id,
       CASE WHEN p.kind = 'user' THEN '@' || u.username ELSE '@' || u.username || '/' || pr.slug END AS handle,
       p.created_at
FROM publishers p
LEFT JOIN projects pr ON pr.id = p.project_id
LEFT JOIN users u ON u.id = COALESCE(p.user_id, pr.owner_id)`

// Repository persists publishing identities.
type Repository struct {
	q database.Querier
}

// NewRepository builds a repository on top of the given querier, which may be a
// pool or a transaction.
func NewRepository(q database.Querier) *Repository {
	return &Repository{q: q}
}

// EnsureForUser returns the personal publisher of a user, creating it when the
// user was registered before publishers existed.
func (r *Repository) EnsureForUser(ctx context.Context, userID uuid.UUID) (Publisher, error) {
	const query = `
INSERT INTO publishers (id, kind, user_id)
VALUES ($1, 'user', $2)
ON CONFLICT (user_id) DO UPDATE SET user_id = EXCLUDED.user_id
RETURNING id`

	var id uuid.UUID
	if err := r.q.QueryRow(ctx, query, uuid.New(), userID).Scan(&id); err != nil {
		return Publisher{}, fmt.Errorf("ensure user publisher: %w", err)
	}
	return r.ByID(ctx, id)
}

// EnsureForProject returns the publisher of a project, creating it when missing.
func (r *Repository) EnsureForProject(ctx context.Context, projectID uuid.UUID) (Publisher, error) {
	const query = `
INSERT INTO publishers (id, kind, project_id)
VALUES ($1, 'project', $2)
ON CONFLICT (project_id) DO UPDATE SET project_id = EXCLUDED.project_id
RETURNING id`

	var id uuid.UUID
	if err := r.q.QueryRow(ctx, query, uuid.New(), projectID).Scan(&id); err != nil {
		return Publisher{}, fmt.Errorf("ensure project publisher: %w", err)
	}
	return r.ByID(ctx, id)
}

// ByID loads a publisher by primary key.
func (r *Repository) ByID(ctx context.Context, id uuid.UUID) (Publisher, error) {
	return r.scanOne(ctx, selectPublisher+` WHERE p.id = $1`, id)
}

// ByHandle resolves "@username" or "@username/project" into a publisher.
func (r *Repository) ByHandle(ctx context.Context, handle string) (Publisher, error) {
	username, projectSlug, err := ParseHandle(handle)
	if err != nil {
		return Publisher{}, err
	}
	if projectSlug == "" {
		return r.scanOne(ctx, selectPublisher+` WHERE p.kind = 'user' AND u.username = $1`, username)
	}
	return r.scanOne(ctx,
		selectPublisher+` WHERE p.kind = 'project' AND u.username = $1 AND pr.slug = $2`,
		username, projectSlug)
}

// ListForUser returns every publisher the user may publish as: their personal
// identity plus the projects they own or are a member of.
func (r *Repository) ListForUser(ctx context.Context, userID uuid.UUID) ([]Publisher, error) {
	const query = selectPublisher + `
LEFT JOIN project_members pm ON pm.project_id = pr.id AND pm.user_id = $1
WHERE (p.kind = 'user' AND p.user_id = $1)
   OR (p.kind = 'project' AND (pr.owner_id = $1 OR pm.user_id IS NOT NULL))
ORDER BY p.kind DESC, p.created_at`

	rows, err := r.q.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list publishers: %w", err)
	}
	defer rows.Close()

	result := make([]Publisher, 0)
	for rows.Next() {
		publisher, err := scanPublisher(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, publisher)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list publishers: %w", err)
	}
	return result, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanPublisher(row scanner) (Publisher, error) {
	var (
		publisher Publisher
		handle    *string
	)
	if err := row.Scan(&publisher.ID, &publisher.Kind, &publisher.UserID, &publisher.ProjectID, &handle, &publisher.CreatedAt); err != nil {
		return Publisher{}, err
	}
	if handle != nil {
		publisher.Handle = *handle
	}
	return publisher, nil
}

func (r *Repository) scanOne(ctx context.Context, query string, args ...any) (Publisher, error) {
	publisher, err := scanPublisher(r.q.QueryRow(ctx, query, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return Publisher{}, apierr.NotFound("publisher not found")
	}
	if err != nil {
		return Publisher{}, fmt.Errorf("load publisher: %w", err)
	}
	return publisher, nil
}
