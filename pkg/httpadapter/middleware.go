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
			ww := &ResponseWriter{ResponseWriter: w, status: 200}
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

// ResponseWriter wraps http.ResponseWriter to capture the status code and
// the number of bytes written. It delegates everything else to the
// underlying ResponseWriter.
type ResponseWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (w *ResponseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *ResponseWriter) Write(b []byte) (int, error) {
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

// APIKeyAuth enforces API key checks using an auth service. If the service is nil,
// it acts as a no-op (for explicit opt-out in dev). Otherwise, requests must
// include X-API-Key (and optionally X-API-Key-ID when hashed keys are used). It
// logs the key label (if provided by the service) for audit.
func APIKeyAuth(authz interface {
	Authorize(key string, keyID ...string) (label string, ok bool)
}, logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if authz != nil {
				apiKey := r.Header.Get("X-API-Key")
				apiKeyID := r.Header.Get("X-API-Key-ID")
				if label, ok := authz.Authorize(apiKey, apiKeyID); !ok {
					w.Header().Set("WWW-Authenticate", "API key required")
					w.WriteHeader(http.StatusUnauthorized)
					_, _ = w.Write([]byte(`{"error": {"message": "unauthorized"}}`))
					if logger != nil {
						logger.Warn("unauthorized request", zap.String("remote", r.RemoteAddr), zap.String("api_key_id", apiKeyID))
					}
					return
				} else if logger != nil && label != "" {
					logger = logger.With(zap.String("api_key_label", label))
				}
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

// SecurityHeaders sets a minimal set of defensive headers to reduce common
// OWASP-category risks (clickjacking, MIME sniffing, XSS). For APIs this is
// lightweight and safe; values are chosen to avoid breaking typical clients.
func SecurityHeaders() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("Referrer-Policy", "no-referrer")
			// Allows browser XSS filters
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			// API responses shouldn't be rendered; lock down content execution.
			w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'")
			// Avoid caching potentially sensitive responses by intermediaries.
			w.Header().Set("Cache-Control", "no-store")
			// Disable access to powerful features by default.
			w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
			next.ServeHTTP(w, r)
		})
	}
}
