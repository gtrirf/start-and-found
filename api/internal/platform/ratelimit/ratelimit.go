// Package ratelimit provides a Redis backed fixed window rate limiter.
//
// It guards the authentication endpoints, which are the only unauthenticated
// write paths of the API. When Redis is unavailable the limiter fails open: a
// cache outage must never take authentication down.
package ratelimit

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/gtrirf/start-and-found/api/internal/platform/apierr"
	"github.com/gtrirf/start-and-found/api/internal/platform/httpx"
)

// Limiter allows limit requests per window for each client.
type Limiter struct {
	client *redis.Client
	logger *slog.Logger
	prefix string
	limit  int64
	window time.Duration
}

// New builds a limiter. A nil client disables rate limiting entirely.
func New(client *redis.Client, logger *slog.Logger, prefix string, limit int, window time.Duration) *Limiter {
	return &Limiter{
		client: client,
		logger: logger,
		prefix: prefix,
		limit:  int64(limit),
		window: window,
	}
}

// Middleware limits requests by client IP address.
func (l *Limiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if l == nil || l.client == nil {
			next.ServeHTTP(w, r)
			return
		}

		key := fmt.Sprintf("ratelimit:%s:%s", l.prefix, clientIP(r))
		allowed, err := l.allow(r.Context(), key)
		if err != nil {
			l.logger.WarnContext(r.Context(), "rate limiter unavailable, allowing request", "error", err)
			next.ServeHTTP(w, r)
			return
		}
		if !allowed {
			w.Header().Set("Retry-After", strconv.Itoa(int(l.window.Seconds())+1))
			httpx.Error(w, r, apierr.TooManyRequests("too many requests, please try again later"))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (l *Limiter) allow(ctx context.Context, key string) (bool, error) {
	pipe := l.client.TxPipeline()
	counter := pipe.Incr(ctx, key)
	pipe.ExpireNX(ctx, key, l.window)
	if _, err := pipe.Exec(ctx); err != nil {
		return false, err
	}
	return counter.Val() <= l.limit, nil
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
