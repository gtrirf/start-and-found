package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/gtrirf/start-and-found/api/internal/platform/apierr"
	"github.com/gtrirf/start-and-found/api/internal/platform/database"
	"github.com/gtrirf/start-and-found/api/internal/platform/ids"
	"github.com/gtrirf/start-and-found/api/internal/platform/validate"
	"github.com/gtrirf/start-and-found/api/internal/publishers"
	"github.com/gtrirf/start-and-found/api/internal/users"
)

// Service implements registration, login and session rotation.
type Service struct {
	db         *database.DB
	users      *users.Repository
	sessions   *SessionsRepository
	publishers *publishers.Repository
	tokens     *TokenManager
	refreshTTL time.Duration
	now        func() time.Time
}

// NewService wires the auth service.
func NewService(
	db *database.DB,
	usersRepo *users.Repository,
	sessionsRepo *SessionsRepository,
	publishersRepo *publishers.Repository,
	tokens *TokenManager,
	refreshTTL time.Duration,
) *Service {
	return &Service{
		db:         db,
		users:      usersRepo,
		sessions:   sessionsRepo,
		publishers: publishersRepo,
		tokens:     tokens,
		refreshTTL: refreshTTL,
		now:        time.Now,
	}
}

// Signup registers an account, creates its personal publisher and opens a
// session, all inside one transaction.
func (s *Service) Signup(ctx context.Context, input SignupInput, meta RequestMeta) (Session, error) {
	username := strings.ToLower(strings.TrimSpace(input.Username))
	if err := validate.Username(username); err != nil {
		return Session{}, err
	}
	email := strings.ToLower(strings.TrimSpace(input.Email))
	if err := validate.Email(email); err != nil {
		return Session{}, err
	}
	displayName := strings.TrimSpace(input.DisplayName)
	if displayName == "" {
		displayName = username
	}
	if err := validate.DisplayName(displayName); err != nil {
		return Session{}, err
	}
	if err := validate.Password(input.Password); err != nil {
		return Session{}, err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return Session{}, fmt.Errorf("hash password: %w", err)
	}

	user := &users.User{
		ID:           ids.New(),
		Username:     username,
		Email:        email,
		DisplayName:  displayName,
		PasswordHash: string(passwordHash),
	}
	refreshToken, refreshHash, err := NewRefreshToken()
	if err != nil {
		return Session{}, err
	}
	refreshExpiresAt := s.now().Add(s.refreshTTL)

	err = s.db.WithTx(ctx, func(q database.Querier) error {
		if err := users.NewRepository(q).Create(ctx, user); err != nil {
			return err
		}
		if _, err := publishers.NewRepository(q).EnsureForUser(ctx, user.ID); err != nil {
			return err
		}
		return NewSessionsRepository(q).Create(ctx, SessionRow{
			ID:        ids.New(),
			UserID:    user.ID,
			TokenHash: refreshHash,
			UserAgent: meta.UserAgent,
			IP:        meta.IP,
			ExpiresAt: refreshExpiresAt,
		})
	})
	if err != nil {
		return Session{}, err
	}

	return s.issue(*user, refreshToken, refreshExpiresAt)
}

// Login verifies credentials and opens a session. Usernames and email addresses
// are both accepted as the identifier.
func (s *Service) Login(ctx context.Context, input LoginInput, meta RequestMeta) (Session, error) {
	identifier := strings.ToLower(strings.TrimSpace(input.Identifier))
	if identifier == "" {
		return Session{}, apierr.Validation("username or email is required", map[string]any{"field": "identifier"})
	}
	if input.Password == "" {
		return Session{}, apierr.Validation("password is required", map[string]any{"field": "password"})
	}

	user, err := s.users.ByIdentifier(ctx, identifier)
	if err != nil {
		if isNotFound(err) {
			// The same message is returned for unknown accounts and wrong
			// passwords so the endpoint cannot be used to enumerate users.
			return Session{}, apierr.Unauthorized("username/email or password is incorrect")
		}
		return Session{}, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		return Session{}, apierr.Unauthorized("username/email or password is incorrect")
	}

	refreshToken, refreshHash, err := NewRefreshToken()
	if err != nil {
		return Session{}, err
	}
	refreshExpiresAt := s.now().Add(s.refreshTTL)

	if err := s.sessions.Create(ctx, SessionRow{
		ID:        ids.New(),
		UserID:    user.ID,
		TokenHash: refreshHash,
		UserAgent: meta.UserAgent,
		IP:        meta.IP,
		ExpiresAt: refreshExpiresAt,
	}); err != nil {
		return Session{}, err
	}

	return s.issue(user, refreshToken, refreshExpiresAt)
}

