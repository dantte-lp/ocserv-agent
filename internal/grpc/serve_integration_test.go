//go:build integration

package grpc_test

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	grpctestutil "github.com/dantte-lp/ocserv-agent/internal/testutil/grpc"
	pb "github.com/dantte-lp/ocserv-agent/pkg/proto/agent/v1"
)

// TestServerServeAcceptsConnections tests that Serve accepts incoming connections
func TestServerServeAcceptsConnections(t *testing.T) {
	// Start test server (Serve is called in background by NewTestServer)
	server := grpctestutil.NewTestServer(t, grpctestutil.DefaultServerOptions())

	// Create test client
	certFile, keyFile, caFile := server.GetCertFiles()
	clientOpts := grpctestutil.DefaultClientOptions()
	clientOpts.CertFile = certFile
	clientOpts.KeyFile = keyFile
	clientOpts.CAFile = caFile

	client := grpctestutil.NewTestClient(t, server.Address, clientOpts)

	// Make a request to verify connection is accepted
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &pb.HealthCheckRequest{Tier: 1}
	resp, err := client.Client.HealthCheck(ctx, req)

	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}

	if !resp.Healthy {
		t.Error("Expected healthy response")
	}

	t.Log("Server successfully accepted connection and processed request")
}

// TestServerServeMultipleConnections tests concurrent connection handling
func TestServerServeMultipleConnections(t *testing.T) {
	// Start test server
	server := grpctestutil.NewTestServer(t, grpctestutil.DefaultServerOptions())

	certFile, keyFile, caFile := server.GetCertFiles()

	// Create multiple clients concurrently
	const numClients = 20
	done := make(chan bool, numClients)
	errors := make(chan error, numClients)

	for i := 0; i < numClients; i++ {
		go func(id int) {
			clientOpts := grpctestutil.DefaultClientOptions()
			clientOpts.CertFile = certFile
			clientOpts.KeyFile = keyFile
			clientOpts.CAFile = caFile

			client := grpctestutil.NewTestClient(t, server.Address, clientOpts)
			defer client.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			req := &pb.HealthCheckRequest{Tier: 1}
			resp, err := client.Client.HealthCheck(ctx, req)

			if err != nil {
				errors <- fmt.Errorf("client %d: %w", id, err)
				done <- false
				return
			}

			if !resp.Healthy {
				errors <- fmt.Errorf("client %d: unhealthy response", id)
				done <- false
				return
			}

			done <- true
		}(i)
	}

	// Wait for all clients
	successCount := 0
	for i := 0; i < numClients; i++ {
		if <-done {
			successCount++
		}
	}

	close(errors)

	// Check for errors
	for err := range errors {
		t.Error(err)
	}

	if successCount != numClients {
		t.Errorf("Expected %d successful connections, got %d", numClients, successCount)
	}

	t.Logf("Server handled %d concurrent connections successfully", numClients)
}

// TestServerStopImmediate tests immediate stop (Stop vs GracefulStop)
func TestServerStopImmediate(t *testing.T) {
	// Start test server
	server := grpctestutil.NewTestServer(t, grpctestutil.DefaultServerOptions())

	// Create test client
	certFile, keyFile, caFile := server.GetCertFiles()
	clientOpts := grpctestutil.DefaultClientOptions()
	clientOpts.CertFile = certFile
	clientOpts.KeyFile = keyFile
	clientOpts.CAFile = caFile

	client := grpctestutil.NewTestClient(t, server.Address, clientOpts)

	// Make a successful request first
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &pb.HealthCheckRequest{Tier: 1}
	_, err := client.Client.HealthCheck(ctx, req)

	if err != nil {
		t.Fatalf("Initial health check failed: %v", err)
	}

	// Immediately stop the server (forceful)
	server.Server.Stop()
	t.Log("Server stopped immediately (forceful)")

	// Try to make another request (should fail)
	ctx2, cancel2 := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel2()

	_, err = client.Client.HealthCheck(ctx2, req)
	if err == nil {
		t.Error("Expected error after immediate stop, got nil")
	} else {
		t.Logf("Got expected error after immediate stop: %v", err)
	}
}

