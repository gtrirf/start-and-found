// Package apierr defines the error model shared by every HTTP handler.
//
// Failures are always rendered with a single envelope:
//
//	{"error": {"code": "not_found", "message": "user not found", "details": {}}}
//
// Domain services return *Error values instead of HTTP details, which keeps the
// domains transport agnostic and makes the API surface consistent.
package apierr

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

// Error is an application error mapped to an HTTP status code.
type Error struct {
	Status  int            `json:"-"`
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`

	cause error
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap exposes the wrapped cause so errors.Is / errors.As keep working.
func (e *Error) Unwrap() error { return e.cause }

// WithCause attaches an internal cause. Causes are logged but never serialized.
func (e *Error) WithCause(err error) *Error {
	e.cause = err
	return e
}

// WithDetails attaches machine readable details such as field violations.
func (e *Error) WithDetails(details map[string]any) *Error {
	e.Details = details
	return e
}

// New creates an Error with an explicit status code.
func New(status int, code, message string) *Error {
	return &Error{Status: status, Code: code, Message: message}
}

// BadRequest reports malformed input.
func BadRequest(message string) *Error {
	return New(http.StatusBadRequest, "bad_request", message)
}

// Validation reports invalid field values.
func Validation(message string, details map[string]any) *Error {
	return New(http.StatusUnprocessableEntity, "validation_failed", message).WithDetails(details)
}

// Unauthorized reports missing or invalid credentials.
func Unauthorized(message string) *Error {
	return New(http.StatusUnauthorized, "unauthorized", message)
}

// Forbidden reports an authenticated caller without the required rights.
func Forbidden(message string) *Error {
	return New(http.StatusForbidden, "forbidden", message)
}

// NotFound reports a missing resource.
func NotFound(message string) *Error {
	return New(http.StatusNotFound, "not_found", message)
}

// Conflict reports a uniqueness or state conflict.
func Conflict(message string) *Error {
	return New(http.StatusConflict, "conflict", message)
}

// PayloadTooLarge reports a body that exceeds the accepted size.
func PayloadTooLarge(message string) *Error {
	return New(http.StatusRequestEntityTooLarge, "payload_too_large", message)
}

// TooManyRequests reports a throttled caller.
func TooManyRequests(message string) *Error {
	return New(http.StatusTooManyRequests, "too_many_requests", message)
}

// NotImplemented reports functionality that is intentionally outside the MVP.
func NotImplemented(message string) *Error {
	return New(http.StatusNotImplemented, "not_implemented", message)
}

// ServiceUnavailable reports a temporarily unusable dependency.
func ServiceUnavailable(message string) *Error {
	return New(http.StatusServiceUnavailable, "service_unavailable", message)
}

// Internal wraps an unexpected failure. The cause is logged, not returned.
func Internal(err error) *Error {
	return New(http.StatusInternalServerError, "internal_error", "internal server error").WithCause(err)
}

// From converts any error into an *Error.
func From(err error) *Error {
	if err == nil {
		return nil
	}
	var apiErr *Error
	if errors.As(err, &apiErr) {
		return apiErr
	}
	return Internal(err)
}

type response struct {
	Error *Error `json:"error"`
}

// Write renders err using the platform error envelope.
func Write(w http.ResponseWriter, r *http.Request, err error) {
	apiErr := From(err)
	if apiErr.Status >= http.StatusInternalServerError {
		slog.ErrorContext(r.Context(), "request failed",
			"error", err,
			"code", apiErr.Code,
			"method", r.Method,
			"path", r.URL.Path,
			"request_id", middleware.GetReqID(r.Context()),
		)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(apiErr.Status)
	if encErr := json.NewEncoder(w).Encode(response{Error: apiErr}); encErr != nil {
		slog.ErrorContext(r.Context(), "failed to encode error response", "error", encErr)
	}
}
