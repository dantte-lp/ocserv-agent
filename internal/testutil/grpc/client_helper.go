package testutil

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"testing"
	"time"

	pb "github.com/dantte-lp/ocserv-agent/pkg/proto/agent/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// TestClient wraps a gRPC client for testing
type TestClient struct {
	Conn   *grpc.ClientConn
	Client pb.AgentServiceClient
	t      *testing.T
}

// ClientOptions configures the test client
type ClientOptions struct {
	// EnableTLS enables mTLS authentication (must match server)
	EnableTLS bool
	// CertFile path to client certificate (required if EnableTLS is true)
	CertFile string
	// KeyFile path to client key (required if EnableTLS is true)
	KeyFile string
	// CAFile path to CA certificate (required if EnableTLS is true)
	CAFile string
	// Timeout for connection (default: 5 seconds)
	Timeout time.Duration
}

// DefaultClientOptions returns default client options with TLS enabled
func DefaultClientOptions() ClientOptions {
	return ClientOptions{
		EnableTLS: true,
		Timeout:   5 * time.Second,
	}
}

// NewTestClient creates a new gRPC test client
func NewTestClient(t *testing.T, address string, opts ClientOptions) *TestClient {
	t.Helper()

	// Set default timeout
	if opts.Timeout == 0 {
		opts.Timeout = 5 * time.Second
	}

	var dialOpts []grpc.DialOption

	// Configure TLS or insecure
	if opts.EnableTLS {
		// Validate required fields
		if opts.CertFile == "" || opts.KeyFile == "" || opts.CAFile == "" {
			t.Fatal("TLS enabled but certificate files not provided")
		}

		tlsCreds, err := loadTLSCredentials(opts.CertFile, opts.KeyFile, opts.CAFile)
		if err != nil {
			t.Fatalf("Failed to load TLS credentials: %v", err)
		}
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(tlsCreds))
		t.Logf("Client using mTLS authentication")
	} else {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
		t.Logf("Client using insecure connection")
	}

	// Add block option to ensure connection is ready
	dialOpts = append(dialOpts, grpc.WithBlock())

	// Create connection with timeout
	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer cancel()

	conn, err := grpc.DialContext(ctx, address, dialOpts...)
	if err != nil {
		t.Fatalf("Failed to connect to gRPC server at %s: %v", address, err)
	}

	client := pb.NewAgentServiceClient(conn)

	tc := &TestClient{
		Conn:   conn,
		Client: client,
		t:      t,
	}

	// Register cleanup
	t.Cleanup(func() {
		tc.Close()
	})

	t.Logf("Test client connected to %s", address)

	return tc
}

// Close closes the client connection
func (tc *TestClient) Close() {
	tc.t.Helper()
	if tc.Conn != nil {
		if err := tc.Conn.Close(); err != nil {
			tc.t.Logf("Error closing client connection: %v", err)
		}
		tc.t.Log("Test client connection closed")
	}
}

// loadTLSCredentials loads client TLS credentials for mTLS
func loadTLSCredentials(certFile, keyFile, caFile string) (credentials.TransportCredentials, error) {
	// Load client certificate and key
	clientCert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load client certificate: %w", err)
	}

	// Load CA certificate
	caCert, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate: %w", err)
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to append CA certificate")
	}

	// Configure TLS
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{clientCert},
		RootCAs:      certPool,
		MinVersion:   tls.VersionTLS13,
		CipherSuites: []uint16{
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
		},
	}

	return credentials.NewTLS(tlsConfig), nil
}
