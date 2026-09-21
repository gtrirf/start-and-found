package posts

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/gtrirf/start-and-found/api/internal/platform/httpx"
)

// Handler exposes the posts HTTP API.
type Handler struct {
	service *Service
	auth    httpx.Auth
}

// NewHandler builds the posts handler.
func NewHandler(service *Service, auth httpx.Auth) *Handler {
	return &Handler{service: service, auth: auth}
}

// Routes returns the post endpoints, mounted at /v1/posts.
func (h *Handler) Routes() chi.Router {
	router := chi.NewRouter()
	router.Use(h.auth.Optional)

	router.Get("/{postID}", h.get)
	router.Get("/{postID}/replies", h.replies)

	router.Group(func(router chi.Router) {
		router.Use(h.auth.Require)
		router.Post("/", h.create)
		router.Patch("/{postID}", h.update)
		router.Delete("/{postID}", h.delete)
		router.Post("/{postID}/replies", h.reply)
	})
	return router
}

// FeedRoutes returns the feed endpoint, mounted at /v1/feed.
func (h *Handler) FeedRoutes() chi.Router {
	router := chi.NewRouter()
	router.Use(h.auth.Optional)
	router.Get("/", h.feed)
	return router
}

// feed handles GET /v1/feed.
func (h *Handler) feed(w http.ResponseWriter, r *http.Request) {
	limit, cursor, err := httpx.PageParams(r)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	page, err := h.service.Feed(r.Context(), limit, cursor)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, page)
}

// get handles GET /v1/posts/{postID}.
func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	postID, err := httpx.PathUUID(r, "postID")
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	post, err := h.service.ByID(r.Context(), postID)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, post)
}

// create handles POST /v1/posts.
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

	post, err := h.service.Create(r.Context(), callerID, input)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, post)
}

// update handles PATCH /v1/posts/{postID}.
func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	postID, err := httpx.PathUUID(r, "postID")
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

	post, err := h.service.Update(r.Context(), postID, callerID, input)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, post)
}

// delete handles DELETE /v1/posts/{postID}.
func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	postID, err := httpx.PathUUID(r, "postID")
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	callerID, err := httpx.CallerID(r)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	if err := h.service.Delete(r.Context(), postID, callerID); err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.NoContent(w)
}

// replies handles GET /v1/posts/{postID}/replies.
func (h *Handler) replies(w http.ResponseWriter, r *http.Request) {
	postID, err := httpx.PathUUID(r, "postID")
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	limit, cursor, err := httpx.PageParams(r)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	page, err := h.service.Replies(r.Context(), postID, limit, cursor)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, page)
}

// reply handles POST /v1/posts/{postID}/replies.
func (h *Handler) reply(w http.ResponseWriter, r *http.Request) {
	postID, err := httpx.PathUUID(r, "postID")
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
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

	post, err := h.service.Reply(r.Context(), postID, callerID, input)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, post)
}
