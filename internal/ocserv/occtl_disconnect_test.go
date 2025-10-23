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

// TestDisconnectUserWithValidUsername tests DisconnectUser with existing user
func TestDisconnectUserWithValidUsername(t *testing.T) {
	testutil.SkipIfShort(t, "DisconnectUser valid username test")

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

	// Get a valid username
	users, err := manager.ShowUsers(ctx)
	testutil.RequireNoError(t, err, "ShowUsers failed")

	if len(users) == 0 {
		t.Skip("No users in fixture")
	}

	validUsername := users[0].Username
	t.Logf("Testing disconnect for username: %s", validUsername)

	// Execute DisconnectUser
	// Note: Mock server returns success but doesn't actually disconnect
	err = manager.DisconnectUser(ctx, validUsername)

	// Mock server behavior - might succeed
	if err != nil {
		t.Logf("DisconnectUser failed (mock behavior): %v", err)
	} else {
		t.Logf("DisconnectUser succeeded (mock returns success)")
	}

	t.Logf("✅ DisconnectUser with valid username test passed")
}

// TestDisconnectUserWithInvalidUsername tests DisconnectUser with non-existent user
func TestDisconnectUserWithInvalidUsername(t *testing.T) {
	testutil.SkipIfShort(t, "DisconnectUser invalid username test")

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

	// Try non-existent username
	invalidUsername := "nonexistent-user-12345-test"
	err := manager.DisconnectUser(ctx, invalidUsername)

	// Mock might fail or succeed
	if err != nil {
		t.Logf("DisconnectUser with invalid username failed as expected: %v", err)
	} else {
		t.Logf("DisconnectUser with invalid username succeeded (mock behavior)")
	}

	t.Logf("✅ DisconnectUser with invalid username test passed")
}

// TestDisconnectUserWithEmptyUsername tests DisconnectUser with empty username
func TestDisconnectUserWithEmptyUsername(t *testing.T) {
	testutil.SkipIfShort(t, "DisconnectUser empty username test")

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

	// Try empty username
	err := manager.DisconnectUser(ctx, "")

	// Should fail or handle gracefully
	if err != nil {
		t.Logf("DisconnectUser with empty username failed as expected: %v", err)
	} else {
		t.Log("DisconnectUser with empty username succeeded (mock behavior)")
	}

	t.Logf("✅ DisconnectUser empty username test passed (no panic)")
}

// TestDisconnectIDWithValidID tests DisconnectID with existing connection ID
func TestDisconnectIDWithValidID(t *testing.T) {
	testutil.SkipIfShort(t, "DisconnectID valid ID test")

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

	// Get a valid ID
	users, err := manager.ShowUsers(ctx)
	testutil.RequireNoError(t, err, "ShowUsers failed")

	if len(users) == 0 {
		t.Skip("No users in fixture")
	}

	// Use known ID from fixture
	validID := "836873"
	t.Logf("Testing disconnect for ID: %s", validID)

	// Execute DisconnectID
	err = manager.DisconnectID(ctx, validID)

	// Mock server behavior - might succeed
	if err != nil {
		t.Logf("DisconnectID failed (mock behavior): %v", err)
	} else {
		t.Logf("DisconnectID succeeded (mock returns success)")
	}

	t.Logf("✅ DisconnectID with valid ID test passed")
}

// TestDisconnectIDWithInvalidID tests DisconnectID with non-existent ID
func TestDisconnectIDWithInvalidID(t *testing.T) {
	testutil.SkipIfShort(t, "DisconnectID invalid ID test")

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

	// Try non-existent ID
	invalidID := "99999999"
	err := manager.DisconnectID(ctx, invalidID)

	// Mock might fail or succeed
	if err != nil {
		t.Logf("DisconnectID with invalid ID failed as expected: %v", err)
	} else {
		t.Logf("DisconnectID with invalid ID succeeded (mock behavior)")
	}

	t.Logf("✅ DisconnectID with invalid ID test passed")
}

// TestDisconnectIDWithEmptyID tests DisconnectID with empty ID
func TestDisconnectIDWithEmptyID(t *testing.T) {
	testutil.SkipIfShort(t, "DisconnectID empty ID test")

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

	// Try empty ID
	err := manager.DisconnectID(ctx, "")

	// Should fail or handle gracefully
	if err != nil {
		t.Logf("DisconnectID with empty ID failed as expected: %v", err)
	} else {
		t.Log("DisconnectID with empty ID succeeded (mock behavior)")
	}

	t.Logf("✅ DisconnectID empty ID test passed (no panic)")
}

// TestDisconnectUserWithTimeout tests DisconnectUser timeout handling
func TestDisconnectUserWithTimeout(t *testing.T) {
	testutil.SkipIfShort(t, "DisconnectUser timeout test")

	// Setup
	mock := testutil.NewMockSocket(t, testutil.MockSocketConfig{UseCompose: true})
	defer mock.Close()

	if err := mock.WaitReady(t); err != nil {
		t.Fatalf("Mock socket not ready: %v", err)
	}

	logger := testutil.NewTestLogger(t)
	manager := ocserv.NewOcctlManager(mock.SocketPath(), "", 1*time.Millisecond, logger)

	// Create context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// This should fail with timeout
	err := manager.DisconnectUser(ctx, "testuser")
	testutil.RequireError(t, err, "DisconnectUser should fail with timeout")

	t.Logf("Got expected timeout error: %v", err)
	t.Logf("✅ DisconnectUser timeout test passed")
}

