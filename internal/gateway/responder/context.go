package responder

import (
	"context"
	"time"
)

type contextKey int

const (
	requestIDKey contextKey = iota
	startTimeKey
)

// WithRequestContext wraps the context with an authoritative request correlation ID and start timestamp.
func WithRequestContext(ctx context.Context, requestID string, startTime time.Time) context.Context {
	ctx = context.WithValue(ctx, requestIDKey, requestID)
	ctx = context.WithValue(ctx, startTimeKey, startTime)
	return ctx
}

// GetRequestID retrieves the correlation request ID from context.
func GetRequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Value(requestIDKey).(string); ok {
		return v
	}
	return ""
}

// GetStartTime retrieves the monotonic arrival time from context.
func GetStartTime(ctx context.Context) time.Time {
	if ctx == nil {
		return time.Now()
	}
	if v, ok := ctx.Value(startTimeKey).(time.Time); ok && !v.IsZero() {
		return v
	}
	return time.Now()
}

// ElapsedMillis returns the elapsed time in milliseconds since request arrival.
func ElapsedMillis(ctx context.Context) float64 {
	start := GetStartTime(ctx)
	return float64(time.Since(start).Nanoseconds()) / 1e6
}
