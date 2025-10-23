package ocserv

import (
	"context"
	"strings"
	"testing"

	"github.com/dantte-lp/ocserv-agent/internal/config"
	"github.com/rs/zerolog"
)

// TestNewManager tests the NewManager factory function
func TestNewManager(t *testing.T) {
	cfg := &config.Config{
		Ocserv: config.OcservConfig{
			ConfigPath:     "/etc/ocserv/ocserv.conf",
			CtlSocket:      "/run/ocserv/occtl.socket",
			SystemdService: "ocserv",
		},
		Security: config.SecurityConfig{
			AllowedCommands:   []string{"systemctl", "occtl"},
			SudoUser:          "",
			MaxCommandTimeout: 30,
		},
	}

	logger := zerolog.New(zerolog.NewTestWriter(t))

	manager := NewManager(cfg, logger)

	if manager == nil {
		t.Fatal("NewManager() returned nil")
	}

	if manager.systemctl == nil {
		t.Error("NewManager() systemctl is nil")
	}

	if manager.occtl == nil {
		t.Error("NewManager() occtl is nil")
	}

	if manager.configReader == nil {
		t.Error("NewManager() configReader is nil")
	}

	if manager.allowedCommands == nil {
		t.Error("NewManager() allowedCommands map is nil")
	}

	if len(manager.allowedCommands) != 2 {
		t.Errorf("NewManager() allowedCommands length = %d, want 2", len(manager.allowedCommands))
	}

	if !manager.allowedCommands["systemctl"] {
		t.Error("NewManager() systemctl not in allowed commands")
	}

	if !manager.allowedCommands["occtl"] {
		t.Error("NewManager() occtl not in allowed commands")
	}
}

// TestIsCommandAllowed tests the isCommandAllowed validation
func TestIsCommandAllowed(t *testing.T) {
	tests := []struct {
		name            string
		allowedCommands []string
		command         string
		want            bool
	}{
		{
			name:            "systemctl is allowed",
			allowedCommands: []string{"systemctl", "occtl"},
			command:         "systemctl",
			want:            true,
		},
		{
			name:            "occtl is allowed",
			allowedCommands: []string{"systemctl", "occtl"},
			command:         "occtl",
			want:            true,
		},
		{
			name:            "unknown command not allowed",
			allowedCommands: []string{"systemctl", "occtl"},
			command:         "rm",
			want:            false,
		},
		{
			name:            "empty command not allowed",
			allowedCommands: []string{"systemctl", "occtl"},
			command:         "",
			want:            false,
		},
		{
			name:            "malicious command not allowed",
			allowedCommands: []string{"systemctl"},
			command:         "rm -rf /",
			want:            false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Ocserv: config.OcservConfig{
					ConfigPath:     "/etc/ocserv/ocserv.conf",
					CtlSocket:      "/run/ocserv/occtl.socket",
					SystemdService: "ocserv",
				},
				Security: config.SecurityConfig{
					AllowedCommands:   tt.allowedCommands,
					MaxCommandTimeout: 30,
				},
			}

			logger := zerolog.New(zerolog.NewTestWriter(t))
			manager := NewManager(cfg, logger)

			got := manager.isCommandAllowed(tt.command)
			if got != tt.want {
				t.Errorf("isCommandAllowed(%q) = %v, want %v", tt.command, got, tt.want)
			}
		})
	}
}

