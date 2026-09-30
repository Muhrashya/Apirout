package httputil

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strings"
)

// WriteJSON sends a JSON response with status code
func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// WriteJSONError sends a JSON error response with logging for 5xx errors
func WriteJSONError(w http.ResponseWriter, status int, message string) {
	if status >= 500 {
		log.Printf("[ERROR] HTTP %d: %s", status, message)
	}
	WriteJSON(w, status, map[string]interface{}{
		"status":  false,
		"message": message,
		"error":   message,
	})
}

// WriteJSONData writes a JSON response with a data wrapper
func WriteJSONData(w http.ResponseWriter, status int, data interface{}) {
	WriteJSON(w, status, map[string]interface{}{"data": data})
}

// GetClientIP extracts the real client IP address considering proxy headers or remote address.
func GetClientIP(r *http.Request) string {
	headers := []string{
		"CF-Connecting-IP",
		"Fastly-Client-IP",
		"True-Client-IP",
		"X-Original-Forwarded-For",
		"X-Forwarded-For",
		"X-Real-IP",
	}

	for _, header := range headers {
		if val := r.Header.Get(header); val != "" {
			return strings.TrimSpace(strings.Split(val, ",")[0])
		}
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
