// Package testutil provides test utilities for integration testing
package testutil

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// MockSocketConfig holds configuration for mock socket server
type MockSocketConfig struct {
	// SocketPath is the path to Unix socket (default: /tmp/occtl-test-{random}.socket)
	SocketPath string

	// FixturesPath is the path to fixtures directory (default: ../../test/fixtures/ocserv/occtl)
	FixturesPath string

	// StartTimeout is max time to wait for socket to become ready (default: 5s)
	StartTimeout time.Duration

	// UseCompose indicates whether to use podman-compose mock-ocserv service
	// If true, assumes mock-ocserv container is running with socket at /var/run/occtl.socket
	UseCompose bool

	// ComposeSocketVolume is the volume name for shared socket (default: mock-socket)
	ComposeSocketVolume string
}

// MockSocket represents a mock ocserv socket server for testing
type MockSocket struct {
	config     MockSocketConfig
	socketPath string
	cleanup    func()
}

// NewMockSocket creates a new mock socket helper
//
// For compose-based testing (recommended):
//
//	mock := NewMockSocket(t, MockSocketConfig{UseCompose: true})
//	defer mock.Close()
//
// For local testing (development only):
//
//	mock := NewMockSocket(t, MockSocketConfig{SocketPath: "/tmp/test.socket"})
//	defer mock.Close()
func NewMockSocket(t *testing.T, cfg MockSocketConfig) *MockSocket {
	t.Helper()

	// Set defaults
	if cfg.StartTimeout == 0 {
		cfg.StartTimeout = 5 * time.Second
	}

	if cfg.UseCompose {
		// Compose mode: socket should already exist in shared volume
		// We don't manage the server lifecycle, just verify socket exists
		if cfg.ComposeSocketVolume == "" {
			cfg.ComposeSocketVolume = "mock-socket"
		}

		// In compose mode, socket is at standard path
		cfg.SocketPath = "/var/run/occtl.socket"

		// Note: This assumes tests run inside a container that shares the volume
		// For host-side testing, need to use podman volume inspect to find mount point
		t.Logf("Using compose mock-ocserv (socket: %s)", cfg.SocketPath)

		return &MockSocket{
			config:     cfg,
			socketPath: cfg.SocketPath,
			cleanup:    func() {}, // No cleanup needed for compose
		}
	}

	// Local mode: for development only
	t.Log("WARNING: Using local mock socket (not recommended for CI)")

	if cfg.SocketPath == "" {
		// Generate random socket path
		cfg.SocketPath = filepath.Join(os.TempDir(), fmt.Sprintf("occtl-test-%d.socket", time.Now().UnixNano()))
	}

	if cfg.FixturesPath == "" {
		cfg.FixturesPath = "../../test/fixtures/ocserv/occtl"
	}

	// For local testing, we'd need to start mock-ocserv binary
	// This is left as TODO since compose is recommended approach
	t.Fatalf("Local mock socket not implemented - use compose mode (UseCompose: true)")

	return nil
}

// SocketPath returns the Unix socket path for connecting to mock server
func (m *MockSocket) SocketPath() string {
	return m.socketPath
}

// WaitReady waits for socket to become available
func (m *MockSocket) WaitReady(t *testing.T) error {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), m.config.StartTimeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for socket: %w", ctx.Err())

		case <-ticker.C:
			// Try to connect to socket
			conn, err := net.DialTimeout("unix", m.socketPath, 100*time.Millisecond)
			if err == nil {
				conn.Close()
				t.Logf("Mock socket ready: %s", m.socketPath)
				return nil
			}

			// Check if socket file exists (even if not accepting connections yet)
			if _, err := os.Stat(m.socketPath); err == nil {
				t.Logf("Socket file exists but not ready yet: %s", m.socketPath)
			}
		}
	}
}

// Close cleans up mock socket resources
func (m *MockSocket) Close() {
	if m.cleanup != nil {
		m.cleanup()
	}
}
