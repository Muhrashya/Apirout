package executor

import (
	"testing"
	"time"

	"github.com/apirout/apirout/internal/core/models"
	"github.com/apirout/apirout/internal/core/upstream"
)

func TestHandler_CircuitBreakerStateAndEventSync(t *testing.T) {
	h := NewHandler(nil, nil, nil, nil, nil, nil)

	cb := h.GetCircuitBreaker()
	if cb == nil {
		t.Fatal("expected handler to initialize passive circuit breaker")
	}

	target1 := models.Target{ID: "tgt-100", Host: "127.0.0.1", Port: 9001}
	target2 := models.Target{ID: "tgt-200", Host: "127.0.0.1", Port: 9002}
	targets := []models.Target{target1, target2}

	// 1. Initial state: both targets available
	avail, allTripped := cb.FilterAvailable(targets)
	if allTripped || len(avail) != 2 {
		t.Fatalf("expected 2 available targets, got %d (allTripped=%v)", len(avail), allTripped)
	}

	// 2. Control Plane EventBus sync: trip tgt-100 via OnTargetHealthChanged
	h.OnTargetHealthChanged("tgt-100", "unhealthy")

	avail, allTripped = cb.FilterAvailable(targets)
	if allTripped || len(avail) != 1 || avail[0].ID != "tgt-200" {
		t.Fatalf("expected tgt-100 tripped and tgt-200 available, got %v", avail)
	}

	// 3. Passive failure tracking: trip tgt-200 with 3 consecutive failures
	cb.RecordFailure("tgt-200")
	cb.RecordFailure("tgt-200")
	avail, allTripped = cb.FilterAvailable(targets)
	if allTripped || len(avail) != 1 {
		t.Fatalf("tgt-200 should still be available before 3rd failure")
	}

	cb.RecordFailure("tgt-200")
	avail, allTripped = cb.FilterAvailable(targets)
	if !allTripped || len(avail) != 0 {
		t.Fatalf("expected all targets tripped, got len=%d allTripped=%v", len(avail), allTripped)
	}

	// 4. Control Plane EventBus sync: recover tgt-100
	h.OnTargetHealthChanged("tgt-100", "healthy")

	avail, allTripped = cb.FilterAvailable(targets)
	if allTripped || len(avail) != 1 || avail[0].ID != "tgt-100" {
		t.Fatalf("expected tgt-100 recovered and available, got %v", avail)
	}

	// 5. Success reset on tgt-100
	cb.RecordSuccess("tgt-100")
	avail, _ = cb.FilterAvailable(targets)
	if len(avail) != 1 || avail[0].ID != "tgt-100" {
		t.Fatalf("expected tgt-100 still available")
	}
}

func TestHandler_CircuitBreakerCanaryRecovery(t *testing.T) {
	cb := upstream.NewPassiveCircuitBreaker(3, 40*time.Millisecond)

	targets := []models.Target{
		{ID: "tgt-single", Host: "127.0.0.1", Port: 8081},
	}

	// Trip the single target
	cb.RecordFailure("tgt-single")
	cb.RecordFailure("tgt-single")
	cb.RecordFailure("tgt-single")

	_, allTripped := cb.FilterAvailable(targets)
	if !allTripped {
		t.Fatal("expected target to be tripped")
	}

	// Wait for cooldown
	time.Sleep(50 * time.Millisecond)

	// FilterAvailable should now allow 1 canary request in half-open state
	avail, allTripped := cb.FilterAvailable(targets)
	if allTripped || len(avail) != 1 {
		t.Fatalf("expected canary request to be allowed in half-open state, got len=%d", len(avail))
	}

	// A second concurrent request before canary finishes should be rejected
	avail2, allTripped2 := cb.FilterAvailable(targets)
	if !allTripped2 || len(avail2) != 0 {
		t.Fatalf("expected subsequent request to be rejected while canary is in-flight")
	}

	// Canary succeeds -> target restored to healthy
	cb.RecordSuccess("tgt-single")
	avail3, allTripped3 := cb.FilterAvailable(targets)
	if allTripped3 || len(avail3) != 1 {
		t.Fatalf("expected target restored to healthy after canary success")
	}
}
