package httpadapter

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.uber.org/zap"
)

// Router is the interface we rely on from chi. Exported so adapters can
// accept a preconfigured router without knowing chi specifics.
type Router = chi.Router

// NewRouter builds a base HTTP router with standard middleware, health,
// and optional metrics. Feature routes should be mounted by callers via
// provided RouteMounters.
func NewRouter(cfg *HTTPConfig, logger *zap.Logger, metrics MetricsProvider, mounters ...RouteMounter) (Router, error) {
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

	r.Get("/healthz", Health)

	// Metrics endpoint
	if metrics != nil {
		r.Use(metrics.Middleware)
		r.Handle("/metrics", metrics.Handler())
	}
	for _, m := range mounters {
		if m != nil {
			m.Mount(r)
		}
	}
	return r, nil
}

type MetricsProvider interface {
	Middleware(next http.Handler) http.Handler
	Handler() http.Handler
}

type RouteMounter interface {
	Mount(router Router)
}
