package router

import (
	"errors"
	"testing"

	"github.com/apirout/apirout/internal/core/models"
)

func TestRadixRouter_ExactMatch(t *testing.T) {
	r := NewRadixRouter()

	ep1 := &models.Endpoint{
		ID:     "ep-1",
		Path:   "/api/v1/users",
		Method: "GET",
	}
	ep2 := &models.Endpoint{
		ID:     "ep-2",
		Path:   "/api/v1/users",
		Method: "POST",
	}

	_ = r.AddRoute(ep1)
	_ = r.AddRoute(ep2)

	// Test GET match
	res, err := r.Match("/api/v1/users", "GET")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Endpoint.ID != "ep-1" {
		t.Fatalf("expected ep-1, got %s", res.Endpoint.ID)
	}

	// Test POST match
	res, err = r.Match("/api/v1/users", "POST")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Endpoint.ID != "ep-2" {
		t.Fatalf("expected ep-2, got %s", res.Endpoint.ID)
	}

	// Test 405 Method Not Allowed
	res, err = r.Match("/api/v1/users", "DELETE")
	if !errors.Is(err, ErrMethodNotAllowed) {
		t.Fatalf("expected ErrMethodNotAllowed, got %v", err)
	}
	if len(res.AllowedMethods) != 2 {
		t.Fatalf("expected 2 allowed methods, got %v", res.AllowedMethods)
	}
}

func TestRadixRouter_ParamAndWildcardPriority(t *testing.T) {
	r := NewRadixRouter()

	exact := &models.Endpoint{ID: "exact", Path: "/users/me", Method: "GET"}
	param := &models.Endpoint{ID: "param", Path: "/users/:id", Method: "GET"}
	wild := &models.Endpoint{ID: "wild", Path: "/users/*", Method: "GET"}

	_ = r.AddRoute(exact)
	_ = r.AddRoute(param)
	_ = r.AddRoute(wild)

	// 1. Exact should match /users/me
	res, err := r.Match("/users/me", "GET")
	if err != nil || res.Endpoint.ID != "exact" {
		t.Fatalf("expected exact match, got %v (err=%v)", res, err)
	}

	// 2. Param should match /users/123 with extracted param
	res, err = r.Match("/users/123", "GET")
	if err != nil || res.Endpoint.ID != "param" {
		t.Fatalf("expected param match, got %v (err=%v)", res, err)
	}
	if res.Params["id"] != "123" {
		t.Fatalf("expected param id=123, got %s", res.Params["id"])
	}

	// 3. Multi-segment under /users/ should hit wildcard or prefix
	res, err = r.Match("/users/123/profile/photos", "GET")
	if err != nil {
		t.Fatalf("expected wildcard match, got error %v", err)
	}
	if res.Endpoint == nil {
		t.Fatalf("expected non-nil endpoint")
	}
}

func TestRadixRouter_NotFound(t *testing.T) {
	r := NewRadixRouter()
	_ = r.AddRoute(&models.Endpoint{ID: "ep", Path: "/api/hello", Method: "GET"})

	_, err := r.Match("/api/unknown", "GET")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func BenchmarkRadixRouter_Match(b *testing.B) {
	r := NewRadixRouter()
	_ = r.AddRoute(&models.Endpoint{ID: "ep-1", Path: "/api/v1/organizations/:org_id/users/:user_id", Method: "GET"})

	path := "/api/v1/organizations/corp-123/users/user-456"

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		res, err := r.Match(path, "GET")
		if err != nil || res.Endpoint == nil {
			b.Fatal("match failed")
		}
	}
}