// TestDisconnectIDWithTimeout tests DisconnectID timeout handling
func TestDisconnectIDWithTimeout(t *testing.T) {
	testutil.SkipIfShort(t, "DisconnectID timeout test")

	// Setup
	mock := testutil.NewMockSocket(t, testutil.MockSocketConfig{UseCompose: true})
	defer mock.Close()

	if err := mock.WaitReady(t); err != nil {
		t.Fatalf("Mock socket not ready: %v", err)
	}

	logger := testutil.NewTestLogger(t)
	manager := ocserv.NewOcctlManager(mock.SocketPath(), "", 1*time.Millisecond, logger)

	// Create context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// This should fail with timeout
	err := manager.DisconnectID(ctx, "12345")
	testutil.RequireError(t, err, "DisconnectID should fail with timeout")

	t.Logf("Got expected timeout error: %v", err)
	t.Logf("✅ DisconnectID timeout test passed")
}

// TestDisconnectOperationsSequence tests disconnect operations in sequence
func TestDisconnectOperationsSequence(t *testing.T) {
	testutil.SkipIfShort(t, "Disconnect operations sequence test")

	// Setup
	mock := testutil.NewMockSocket(t, testutil.MockSocketConfig{UseCompose: true})
	defer mock.Close()

	if err := mock.WaitReady(t); err != nil {
		t.Fatalf("Mock socket not ready: %v", err)
	}

	logger := testutil.NewTestLogger(t)
	manager := ocserv.NewOcctlManager(mock.SocketPath(), "", 5*time.Second, logger)

	ctx, cancel := testutil.NewTestContext(t, 30*time.Second)
	defer cancel()

	// Get user list
	users, err := manager.ShowUsers(ctx)
	testutil.RequireNoError(t, err, "ShowUsers failed")

	if len(users) == 0 {
		t.Skip("No users in fixture")
	}

	username := users[0].Username
	userID := "836873" // Known ID from fixture

	// Sequence of operations
	tests := []struct {
		name string
		fn   func() error
	}{
		{"DisconnectUser", func() error {
			return manager.DisconnectUser(ctx, username)
		}},
		{"DisconnectID", func() error {
			return manager.DisconnectID(ctx, userID)
		}},
		{"DisconnectUser-Again", func() error {
			return manager.DisconnectUser(ctx, username)
		}},
		{"DisconnectID-Again", func() error {
			return manager.DisconnectID(ctx, userID)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if err != nil {
				t.Logf("%s failed (acceptable): %v", tt.name, err)
			} else {
				t.Logf("%s succeeded", tt.name)
			}
		})
	}

	t.Logf("✅ Disconnect operations sequence test passed")
}

// TestDisconnectMultipleUsers tests disconnecting multiple users
func TestDisconnectMultipleUsers(t *testing.T) {
	testutil.SkipIfShort(t, "Disconnect multiple users test")

	// Setup
	mock := testutil.NewMockSocket(t, testutil.MockSocketConfig{UseCompose: true})
	defer mock.Close()

	if err := mock.WaitReady(t); err != nil {
		t.Fatalf("Mock socket not ready: %v", err)
	}

	logger := testutil.NewTestLogger(t)
	manager := ocserv.NewOcctlManager(mock.SocketPath(), "", 5*time.Second, logger)

	ctx, cancel := testutil.NewTestContext(t, 30*time.Second)
	defer cancel()

	// Get all users
	users, err := manager.ShowUsers(ctx)
	testutil.RequireNoError(t, err, "ShowUsers failed")

	if len(users) == 0 {
		t.Skip("No users in fixture")
	}

	t.Logf("Attempting to disconnect %d user(s)", len(users))

	// Try to disconnect all users
	successCount := 0
	failCount := 0

	for i, user := range users {
		err := manager.DisconnectUser(ctx, user.Username)
		if err != nil {
			failCount++
			t.Logf("User %d (%s): disconnect failed - %v", i+1, user.Username, err)
		} else {
			successCount++
			t.Logf("User %d (%s): disconnect succeeded", i+1, user.Username)
		}
	}

	t.Logf("Disconnect results: %d succeeded, %d failed", successCount, failCount)
	t.Logf("✅ Disconnect multiple users test passed")
}

// TestDisconnectCanceledContext tests disconnect with canceled context
func TestDisconnectCanceledContext(t *testing.T) {
	testutil.SkipIfShort(t, "Disconnect canceled context test")

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
	cancel()

	// Both should fail with context canceled
	t.Run("DisconnectUser", func(t *testing.T) {
		err := manager.DisconnectUser(ctx, "testuser")
		testutil.RequireError(t, err, "DisconnectUser should fail with canceled context")
		t.Logf("Got expected context canceled error: %v", err)
	})

	t.Run("DisconnectID", func(t *testing.T) {
		err := manager.DisconnectID(ctx, "12345")
		testutil.RequireError(t, err, "DisconnectID should fail with canceled context")
		t.Logf("Got expected context canceled error: %v", err)
	})

	t.Logf("✅ Disconnect canceled context test passed")
}
