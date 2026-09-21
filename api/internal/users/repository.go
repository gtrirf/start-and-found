package users

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/gtrirf/start-and-found/api/internal/platform/apierr"
	"github.com/gtrirf/start-and-found/api/internal/platform/database"
)

const selectUser = `
SELECT id, username, email, display_name, avatar_url, bio, password_hash, created_at, updated_at
FROM users`

// Repository persists user accounts.
type Repository struct {
	q database.Querier
}

// NewRepository builds a repository on top of the given querier.
func NewRepository(q database.Querier) *Repository {
	return &Repository{q: q}
}

// Create inserts a new account. Uniqueness violations are translated into
// conflict errors so concurrent signups cannot create duplicates.
func (r *Repository) Create(ctx context.Context, user *User) error {
	const query = `
INSERT INTO users (id, username, email, display_name, avatar_url, bio, password_hash)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING created_at, updated_at`

	err := r.q.QueryRow(ctx, query,
		user.ID, user.Username, user.Email, user.DisplayName, user.AvatarURL, user.Bio, user.PasswordHash,
	).Scan(&user.CreatedAt, &user.UpdatedAt)
	switch {
	case err == nil:
		return nil
	case database.IsUniqueViolation(err) && strings.Contains(database.ConstraintName(err), "username"):
		return apierr.Conflict("username is already taken")
	case database.IsUniqueViolation(err) && strings.Contains(database.ConstraintName(err), "email"):
		return apierr.Conflict("email is already registered")
	case database.IsUniqueViolation(err):
		return apierr.Conflict("account already exists")
	default:
		return fmt.Errorf("create user: %w", err)
	}
}

// ByID loads an account by id.
func (r *Repository) ByID(ctx context.Context, id uuid.UUID) (User, error) {
	return r.scanOne(ctx, selectUser+` WHERE id = $1`, id)
}

// ByUsername loads an account by username. Matching is case insensitive because
// the column is citext.
func (r *Repository) ByUsername(ctx context.Context, username string) (User, error) {
	return r.scanOne(ctx, selectUser+` WHERE username = $1`, username)
}

// ByIdentifier loads an account by username or email, which lets one login form
// accept both.
func (r *Repository) ByIdentifier(ctx context.Context, identifier string) (User, error) {
	return r.scanOne(ctx, selectUser+` WHERE username = $1 OR email = $1`, identifier)
}

// UpdateProfile applies a partial profile update and returns the new state.
func (r *Repository) UpdateProfile(ctx context.Context, id uuid.UUID, input UpdateProfileInput) (User, error) {
	const query = `
UPDATE users
SET display_name = COALESCE($2, display_name),
    avatar_url = COALESCE($3, avatar_url),
    bio = COALESCE($4, bio)
WHERE id = $1
RETURNING id, username, email, display_name, avatar_url, bio, password_hash, created_at, updated_at`

	user, err := scanUser(r.q.QueryRow(ctx, query, id, input.DisplayName, input.AvatarURL, input.Bio))
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, apierr.NotFound("user not found")
	}
	if err != nil {
		return User{}, fmt.Errorf("update user profile: %w", err)
	}
	return user, nil
}

// UpdatePassword stores a new password hash.
func (r *Repository) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	const query = `UPDATE users SET password_hash = $2 WHERE id = $1`
	if _, err := r.q.Exec(ctx, query, id, passwordHash); err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	return nil
}

func scanUser(row scanner) (User, error) {
	var user User
	err := row.Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.DisplayName,
		&user.AvatarURL,
		&user.Bio,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	return user, err
}

func (r *Repository) scanOne(ctx context.Context, query string, args ...any) (User, error) {
	user, err := scanUser(r.q.QueryRow(ctx, query, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, apierr.NotFound("user not found")
	}
	if err != nil {
		return User{}, fmt.Errorf("load user: %w", err)
	}
	return user, nil
}

type scanner interface {
	Scan(dest ...any) error
}
