package responder

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// SetTelemetryHeaders sets correlation and latency headers on the outbound response.
func SetTelemetryHeaders(w http.ResponseWriter, r *http.Request, upstreamDurMs ...int) {
	reqID := GetRequestID(r.Context())
	if reqID != "" {
		w.Header().Set(HeaderRequestID, reqID)
	}

	gwDur := ElapsedMillis(r.Context())
	w.Header().Set(HeaderGatewayLatency, fmt.Sprintf("%.2fms", gwDur))

	if len(upstreamDurMs) > 0 && upstreamDurMs[0] > 0 {
		w.Header().Set(HeaderServerTiming, fmt.Sprintf("gateway;dur=%.2f, upstream;dur=%d", gwDur, upstreamDurMs[0]))
	} else {
		w.Header().Set(HeaderServerTiming, fmt.Sprintf("gateway;dur=%.2f", gwDur))
	}
}

// Error writes a standardized hybrid error envelope to the response writer.
func Error(w http.ResponseWriter, r *http.Request, status int, code ErrorCode, message string, extraHeaders ...map[string]string) {
	SetTelemetryHeaders(w, r)

	for _, headers := range extraHeaders {
		for k, v := range headers {
			w.Header().Set(k, v)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	reqID := GetRequestID(r.Context())
	envelope := ErrorEnvelope{
		Status:    false,
		Message:   message,
		Code:      code,
		RequestID: reqID,
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
	}

	_ = json.NewEncoder(w).Encode(envelope)
}

// JSON writes a JSON representation to the response writer with telemetry headers.
func JSON(w http.ResponseWriter, r *http.Request, status int, data any, extraHeaders ...map[string]string) {
	SetTelemetryHeaders(w, r)

	for _, headers := range extraHeaders {
		for k, v := range headers {
			w.Header().Set(k, v)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
