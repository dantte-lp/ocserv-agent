package grpc

import (
	"context"
	"strings"
	"testing"

	"github.com/dantte-lp/ocserv-agent/internal/config"
	pb "github.com/dantte-lp/ocserv-agent/pkg/proto/agent/v1"
	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestHealthCheck tests the HealthCheck RPC handler
func TestHealthCheck(t *testing.T) {
	tests := []struct {
		name           string
		tier           int32
		wantErr        bool
		wantErrCode    codes.Code
		expectedChecks []string // Keys to verify exist in response
	}{
		{
			name:           "tier 1 basic heartbeat",
			tier:           1,
			wantErr:        false,
			expectedChecks: []string{"agent", "config", "uptime"},
		},
		{
			name:           "tier 2 deep check",
			tier:           2,
			wantErr:        false,
			expectedChecks: []string{"agent", "memory", "cpu", "ocserv_process", "ocserv_socket"},
		},
		{
			name:           "tier 3 application check",
			tier:           3,
			wantErr:        false,
			expectedChecks: []string{"agent", "memory", "cpu", "ocserv_process", "occtl", "config_dirs"},
		},
		{
			name:        "invalid tier - too low",
			tier:        0,
			wantErr:     true,
			wantErrCode: codes.InvalidArgument,
		},
		{
			name:        "invalid tier - too high",
			tier:        4,
			wantErr:     true,
			wantErrCode: codes.InvalidArgument,
		},
		{
			name:        "invalid tier - negative",
			tier:        -1,
			wantErr:     true,
			wantErrCode: codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			req := &pb.HealthCheckRequest{
				Tier: tt.tier,
			}

			resp, err := server.HealthCheck(context.Background(), req)

			if tt.wantErr {
				if err == nil {
					t.Errorf("HealthCheck() expected error with code %v, got nil", tt.wantErrCode)
					return
				}

				st, ok := status.FromError(err)
				if !ok {
					t.Errorf("HealthCheck() error is not a gRPC status error: %v", err)
					return
				}

				if st.Code() != tt.wantErrCode {
					t.Errorf("HealthCheck() error code = %v, want %v", st.Code(), tt.wantErrCode)
				}
				return
			}

			if err != nil {
				t.Errorf("HealthCheck() unexpected error = %v", err)
				return
			}

			if resp == nil {
				t.Error("HealthCheck() returned nil response without error")
				return
			}

			// Healthy status depends on real system state (ocserv running, etc.)
			// We don't assert on Healthy value, just verify response is valid

			if resp.StatusMessage == "" {
				t.Error("HealthCheck() StatusMessage is empty")
			}

			if resp.Timestamp == nil {
				t.Error("HealthCheck() Timestamp is nil")
			}

			if len(resp.Checks) == 0 {
				t.Error("HealthCheck() Checks map is empty")
			}

			// Verify all expected checks are present
			for _, check := range tt.expectedChecks {
				if _, ok := resp.Checks[check]; !ok {
					t.Errorf("HealthCheck() expected check %q not found in response", check)
				}
			}
		})
	}
}

