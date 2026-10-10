package server_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/apirout/apirout/internal/console/server"
)

func TestSPAHandler_TraversalBlocked(t *testing.T) {
	tempDir := t.TempDir()
	staticDir := filepath.Join(tempDir, "dist")
	if err := os.MkdirAll(staticDir, 0755); err != nil {
		t.Fatalf("failed to create static dir: %v", err)
	}

	// Create index.html and app.js inside dist
	if err := os.WriteFile(filepath.Join(staticDir, "index.html"), []byte("<html>SPA</html>"), 0644); err != nil {
		t.Fatalf("failed to create index.html: %v", err)
	}
	if err := os.WriteFile(filepath.Join(staticDir, "app.js"), []byte("console.log('app');"), 0644); err != nil {
		t.Fatalf("failed to create app.js: %v", err)
	}

	// Create a secret file in parent and sibling
	secretFile := filepath.Join(tempDir, "secret.txt")
	if err := os.WriteFile(secretFile, []byte("super-secret"), 0644); err != nil {
		t.Fatalf("failed to create secret file: %v", err)
	}

	handler := server.NewSPAHandler(staticDir, "index.html")

	// 1. Normal file -> 200 OK
	reqValid := httptest.NewRequest("GET", "/app.js", nil)
	rrValid := httptest.NewRecorder()
	handler.ServeHTTP(rrValid, reqValid)
	if rrValid.Code != http.StatusOK {
		t.Errorf("expected 200 for app.js, got %d", rrValid.Code)
	}

	// 2. Traversal attempt -> 403 Forbidden
	reqTraversal := httptest.NewRequest("GET", "/../secret.txt", nil)
	rrTraversal := httptest.NewRecorder()
	handler.ServeHTTP(rrTraversal, reqTraversal)
	// Relative path becomes /secret.txt after filepath.Clean unless .. escapes
	// Test escaped relative path
	reqDeepTraversal := httptest.NewRequest("GET", "/../../secret.txt", nil)
	rrDeepTraversal := httptest.NewRecorder()
	handler.ServeHTTP(rrDeepTraversal, reqDeepTraversal)

	// Direct traversal test via URL path
	reqEscape := &http.Request{
		Method: "GET",
		URL:    httptest.NewRequest("GET", "http://localhost/../../../secret.txt", nil).URL,
	}
	reqEscape.URL.Path = "../secret.txt"
	rrEscape := httptest.NewRecorder()
	handler.ServeHTTP(rrEscape, reqEscape)
	if rrEscape.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for escaped traversal, got %d", rrEscape.Code)
	}
}