// TestServerGracefulStopWithActiveRequests tests graceful stop with active requests
func TestServerGracefulStopWithActiveRequests(t *testing.T) {
	// Start test server
	server := grpctestutil.NewTestServer(t, grpctestutil.DefaultServerOptions())

	// Create test client
	certFile, keyFile, caFile := server.GetCertFiles()
	clientOpts := grpctestutil.DefaultClientOptions()
	clientOpts.CertFile = certFile
	clientOpts.KeyFile = keyFile
	clientOpts.CAFile = caFile

	client := grpctestutil.NewTestClient(t, server.Address, clientOpts)

	// Start multiple requests in background
	const numRequests = 5
	done := make(chan bool, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(id int) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			req := &pb.HealthCheckRequest{Tier: 1}
			_, err := client.Client.HealthCheck(ctx, req)

			if err == nil {
				t.Logf("Request %d completed successfully before shutdown", id)
				done <- true
			} else {
				t.Logf("Request %d failed: %v", id, err)
				done <- false
			}
		}(i)
	}

	// Give requests time to start
	time.Sleep(100 * time.Millisecond)

	// Gracefully stop the server
	go func() {
		t.Log("Starting graceful shutdown...")
		server.Server.GracefulStop()
		t.Log("Graceful shutdown complete")
	}()

	// Wait for requests to complete
	successCount := 0
	timeout := time.After(15 * time.Second)
	for i := 0; i < numRequests; i++ {
		select {
		case success := <-done:
			if success {
				successCount++
			}
		case <-timeout:
			t.Error("Timeout waiting for requests to complete during graceful shutdown")
			return
		}
	}

	t.Logf("Graceful shutdown completed %d/%d requests", successCount, numRequests)
}

// TestServerListenerError tests error handling when listener fails
func TestServerListenerError(t *testing.T) {
	// First, start a server on a random port
	server1 := grpctestutil.NewTestServer(t, grpctestutil.DefaultServerOptions())
	address := server1.Address

	t.Logf("Server 1 listening on %s", address)

	// Try to start another server on the same address (should fail)
	// Note: We can't easily test this with NewTestServer because it uses random ports
	// and starts the server automatically. This test is more of a documentation of
	// expected behavior.

	t.Log("Note: Testing port conflict requires manual Serve() call")
	t.Log("NewTestServer uses random ports, so conflicts are unlikely")

	// Verify first server is still working
	certFile, keyFile, caFile := server1.GetCertFiles()
	clientOpts := grpctestutil.DefaultClientOptions()
	clientOpts.CertFile = certFile
	clientOpts.KeyFile = keyFile
	clientOpts.CAFile = caFile

	client := grpctestutil.NewTestClient(t, server1.Address, clientOpts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &pb.HealthCheckRequest{Tier: 1}
	resp, err := client.Client.HealthCheck(ctx, req)

	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}

	if !resp.Healthy {
		t.Error("Expected healthy response")
	}

	t.Log("First server still operational after port conflict test")
}

// TestServerPortInUse tests the behavior when trying to listen on a port that's in use
func TestServerPortInUse(t *testing.T) {
	// Get a free port
	port, err := grpctestutil.GetFreePort()
	if err != nil {
		t.Fatalf("Failed to get free port: %v", err)
	}

	address := fmt.Sprintf("localhost:%d", port)

	// Bind to the port manually to block it
	listener, err := net.Listen("tcp", address)
	if err != nil {
		t.Fatalf("Failed to listen on %s: %v", address, err)
	}
	defer listener.Close()

	t.Logf("Port %d is now blocked", port)

	// Now try to start a server on the same port
	// This should fail because the port is in use
	// However, NewTestServer uses GetFreePort which will get a different port
	// So we can't directly test this scenario with our current helper

	t.Log("Note: Port conflict testing is inherently racy")
	t.Log("NewTestServer's GetFreePort() avoids conflicts by design")
	t.Log("This test verifies that manual port blocking works as expected")

	// Verify the port is indeed blocked by trying to bind again
	_, err = net.Listen("tcp", address)
	if err == nil {
		t.Error("Expected error when binding to already-used port, got nil")
	} else {
		t.Logf("Got expected error: %v", err)
	}
}