// TestExecuteCommand tests the ExecuteCommand RPC handler
func TestExecuteCommand(t *testing.T) {
	t.Run("command not allowed", func(t *testing.T) {
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
			Security: config.SecurityConfig{
				AllowedCommands:   []string{"systemctl"}, // Only systemctl allowed
				MaxCommandTimeout: 30,
			},
		}

		logger := zerolog.New(zerolog.NewTestWriter(t))

		server, err := New(cfg, logger)
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}

		req := &pb.CommandRequest{
			RequestId:   "test-req-1",
			CommandType: "occtl", // Not allowed
			Args:        []string{"show", "users"},
		}

		resp, err := server.ExecuteCommand(context.Background(), req)

		// ExecuteCommand should not return gRPC errors, it wraps them in response
		if err != nil {
			t.Errorf("ExecuteCommand() returned gRPC error (should wrap in response): %v", err)
		}

		if resp == nil {
			t.Fatal("ExecuteCommand() returned nil response")
		}

		if resp.RequestId != req.RequestId {
			t.Errorf("ExecuteCommand() RequestId = %v, want %v", resp.RequestId, req.RequestId)
		}

		if resp.Success {
			t.Error("ExecuteCommand() Success = true, want false for disallowed command")
		}

		if !strings.Contains(resp.ErrorMessage, "command not allowed") {
			t.Errorf("ExecuteCommand() ErrorMessage = %q, want containing 'command not allowed'", resp.ErrorMessage)
		}
	})

	t.Run("invalid arguments - injection attempt", func(t *testing.T) {
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
			Security: config.SecurityConfig{
				AllowedCommands:   []string{"systemctl"},
				MaxCommandTimeout: 30,
			},
		}

		logger := zerolog.New(zerolog.NewTestWriter(t))

		server, err := New(cfg, logger)
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}

		req := &pb.CommandRequest{
			RequestId:   "test-req-2",
			CommandType: "systemctl",
			Args:        []string{"start; rm -rf /"}, // Injection attempt
		}

		resp, err := server.ExecuteCommand(context.Background(), req)

		if err != nil {
			t.Errorf("ExecuteCommand() returned gRPC error: %v", err)
		}

		if resp == nil {
			t.Fatal("ExecuteCommand() returned nil response")
		}

		if resp.Success {
			t.Error("ExecuteCommand() Success = true, want false for invalid arguments")
		}

		if !strings.Contains(resp.ErrorMessage, "invalid arguments") {
			t.Errorf("ExecuteCommand() ErrorMessage = %q, want containing 'invalid arguments'", resp.ErrorMessage)
		}
	})

	t.Run("backtick injection blocked", func(t *testing.T) {
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
			Security: config.SecurityConfig{
				AllowedCommands:   []string{"occtl"},
				MaxCommandTimeout: 30,
			},
		}

		logger := zerolog.New(zerolog.NewTestWriter(t))

		server, err := New(cfg, logger)
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}

		req := &pb.CommandRequest{
			RequestId:   "test-req-3",
			CommandType: "occtl",
			Args:        []string{"show", "users`whoami`"}, // Backtick injection
		}

		resp, err := server.ExecuteCommand(context.Background(), req)

		if err != nil {
			t.Errorf("ExecuteCommand() returned gRPC error: %v", err)
		}

		if resp == nil {
			t.Fatal("ExecuteCommand() returned nil response")
		}

		if resp.Success {
			t.Error("ExecuteCommand() Success = true, want false for backtick injection")
		}

		if !strings.Contains(resp.ErrorMessage, "dangerous characters") {
			t.Errorf("ExecuteCommand() ErrorMessage = %q, want containing 'dangerous characters'", resp.ErrorMessage)
		}
	})

	t.Run("request ID propagation", func(t *testing.T) {
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
			Security: config.SecurityConfig{
				AllowedCommands:   []string{"systemctl"},
				MaxCommandTimeout: 30,
			},
		}

		logger := zerolog.New(zerolog.NewTestWriter(t))

		server, err := New(cfg, logger)
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}

		testRequestID := "unique-test-id-12345"
		req := &pb.CommandRequest{
			RequestId:   testRequestID,
			CommandType: "systemctl",
			Args:        []string{}, // Empty args will fail
		}

		resp, err := server.ExecuteCommand(context.Background(), req)

		if err != nil {
			t.Errorf("ExecuteCommand() returned gRPC error: %v", err)
		}

		if resp == nil {
			t.Fatal("ExecuteCommand() returned nil response")
		}

		if resp.RequestId != testRequestID {
			t.Errorf("ExecuteCommand() RequestId = %q, want %q", resp.RequestId, testRequestID)
		}
	})
}

