package users

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/gtrirf/start-and-found/api/internal/platform/apierr"
	"github.com/gtrirf/start-and-found/api/internal/platform/httpx"
)

// Handler exposes the users HTTP API.
type Handler struct {
	service *Service
	auth    httpx.Auth
}

// NewHandler builds the users handler. The auth middleware is injected by the
// composition root so this package stays decoupled from the auth domain.
func NewHandler(service *Service, auth httpx.Auth) *Handler {
	return &Handler{service: service, auth: auth}
}

// profile handles GET /v1/users/{username}.
func (h *Handler) profile(w http.ResponseWriter, r *http.Request) {
	username, err := httpx.PathParam(r, "username")
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	limit, cursor, err := httpx.PageParams(r)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	view, err := h.service.Profile(r.Context(), username, limit, cursor)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, view)
}

// projects handles GET /v1/users/{username}/projects.
func (h *Handler) projects(w http.ResponseWriter, r *http.Request) {
	username, err := httpx.PathParam(r, "username")
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	showcase, err := h.service.Projects(r.Context(), username)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": showcase})
}

// posts handles GET /v1/users/{username}/posts.
func (h *Handler) posts(w http.ResponseWriter, r *http.Request) {
	username, err := httpx.PathParam(r, "username")
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	scope, err := parseScope(r)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	limit, cursor, err := httpx.PageParams(r)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	page, err := h.service.Activity(r.Context(), username, scope, limit, cursor)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, page)
}

// me handles GET /v1/me.
func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	callerID, err := httpx.CallerID(r)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	account, err := h.service.Account(r.Context(), callerID)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, account)
}

// updateMe handles PATCH /v1/me.
func (h *Handler) updateMe(w http.ResponseWriter, r *http.Request) {
	callerID, err := httpx.CallerID(r)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	input, err := httpx.Decode[UpdateProfileInput](r)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	account, err := h.service.UpdateAccount(r.Context(), callerID, input)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, account)
}

// publishers handles GET /v1/me/publishers.
func (h *Handler) publishers(w http.ResponseWriter, r *http.Request) {
	callerID, err := httpx.CallerID(r)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	identities, err := h.service.Publishers(r.Context(), callerID)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": identities})
}

func parseScope(r *http.Request) (Scope, error) {
	switch Scope(httpx.Query(r, "scope")) {
	case "", ScopeAll:
		return ScopeAll, nil
	case ScopeUser:
		return ScopeUser, nil
	case ScopeProjects:
		return ScopeProjects, nil
	default:
		return "", apierr.BadRequest("scope must be one of: all, user, projects")
	}
}

// Routes returns the public profile endpoints, mounted at /v1/users.
func (h *Handler) Routes() chi.Router {
	router := chi.NewRouter()
	router.Use(h.auth.Optional)
	router.Get("/{username}", h.profile)
	router.Get("/{username}/projects", h.projects)
	router.Get("/{username}/posts", h.posts)
	return router
}

// MeRoutes returns the authenticated account endpoints, mounted at /v1/me.
func (h *Handler) MeRoutes() chi.Router {
	router := chi.NewRouter()
	router.Use(h.auth.Require)
	router.Get("/", h.me)
	router.Patch("/", h.updateMe)
	router.Get("/publishers", h.publishers)
	return router
}
