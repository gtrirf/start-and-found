package auth

import (
	"github.com/go-chi/chi/v5"
)

// Routes returns the auth endpoints. The composition root mounts them at
// /v1/auth behind a rate limiter, because they are the only unauthenticated
// write paths of the API.
func (h *Handler) Routes() chi.Router {
	router := chi.NewRouter()
	router.Post("/signup", h.signup)
	router.Post("/login", h.login)
	router.Post("/refresh", h.refresh)
	router.Post("/logout", h.logout)
	return router
}
