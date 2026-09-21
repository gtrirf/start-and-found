package auth

import (
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/gtrirf/start-and-found/api/internal/platform/httpx"
	"github.com/gtrirf/start-and-found/api/internal/users"
)

// Handler exposes the auth HTTP API.
type Handler struct {
	service *Service
	logger  *slog.Logger
}

// NewHandler builds the auth handler.
func NewHandler(service *Service, logger *slog.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

type sessionResponse struct {
	AccessToken      string        `json:"access_token"`
	AccessExpiresAt  time.Time     `json:"access_expires_at"`
	RefreshToken     string        `json:"refresh_token"`
	RefreshExpiresAt time.Time     `json:"refresh_expires_at"`
	User             users.Account `json:"user"`
}

func newSessionResponse(session Session) sessionResponse {
	return sessionResponse{
		AccessToken:      session.AccessToken,
		AccessExpiresAt:  session.AccessExpiresAt,
		RefreshToken:     session.RefreshToken,
		RefreshExpiresAt: session.RefreshExpiresAt,
		User:             session.User.Account(),
	}
}

// signup handles POST /v1/auth/signup.
func (h *Handler) signup(w http.ResponseWriter, r *http.Request) {
	input, err := httpx.Decode[SignupInput](r)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	session, err := h.service.Signup(r.Context(), input, requestMeta(r))
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, newSessionResponse(session))
}

// login handles POST /v1/auth/login.
func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	input, err := httpx.Decode[LoginInput](r)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	session, err := h.service.Login(r.Context(), input, requestMeta(r))
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, newSessionResponse(session))
}

// refresh handles POST /v1/auth/refresh and rotates the refresh token.
func (h *Handler) refresh(w http.ResponseWriter, r *http.Request) {
	input, err := httpx.Decode[RefreshInput](r)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	session, err := h.service.Refresh(r.Context(), input.RefreshToken, requestMeta(r))
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, newSessionResponse(session))
}

// logout handles POST /v1/auth/logout.
func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	input, err := httpx.Decode[LogoutInput](r)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	if err := h.service.Logout(r.Context(), input.RefreshToken); err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.NoContent(w)
}

func requestMeta(r *http.Request) RequestMeta {
	return RequestMeta{
		UserAgent: r.UserAgent(),
		IP:        clientIP(r),
	}
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
