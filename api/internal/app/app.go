// Package app is the composition root of the API: it connects the infrastructure
// and wires every domain together.
//
// Wiring lives here (and only here) so domains can depend on each other's
// abstractions without importing each other's transport or infrastructure.
package app

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/redis/go-redis/v9"

	"github.com/gtrirf/start-and-found/api/internal/platform/cache"
	"github.com/gtrirf/start-and-found/api/internal/platform/config"
	"github.com/gtrirf/start-and-found/api/internal/platform/database"
	"github.com/gtrirf/start-and-found/api/internal/platform/storage"
)

// App owns the API dependencies and its HTTP handler.
type App struct {
	Config  config.Config
	Logger  *slog.Logger
	DB      *database.DB
	Redis   *redis.Client
	Storage *storage.Client

	handler http.Handler
	closers []func()
}

// New connects the infrastructure and builds the HTTP handler.
func New(ctx context.Context, cfg config.Config, logger *slog.Logger) (*App, error) {
	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	logger.Info("connected to postgres")

	redisClient, err := cache.Connect(ctx, cfg.RedisURL)
	if err != nil {
		// Redis powers rate limiting and caching only. The API runs degraded
		// instead of refusing to start when the cache is unreachable.
		logger.Warn("redis is unavailable, rate limiting is disabled", "error", err)
		redisClient = nil
	}

	store, err := storage.New(storage.Config{
		Endpoint:      cfg.Storage.Endpoint,
		Region:        cfg.Storage.Region,
		Bucket:        cfg.Storage.Bucket,
		AccessKey:     cfg.Storage.AccessKey,
		SecretKey:     cfg.Storage.SecretKey,
		UseSSL:        cfg.Storage.UseSSL,
		PublicBaseURL: cfg.Storage.PublicBaseURL,
	})
	if err != nil {
		db.Close()
		return nil, err
	}

	application := &App{
		Config:  cfg,
		Logger:  logger,
		DB:      db,
		Redis:   redisClient,
		Storage: store,
	}
	application.closers = append(application.closers, db.Close)
	application.handler = newRouter(Dependencies{
		Config:  cfg,
		Logger:  logger,
		DB:      db,
		Redis:   redisClient,
		Storage: store,
	})
	return application, nil
}

// Router returns the HTTP handler of the API.
func (a *App) Router() http.Handler { return a.handler }

// Close releases the infrastructure connections.
func (a *App) Close() {
	for i := len(a.closers) - 1; i >= 0; i-- {
		a.closers[i]()
	}
	if a.Redis != nil {
		_ = a.Redis.Close()
	}
}
