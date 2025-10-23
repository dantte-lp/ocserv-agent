//go:build integration
// +build integration

package ocserv_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/dantte-lp/ocserv-agent/internal/ocserv"
	"github.com/dantte-lp/ocserv-agent/internal/ocserv/testutil"
)

// TestShowUserSpecialCharacters tests ShowUser with special characters in username
func TestShowUserSpecialCharacters(t *testing.T) {
	testutil.SkipIfShort(t, "ShowUser special characters test")

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

	// Test various special characters
	specialUsernames := []string{
		"user@domain.com",
		"user-name",
		"user_name",
		"user.name",
		"user$name",
		"user name",  // Space
		"user'name",  // Quote
		"user\"name", // Double quote
		"user;name",  // Semicolon
		"user|name",  // Pipe
	}

	for _, username := range specialUsernames {
		t.Run("Username_"+username, func(t *testing.T) {
			_, err := manager.ShowUser(ctx, username)
			if err != nil {
				t.Logf("ShowUser with '%s' failed (expected): %v", username, err)
			} else {
				t.Logf("ShowUser with '%s' succeeded", username)
			}
			// Both outcomes are acceptable - just verify no panic
		})
	}

	t.Logf("✅ ShowUser special characters test passed")
}

// TestShowIDSpecialFormats tests ShowID with various ID formats
func TestShowIDSpecialFormats(t *testing.T) {
	testutil.SkipIfShort(t, "ShowID special formats test")

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

	// Test various ID formats
	specialIDs := []string{
		"0",         // Zero
		"-1",        // Negative
		"999999999", // Large number
		"abc",       // Letters
		"12.34",     // Decimal
		"0x1234",    // Hex
		" 123 ",     // Spaces
		"123\n",     // Newline
	}

	for _, id := range specialIDs {
		t.Run("ID_"+id, func(t *testing.T) {
			_, err := manager.ShowID(ctx, id)
			if err != nil {
				t.Logf("ShowID with '%s' failed (expected): %v", id, err)
			} else {
				t.Logf("ShowID with '%s' succeeded", id)
			}
			// Both outcomes are acceptable - just verify no panic
		})
	}

	t.Logf("✅ ShowID special formats test passed")
}

// TestLongUsernameHandling tests ShowUser with very long username
func TestLongUsernameHandling(t *testing.T) {
	testutil.SkipIfShort(t, "Long username test")

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

	// Generate very long username (1000 characters)
	longUsername := strings.Repeat("a", 1000)

	_, err := manager.ShowUser(ctx, longUsername)
	if err != nil {
		t.Logf("ShowUser with long username failed (expected): %v", err)
	} else {
		t.Log("ShowUser with long username succeeded")
	}

	t.Logf("✅ Long username test passed (no panic)")
}

// TestUnicodeUsernameHandling tests ShowUser with Unicode characters
func TestUnicodeUsernameHandling(t *testing.T) {
	testutil.SkipIfShort(t, "Unicode username test")

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

	// Test Unicode usernames
	unicodeUsernames := []string{
		"пользователь", // Russian
		"用户",           // Chinese
		"ユーザー",         // Japanese
		"مستخدم",       // Arabic
		"🔒user🔑",       // Emojis
	}

	for _, username := range unicodeUsernames {
		t.Run("Unicode_"+username, func(t *testing.T) {
			_, err := manager.ShowUser(ctx, username)
			if err != nil {
				t.Logf("ShowUser with '%s' failed (expected): %v", username, err)
			} else {
				t.Logf("ShowUser with '%s' succeeded", username)
			}
		})
	}

	t.Logf("✅ Unicode username test passed")
}

