package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/gtrirf/start-and-found/api/internal/platform/apierr"
	"github.com/gtrirf/start-and-found/api/internal/platform/database"
)

// Session is one stored refresh session.
type Session struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	UserAgent string
	IP        string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}

// SessionsRepository persists refresh sessions.
type SessionsRepository struct {
	q database.Querier
}

// NewSessionsRepository builds a repository on top of the given querier.
func NewSessionsRepository(q database.Querier) *SessionsRepository {
	return &SessionsRepository{q: q}
}

// Create stores a refresh session.
func (r *SessionsRepository) Create(ctx context.Context, session Session) error {
	const query = `
INSERT INTO sessions (id, user_id, token_hash, user_agent, ip, expires_at)
VALUES ($1, $2, $3, $4, $5, $6)`

	if _, err := r.q.Exec(ctx, query,
		session.ID, session.UserID, session.TokenHash, session.UserAgent, session.IP, session.ExpiresAt,
	); err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

// ByTokenHash loads an active session, rejecting revoked and expired ones.
func (r *SessionsRepository) ByTokenHash(ctx context.Context, tokenHash string) (Session, error) {
	const query = `
SELECT id, user_id, token_hash, user_agent, ip, expires_at, revoked_at, created_at
FROM sessions
WHERE token_hash = $1`

	session, err := scanSession(r.q.QueryRow(ctx, query, tokenHash))
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, apierr.Unauthorized("refresh token is invalid")
	}
	if err != nil {
		return Session{}, fmt.Errorf("load session: %w", err)
	}
	if session.RevokedAt != nil {
		return Session{}, apierr.Unauthorized("refresh token was revoked")
	}
	if time.Now().After(session.ExpiresAt) {
		return Session{}, apierr.Unauthorized("refresh token has expired")
	}
	return session, nil
}

// Revoke marks a session as revoked.
func (r *SessionsRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	if _, err := r.q.Exec(ctx, `UPDATE sessions SET revoked_at = now() WHERE id = $1 AND revoked_at IS NULL`, id); err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	return nil
}

// RevokeAllForUser revokes every session of a user, for example after a
// password change.
func (r *SessionsRepository) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	if _, err := r.q.Exec(ctx, `UPDATE sessions SET revoked_at = now() WHERE user_id = $1 AND revoked_at IS NULL`, userID); err != nil {
		return fmt.Errorf("revoke user sessions: %w", err)
	}
	return nil
}

// DeleteExpired removes sessions that expired before the given time.
func (r *SessionsRepository) DeleteExpired(ctx context.Context, before time.Time) (int64, error) {
	tag, err := r.q.Exec(ctx, `DELETE FROM sessions WHERE expires_at < $1`, before)
	if err != nil {
		return 0, fmt.Errorf("delete expired sessions: %w", err)
	}
	return tag.RowsAffected(), nil
}

func scanSession(row scanner) (Session, error) {
	var session Session
	err := row.Scan(
		&session.ID,
		&session.UserID,
		&session.TokenHash,
		&session.UserAgent,
		&session.IP,
		&session.ExpiresAt,
		&session.RevokedAt,
		&session.CreatedAt,
	)
	return session, err
}

type scanner interface {
	Scan(dest ...any) error
}
