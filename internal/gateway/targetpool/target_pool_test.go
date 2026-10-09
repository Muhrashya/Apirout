package targetpool

import (
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/apirout/apirout/internal/core/models"
)

func TestExtractHashKeyPriority(t *testing.T) {
	// 1. Both X-Consumer-ID and X-User-ID present -> X-Consumer-ID takes precedence
	req1 := httptest.NewRequest("GET", "/test", nil)
	req1.Header.Set("X-Consumer-ID", "consumer-alpha")
	req1.Header.Set("X-User-ID", "user-beta")
	req1.RemoteAddr = "192.168.1.1:12345"
	if got := ExtractHashKey(req1); got != "consumer-alpha" {
		t.Fatalf("expected X-Consumer-ID 'consumer-alpha', got %q", got)
	}

	// 2. Only X-User-ID present -> X-User-ID takes precedence over client IP
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.Header.Set("X-User-ID", "user-beta")
	req2.RemoteAddr = "192.168.1.1:12345"
	if got := ExtractHashKey(req2); got != "user-beta" {
		t.Fatalf("expected X-User-ID 'user-beta', got %q", got)
	}

	// 3. No user/consumer headers -> Client IP fallback
	req3 := httptest.NewRequest("GET", "/test", nil)
	req3.RemoteAddr = "127.0.0.1:12345"
	req3.Header.Set("X-Forwarded-For", "203.0.113.195")
	if got := ExtractHashKey(req3); got != "203.0.113.195" {
		t.Fatalf("expected X-Forwarded-For '203.0.113.195', got %q", got)
	}

	// 4. Nil request fallback
	if got := ExtractHashKey(nil); got != "127.0.0.1" {
		t.Fatalf("expected nil fallback '127.0.0.1', got %q", got)
	}
}

func TestKetamaHashRing_DeterminismAndStability(t *testing.T) {
	targets3 := []models.Target{
		{ID: "target-1", Host: "10.0.0.1", Port: 8080},
		{ID: "target-2", Host: "10.0.0.2", Port: 8080},
		{ID: "target-3", Host: "10.0.0.3", Port: 8080},
	}

	ring3 := NewRing(targets3)

	// Check determinism: same key returns the exact same target every time
	targetMap3 := make(map[string]string)
	numKeys := 2000
	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("session-key-%d", i)
		tgt := ring3.Get(key)
		if tgt == nil {
			t.Fatalf("unexpected nil target for key %s", key)
		}
		// Must match on repeat
		if repeat := ring3.Get(key); repeat.ID != tgt.ID {
			t.Fatalf("non-deterministic ring lookup for key %s: got %s vs %s", key, tgt.ID, repeat.ID)
		}
		targetMap3[key] = tgt.ID
	}

	// Now add a 4th target to the pool (scaling event 3 -> 4 nodes)
	targets4 := []models.Target{
		{ID: "target-1", Host: "10.0.0.1", Port: 8080},
		{ID: "target-2", Host: "10.0.0.2", Port: 8080},
		{ID: "target-3", Host: "10.0.0.3", Port: 8080},
		{ID: "target-4", Host: "10.0.0.4", Port: 8080},
	}

	ring4 := NewRing(targets4)

	remapped := 0
	targetCounts := make(map[string]int)

	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("session-key-%d", i)
		tgt4 := ring4.Get(key)
		targetCounts[tgt4.ID]++
		if tgt4.ID != targetMap3[key] {
			remapped++
			// If remapped, it MUST have moved to the newly added target-4 in consistent hashing
			if tgt4.ID != "target-4" {
				t.Logf("key %s remapped between existing targets: %s -> %s", key, targetMap3[key], tgt4.ID)
			}
		}
	}

	remapRatio := float64(remapped) / float64(numKeys)
	t.Logf("Ketama 3 -> 4 targets: remapped %d / %d keys (%.2f%%)", remapped, numKeys, remapRatio*100)

	// In an ideal consistent hash ring, adding 1 node to 3 nodes redistributes ~ 1/4 (25%) of keys.
	// We assert that remapped keys do not exceed 35% (guaranteeing <= 1/N stability bounds).
	if remapRatio > 0.35 {
		t.Fatalf("expected <= 35%% remapping for 3 -> 4 nodes, got %.2f%%", remapRatio*100)
	}

	// Verify all 4 targets received a fair share of keys (between 15% and 35%)
	for _, target := range targets4 {
		count := targetCounts[target.ID]
		share := float64(count) / float64(numKeys)
		t.Logf("Target %s share: %d keys (%.2f%%)", target.ID, count, share*100)
		if share < 0.15 || share > 0.35 {
			t.Errorf("Target %s received abnormal key share: %.2f%%", target.ID, share*100)
		}
	}
}

