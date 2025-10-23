package testutil

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// SystemdTestHelper helps manage test systemd services
type SystemdTestHelper struct {
	ServiceName    string
	ServiceFile    string
	UserMode       bool
	UnitDir        string
	originalDir    string
	setupCompleted bool
}

// NewSystemdTestHelper creates a new systemd test helper
func NewSystemdTestHelper(t *testing.T, serviceName string) *SystemdTestHelper {
	t.Helper()

	// Check if systemd is available
	if !IsSystemdAvailable() {
		t.Skip("systemd not available")
	}

	return &SystemdTestHelper{
		ServiceName: serviceName,
		UserMode:    true, // Use user systemd by default
	}
}

// IsSystemdAvailable checks if systemd is available on the system
func IsSystemdAvailable() bool {
	cmd := exec.Command("systemctl", "--version")
	return cmd.Run() == nil
}

// Setup prepares the test service
func (h *SystemdTestHelper) Setup(t *testing.T) error {
	t.Helper()

	if h.setupCompleted {
		return nil
	}

	// Get user systemd unit directory
	var err error
	h.UnitDir, err = h.getUserUnitDir()
	if err != nil {
		return fmt.Errorf("failed to get user unit dir: %w", err)
	}

	// Create unit directory if it doesn't exist
	if err := os.MkdirAll(h.UnitDir, 0755); err != nil {
		return fmt.Errorf("failed to create unit dir: %w", err)
	}

	// Find test service file in fixtures
	// Try multiple possible paths
	possiblePaths := []string{
		filepath.Join("../../../test/fixtures/systemd", h.ServiceName+".service"),
		filepath.Join("../../test/fixtures/systemd", h.ServiceName+".service"),
		filepath.Join("test/fixtures/systemd", h.ServiceName+".service"),
	}

	var fixtureFile string
	var found bool
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			fixtureFile = path
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("fixture service file not found in any of: %v", possiblePaths)
	}

	// Copy service file to user systemd directory
	h.ServiceFile = filepath.Join(h.UnitDir, h.ServiceName+".service")
	if err := h.copyFile(fixtureFile, h.ServiceFile); err != nil {
		return fmt.Errorf("failed to copy service file: %w", err)
	}

	t.Logf("Created test service: %s", h.ServiceFile)

	// Reload systemd to pick up the new service
	if err := h.reloadDaemon(); err != nil {
		return fmt.Errorf("failed to reload systemd: %w", err)
	}

	h.setupCompleted = true
	return nil
}

// Cleanup removes the test service
func (h *SystemdTestHelper) Cleanup(t *testing.T) {
	t.Helper()

	if !h.setupCompleted {
		return
	}

	// Stop the service if it's running
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_ = h.stopService(ctx)

	// Remove service file
	if h.ServiceFile != "" {
		if err := os.Remove(h.ServiceFile); err != nil {
			t.Logf("Warning: failed to remove service file: %v", err)
		}
	}

	// Reload systemd
	_ = h.reloadDaemon()

	t.Logf("Cleaned up test service: %s", h.ServiceName)
}

// getUserUnitDir returns the user systemd unit directory
func (h *SystemdTestHelper) getUserUnitDir() (string, error) {
	// Try XDG_CONFIG_HOME first
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		return filepath.Join(xdgConfig, "systemd", "user"), nil
	}

	// Fall back to ~/.config/systemd/user
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, ".config", "systemd", "user"), nil
}

// copyFile copies a file from src to dst
func (h *SystemdTestHelper) copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst, data, 0644)
}

// reloadDaemon reloads systemd daemon
func (h *SystemdTestHelper) reloadDaemon() error {
	args := []string{"daemon-reload"}
	if h.UserMode {
		args = append([]string{"--user"}, args...)
	}

	cmd := exec.Command("systemctl", args...)
	return cmd.Run()
}

// stopService stops the test service
func (h *SystemdTestHelper) stopService(ctx context.Context) error {
	args := []string{"stop", h.ServiceName}
	if h.UserMode {
		args = append([]string{"--user"}, args...)
	}

	cmd := exec.CommandContext(ctx, "systemctl", args...)
	return cmd.Run()
}

// GetServiceName returns the full service name
func (h *SystemdTestHelper) GetServiceName() string {
	return h.ServiceName
}

// RunSystemctl runs a systemctl command for testing
func (h *SystemdTestHelper) RunSystemctl(ctx context.Context, args ...string) (string, string, error) {
	if h.UserMode {
		args = append([]string{"--user"}, args...)
	}

	cmd := exec.CommandContext(ctx, "systemctl", args...)

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// WaitForState waits for the service to reach a specific state
func (h *SystemdTestHelper) WaitForState(ctx context.Context, expectedState string, maxWait time.Duration) error {
	deadline := time.Now().Add(maxWait)

	for time.Now().Before(deadline) {
		stdout, _, err := h.RunSystemctl(ctx, "is-active", h.ServiceName)
		state := strings.TrimSpace(stdout)

		if state == expectedState {
			return nil
		}

		// If there's an error but we're waiting for "inactive" or "failed", that's OK
		if err != nil && (expectedState == "inactive" || expectedState == "failed") {
			if state == expectedState {
				return nil
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
			// Continue polling
		}
	}

	return fmt.Errorf("timeout waiting for state %s", expectedState)
}