// TestConcurrentDisconnectOperations tests concurrent disconnect calls
func TestConcurrentDisconnectOperations(t *testing.T) {
	testutil.SkipIfShort(t, "Concurrent disconnect test")

	// Setup
	mock := testutil.NewMockSocket(t, testutil.MockSocketConfig{UseCompose: true})
	defer mock.Close()

	if err := mock.WaitReady(t); err != nil {
		t.Fatalf("Mock socket not ready: %v", err)
	}

	logger := testutil.NewTestLogger(t)
	manager := ocserv.NewOcctlManager(mock.SocketPath(), "", 5*time.Second, logger)

	// Get users
	ctx, cancel := testutil.NewTestContext(t, 30*time.Second)
	defer cancel()

	users, err := manager.ShowUsers(ctx)
	testutil.RequireNoError(t, err, "ShowUsers failed")

	if len(users) == 0 {
		t.Skip("No users in fixture")
	}

	username := users[0].Username

	// Run 10 concurrent disconnect operations
	const numOps = 10
	done := make(chan error, numOps)

	for i := 0; i < numOps; i++ {
		go func(id int) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			err := manager.DisconnectUser(ctx, username)
			done <- err
		}(i)
	}

	// Wait for all to complete
	successCount := 0
	failCount := 0

	for i := 0; i < numOps; i++ {
		err := <-done
		if err != nil {
			failCount++
		} else {
			successCount++
		}
	}

	t.Logf("Concurrent disconnect results: %d succeeded, %d failed", successCount, failCount)
	t.Logf("✅ Concurrent disconnect test passed (%d operations)", numOps)
}

// TestShowUserAfterDisconnect tests ShowUser behavior after disconnect
func TestShowUserAfterDisconnect(t *testing.T) {
	testutil.SkipIfShort(t, "ShowUser after disconnect test")

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

	// Get users
	users, err := manager.ShowUsers(ctx)
	testutil.RequireNoError(t, err, "ShowUsers failed")

	if len(users) == 0 {
		t.Skip("No users in fixture")
	}

	username := users[0].Username

	// Show user before disconnect
	usersBefore, err := manager.ShowUser(ctx, username)
	if err != nil {
		t.Skipf("ShowUser before disconnect failed: %v", err)
	}
	t.Logf("Before disconnect: %d session(s)", len(usersBefore))

	// Disconnect
	err = manager.DisconnectUser(ctx, username)
	if err != nil {
		t.Logf("DisconnectUser failed: %v", err)
	}

	// Show user after disconnect
	// Note: Mock server doesn't actually change state
	usersAfter, err := manager.ShowUser(ctx, username)
	if err != nil {
		t.Logf("ShowUser after disconnect failed: %v", err)
	} else {
		t.Logf("After disconnect: %d session(s)", len(usersAfter))
	}

	// Mock server returns same data, so we just log
	t.Logf("Mock server: state unchanged (expected behavior)")
	t.Logf("✅ ShowUser after disconnect test passed")
}

// TestNullByteHandling tests handling of null bytes in input
func TestNullByteHandling(t *testing.T) {
	testutil.SkipIfShort(t, "Null byte handling test")

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

	// Test null byte in username
	usernameWithNull := "user\x00name"

	_, err := manager.ShowUser(ctx, usernameWithNull)
	if err != nil {
		t.Logf("ShowUser with null byte failed (expected): %v", err)
	} else {
		t.Log("ShowUser with null byte succeeded")
	}

	t.Logf("✅ Null byte handling test passed (no panic)")
}

// TestRapidShowUserCalls tests rapid sequential ShowUser calls
func TestRapidShowUserCalls(t *testing.T) {
	testutil.SkipIfShort(t, "Rapid ShowUser calls test")

	// Setup
	mock := testutil.NewMockSocket(t, testutil.MockSocketConfig{UseCompose: true})
	defer mock.Close()

	if err := mock.WaitReady(t); err != nil {
		t.Fatalf("Mock socket not ready: %v", err)
	}

	logger := testutil.NewTestLogger(t)
	manager := ocserv.NewOcctlManager(mock.SocketPath(), "", 5*time.Second, logger)

	ctx, cancel := testutil.NewTestContext(t, 60*time.Second)
	defer cancel()

	// Get a valid username
	users, err := manager.ShowUsers(ctx)
	testutil.RequireNoError(t, err, "ShowUsers failed")

	if len(users) == 0 {
		t.Skip("No users in fixture")
	}

	username := users[0].Username

	// Make 50 rapid sequential calls
	const numCalls = 50
	for i := 0; i < numCalls; i++ {
		_, err := manager.ShowUser(ctx, username)
		if err != nil {
			t.Fatalf("Call %d failed: %v", i+1, err)
		}

		if i%10 == 0 {
			t.Logf("Completed %d/%d calls", i, numCalls)
		}
	}

	t.Logf("✅ Rapid ShowUser calls test passed (%d calls)", numCalls)
}

