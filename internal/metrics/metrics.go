package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// RequestsTotal counts incoming webhook requests by status and path.
	RequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "webhook_requests_total",
			Help: "Total number of webhook requests received.",
		},
		[]string{"status", "path"},
	)

	// RequestDuration observes webhook request processing duration.
	RequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "webhook_request_duration_seconds",
			Help:    "Duration of webhook request processing in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"path"},
	)

	// DispatchTotal counts dispatch attempts by rule, target, and status.
	DispatchTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "webhook_dispatch_total",
			Help: "Total number of webhook dispatch attempts.",
		},
		[]string{"rule_name", "target", "status"},
	)

	// DispatchDuration observes dispatch duration by rule.
	DispatchDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "webhook_dispatch_duration_seconds",
			Help:    "Duration of webhook dispatch in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"rule_name"},
	)

	// RulesLoaded tracks the current number of loaded rules.
	RulesLoaded = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "webhook_rules_loaded",
			Help: "Number of currently loaded webhook rules.",
		},
	)
)
