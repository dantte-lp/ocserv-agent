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
		name          string
		tier          int32
		wantHealthy   bool
		wantErr       bool
		wantErrCode   codes.Code
		expectedCheck string // Key check to verify
	}{
		{
			name:          "tier 1 basic heartbeat",
			tier:          1,
			wantHealthy:   true,
			wantErr:       false,
			expectedCheck: "agent",
		},
		{
			name:          "tier 2 deep check",
			tier:          2,
			wantHealthy:   true,
			wantErr:       false,
			expectedCheck: "ocserv_process",
		},
		{
			name:          "tier 3 application check",
			tier:          3,
			wantHealthy:   false, // Not implemented, so not healthy
			wantErr:       false,
			expectedCheck: "end_to_end",
		},
		{
			name:        "invalid tier - too low",
			tier:        0,
			wantHealthy: false,
			wantErr:     true,
			wantErrCode: codes.InvalidArgument,
		},
		{
			name:        "invalid tier - too high",
			tier:        4,
			wantHealthy: false,
			wantErr:     true,
			wantErrCode: codes.InvalidArgument,
		},
		{
			name:        "invalid tier - negative",
			tier:        -1,
			wantHealthy: false,
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

			if resp.Healthy != tt.wantHealthy {
				t.Errorf("HealthCheck() Healthy = %v, want %v", resp.Healthy, tt.wantHealthy)
			}

			if resp.StatusMessage == "" {
				t.Error("HealthCheck() StatusMessage is empty")
			}

			if resp.Timestamp == nil {
				t.Error("HealthCheck() Timestamp is nil")
			}

			if len(resp.Checks) == 0 {
				t.Error("HealthCheck() Checks map is empty")
			}

			if tt.expectedCheck != "" {
				if _, ok := resp.Checks[tt.expectedCheck]; !ok {
					t.Errorf("HealthCheck() expected check %q not found in response", tt.expectedCheck)
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

	req := &pb.ConfigUpdateRequest{
		RequestId:     "test-update-1",
		ConfigType:    pb.ConfigType_CONFIG_TYPE_MAIN,
		ConfigName:    "test-config",
		ConfigContent: "# test content",
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

	// Should return not implemented
	if resp.Success {
		t.Error("UpdateConfig() Success = true, want false (not implemented)")
	}

	if resp.ErrorMessage != "not implemented yet" {
		t.Errorf("UpdateConfig() ErrorMessage = %q, want %q", resp.ErrorMessage, "not implemented yet")
	}
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
	// For now, we'll test with nil stream and expect Unimplemented error
	err = server.AgentStream(nil)

	if err == nil {
		t.Error("AgentStream() expected error, got nil")
		return
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Errorf("AgentStream() error is not a gRPC status error: %v", err)
		return
	}

	if st.Code() != codes.Unimplemented {
		t.Errorf("AgentStream() error code = %v, want %v", st.Code(), codes.Unimplemented)
	}

	if st.Message() != "not implemented yet" {
		t.Errorf("AgentStream() error message = %q, want %q", st.Message(), "not implemented yet")
	}
}
