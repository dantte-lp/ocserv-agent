package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestValidate tests the main Validate function
func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid minimal config",
			cfg: &Config{
				AgentID: "test-agent",
				ControlServer: ControlServerConfig{
					Address: "localhost:9090",
					Reconnect: ReconnectConfig{
						InitialDelay: 1 * time.Second,
						MaxDelay:     60 * time.Second,
						Multiplier:   2.0,
						MaxAttempts:  5,
					},
				},
				TLS: TLSConfig{
					Enabled:    false,
					MinVersion: "TLS1.3",
				},
				Ocserv: OcservConfig{
					ConfigPath:     "/etc/ocserv/ocserv.conf",
					CtlSocket:      "/run/ocserv/occtl.socket",
					SystemdService: "ocserv",
					BackupDir:      "/var/backups",
				},
				Health: HealthConfig{
					HeartbeatInterval: 15 * time.Second,
					DeepCheckInterval: 2 * time.Minute,
					MetricsInterval:   30 * time.Second,
				},
				Logging: LoggingConfig{
					Level:  "info",
					Format: "json",
					Output: "stdout",
				},
				Security: SecurityConfig{
					AllowedCommands:   []string{"occtl"},
					MaxCommandTimeout: 300 * time.Second,
				},
			},
			wantErr: false,
		},
		{
			name: "missing agent_id",
			cfg: &Config{
				ControlServer: ControlServerConfig{
					Address: "localhost:9090",
				},
			},
			wantErr: true,
			errMsg:  "agent_id is required",
		},
		{
			name: "missing control server address",
			cfg: &Config{
				AgentID: "test-agent",
				ControlServer: ControlServerConfig{
					Address: "",
				},
			},
			wantErr: true,
			errMsg:  "control_server.address is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.cfg)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() expected error, got nil")
					return
				}
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error = %v", err)
				}
			}
		})
	}
}

// TestValidateTLS tests TLS validation
func TestValidateTLS(t *testing.T) {
	// Create temp dir and test files
	tmpDir := t.TempDir()
	certFile := filepath.Join(tmpDir, "cert.pem")
	keyFile := filepath.Join(tmpDir, "key.pem")
	caFile := filepath.Join(tmpDir, "ca.pem")

	// Create test certificate files
	os.WriteFile(certFile, []byte("cert"), 0644)
	os.WriteFile(keyFile, []byte("key"), 0600)
	os.WriteFile(caFile, []byte("ca"), 0644)

	tests := []struct {
		name    string
		tls     *TLSConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "TLS disabled - no validation",
			tls: &TLSConfig{
				Enabled: false,
			},
			wantErr: false,
		},
		{
			name: "TLS enabled with auto_generate - paths required",
			tls: &TLSConfig{
				Enabled:      true,
				AutoGenerate: true,
				CertFile:     certFile,
				KeyFile:      keyFile,
				CAFile:       caFile,
				MinVersion:   "TLS1.3",
			},
			wantErr: false,
		},
		{
			name: "TLS enabled with existing files",
			tls: &TLSConfig{
				Enabled:      true,
				AutoGenerate: false,
				CertFile:     certFile,
				KeyFile:      keyFile,
				CAFile:       caFile,
				MinVersion:   "TLS1.3",
			},
			wantErr: false,
		},
		{
			name: "TLS enabled without cert_file",
			tls: &TLSConfig{
				Enabled: true,
				KeyFile: keyFile,
				CAFile:  caFile,
			},
			wantErr: true,
			errMsg:  "cert_file is required",
		},
		{
			name: "TLS enabled without key_file",
			tls: &TLSConfig{
				Enabled:  true,
				CertFile: certFile,
				CAFile:   caFile,
			},
			wantErr: true,
			errMsg:  "key_file is required",
		},
		{
			name: "TLS enabled without ca_file",
			tls: &TLSConfig{
				Enabled:  true,
				CertFile: certFile,
				KeyFile:  keyFile,
			},
			wantErr: true,
			errMsg:  "ca_file is required",
		},
		{
			name: "TLS enabled with nonexistent cert file",
			tls: &TLSConfig{
				Enabled:      true,
				AutoGenerate: false,
				CertFile:     "/nonexistent/cert.pem",
				KeyFile:      keyFile,
				CAFile:       caFile,
			},
			wantErr: true,
			errMsg:  "cert_file not found",
		},
		{
			name: "invalid TLS version",
			tls: &TLSConfig{
				Enabled:      true,
				AutoGenerate: true,
				CertFile:     certFile,
				KeyFile:      keyFile,
				CAFile:       caFile,
				MinVersion:   "TLS1.0",
			},
			wantErr: true,
			errMsg:  "invalid min_version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTLS(tt.tls)

			if tt.wantErr {
				if err == nil {
					t.Errorf("validateTLS() expected error, got nil")
					return
				}
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("validateTLS() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateTLS() unexpected error = %v", err)
				}
			}
		})
	}
}

