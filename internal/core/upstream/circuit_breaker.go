package upstream

import (
	"sync"
	"time"

	"github.com/apirout/apirout/internal/core/models"
)

// CircuitState represents the current health state of an upstream target
type CircuitState string

const (
	StateHealthy  CircuitState = "HEALTHY"
	StateTripped  CircuitState = "TRIPPED"
	StateHalfOpen CircuitState = "HALF_OPEN"
)

const (
	DefaultFailureThreshold = 3
	DefaultCooldownDuration = 15 * time.Second
)

type targetStats struct {
	state            CircuitState
	consecutiveFails int
	trippedAt        time.Time
	canaryInFlight   bool
}

// PassiveCircuitBreaker provides edge-local target failure tracking and fast failover
type PassiveCircuitBreaker struct {
	mu               sync.RWMutex
	stats            map[string]*targetStats // targetID -> stats
	failureThreshold int
	cooldown         time.Duration
}

// NewPassiveCircuitBreaker creates a new circuit breaker with default thresholds
func NewPassiveCircuitBreaker(threshold int, cooldown time.Duration) *PassiveCircuitBreaker {
	if threshold <= 0 {
		threshold = DefaultFailureThreshold
	}
	if cooldown <= 0 {
		cooldown = DefaultCooldownDuration
	}

	return &PassiveCircuitBreaker{
		stats:            make(map[string]*targetStats),
		failureThreshold: threshold,
		cooldown:         cooldown,
	}
}

// RecordSuccess marks a successful request against a target, resetting failures and recovering half-open states
func (cb *PassiveCircuitBreaker) RecordSuccess(targetID string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	s, exists := cb.stats[targetID]
	if !exists {
		return
	}

	s.consecutiveFails = 0
	s.state = StateHealthy
	s.canaryInFlight = false
}

// RecordFailure records a connection error or 5xx response against a target
func (cb *PassiveCircuitBreaker) RecordFailure(targetID string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	s, exists := cb.stats[targetID]
	if !exists {
		s = &targetStats{state: StateHealthy}
		cb.stats[targetID] = s
	}

	s.consecutiveFails++
	s.canaryInFlight = false

	if s.consecutiveFails >= cb.failureThreshold || s.state == StateHalfOpen {
		s.state = StateTripped
		s.trippedAt = time.Now()
	}
}

// TripTarget immediately trips a target into cooldown (e.g. from active health check event)
func (cb *PassiveCircuitBreaker) TripTarget(targetID string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	s, exists := cb.stats[targetID]
	if !exists {
		s = &targetStats{}
		cb.stats[targetID] = s
	}
	s.state = StateTripped
	s.trippedAt = time.Now()
	s.consecutiveFails = cb.failureThreshold
}

// ResetTarget resets a target back to healthy status
func (cb *PassiveCircuitBreaker) ResetTarget(targetID string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	delete(cb.stats, targetID)
}

// FilterAvailable evaluates all targets in an upstream pool and returns available candidates
// If all targets in the pool are tripped, it returns an empty slice and allTripped = true
func (cb *PassiveCircuitBreaker) FilterAvailable(targets []models.Target) (available []models.Target, allTripped bool) {
	if len(targets) == 0 {
		return nil, false
	}

	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()
	available = make([]models.Target, 0, len(targets))

	for _, t := range targets {
		s, exists := cb.stats[t.ID]
		if !exists {
			available = append(available, t)
			continue
		}

		switch s.state {
		case StateHealthy:
			available = append(available, t)

		case StateTripped:
			// Check if cooldown has elapsed
			if now.Sub(s.trippedAt) >= cb.cooldown {
				s.state = StateHalfOpen
				s.canaryInFlight = true
				available = append(available, t) // Allow 1 canary probe
			}

		case StateHalfOpen:
			if !s.canaryInFlight {
				s.canaryInFlight = true
				available = append(available, t)
			}
		}
	}

	if len(available) == 0 && len(targets) > 0 {
		return nil, true
	}

	return available, false
}

// Allow evaluates whether a target or direct service is available to receive traffic
func (cb *PassiveCircuitBreaker) Allow(targetID string) bool {
	if cb == nil || targetID == "" {
		return true
	}

	cb.mu.Lock()
	defer cb.mu.Unlock()

	s, exists := cb.stats[targetID]
	if !exists {
		return true
	}

	now := time.Now()
	switch s.state {
	case StateHealthy:
		return true
	case StateTripped:
		if now.Sub(s.trippedAt) >= cb.cooldown {
			s.state = StateHalfOpen
			s.canaryInFlight = true
			return true
		}
		return false
	case StateHalfOpen:
		if !s.canaryInFlight {
			s.canaryInFlight = true
			return true
		}
		return false
	}
	return true
}