// issue signs the access token and assembles the session returned to a client.
func (s *Service) issue(user users.User, refreshToken string, refreshExpiresAt time.Time) (Session, error) {
	accessToken, accessExpiresAt, err := s.tokens.Issue(user.ID, user.Username)
	if err != nil {
		return Session{}, err
	}
	return Session{
		User:             user,
		AccessToken:      accessToken,
		AccessExpiresAt:  accessExpiresAt,
		RefreshToken:     refreshToken,
		RefreshExpiresAt: refreshExpiresAt,
	}, nil
}

func isNotFound(err error) bool {
	return apierr.From(err).Status == http.StatusNotFound
}

func isUnauthorized(err error) bool {
	return apierr.From(err).Status == http.StatusUnauthorized
}

// Refresh rotates a refresh token: the presented session is revoked and a new
// one is created. Rotation keeps a stolen token usable only once.
func (s *Service) Refresh(ctx context.Context, refreshToken string, meta RequestMeta) (Session, error) {
	if strings.TrimSpace(refreshToken) == "" {
		return Session{}, apierr.Unauthorized("refresh token is required")
	}

	existing, err := s.sessions.ByTokenHash(ctx, HashRefreshToken(refreshToken))
	if err != nil {
		return Session{}, err
	}
	user, err := s.users.ByID(ctx, existing.UserID)
	if err != nil {
		return Session{}, err
	}

	newToken, newHash, err := NewRefreshToken()
	if err != nil {
		return Session{}, err
	}
	refreshExpiresAt := s.now().Add(s.refreshTTL)

	err = s.db.WithTx(ctx, func(q database.Querier) error {
		repo := NewSessionsRepository(q)
		if err := repo.Revoke(ctx, existing.ID); err != nil {
			return err
		}
		return repo.Create(ctx, SessionRow{
			ID:        ids.New(),
			UserID:    user.ID,
			TokenHash: newHash,
			UserAgent: meta.UserAgent,
			IP:        meta.IP,
			ExpiresAt: refreshExpiresAt,
		})
	})
	if err != nil {
		return Session{}, err
	}

	return s.issue(user, newToken, refreshExpiresAt)
}

// Logout revokes the session behind a refresh token. It is idempotent: logging
// out twice, or with an unknown token, still succeeds.
func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	if strings.TrimSpace(refreshToken) == "" {
		return nil
	}
	session, err := s.sessions.ByTokenHash(ctx, HashRefreshToken(refreshToken))
	if err != nil {
		if isUnauthorized(err) {
			return nil
		}
		return err
	}
	return s.sessions.Revoke(ctx, session.ID)
}

// Authenticate resolves an access token into the user it belongs to. It is used
// by the middleware on every authenticated request.
func (s *Service) Authenticate(ctx context.Context, accessToken string) (users.User, error) {
	claims, err := s.tokens.Verify(accessToken)
	if err != nil {
		return users.User{}, err
	}
	userID, err := claims.UserID()
	if err != nil {
		return users.User{}, err
	}
	return s.users.ByID(ctx, userID)
}

// UserByID loads the account behind an id. The users domain reuses it for
// authenticated endpoints such as GET /v1/me.
func (s *Service) UserByID(ctx context.Context, id uuid.UUID) (users.User, error) {
	return s.users.ByID(ctx, id)
}
