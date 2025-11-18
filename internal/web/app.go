package web

import (
	"github.com/thetrollfarmercodes/todoctl/todo/internal/platform/auth"
	"github.com/thetrollfarmercodes/todoctl/todo/internal/platform/config"
	"github.com/thetrollfarmercodes/todoctl/todo/internal/platform/metrics"
	todousecase "github.com/thetrollfarmercodes/todoctl/todo/internal/service/todo"
	"github.com/thetrollfarmercodes/todoctl/todo/pkg/httpadapter"
	"go.uber.org/zap"
)

// App encapsulates all HTTP handlers and their dependencies. It
// delegates to the todo Service for business logic. The methods on
// App must be registered with the router in router.go.
type App struct {
	cfg         *config.Config
	svc         *todousecase.Service
	auth        *auth.Service
	rateLimiter *httpadapter.RateLimiter
	metrics     *metrics.Provider
	logger      *zap.Logger
}

// New constructs a App with the provided service.
func New(cfg *config.Config, svc *todousecase.Service, authz *auth.Service, limiter *httpadapter.RateLimiter, metrics *metrics.Provider, logger *zap.Logger) *App {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &App{
		cfg:         cfg,
		svc:         svc,
		auth:        authz,
		rateLimiter: limiter,
		metrics:     metrics,
		logger:      logger,
	}
}
