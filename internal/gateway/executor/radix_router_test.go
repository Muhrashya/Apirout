package executor

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/apirout/apirout/internal/core/models"
	"github.com/apirout/apirout/internal/core/router"
)

func TestHandler_RadixRouterMatchingAndMethodNotAllowed(t *testing.T) {
	// Setup handler with in-memory radix router
	h := NewHandler(nil, nil, nil, nil, nil, nil)

	r := router.NewRadixRouter()
	_ = r.AddRoute(&models.Endpoint{
		ID:       "ep-get-users",
		Name:     "Get Users",
		Path:     "/api/v1/users",
		Method:   "GET",
		Enabled:  true,
		IsPublic: true,
	})
	_ = r.AddRoute(&models.Endpoint{
		ID:       "ep-post-users",
		Name:     "Create User",
		Path:     "/api/v1/users",
		Method:   "POST",
		Enabled:  true,
		IsPublic: true,
	})
	h.SetRouter(r)

	// 1. Test Method Not Allowed (DELETE)
	req := httptest.NewRequest("DELETE", "/api/v1/users", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405 Method Not Allowed, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	allowHeader := rec.Header().Get("Allow")
	if !strings.Contains(allowHeader, "GET") || !strings.Contains(allowHeader, "POST") {
		t.Fatalf("expected Allow header containing GET, POST, got: %s", allowHeader)
	}

	// 2. Test Not Found route
	req404 := httptest.NewRequest("GET", "/api/v1/nonexistent", nil)
	rec404 := httptest.NewRecorder()

	h.ServeHTTP(rec404, req404)

	if rec404.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rec404.Code)
	}
}

func TestHandler_RadixRouterDynamicAtomicSwap(t *testing.T) {
	h := NewHandler(nil, nil, nil, nil, nil, nil)

	// Initial router has only /api/old
	r1 := router.NewRadixRouter()
	_ = r1.AddRoute(&models.Endpoint{
		ID:       "ep-old",
		Path:     "/api/old",
		Method:   "GET",
		Enabled:  true,
		IsPublic: true,
	})
	h.SetRouter(r1)

	req1 := httptest.NewRequest("DELETE", "/api/old", nil)
	rec1 := httptest.NewRecorder()
	h.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 on old route, got %d", rec1.Code)
	}

	// Hot-swap router with /api/new
	r2 := router.NewRadixRouter()
	_ = r2.AddRoute(&models.Endpoint{
		ID:       "ep-new",
		Path:     "/api/new",
		Method:   "POST",
		Enabled:  true,
		IsPublic: true,
	})
	h.SetRouter(r2)

	// Old route should now be 404
	reqOld := httptest.NewRequest("GET", "/api/old", nil)
	recOld := httptest.NewRecorder()
	h.ServeHTTP(recOld, reqOld)
	if recOld.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for old route after swap, got %d", recOld.Code)
	}

	// New route should be recognized (405 for GET since method is POST)
	reqNew := httptest.NewRequest("GET", "/api/new", nil)
	recNew := httptest.NewRecorder()
	h.ServeHTTP(recNew, reqNew)
	if recNew.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 for GET /api/new, got %d", recNew.Code)
	}
}
