package httpx

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/gtrirf/start-and-found/api/internal/platform/apierr"
	"github.com/gtrirf/start-and-found/api/internal/platform/pagination"
)

// PathParam returns a required path parameter.
func PathParam(r *http.Request, name string) (string, error) {
	value := strings.TrimSpace(chi.URLParam(r, name))
	if value == "" {
		return "", apierr.BadRequest("missing path parameter " + name)
	}
	return value, nil
}

// PathUUID returns a required path parameter parsed as a UUID.
func PathUUID(r *http.Request, name string) (uuid.UUID, error) {
	value, err := PathParam(r, name)
	if err != nil {
		return uuid.Nil, err
	}
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, apierr.BadRequest(name + " must be a valid UUID")
	}
	return id, nil
}

// Query returns a trimmed query parameter value.
func Query(r *http.Request, name string) string {
	return strings.TrimSpace(r.URL.Query().Get(name))
}

// QueryInt returns an integer query parameter, falling back to fallback when
// the parameter is absent.
func QueryInt(r *http.Request, name string, fallback int) (int, error) {
	raw := Query(r, name)
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, apierr.BadRequest(name + " must be an integer")
	}
	return value, nil
}

// QueryBool returns a boolean query parameter.
func QueryBool(r *http.Request, name string, fallback bool) (bool, error) {
	raw := Query(r, name)
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false, apierr.BadRequest(name + " must be a boolean")
	}
	return value, nil
}

// PageParams parses ?limit and ?cursor into keyset pagination inputs.
func PageParams(r *http.Request) (int, *pagination.Cursor, error) {
	limit, err := QueryInt(r, "limit", pagination.DefaultLimit)
	if err != nil {
		return 0, nil, err
	}
	raw := Query(r, "cursor")
	if raw == "" {
		return pagination.NormalizeLimit(limit), nil, nil
	}
	cursor, err := pagination.Decode(raw)
	if err != nil {
		return 0, nil, apierr.BadRequest("cursor is invalid")
	}
	return pagination.NormalizeLimit(limit), &cursor, nil
}
