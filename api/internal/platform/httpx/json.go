// Package httpx contains the HTTP plumbing shared by every domain: request
// decoding, response encoding, middleware and parameter parsing.
package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/gtrirf/start-and-found/api/internal/platform/apierr"
)

// JSON writes payload as a JSON response.
func JSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if payload == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Error("failed to encode response", "error", err)
	}
}

// NoContent writes an empty 204 response.
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// Error renders err with the shared error envelope.
func Error(w http.ResponseWriter, r *http.Request, err error) {
	apierr.Write(w, r, err)
}

// Decode reads a JSON request body into T. Unknown fields, empty bodies and
// trailing data are rejected so clients fail loudly instead of silently.
func Decode[T any](r *http.Request) (T, error) {
	var payload T
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		var maxBytesErr *http.MaxBytesError
		var syntaxErr *json.SyntaxError
		var typeErr *json.UnmarshalTypeError
		switch {
		case errors.As(err, &maxBytesErr):
			return payload, apierr.PayloadTooLarge(fmt.Sprintf("request body must not exceed %d bytes", maxBytesErr.Limit))
		case errors.Is(err, io.EOF):
			return payload, apierr.BadRequest("request body is required")
		case errors.As(err, &syntaxErr):
			return payload, apierr.BadRequest(fmt.Sprintf("malformed JSON at position %d", syntaxErr.Offset))
		case errors.As(err, &typeErr):
			return payload, apierr.BadRequest(fmt.Sprintf("field %q has an invalid type", typeErr.Field))
		default:
			return payload, apierr.BadRequest(err.Error())
		}
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return payload, apierr.BadRequest("request body must contain a single JSON object")
	}
	return payload, nil
}