// TestValidateArguments tests the validateArguments security function
func TestValidateArguments(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid simple args",
			args:    []string{"start", "status"},
			wantErr: false,
		},
		{
			name:    "valid username",
			args:    []string{"show", "user", "testuser"},
			wantErr: false,
		},
		{
			name:    "empty args list",
			args:    []string{},
			wantErr: false,
		},
		{
			name:    "semicolon injection",
			args:    []string{"start; rm -rf /"},
			wantErr: true,
			errMsg:  "dangerous characters",
		},
		{
			name:    "pipe injection",
			args:    []string{"status | cat"},
			wantErr: true,
			errMsg:  "dangerous characters",
		},
		{
			name:    "ampersand injection",
			args:    []string{"start & echo hack"},
			wantErr: true,
			errMsg:  "dangerous characters",
		},
		{
			name:    "redirect output",
			args:    []string{"status > /tmp/out"},
			wantErr: true,
			errMsg:  "dangerous characters",
		},
		{
			name:    "redirect input",
			args:    []string{"start < /tmp/in"},
			wantErr: true,
			errMsg:  "dangerous characters",
		},
		{
			name:    "dollar sign variable",
			args:    []string{"echo $HOME"},
			wantErr: true,
			errMsg:  "dangerous characters",
		},
		{
			name:    "command substitution parentheses",
			args:    []string{"$(whoami)"},
			wantErr: true,
			errMsg:  "dangerous characters",
		},
		{
			name:    "curly braces",
			args:    []string{"{test}"},
			wantErr: true,
			errMsg:  "dangerous characters",
		},
		{
			name:    "square brackets",
			args:    []string{"[test]"},
			wantErr: true,
			errMsg:  "dangerous characters",
		},
		{
			name:    "directory traversal",
			args:    []string{"../../etc/passwd"},
			wantErr: true,
			errMsg:  "dangerous characters",
		},
		{
			name:    "flag starting with dash",
			args:    []string{"-rf"},
			wantErr: true,
			errMsg:  "dangerous characters",
		},
		{
			name:    "null byte injection",
			args:    []string{"test\x00hack"},
			wantErr: true,
			errMsg:  "dangerous characters", // Caught by control character check
		},
		{
			name:    "backtick command substitution",
			args:    []string{"test`whoami`"},
			wantErr: true,
			errMsg:  "dangerous characters",
		},
		{
			name:    "backtick at start",
			args:    []string{"`ls -la`"},
			wantErr: true,
			errMsg:  "dangerous characters",
		},
		{
			name:    "escaped semicolon",
			args:    []string{"test\\;whoami"},
			wantErr: true,
			errMsg:  "dangerous characters",
		},
		{
			name:    "escaped pipe",
			args:    []string{"test\\|cat"},
			wantErr: true,
			errMsg:  "dangerous characters",
		},
		{
			name:    "escaped dollar",
			args:    []string{"test\\$HOME"},
			wantErr: true,
			errMsg:  "dangerous characters",
		},
		{
			name:    "escaped backtick",
			args:    []string{"test\\`whoami\\`"},
			wantErr: true,
			errMsg:  "dangerous characters",
		},
		{
			name:    "newline injection LF",
			args:    []string{"test\nwhoami"},
			wantErr: true,
			errMsg:  "dangerous characters",
		},
		{
			name:    "newline injection CR",
			args:    []string{"test\rwhoami"},
			wantErr: true,
			errMsg:  "dangerous characters",
		},
		{
			name:    "newline injection CRLF",
			args:    []string{"test\r\nwhoami"},
			wantErr: true,
			errMsg:  "dangerous characters",
		},
		{
			name:    "control character BEL",
			args:    []string{"test\x07alert"},
			wantErr: true,
			errMsg:  "dangerous characters",
		},
		{
			name:    "control character ESC",
			args:    []string{"test\x1Bescape"},
			wantErr: true,
			errMsg:  "dangerous characters",
		},
		{
			name:    "tab character allowed",
			args:    []string{"test\ttab"},
			wantErr: false, // Tab is allowed (not in control char range)
		},
		{
			name:    "argument too long",
			args:    []string{strings.Repeat("a", 257)},
			wantErr: true,
			errMsg:  "too long",
		},
		{
			name:    "max length argument ok",
			args:    []string{strings.Repeat("a", 256)},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Ocserv: config.OcservConfig{
					ConfigPath:     "/etc/ocserv/ocserv.conf",
					CtlSocket:      "/run/ocserv/occtl.socket",
					SystemdService: "ocserv",
				},
				Security: config.SecurityConfig{
					AllowedCommands:   []string{"systemctl", "occtl"},
					MaxCommandTimeout: 30,
				},
			}

			logger := zerolog.New(zerolog.NewTestWriter(t))
			manager := NewManager(cfg, logger)

			err := manager.validateArguments(tt.args)

			if tt.wantErr {
				if err == nil {
					t.Errorf("validateArguments() expected error containing %q, got nil", tt.errMsg)
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validateArguments() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateArguments() unexpected error = %v", err)
				}
			}
		})
	}
}