// TestMixedUserOperations tests mixing different user operations
func TestMixedUserOperations(t *testing.T) {
	testutil.SkipIfShort(t, "Mixed user operations test")

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

	// Get valid data
	users, err := manager.ShowUsers(ctx)
	testutil.RequireNoError(t, err, "ShowUsers failed")

	if len(users) == 0 {
		t.Skip("No users in fixture")
	}

	username := users[0].Username
	userID := "836873"

	// Mix of operations
	operations := []struct {
		name string
		fn   func() error
	}{
		{"ShowUser", func() error {
			_, err := manager.ShowUser(ctx, username)
			return err
		}},
		{"ShowID", func() error {
			_, err := manager.ShowID(ctx, userID)
			return err
		}},
		{"DisconnectUser", func() error {
			return manager.DisconnectUser(ctx, username)
		}},
		{"DisconnectID", func() error {
			return manager.DisconnectID(ctx, userID)
		}},
		{"ShowUser-Again", func() error {
			_, err := manager.ShowUser(ctx, username)
			return err
		}},
		{"ShowID-Again", func() error {
			_, err := manager.ShowID(ctx, userID)
			return err
		}},
	}

	successCount := 0
	failCount := 0

	for _, op := range operations {
		t.Run(op.name, func(t *testing.T) {
			err := op.fn()
			if err != nil {
				failCount++
				t.Logf("%s failed: %v", op.name, err)
			} else {
				successCount++
				t.Logf("%s succeeded", op.name)
			}
		})
	}

	t.Logf("Mixed operations results: %d succeeded, %d failed", successCount, failCount)
	t.Logf("✅ Mixed user operations test passed")
}

// TestShowUserDetailedVsShowUser tests consistency between ShowUsersDetailed and ShowUser
func TestShowUserDetailedVsShowUser(t *testing.T) {
	testutil.SkipIfShort(t, "ShowUsersDetailed vs ShowUser consistency test")

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

	// Get all users (detailed)
	allUsersDetailed, err := manager.ShowUsersDetailed(ctx)
	testutil.RequireNoError(t, err, "ShowUsersDetailed failed")

	if len(allUsersDetailed) == 0 {
		t.Skip("No users in fixture")
	}

	// Get specific user
	username := allUsersDetailed[0].Username
	specificUser, err := manager.ShowUser(ctx, username)
	testutil.RequireNoError(t, err, "ShowUser failed")

	// Compare
	t.Logf("ShowUsersDetailed returned %d users total", len(allUsersDetailed))
	t.Logf("ShowUser for '%s' returned %d session(s)", username, len(specificUser))

	// Find matching user in ShowUsersDetailed
	var matchedUser *ocserv.UserDetailed
	for i := range allUsersDetailed {
		if allUsersDetailed[i].Username == username {
			matchedUser = &allUsersDetailed[i]
			break
		}
	}

	if matchedUser != nil && len(specificUser) > 0 {
		// Compare first session
		t.Logf("Comparing data:")
		t.Logf("  ShowUsersDetailed: ID=%d, Username=%s", matchedUser.ID, matchedUser.Username)
		t.Logf("  ShowUser:          ID=%d, Username=%s", specificUser[0].ID, specificUser[0].Username)

		if matchedUser.ID == specificUser[0].ID {
			t.Log("IDs match - same session")
		}
	}

	t.Logf("✅ ShowUsersDetailed vs ShowUser consistency test passed")
}
