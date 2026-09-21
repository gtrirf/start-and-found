package httpx

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/go-chi/chi/v5/middleware"

	"github.com/gtrirf/start-and-found/api/internal/platform/apierr"
)

// Logger logs one structured line per request.
func Logger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			wrapped := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			started := time.Now()
			defer func() {
				status := wrapped.Status()
				if status == 0 {
					status = http.StatusOK
				}
				logger.LogAttrs(r.Context(), slog.LevelInfo, "http request",
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.Int("status", status),
					slog.Int("bytes", wrapped.BytesWritten()),
					slog.Duration("duration", time.Since(started)),
					slog.String("request_id", middleware.GetReqID(r.Context())),
					slog.String("remote_addr", r.RemoteAddr),
				)
			}()
			next.ServeHTTP(wrapped, r)
		})
	}
}

// Recoverer converts panics into the shared error envelope.
func Recoverer(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				recovered := recover()
				if recovered == nil {
					return
				}
				if recovered == http.ErrAbortHandler {
					panic(recovered)
				}
				logger.ErrorContext(r.Context(), "panic recovered",
					"panic", fmt.Sprintf("%v", recovered),
					"path", r.URL.Path,
					"stack", string(debug.Stack()),
				)
				apierr.Write(w, r, apierr.Internal(fmt.Errorf("panic: %v", recovered)))
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// MaxBytes limits how large a request body may be.
func MaxBytes(limit int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, limit)
			next.ServeHTTP(w, r)
		})
	}
}

// Timeout aborts requests that take longer than the given duration.
func Timeout(duration time.Duration) func(http.Handler) http.Handler {
	return middleware.Timeout(duration)
}
