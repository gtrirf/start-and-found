package app

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gtrirf/start-and-found/api/internal/platform/config"
)

// newTestRouter builds the real router with nil infrastructure. Routing,
// middleware and request decoding can therefore be tested without PostgreSQL.
func newTestRouter(t *testing.T) http.Handler {
	t.Helper()

	cfg := config.Config{
		Env:             config.EnvTest,
		JWTSecret:       "test-secret-value-long-enough-for-hs256",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 30 * 24 * time.Hour,
		CORSOrigins:     []string{"http://localhost:3000"},
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	return newRouter(Dependencies{Config: cfg, Logger: logger})
}

func TestLiveness(t *testing.T) {
	recorder := httptest.NewRecorder()
	newTestRouter(t).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	var payload map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if payload["status"] != "ok" {
		t.Fatalf("status = %q, want ok", payload["status"])
	}
}

func TestReadinessWithoutDatabase(t *testing.T) {
	recorder := httptest.NewRecorder()
	newTestRouter(t).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
	var payload struct {
		Status string            `json:"status"`
		Checks map[string]string `json:"checks"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if payload.Status != "unavailable" || payload.Checks["postgres"] != "not configured" {
		t.Fatalf("unexpected readiness payload: %+v", payload)
	}
}

func TestProtectedRoutesRejectAnonymousCallers(t *testing.T) {
	router := newTestRouter(t)
	postID := "11111111-1111-4111-8111-111111111111"

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{name: "me", method: http.MethodGet, path: "/v1/me"},
		{name: "my publishers", method: http.MethodGet, path: "/v1/me/publishers"},
		{name: "create post", method: http.MethodPost, path: "/v1/posts"},
		{name: "create reply", method: http.MethodPost, path: "/v1/posts/" + postID + "/replies"},
		{name: "create project", method: http.MethodPost, path: "/v1/projects"},
		{name: "presign upload", method: http.MethodPost, path: "/v1/media/presign"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(test.method, test.path, nil))

			if recorder.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
			}
			var payload struct {
				Error struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if payload.Error.Code != "unauthorized" {
				t.Fatalf("error code = %q, want unauthorized", payload.Error.Code)
			}
		})
	}
}

func TestAuthLoginValidatesBodyBeforeTouchingTheDatabase(t *testing.T) {
	// The auth endpoints are public, so an empty body must be rejected by the
	// decoder without any repository call.
	recorder := httptest.NewRecorder()
	newTestRouter(t).ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/v1/auth/login", nil))

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestFutureFeatureRoutesAnswerNotImplemented(t *testing.T) {
	router := newTestRouter(t)
	postID := "11111111-1111-4111-8111-111111111111"
	publisherID := "22222222-2222-4222-8222-222222222222"

	tests := []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/v1/posts/" + postID + "/reactions"},
		{method: http.MethodPost, path: "/v1/posts/" + postID + "/reactions"},
		{method: http.MethodPost, path: "/v1/publishers/" + publisherID + "/follow"},
		{method: http.MethodGet, path: "/v1/notifications"},
	}

	for _, test := range tests {
		t.Run(test.method+" "+test.path, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(test.method, test.path, nil))

			if recorder.Code != http.StatusNotImplemented {
				t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotImplemented)
			}
			var payload struct {
				Error struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if payload.Error.Code != "not_implemented" {
				t.Fatalf("error code = %q, want not_implemented", payload.Error.Code)
			}
		})
	}
}

func TestUnknownRouteIsNotFound(t *testing.T) {
	recorder := httptest.NewRecorder()
	newTestRouter(t).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/nope", nil))

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
}

func TestCORSPreflightAllowsTheWebApp(t *testing.T) {
	request := httptest.NewRequest(http.MethodOptions, "/v1/feed", nil)
	request.Header.Set("Origin", "http://localhost:3000")
	request.Header.Set("Access-Control-Request-Method", http.MethodGet)

	recorder := httptest.NewRecorder()
	newTestRouter(t).ServeHTTP(recorder, request)

	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want the web origin", got)
	}
}
