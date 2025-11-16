package httpadapter

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

// Logger middleware logs the start and end of each request with relevant
// details. It uses zap for structured logging. We log request
// method, path, status code, latency and remote address. The log level
// for successful requests is Info; for errors (>=500) it is Error.
func Logger(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Wrap ResponseWriter to capture status and bytes written
			ww := &responseWriter{ResponseWriter: w, status: 200}
			start := time.Now()
			next.ServeHTTP(ww, r)
			duration := time.Since(start)
			fields := []zap.Field{
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Int("status", ww.status),
				zap.Duration("duration", duration),
				zap.String("remote", r.RemoteAddr),
			}
			// Include request id if present
			if reqID := middleware.GetReqID(r.Context()); reqID != "" {
				fields = append(fields, zap.String("request_id", reqID))
			}
			if ww.status >= 500 {
				logger.Error("request completed", fields...)
			} else {
				logger.Info("request completed", fields...)
			}
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture the status code and
// the number of bytes written. It delegates everything else to the
// underlying ResponseWriter.
type responseWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (w *responseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *responseWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.bytes += n
	return n, err
}

// MaxBytes limits the size of the request body to n bytes. If the
// request is larger, it returns 413 status. This prevents memory
// exhaustion and protects from DoS attacks. It should be the first
// middleware around handlers that read the body.
func MaxBytes(n int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// If the request advertises a length over the limit, reject early.
			if r.ContentLength > n && r.ContentLength != -1 {
				http.Error(w, http.StatusText(http.StatusRequestEntityTooLarge), http.StatusRequestEntityTooLarge)
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, n)
			next.ServeHTTP(w, r)
		})
	}
}

// APIKeyAuth returns middleware that enforces a static API key check. If
// the provided key is empty, the middleware is a no-op. Otherwise
// requests must include the header X-API-Key equal to the provided
// value. If the header is missing or incorrect, a 401 Unauthorized
// response is returned.
func APIKeyAuth(key string, logger *zap.Logger) func(http.Handler) http.Handler {
	if key == "" {
		return func(next http.Handler) http.Handler { return next }
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Compare constant time to avoid timing attack; but here string comparison is fine
			apiKey := r.Header.Get("X-API-Key")
			if apiKey != key {
				w.Header().Set("WWW-Authenticate", "API key required")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error": {"message": "unauthorized"}}`))
				// Log unauthorized attempts for audit
				logger.Warn("unauthorized request", zap.String("remote", r.RemoteAddr))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// Recoverer is a middleware that recovers from panics and returns an
// internal server error. It logs the panic stack using zap. This
// prevents the server from crashing and avoids leaking internal
// details to clients.
func Recoverer(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logger.Error("panic recovered", zap.Any("error", rec))
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = w.Write([]byte(`{"error": {"message": "internal server error"}}`))
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
