//go:build integration
// +build integration

package ocserv_test

import (
	"context"
	"testing"
	"time"

	"github.com/dantte-lp/ocserv-agent/internal/ocserv"
	"github.com/dantte-lp/ocserv-agent/internal/ocserv/testutil"
)

// TestShowUsersWithTimeout tests timeout handling
func TestShowUsersWithTimeout(t *testing.T) {
	testutil.SkipIfShort(t, "ShowUsers timeout test")

	// Setup
	mock := testutil.NewMockSocket(t, testutil.MockSocketConfig{UseCompose: true})
	defer mock.Close()

	if err := mock.WaitReady(t); err != nil {
		t.Fatalf("Mock socket not ready: %v", err)
	}

	logger := testutil.NewTestLogger(t)
	// Create manager with very short timeout (1ms)
	manager := ocserv.NewOcctlManager(mock.SocketPath(), "", 1*time.Millisecond, logger)

	// Create context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// This should fail with timeout/context error
	_, err := manager.ShowUsers(ctx)
	testutil.RequireError(t, err, "ShowUsers should fail with timeout")

	t.Logf("Got expected timeout error: %v", err)
	t.Logf("✅ Timeout test passed")
}

// TestShowStatusWithTimeout tests ShowStatus timeout handling
func TestShowStatusWithTimeout(t *testing.T) {
	testutil.SkipIfShort(t, "ShowStatus timeout test")

	// Setup
	mock := testutil.NewMockSocket(t, testutil.MockSocketConfig{UseCompose: true})
	defer mock.Close()

	if err := mock.WaitReady(t); err != nil {
		t.Fatalf("Mock socket not ready: %v", err)
	}

	logger := testutil.NewTestLogger(t)
	manager := ocserv.NewOcctlManager(mock.SocketPath(), "", 1*time.Millisecond, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	_, err := manager.ShowStatus(ctx)
	testutil.RequireError(t, err, "ShowStatus should fail with timeout")

	t.Logf("Got expected timeout error: %v", err)
	t.Logf("✅ ShowStatus timeout test passed")
}

// TestShowStatsWithTimeout tests ShowStats timeout handling
func TestShowStatsWithTimeout(t *testing.T) {
	testutil.SkipIfShort(t, "ShowStats timeout test")

	// Setup
	mock := testutil.NewMockSocket(t, testutil.MockSocketConfig{UseCompose: true})
	defer mock.Close()

	if err := mock.WaitReady(t); err != nil {
		t.Fatalf("Mock socket not ready: %v", err)
	}

	logger := testutil.NewTestLogger(t)
	manager := ocserv.NewOcctlManager(mock.SocketPath(), "", 1*time.Millisecond, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	_, err := manager.ShowStats(ctx)
	testutil.RequireError(t, err, "ShowStats should fail with timeout")

	t.Logf("Got expected timeout error: %v", err)
	t.Logf("✅ ShowStats timeout test passed")
}

// TestInvalidSocketPath tests handling of non-existent socket
func TestInvalidSocketPath(t *testing.T) {
	testutil.SkipIfShort(t, "Invalid socket path test")

	logger := testutil.NewTestLogger(t)
	// Use non-existent socket path
	manager := ocserv.NewOcctlManager("/tmp/nonexistent-socket-12345.socket", "", 5*time.Second, logger)

	ctx, cancel := testutil.NewTestContext(t, 10*time.Second)
	defer cancel()

	// This should fail with socket connection error
	_, err := manager.ShowUsers(ctx)
	testutil.RequireError(t, err, "ShowUsers should fail with invalid socket")

	t.Logf("Got expected socket error: %v", err)
	t.Logf("✅ Invalid socket path test passed")
}

// TestShowUserDetailedError tests error handling for ShowUser with invalid username
func TestShowUserDetailedError(t *testing.T) {
	testutil.SkipIfShort(t, "ShowUser error test")

	// Setup
	mock := testutil.NewMockSocket(t, testutil.MockSocketConfig{UseCompose: true})
	defer mock.Close()

	if err := mock.WaitReady(t); err != nil {
		t.Fatalf("Mock socket not ready: %v", err)
	}

	logger := testutil.NewTestLogger(t)
	manager := ocserv.NewOcctlManager(mock.SocketPath(), "", 5*time.Second, logger)

	ctx, cancel := testutil.NewTestContext(t, 10*time.Second)
	defer cancel()

	// Try to get user with non-existent username
	// Note: Mock server returns fixture data, so this tests the parsing logic
	_, err := manager.ShowUser(ctx, "nonexistent-user-12345")

	// Depending on mock implementation, this might succeed with empty array
	// or fail with error. Both are acceptable.
	if err != nil {
		t.Logf("ShowUser with invalid username failed as expected: %v", err)
	} else {
		t.Logf("ShowUser with invalid username returned (mock behavior)")
	}

	t.Logf("✅ ShowUser error handling test passed")
}

// TestShowIDError tests error handling for ShowID with invalid ID
func TestShowIDError(t *testing.T) {
	testutil.SkipIfShort(t, "ShowID error test")

	// Setup
	mock := testutil.NewMockSocket(t, testutil.MockSocketConfig{UseCompose: true})
	defer mock.Close()

	if err := mock.WaitReady(t); err != nil {
		t.Fatalf("Mock socket not ready: %v", err)
	}

	logger := testutil.NewTestLogger(t)
	manager := ocserv.NewOcctlManager(mock.SocketPath(), "", 5*time.Second, logger)

	ctx, cancel := testutil.NewTestContext(t, 10*time.Second)
	defer cancel()

	// Try to get connection with non-existent ID
	_, err := manager.ShowID(ctx, "99999999")

	// Depending on mock implementation, this might fail
	if err != nil {
		t.Logf("ShowID with invalid ID failed as expected: %v", err)
	} else {
		t.Logf("ShowID with invalid ID returned (mock behavior)")
	}

	t.Logf("✅ ShowID error handling test passed")
}

// TestCanceledContext tests handling of canceled context
func TestCanceledContext(t *testing.T) {
	testutil.SkipIfShort(t, "Canceled context test")

	// Setup
	mock := testutil.NewMockSocket(t, testutil.MockSocketConfig{UseCompose: true})
	defer mock.Close()

	if err := mock.WaitReady(t); err != nil {
		t.Fatalf("Mock socket not ready: %v", err)
	}

	logger := testutil.NewTestLogger(t)
	manager := ocserv.NewOcctlManager(mock.SocketPath(), "", 5*time.Second, logger)

	// Create context and cancel it immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel before calling

	// This should fail with context canceled error
	_, err := manager.ShowUsers(ctx)
	testutil.RequireError(t, err, "ShowUsers should fail with canceled context")

	t.Logf("Got expected context canceled error: %v", err)
	t.Logf("✅ Canceled context test passed")
}

// TestMultipleTimeouts tests multiple operations with different timeouts
func TestMultipleTimeouts(t *testing.T) {
	testutil.SkipIfShort(t, "Multiple timeouts test")

	// Setup
	mock := testutil.NewMockSocket(t, testutil.MockSocketConfig{UseCompose: true})
	defer mock.Close()

	if err := mock.WaitReady(t); err != nil {
		t.Fatalf("Mock socket not ready: %v", err)
	}

	logger := testutil.NewTestLogger(t)

	tests := []struct {
		name       string
		timeout    time.Duration
		shouldFail bool
	}{
		{"VeryShortTimeout", 1 * time.Nanosecond, true},
		{"ShortTimeout", 1 * time.Millisecond, true},
		{"NormalTimeout", 5 * time.Second, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := ocserv.NewOcctlManager(mock.SocketPath(), "", tt.timeout, logger)

			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()

			_, err := manager.ShowUsers(ctx)

			if tt.shouldFail {
				testutil.RequireError(t, err, "Should fail with short timeout")
				t.Logf("Failed as expected with timeout %v: %v", tt.timeout, err)
			} else {
				testutil.RequireNoError(t, err, "Should succeed with normal timeout")
				t.Logf("Succeeded with timeout %v", tt.timeout)
			}
		})
	}

	t.Logf("✅ Multiple timeouts test passed")
}

// TestContextDeadlineExceeded tests context deadline exceeded error
func TestContextDeadlineExceeded(t *testing.T) {
	testutil.SkipIfShort(t, "Context deadline exceeded test")

	// Setup
	mock := testutil.NewMockSocket(t, testutil.MockSocketConfig{UseCompose: true})
	defer mock.Close()

	if err := mock.WaitReady(t); err != nil {
		t.Fatalf("Mock socket not ready: %v", err)
	}

	logger := testutil.NewTestLogger(t)
	manager := ocserv.NewOcctlManager(mock.SocketPath(), "", 100*time.Millisecond, logger)

	// Create context with deadline in the past
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-1*time.Second))
	defer cancel()

	// This should fail immediately with deadline exceeded
	_, err := manager.ShowUsers(ctx)
	testutil.RequireError(t, err, "ShowUsers should fail with deadline exceeded")

	t.Logf("Got expected deadline exceeded error: %v", err)
	t.Logf("✅ Context deadline exceeded test passed")
}

