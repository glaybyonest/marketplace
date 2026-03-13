package observability

import (
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	registry            *prometheus.Registry
	inFlightRequests    prometheus.Gauge
	httpRequestsTotal   *prometheus.CounterVec
	httpRequestDuration *prometheus.HistogramVec
	errorsTotal         *prometheus.CounterVec
	auditEventsTotal    *prometheus.CounterVec
}

func NewMetrics(db *pgxpool.Pool) *Metrics {
	registry := prometheus.NewRegistry()
	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	metrics := &Metrics{
		registry: registry,
		inFlightRequests: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "marketplace_http_requests_in_flight",
			Help: "Current number of in-flight HTTP requests.",
		}),
		httpRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "marketplace_http_requests_total",
				Help: "Total number of handled HTTP requests.",
			},
			[]string{"method", "route", "status"},
		),
		httpRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "marketplace_http_request_duration_seconds",
				Help:    "HTTP request duration in seconds.",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "route", "status"},
		),
		errorsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "marketplace_errors_total",
				Help: "Total number of tracked backend errors.",
			},
			[]string{"severity", "code"},
		),
		auditEventsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "marketplace_audit_events_total",
				Help: "Total number of recorded audit events.",
			},
			[]string{"action"},
		),
	}

	registry.MustRegister(
		metrics.inFlightRequests,
		metrics.httpRequestsTotal,
		metrics.httpRequestDuration,
		metrics.errorsTotal,
		metrics.auditEventsTotal,
	)

	if db != nil {
		registry.MustRegister(NewDBPoolCollector(db))
	}

	return metrics
}

func (m *Metrics) Handler() http.Handler {
	if m == nil {
		return http.NotFoundHandler()
	}
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{Registry: m.registry})
}

func (m *Metrics) ObserveHTTPRequest(method, route string, status int, duration time.Duration) {
	if m == nil {
		return
	}
	statusText := strconv.Itoa(status)
	m.httpRequestsTotal.WithLabelValues(method, route, statusText).Inc()
	m.httpRequestDuration.WithLabelValues(method, route, statusText).Observe(duration.Seconds())
}

func (m *Metrics) IncInFlight() {
	if m == nil {
		return
	}
	m.inFlightRequests.Inc()
}

func (m *Metrics) DecInFlight() {
	if m == nil {
		return
	}
	m.inFlightRequests.Dec()
}

func (m *Metrics) RecordError(severity, code string) {
	if m == nil {
		return
	}
	m.errorsTotal.WithLabelValues(severity, code).Inc()
}

func (m *Metrics) RecordAudit(action string) {
	if m == nil {
		return
	}
	m.auditEventsTotal.WithLabelValues(action).Inc()
}
