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

// TestShowIPBanPointsStructure tests ShowIPBanPoints structure validation
func TestShowIPBanPointsStructure(t *testing.T) {
	testutil.SkipIfShort(t, "ShowIPBanPoints structure test")

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

	// Get IP ban points
	points, err := manager.ShowIPBanPoints(ctx)
	testutil.RequireNoError(t, err, "ShowIPBanPoints failed")

	// Validate structure
	if len(points) == 0 {
		t.Skip("No IP ban points in fixture")
	}

	// Check first entry
	point := points[0]
	testutil.AssertNotEmpty(t, point.IP, "IP address")
	if point.Points <= 0 {
		t.Errorf("Invalid points value: %d (expected > 0)", point.Points)
	}

	t.Logf("First entry: IP=%s, Points=%d", point.IP, point.Points)
	t.Logf("Total IPs with ban points: %d", len(points))
	t.Logf("✅ ShowIPBanPoints structure test passed")
}

// TestShowIPBanPointsScoreRanges tests score validation
func TestShowIPBanPointsScoreRanges(t *testing.T) {
	testutil.SkipIfShort(t, "ShowIPBanPoints score ranges test")

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

	// Get IP ban points
	points, err := manager.ShowIPBanPoints(ctx)
	testutil.RequireNoError(t, err, "ShowIPBanPoints failed")

	if len(points) == 0 {
		t.Skip("No IP ban points in fixture")
	}

	// Analyze points distribution
	var lowPoints, mediumPoints, highPoints int
	for _, point := range points {
		if point.Points < 5 {
			lowPoints++
		} else if point.Points < 20 {
			mediumPoints++
		} else {
			highPoints++
		}

		// Validate points is positive
		if point.Points <= 0 {
			t.Errorf("Invalid points %d for IP %s", point.Points, point.IP)
		}
	}

	t.Logf("Points distribution: low(<5)=%d, medium(5-19)=%d, high(>=20)=%d",
		lowPoints, mediumPoints, highPoints)
	t.Logf("✅ ShowIPBanPoints score ranges test passed")
}

// TestShowIPBanPointsMultipleIPs tests handling multiple IPs
func TestShowIPBanPointsMultipleIPs(t *testing.T) {
	testutil.SkipIfShort(t, "ShowIPBanPoints multiple IPs test")

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

	// Get IP ban points
	points, err := manager.ShowIPBanPoints(ctx)
	testutil.RequireNoError(t, err, "ShowIPBanPoints failed")

	if len(points) == 0 {
		t.Skip("No IP ban points in fixture")
	}

	// Check for unique IPs
	ipMap := make(map[string]int)
	for _, point := range points {
		ipMap[point.IP] = point.Points
	}

	if len(ipMap) != len(points) {
		t.Errorf("Duplicate IPs found: %d unique IPs vs %d total entries",
			len(ipMap), len(points))
	}

	t.Logf("Unique IPs: %d", len(ipMap))
	t.Logf("✅ ShowIPBanPoints multiple IPs test passed")
}

// TestShowIPBanPointsIPFormats tests IP address format validation
func TestShowIPBanPointsIPFormats(t *testing.T) {
	testutil.SkipIfShort(t, "ShowIPBanPoints IP formats test")

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

	// Get IP ban points
	points, err := manager.ShowIPBanPoints(ctx)
	testutil.RequireNoError(t, err, "ShowIPBanPoints failed")

	if len(points) == 0 {
		t.Skip("No IP ban points in fixture")
	}

	// Validate IP formats (basic validation)
	for _, point := range points {
		// Check IP is not empty
		if point.IP == "" {
			t.Errorf("Empty IP address found")
		}

		// Check IP contains dots (IPv4) or colons (IPv6)
		if !containsDot(point.IP) && !containsColon(point.IP) {
			t.Errorf("Invalid IP format: %s", point.IP)
		}
	}

	t.Logf("All IPs validated (%d entries)", len(points))
	t.Logf("✅ ShowIPBanPoints IP formats test passed")
}

