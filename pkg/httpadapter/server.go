package httpadapter

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

// StartHTTPServer registers an HTTP server with Fx's lifecycle. It listens on
// the configured port and gracefully shuts down on application stop.
func StartHTTPServer(lc fx.Lifecycle, cfg *HTTPConfig, router http.Handler, logger *zap.Logger) {
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
				if err != nil && !errors.Is(err, http.ErrServerClosed) {
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
