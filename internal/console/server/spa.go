package server

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// SPAHandler serves a single page application
type SPAHandler struct {
	staticPath string
	indexPath  string
}

// NewSPAHandler creates a new SPA handler
func NewSPAHandler(staticPath, indexPath string) *SPAHandler {
	return &SPAHandler{
		staticPath: staticPath,
		indexPath:  indexPath,
	}
}

// ServeHTTP serves static files or falls back to index.html for SPA routing
func (h *SPAHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Clean the path to prevent directory traversal and ensure it's relative
	cleanPath := filepath.Clean(r.URL.Path)
	// Strip leading slash to make it relative for filepath.Join
	relativePath := strings.TrimPrefix(cleanPath, "/")

	path := filepath.Join(h.staticPath, relativePath)

	// Defense-in-depth: Ensure resolved path does not escape staticPath directory
	if cleanStatic, err := filepath.Abs(h.staticPath); err == nil {
		if cleanTarget, err := filepath.Abs(path); err == nil {
			if !strings.HasPrefix(cleanTarget, cleanStatic) {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
		}
	}

	// Check if file exists
	fi, err := os.Stat(path)
	if err != nil {
		if !os.IsNotExist(err) {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		// Serve index.html for SPA routing with no-cache headers
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		http.ServeFile(w, r, filepath.Join(h.staticPath, h.indexPath))
		return
	}

	if fi.IsDir() {
		// Serve index.html for SPA routing with no-cache headers
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		http.ServeFile(w, r, filepath.Join(h.staticPath, h.indexPath))
		return
	}

	// Serve static file
	http.FileServer(http.Dir(h.staticPath)).ServeHTTP(w, r)
}