// TestExecuteCommand_Validation tests ExecuteCommand validation logic
func TestExecuteCommand_Validation(t *testing.T) {
	tests := []struct {
		name            string
		allowedCommands []string
		commandType     string
		args            []string
		wantErr         bool
		errMsg          string
	}{
		{
			name:            "command not allowed",
			allowedCommands: []string{"systemctl"},
			commandType:     "occtl",
			args:            []string{"show", "users"},
			wantErr:         true,
			errMsg:          "command not allowed",
		},
		{
			name:            "invalid arguments",
			allowedCommands: []string{"systemctl"},
			commandType:     "systemctl",
			args:            []string{"start; rm -rf /"},
			wantErr:         true,
			errMsg:          "invalid arguments",
		},
		{
			name:            "unknown command type",
			allowedCommands: []string{"unknown"},
			commandType:     "unknown",
			args:            []string{"test"},
			wantErr:         true,
			errMsg:          "unknown command type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Ocserv: config.OcservConfig{
					ConfigPath:     "/etc/ocserv/ocserv.conf",
					CtlSocket:      "/run/ocserv/occtl.socket",
					SystemdService: "ocserv",
				},
				Security: config.SecurityConfig{
					AllowedCommands:   tt.allowedCommands,
					MaxCommandTimeout: 30,
				},
			}

			logger := zerolog.New(zerolog.NewTestWriter(t))
			manager := NewManager(cfg, logger)

			result, err := manager.ExecuteCommand(context.Background(), tt.commandType, tt.args)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ExecuteCommand() expected error containing %q, got nil", tt.errMsg)
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ExecuteCommand() error = %v, want error containing %q", err, tt.errMsg)
				}

				if result == nil {
					t.Error("ExecuteCommand() result is nil (should return result even on error)")
					return
				}

				if result.Success {
					t.Error("ExecuteCommand() result.Success = true, want false on error")
				}

				if !strings.Contains(result.ErrorMsg, tt.errMsg) {
					t.Errorf("ExecuteCommand() result.ErrorMsg = %q, want containing %q", result.ErrorMsg, tt.errMsg)
				}
			}
		})
	}
}

// TestGetters tests the getter methods
func TestGetters(t *testing.T) {
	cfg := &config.Config{
		Ocserv: config.OcservConfig{
			ConfigPath:     "/etc/ocserv/ocserv.conf",
			CtlSocket:      "/run/ocserv/occtl.socket",
			SystemdService: "ocserv",
		},
		Security: config.SecurityConfig{
			AllowedCommands:   []string{"systemctl", "occtl"},
			MaxCommandTimeout: 30,
		},
	}

	logger := zerolog.New(zerolog.NewTestWriter(t))
	manager := NewManager(cfg, logger)

	t.Run("GetSystemctlManager", func(t *testing.T) {
		systemctl := manager.GetSystemctlManager()
		if systemctl == nil {
			t.Error("GetSystemctlManager() returned nil")
		}
		if systemctl != manager.systemctl {
			t.Error("GetSystemctlManager() returned different instance")
		}
	})

	t.Run("GetOcctlManager", func(t *testing.T) {
		occtl := manager.GetOcctlManager()
		if occtl == nil {
			t.Error("GetOcctlManager() returned nil")
		}
		if occtl != manager.occtl {
			t.Error("GetOcctlManager() returned different instance")
		}
	})

	t.Run("GetConfigReader", func(t *testing.T) {
		configReader := manager.GetConfigReader()
		if configReader == nil {
			t.Error("GetConfigReader() returned nil")
		}
		if configReader != manager.configReader {
			t.Error("GetConfigReader() returned different instance")
		}
	})
}

