package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// Validate checks if the configuration is valid
func Validate(cfg *Config) error {
	var errs []error

	// Agent ID is required
	if cfg.AgentID == "" {
		errs = append(errs, errors.New("agent_id is required"))
	}

	// Control server address is required
	if cfg.ControlServer.Address == "" {
		errs = append(errs, errors.New("control_server.address is required"))
	}

	// Validate TLS config
	if err := validateTLS(&cfg.TLS); err != nil {
		errs = append(errs, fmt.Errorf("tls: %w", err))
	}

	// Validate ocserv config
	if err := validateOcserv(&cfg.Ocserv); err != nil {
		errs = append(errs, fmt.Errorf("ocserv: %w", err))
	}

	// Validate health config
	if err := validateHealth(&cfg.Health); err != nil {
		errs = append(errs, fmt.Errorf("health: %w", err))
	}

	// Validate logging config
	if err := validateLogging(&cfg.Logging); err != nil {
		errs = append(errs, fmt.Errorf("logging: %w", err))
	}

	// Validate security config
	if err := validateSecurity(&cfg.Security); err != nil {
		errs = append(errs, fmt.Errorf("security: %w", err))
	}

	// Validate reconnect config
	if err := validateReconnect(&cfg.ControlServer.Reconnect); err != nil {
		errs = append(errs, fmt.Errorf("control_server.reconnect: %w", err))
	}

	// Return combined errors
	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// validateTLS checks TLS configuration
func validateTLS(tls *TLSConfig) error {
	if !tls.Enabled {
		return nil
	}

	var errs []error

	// Certificate files must exist
	if tls.CertFile == "" {
		errs = append(errs, errors.New("cert_file is required when TLS is enabled"))
	} else if _, err := os.Stat(tls.CertFile); err != nil {
		errs = append(errs, fmt.Errorf("cert_file not found: %s", tls.CertFile))
	}

	if tls.KeyFile == "" {
		errs = append(errs, errors.New("key_file is required when TLS is enabled"))
	} else if _, err := os.Stat(tls.KeyFile); err != nil {
		errs = append(errs, fmt.Errorf("key_file not found: %s", tls.KeyFile))
	}

	if tls.CAFile == "" {
		errs = append(errs, errors.New("ca_file is required when TLS is enabled"))
	} else if _, err := os.Stat(tls.CAFile); err != nil {
		errs = append(errs, fmt.Errorf("ca_file not found: %s", tls.CAFile))
	}

	// Validate TLS version
	validVersions := []string{"TLS1.2", "TLS1.3"}
	valid := false
	for _, v := range validVersions {
		if tls.MinVersion == v {
			valid = true
			break
		}
	}
	if !valid {
		errs = append(errs, fmt.Errorf("invalid min_version: %s (must be one of: %s)",
			tls.MinVersion, strings.Join(validVersions, ", ")))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// validateOcserv checks ocserv configuration
func validateOcserv(ocserv *OcservConfig) error {
	var errs []error

	if ocserv.ConfigPath == "" {
		errs = append(errs, errors.New("config_path is required"))
	}

	if ocserv.CtlSocket == "" {
		errs = append(errs, errors.New("ctl_socket is required"))
	}

	if ocserv.SystemdService == "" {
		errs = append(errs, errors.New("systemd_service is required"))
	}

	if ocserv.BackupDir == "" {
		errs = append(errs, errors.New("backup_dir is required"))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// validateHealth checks health configuration
func validateHealth(health *HealthConfig) error {
	var errs []error

	if health.HeartbeatInterval <= 0 {
		errs = append(errs, errors.New("heartbeat_interval must be > 0"))
	}

	if health.DeepCheckInterval <= 0 {
		errs = append(errs, errors.New("deep_check_interval must be > 0"))
	}

	if health.MetricsInterval <= 0 {
		errs = append(errs, errors.New("metrics_interval must be > 0"))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// validateLogging checks logging configuration
func validateLogging(logging *LoggingConfig) error {
	var errs []error

	// Validate log level
	validLevels := []string{"debug", "info", "warn", "error"}
	valid := false
	for _, l := range validLevels {
		if logging.Level == l {
			valid = true
			break
		}
	}
	if !valid {
		errs = append(errs, fmt.Errorf("invalid level: %s (must be one of: %s)",
			logging.Level, strings.Join(validLevels, ", ")))
	}

	// Validate format
	validFormats := []string{"json", "text"}
	valid = false
	for _, f := range validFormats {
		if logging.Format == f {
			valid = true
			break
		}
	}
	if !valid {
		errs = append(errs, fmt.Errorf("invalid format: %s (must be one of: %s)",
			logging.Format, strings.Join(validFormats, ", ")))
	}

	// Validate output
	validOutputs := []string{"stdout", "file"}
	valid = false
	for _, o := range validOutputs {
		if logging.Output == o {
			valid = true
			break
		}
	}
	if !valid {
		errs = append(errs, fmt.Errorf("invalid output: %s (must be one of: %s)",
			logging.Output, strings.Join(validOutputs, ", ")))
	}

	// If output is file, file_path is required
	if logging.Output == "file" && logging.FilePath == "" {
		errs = append(errs, errors.New("file_path is required when output is 'file'"))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// validateSecurity checks security configuration
func validateSecurity(security *SecurityConfig) error {
	var errs []error

	if len(security.AllowedCommands) == 0 {
		errs = append(errs, errors.New("allowed_commands cannot be empty"))
	}

	if security.MaxCommandTimeout <= 0 {
		errs = append(errs, errors.New("max_command_timeout must be > 0"))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// validateReconnect checks reconnect configuration
func validateReconnect(reconnect *ReconnectConfig) error {
	var errs []error

	if reconnect.InitialDelay <= 0 {
		errs = append(errs, errors.New("initial_delay must be > 0"))
	}

	if reconnect.MaxDelay <= 0 {
		errs = append(errs, errors.New("max_delay must be > 0"))
	}

	if reconnect.MaxDelay < reconnect.InitialDelay {
		errs = append(errs, errors.New("max_delay must be >= initial_delay"))
	}

	if reconnect.Multiplier <= 1.0 {
		errs = append(errs, errors.New("multiplier must be > 1.0"))
	}

	if reconnect.MaxAttempts <= 0 {
		errs = append(errs, errors.New("max_attempts must be > 0"))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}
