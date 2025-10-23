// Package testutil provides testing utilities for gRPC integration tests
package testutil

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dantte-lp/ocserv-agent/internal/cert"
	"github.com/dantte-lp/ocserv-agent/internal/config"
	grpcserver "github.com/dantte-lp/ocserv-agent/internal/grpc"
	"github.com/rs/zerolog"
)

// TestServer represents a gRPC server instance for testing
type TestServer struct {
	Server  *grpcserver.Server
	Address string
	Config  *config.Config
	CertDir string
	t       *testing.T
	cancel  context.CancelFunc
}

// ServerOptions configures the test server
type ServerOptions struct {
	// EnableTLS enables mTLS authentication (default: true)
	EnableTLS bool
	// MockOcservSocket path to mock ocserv socket (optional)
	MockOcservSocket string
	// AgentID for the server (default: "test-agent")
	AgentID string
	// Address to listen on (default: auto-allocated)
	Address string
}

// DefaultServerOptions returns default server options with TLS enabled
func DefaultServerOptions() ServerOptions {
	return ServerOptions{
		EnableTLS: true,
		AgentID:   "test-agent",
	}
}

// NewTestServer creates and starts a new gRPC test server
func NewTestServer(t *testing.T, opts ServerOptions) *TestServer {
	t.Helper()

	// Get free port if address not specified
	address := opts.Address
	if address == "" {
		var err error
		address, err = GetFreeAddress()
		if err != nil {
			t.Fatalf("Failed to get free address: %v", err)
		}
	}

	// Create temporary directory for certificates
	certDir := t.TempDir()

	// Setup config
	cfg := &config.Config{
		AgentID: opts.AgentID,
		TLS: config.TLSConfig{
			Enabled: opts.EnableTLS,
		},
		Ocserv: config.OcservConfig{
			ConfigPath:     "/etc/ocserv/ocserv.conf", // Mock path
			SystemdService: "ocserv",
		},
	}

	// Setup mock ocserv socket if provided
	if opts.MockOcservSocket != "" {
		cfg.Ocserv.CtlSocket = opts.MockOcservSocket
	} else {
		// Default to a temporary socket path
		cfg.Ocserv.CtlSocket = filepath.Join(certDir, "occtl-test.socket")
	}

	// Generate certificates if TLS is enabled
	if opts.EnableTLS {
		certInfo, err := cert.GenerateSelfSignedCerts(certDir, "localhost")
		if err != nil {
			t.Fatalf("Failed to generate test certificates: %v", err)
		}

		cfg.TLS.CertFile = filepath.Join(certDir, "agent.crt")
		cfg.TLS.KeyFile = filepath.Join(certDir, "agent.key")
		cfg.TLS.CAFile = filepath.Join(certDir, "ca.crt")
		cfg.TLS.MinVersion = "TLS1.3"

		t.Logf("Generated test certificates in %s", certDir)
		t.Logf("  CA fingerprint: %s", certInfo.CAFingerprint)
		t.Logf("  Cert fingerprint: %s", certInfo.CertFingerprint)

		// Verify files exist
		for _, file := range []string{cfg.TLS.CertFile, cfg.TLS.KeyFile, cfg.TLS.CAFile} {
			if _, err := os.Stat(file); os.IsNotExist(err) {
				t.Fatalf("Expected certificate file not created: %s", file)
			}
		}
	}

	// Create logger
	logger := zerolog.New(zerolog.NewTestWriter(t)).With().
		Timestamp().
		Str("component", "test-server").
		Logger()

	// Create gRPC server
	server, err := grpcserver.New(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create gRPC server: %v", err)
	}

	ts := &TestServer{
		Server:  server,
		Address: address,
		Config:  cfg,
		CertDir: certDir,
		t:       t,
	}

	// Start server in background
	_, cancel := context.WithCancel(context.Background())
	ts.cancel = cancel

	errChan := make(chan error, 1)
	go func() {
		err := server.Serve(address)
		if err != nil {
			errChan <- err
		}
	}()

	// Wait for server to start (with timeout)
	started := make(chan bool, 1)
	go func() {
		// Try to connect to verify server is ready
		for i := 0; i < 50; i++ { // 5 seconds max (50 * 100ms)
			time.Sleep(100 * time.Millisecond)
			// Just check if we can get an error from errChan
			select {
			case err := <-errChan:
				t.Errorf("Server failed to start: %v", err)
				return
			default:
			}
			// If we get here without error, server is likely started
			if i > 5 { // Give it at least 500ms
				started <- true
				return
			}
		}
		started <- false
	}()

	select {
	case <-started:
		t.Logf("Test gRPC server started on %s (TLS: %v)", address, opts.EnableTLS)
	case <-time.After(10 * time.Second):
		ts.Shutdown()
		t.Fatal("Timeout waiting for server to start")
	}

	// Register cleanup
	t.Cleanup(func() {
		ts.Shutdown()
	})

	return ts
}

// Shutdown gracefully stops the test server
func (ts *TestServer) Shutdown() {
	ts.t.Helper()
	if ts.cancel != nil {
		ts.cancel()
	}
	if ts.Server != nil {
		ts.Server.GracefulStop()
		ts.t.Log("Test server stopped")
	}
}

// GetCertFiles returns paths to the certificate files (for client creation)
func (ts *TestServer) GetCertFiles() (certFile, keyFile, caFile string) {
	return ts.Config.TLS.CertFile, ts.Config.TLS.KeyFile, ts.Config.TLS.CAFile
}
