package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/apirout/apirout/internal/gateway/metrics"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// MetricsTracker instruments HTTP requests and exports to Prometheus
func MetricsTracker(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(rw, r)

		duration := time.Since(start).Seconds()

		// Sanitize path for metrics cardinality (avoid unique query string exploding cardinality)
		path := r.URL.Path
		if len(path) > 32 {
			path = path[:32] + "..."
		}

		statusStr := fmt.Sprintf("%d", rw.statusCode)
		metrics.HTTPRequestsTotal.WithLabelValues(r.Method, path, statusStr).Inc()
		metrics.HTTPRequestDuration.WithLabelValues(r.Method, path).Observe(duration)
	})
}
