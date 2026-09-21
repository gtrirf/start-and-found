package projects

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/gtrirf/start-and-found/api/internal/platform/httpx"
)

// Handler exposes the projects HTTP API.
type Handler struct {
	service *Service
	auth    httpx.Auth
}

// NewHandler builds the projects handler.
func NewHandler(service *Service, auth httpx.Auth) *Handler {
	return &Handler{service: service, auth: auth}
}

// Routes returns the project endpoints, mounted at /v1/projects.
//
// Projects are addressed by owner username and slug (the same coordinates as
// their handle @owner/slug), which keeps URLs human readable. Reads are public,
// writes require authentication and ownership.
func (h *Handler) Routes() chi.Router {
	router := chi.NewRouter()
	router.Use(h.auth.Optional)

	router.Get("/{owner}/{slug}", h.get)
	router.Get("/{owner}/{slug}/members", h.members)

	router.Group(func(router chi.Router) {
		router.Use(h.auth.Require)
		router.Post("/", h.create)
		router.Patch("/{owner}/{slug}", h.update)
		router.Delete("/{owner}/{slug}", h.delete)
		router.Post("/{owner}/{slug}/members", h.addMember)
		router.Delete("/{owner}/{slug}/members/{userID}", h.removeMember)
	})
	return router
}

// get handles GET /v1/projects/{owner}/{slug}.
func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	owner, slug, err := projectPath(r)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	project, err := h.service.Get(r.Context(), owner, slug)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, project)
}

// create handles POST /v1/projects.
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	callerID, err := httpx.CallerID(r)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	input, err := httpx.Decode[CreateInput](r)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	project, publisher, err := h.service.Create(r.Context(), callerID, input)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, map[string]any{"project": project, "publisher": publisher})
}

// update handles PATCH /v1/projects/{owner}/{slug}.
func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	owner, slug, err := projectPath(r)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	callerID, err := httpx.CallerID(r)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	input, err := httpx.Decode[UpdateInput](r)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	project, err := h.service.Get(r.Context(), owner, slug)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	updated, err := h.service.Update(r.Context(), project.ID, callerID, input)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, updated)
}

// delete handles DELETE /v1/projects/{owner}/{slug}.
func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	owner, slug, err := projectPath(r)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	callerID, err := httpx.CallerID(r)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	project, err := h.service.Get(r.Context(), owner, slug)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	if err := h.service.Delete(r.Context(), project.ID, callerID); err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.NoContent(w)
}

// members handles GET /v1/projects/{owner}/{slug}/members.
func (h *Handler) members(w http.ResponseWriter, r *http.Request) {
	owner, slug, err := projectPath(r)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	project, err := h.service.Get(r.Context(), owner, slug)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	members, err := h.service.Members(r.Context(), project.ID)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": members})
}

// addMember handles POST /v1/projects/{owner}/{slug}/members.
func (h *Handler) addMember(w http.ResponseWriter, r *http.Request) {
	owner, slug, err := projectPath(r)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	callerID, err := httpx.CallerID(r)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	input, err := httpx.Decode[AddMemberInput](r)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	project, err := h.service.Get(r.Context(), owner, slug)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	members, err := h.service.AddMember(r.Context(), project.ID, callerID, input)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": members})
}

// removeMember handles DELETE /v1/projects/{owner}/{slug}/members/{userID}.
func (h *Handler) removeMember(w http.ResponseWriter, r *http.Request) {
	owner, slug, err := projectPath(r)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	callerID, err := httpx.CallerID(r)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	memberID, err := httpx.PathUUID(r, "userID")
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	project, err := h.service.Get(r.Context(), owner, slug)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	members, err := h.service.RemoveMember(r.Context(), project.ID, callerID, memberID)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": members})
}

func projectPath(r *http.Request) (owner string, slug string, err error) {
	owner, err = httpx.PathParam(r, "owner")
	if err != nil {
		return "", "", err
	}
	slug, err = httpx.PathParam(r, "slug")
	if err != nil {
		return "", "", err
	}
	return owner, slug, nil
}
