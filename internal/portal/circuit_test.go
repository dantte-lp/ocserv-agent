package portal

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"
)

func TestCircuitBreaker_Execute_Success(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := DefaultCircuitBreakerConfig()
	cfg.FailureThreshold = 3

	cb := NewCircuitBreaker(cfg, logger)

	// Execute successful requests
	for i := 0; i < 5; i++ {
		err := cb.Execute(context.Background(), func(ctx context.Context) error {
			return nil
		})
		if err != nil {
			t.Errorf("Execute() failed: %v", err)
		}
	}

	if cb.State() != StateClosed {
		t.Errorf("state = %v, want StateClosed", cb.State())
	}

	counts := cb.Counts()
	if counts.TotalSuccesses != 5 {
		t.Errorf("total successes = %d, want 5", counts.TotalSuccesses)
	}
}

func TestCircuitBreaker_Execute_OpenOnFailures(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := DefaultCircuitBreakerConfig()
	cfg.FailureThreshold = 3
	cfg.Timeout = 100 * time.Millisecond

	cb := NewCircuitBreaker(cfg, logger)

	testErr := errors.New("test error")

	// Execute failing requests until circuit opens
	for i := 0; i < 3; i++ {
		err := cb.Execute(context.Background(), func(ctx context.Context) error {
			return testErr
		})
		if err == nil {
			t.Error("Execute() should have returned error")
		}
	}

	// Circuit should be open now
	if cb.State() != StateOpen {
		t.Errorf("state = %v, want StateOpen", cb.State())
	}

	// Next request should be rejected
	err := cb.Execute(context.Background(), func(ctx context.Context) error {
		return nil
	})
	if !errors.Is(err, ErrCircuitOpen) {
		t.Errorf("Execute() error = %v, want ErrCircuitOpen", err)
	}
}

func TestCircuitBreaker_HalfOpen_Recovery(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := DefaultCircuitBreakerConfig()
	cfg.FailureThreshold = 2
	cfg.Timeout = 50 * time.Millisecond
	cfg.MaxRequests = 2

	cb := NewCircuitBreaker(cfg, logger)

	testErr := errors.New("test error")

	// Open circuit with failures
	for i := 0; i < 2; i++ {
		_ = cb.Execute(context.Background(), func(ctx context.Context) error {
			return testErr
		})
	}

	if cb.State() != StateOpen {
		t.Errorf("state = %v, want StateOpen", cb.State())
	}

	// Wait for timeout
	time.Sleep(60 * time.Millisecond)

	// Circuit should be half-open now
	// First successful request
	err := cb.Execute(context.Background(), func(ctx context.Context) error {
		return nil
	})
	if err != nil {
		t.Errorf("Execute() error = %v, want nil", err)
	}

	if cb.State() != StateHalfOpen {
		t.Errorf("state = %v, want StateHalfOpen", cb.State())
	}

	// Second successful request should close circuit
	err = cb.Execute(context.Background(), func(ctx context.Context) error {
		return nil
	})
	if err != nil {
		t.Errorf("Execute() error = %v, want nil", err)
	}

	if cb.State() != StateClosed {
		t.Errorf("state = %v, want StateClosed", cb.State())
	}
}

func TestCircuitBreaker_HalfOpen_FailureReopens(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := DefaultCircuitBreakerConfig()
	cfg.FailureThreshold = 2
	cfg.Timeout = 50 * time.Millisecond
	cfg.MaxRequests = 1

	cb := NewCircuitBreaker(cfg, logger)

	testErr := errors.New("test error")

	// Open circuit
	for i := 0; i < 2; i++ {
		_ = cb.Execute(context.Background(), func(ctx context.Context) error {
			return testErr
		})
	}

	// Wait for timeout
	time.Sleep(60 * time.Millisecond)

	// Fail in half-open state
	err := cb.Execute(context.Background(), func(ctx context.Context) error {
		return testErr
	})
	if err == nil {
		t.Error("Execute() should return error")
	}

	// Circuit should be open again
	if cb.State() != StateOpen {
		t.Errorf("state = %v, want StateOpen", cb.State())
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := DefaultCircuitBreakerConfig()
	cfg.FailureThreshold = 2

	cb := NewCircuitBreaker(cfg, logger)

	testErr := errors.New("test error")

	// Open circuit
	for i := 0; i < 2; i++ {
		_ = cb.Execute(context.Background(), func(ctx context.Context) error {
			return testErr
		})
	}

	if cb.State() != StateOpen {
		t.Errorf("state = %v, want StateOpen", cb.State())
	}

	// Reset
	cb.Reset()

	if cb.State() != StateClosed {
		t.Errorf("state after reset = %v, want StateClosed", cb.State())
	}

	counts := cb.Counts()
	if counts.TotalFailures != 0 {
		t.Errorf("total failures after reset = %d, want 0", counts.TotalFailures)
	}
}

func TestCircuitBreaker_OnStateChange(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	var stateChanges []struct {
		from CircuitState
		to   CircuitState
	}

	cfg := DefaultCircuitBreakerConfig()
	cfg.FailureThreshold = 2
	cfg.OnStateChange = func(from, to CircuitState) {
		stateChanges = append(stateChanges, struct {
			from CircuitState
			to   CircuitState
		}{from, to})
	}

	cb := NewCircuitBreaker(cfg, logger)

	testErr := errors.New("test error")

	// Open circuit
	for i := 0; i < 2; i++ {
		_ = cb.Execute(context.Background(), func(ctx context.Context) error {
			return testErr
		})
	}

	if len(stateChanges) != 1 {
		t.Errorf("state changes = %d, want 1", len(stateChanges))
	}

	if stateChanges[0].from != StateClosed || stateChanges[0].to != StateOpen {
		t.Errorf("state change = %v -> %v, want StateClosed -> StateOpen",
			stateChanges[0].from, stateChanges[0].to)
	}
}

func TestCircuitBreaker_Stats(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := DefaultCircuitBreakerConfig()
	cfg.FailureThreshold = 5

	cb := NewCircuitBreaker(cfg, logger)

	// Execute mixed requests
	for i := 0; i < 3; i++ {
		_ = cb.Execute(context.Background(), func(ctx context.Context) error {
			return nil
		})
	}
	for i := 0; i < 2; i++ {
		_ = cb.Execute(context.Background(), func(ctx context.Context) error {
			return errors.New("error")
		})
	}

	stats := cb.Stats()
	if stats == "" {
		t.Error("Stats() returned empty string")
	}

	counts := cb.Counts()
	if counts.TotalSuccesses != 3 {
		t.Errorf("total successes = %d, want 3", counts.TotalSuccesses)
	}
	if counts.TotalFailures != 2 {
		t.Errorf("total failures = %d, want 2", counts.TotalFailures)
	}
	if counts.Requests != 5 {
		t.Errorf("total requests = %d, want 5", counts.Requests)
	}
}