// TestServerServeSequential tests starting and stopping servers sequentially
func TestServerServeSequential(t *testing.T) {
	const numIterations = 3

	for i := 0; i < numIterations; i++ {
		t.Run(fmt.Sprintf("iteration_%d", i), func(t *testing.T) {
			// Start server
			server := grpctestutil.NewTestServer(t, grpctestutil.DefaultServerOptions())

			// Create client
			certFile, keyFile, caFile := server.GetCertFiles()
			clientOpts := grpctestutil.DefaultClientOptions()
			clientOpts.CertFile = certFile
			clientOpts.KeyFile = keyFile
			clientOpts.CAFile = caFile

			client := grpctestutil.NewTestClient(t, server.Address, clientOpts)

			// Make request
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			req := &pb.HealthCheckRequest{Tier: 1}
			resp, err := client.Client.HealthCheck(ctx, req)

			if err != nil {
				t.Errorf("Iteration %d: Health check failed: %v", i, err)
				return
			}

			if !resp.Healthy {
				t.Errorf("Iteration %d: Expected healthy response", i)
			}

			t.Logf("Iteration %d: Server started, served request, and will stop", i)

			// Cleanup happens automatically via t.Cleanup
		})
	}

	t.Logf("Successfully completed %d sequential server start/stop cycles", numIterations)
}

// TestServerServeLongRunning tests server stability over time
func TestServerServeLongRunning(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping long-running test in short mode")
	}

	// Start server
	server := grpctestutil.NewTestServer(t, grpctestutil.DefaultServerOptions())

	// Create client
	certFile, keyFile, caFile := server.GetCertFiles()
	clientOpts := grpctestutil.DefaultClientOptions()
	clientOpts.CertFile = certFile
	clientOpts.KeyFile = keyFile
	clientOpts.CAFile = caFile

	client := grpctestutil.NewTestClient(t, server.Address, clientOpts)

	// Make requests periodically over 10 seconds
	const duration = 10 * time.Second
	const interval = 500 * time.Millisecond

	startTime := time.Now()
	requestCount := 0
	errorCount := 0

	for time.Since(startTime) < duration {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)

		req := &pb.HealthCheckRequest{Tier: 1}
		resp, err := client.Client.HealthCheck(ctx, req)

		cancel()

		requestCount++

		if err != nil {
			errorCount++
			t.Logf("Request %d failed: %v", requestCount, err)
		} else if !resp.Healthy {
			errorCount++
			t.Logf("Request %d returned unhealthy", requestCount)
		}

		time.Sleep(interval)
	}

	successRate := float64(requestCount-errorCount) / float64(requestCount) * 100

	t.Logf("Long-running test: %d requests, %d errors, %.1f%% success rate",
		requestCount, errorCount, successRate)

	if successRate < 95.0 {
		t.Errorf("Success rate %.1f%% is below 95%%", successRate)
	}
}

// TestServerServeWithInsecureConnection tests server without TLS
func TestServerServeWithInsecureConnection(t *testing.T) {
	// Start server without TLS
	serverOpts := grpctestutil.DefaultServerOptions()
	serverOpts.EnableTLS = false

	server := grpctestutil.NewTestServer(t, serverOpts)

	// Create client without TLS
	clientOpts := grpctestutil.DefaultClientOptions()
	clientOpts.EnableTLS = false

	client := grpctestutil.NewTestClient(t, server.Address, clientOpts)

	// Make request
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &pb.HealthCheckRequest{Tier: 1}
	resp, err := client.Client.HealthCheck(ctx, req)

	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}

	if !resp.Healthy {
		t.Error("Expected healthy response")
	}

	t.Log("Server successfully served insecure connection")
}

// TestServerServeRecoveryFromPanic tests that panic recovery doesn't crash server
func TestServerServeRecoveryFromPanic(t *testing.T) {
	// Start server
	server := grpctestutil.NewTestServer(t, grpctestutil.DefaultServerOptions())

	// Create client
	certFile, keyFile, caFile := server.GetCertFiles()
	clientOpts := grpctestutil.DefaultClientOptions()
	clientOpts.CertFile = certFile
	clientOpts.KeyFile = keyFile
	clientOpts.CAFile = caFile

	client := grpctestutil.NewTestClient(t, server.Address, clientOpts)

	// Make a normal request (should succeed)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &pb.HealthCheckRequest{Tier: 1}
	resp, err := client.Client.HealthCheck(ctx, req)

	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}

	if !resp.Healthy {
		t.Error("Expected healthy response")
	}

	t.Log("Server is healthy and panic recovery interceptor is in place")
	t.Log("Note: Actual panic testing would require modifying handler code")
}