// TestValidateOcserv tests ocserv configuration validation
func TestValidateOcserv(t *testing.T) {
	tests := []struct {
		name    string
		ocserv  *OcservConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			ocserv: &OcservConfig{
				ConfigPath:     "/etc/ocserv/ocserv.conf",
				CtlSocket:      "/run/ocserv/occtl.socket",
				SystemdService: "ocserv",
				BackupDir:      "/var/backups",
			},
			wantErr: false,
		},
		{
			name: "missing config_path",
			ocserv: &OcservConfig{
				ConfigPath: "",
			},
			wantErr: true,
			errMsg:  "config_path is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateOcserv(tt.ocserv)

			if tt.wantErr {
				if err == nil {
					t.Errorf("validateOcserv() expected error, got nil")
					return
				}
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("validateOcserv() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateOcserv() unexpected error = %v", err)
				}
			}
		})
	}
}

// TestValidateHealth tests health check configuration validation
func TestValidateHealth(t *testing.T) {
	tests := []struct {
		name    string
		health  *HealthConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			health: &HealthConfig{
				HeartbeatInterval: 15 * time.Second,
				DeepCheckInterval: 2 * time.Minute,
				MetricsInterval:   30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "heartbeat zero",
			health: &HealthConfig{
				HeartbeatInterval: 0,
				DeepCheckInterval: 2 * time.Minute,
				MetricsInterval:   30 * time.Second,
			},
			wantErr: true,
			errMsg:  "heartbeat_interval must be > 0",
		},
		{
			name: "deep check zero",
			health: &HealthConfig{
				HeartbeatInterval: 15 * time.Second,
				DeepCheckInterval: 0,
				MetricsInterval:   30 * time.Second,
			},
			wantErr: true,
			errMsg:  "deep_check_interval must be > 0",
		},
		{
			name: "metrics interval zero",
			health: &HealthConfig{
				HeartbeatInterval: 15 * time.Second,
				DeepCheckInterval: 2 * time.Minute,
				MetricsInterval:   0,
			},
			wantErr: true,
			errMsg:  "metrics_interval must be > 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateHealth(tt.health)

			if tt.wantErr {
				if err == nil {
					t.Errorf("validateHealth() expected error, got nil")
					return
				}
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("validateHealth() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateHealth() unexpected error = %v", err)
				}
			}
		})
	}
}

