package metrics

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Provider wraps Prometheus instrumentation used by the service.
type Provider struct {
	Registry *prometheus.Registry
	HTTP     *HTTPMetrics
}

// New initialises a Prometheus registry with Go/runtime collectors and
// HTTP request instrumentation.
func New() *Provider {
	reg := prometheus.NewRegistry()
	reg.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	reg.MustRegister(collectors.NewGoCollector())
	httpMetrics := newHTTPMetrics(reg)
	return &Provider{
		Registry: reg,
		HTTP:     httpMetrics,
	}
}

// Handler exposes the metrics endpoint handler.
func (p *Provider) Handler() http.Handler {
	return promhttp.HandlerFor(p.Registry, promhttp.HandlerOpts{})
}

// RegisterDBMetrics exports database pool statistics.
func (p *Provider) RegisterDBMetrics(db *sql.DB) {
	if db == nil {
		return
	}
	p.Registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "db_max_open_connections",
		Help: "Maximum number of open connections to the database.",
	}, func() float64 {
		return float64(db.Stats().MaxOpenConnections)
	}))
	p.Registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "db_open_connections",
		Help: "Number of established database connections.",
	}, func() float64 {
		return float64(db.Stats().OpenConnections)
	}))
	p.Registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "db_in_use_connections",
		Help: "Number of connections currently in use.",
	}, func() float64 {
		return float64(db.Stats().InUse)
	}))
	p.Registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "db_idle_connections",
		Help: "Number of idle connections in the pool.",
	}, func() float64 {
		return float64(db.Stats().Idle)
	}))
	p.Registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "db_wait_count",
		Help: "Total number of connections waited for.",
	}, func() float64 {
		return float64(db.Stats().WaitCount)
	}))
	p.Registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "db_wait_duration_seconds",
		Help: "Total time blocked waiting for a new connection.",
	}, func() float64 {
		return db.Stats().WaitDuration.Seconds()
	}))
}

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
