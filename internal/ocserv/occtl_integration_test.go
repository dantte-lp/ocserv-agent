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

// TestMain validates fixtures before running tests
func TestMain(m *testing.M) {
	// Note: TestMain does not receive *testing.T, so we skip validation here
	// Individual tests will validate fixtures as needed
	m.Run()
}

// TestFixturesValidation validates all fixtures are present and valid
func TestFixturesValidation(t *testing.T) {
	testutil.SkipIfShort(t, "fixture validation")

	t.Log("Validating all occtl fixtures...")
	testutil.ValidateAllFixtures(t)
}

// TestMockSocketConnection tests basic connection to mock socket
func TestMockSocketConnection(t *testing.T) {
	testutil.SkipIfShort(t, "mock socket connection")

	// Create mock socket (assumes compose environment)
	mock := testutil.NewMockSocket(t, testutil.MockSocketConfig{
		UseCompose: true,
	})
	defer mock.Close()

	// Wait for socket to be ready
	if err := mock.WaitReady(t); err != nil {
		t.Fatalf("Mock socket not ready: %v", err)
	}

	// Create occtl manager
	logger := testutil.NewTestLogger(t)
	manager := ocserv.NewOcctlManager(
		mock.SocketPath(),
		"", // No sudo in tests
		5*time.Second,
		logger,
	)

	// Test basic command - ShowUsers
	ctx, cancel := testutil.NewTestContext(t, 10*time.Second)
	defer cancel()

	users, err := manager.ShowUsers(ctx)
	testutil.RequireNoError(t, err, "ShowUsers failed")

	t.Logf("Retrieved %d users from mock socket", len(users))

	// Validate we got expected number of users from fixture
	expectedCount := testutil.ExpectedUsersCount(t)
	testutil.AssertEqual(t, expectedCount, len(users), "User count mismatch")

	t.Logf("✅ Mock socket connection test passed")
}

// TestShowUsers tests ShowUsers command with JSON parsing
func TestShowUsers(t *testing.T) {
	testutil.SkipIfShort(t, "ShowUsers integration test")

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

	// Execute
	users, err := manager.ShowUsers(ctx)
	testutil.RequireNoError(t, err, "ShowUsers failed")

	// Validate response structure
	if len(users) == 0 {
		t.Skip("No users in fixture (empty test)")
	}

	// Check first user has required fields
	user := users[0]
	testutil.AssertNotEmpty(t, user.Username, "Username should not be empty")
	testutil.AssertNotEmpty(t, user.RemoteIP, "RemoteIP should not be empty")
	testutil.AssertNotEmpty(t, user.IPv4, "IPv4 should not be empty")

	t.Logf("First user: ID=%d, Username=%s, RemoteIP=%s, IPv4=%s",
		user.ID, user.Username, user.RemoteIP, user.IPv4)

	// Validate all users have IDs
	for i, u := range users {
		if u.ID == 0 {
			t.Errorf("User %d has zero ID", i)
		}
	}

	t.Logf("✅ ShowUsers test passed (%d users)", len(users))
}

// TestShowUsersDetailed tests ShowUsersDetailed command
func TestShowUsersDetailed(t *testing.T) {
	testutil.SkipIfShort(t, "ShowUsersDetailed integration test")

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

	// Execute
	users, err := manager.ShowUsersDetailed(ctx)
	testutil.RequireNoError(t, err, "ShowUsersDetailed failed")

	// Validate
	expectedCount := testutil.ExpectedUsersCount(t)
	testutil.AssertEqual(t, expectedCount, len(users), "Detailed user count mismatch")

	if len(users) > 0 {
		// Check detailed fields exist
		user := users[0]
		testutil.AssertNotEmpty(t, user.Username, "Username")
		testutil.AssertNotEmpty(t, user.RemoteIP, "RemoteIP")

		t.Logf("First detailed user: %s from %s", user.Username, user.RemoteIP)
	}

	t.Logf("✅ ShowUsersDetailed test passed (%d users)", len(users))
}

