package ocserv

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/dantte-lp/ocserv-agent/internal/config"
	"github.com/rs/zerolog"
)

// Manager provides high-level ocserv management with security
type Manager struct {
	systemctl      *SystemctlManager
	occtl          *OcctlManager
	allowedCommands map[string]bool
	logger         zerolog.Logger
}

// NewManager creates a new ocserv manager
func NewManager(cfg *config.Config, logger zerolog.Logger) *Manager {
	// Create systemctl manager
	systemctl := NewSystemctlManager(
		cfg.Ocserv.SystemdService,
		cfg.Security.SudoUser,
		cfg.Security.MaxCommandTimeout,
		logger,
	)

	// Create occtl manager
	occtl := NewOcctlManager(
		cfg.Ocserv.CtlSocket,
		cfg.Security.SudoUser,
		cfg.Security.MaxCommandTimeout,
		logger,
	)

	// Build allowed commands map
	allowedMap := make(map[string]bool)
	for _, cmd := range cfg.Security.AllowedCommands {
		allowedMap[cmd] = true
	}

	return &Manager{
		systemctl:       systemctl,
		occtl:           occtl,
		allowedCommands: allowedMap,
		logger:          logger,
	}
}

// CommandResult represents the result of a command execution
type CommandResult struct {
	Success   bool
	Stdout    string
	Stderr    string
	ExitCode  int
	ErrorMsg  string
}

// ExecuteCommand executes a validated command
func (m *Manager) ExecuteCommand(ctx context.Context, commandType string, args []string) (*CommandResult, error) {
	// Validate command is allowed
	if !m.isCommandAllowed(commandType) {
		return &CommandResult{
			Success:  false,
			ErrorMsg: fmt.Sprintf("command not allowed: %s", commandType),
		}, fmt.Errorf("command not allowed: %s", commandType)
	}

	// Validate arguments
	if err := m.validateArguments(args); err != nil {
		return &CommandResult{
			Success:  false,
			ErrorMsg: fmt.Sprintf("invalid arguments: %v", err),
		}, fmt.Errorf("invalid arguments: %w", err)
	}

	m.logger.Info().
		Str("command", commandType).
		Strs("args", args).
		Msg("Executing command")

	// Route to appropriate handler
	switch commandType {
	case "systemctl":
		return m.executeSystemctl(ctx, args)
	case "occtl":
		return m.executeOcctl(ctx, args)
	default:
		return &CommandResult{
			Success:  false,
			ErrorMsg: fmt.Sprintf("unknown command type: %s", commandType),
		}, fmt.Errorf("unknown command type: %s", commandType)
	}
}

// executeSystemctl executes systemctl commands
func (m *Manager) executeSystemctl(ctx context.Context, args []string) (*CommandResult, error) {
	if len(args) == 0 {
		return &CommandResult{
			Success:  false,
			ErrorMsg: "systemctl requires action argument",
		}, fmt.Errorf("systemctl requires action argument")
	}

	action := args[0]

	var err error

	switch action {
	case "start":
		err = m.systemctl.Start(ctx)

	case "stop":
		err = m.systemctl.Stop(ctx)

	case "restart":
		err = m.systemctl.Restart(ctx)

	case "reload":
		err = m.systemctl.Reload(ctx)

	case "status":
		status, statusErr := m.systemctl.Status(ctx)
		if statusErr != nil {
			return &CommandResult{
				Success:  false,
				ErrorMsg: statusErr.Error(),
			}, statusErr
		}

		return &CommandResult{
			Success: true,
			Stdout: fmt.Sprintf("Active: %v\nState: %s\nSubState: %s\nDescription: %s\nMainPID: %d",
				status.Active, status.State, status.SubState, status.Description, status.MainPID),
		}, nil

	case "is-active":
		active, activeErr := m.systemctl.IsActive(ctx)
		if activeErr != nil {
			return &CommandResult{
				Success:  false,
				ErrorMsg: activeErr.Error(),
			}, activeErr
		}

		state := "inactive"
		if active {
			state = "active"
		}

		return &CommandResult{
			Success: true,
			Stdout:  state,
		}, nil

	case "is-enabled":
		enabled, enabledErr := m.systemctl.IsEnabled(ctx)
		if enabledErr != nil {
			return &CommandResult{
				Success:  false,
				ErrorMsg: enabledErr.Error(),
			}, enabledErr
		}

		state := "disabled"
		if enabled {
			state = "enabled"
		}

		return &CommandResult{
			Success: true,
			Stdout:  state,
		}, nil

	default:
		return &CommandResult{
			Success:  false,
			ErrorMsg: fmt.Sprintf("unknown systemctl action: %s", action),
		}, fmt.Errorf("unknown systemctl action: %s", action)
	}

	if err != nil {
		return &CommandResult{
			Success:  false,
			ErrorMsg: err.Error(),
		}, err
	}

	return &CommandResult{
		Success: true,
		Stdout:  fmt.Sprintf("systemctl %s completed successfully", action),
	}, nil
}

