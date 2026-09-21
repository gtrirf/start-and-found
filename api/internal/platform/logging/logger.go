// Package logging builds the structured logger used across the API.
package logging

import (
	"log/slog"
	"os"
	"strings"
)

// New returns a logger writing to stdout: human readable text during
// development, JSON in production so log aggregators can parse it.
func New(env, level string) *slog.Logger {
	options := &slog.HandlerOptions{Level: parseLevel(level)}

	var handler slog.Handler
	if env == "production" {
		handler = slog.NewJSONHandler(os.Stdout, options)
	} else {
		handler = slog.NewTextHandler(os.Stdout, options)
	}
	return slog.New(handler)
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
