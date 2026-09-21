package httpx

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/gtrirf/start-and-found/api/internal/platform/apierr"
	"github.com/gtrirf/start-and-found/api/internal/platform/ctxauth"
)

// Auth carries the middleware a domain handler uses to protect its routes.
//
// The fields are plain net/http middleware, so a domain never imports the auth
// package: the composition root (internal/app) wires the real implementation in.
type Auth struct {
	Require  func(http.Handler) http.Handler
	Optional func(http.Handler) http.Handler
}

// CallerID returns the authenticated user id of a request. Handlers mounted
// behind RequireAuth can rely on it being present.
func CallerID(r *http.Request) (uuid.UUID, error) {
	id, ok := ctxauth.ID(r.Context())
	if !ok {
		return uuid.Nil, apierr.Unauthorized("authentication is required")
	}
	return id, nil
}