// executeOcctl executes occtl commands
func (m *Manager) executeOcctl(ctx context.Context, args []string) (*CommandResult, error) {
	if len(args) < 2 {
		return &CommandResult{
			Success:  false,
			ErrorMsg: "occtl requires action and subcommand",
		}, fmt.Errorf("occtl requires action and subcommand")
	}

	action := args[0]
	subcommand := args[1]

	switch action {
	case "show":
		switch subcommand {
		case "users":
			users, err := m.occtl.ShowUsers(ctx)
			if err != nil {
				return &CommandResult{
					Success:  false,
					ErrorMsg: err.Error(),
				}, err
			}

			return &CommandResult{
				Success: true,
				Stdout:  fmt.Sprintf("Connected users: %d", len(users)),
			}, nil

		case "status":
			status, err := m.occtl.ShowStatus(ctx)
			if err != nil {
				return &CommandResult{
					Success:  false,
					ErrorMsg: err.Error(),
				}, err
			}

			return &CommandResult{
				Success: true,
				Stdout: fmt.Sprintf("Status: %s\nSec-mod: %s\nCompression: %s\nUptime: %d seconds",
					status.Status, status.SecMod, status.Compression, status.Uptime),
			}, nil

		case "stats":
			stats, err := m.occtl.ShowStats(ctx)
			if err != nil {
				return &CommandResult{
					Success:  false,
					ErrorMsg: err.Error(),
				}, err
			}

			return &CommandResult{
				Success: true,
				Stdout: fmt.Sprintf("Active users: %d\nTotal sessions: %d\nBytes in: %d\nBytes out: %d",
					stats.ActiveUsers, stats.TotalSessions, stats.TotalBytesIn, stats.TotalBytesOut),
			}, nil

		default:
			return &CommandResult{
				Success:  false,
				ErrorMsg: fmt.Sprintf("unknown occtl show subcommand: %s", subcommand),
			}, fmt.Errorf("unknown occtl show subcommand: %s", subcommand)
		}

	case "disconnect":
		if len(args) < 3 {
			return &CommandResult{
				Success:  false,
				ErrorMsg: "disconnect requires user or id and value",
			}, fmt.Errorf("disconnect requires user or id and value")
		}

		target := args[2]

		switch subcommand {
		case "user":
			err := m.occtl.DisconnectUser(ctx, target)
			if err != nil {
				return &CommandResult{
					Success:  false,
					ErrorMsg: err.Error(),
				}, err
			}

			return &CommandResult{
				Success: true,
				Stdout:  fmt.Sprintf("Disconnected user: %s", target),
			}, nil

		case "id":
			err := m.occtl.DisconnectID(ctx, target)
			if err != nil {
				return &CommandResult{
					Success:  false,
					ErrorMsg: err.Error(),
				}, err
			}

			return &CommandResult{
				Success: true,
				Stdout:  fmt.Sprintf("Disconnected session: %s", target),
			}, nil

		default:
			return &CommandResult{
				Success:  false,
				ErrorMsg: fmt.Sprintf("unknown disconnect target: %s", subcommand),
			}, fmt.Errorf("unknown disconnect target: %s", subcommand)
		}

	default:
		return &CommandResult{
			Success:  false,
			ErrorMsg: fmt.Sprintf("unknown occtl action: %s", action),
		}, fmt.Errorf("unknown occtl action: %s", action)
	}
}

// isCommandAllowed checks if a command is in the whitelist
func (m *Manager) isCommandAllowed(command string) bool {
	return m.allowedCommands[command]
}

// validateArguments validates command arguments for safety
func (m *Manager) validateArguments(args []string) error {
	// Prevent command injection
	dangerousPatterns := []*regexp.Regexp{
		regexp.MustCompile(`[;&|><\$\(\)\{\}\[\]]`),  // Shell metacharacters
		regexp.MustCompile(`\.\./`),                   // Directory traversal
		regexp.MustCompile(`^-`),                      // Flags starting with dash (potential flag injection)
	}

	for _, arg := range args {
		// Check for dangerous patterns
		for _, pattern := range dangerousPatterns {
			if pattern.MatchString(arg) {
				return fmt.Errorf("argument contains dangerous characters: %s", arg)
			}
		}

		// Check for null bytes
		if strings.Contains(arg, "\x00") {
			return fmt.Errorf("argument contains null bytes")
		}

		// Check length
		if len(arg) > 256 {
			return fmt.Errorf("argument too long (max 256 characters): %s", arg)
		}
	}

	return nil
}

// GetSystemctlManager returns the systemctl manager (for direct access if needed)
func (m *Manager) GetSystemctlManager() *SystemctlManager {
	return m.systemctl
}

// GetOcctlManager returns the occtl manager (for direct access if needed)
func (m *Manager) GetOcctlManager() *OcctlManager {
	return m.occtl
}
