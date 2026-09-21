package threads

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/gtrirf/start-and-found/api/internal/platform/httpx"
)

// Handler exposes the threads HTTP API.
type Handler struct {
	service *Service
	auth    httpx.Auth
}

// NewHandler builds the threads handler.
func NewHandler(service *Service, auth httpx.Auth) *Handler {
	return &Handler{service: service, auth: auth}
}

// Routes returns the thread endpoints, mounted at /v1/threads.
func (h *Handler) Routes() chi.Router {
	router := chi.NewRouter()
	router.Use(h.auth.Optional)
	router.Get("/{postID}", h.get)
	return router
}

// get handles GET /v1/threads/{postID} and returns the full discussion tree.
func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	postID, err := httpx.PathUUID(r, "postID")
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	view, err := h.service.Thread(r.Context(), postID)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, view)
}