// TestExecuteSystemctl_NoArgs tests executeSystemctl with no args
func TestExecuteSystemctl_NoArgs(t *testing.T) {
	cfg := &config.Config{
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
	manager := NewManager(cfg, logger)

	result, err := manager.ExecuteCommand(context.Background(), "systemctl", []string{})

	if err == nil {
		t.Error("ExecuteCommand(systemctl, []) expected error, got nil")
	}

	if result == nil {
		t.Error("ExecuteCommand() returned nil result")
		return
	}

	if result.Success {
		t.Error("ExecuteCommand() result.Success = true, want false")
	}

	if !strings.Contains(result.ErrorMsg, "requires action argument") {
		t.Errorf("ExecuteCommand() result.ErrorMsg = %q, want containing 'requires action argument'", result.ErrorMsg)
	}
}

// TestExecuteOcctl_InsufficientArgs tests executeOcctl with insufficient args
func TestExecuteOcctl_InsufficientArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "no args",
			args: []string{},
		},
		{
			name: "one arg",
			args: []string{"show"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
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
			manager := NewManager(cfg, logger)

			result, err := manager.ExecuteCommand(context.Background(), "occtl", tt.args)

			if err == nil {
				t.Errorf("ExecuteCommand(occtl, %v) expected error, got nil", tt.args)
			}

			if result == nil {
				t.Error("ExecuteCommand() returned nil result")
				return
			}

			if result.Success {
				t.Error("ExecuteCommand() result.Success = true, want false")
			}

			if !strings.Contains(result.ErrorMsg, "requires action and subcommand") {
				t.Errorf("ExecuteCommand() result.ErrorMsg = %q, want containing 'requires action and subcommand'", result.ErrorMsg)
			}
		})
	}
}

// TestManagerDelegationMethods tests the delegation methods to configReader
func TestManagerDelegationMethods(t *testing.T) {
	cfg := &config.Config{
		Ocserv: config.OcservConfig{
			ConfigPath:     "/etc/ocserv/ocserv.conf",
			CtlSocket:      "/run/ocserv/occtl.socket",
			SystemdService: "ocserv",
		},
		Security: config.SecurityConfig{
			AllowedCommands:   []string{"systemctl", "occtl"},
			MaxCommandTimeout: 30,
		},
	}

	logger := zerolog.New(zerolog.NewTestWriter(t))
	manager := NewManager(cfg, logger)
	ctx := context.Background()

	t.Run("ReadOcservConf", func(t *testing.T) {
		// This will fail because the file doesn't exist, but we're testing the delegation
		_, err := manager.ReadOcservConf(ctx, "/nonexistent/ocserv.conf")
		if err == nil {
			t.Error("ReadOcservConf() expected error for nonexistent file, got nil")
		}
		// Verify the error is from the config reader (file not found)
		if !strings.Contains(err.Error(), "no such file") && !strings.Contains(err.Error(), "failed to read") {
			t.Logf("ReadOcservConf() error = %v (expected file not found error)", err)
		}
	})

	t.Run("ReadUserConfig", func(t *testing.T) {
		_, err := manager.ReadUserConfig(ctx, "/nonexistent", "testuser")
		if err == nil {
			t.Error("ReadUserConfig() expected error for nonexistent dir, got nil")
		}
	})

	t.Run("ReadGroupConfig", func(t *testing.T) {
		_, err := manager.ReadGroupConfig(ctx, "/nonexistent", "testgroup")
		if err == nil {
			t.Error("ReadGroupConfig() expected error for nonexistent dir, got nil")
		}
	})

	t.Run("ListUserConfigs", func(t *testing.T) {
		// ListUserConfigs returns empty list for nonexistent dir, not an error
		configs, err := manager.ListUserConfigs(ctx, "/nonexistent")
		if err != nil {
			t.Errorf("ListUserConfigs() unexpected error = %v", err)
		}
		if len(configs) != 0 {
			t.Errorf("ListUserConfigs() returned %d configs for nonexistent dir, want 0", len(configs))
		}
	})

	t.Run("ListGroupConfigs", func(t *testing.T) {
		// ListGroupConfigs returns empty list for nonexistent dir, not an error
		configs, err := manager.ListGroupConfigs(ctx, "/nonexistent")
		if err != nil {
			t.Errorf("ListGroupConfigs() unexpected error = %v", err)
		}
		if len(configs) != 0 {
			t.Errorf("ListGroupConfigs() returned %d configs for nonexistent dir, want 0", len(configs))
		}
	})
}