// TestUpdateConfig tests the UpdateConfig RPC handler
func TestUpdateConfig(t *testing.T) {
	t.Run("without config generator", func(t *testing.T) {
		cfg := &config.Config{
			AgentID: "test-agent",
			TLS: config.TLSConfig{
				Enabled: false,
			},
			Ocserv: config.OcservConfig{
				ConfigPath:     "/etc/ocserv/ocserv.conf",
				CtlSocket:      "/run/ocserv/occtl.socket",
				SystemdService: "ocserv",
				// No ConfigPerUserDir configured
			},
		}

		logger := zerolog.New(zerolog.NewTestWriter(t))

		server, err := New(cfg, logger)
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}

		req := &pb.ConfigUpdateRequest{
			RequestId:     "test-update-1",
			ConfigType:    pb.ConfigType_CONFIG_TYPE_PER_USER,
			ConfigName:    "testuser",
			ConfigContent: `{"routes":["10.0.0.0/8"],"dns":["8.8.8.8"]}`,
		}

		resp, err := server.UpdateConfig(context.Background(), req)

		if err != nil {
			t.Errorf("UpdateConfig() unexpected error = %v", err)
		}

		if resp == nil {
			t.Error("UpdateConfig() returned nil response")
			return
		}

		if resp.RequestId != req.RequestId {
			t.Errorf("UpdateConfig() RequestId = %v, want %v", resp.RequestId, req.RequestId)
		}

		// Should fail without config generator
		if resp.Success {
			t.Error("UpdateConfig() Success = true, want false (no config generator)")
		}

		if !strings.Contains(resp.ErrorMessage, "config generator not initialized") {
			t.Errorf("UpdateConfig() ErrorMessage = %q, want containing 'config generator not initialized'", resp.ErrorMessage)
		}
	})

	t.Run("main config not supported", func(t *testing.T) {
		cfg := &config.Config{
			AgentID: "test-agent",
			TLS: config.TLSConfig{
				Enabled: false,
			},
			Ocserv: config.OcservConfig{
				ConfigPath:        "/etc/ocserv/ocserv.conf",
				CtlSocket:         "/run/ocserv/occtl.socket",
				SystemdService:    "ocserv",
				ConfigPerUserDir:  t.TempDir(),
				ConfigPerGroupDir: t.TempDir(),
			},
		}

		logger := zerolog.New(zerolog.NewTestWriter(t))

		server, err := New(cfg, logger)
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}

		req := &pb.ConfigUpdateRequest{
			RequestId:     "test-update-2",
			ConfigType:    pb.ConfigType_CONFIG_TYPE_MAIN,
			ConfigName:    "test-config",
			ConfigContent: "# test content",
		}

		resp, err := server.UpdateConfig(context.Background(), req)

		if err != nil {
			t.Errorf("UpdateConfig() unexpected error = %v", err)
		}

		if resp.Success {
			t.Error("UpdateConfig() Success = true, want false for main config")
		}

		if !strings.Contains(resp.ErrorMessage, "not supported for safety") {
			t.Errorf("UpdateConfig() ErrorMessage = %q, want containing 'not supported for safety'", resp.ErrorMessage)
		}
	})

	t.Run("validate only mode", func(t *testing.T) {
		cfg := &config.Config{
			AgentID: "test-agent",
			TLS: config.TLSConfig{
				Enabled: false,
			},
			Ocserv: config.OcservConfig{
				ConfigPath:        "/etc/ocserv/ocserv.conf",
				CtlSocket:         "/run/ocserv/occtl.socket",
				SystemdService:    "ocserv",
				ConfigPerUserDir:  t.TempDir(),
				ConfigPerGroupDir: t.TempDir(),
			},
		}

		logger := zerolog.New(zerolog.NewTestWriter(t))

		server, err := New(cfg, logger)
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}

		req := &pb.ConfigUpdateRequest{
			RequestId:     "test-update-3",
			ConfigType:    pb.ConfigType_CONFIG_TYPE_PER_USER,
			ConfigName:    "testuser",
			ConfigContent: `{"routes":["10.0.0.0/8"],"dns":["8.8.8.8"]}`,
			ValidateOnly:  true,
		}

		resp, err := server.UpdateConfig(context.Background(), req)

		if err != nil {
			t.Errorf("UpdateConfig() unexpected error = %v", err)
		}

		if !resp.Success {
			t.Errorf("UpdateConfig() Success = false, want true for valid config")
		}

		if resp.ValidationResult != "validation passed" {
			t.Errorf("UpdateConfig() ValidationResult = %q, want 'validation passed'", resp.ValidationResult)
		}
	})

	t.Run("invalid routes", func(t *testing.T) {
		cfg := &config.Config{
			AgentID: "test-agent",
			TLS: config.TLSConfig{
				Enabled: false,
			},
			Ocserv: config.OcservConfig{
				ConfigPath:        "/etc/ocserv/ocserv.conf",
				CtlSocket:         "/run/ocserv/occtl.socket",
				SystemdService:    "ocserv",
				ConfigPerUserDir:  t.TempDir(),
				ConfigPerGroupDir: t.TempDir(),
			},
		}

		logger := zerolog.New(zerolog.NewTestWriter(t))

		server, err := New(cfg, logger)
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}

		req := &pb.ConfigUpdateRequest{
			RequestId:     "test-update-4",
			ConfigType:    pb.ConfigType_CONFIG_TYPE_PER_USER,
			ConfigName:    "testuser",
			ConfigContent: `{"routes":["invalid-route"]}`,
		}

		resp, err := server.UpdateConfig(context.Background(), req)

		if err != nil {
			t.Errorf("UpdateConfig() unexpected error = %v", err)
		}

		if resp.Success {
			t.Error("UpdateConfig() Success = true, want false for invalid routes")
		}

		if !strings.Contains(resp.ValidationResult, "invalid routes") {
			t.Errorf("UpdateConfig() ValidationResult = %q, want containing 'invalid routes'", resp.ValidationResult)
		}
	})
}

// TestStreamLogs tests the StreamLogs RPC handler
func TestStreamLogs(t *testing.T) {
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

	req := &pb.LogStreamRequest{
		LogSource: "ocserv",
		Follow:    true,
	}

	// StreamLogs requires a stream parameter, which is complex to mock
	// For now, we'll test with nil stream and expect Unimplemented error
	err = server.StreamLogs(req, nil)

	if err == nil {
		t.Error("StreamLogs() expected error, got nil")
		return
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Errorf("StreamLogs() error is not a gRPC status error: %v", err)
		return
	}

	if st.Code() != codes.Unimplemented {
		t.Errorf("StreamLogs() error code = %v, want %v", st.Code(), codes.Unimplemented)
	}

	if st.Message() != "not implemented yet" {
		t.Errorf("StreamLogs() error message = %q, want %q", st.Message(), "not implemented yet")
	}
}

// TestAgentStream tests the AgentStream RPC handler
func TestAgentStream(t *testing.T) {
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

	// AgentStream requires a bidirectional stream parameter
	// With nil stream, it will fail when trying to receive
	err = server.AgentStream(nil)

	// Should error because nil stream can't be used
	if err == nil {
		t.Error("AgentStream() expected error with nil stream, got nil")
	}
}