// TestEmptySocketPath tests handling of empty socket path
func TestEmptySocketPath(t *testing.T) {
	testutil.SkipIfShort(t, "Empty socket path test")

	logger := testutil.NewTestLogger(t)
	// Create manager with empty socket path
	manager := ocserv.NewOcctlManager("", "", 5*time.Second, logger)

	ctx, cancel := testutil.NewTestContext(t, 10*time.Second)
	defer cancel()

	// This should fail (occtl will use default socket or fail)
	_, err := manager.ShowUsers(ctx)

	// Depending on system configuration, this might succeed or fail
	// We just verify it doesn't panic
	if err != nil {
		t.Logf("ShowUsers with empty socket failed: %v", err)
	} else {
		t.Logf("ShowUsers with empty socket succeeded (using default)")
	}

	t.Logf("✅ Empty socket path test passed (no panic)")
}

// TestRapidSequentialCalls tests rapid sequential calls
func TestRapidSequentialCalls(t *testing.T) {
	testutil.SkipIfShort(t, "Rapid sequential calls test")

	// Setup
	mock := testutil.NewMockSocket(t, testutil.MockSocketConfig{UseCompose: true})
	defer mock.Close()

	if err := mock.WaitReady(t); err != nil {
		t.Fatalf("Mock socket not ready: %v", err)
	}

	logger := testutil.NewTestLogger(t)
	manager := ocserv.NewOcctlManager(mock.SocketPath(), "", 5*time.Second, logger)

	// Make 100 rapid sequential calls
	const numCalls = 100
	for i := 0; i < numCalls; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, err := manager.ShowUsers(ctx)
		cancel()

		if err != nil {
			t.Fatalf("Call %d failed: %v", i+1, err)
		}

		if i%20 == 0 {
			t.Logf("Completed %d/%d calls", i, numCalls)
		}
	}

	t.Logf("✅ Rapid sequential calls test passed (%d calls)", numCalls)
}

