package ocserv

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// SystemctlManager handles systemctl operations for ocserv service
type SystemctlManager struct {
	serviceName string
	sudoUser    string
	timeout     time.Duration
	logger      zerolog.Logger
}

// NewSystemctlManager creates a new systemctl manager
func NewSystemctlManager(serviceName, sudoUser string, timeout time.Duration, logger zerolog.Logger) *SystemctlManager {
	return &SystemctlManager{
		serviceName: serviceName,
		sudoUser:    sudoUser,
		timeout:     timeout,
		logger:      logger,
	}
}

// ServiceStatus represents the status of a systemd service
type ServiceStatus struct {
	Active      bool
	State       string // "running", "dead", "failed", etc.
	SubState    string // "running", "exited", etc.
	Description string
	MainPID     int
	LoadState   string // "loaded", "not-found", etc.
}

// Start starts the ocserv service
func (m *SystemctlManager) Start(ctx context.Context) error {
	m.logger.Info().Str("service", m.serviceName).Msg("Starting service")

	return m.execute(ctx, "start")
}

// Stop stops the ocserv service
func (m *SystemctlManager) Stop(ctx context.Context) error {
	m.logger.Info().Str("service", m.serviceName).Msg("Stopping service")

	return m.execute(ctx, "stop")
}

// Restart restarts the ocserv service
func (m *SystemctlManager) Restart(ctx context.Context) error {
	m.logger.Info().Str("service", m.serviceName).Msg("Restarting service")

	return m.execute(ctx, "restart")
}

// Reload reloads the ocserv service configuration
func (m *SystemctlManager) Reload(ctx context.Context) error {
	m.logger.Info().Str("service", m.serviceName).Msg("Reloading service")

	return m.execute(ctx, "reload")
}

// Status gets the current status of the ocserv service
func (m *SystemctlManager) Status(ctx context.Context) (*ServiceStatus, error) {
	m.logger.Debug().Str("service", m.serviceName).Msg("Getting service status")

	// Use 'systemctl show' for machine-readable output
	stdout, stderr, err := m.executeWithOutput(ctx, "show", m.serviceName)

	// Note: systemctl show returns 0 even if service doesn't exist,
	// so we need to check the output
	status := &ServiceStatus{}

	// Parse the output
	lines := strings.Split(stdout, "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "ActiveState":
			status.State = value
			status.Active = (value == "active")
		case "SubState":
			status.SubState = value
		case "Description":
			status.Description = value
		case "MainPID":
			fmt.Sscanf(value, "%d", &status.MainPID)
		case "LoadState":
			status.LoadState = value
		}
	}

	// Check if service exists
	if status.LoadState == "not-found" {
		return nil, fmt.Errorf("service %s not found", m.serviceName)
	}

	if err != nil {
		m.logger.Warn().
			Err(err).
			Str("stderr", stderr).
			Msg("Error getting service status")
		return status, fmt.Errorf("failed to get service status: %w", err)
	}

	m.logger.Debug().
		Bool("active", status.Active).
		Str("state", status.State).
		Str("substate", status.SubState).
		Msg("Service status retrieved")

	return status, nil
}

// IsActive checks if the service is active
func (m *SystemctlManager) IsActive(ctx context.Context) (bool, error) {
	stdout, _, err := m.executeWithOutput(ctx, "is-active", m.serviceName)

	// is-active returns exit code 0 if active, non-zero otherwise
	// stdout will be "active" or "inactive"/"failed"/etc.
	active := strings.TrimSpace(stdout) == "active"

	if err != nil && !active {
		// This is expected if service is not active
		return false, nil
	}

	return active, nil
}

// IsEnabled checks if the service is enabled
func (m *SystemctlManager) IsEnabled(ctx context.Context) (bool, error) {
	stdout, _, err := m.executeWithOutput(ctx, "is-enabled", m.serviceName)

	// is-enabled returns exit code 0 if enabled
	// stdout will be "enabled", "disabled", "static", etc.
	enabled := strings.TrimSpace(stdout) == "enabled"

	if err != nil && !enabled {
		// This is expected if service is not enabled
		return false, nil
	}

	return enabled, nil
}

// execute runs a systemctl command without capturing output
func (m *SystemctlManager) execute(ctx context.Context, action string) error {
	_, _, err := m.executeWithOutput(ctx, action, m.serviceName)
	return err
}

// executeWithOutput runs a systemctl command and captures output
func (m *SystemctlManager) executeWithOutput(ctx context.Context, args ...string) (string, string, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()

	// Build command
	var cmd *exec.Cmd
	if m.sudoUser != "" {
		// Run with sudo
		cmdArgs := []string{"sudo", "-n", "systemctl"}
		cmdArgs = append(cmdArgs, args...)
		cmd = exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)
	} else {
		cmd = exec.CommandContext(ctx, "systemctl", args...)
	}

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute
	m.logger.Debug().
		Str("command", "systemctl").
		Strs("args", args).
		Msg("Executing systemctl command")

	err := cmd.Run()

	stdoutStr := stdout.String()
	stderrStr := stderr.String()

	if err != nil {
		m.logger.Warn().
			Err(err).
			Str("stdout", stdoutStr).
			Str("stderr", stderrStr).
			Strs("args", args).
			Msg("systemctl command failed")
	}

	return stdoutStr, stderrStr, err
}
