package responder

import (
	"fmt"
	"net/http"
	"time"
)

// CircuitBreakerTracker defines the failure/success reporting interface for upstream targets.
type CircuitBreakerTracker interface {
	RecordSuccess(targetID string)
	RecordFailure(targetID string)
}

// HandleProxyError handles reverse proxy network failures, recording circuit breaker stats,
// tracking latency, and emitting a sanitized HTTP 502 error envelope to callers.
func HandleProxyError(
	rw http.ResponseWriter,
	r *http.Request,
	serviceName string,
	proxyErr error,
	targetID string,
	cb CircuitBreakerTracker,
	upDuration *int,
	upStart time.Time,
) {
	if upDuration != nil && !upStart.IsZero() {
		*upDuration = int(time.Since(upStart).Milliseconds())
	}

	if cb != nil && targetID != "" {
		cb.RecordFailure(targetID)
	}

	sanitizedMsg := fmt.Sprintf("Backend service %s unreachable", serviceName)
	if serviceName == "" {
		sanitizedMsg = "Backend service unreachable"
	}

	Error(rw, r, http.StatusBadGateway, ErrCodeUpstreamUnreachable, sanitizedMsg)
}

// ModifyProxyResponse transforms outbound responses from upstream targets, updating the
// circuit breaker, stripping internal backend banners, and injecting correlation and Server-Timing headers.
func ModifyProxyResponse(
	r *http.Request,
	res *http.Response,
	targetID string,
	cb CircuitBreakerTracker,
	upDuration *int,
	upStart time.Time,
) error {
	dur := 0
	if !upStart.IsZero() {
		dur = int(time.Since(upStart).Milliseconds())
	}
	if upDuration != nil {
		*upDuration = dur
	}

	if cb != nil && targetID != "" {
		if res.StatusCode >= 500 {
			cb.RecordFailure(targetID)
		} else {
			cb.RecordSuccess(targetID)
		}
	}

	// Strip hop-by-hop & internal server disclosure headers
	res.Header.Del("Server")
	res.Header.Del("X-Powered-By")
	res.Header.Del("X-AspNet-Version")
	res.Header.Del("X-Runtime")

	// Inject correlation and telemetry headers into upstream response
	reqID := GetRequestID(r.Context())
	if reqID != "" {
		res.Header.Set(HeaderRequestID, reqID)
	}

	gwDur := ElapsedMillis(r.Context())
	res.Header.Set(HeaderGatewayLatency, fmt.Sprintf("%.2fms", gwDur))
	res.Header.Set(HeaderServerTiming, fmt.Sprintf("gateway;dur=%.2f, upstream;dur=%d", gwDur, dur))

	return nil
}
