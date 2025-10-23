//go:build integration
// +build integration

package ocserv_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/dantte-lp/ocserv-agent/internal/ocserv"
	"github.com/dantte-lp/ocserv-agent/internal/ocserv/testutil"
)

// TestShowUsersStructure tests ShowUsers response structure in detail
func TestShowUsersStructure(t *testing.T) {
	testutil.SkipIfShort(t, "ShowUsers structure test")

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

	if len(users) == 0 {
		t.Skip("No users in fixture")
	}

	// Validate first user structure
	user := users[0]

	t.Run("RequiredFields", func(t *testing.T) {
		// ID should be non-zero
		if user.ID == 0 {
			t.Error("User ID is zero")
		}

		// Username should not be empty
		testutil.AssertNotEmpty(t, user.Username, "Username")

		// State should not be empty
		testutil.AssertNotEmpty(t, user.State, "State")

		// RemoteIP should not be empty
		testutil.AssertNotEmpty(t, user.RemoteIP, "RemoteIP")

		// IPv4 should not be empty
		testutil.AssertNotEmpty(t, user.IPv4, "IPv4")
	})

	t.Run("OptionalFields", func(t *testing.T) {
		// Check that optional fields are parsed (may be empty)
		t.Logf("Groupname: %s", user.Groupname)
		t.Logf("Device: %s", user.Device)
		t.Logf("MTU: %s", user.MTU)
		t.Logf("UserAgent: %s", user.UserAgent)
	})

	t.Run("NetworkFields", func(t *testing.T) {
		// IPv4 address format
		if user.IPv4 != "" {
			t.Logf("IPv4: %s", user.IPv4)
		}

		// Point-to-Point IPv4
		if user.PtPIPv4 != "" {
			t.Logf("P-t-P IPv4: %s", user.PtPIPv4)
		}

		// IPv6 (may be empty)
		if user.IPv6 != "" {
			t.Logf("IPv6: %s", user.IPv6)
		}
	})

	t.Run("TrafficStats", func(t *testing.T) {
		// RX/TX should be present
		testutil.AssertNotEmpty(t, user.RX, "RX")
		testutil.AssertNotEmpty(t, user.TX, "TX")

		// Readable versions
		t.Logf("RX: %s (%s)", user.RX, user.ReadableRX)
		t.Logf("TX: %s (%s)", user.TX, user.ReadableTX)
	})

	t.Run("ConnectionDetails", func(t *testing.T) {
		// Connected at timestamp
		testutil.AssertNotEmpty(t, user.ConnectedAt, "ConnectedAt")

		// Raw timestamp
		if user.RawConnectedAt == 0 {
			t.Error("RawConnectedAt is zero")
		}

		// Session info
		testutil.AssertNotEmpty(t, user.Session, "Session")

		t.Logf("Connected at: %s (raw: %d)", user.ConnectedAt, user.RawConnectedAt)
		t.Logf("Session: %s", user.Session)
	})

	t.Run("SecurityInfo", func(t *testing.T) {
		// TLS/DTLS ciphers
		if user.TLSCiphersuite != "" {
			t.Logf("TLS: %s", user.TLSCiphersuite)
		}
		if user.DTLSCipher != "" {
			t.Logf("DTLS: %s", user.DTLSCipher)
		}
	})

	t.Run("Routes", func(t *testing.T) {
		// Routes can be array or string
		switch routes := user.Routes.(type) {
		case []interface{}:
			t.Logf("Routes (array): %v", routes)
		case string:
			t.Logf("Routes (string): %s", routes)
		case nil:
			t.Log("No routes")
		default:
			t.Logf("Routes (unknown type): %T", routes)
		}

		// DNS servers
		if len(user.DNS) > 0 {
			t.Logf("DNS servers: %v", user.DNS)
		}
	})

	t.Logf("✅ User structure validation passed for %s", user.Username)
}

