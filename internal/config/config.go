package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dantte-lp/ocserv-agent/internal/cert"
	"gopkg.in/yaml.v3"
)

// Config represents the complete agent configuration
type Config struct {
	AgentID  string `yaml:"agent_id"`
	Hostname string `yaml:"hostname"`

	ControlServer ControlServerConfig `yaml:"control_server"`
	TLS           TLSConfig           `yaml:"tls"`
	Ocserv        OcservConfig        `yaml:"ocserv"`
	Health        HealthConfig        `yaml:"health"`
	Telemetry     TelemetryConfig     `yaml:"telemetry"`
	Logging       LoggingConfig       `yaml:"logging"`
	Security      SecurityConfig      `yaml:"security"`
}

// ControlServerConfig defines connection settings to control server
type ControlServerConfig struct {
	Address        string               `yaml:"address"`
	Reconnect      ReconnectConfig      `yaml:"reconnect"`
	CircuitBreaker CircuitBreakerConfig `yaml:"circuit_breaker"`
}

// ReconnectConfig defines reconnection behavior
type ReconnectConfig struct {
	InitialDelay time.Duration `yaml:"initial_delay"`
	MaxDelay     time.Duration `yaml:"max_delay"`
	Multiplier   float64       `yaml:"multiplier"`
	MaxAttempts  int           `yaml:"max_attempts"`
}

// CircuitBreakerConfig defines circuit breaker settings
type CircuitBreakerConfig struct {
	FailureThreshold int           `yaml:"failure_threshold"`
	Timeout          time.Duration `yaml:"timeout"`
}

// TLSConfig defines mTLS configuration
type TLSConfig struct {
	Enabled      bool   `yaml:"enabled"`
	AutoGenerate bool   `yaml:"auto_generate"` // Auto-generate self-signed certs if missing
	CertFile     string `yaml:"cert_file"`
	KeyFile      string `yaml:"key_file"`
	CAFile       string `yaml:"ca_file"`
	ServerName   string `yaml:"server_name"`
	MinVersion   string `yaml:"min_version"`
}

// OcservConfig defines ocserv paths and settings
type OcservConfig struct {
	ConfigPath        string `yaml:"config_path"`
	ConfigPerUserDir  string `yaml:"config_per_user_dir"`
	ConfigPerGroupDir string `yaml:"config_per_group_dir"`
	CtlSocket         string `yaml:"ctl_socket"`
	SystemdService    string `yaml:"systemd_service"`
	BackupDir         string `yaml:"backup_dir"`
}

// HealthConfig defines health check intervals
type HealthConfig struct {
	HeartbeatInterval time.Duration `yaml:"heartbeat_interval"`
	DeepCheckInterval time.Duration `yaml:"deep_check_interval"`
	MetricsInterval   time.Duration `yaml:"metrics_interval"`
}

// TelemetryConfig defines OpenTelemetry settings
type TelemetryConfig struct {
	Enabled        bool    `yaml:"enabled"`
	Endpoint       string  `yaml:"endpoint"`
	ServiceName    string  `yaml:"service_name"`
	ServiceVersion string  `yaml:"service_version"`
	SampleRate     float64 `yaml:"sample_rate"`
}

// LoggingConfig defines logging behavior
type LoggingConfig struct {
	Level      string `yaml:"level"`
	Format     string `yaml:"format"`
	Output     string `yaml:"output"`
	FilePath   string `yaml:"file_path"`
	MaxSizeMB  int    `yaml:"max_size_mb"`
	MaxBackups int    `yaml:"max_backups"`
	MaxAgeDays int    `yaml:"max_age_days"`
}

// SecurityConfig defines security constraints
type SecurityConfig struct {
	AllowedCommands   []string      `yaml:"allowed_commands"`
	SudoUser          string        `yaml:"sudo_user"`
	MaxCommandTimeout time.Duration `yaml:"max_command_timeout"`
}

// Load reads configuration from a YAML file and applies environment variable overrides
func Load(path string) (*Config, error) {
	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Apply environment variable overrides
	applyEnvOverrides(&cfg)

	// Auto-detect hostname if not set
	if cfg.Hostname == "" {
		hostname, err := os.Hostname()
		if err == nil {
			cfg.Hostname = hostname
		}
	}

	// Set defaults
	setDefaults(&cfg)

	// Validate
	if err := Validate(&cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	// Bootstrap certificates if auto_generate is enabled and certs don't exist
	if err := bootstrapCertificates(&cfg); err != nil {
		return nil, fmt.Errorf("certificate bootstrap failed: %w", err)
	}

	return &cfg, nil
}

// applyEnvOverrides applies environment variable overrides to config
func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("AGENT_ID"); v != "" {
		cfg.AgentID = v
	}
	if v := os.Getenv("CONTROL_SERVER_ADDRESS"); v != "" {
		cfg.ControlServer.Address = v
	}
	if v := os.Getenv("TLS_CERT_FILE"); v != "" {
		cfg.TLS.CertFile = v
	}
	if v := os.Getenv("TLS_KEY_FILE"); v != "" {
		cfg.TLS.KeyFile = v
	}
	if v := os.Getenv("TLS_CA_FILE"); v != "" {
		cfg.TLS.CAFile = v
	}
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		cfg.Logging.Level = v
	}
	if v := os.Getenv("TELEMETRY_ENDPOINT"); v != "" {
		cfg.Telemetry.Endpoint = v
	}
}

