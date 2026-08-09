// Package metrics exposes Prometheus instrumentation for the HTTP layer.
package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Instrumented metrics (registered via promauto on first use).
var (
	// ReqTotal counts requests by route, method and HTTP status.
	ReqTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "docuflow_http_requests_total",
		Help: "Total HTTP requests processed.",
	}, []string{"route", "method", "status"})

	// ReqDuration measures request latency in seconds.
	ReqDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "docuflow_http_request_duration_seconds",
		Help:    "HTTP request latency in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"route", "method"})

	// InFlight gauges requests currently being handled.
	InFlight = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "docuflow_http_in_flight_requests",
		Help: "Requests currently in flight.",
	})

	// OutboxPending gauges pending outbox messages (set by the dispatcher).
	OutboxPending = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "docuflow_outbox_pending_messages",
		Help: "Number of outbox messages waiting to be published.",
	})

	// BreakerState is 0=closed 1=open 2=half-open per named dependency.
	BreakerState = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "docuflow_circuit_breaker_state",
		Help: "Circuit breaker state: 0 closed, 1 open, 2 half-open.",
	}, []string{"dependency"})
)

// Observe records one request.
func Observe(route, method, status string, start time.Time) {
	ReqTotal.WithLabelValues(route, method, status).Inc()
	ReqDuration.WithLabelValues(route, method).Observe(time.Since(start).Seconds())
}
