package app

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/gtrirf/start-and-found/api/internal/auth"
	"github.com/gtrirf/start-and-found/api/internal/follows"
	"github.com/gtrirf/start-and-found/api/internal/media"
	"github.com/gtrirf/start-and-found/api/internal/notifications"
	"github.com/gtrirf/start-and-found/api/internal/platform/config"
	"github.com/gtrirf/start-and-found/api/internal/platform/database"
	"github.com/gtrirf/start-and-found/api/internal/platform/httpx"
	"github.com/gtrirf/start-and-found/api/internal/platform/ratelimit"
	"github.com/gtrirf/start-and-found/api/internal/platform/storage"
	"github.com/gtrirf/start-and-found/api/internal/posts"
	"github.com/gtrirf/start-and-found/api/internal/projects"
	"github.com/gtrirf/start-and-found/api/internal/publishers"
	"github.com/gtrirf/start-and-found/api/internal/reactions"
	"github.com/gtrirf/start-and-found/api/internal/threads"
	"github.com/gtrirf/start-and-found/api/internal/users"
)

// Request limits.
const (
	requestTimeout     = 30 * time.Second
	maxRequestBodySize = 1 << 20 // 1 MiB: uploads go straight to object storage
	authRateLimit      = 30
	authRateWindow     = time.Minute
)

// Dependencies are the infrastructure pieces the HTTP layer needs. Keeping them
// in a struct lets tests build the router without a database.
type Dependencies struct {
	Config  config.Config
	Logger  *slog.Logger
	DB      *database.DB
	Redis   *redis.Client
	Storage *storage.Client
}

// newRouter wires the domains, the middleware stack and the routes.
func newRouter(deps Dependencies) http.Handler {
	cfg := deps.Config

	// Repositories share one querier: the pool. Transactions build their own
	// querier scoped to the transaction (see database.DB.WithTx).
	var querier database.Querier
	if deps.DB != nil {
		querier = deps.DB.Pool()
	}

	publishersRepository := publishers.NewRepository(querier)
	usersRepository := users.NewRepository(querier)
	projectsRepository := projects.NewRepository(querier)
	postsRepository := posts.NewRepository(querier)
	mediaRepository := media.NewRepository(querier)
	sessionsRepository := auth.NewSessionsRepository(querier)

	tokens := auth.NewTokenManager(cfg.JWTSecret, cfg.AccessTokenTTL)
	authService := auth.NewService(deps.DB, usersRepository, sessionsRepository, publishersRepository, tokens, cfg.RefreshTokenTTL)
	authMiddleware := auth.NewMiddleware(authService)

	postService := posts.NewService(deps.DB, postsRepository, publishersRepository, projectsRepository, deps.Storage)
	mediaService := media.NewService(mediaRepository, deps.Storage)
	projectService := projects.NewService(deps.DB, projectsRepository, projectOwners{users: usersRepository}, publishersRepository)
	userService := users.NewService(usersRepository, projectsRepository, postService, publishersRepository)
	threadService := threads.NewService(postService)

	authz := httpx.Auth{
		Require:  authMiddleware.RequireAuth,
		Optional: authMiddleware.OptionalAuth,
	}

	authHandler := auth.NewHandler(authService, deps.Logger)
	userHandler := users.NewHandler(userService, authz)
	projectHandler := projects.NewHandler(projectService, authz)
	postHandler := posts.NewHandler(postService, authz)
	threadHandler := threads.NewHandler(threadService, authz)
	mediaHandler := media.NewHandler(mediaService, authz)

	authLimiter := ratelimit.New(deps.Redis, deps.Logger, "auth", authRateLimit, authRateWindow)

	router := chi.NewRouter()
	router.Use(
		middleware.RequestID,
		middleware.RealIP,
		httpx.Logger(deps.Logger),
		httpx.Recoverer(deps.Logger),
		httpx.Timeout(requestTimeout),
		httpx.MaxBytes(maxRequestBodySize),
		cors.Handler(cors.Options{
			AllowedOrigins: cfg.CORSOrigins,
			AllowedMethods: []string{http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodDelete, http.MethodOptions},
			AllowedHeaders: []string{"Authorization", "Content-Type"},
			MaxAge:         300,
		}),
	)

	router.Get("/healthz", liveness)
	router.Get("/readyz", readiness(deps))
	registerRoutes(router, routeHandlers{
		auth:     authHandler,
		users:    userHandler,
		projects: projectHandler,
		posts:    postHandler,
		threads:  threadHandler,
		media:    mediaHandler,
		limiter:  authLimiter,
	})
	return router
}