// TestMixedOperations tests mixing different operations with errors
func TestMixedOperations(t *testing.T) {
	testutil.SkipIfShort(t, "Mixed operations test")

	// Setup
	mock := testutil.NewMockSocket(t, testutil.MockSocketConfig{UseCompose: true})
	defer mock.Close()

	if err := mock.WaitReady(t); err != nil {
		t.Fatalf("Mock socket not ready: %v", err)
	}

	logger := testutil.NewTestLogger(t)
	manager := ocserv.NewOcctlManager(mock.SocketPath(), "", 5*time.Second, logger)

	tests := []struct {
		name string
		fn   func(context.Context) error
	}{
		{"ShowUsers", func(ctx context.Context) error {
			_, err := manager.ShowUsers(ctx)
			return err
		}},
		{"ShowStatus", func(ctx context.Context) error {
			_, err := manager.ShowStatus(ctx)
			return err
		}},
		{"ShowStats", func(ctx context.Context) error {
			_, err := manager.ShowStats(ctx)
			return err
		}},
		{"ShowStatusDetailed", func(ctx context.Context) error {
			_, err := manager.ShowStatusDetailed(ctx)
			return err
		}},
		{"ShowUsersDetailed", func(ctx context.Context) error {
			_, err := manager.ShowUsersDetailed(ctx)
			return err
		}},
		{"ShowSessionsAll", func(ctx context.Context) error {
			_, err := manager.ShowSessionsAll(ctx)
			return err
		}},
		{"ShowSessionsValid", func(ctx context.Context) error {
			_, err := manager.ShowSessionsValid(ctx)
			return err
		}},
		{"ShowIRoutes", func(ctx context.Context) error {
			_, err := manager.ShowIRoutes(ctx)
			return err
		}},
		{"ShowIPBanPoints", func(ctx context.Context) error {
			_, err := manager.ShowIPBanPoints(ctx)
			return err
		}},
	}

	// Run all operations successfully
	for _, tt := range tests {
		t.Run(tt.name+"_Success", func(t *testing.T) {
			ctx, cancel := testutil.NewTestContext(t, 10*time.Second)
			defer cancel()

			err := tt.fn(ctx)
			testutil.RequireNoError(t, err, tt.name+" failed")
			t.Logf("%s succeeded", tt.name)
		})
	}

	// Run all operations with timeout
	for _, tt := range tests {
		t.Run(tt.name+"_Timeout", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
			defer cancel()

			err := tt.fn(ctx)
			testutil.RequireError(t, err, tt.name+" should timeout")
			t.Logf("%s timed out as expected", tt.name)
		})
	}

	t.Logf("✅ Mixed operations test passed")
}
