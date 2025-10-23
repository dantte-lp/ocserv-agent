package testutil

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

// NewTestLogger creates a logger for testing
func NewTestLogger(t *testing.T) zerolog.Logger {
	t.Helper()

	// Create test logger that writes to testing.T
	return zerolog.New(zerolog.NewTestWriter(t)).
		With().
		Timestamp().
		Logger()
}

// NewTestContext creates a context with timeout for testing
func NewTestContext(t *testing.T, timeout time.Duration) (context.Context, context.CancelFunc) {
	t.Helper()

	if timeout == 0 {
		timeout = 10 * time.Second
	}

	return context.WithTimeout(context.Background(), timeout)
}

// RequireNoError fails test if error is not nil
func RequireNoError(t *testing.T, err error, msg string) {
	t.Helper()

	if err != nil {
		t.Fatalf("%s: %v", msg, err)
	}
}

// RequireError fails test if error is nil
func RequireError(t *testing.T, err error, msg string) {
	t.Helper()

	if err == nil {
		t.Fatalf("%s: expected error but got nil", msg)
	}
}

// AssertEqual fails test if expected != actual
func AssertEqual(t *testing.T, expected, actual interface{}, msg string) {
	t.Helper()

	if expected != actual {
		t.Fatalf("%s: expected %v, got %v", msg, expected, actual)
	}
}

// AssertNotEmpty fails test if value is empty
func AssertNotEmpty(t *testing.T, value string, msg string) {
	t.Helper()

	if value == "" {
		t.Fatalf("%s: value is empty", msg)
	}
}

// AssertLenEqual fails test if slice length != expected
func AssertLenEqual(t *testing.T, expected int, slice interface{}, msg string) {
	t.Helper()

	var actual int
	switch v := slice.(type) {
	case []interface{}:
		actual = len(v)
	case []string:
		actual = len(v)
	default:
		t.Fatalf("AssertLenEqual: unsupported type %T", slice)
	}

	if actual != expected {
		t.Fatalf("%s: expected length %d, got %d", msg, expected, actual)
	}
}

// SkipIfShort skips test if -short flag is set
func SkipIfShort(t *testing.T, reason string) {
	t.Helper()

	if testing.Short() {
		t.Skipf("Skipping integration test in short mode: %s", reason)
	}
}

// Cleanup registers a cleanup function that logs on error
func Cleanup(t *testing.T, name string, fn func() error) {
	t.Helper()

	t.Cleanup(func() {
		if err := fn(); err != nil {
			t.Logf("Cleanup %s failed: %v", name, err)
		}
	})
}

// RetryWithTimeout retries a function until it succeeds or timeout
func RetryWithTimeout(t *testing.T, timeout time.Duration, interval time.Duration, fn func() error) error {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var lastErr error

	for {
		select {
		case <-ctx.Done():
			if lastErr != nil {
				return lastErr
			}
			return ctx.Err()

		case <-ticker.C:
			if err := fn(); err != nil {
				lastErr = err
				continue
			}
			return nil
		}
	}
}
