package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestLoad tests the Load function with various config files
func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid config",
			path:    "../../test/fixtures/config/valid.yaml",
			wantErr: false,
		},
		{
			name:    "minimal config with defaults",
			path:    "../../test/fixtures/config/minimal.yaml",
			wantErr: false,
		},
		{
			name:    "missing agent_id",
			path:    "../../test/fixtures/config/invalid_missing_agent_id.yaml",
			wantErr: true,
			errMsg:  "agent_id is required",
		},
		{
			name:    "missing control_server",
			path:    "../../test/fixtures/config/invalid_missing_control_server.yaml",
			wantErr: true,
			errMsg:  "control_server.address is required",
		},
		{
			name:    "file not found",
			path:    "nonexistent.yaml",
			wantErr: true,
			errMsg:  "failed to read config file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := Load(tt.path)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Load() expected error containing %q, got nil", tt.errMsg)
					return
				}
				if tt.errMsg != "" && !containsError(err.Error(), tt.errMsg) {
					t.Errorf("Load() error = %v, want error containing %q", err, tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("Load() unexpected error = %v", err)
				return
			}

			if cfg == nil {
				t.Error("Load() returned nil config without error")
			}
		})
	}
}

// TestLoadWithEnvOverrides tests environment variable overrides
func TestLoadWithEnvOverrides(t *testing.T) {
	// Save original env vars
	origAgentID := os.Getenv("AGENT_ID")
	origControlAddr := os.Getenv("CONTROL_SERVER_ADDRESS")
	origLogLevel := os.Getenv("LOG_LEVEL")

	// Restore env vars after test
	defer func() {
		os.Setenv("AGENT_ID", origAgentID)
		os.Setenv("CONTROL_SERVER_ADDRESS", origControlAddr)
		os.Setenv("LOG_LEVEL", origLogLevel)
	}()

	// Set test env vars
	os.Setenv("AGENT_ID", "env-override-agent")
	os.Setenv("CONTROL_SERVER_ADDRESS", "env-control:8080")
	os.Setenv("LOG_LEVEL", "debug")

	cfg, err := Load("../../test/fixtures/config/minimal.yaml")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify env overrides were applied
	if cfg.AgentID != "env-override-agent" {
		t.Errorf("AgentID = %v, want %v", cfg.AgentID, "env-override-agent")
	}
	if cfg.ControlServer.Address != "env-control:8080" {
		t.Errorf("ControlServer.Address = %v, want %v", cfg.ControlServer.Address, "env-control:8080")
	}
	if cfg.Logging.Level != "debug" {
		t.Errorf("Logging.Level = %v, want %v", cfg.Logging.Level, "debug")
	}
}

// TestSetDefaults tests default value application
func TestSetDefaults(t *testing.T) {
	cfg := &Config{}
	setDefaults(cfg)

	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"logging level", cfg.Logging.Level, "info"},
		{"logging format", cfg.Logging.Format, "json"},
		{"logging output", cfg.Logging.Output, "stdout"},
		{"heartbeat interval", cfg.Health.HeartbeatInterval, 15 * time.Second},
		{"deep check interval", cfg.Health.DeepCheckInterval, 2 * time.Minute},
		{"metrics interval", cfg.Health.MetricsInterval, 30 * time.Second},
		{"reconnect initial delay", cfg.ControlServer.Reconnect.InitialDelay, 1 * time.Second},
		{"reconnect max delay", cfg.ControlServer.Reconnect.MaxDelay, 60 * time.Second},
		{"reconnect multiplier", cfg.ControlServer.Reconnect.Multiplier, 2.0},
		{"reconnect max attempts", cfg.ControlServer.Reconnect.MaxAttempts, 5},
		{"max command timeout", cfg.Security.MaxCommandTimeout, 300 * time.Second},
		{"telemetry service name", cfg.Telemetry.ServiceName, "ocserv-agent"},
		{"telemetry sample rate", cfg.Telemetry.SampleRate, 1.0},
		{"TLS min version", cfg.TLS.MinVersion, "TLS1.3"},
		{"systemd service", cfg.Ocserv.SystemdService, "ocserv"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.expected)
			}
		})
	}
}

// TestSetDefaults_NoOverride tests that defaults don't override existing values
func TestSetDefaults_NoOverride(t *testing.T) {
	cfg := &Config{
		Logging: LoggingConfig{
			Level:  "warn",
			Format: "text",
			Output: "file",
		},
		Health: HealthConfig{
			HeartbeatInterval: 10 * time.Second,
			DeepCheckInterval: 5 * time.Minute,
		},
		Telemetry: TelemetryConfig{
			ServiceName: "custom-agent",
			SampleRate:  0.5,
		},
	}

	setDefaults(cfg)

	// Verify existing values were not overridden
	if cfg.Logging.Level != "warn" {
		t.Errorf("Logging.Level was overridden: got %v, want warn", cfg.Logging.Level)
	}
	if cfg.Logging.Format != "text" {
		t.Errorf("Logging.Format was overridden: got %v, want text", cfg.Logging.Format)
	}
	if cfg.Health.HeartbeatInterval != 10*time.Second {
		t.Errorf("HeartbeatInterval was overridden: got %v, want 10s", cfg.Health.HeartbeatInterval)
	}
	if cfg.Telemetry.ServiceName != "custom-agent" {
		t.Errorf("Telemetry.ServiceName was overridden: got %v, want custom-agent", cfg.Telemetry.ServiceName)
	}
}

