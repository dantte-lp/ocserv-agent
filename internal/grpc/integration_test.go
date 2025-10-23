//go:build integration

package grpc_test

import (
	"context"
	"testing"
	"time"

	grpctestutil "github.com/dantte-lp/ocserv-agent/internal/testutil/grpc"
	pb "github.com/dantte-lp/ocserv-agent/pkg/proto/agent/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestGRPCServerStartup tests that the gRPC server starts successfully
func TestGRPCServerStartup(t *testing.T) {
	tests := []struct {
		name      string
		enableTLS bool
	}{
		{
			name:      "with mTLS",
			enableTLS: true,
		},
		{
			name:      "without TLS",
			enableTLS: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Start test server
			serverOpts := grpctestutil.DefaultServerOptions()
			serverOpts.EnableTLS = tt.enableTLS

			server := grpctestutil.NewTestServer(t, serverOpts)

			// Verify server is running
			if server.Server == nil {
				t.Error("Server is nil")
			}

			if server.Address == "" {
				t.Error("Server address is empty")
			}

			t.Logf("Server started successfully at %s", server.Address)
		})
	}
}

// TestGRPCClientConnection tests that clients can connect to the server
func TestGRPCClientConnection(t *testing.T) {
	tests := []struct {
		name      string
		enableTLS bool
	}{
		{
			name:      "with mTLS",
			enableTLS: true,
		},
		{
			name:      "without TLS",
			enableTLS: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Start test server
			serverOpts := grpctestutil.DefaultServerOptions()
			serverOpts.EnableTLS = tt.enableTLS

			server := grpctestutil.NewTestServer(t, serverOpts)

			// Create test client
			clientOpts := grpctestutil.DefaultClientOptions()
			clientOpts.EnableTLS = tt.enableTLS

			if tt.enableTLS {
				certFile, keyFile, caFile := server.GetCertFiles()
				clientOpts.CertFile = certFile
				clientOpts.KeyFile = keyFile
				clientOpts.CAFile = caFile
			}

			client := grpctestutil.NewTestClient(t, server.Address, clientOpts)

			// Verify client is connected
			if client.Client == nil {
				t.Error("Client is nil")
			}

			if client.Conn == nil {
				t.Error("Connection is nil")
			}

			t.Logf("Client connected successfully")
		})
	}
}

// TestHealthCheckRPC tests the HealthCheck RPC call
func TestHealthCheckRPC(t *testing.T) {
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
		tier        int32
		wantHealthy bool
		wantErr     bool
		errCode     codes.Code
	}{
		{
			name:        "tier 1 - basic heartbeat",
			tier:        1,
			wantHealthy: true,
			wantErr:     false,
		},
		{
			name:        "tier 2 - deep check",
			tier:        2,
			wantHealthy: true,
			wantErr:     false,
		},
		{
			name:        "tier 3 - not implemented",
			tier:        3,
			wantHealthy: false,
			wantErr:     false,
		},
		{
			name:    "invalid tier - too low",
			tier:    0,
			wantErr: true,
			errCode: codes.InvalidArgument,
		},
		{
			name:    "invalid tier - too high",
			tier:    4,
			wantErr: true,
			errCode: codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			req := &pb.HealthCheckRequest{
				Tier: tt.tier,
			}

			resp, err := client.Client.HealthCheck(ctx, req)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got nil")
					return
				}

				st, ok := status.FromError(err)
				if !ok {
					t.Errorf("Error is not a gRPC status error: %v", err)
					return
				}

				if st.Code() != tt.errCode {
					t.Errorf("Expected error code %v, got %v", tt.errCode, st.Code())
				}

				t.Logf("Got expected error: %v", err)
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if resp.Healthy != tt.wantHealthy {
				t.Errorf("Expected healthy=%v, got %v", tt.wantHealthy, resp.Healthy)
			}

			if resp.Timestamp == nil {
				t.Error("Timestamp is nil")
			}

			if resp.Checks == nil {
				t.Error("Checks map is nil")
			}

			t.Logf("HealthCheck response: healthy=%v, message=%q, checks=%d",
				resp.Healthy, resp.StatusMessage, len(resp.Checks))
		})
	}
}