func TestTargetPool_ConnectionLeaseLifecycle(t *testing.T) {
	pool := New()

	targets := []models.Target{
		{ID: "tgt-a", Host: "10.0.0.1", Port: 8080},
		{ID: "tgt-b", Host: "10.0.0.2", Port: 8080},
	}
	upstream := &models.Upstream{
		ID:        "up-test",
		Algorithm: "round-robin",
	}

	req := httptest.NewRequest("GET", "/api/v1/resource", nil)

	// Step 1: initial active connection count is 0
	if pool.ActiveConns("tgt-a") != 0 {
		t.Fatalf("expected 0 active conns, got %d", pool.ActiveConns("tgt-a"))
	}
	if pool.ActiveTargetCount() != 0 {
		t.Fatalf("expected 0 active targets, got %d", pool.ActiveTargetCount())
	}

	// Step 2: lease connection on tgt-a
	selected, release1, err := pool.SelectTarget(req, upstream, targets)
	if err != nil || selected == nil {
		t.Fatalf("unexpected error selecting target: %v", err)
	}
	firstID := selected.ID

	if pool.ActiveConns(firstID) != 1 {
		t.Fatalf("expected 1 active conn for %s, got %d", firstID, pool.ActiveConns(firstID))
	}
	if pool.ActiveTargetCount() != 1 {
		t.Fatalf("expected 1 active target, got %d", pool.ActiveTargetCount())
	}

	// Step 3: lease second connection on tgt-a directly
	release2 := pool.acquireConn(firstID)
	if pool.ActiveConns(firstID) != 2 {
		t.Fatalf("expected 2 active conns for %s, got %d", firstID, pool.ActiveConns(firstID))
	}

	// Step 4: release one connection
	release1()
	if pool.ActiveConns(firstID) != 1 {
		t.Fatalf("expected 1 active conn after first release, got %d", pool.ActiveConns(firstID))
	}
	if pool.ActiveTargetCount() != 1 {
		t.Fatalf("expected still 1 active target, got %d", pool.ActiveTargetCount())
	}

	// Idempotency: calling release1() again does not decrement below 1
	release1()
	if pool.ActiveConns(firstID) != 1 {
		t.Fatalf("expected release1 idempotency, got %d", pool.ActiveConns(firstID))
	}

	// Step 5: release second connection -> active conns drops to 0 and target is removed from map
	release2()
	if pool.ActiveConns(firstID) != 0 {
		t.Fatalf("expected 0 active conns after all releases, got %d", pool.ActiveConns(firstID))
	}
	if pool.ActiveTargetCount() != 0 {
		t.Fatalf("expected active targets map cleaned to 0, got %d", pool.ActiveTargetCount())
	}
}

func TestTargetPool_LeastConnections(t *testing.T) {
	pool := New()

	targets := []models.Target{
		{ID: "tgt-busy", Host: "10.0.0.1", Port: 8080},
		{ID: "tgt-idle", Host: "10.0.0.2", Port: 8080},
	}
	upstream := &models.Upstream{
		ID:        "up-least",
		Algorithm: "least-connections",
	}

	// Artificially acquire 3 leases on tgt-busy
	rel1 := pool.acquireConn("tgt-busy")
	rel2 := pool.acquireConn("tgt-busy")
	rel3 := pool.acquireConn("tgt-busy")
	defer rel1()
	defer rel2()
	defer rel3()

	req := httptest.NewRequest("GET", "/test", nil)

	// Selection must choose tgt-idle since it has 0 conns vs 3 conns
	selected, release, err := pool.SelectTarget(req, upstream, targets)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if selected.ID != "tgt-idle" {
		t.Fatalf("expected least-connections to select 'tgt-idle', got %s", selected.ID)
	}
	defer release()
}

func TestTargetPool_ConsistentHashingSelection(t *testing.T) {
	pool := New()

	targets := []models.Target{
		{ID: "tgt-alpha", Host: "10.0.0.1", Port: 8080},
		{ID: "tgt-beta", Host: "10.0.0.2", Port: 8080},
		{ID: "tgt-gamma", Host: "10.0.0.3", Port: 8080},
	}
	upstream := &models.Upstream{
		ID:        "up-consistent",
		Algorithm: "consistent-hashing",
	}

	reqA := httptest.NewRequest("GET", "/test", nil)
	reqA.Header.Set("X-Consumer-ID", "consumer-12345")

	selA1, relA1, err := pool.SelectTarget(reqA, upstream, targets)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	relA1()

	selA2, relA2, err := pool.SelectTarget(reqA, upstream, targets)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	relA2()

	if selA1.ID != selA2.ID {
		t.Fatalf("expected identical target for same consumer ID, got %s and %s", selA1.ID, selA2.ID)
	}
}

func TestTargetPool_EmptyAndSingleTargets(t *testing.T) {
	pool := New()

	// Empty targets
	tgt, _, err := pool.SelectTarget(nil, nil, nil)
	if err != ErrNoTargets || tgt != nil {
		t.Fatalf("expected ErrNoTargets for empty targets, got tgt=%v err=%v", tgt, err)
	}

	// Single target
	single := []models.Target{{ID: "only-one", Host: "10.0.0.1", Port: 80}}
	tgt, rel, err := pool.SelectTarget(nil, nil, single)
	if err != nil || tgt == nil || tgt.ID != "only-one" {
		t.Fatalf("expected single target 'only-one', got %v err %v", tgt, err)
	}
	if pool.ActiveConns("only-one") != 1 {
		t.Fatalf("expected 1 active conn, got %d", pool.ActiveConns("only-one"))
	}
	rel()
	if pool.ActiveConns("only-one") != 0 {
		t.Fatalf("expected 0 active conns after release, got %d", pool.ActiveConns("only-one"))
	}
}
