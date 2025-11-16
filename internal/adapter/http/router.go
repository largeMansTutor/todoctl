package httpadapter

import (
	"context"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/thetrollfarmercodes/todoctl/todo/internal/platform/config"
	"github.com/thetrollfarmercodes/todoctl/todo/internal/platform/metrics"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// NewRouter builds a new HTTP router with all endpoints and
// middleware wired in. It is an Fx provider and will be injected into
// the HTTP server. It depends on the Handler, Config and Logger.
func NewRouter(cfg *config.Config, handler *Handler, logger *zap.Logger, metrics *metrics.Provider) (http.Handler, error) {
	r := chi.NewRouter()
	// Standard middlewares
	r.Use(otelhttp.NewMiddleware(cfg.OTELServiceName))
	r.Use(middleware.RealIP)
	r.Use(middleware.RequestID)
	r.Use(Logger(logger))
	r.Use(Recoverer(logger))
	r.Use(MaxBytes(cfg.MaxBodyBytes))
	// Timeout: create a per-request context with a deadline.
	r.Use(middleware.Timeout(cfg.ReadTimeout))

	// Routes
	r.Get("/healthz", handler.Healthz)
	r.Get("/readyz", handler.Readyz)

	// Metrics endpoint (unauthenticated to allow scraping)
	if metrics != nil {
		r.Use(metrics.HTTP.Middleware)
		r.Handle("/metrics", metrics.Handler())
	}

	// Protected API routes
	r.Group(func(protected chi.Router) {
		protected.Use(APIKeyAuth(cfg.APIKey, logger))
		if handler.rateLimiter != nil {
			protected.Use(handler.rateLimiter.Middleware)
		}
		protected.Post("/todos", handler.CreateTodos)
		protected.Patch("/todos", handler.UpdateTodos)
		protected.Get("/todos", handler.ListTodos)
	})

	return r, nil
}

// StartHTTPServer registers an HTTP server with Fx's lifecycle. It
// listens on the configured port and gracefully shuts down on
// application stop. The router is injected. This function is an
// invoke, not a provider; it doesn't return anything.
func StartHTTPServer(lc fx.Lifecycle, cfg *config.Config, router http.Handler, logger *zap.Logger) {
	srv := &http.Server{
		Addr:         ":" + strconv.Itoa(cfg.HTTPPort),
		Handler:      router,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				var err error
				if cfg.TLSCertFile != "" && cfg.TLSKeyFile != "" {
					err = srv.ListenAndServeTLS(cfg.TLSCertFile, cfg.TLSKeyFile)
				} else {
					err = srv.ListenAndServe()
				}
				if err != nil && err != http.ErrServerClosed {
					logger.Error("HTTP server error", zap.Error(err))
				}
			}()
			logger.Info("HTTP server started", zap.Int("port", cfg.HTTPPort))
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Shutting down HTTP server")
			return srv.Shutdown(ctx)
		},
	})
}
