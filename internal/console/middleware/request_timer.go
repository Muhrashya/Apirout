package middleware

import (
	"context"
	"net/http"
	"time"
)

type contextKey string

const StartTimeKey contextKey = "startTime"

// RequestTimer adds the start time to the request context
func RequestTimer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ctx := context.WithValue(r.Context(), StartTimeKey, start)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetStartTime retrieves the start time from the context
func GetStartTime(ctx context.Context) (time.Time, bool) {
	start, ok := ctx.Value(StartTimeKey).(time.Time)
	return start, ok
}
