package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/gtrirf/start-and-found/api/internal/platform/apierr"
	"github.com/gtrirf/start-and-found/api/internal/platform/ctxauth"
	"github.com/gtrirf/start-and-found/api/internal/users"
)

// Middleware authenticates requests and attaches the identity to the context.
type Middleware struct {
	service *Service
}

// NewMiddleware builds the auth middleware.
func NewMiddleware(service *Service) *Middleware {
	return &Middleware{service: service}
}

// RequireAuth rejects requests without a valid access token.
func (m *Middleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := bearerToken(r)
		if !ok {
			apierr.Write(w, r, apierr.Unauthorized("authorization header is missing"))
			return
		}

		user, err := m.service.Authenticate(r.Context(), token)
		if err != nil {
			apierr.Write(w, r, err)
			return
		}
		next.ServeHTTP(w, r.WithContext(withUser(r.Context(), user)))
	})
}

// OptionalAuth attaches the identity when a valid token is present and lets the
// request through either way.
func (m *Middleware) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := bearerToken(r)
		if !ok {
			next.ServeHTTP(w, r)
			return
		}
		user, err := m.service.Authenticate(r.Context(), token)
		if err != nil {
			// An invalid token on a public endpoint is not an error: the visitor
			// is simply treated as anonymous.
			next.ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(w, r.WithContext(withUser(r.Context(), user)))
	})
}

// User returns the authenticated user of a request, if any.
func User(ctx context.Context) (users.User, bool) {
	principal, ok := ctxauth.From(ctx)
	if !ok {
		return users.User{}, false
	}
	return users.User{
		ID:          principal.ID,
		Username:    principal.Username,
		DisplayName: principal.DisplayName,
		AvatarURL:   principal.AvatarURL,
	}, true
}

func withUser(ctx context.Context, user users.User) context.Context {
	return ctxauth.With(ctx, ctxauth.User{
		ID:          user.ID,
		Username:    user.Username,
		DisplayName: user.DisplayName,
		AvatarURL:   user.AvatarURL,
	})
}

func bearerToken(r *http.Request) (string, bool) {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if header == "" {
		return "", false
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}
	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", false
	}
	return token, true
}
