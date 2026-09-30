package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTPRequestsTotal counts all requests processed by the Data Plane Gateway
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "apirout_http_requests_total",
			Help: "Total number of HTTP requests processed by Data Plane Gateway",
		},
		[]string{"method", "path", "status"},
	)

	// HTTPRequestDuration tracks latency of API requests
	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "apirout_http_request_duration_seconds",
			Help:    "Histogram of request latencies in seconds",
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"method", "path"},
	)

	// TargetDBDuration tracks latency of queries sent to target databases
	TargetDBDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "apirout_target_db_duration_seconds",
			Help:    "Histogram of target database query latencies",
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
		},
		[]string{"db_type", "connection_id"},
	)
)
