package middleware

import (
	"net/http"
	"strings"
)

// SecurityResponseWriter intercepts headers to modify them before writing
type SecurityResponseWriter struct {
	http.ResponseWriter
	wroteHeader bool
}

func (w *SecurityResponseWriter) secureCookies() {
	cookies := w.Header()["Set-Cookie"]
	if len(cookies) > 0 {
		w.Header().Del("Set-Cookie")
		for _, cookie := range cookies {
			lowerCookie := strings.ToLower(cookie)
			if !strings.Contains(lowerCookie, "secure") {
				cookie += "; Secure"
			}
			if !strings.Contains(lowerCookie, "httponly") {
				cookie += "; HttpOnly"
			}
			w.Header().Add("Set-Cookie", cookie)
		}
	}
}

func (w *SecurityResponseWriter) WriteHeader(statusCode int) {
	if w.wroteHeader {
		return
	}
	w.secureCookies()
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *SecurityResponseWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.secureCookies()
		w.wroteHeader = true
	}
	return w.ResponseWriter.Write(b)
}

// SecurityHeaders adds security-related headers to responses
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		// HSTS header
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		// Content-Security-Policy: Secure configuration (no inline scripts/styles, explicit fallbacks)
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com; img-src 'self' data:; connect-src 'self'; base-uri 'self'; form-action 'self'; frame-ancestors 'none'; object-src 'none';")
		
		sw := &SecurityResponseWriter{ResponseWriter: w}
		next.ServeHTTP(sw, r)
	})
}
