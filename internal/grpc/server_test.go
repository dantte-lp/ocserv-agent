package grpc

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/dantte-lp/ocserv-agent/internal/cert"
	"github.com/dantte-lp/ocserv-agent/internal/config"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
)

// TestNew tests the New function
func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *config.Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config without TLS",
			cfg: &config.Config{
				AgentID: "test-agent",
				TLS: config.TLSConfig{
					Enabled: false,
				},
				Ocserv: config.OcservConfig{
					ConfigPath:     "/etc/ocserv/ocserv.conf",
					CtlSocket:      "/run/ocserv/occtl.socket",
					SystemdService: "ocserv",
				},
			},
			wantErr: false,
		},
		{
			name: "valid config with TLS but missing certs",
			cfg: &config.Config{
				AgentID: "test-agent",
				TLS: config.TLSConfig{
					Enabled:    true,
					CertFile:   "/nonexistent/cert.pem",
					KeyFile:    "/nonexistent/key.pem",
					CAFile:     "/nonexistent/ca.pem",
					MinVersion: "TLS1.3",
				},
				Ocserv: config.OcservConfig{
					ConfigPath:     "/etc/ocserv/ocserv.conf",
					CtlSocket:      "/run/ocserv/occtl.socket",
					SystemdService: "ocserv",
				},
			},
			wantErr: true,
			errMsg:  "failed to create gRPC server",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zerolog.New(zerolog.NewTestWriter(t))

			server, err := New(tt.cfg, logger)

			if tt.wantErr {
				if err == nil {
					t.Errorf("New() expected error containing %q, got nil", tt.errMsg)
					return
				}
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("New() error = %v, want error containing %q", err, tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("New() unexpected error = %v", err)
				return
			}

			if server == nil {
				t.Error("New() returned nil server without error")
				return
			}

			// Verify server fields
			if server.config != tt.cfg {
				t.Error("New() server.config not set correctly")
			}

			if server.server == nil {
				t.Error("New() server.server (gRPC server) not initialized")
			}

			if server.ocservManager == nil {
				t.Error("New() server.ocservManager not initialized")
			}
		})
	}
}

