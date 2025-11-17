package metrics

import (
	"database/sql"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Provider wraps Prometheus instrumentation used by the service.
type Provider struct {
	*HTTPMetrics
	Registry *prometheus.Registry
}

// New initialises a Prometheus registry with Go/runtime collectors and
// HTTP request instrumentation.
func New() *Provider {
	reg := prometheus.NewRegistry()
	reg.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	reg.MustRegister(collectors.NewGoCollector())
	httpMetrics := newHTTPMetrics(reg)
	return &Provider{
		Registry:    reg,
		HTTPMetrics: httpMetrics,
	}
}

// Handler exposes the metrics endpoint handler.
func (p *Provider) Handler() http.Handler {
	return promhttp.HandlerFor(p.Registry, promhttp.HandlerOpts{})
}

// Middleware satisfies httpadapter.MetricsProvider for HTTP instrumentation.
func (p *Provider) Middleware(next http.Handler) http.Handler {
	if p == nil || p.HTTPMetrics == nil {
		return next
	}
	return p.HTTPMetrics.Middleware(next)
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