// TestHealthCheckConcurrent tests concurrent health check requests
func TestHealthCheckConcurrent(t *testing.T) {
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

			req := &pb.HealthCheckRequest{Tier: 1}
			resp, err := client.Client.HealthCheck(ctx, req)

			if err != nil {
				errors <- err
				done <- false
				return
			}

			if !resp.Healthy {
				errors <- err
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
		t.Errorf("Concurrent request failed: %v", err)
	}

	if successCount != numRequests {
		t.Errorf("Expected %d successful requests, got %d", numRequests, successCount)
	}

	t.Logf("All %d concurrent requests succeeded", numRequests)
}

// TestServerGracefulShutdown tests graceful server shutdown
func TestServerGracefulShutdown(t *testing.T) {
	// Start test server
	server := grpctestutil.NewTestServer(t, grpctestutil.DefaultServerOptions())

	// Create test client
	certFile, keyFile, caFile := server.GetCertFiles()
	clientOpts := grpctestutil.DefaultClientOptions()
	clientOpts.CertFile = certFile
	clientOpts.KeyFile = keyFile
	clientOpts.CAFile = caFile

	client := grpctestutil.NewTestClient(t, server.Address, clientOpts)

	// Make a successful request
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &pb.HealthCheckRequest{Tier: 1}
	resp, err := client.Client.HealthCheck(ctx, req)

	if err != nil {
		t.Fatalf("Initial health check failed: %v", err)
	}

	if !resp.Healthy {
		t.Error("Expected healthy status, got unhealthy")
	}

	// Gracefully stop the server
	server.Server.GracefulStop()
	t.Log("Server gracefully stopped")

	// Try to make another request (should fail)
	ctx2, cancel2 := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel2()

	_, err = client.Client.HealthCheck(ctx2, req)
	if err == nil {
		t.Error("Expected error after server shutdown, got nil")
	} else {
		t.Logf("Got expected error after shutdown: %v", err)
	}
}

// TestMultipleClients tests multiple clients connecting to the same server
func TestMultipleClients(t *testing.T) {
	// Start test server
	server := grpctestutil.NewTestServer(t, grpctestutil.DefaultServerOptions())

	certFile, keyFile, caFile := server.GetCertFiles()

	// Create multiple clients
	const numClients = 5
	clients := make([]*grpctestutil.TestClient, numClients)

	for i := 0; i < numClients; i++ {
		clientOpts := grpctestutil.DefaultClientOptions()
		clientOpts.CertFile = certFile
		clientOpts.KeyFile = keyFile
		clientOpts.CAFile = caFile

		clients[i] = grpctestutil.NewTestClient(t, server.Address, clientOpts)
	}

	// Each client makes a request
	for i, client := range clients {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		req := &pb.HealthCheckRequest{Tier: 1}
		resp, err := client.Client.HealthCheck(ctx, req)

		if err != nil {
			t.Errorf("Client %d health check failed: %v", i, err)
			continue
		}

		if !resp.Healthy {
			t.Errorf("Client %d: Expected healthy status, got unhealthy", i)
		}

		t.Logf("Client %d: Health check succeeded", i)
	}
}

// TestPortAllocation tests that multiple servers can run on different ports
func TestPortAllocation(t *testing.T) {
	// Start multiple servers
	const numServers = 3
	servers := make([]*grpctestutil.TestServer, numServers)

	for i := 0; i < numServers; i++ {
		serverOpts := grpctestutil.DefaultServerOptions()
		serverOpts.AgentID = "test-agent-" + string(rune('A'+i))

		servers[i] = grpctestutil.NewTestServer(t, serverOpts)
	}

	// Verify all servers have different addresses
	addresses := make(map[string]bool)
	for i, server := range servers {
		if addresses[server.Address] {
			t.Errorf("Server %d has duplicate address: %s", i, server.Address)
		}
		addresses[server.Address] = true
		t.Logf("Server %d listening on %s", i, server.Address)
	}

	// Connect to each server and verify
	for i, server := range servers {
		certFile, keyFile, caFile := server.GetCertFiles()
		clientOpts := grpctestutil.DefaultClientOptions()
		clientOpts.CertFile = certFile
		clientOpts.KeyFile = keyFile
		clientOpts.CAFile = caFile

		client := grpctestutil.NewTestClient(t, server.Address, clientOpts)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		req := &pb.HealthCheckRequest{Tier: 1}
		resp, err := client.Client.HealthCheck(ctx, req)

		if err != nil {
			t.Errorf("Server %d health check failed: %v", i, err)
			continue
		}

		if !resp.Healthy {
			t.Errorf("Server %d: Expected healthy status, got unhealthy", i)
		}
	}

	t.Logf("All %d servers running on different ports", numServers)
}