// setDefaults applies default values if not set
func setDefaults(cfg *Config) {
	if cfg.Logging.Level == "" {
		cfg.Logging.Level = "info"
	}
	if cfg.Logging.Format == "" {
		cfg.Logging.Format = "json"
	}
	if cfg.Logging.Output == "" {
		cfg.Logging.Output = "stdout"
	}

	if cfg.Health.HeartbeatInterval == 0 {
		cfg.Health.HeartbeatInterval = 15 * time.Second
	}
	if cfg.Health.DeepCheckInterval == 0 {
		cfg.Health.DeepCheckInterval = 2 * time.Minute
	}
	if cfg.Health.MetricsInterval == 0 {
		cfg.Health.MetricsInterval = 30 * time.Second
	}

	if cfg.ControlServer.Reconnect.InitialDelay == 0 {
		cfg.ControlServer.Reconnect.InitialDelay = 1 * time.Second
	}
	if cfg.ControlServer.Reconnect.MaxDelay == 0 {
		cfg.ControlServer.Reconnect.MaxDelay = 60 * time.Second
	}
	if cfg.ControlServer.Reconnect.Multiplier == 0 {
		cfg.ControlServer.Reconnect.Multiplier = 2.0
	}
	if cfg.ControlServer.Reconnect.MaxAttempts == 0 {
		cfg.ControlServer.Reconnect.MaxAttempts = 5
	}

	if cfg.Security.MaxCommandTimeout == 0 {
		cfg.Security.MaxCommandTimeout = 300 * time.Second
	}

	if cfg.Telemetry.ServiceName == "" {
		cfg.Telemetry.ServiceName = "ocserv-agent"
	}
	if cfg.Telemetry.SampleRate == 0 {
		cfg.Telemetry.SampleRate = 1.0
	}

	if cfg.TLS.MinVersion == "" {
		cfg.TLS.MinVersion = "TLS1.3"
	}

	if cfg.Ocserv.SystemdService == "" {
		cfg.Ocserv.SystemdService = "ocserv"
	}
}

// bootstrapCertificates generates self-signed certificates if auto_generate is enabled
// and certificate files don't exist
func bootstrapCertificates(cfg *Config) error {
	// Skip if TLS is disabled
	if !cfg.TLS.Enabled {
		return nil
	}

	// Skip if auto_generate is disabled
	if !cfg.TLS.AutoGenerate {
		return nil
	}

	// Check if certificates already exist
	if cert.CertsExist(cfg.TLS.CertFile, cfg.TLS.KeyFile, cfg.TLS.CAFile) {
		return nil
	}

	// Determine output directory from cert file path
	outputDir := filepath.Dir(cfg.TLS.CertFile)

	// Generate self-signed certificates
	info, err := cert.GenerateSelfSignedCerts(outputDir, cfg.Hostname)
	if err != nil {
		return fmt.Errorf("failed to generate certificates: %w", err)
	}

	// Log certificate information (using fmt.Printf for now, will be replaced with logger)
	fmt.Printf("🔐 Generated self-signed certificates for bootstrap mode\n")
	fmt.Printf("   CA Fingerprint:   %s\n", info.CAFingerprint)
	fmt.Printf("   Cert Fingerprint: %s\n", info.CertFingerprint)
	fmt.Printf("   Subject:          %s\n", info.Subject)
	fmt.Printf("   Valid:            %s - %s\n", info.ValidFrom.Format("2006-01-02"), info.ValidUntil.Format("2006-01-02"))
	fmt.Printf("   Location:         %s\n", outputDir)
	fmt.Printf("\n")
	fmt.Printf("⚠️  These are self-signed certificates for autonomous operation.\n")
	fmt.Printf("   To connect to a control server, replace with CA-signed certificates:\n")
	fmt.Printf("   - Use: ocserv-agent gencert --ca /path/to/server-ca.crt\n")
	fmt.Printf("\n")

	return nil
}
