package github

import (
	"errors"
	"log/slog"
	"sync"
	"time"
)

var ErrCircuitOpen = errors.New("github circuit breaker is open")

type cbState int

const (
	cbClosed cbState = iota
	cbOpen
	cbHalfOpen
)

func (s cbState) String() string {
	switch s {
	case cbClosed:
		return "closed"
	case cbOpen:
		return "open"
	case cbHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// circuitBreaker protects against cascading failures when the GitHub API is
// unhealthy. It trips after maxFailures consecutive errors, stays open for the
// configured timeout, then transitions to half-open to test recovery.
type circuitBreaker struct {
	mu sync.Mutex

	state             cbState
	failures          int
	lastFailure       time.Time
	halfOpenInFlight  int
	halfOpenSuccesses int

	maxFailures int
	timeout     time.Duration
	halfOpenMax int

	logger *slog.Logger
}

func newCircuitBreaker(maxFailures int, timeout time.Duration, logger *slog.Logger) *circuitBreaker {
	return &circuitBreaker{
		maxFailures: maxFailures,
		timeout:     timeout,
		halfOpenMax: 1,
		logger:      logger,
	}
}

func (cb *circuitBreaker) logTransition(from, to cbState) {
	if cb.logger == nil {
		return
	}
	cb.logger.Warn("github circuit breaker state changed",
		"from", from,
		"to", to,
		"failures", cb.failures,
	)
}

// call executes fn if the breaker allows it. If the breaker is open, it
// returns ErrCircuitOpen immediately.
func (cb *circuitBreaker) call(fn func() error) error {
	cb.mu.Lock()

	switch cb.state {
	case cbOpen:
		if time.Since(cb.lastFailure) < cb.timeout {
			cb.mu.Unlock()
			return ErrCircuitOpen
		}
		// Timeout elapsed; allow a probe request in half-open state.
		prev := cb.state
		cb.state = cbHalfOpen
		cb.halfOpenInFlight = 0
		cb.halfOpenSuccesses = 0
		cb.logTransition(prev, cb.state)
		fallthrough
	case cbHalfOpen:
		if cb.halfOpenInFlight >= cb.halfOpenMax {
			cb.mu.Unlock()
			return ErrCircuitOpen
		}
		cb.halfOpenInFlight++
	}

	cb.mu.Unlock()

	err := fn()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == cbHalfOpen {
		cb.halfOpenInFlight--
		if err != nil {
			prev := cb.state
			cb.failures++
			cb.lastFailure = time.Now()
			cb.state = cbOpen
			cb.logTransition(prev, cb.state)
			return err
		}
		cb.halfOpenSuccesses++
		if cb.halfOpenSuccesses >= cb.halfOpenMax {
			prev := cb.state
			cb.state = cbClosed
			cb.failures = 0
			cb.logTransition(prev, cb.state)
		}
		return nil
	}

	if err != nil {
		cb.failures++
		cb.lastFailure = time.Now()
		if cb.failures >= cb.maxFailures {
			prev := cb.state
			cb.state = cbOpen
			cb.logTransition(prev, cb.state)
		}
		return err
	}

	cb.failures = 0
	return nil
}
