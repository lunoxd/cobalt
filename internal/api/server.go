package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/lunoxd/cobalt/internal/auth"
	"github.com/lunoxd/cobalt/internal/config"
	"github.com/lunoxd/cobalt/internal/database"
	"github.com/lunoxd/cobalt/internal/inspector"
	"github.com/lunoxd/cobalt/internal/jobs"
	"github.com/lunoxd/cobalt/internal/mcp"
	"github.com/lunoxd/cobalt/internal/middleware"
	"github.com/lunoxd/cobalt/internal/projects"
	"github.com/lunoxd/cobalt/internal/webhooks"
)

// Server wraps the Chi router and all dependencies.
type Server struct {
	cfg         *config.Config
	db          *database.DB
	pool        *jobs.Pool
	rateLimiter *middleware.RateLimiter
	router      chi.Router
}

// NewServer initializes the HTTP router and wires all components.
func NewServer(cfg *config.Config, db *database.DB, pool *jobs.Pool) *Server {
	r := chi.NewRouter()

	// 1. Core middleware
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestLogger)
	r.Use(middleware.SecurityHeaders)
	r.Use(chiMiddleware.RealIP)

	// 2. CORS configuration
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORSAllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-API-Key", "X-Request-ID"},
		ExposedHeaders:   []string{"Link", "X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// 3. In-memory Rate Limiting
	rateLimiter := middleware.NewRateLimiter(cfg.RateLimitRPS, cfg.RateLimitBurst)
	r.Use(rateLimiter.Limit)

	// 4. Authenticator setup
	apiKeyAuth := auth.NewAPIKeyAuthenticator(db, cfg.AdminAPIKey)
	extAuth := auth.NewExternalAuthProvider(cfg.AuthJWTSecret)
	compositeAuth := auth.NewChainedAuthenticator(apiKeyAuth, extAuth)

	// Global auth middleware (identifies caller if token provided, allows unauthenticated routes)
	r.Use(auth.AuthenticateMiddleware(compositeAuth))

	// 5. Domain services
	var projRepo projects.Repository
	if db != nil && db.Pool != nil {
		projRepo = projects.NewPostgresRepository(db)
	} else {
		projRepo = projects.NewMemoryRepository()
	}
	projService := projects.NewService(projRepo)
	apiKeyService := auth.NewAPIKeyService(db)
	ins := inspector.NewInspector(db)
	mcpServer := mcp.NewServer(ins, projService)
	webhookHandler := webhooks.NewHandler(db, pool, cfg.WebhookSecret)

	// Register background job workers
	if pool != nil {
		webhooks.RegisterWebhookJobHandler(pool, db)
	}

	// 6. Route Registration
	// Health & readiness
	r.Get("/health", HealthHandler)
	r.Get("/ready", ReadyHandler(db))

	// Projects REST API
	projectsHandler := NewProjectsHandler(projService)
	r.Mount("/api/projects", projectsHandler.Routes())

	// Admin API (Inspector, System Stats, API Keys)
	adminHandler := NewAdminHandler(db, apiKeyService, ins)
	r.Mount("/api/admin", adminHandler.Routes())

	// MCP JSON-RPC 2.0 Server
	r.HandleFunc("/mcp", mcpServer.Handler())

	// Webhooks
	r.Mount("/webhooks", webhookHandler.Routes())

	return &Server{
		cfg:         cfg,
		db:          db,
		pool:        pool,
		rateLimiter: rateLimiter,
		router:      r,
	}
}

// Router returns the underlying Chi router (useful for testing).
func (s *Server) Router() chi.Router {
	return s.router
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}
