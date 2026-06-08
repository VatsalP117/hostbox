package github

import (
	"errors"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
}

func TestCircuitBreaker_AllowsRequestsWhenClosed(t *testing.T) {
	cb := newCircuitBreaker(3, 100*time.Millisecond, testLogger())

	for i := 0; i < 5; i++ {
		err := cb.call(func() error { return nil })
		if err != nil {
			t.Fatalf("expected success on call %d, got %v", i+1, err)
		}
	}
}

func TestCircuitBreaker_OpensAfterMaxFailures(t *testing.T) {
	cb := newCircuitBreaker(3, 100*time.Millisecond, testLogger())

	// First two failures should stay closed
	cb.call(func() error { return errors.New("boom") })
	cb.call(func() error { return errors.New("boom") })

	if cb.state != cbClosed {
		t.Fatalf("expected state closed after 2 failures, got %s", cb.state)
	}

	// Third failure should trip the breaker
	cb.call(func() error { return errors.New("boom") })

	if cb.state != cbOpen {
		t.Fatalf("expected state open after 3 failures, got %s", cb.state)
	}

	// Subsequent requests should be rejected immediately
	err := cb.call(func() error { return nil })
	if !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("expected ErrCircuitOpen, got %v", err)
	}
}

func TestCircuitBreaker_ResetsAfterSuccess(t *testing.T) {
	cb := newCircuitBreaker(3, 100*time.Millisecond, testLogger())

	cb.call(func() error { return errors.New("boom") })
	cb.call(func() error { return errors.New("boom") })

	if cb.failures != 2 {
		t.Fatalf("expected 2 failures, got %d", cb.failures)
	}

	// Success should reset the failure count
	cb.call(func() error { return nil })

	if cb.failures != 0 {
		t.Fatalf("expected failures reset to 0, got %d", cb.failures)
	}

	// Two more failures should not trip because we need 3 consecutive
	cb.call(func() error { return errors.New("boom") })
	cb.call(func() error { return errors.New("boom") })

	if cb.state != cbClosed {
		t.Fatalf("expected state closed, got %s", cb.state)
	}
}

func TestCircuitBreaker_HalfOpenRecovers(t *testing.T) {
	cb := newCircuitBreaker(2, 50*time.Millisecond, testLogger())

	// Trip the breaker
	cb.call(func() error { return errors.New("boom") })
	cb.call(func() error { return errors.New("boom") })

	if cb.state != cbOpen {
		t.Fatalf("expected state open, got %s", cb.state)
	}

	// Wait for timeout
	time.Sleep(60 * time.Millisecond)

	// The next call should transition to half-open and succeed
	err := cb.call(func() error { return nil })
	if err != nil {
		t.Fatalf("expected success after recovery, got %v", err)
	}

	if cb.state != cbClosed {
		t.Fatalf("expected state closed after recovery, got %s", cb.state)
	}
}

func TestCircuitBreaker_HalfOpenReOpensOnFailure(t *testing.T) {
	cb := newCircuitBreaker(2, 50*time.Millisecond, testLogger())

	// Trip the breaker
	cb.call(func() error { return errors.New("boom") })
	cb.call(func() error { return errors.New("boom") })

	// Wait for timeout
	time.Sleep(60 * time.Millisecond)

	// The next call should be in half-open and fail again
	err := cb.call(func() error { return errors.New("still broken") })
	if err == nil || err.Error() != "still broken" {
		t.Fatalf("expected 'still broken' error, got %v", err)
	}

	if cb.state != cbOpen {
		t.Fatalf("expected state open after half-open failure, got %s", cb.state)
	}
}

func TestCircuitBreaker_ConcurrentSafety(t *testing.T) {
	cb := newCircuitBreaker(5, 50*time.Millisecond, testLogger())

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cb.call(func() error { return nil })
		}()
	}
	wg.Wait()

	if cb.failures != 0 {
		t.Fatalf("expected 0 failures after concurrent successes, got %d", cb.failures)
	}
}

func TestCircuitBreaker_HalfOpenOnlyAllowsOneProbe(t *testing.T) {
	cb := newCircuitBreaker(1, 50*time.Millisecond, testLogger())

	// Trip the breaker with a single failure (maxFailures=1)
	cb.call(func() error { return errors.New("boom") })

	if cb.state != cbOpen {
		t.Fatalf("expected state open, got %s", cb.state)
	}

	// Wait for timeout
	time.Sleep(60 * time.Millisecond)

	// Spin up two goroutines: one should get the probe, the other should get ErrCircuitOpen
	var wg sync.WaitGroup
	results := make(chan error, 2)

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := cb.call(func() error {
				// Simulate slow probe
				time.Sleep(20 * time.Millisecond)
				return nil
			})
			results <- err
		}()
	}

	wg.Wait()
	close(results)

	var gotOpen, gotSuccess int
	for err := range results {
		if errors.Is(err, ErrCircuitOpen) {
			gotOpen++
		} else if err == nil {
			gotSuccess++
		} else {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	if gotSuccess != 1 || gotOpen != 1 {
		t.Fatalf("expected exactly 1 success and 1 ErrCircuitOpen, got %d success and %d open", gotSuccess, gotOpen)
	}
}

func TestCircuitBreaker_NoLogger(t *testing.T) {
	cb := newCircuitBreaker(2, 50*time.Millisecond, nil)

	// Should not panic without a logger
	cb.call(func() error { return errors.New("boom") })
	cb.call(func() error { return errors.New("boom") })

	if cb.state != cbOpen {
		t.Fatalf("expected state open, got %s", cb.state)
	}
}

func TestCircuitBreaker_NilBreaker(t *testing.T) {
	// doWithBreaker should work when breaker is nil
	c := &Client{logger: testLogger()}

	err := c.doWithBreaker(func() error { return nil })
	if err != nil {
		t.Fatalf("expected nil breaker to pass through, got %v", err)
	}
}