// TestShowIPBans tests ShowIPBans command
func TestShowIPBans(t *testing.T) {
	testutil.SkipIfShort(t, "ShowIPBans test")

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

	// Get banned IPs
	bans, err := manager.ShowIPBans(ctx)
	testutil.RequireNoError(t, err, "ShowIPBans failed")

	// Fixture has empty array (no banned IPs)
	testutil.AssertEqual(t, 0, len(bans), "Banned IPs count")

	t.Logf("Banned IPs: %d", len(bans))
	t.Logf("✅ ShowIPBans test passed")
}

// TestShowIPBansStructure tests ShowIPBans structure (if bans exist)
func TestShowIPBansStructure(t *testing.T) {
	testutil.SkipIfShort(t, "ShowIPBans structure test")

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

	// Get banned IPs
	bans, err := manager.ShowIPBans(ctx)
	testutil.RequireNoError(t, err, "ShowIPBans failed")

	if len(bans) == 0 {
		t.Log("No banned IPs (expected for mock fixture)")
		t.Logf("✅ ShowIPBans structure test passed (empty)")
		return
	}

	// Validate structure if bans exist
	ban := bans[0]
	testutil.AssertNotEmpty(t, ban.IP, "Banned IP address")

	t.Logf("First banned IP: %s", ban.IP)
	t.Logf("✅ ShowIPBans structure test passed")
}

// TestUnbanIPWithValidIP tests UnbanIP with valid IP
func TestUnbanIPWithValidIP(t *testing.T) {
	testutil.SkipIfShort(t, "UnbanIP valid IP test")

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

	// Unban a sample IP
	testIP := "185.224.128.136"
	err := manager.UnbanIP(ctx, testIP)
	testutil.RequireNoError(t, err, "UnbanIP failed")

	t.Logf("Successfully unbanned IP: %s", testIP)
	t.Logf("✅ UnbanIP valid IP test passed")
}

// TestUnbanIPWithInvalidIP tests UnbanIP with invalid IP format
func TestUnbanIPWithInvalidIP(t *testing.T) {
	testutil.SkipIfShort(t, "UnbanIP invalid IP test")

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

	// Test various invalid IPs
	invalidIPs := []string{
		"",                // Empty
		"not-an-ip",       // Invalid format
		"999.999.999.999", // Invalid octets
		"192.168.1",       // Incomplete
	}

	for _, ip := range invalidIPs {
		t.Run("IP_"+ip, func(t *testing.T) {
			// Mock server will accept any argument, validation should be done in real ocserv
			// Here we just verify the command doesn't panic
			_ = manager.UnbanIP(ctx, ip)
			t.Logf("UnbanIP with '%s' completed (no panic)", ip)
		})
	}

	t.Logf("✅ UnbanIP invalid IP test passed")
}

// TestUnbanIPWithTimeout tests UnbanIP timeout handling
func TestUnbanIPWithTimeout(t *testing.T) {
	testutil.SkipIfShort(t, "UnbanIP timeout test")

	// Setup
	mock := testutil.NewMockSocket(t, testutil.MockSocketConfig{UseCompose: true})
	defer mock.Close()

	if err := mock.WaitReady(t); err != nil {
		t.Fatalf("Mock socket not ready: %v", err)
	}

	logger := testutil.NewTestLogger(t)
	manager := ocserv.NewOcctlManager(mock.SocketPath(), "", 5*time.Second, logger)

	// Very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Wait for context to expire
	time.Sleep(10 * time.Millisecond)

	// Try to unban IP
	err := manager.UnbanIP(ctx, "192.168.1.1")
	testutil.RequireError(t, err, "UnbanIP should fail with timeout")

	t.Logf("Expected timeout error: %v", err)
	t.Logf("✅ UnbanIP timeout test passed")
}