// TestNewWithTLS tests server creation with TLS credentials
func TestNewWithTLS(t *testing.T) {
	t.Run("valid TLS configuration", func(t *testing.T) {
		certFile, keyFile, caFile, cleanup := createTestCerts(t)
		defer cleanup()

		cfg := &config.Config{
			AgentID: "test-agent",
			TLS: config.TLSConfig{
				Enabled:    true,
				CertFile:   certFile,
				KeyFile:    keyFile,
				CAFile:     caFile,
				MinVersion: "TLS1.3",
			},
			Ocserv: config.OcservConfig{
				ConfigPath:     "/etc/ocserv/ocserv.conf",
				CtlSocket:      "/run/ocserv/occtl.socket",
				SystemdService: "ocserv",
			},
		}

		logger := zerolog.New(zerolog.NewTestWriter(t))

		server, err := New(cfg, logger)
		if err != nil {
			t.Fatalf("New() with TLS failed: %v", err)
		}

		if server == nil {
			t.Fatal("New() returned nil server")
		}

		if server.server == nil {
			t.Error("New() gRPC server not initialized with TLS")
		}

		if server.config != cfg {
			t.Error("New() server.config not set correctly")
		}
	})

	t.Run("invalid CA file", func(t *testing.T) {
		certFile, keyFile, _, cleanup := createTestCerts(t)
		defer cleanup()

		cfg := &config.Config{
			AgentID: "test-agent",
			TLS: config.TLSConfig{
				Enabled:    true,
				CertFile:   certFile,
				KeyFile:    keyFile,
				CAFile:     "/nonexistent/ca.crt",
				MinVersion: "TLS1.3",
			},
			Ocserv: config.OcservConfig{
				ConfigPath:     "/etc/ocserv/ocserv.conf",
				CtlSocket:      "/run/ocserv/occtl.socket",
				SystemdService: "ocserv",
			},
		}

		logger := zerolog.New(zerolog.NewTestWriter(t))

		_, err := New(cfg, logger)
		if err == nil {
			t.Error("New() expected error for invalid CA file, got nil")
		}
	})

	t.Run("invalid cert/key pair", func(t *testing.T) {
		_, _, caFile, cleanup := createTestCerts(t)
		defer cleanup()

		cfg := &config.Config{
			AgentID: "test-agent",
			TLS: config.TLSConfig{
				Enabled:    true,
				CertFile:   "/nonexistent/cert.crt",
				KeyFile:    "/nonexistent/key.key",
				CAFile:     caFile,
				MinVersion: "TLS1.3",
			},
			Ocserv: config.OcservConfig{
				ConfigPath:     "/etc/ocserv/ocserv.conf",
				CtlSocket:      "/run/ocserv/occtl.socket",
				SystemdService: "ocserv",
			},
		}

		logger := zerolog.New(zerolog.NewTestWriter(t))

		_, err := New(cfg, logger)
		if err == nil {
			t.Error("New() expected error for invalid cert/key, got nil")
		}
	})

	t.Run("invalid CA certificate format", func(t *testing.T) {
		certFile, keyFile, _, cleanup := createTestCerts(t)
		defer cleanup()

		// Create a file with invalid PEM content
		tempDir := t.TempDir()
		invalidCAFile := filepath.Join(tempDir, "invalid_ca.crt")
		if err := os.WriteFile(invalidCAFile, []byte("invalid PEM content"), 0600); err != nil {
			t.Fatalf("Failed to create invalid CA file: %v", err)
		}

		cfg := &config.Config{
			AgentID: "test-agent",
			TLS: config.TLSConfig{
				Enabled:    true,
				CertFile:   certFile,
				KeyFile:    keyFile,
				CAFile:     invalidCAFile,
				MinVersion: "TLS1.3",
			},
			Ocserv: config.OcservConfig{
				ConfigPath:     "/etc/ocserv/ocserv.conf",
				CtlSocket:      "/run/ocserv/occtl.socket",
				SystemdService: "ocserv",
			},
		}

		logger := zerolog.New(zerolog.NewTestWriter(t))

		_, err := New(cfg, logger)
		if err == nil {
			t.Error("New() expected error for invalid CA format, got nil")
		}
		if err != nil && !contains(err.Error(), "failed to append CA certificate") {
			t.Errorf("New() error = %v, want error containing 'failed to append CA certificate'", err)
		}
	})
}

// TestGetTLSVersion tests TLS version parsing
func TestGetTLSVersion(t *testing.T) {
	tests := []struct {
		name       string
		minVersion string
		want       uint16
	}{
		{
			name:       "TLS 1.3",
			minVersion: "TLS1.3",
			want:       0x0304, // tls.VersionTLS13
		},
		{
			name:       "TLS 1.2",
			minVersion: "TLS1.2",
			want:       0x0303, // tls.VersionTLS12
		},
		{
			name:       "default to TLS 1.3",
			minVersion: "invalid",
			want:       0x0304, // tls.VersionTLS13
		},
		{
			name:       "empty defaults to TLS 1.3",
			minVersion: "",
			want:       0x0304, // tls.VersionTLS13
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				TLS: config.TLSConfig{
					MinVersion: tt.minVersion,
				},
			}

			logger := zerolog.New(zerolog.NewTestWriter(t))

			s := &Server{
				config: cfg,
				logger: logger,
			}

			got := s.getTLSVersion()
			if got != tt.want {
				t.Errorf("getTLSVersion() = %x, want %x", got, tt.want)
			}
		})
	}
}

