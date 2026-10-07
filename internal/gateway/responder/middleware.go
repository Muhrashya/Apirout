package responder

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// HeaderRequestID is the canonical HTTP header name for request correlation tracing.
const HeaderRequestID = "X-Request-ID"

// HeaderGatewayLatency is the HTTP header reporting gateway routing overhead.
const HeaderGatewayLatency = "X-Gateway-Latency"

// HeaderServerTiming is the W3C RFC standard header reporting multi-metric latency.
const HeaderServerTiming = "Server-Timing"

// Middleware intercepts all inbound requests at the Data Plane ingress boundary, ensuring an
// authoritative request ID exists, recording the arrival timestamp, and injecting state into context.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := strings.TrimSpace(r.Header.Get(HeaderRequestID))
		if reqID == "" {
			reqID = strings.TrimSpace(r.Header.Get("X-Correlation-ID"))
		}
		if reqID == "" {
			reqID = fmt.Sprintf("req-%s", uuid.New().String())
		}

		start := time.Now()
		ctx := WithRequestContext(r.Context(), reqID, start)

		// Pre-populate response header so early returns or streaming writes always contain the correlation trace.
		w.Header().Set(HeaderRequestID, reqID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