// routeHandlers groups the domain handlers so route registration stays readable.
type routeHandlers struct {
	auth     *auth.Handler
	users    *users.Handler
	projects *projects.Handler
	posts    *posts.Handler
	threads  *threads.Handler
	media    *media.Handler
	limiter  *ratelimit.Limiter
}

// projectOwners adapts the users repository to the narrow lookup the projects
// domain declares. The adapter lives in the composition root, which is what keeps
// the users and projects domains free of an import cycle.
type projectOwners struct {
	users *users.Repository
}

// ByID implements projects.OwnerLookup.
func (o projectOwners) ByID(ctx context.Context, id uuid.UUID) (projects.Owner, error) {
	user, err := o.users.ByID(ctx, id)
	if err != nil {
		return projects.Owner{}, err
	}
	return projects.Owner{ID: user.ID, Username: user.Username}, nil
}

// ByUsername implements projects.OwnerLookup.
func (o projectOwners) ByUsername(ctx context.Context, username string) (projects.Owner, error) {
	user, err := o.users.ByUsername(ctx, username)
	if err != nil {
		return projects.Owner{}, err
	}
	return projects.Owner{ID: user.ID, Username: user.Username}, nil
}

// registerRoutes mounts every domain under /v1.
//
// Public reads are wrapped with OptionalAuth so they can recognise a signed in
// visitor, while writes live behind RequireAuth inside each domain router.
func registerRoutes(router chi.Router, handlers routeHandlers) {
	router.Route("/v1", func(r chi.Router) {
		// Unauthenticated writes: throttled per client IP.
		r.Group(func(r chi.Router) {
			r.Use(handlers.limiter.Middleware)
			r.Mount("/auth", handlers.auth.Routes())
		})

		r.Mount("/users", handlers.users.Routes())
		r.Mount("/me", handlers.users.MeRoutes())
		r.Mount("/projects", handlers.projects.Routes())
		r.Mount("/posts", handlers.posts.Routes())
		r.Mount("/feed", handlers.posts.FeedRoutes())
		r.Mount("/threads", handlers.threads.Routes())
		r.Mount("/media", handlers.media.Routes())

		// Endpoints that belong to the API surface but are intentionally outside
		// the MVP (README: Future Direction). They answer 501 using the shared
		// error envelope, so clients can already target them.
		r.Method(http.MethodGet, "/posts/{postID}/reactions", reactions.NotImplemented())
		r.Method(http.MethodPost, "/posts/{postID}/reactions", reactions.NotImplemented())
		r.Method(http.MethodDelete, "/posts/{postID}/reactions", reactions.NotImplemented())
		r.Method(http.MethodPost, "/publishers/{publisherID}/follow", follows.NotImplemented())
		r.Method(http.MethodDelete, "/publishers/{publisherID}/follow", follows.NotImplemented())
		r.Method(http.MethodGet, "/notifications", notifications.NotImplemented())
		r.Method(http.MethodPost, "/notifications/read", notifications.NotImplemented())
	})
}

// liveness reports that the process is up.
func liveness(w http.ResponseWriter, r *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

// readiness reports the state of the dependencies. Only PostgreSQL decides the
// status code: without it the API cannot serve at all, while an unavailable
// Redis degrades rate limiting and object storage degrades uploads.
func readiness(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		checks := map[string]string{}
		healthy := true

		switch {
		case deps.DB == nil:
			checks["postgres"] = "not configured"
			healthy = false
		default:
			if err := deps.DB.Ping(ctx); err != nil {
				checks["postgres"] = "unavailable"
				healthy = false
			} else {
				checks["postgres"] = "ok"
			}
		}

		if deps.Redis != nil {
			if err := deps.Redis.Ping(ctx).Err(); err != nil {
				checks["redis"] = "unavailable"
			} else {
				checks["redis"] = "ok"
			}
		} else {
			checks["redis"] = "not configured"
		}

		if deps.Storage != nil {
			checks["object_storage"] = "configured"
		} else {
			checks["object_storage"] = "not configured"
		}

		status := "ok"
		statusCode := http.StatusOK
		if !healthy {
			status = "unavailable"
			statusCode = http.StatusServiceUnavailable
		}
		httpx.JSON(w, statusCode, map[string]any{"status": status, "checks": checks})
	}
}
