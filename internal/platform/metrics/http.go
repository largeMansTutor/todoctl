package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// HTTPMetrics tracks HTTP request counts and latencies.
type HTTPMetrics struct {
	requests  *prometheus.CounterVec
	latencies *prometheus.HistogramVec
}

func newHTTPMetrics(reg *prometheus.Registry) *HTTPMetrics {
	h := &HTTPMetrics{
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests processed.",
		}, []string{"method", "path", "status"}),
		latencies: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Latency distributions of HTTP requests.",
			Buckets: prometheus.ExponentialBuckets(0.005, 2, 10),
		}, []string{"method", "path", "status"}),
	}
	reg.MustRegister(h.requests, h.latencies)
	return h
}

// Middleware records request counts and latencies.
func (h *HTTPMetrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(ww, r)
		status := ww.status
		labels := prometheus.Labels{
			"method": r.Method,
			"path":   r.URL.Path,
			"status": http.StatusText(status),
		}
		h.requests.With(labels).Inc()
		h.latencies.With(labels).Observe(time.Since(start).Seconds())
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (w *responseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}