// TestLoggingInterceptor tests the logging interceptor
func TestLoggingInterceptor(t *testing.T) {
	cfg := &config.Config{
		AgentID: "test-agent",
		TLS: config.TLSConfig{
			Enabled: false,
		},
		Ocserv: config.OcservConfig{
			ConfigPath:     "/etc/ocserv/ocserv.conf",
			CtlSocket:      "/run/ocserv/occtl.socket",
			SystemdService: "ocserv",
		},
	}

	logger := zerolog.New(zerolog.NewTestWriter(t))

	s := &Server{
		config: cfg,
		logger: logger,
	}

	interceptor := s.loggingInterceptor()

	ctx := context.Background()
	req := struct{}{}
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/TestMethod",
	}

	t.Run("successful call", func(t *testing.T) {
		handlerCalled := false
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			handlerCalled = true
			return "response", nil
		}

		resp, err := interceptor(ctx, req, info, handler)

		if err != nil {
			t.Errorf("loggingInterceptor() unexpected error = %v", err)
		}

		if !handlerCalled {
			t.Error("loggingInterceptor() did not call handler")
		}

		if resp != "response" {
			t.Errorf("loggingInterceptor() resp = %v, want %v", resp, "response")
		}
	})

	t.Run("failed call", func(t *testing.T) {
		handlerCalled := false
		testErr := fmt.Errorf("test error")
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			handlerCalled = true
			return nil, testErr
		}

		resp, err := interceptor(ctx, req, info, handler)

		if err != testErr {
			t.Errorf("loggingInterceptor() error = %v, want %v", err, testErr)
		}

		if !handlerCalled {
			t.Error("loggingInterceptor() did not call handler")
		}

		if resp != nil {
			t.Errorf("loggingInterceptor() resp = %v, want nil", resp)
		}
	})
}

// TestRecoveryInterceptor tests panic recovery
func TestRecoveryInterceptor(t *testing.T) {
	cfg := &config.Config{
		AgentID: "test-agent",
		TLS: config.TLSConfig{
			Enabled: false,
		},
		Ocserv: config.OcservConfig{
			ConfigPath:     "/etc/ocserv/ocserv.conf",
			CtlSocket:      "/run/ocserv/occtl.socket",
			SystemdService: "ocserv",
		},
	}

	logger := zerolog.New(zerolog.NewTestWriter(t))

	s := &Server{
		config: cfg,
		logger: logger,
	}

	interceptor := s.recoveryInterceptor()

	ctx := context.Background()
	req := struct{}{}
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/TestMethod",
	}

	// Test panic recovery
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		panic("test panic")
	}

	resp, err := interceptor(ctx, req, info, handler)

	if err == nil {
		t.Error("recoveryInterceptor() expected error after panic, got nil")
	}

	if resp != nil {
		t.Errorf("recoveryInterceptor() resp = %v, want nil after panic", resp)
	}

	if err.Error() != "internal server error" {
		t.Errorf("recoveryInterceptor() error = %v, want 'internal server error'", err)
	}
}

// TestGracefulStopAndStop tests server shutdown methods
func TestGracefulStopAndStop(t *testing.T) {
	cfg := &config.Config{
		AgentID: "test-agent",
		TLS: config.TLSConfig{
			Enabled: false,
		},
		Ocserv: config.OcservConfig{
			ConfigPath:     "/etc/ocserv/ocserv.conf",
			CtlSocket:      "/run/ocserv/occtl.socket",
			SystemdService: "ocserv",
		},
	}

	logger := zerolog.New(zerolog.NewTestWriter(t))

	server, err := New(cfg, logger)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	// Test GracefulStop (should not panic)
	server.GracefulStop()

	// Create new server for Stop test
	server2, err := New(cfg, logger)
	if err != nil {
		t.Fatalf("New() failed for second server: %v", err)
	}

	// Test Stop (should not panic)
	server2.Stop()
}

// mockServerStream is a mock implementation of grpc.ServerStream for testing
type mockServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (m *mockServerStream) Context() context.Context {
	if m.ctx != nil {
		return m.ctx
	}
	return context.Background()
}

