package media

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/gtrirf/start-and-found/api/internal/platform/httpx"
)

// Handler exposes the media HTTP API.
type Handler struct {
	service *Service
	auth    httpx.Auth
}

// NewHandler builds the media handler.
func NewHandler(service *Service, auth httpx.Auth) *Handler {
	return &Handler{service: service, auth: auth}
}

// Routes returns the media endpoints, mounted at /v1/media.
func (h *Handler) Routes() chi.Router {
	router := chi.NewRouter()
	router.Use(h.auth.Optional)

	router.Get("/{mediaID}", h.get)

	router.Group(func(router chi.Router) {
		router.Use(h.auth.Require)
		router.Post("/presign", h.presign)
		router.Post("/{mediaID}/complete", h.complete)
	})
	return router
}

// presign handles POST /v1/media/presign.
func (h *Handler) presign(w http.ResponseWriter, r *http.Request) {
	callerID, err := httpx.CallerID(r)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	input, err := httpx.Decode[PresignInput](r)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	upload, err := h.service.Presign(r.Context(), callerID, input)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, upload)
}

// complete handles POST /v1/media/{mediaID}/complete.
func (h *Handler) complete(w http.ResponseWriter, r *http.Request) {
	callerID, err := httpx.CallerID(r)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	mediaID, err := httpx.PathUUID(r, "mediaID")
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	item, err := h.service.Complete(r.Context(), callerID, mediaID)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

// get handles GET /v1/media/{mediaID}.
func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	mediaID, err := httpx.PathUUID(r, "mediaID")
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	item, err := h.service.ByID(r.Context(), mediaID)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}