// TestShowUsersMultipleUsers tests handling of multiple users
func TestShowUsersMultipleUsers(t *testing.T) {
	testutil.SkipIfShort(t, "ShowUsers multiple users test")

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

	// Get expected count from fixture
	expectedCount := testutil.ExpectedUsersCount(t)

	// Validate count
	testutil.AssertEqual(t, expectedCount, len(users), "User count mismatch")

	// Validate all users have unique IDs
	seenIDs := make(map[int]bool)
	for i, user := range users {
		if seenIDs[user.ID] {
			t.Errorf("Duplicate user ID %d at index %d", user.ID, i)
		}
		seenIDs[user.ID] = true

		// Log user summary
		t.Logf("User %d: ID=%d, Username=%s, IP=%s, State=%s",
			i+1, user.ID, user.Username, user.IPv4, user.State)
	}

	t.Logf("✅ Multiple users test passed (%d users)", len(users))
}

// TestShowUsersJSONParsing tests JSON parsing edge cases
func TestShowUsersJSONParsing(t *testing.T) {
	testutil.SkipIfShort(t, "ShowUsers JSON parsing test")

	// Load fixture directly
	fixtureData := testutil.LoadFixtureJSON(t, "occtl -j show users")

	// Parse as JSON array
	var users []ocserv.User
	if err := json.Unmarshal(fixtureData, &users); err != nil {
		t.Fatalf("Failed to unmarshal fixture: %v", err)
	}

	t.Logf("Parsed %d users from fixture", len(users))

	// Validate each user can be marshaled back
	for i, user := range users {
		data, err := json.Marshal(user)
		if err != nil {
			t.Errorf("Failed to marshal user %d: %v", i, err)
			continue
		}

		// Unmarshal again to verify round-trip
		var user2 ocserv.User
		if err := json.Unmarshal(data, &user2); err != nil {
			t.Errorf("Failed to unmarshal user %d after marshal: %v", i, err)
			continue
		}

		// Verify key fields match
		if user.ID != user2.ID {
			t.Errorf("User %d: ID mismatch after round-trip: %d != %d", i, user.ID, user2.ID)
		}
		if user.Username != user2.Username {
			t.Errorf("User %d: Username mismatch after round-trip: %s != %s", i, user.Username, user2.Username)
		}
	}

	t.Logf("✅ JSON parsing round-trip test passed")
}

// TestShowUsersFieldTypes tests that all field types are correct
func TestShowUsersFieldTypes(t *testing.T) {
	testutil.SkipIfShort(t, "ShowUsers field types test")

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

	if len(users) == 0 {
		t.Skip("No users in fixture")
	}

	user := users[0]

	// Test numeric fields
	t.Run("NumericFields", func(t *testing.T) {
		if user.ID <= 0 {
			t.Errorf("ID should be positive: %d", user.ID)
		}

		if user.RawConnectedAt <= 0 {
			t.Errorf("RawConnectedAt should be positive: %d", user.RawConnectedAt)
		}
	})

	// Test string fields
	t.Run("StringFields", func(t *testing.T) {
		stringFields := map[string]string{
			"Username": user.Username,
			"State":    user.State,
			"RemoteIP": user.RemoteIP,
			"IPv4":     user.IPv4,
			"Device":   user.Device,
			"Session":  user.Session,
			"RX":       user.RX,
			"TX":       user.TX,
		}

		for name, value := range stringFields {
			if value == "" && (name == "Username" || name == "State" || name == "RemoteIP" || name == "IPv4") {
				t.Errorf("Required field %s is empty", name)
			}
		}
	})

	// Test array fields
	t.Run("ArrayFields", func(t *testing.T) {
		// DNS can be empty array
		if user.DNS == nil {
			t.Log("DNS field is nil (acceptable)")
		} else {
			t.Logf("DNS: %v", user.DNS)
		}

		// NBNS can be empty array
		if user.NBNS == nil {
			t.Log("NBNS field is nil (acceptable)")
		} else {
			t.Logf("NBNS: %v", user.NBNS)
		}
	})

	t.Logf("✅ Field types test passed")
}

// TestShowUsersEmptyResult tests handling when no users are connected
// Note: This test uses the fixture which always has users, so it's mainly
// for documentation and future when we have empty fixture
func TestShowUsersEmptyResultHandling(t *testing.T) {
	testutil.SkipIfShort(t, "ShowUsers empty result test")

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
	testutil.RequireNoError(t, err, "ShowUsers should not fail on empty result")

	// With current fixture, we have users
	// But if empty, should return empty slice, not nil
	if users == nil {
		t.Error("ShowUsers should return empty slice, not nil")
	}

	t.Logf("✅ Empty result handling test passed (got %d users)", len(users))
}
