package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"sync"
)

// gzipResponseWriter wraps http.ResponseWriter with gzip compression
type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
	wroteHeader bool
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func (w *gzipResponseWriter) WriteHeader(code int) {
	if !w.wroteHeader {
		w.ResponseWriter.Header().Del("Content-Length")
		w.ResponseWriter.Header().Set("Content-Encoding", "gzip")
		w.wroteHeader = true
	}
	w.ResponseWriter.WriteHeader(code)
}

// gzip writer pool to reuse gzip writers
var gzipWriterPool = sync.Pool{
	New: func() interface{} {
		return gzip.NewWriter(io.Discard)
	},
}

// GzipMiddleware adds gzip compression to responses
func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if client accepts gzip
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		// Skip for small responses or specific paths
		path := r.URL.Path
		if strings.HasPrefix(path, "/api/") && r.Method == "GET" {
			// Only compress API GET responses
		} else if strings.HasSuffix(path, ".js") || strings.HasSuffix(path, ".css") || strings.HasSuffix(path, ".json") {
			// Compress static assets
		} else {
			next.ServeHTTP(w, r)
			return
		}

		// Get gzip writer from pool
		gz := gzipWriterPool.Get().(*gzip.Writer)
		gz.Reset(w)
		defer func() {
			gz.Close()
			gzipWriterPool.Put(gz)
		}()

		gzw := &gzipResponseWriter{
			Writer:         gz,
			ResponseWriter: w,
		}

		next.ServeHTTP(gzw, r)
	})
}

// ConditionalGzipMiddleware creates a middleware that can be toggled
type ConditionalGzipMiddleware struct {
	enabled bool
	minSize int // minimum response size to compress
}

// NewConditionalGzipMiddleware creates a new conditional gzip middleware
func NewConditionalGzipMiddleware(enabled bool, minSize int) *ConditionalGzipMiddleware {
	return &ConditionalGzipMiddleware{
		enabled: enabled,
		minSize: minSize,
	}
}

// Handler returns the middleware handler
func (m *ConditionalGzipMiddleware) Handler(next http.Handler) http.Handler {
	if !m.enabled {
		return next
	}
	return GzipMiddleware(next)
}

// SetEnabled toggles compression on/off
func (m *ConditionalGzipMiddleware) SetEnabled(enabled bool) {
	m.enabled = enabled
}
