package sync

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// CircuitBreaker tracks consecutive sync failures per company.
// When a company fails `threshold` times in a row, further sync attempts
// are skipped for `cooldown` duration. This prevents one broken iiko server
// from burning worker slots that could sync healthy companies.
//
// Safe for concurrent use by the worker pool.
type CircuitBreaker struct {
	mu        sync.Mutex
	state     map[uuid.UUID]*breakerState
	threshold int
	cooldown  time.Duration
}

type breakerState struct {
	failures int
	openedAt time.Time // zero when closed
}

// NewCircuitBreaker creates a breaker that opens after `threshold` consecutive
// failures and stays open for `cooldown` duration.
func NewCircuitBreaker(threshold int, cooldown time.Duration) *CircuitBreaker {
	if threshold < 1 {
		threshold = 3
	}
	if cooldown < time.Minute {
		cooldown = time.Hour
	}
	return &CircuitBreaker{
		state:     make(map[uuid.UUID]*breakerState),
		threshold: threshold,
		cooldown:  cooldown,
	}
}

// Allow returns true if the company should be synced, false if the breaker is open.
func (cb *CircuitBreaker) Allow(companyID uuid.UUID) bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	s, ok := cb.state[companyID]
	if !ok {
		return true
	}
	// Breaker was opened — check if cooldown has expired
	if !s.openedAt.IsZero() {
		if time.Since(s.openedAt) >= cb.cooldown {
			// Half-open: allow one probe, reset state
			s.openedAt = time.Time{}
			s.failures = 0
			return true
		}
		return false
	}
	return true
}

// RecordFailure increments the failure counter. Opens the breaker if threshold reached.
func (cb *CircuitBreaker) RecordFailure(companyID uuid.UUID) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	s, ok := cb.state[companyID]
	if !ok {
		s = &breakerState{}
		cb.state[companyID] = s
	}
	s.failures++
	if s.failures >= cb.threshold && s.openedAt.IsZero() {
		s.openedAt = time.Now()
	}
}

// RecordSuccess resets the failure counter and closes the breaker.
func (cb *CircuitBreaker) RecordSuccess(companyID uuid.UUID) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if s, ok := cb.state[companyID]; ok {
		s.failures = 0
		s.openedAt = time.Time{}
	}
}