// TestBootstrapCertificates tests certificate auto-generation
func TestBootstrapCertificates(t *testing.T) {
	tests := []struct {
		name         string
		tlsEnabled   bool
		autoGen      bool
		wantGenerate bool
	}{
		{
			name:         "TLS disabled - skip bootstrap",
			tlsEnabled:   false,
			autoGen:      true,
			wantGenerate: false,
		},
		{
			name:         "auto_generate disabled - skip bootstrap",
			tlsEnabled:   true,
			autoGen:      false,
			wantGenerate: false,
		},
		{
			name:         "TLS and auto_generate enabled - generate certs",
			tlsEnabled:   true,
			autoGen:      true,
			wantGenerate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory for test certs
			tmpDir := t.TempDir()

			cfg := &Config{
				Hostname: "test-host",
				TLS: TLSConfig{
					Enabled:      tt.tlsEnabled,
					AutoGenerate: tt.autoGen,
					CertFile:     filepath.Join(tmpDir, "agent.crt"),
					KeyFile:      filepath.Join(tmpDir, "agent.key"),
					CAFile:       filepath.Join(tmpDir, "ca.crt"),
				},
			}

			err := bootstrapCertificates(cfg)
			if err != nil {
				t.Errorf("bootstrapCertificates() unexpected error = %v", err)
				return
			}

			// Check if certificates were generated
			certExists := fileExists(cfg.TLS.CertFile)
			keyExists := fileExists(cfg.TLS.KeyFile)
			caExists := fileExists(cfg.TLS.CAFile)

			if tt.wantGenerate {
				if !certExists || !keyExists || !caExists {
					t.Errorf("Expected certificates to be generated, but some are missing: cert=%v, key=%v, ca=%v",
						certExists, keyExists, caExists)
				}
			} else {
				if certExists || keyExists || caExists {
					t.Errorf("Expected no certificates, but some were generated: cert=%v, key=%v, ca=%v",
						certExists, keyExists, caExists)
				}
			}
		})
	}
}

// TestApplyEnvOverrides tests environment variable application
func TestApplyEnvOverrides(t *testing.T) {
	// Save original env
	originalEnv := map[string]string{
		"AGENT_ID":               os.Getenv("AGENT_ID"),
		"CONTROL_SERVER_ADDRESS": os.Getenv("CONTROL_SERVER_ADDRESS"),
		"TLS_CERT_FILE":          os.Getenv("TLS_CERT_FILE"),
		"TLS_KEY_FILE":           os.Getenv("TLS_KEY_FILE"),
		"TLS_CA_FILE":            os.Getenv("TLS_CA_FILE"),
		"LOG_LEVEL":              os.Getenv("LOG_LEVEL"),
		"TELEMETRY_ENDPOINT":     os.Getenv("TELEMETRY_ENDPOINT"),
	}

	// Restore env after test
	defer func() {
		for k, v := range originalEnv {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	}()

	// Set test env vars
	testEnv := map[string]string{
		"AGENT_ID":               "test-agent-env",
		"CONTROL_SERVER_ADDRESS": "control-env:9999",
		"TLS_CERT_FILE":          "/env/cert.pem",
		"TLS_KEY_FILE":           "/env/key.pem",
		"TLS_CA_FILE":            "/env/ca.pem",
		"LOG_LEVEL":              "trace",
		"TELEMETRY_ENDPOINT":     "http://otel-env:4318",
	}

	for k, v := range testEnv {
		os.Setenv(k, v)
	}

	cfg := &Config{}
	applyEnvOverrides(cfg)

	// Verify all env vars were applied
	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{"AGENT_ID", cfg.AgentID, "test-agent-env"},
		{"CONTROL_SERVER_ADDRESS", cfg.ControlServer.Address, "control-env:9999"},
		{"TLS_CERT_FILE", cfg.TLS.CertFile, "/env/cert.pem"},
		{"TLS_KEY_FILE", cfg.TLS.KeyFile, "/env/key.pem"},
		{"TLS_CA_FILE", cfg.TLS.CAFile, "/env/ca.pem"},
		{"LOG_LEVEL", cfg.Logging.Level, "trace"},
		{"TELEMETRY_ENDPOINT", cfg.Telemetry.OTLP.Endpoint, "http://otel-env:4318"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.expected)
			}
		})
	}
}

// Helper functions

func containsError(got, want string) bool {
	return len(got) > 0 && len(want) > 0 &&
		(got == want || len(got) >= len(want) && contains(got, want))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || findSubstring(s, substr))
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

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
