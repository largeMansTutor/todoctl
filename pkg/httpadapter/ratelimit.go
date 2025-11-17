package httpadapter

import (
	"net"
	"net/http"
	"strings"
	"sync"

	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

// RateLimiter enforces per-key throttling using a token bucket.
type RateLimiter struct {
	limit    rate.Limit
	burst    int
	logger   *zap.Logger
	limiters sync.Map // map[string]*rate.Limiter
}

func NewRateLimiter(rps int, burst int, logger *zap.Logger) *RateLimiter {
	if rps <= 0 || burst <= 0 {
		return nil
	}
	return &RateLimiter{
		limit:  rate.Limit(rps),
		burst:  burst,
		logger: logger,
	}
}

// Middleware applies rate limiting by API key if present, otherwise by client IP.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if rl == nil {
			next.ServeHTTP(w, r)
			return
		}
		key := r.Header.Get("X-API-Key")
		if key == "" {
			key = clientIP(r)
		}
		limiter := rl.getLimiter(key)
		if !limiter.Allow() {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":{"message":"rate limit exceeded"}}`))
			if rl.logger != nil {
				rl.logger.Warn("rate limit exceeded", zap.String("key", key), zap.String("path", r.URL.Path))
			}
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) getLimiter(key string) *rate.Limiter {
	actual, _ := rl.limiters.LoadOrStore(key, rate.NewLimiter(rl.limit, rl.burst))
	return actual.(*rate.Limiter)
}
func clientIP(r *http.Request) string {
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
