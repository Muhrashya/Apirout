package executor

import (
	"net/http/httptest"
	"testing"

	"github.com/apirout/apirout/internal/core/models"
)

func TestHandler_TargetPoolIntegration(t *testing.T) {
	h := NewHandler(nil, nil, nil, nil, nil, nil)
	pool := h.GetTargetPool()
	if pool == nil {
		t.Fatal("expected handler to initialize non-nil TargetPool")
	}

	targets := []models.Target{
		{ID: "tgt-srv-1", Host: "10.0.1.1", Port: 8080},
		{ID: "tgt-srv-2", Host: "10.0.1.2", Port: 8080},
	}

	// 1. Round-Robin balancing
	upRR := &models.Upstream{ID: "up-rr", Algorithm: "round-robin"}
	req1 := httptest.NewRequest("GET", "/api/test", nil)
	t1, rel1, err := pool.SelectTarget(req1, upRR, targets)
	if err != nil || t1 == nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req2 := httptest.NewRequest("GET", "/api/test", nil)
	t2, rel2, err := pool.SelectTarget(req2, upRR, targets)
	if err != nil || t2 == nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if t1.ID == t2.ID {
		t.Fatalf("expected round robin to cycle different targets: %s vs %s", t1.ID, t2.ID)
	}
	rel1()
	rel2()

	// 2. Consistent Hashing affinity
	upHash := &models.Upstream{ID: "up-hash", Algorithm: "consistent-hashing"}
	reqHashA := httptest.NewRequest("GET", "/api/test", nil)
	reqHashA.Header.Set("X-Consumer-ID", "cons-777")
	th1, relh1, _ := pool.SelectTarget(reqHashA, upHash, targets)
	relh1()

	reqHashB := httptest.NewRequest("GET", "/api/test", nil)
	reqHashB.Header.Set("X-Consumer-ID", "cons-777")
	th2, relh2, _ := pool.SelectTarget(reqHashB, upHash, targets)
	relh2()

	if th1.ID != th2.ID {
		t.Fatalf("expected consistent hash to route same consumer to identical target: %s != %s", th1.ID, th2.ID)
	}

	// 3. Connection lease cleanup
	if pool.ActiveConns(th1.ID) != 0 {
		t.Fatalf("expected lease count to be 0 after releases, got %d", pool.ActiveConns(th1.ID))
	}
}