// TestUnbanIPMultiple tests unbanning multiple IPs
func TestUnbanIPMultiple(t *testing.T) {
	testutil.SkipIfShort(t, "UnbanIP multiple IPs test")

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

	// Unban multiple IPs
	testIPs := []string{
		"185.224.128.136",
		"95.214.210.2",
		"64.62.197.77",
		"167.94.138.33",
		"204.76.203.30",
	}

	for _, ip := range testIPs {
		err := manager.UnbanIP(ctx, ip)
		testutil.RequireNoError(t, err, "UnbanIP failed for "+ip)
		t.Logf("Unbanned: %s", ip)
	}

	t.Logf("✅ UnbanIP multiple IPs test passed (%d IPs)", len(testIPs))
}

// TestReload tests Reload command
func TestReload(t *testing.T) {
	testutil.SkipIfShort(t, "Reload test")

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

	// Reload configuration
	err := manager.Reload(ctx)
	testutil.RequireNoError(t, err, "Reload failed")

	t.Log("Successfully reloaded configuration")
	t.Logf("✅ Reload test passed")
}

// TestReloadWithTimeout tests Reload timeout handling
func TestReloadWithTimeout(t *testing.T) {
	testutil.SkipIfShort(t, "Reload timeout test")

	// Setup
	mock := testutil.NewMockSocket(t, testutil.MockSocketConfig{UseCompose: true})
	defer mock.Close()

	if err := mock.WaitReady(t); err != nil {
		t.Fatalf("Mock socket not ready: %v", err)
	}

	logger := testutil.NewTestLogger(t)
	manager := ocserv.NewOcctlManager(mock.SocketPath(), "", 5*time.Second, logger)

	// Very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Wait for context to expire
	time.Sleep(10 * time.Millisecond)

	// Try to reload
	err := manager.Reload(ctx)
	testutil.RequireError(t, err, "Reload should fail with timeout")

	t.Logf("Expected timeout error: %v", err)
	t.Logf("✅ Reload timeout test passed")
}

// TestReloadMultipleCalls tests multiple reload calls
func TestReloadMultipleCalls(t *testing.T) {
	testutil.SkipIfShort(t, "Reload multiple calls test")

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

	// Multiple reload calls
	const numReloads = 5
	for i := 0; i < numReloads; i++ {
		err := manager.Reload(ctx)
		testutil.RequireNoError(t, err, "Reload failed")
		t.Logf("Reload %d/%d completed", i+1, numReloads)
	}

	t.Logf("✅ Reload multiple calls test passed (%d reloads)", numReloads)
}

// TestIPManagementOperationsSequence tests sequence of IP management operations
func TestIPManagementOperationsSequence(t *testing.T) {
	testutil.SkipIfShort(t, "IP management operations sequence test")

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

	// 1. Check ban points
	points, err := manager.ShowIPBanPoints(ctx)
	testutil.RequireNoError(t, err, "ShowIPBanPoints failed")
	t.Logf("Step 1: Found %d IPs with ban points", len(points))

	// 2. Check banned IPs
	bans, err := manager.ShowIPBans(ctx)
	testutil.RequireNoError(t, err, "ShowIPBans failed")
	t.Logf("Step 2: Found %d banned IPs", len(bans))

	// 3. Unban an IP (if any)
	if len(points) > 0 {
		testIP := points[0].IP
		err = manager.UnbanIP(ctx, testIP)
		testutil.RequireNoError(t, err, "UnbanIP failed")
		t.Logf("Step 3: Unbanned IP %s", testIP)
	} else {
		t.Log("Step 3: No IPs to unban")
	}

	// 4. Reload configuration
	err = manager.Reload(ctx)
	testutil.RequireNoError(t, err, "Reload failed")
	t.Log("Step 4: Reloaded configuration")

	// 5. Check ban points again
	pointsAfter, err := manager.ShowIPBanPoints(ctx)
	testutil.RequireNoError(t, err, "ShowIPBanPoints failed")
	t.Logf("Step 5: Found %d IPs with ban points (after reload)", len(pointsAfter))

	t.Logf("✅ IP management operations sequence test passed")
}

// Helper functions

func containsDot(s string) bool {
	for _, c := range s {
		if c == '.' {
			return true
		}
	}
	return false
}

func containsColon(s string) bool {
	for _, c := range s {
		if c == ':' {
			return true
		}
	}
	return false
}