// TestValidateLogging tests logging configuration validation
func TestValidateLogging(t *testing.T) {
	tests := []struct {
		name    string
		logging *LoggingConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid json format",
			logging: &LoggingConfig{
				Level:  "info",
				Format: "json",
				Output: "stdout",
			},
			wantErr: false,
		},
		{
			name: "valid text format",
			logging: &LoggingConfig{
				Level:    "debug",
				Format:   "text",
				Output:   "file",
				FilePath: "/tmp/test.log",
			},
			wantErr: false,
		},
		{
			name: "invalid log level",
			logging: &LoggingConfig{
				Level:  "invalid",
				Format: "json",
				Output: "stdout",
			},
			wantErr: true,
			errMsg:  "invalid level",
		},
		{
			name: "invalid format",
			logging: &LoggingConfig{
				Level:  "info",
				Format: "xml",
				Output: "stdout",
			},
			wantErr: true,
			errMsg:  "invalid format",
		},
		{
			name: "invalid output",
			logging: &LoggingConfig{
				Level:  "info",
				Format: "json",
				Output: "network",
			},
			wantErr: true,
			errMsg:  "invalid output",
		},
		{
			name: "file output without path",
			logging: &LoggingConfig{
				Level:  "info",
				Format: "json",
				Output: "file",
			},
			wantErr: true,
			errMsg:  "file_path is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateLogging(tt.logging)

			if tt.wantErr {
				if err == nil {
					t.Errorf("validateLogging() expected error, got nil")
					return
				}
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("validateLogging() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateLogging() unexpected error = %v", err)
				}
			}
		})
	}
}

// TestValidateSecurity tests security configuration validation
func TestValidateSecurity(t *testing.T) {
	tests := []struct {
		name     string
		security *SecurityConfig
		wantErr  bool
		errMsg   string
	}{
		{
			name: "valid config",
			security: &SecurityConfig{
				AllowedCommands:   []string{"occtl", "systemctl"},
				MaxCommandTimeout: 300 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "empty allowed commands",
			security: &SecurityConfig{
				AllowedCommands:   []string{},
				MaxCommandTimeout: 300 * time.Second,
			},
			wantErr: true,
			errMsg:  "allowed_commands cannot be empty",
		},
		{
			name: "timeout zero",
			security: &SecurityConfig{
				AllowedCommands:   []string{"occtl"},
				MaxCommandTimeout: 0,
			},
			wantErr: true,
			errMsg:  "max_command_timeout must be > 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSecurity(tt.security)

			if tt.wantErr {
				if err == nil {
					t.Errorf("validateSecurity() expected error, got nil")
					return
				}
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("validateSecurity() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateSecurity() unexpected error = %v", err)
				}
			}
		})
	}
}

// TestValidateReconnect tests reconnect configuration validation
func TestValidateReconnect(t *testing.T) {
	tests := []struct {
		name      string
		reconnect *ReconnectConfig
		wantErr   bool
		errMsg    string
	}{
		{
			name: "valid config",
			reconnect: &ReconnectConfig{
				InitialDelay: 1 * time.Second,
				MaxDelay:     60 * time.Second,
				Multiplier:   2.0,
				MaxAttempts:  5,
			},
			wantErr: false,
		},
		{
			name: "initial delay zero",
			reconnect: &ReconnectConfig{
				InitialDelay: 0,
				MaxDelay:     60 * time.Second,
				Multiplier:   2.0,
				MaxAttempts:  5,
			},
			wantErr: true,
			errMsg:  "initial_delay must be > 0",
		},
		{
			name: "max delay less than initial",
			reconnect: &ReconnectConfig{
				InitialDelay: 5 * time.Second,
				MaxDelay:     2 * time.Second,
				Multiplier:   2.0,
				MaxAttempts:  5,
			},
			wantErr: true,
			errMsg:  "max_delay must be >= initial_delay",
		},
		{
			name: "multiplier too low",
			reconnect: &ReconnectConfig{
				InitialDelay: 1 * time.Second,
				MaxDelay:     60 * time.Second,
				Multiplier:   1.0,
				MaxAttempts:  5,
			},
			wantErr: true,
			errMsg:  "multiplier must be > 1.0",
		},
		{
			name: "max attempts zero",
			reconnect: &ReconnectConfig{
				InitialDelay: 1 * time.Second,
				MaxDelay:     60 * time.Second,
				Multiplier:   2.0,
				MaxAttempts:  0,
			},
			wantErr: true,
			errMsg:  "max_attempts must be > 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateReconnect(tt.reconnect)

			if tt.wantErr {
				if err == nil {
					t.Errorf("validateReconnect() expected error, got nil")
					return
				}
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("validateReconnect() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateReconnect() unexpected error = %v", err)
				}
			}
		})
	}
}