// TestShowStatusDetailed tests ShowStatusDetailed command
func TestShowStatusDetailed(t *testing.T) {
	testutil.SkipIfShort(t, "ShowStatusDetailed integration test")

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

	// Execute
	status, err := manager.ShowStatusDetailed(ctx)
	testutil.RequireNoError(t, err, "ShowStatusDetailed failed")

	// Validate
	testutil.AssertNotEmpty(t, status.Status, "Status field should not be empty")

	t.Logf("Server status: %s, Active sessions: %d, Uptime: %d",
		status.Status, status.ActiveSessions, status.Uptime)

	t.Logf("✅ ShowStatusDetailed test passed")
}

// TestShowSessions tests session-related commands
func TestShowSessions(t *testing.T) {
	testutil.SkipIfShort(t, "ShowSessions integration test")

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

	// Test ShowSessionsAll
	t.Run("ShowSessionsAll", func(t *testing.T) {
		sessions, err := manager.ShowSessionsAll(ctx)
		testutil.RequireNoError(t, err, "ShowSessionsAll failed")

		expectedCount := testutil.ExpectedSessionsCount(t)
		testutil.AssertEqual(t, expectedCount, len(sessions), "Sessions count mismatch")

		t.Logf("Retrieved %d sessions", len(sessions))
	})

	// Test ShowSessionsValid
	t.Run("ShowSessionsValid", func(t *testing.T) {
		sessions, err := manager.ShowSessionsValid(ctx)
		testutil.RequireNoError(t, err, "ShowSessionsValid failed")

		t.Logf("Retrieved %d valid sessions", len(sessions))
	})
}

// TestShowIRoutes tests ShowIRoutes command
func TestShowIRoutes(t *testing.T) {
	testutil.SkipIfShort(t, "ShowIRoutes integration test")

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

	// Execute
	iroutes, err := manager.ShowIRoutes(ctx)
	testutil.RequireNoError(t, err, "ShowIRoutes failed")

	t.Logf("Retrieved %d iroutes", len(iroutes))

	// Validate structure if iroutes exist
	if len(iroutes) > 0 {
		route := iroutes[0]
		testutil.AssertNotEmpty(t, route.Username, "IRoute username")
		// IRoutes is an array, just check it's not nil
		if route.IRoutes == nil {
			t.Error("IRoutes is nil")
		}

		t.Logf("First iroute: %s (routes: %d)", route.Username, len(route.IRoutes))
	}

	t.Logf("✅ ShowIRoutes test passed")
}

// TestShowIPBanPoints tests ShowIPBanPoints command
func TestShowIPBanPoints(t *testing.T) {
	testutil.SkipIfShort(t, "ShowIPBanPoints integration test")

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

	// Execute
	points, err := manager.ShowIPBanPoints(ctx)
	testutil.RequireNoError(t, err, "ShowIPBanPoints failed")

	t.Logf("Retrieved %d IP ban points entries", len(points))

	t.Logf("✅ ShowIPBanPoints test passed")
}

// TestContextTimeout tests that operations respect context timeout
func TestContextTimeout(t *testing.T) {
	testutil.SkipIfShort(t, "context timeout test")

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
	_, err := manager.ShowUsers(ctx)
	if err == nil {
		t.Fatal("Expected timeout error, got nil")
	}

	t.Logf("Correctly received error on timeout: %v", err)
	t.Logf("✅ Context timeout test passed")
}

// TestConcurrentRequests tests concurrent access to mock socket
func TestConcurrentRequests(t *testing.T) {
	testutil.SkipIfShort(t, "concurrent requests test")

	// Setup
	mock := testutil.NewMockSocket(t, testutil.MockSocketConfig{UseCompose: true})
	defer mock.Close()

	if err := mock.WaitReady(t); err != nil {
		t.Fatalf("Mock socket not ready: %v", err)
	}

	logger := testutil.NewTestLogger(t)
	manager := ocserv.NewOcctlManager(mock.SocketPath(), "", 5*time.Second, logger)

	// Run 10 concurrent requests
	const numRequests = 10
	done := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(id int) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			_, err := manager.ShowUsers(ctx)
			done <- err
		}(i)
	}

	// Wait for all to complete
	for i := 0; i < numRequests; i++ {
		if err := <-done; err != nil {
			t.Errorf("Concurrent request %d failed: %v", i, err)
		}
	}

	t.Logf("✅ Concurrent requests test passed (%d requests)", numRequests)
}
