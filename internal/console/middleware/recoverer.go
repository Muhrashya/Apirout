package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/apirout/apirout/internal/core/httputil"
)

// Recoverer converts panics into 500 responses and structured log entries
// instead of letting net/http close the connection abruptly.
func Recoverer(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logger.Error("panic recovered",
						"panic", rec,
						"stack", string(debug.Stack()),
						"method", r.Method,
						"path", r.URL.Path,
						"request_id", GetRequestID(r.Context()),
					)
					httputil.WriteJSONError(w, http.StatusInternalServerError, "Internal server error")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