// TestStreamLoggingInterceptor tests the stream logging interceptor
func TestStreamLoggingInterceptor(t *testing.T) {
	cfg := &config.Config{
		AgentID: "test-agent",
		TLS: config.TLSConfig{
			Enabled: false,
		},
		Ocserv: config.OcservConfig{
			ConfigPath:     "/etc/ocserv/ocserv.conf",
			CtlSocket:      "/run/ocserv/occtl.socket",
			SystemdService: "ocserv",
		},
	}

	logger := zerolog.New(zerolog.NewTestWriter(t))

	s := &Server{
		config: cfg,
		logger: logger,
	}

	interceptor := s.streamLoggingInterceptor()

	srv := struct{}{}
	ss := &mockServerStream{ctx: context.Background()}
	info := &grpc.StreamServerInfo{
		FullMethod:     "/test.Service/TestStream",
		IsClientStream: true,
		IsServerStream: true,
	}

	t.Run("successful stream", func(t *testing.T) {
		handlerCalled := false
		handler := func(srv interface{}, stream grpc.ServerStream) error {
			handlerCalled = true
			return nil
		}

		err := interceptor(srv, ss, info, handler)

		if err != nil {
			t.Errorf("streamLoggingInterceptor() unexpected error = %v", err)
		}

		if !handlerCalled {
			t.Error("streamLoggingInterceptor() did not call handler")
		}
	})

	t.Run("failed stream", func(t *testing.T) {
		handlerCalled := false
		testErr := fmt.Errorf("stream error")
		handler := func(srv interface{}, stream grpc.ServerStream) error {
			handlerCalled = true
			return testErr
		}

		err := interceptor(srv, ss, info, handler)

		if err != testErr {
			t.Errorf("streamLoggingInterceptor() error = %v, want %v", err, testErr)
		}

		if !handlerCalled {
			t.Error("streamLoggingInterceptor() did not call handler")
		}
	})

	t.Run("server stream only", func(t *testing.T) {
		serverStreamInfo := &grpc.StreamServerInfo{
			FullMethod:     "/test.Service/ServerStream",
			IsClientStream: false,
			IsServerStream: true,
		}

		handler := func(srv interface{}, stream grpc.ServerStream) error {
			return nil
		}

		err := interceptor(srv, ss, serverStreamInfo, handler)

		if err != nil {
			t.Errorf("streamLoggingInterceptor() unexpected error = %v", err)
		}
	})

	t.Run("client stream only", func(t *testing.T) {
		clientStreamInfo := &grpc.StreamServerInfo{
			FullMethod:     "/test.Service/ClientStream",
			IsClientStream: true,
			IsServerStream: false,
		}

		handler := func(srv interface{}, stream grpc.ServerStream) error {
			return nil
		}

		err := interceptor(srv, ss, clientStreamInfo, handler)

		if err != nil {
			t.Errorf("streamLoggingInterceptor() unexpected error = %v", err)
		}
	})
}

// createTestCerts generates test certificates for TLS testing
// Returns paths to cert, key, and CA files in a temporary directory
func createTestCerts(t *testing.T) (certFile, keyFile, caFile string, cleanup func()) {
	t.Helper()

	// Create temporary directory for test certificates
	tempDir := t.TempDir()

	// Generate self-signed certificates using internal/cert
	certInfo, err := cert.GenerateSelfSignedCerts(tempDir, "localhost")
	if err != nil {
		t.Fatalf("Failed to generate test certificates: %v", err)
	}

	certFile = filepath.Join(tempDir, "agent.crt")
	keyFile = filepath.Join(tempDir, "agent.key")
	caFile = filepath.Join(tempDir, "ca.crt")

	// Verify files exist
	for _, file := range []string{certFile, keyFile, caFile} {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			t.Fatalf("Expected certificate file not created: %s", file)
		}
	}

	// Cleanup function (t.TempDir() handles cleanup automatically, but we provide this for consistency)
	cleanup = func() {
		// t.TempDir() automatically cleans up, so this is a no-op
	}

	t.Logf("Test certificates created in %s", tempDir)
	t.Logf("  CA cert: %s (fingerprint: %s)", caFile, certInfo.CAFingerprint)
	t.Logf("  Agent cert: %s (fingerprint: %s)", certFile, certInfo.CertFingerprint)
	t.Logf("  Agent key: %s", keyFile)

	return certFile, keyFile, caFile, cleanup
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
