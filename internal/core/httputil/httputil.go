package httputil

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/apirout/apirout/internal/core/iputil"
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

// GetClientIP extracts the real client IP address considering configured trusted proxy policies.
func GetClientIP(r *http.Request) string {
	return iputil.ExtractClientIP(r)
}

// SetTrustedProxies sets the global trusted proxy CIDR policy for client IP resolution.
func SetTrustedProxies(rules string) {
	iputil.SetGlobalTrustedProxies(rules)
}
