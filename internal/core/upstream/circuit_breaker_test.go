package upstream

import (
	"testing"
	"time"

	"github.com/apirout/apirout/internal/core/models"
)

func TestPassiveCircuitBreaker_TrippingAndRecovery(t *testing.T) {
	cb := NewPassiveCircuitBreaker(3, 50*time.Millisecond)

	targets := []models.Target{
		{ID: "t-1", Host: "10.0.0.1", Port: 8080},
		{ID: "t-2", Host: "10.0.0.2", Port: 8080},
	}

	// 1. Initial state: both targets available
	avail, allTripped := cb.FilterAvailable(targets)
	if allTripped || len(avail) != 2 {
		t.Fatalf("expected 2 available targets, got %d (allTripped=%v)", len(avail), allTripped)
	}

	// 2. 2 failures on t-1: still available (threshold is 3)
	cb.RecordFailure("t-1")
	cb.RecordFailure("t-1")
	avail, _ = cb.FilterAvailable(targets)
	if len(avail) != 2 {
		t.Fatalf("expected t-1 still available after 2 failures, got %d", len(avail))
	}

	// 3. 3rd failure trips t-1
	cb.RecordFailure("t-1")
	avail, allTripped = cb.FilterAvailable(targets)
	if allTripped || len(avail) != 1 || avail[0].ID != "t-2" {
		t.Fatalf("expected only t-2 available, got %v", avail)
	}

	// 4. Trip t-2 as well -> allTripped should be true
	cb.RecordFailure("t-2")
	cb.RecordFailure("t-2")
	cb.RecordFailure("t-2")
	avail, allTripped = cb.FilterAvailable(targets)
	if !allTripped || len(avail) != 0 {
		t.Fatalf("expected allTripped=true and 0 available, got len=%d allTripped=%v", len(avail), allTripped)
	}

	// 5. Wait for cooldown (50ms) to elapse -> half-open canary probe
	time.Sleep(60 * time.Millisecond)
	avail, allTripped = cb.FilterAvailable(targets)
	if allTripped || len(avail) != 2 {
		t.Fatalf("expected both to enter half-open allowing canary, got %d (allTripped=%v)", len(avail), allTripped)
	}

	// 6. t-1 succeeds on canary -> recovers to healthy
	cb.RecordSuccess("t-1")
	// t-2 fails on canary -> re-trips into cooldown
	cb.RecordFailure("t-2")

	avail, allTripped = cb.FilterAvailable(targets)
	if allTripped || len(avail) != 1 || avail[0].ID != "t-1" {
		t.Fatalf("expected t-1 healthy and t-2 re-tripped, got %v", avail)
	}
}

func TestPassiveCircuitBreaker_ExternalTrip(t *testing.T) {
	cb := NewPassiveCircuitBreaker(3, 10*time.Second)

	targets := []models.Target{
		{ID: "t-1", Host: "10.0.0.1", Port: 8080},
	}

	// Trip directly via EventBus action
	cb.TripTarget("t-1")

	avail, allTripped := cb.FilterAvailable(targets)
	if !allTripped || len(avail) != 0 {
		t.Fatalf("expected target to be tripped by external event")
	}

	// Reset directly
	cb.ResetTarget("t-1")
	avail, allTripped = cb.FilterAvailable(targets)
	if allTripped || len(avail) != 1 {
		t.Fatalf("expected target to be reset to available")
	}
}
