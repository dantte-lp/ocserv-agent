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
	systemctl       *SystemctlManager
	occtl           *OcctlManager
	configReader    *ConfigReader
	allowedCommands map[string]bool
	logger          zerolog.Logger
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

	// Create config reader
	configReader := NewConfigReader(logger)

	// Build allowed commands map
	allowedMap := make(map[string]bool)
	for _, cmd := range cfg.Security.AllowedCommands {
		allowedMap[cmd] = true
	}

	return &Manager{
		systemctl:       systemctl,
		occtl:           occtl,
		configReader:    configReader,
		allowedCommands: allowedMap,
		logger:          logger,
	}
}

// Occtl returns the underlying OcctlManager for direct access to occtl methods
func (m *Manager) Occtl() *OcctlManager {
	return m.occtl
}

// CommandResult represents the result of a command execution
type CommandResult struct {
	Success  bool
	Stdout   string
	Stderr   string
	ExitCode int
	ErrorMsg string
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

		case "user":
			if len(args) < 3 {
				return &CommandResult{
					Success:  false,
					ErrorMsg: "show user requires username",
				}, fmt.Errorf("show user requires username")
			}

			users, err := m.occtl.ShowUser(ctx, args[2])
			if err != nil {
				return &CommandResult{
					Success:  false,
					ErrorMsg: err.Error(),
				}, err
			}

			if len(users) == 0 {
				return &CommandResult{
					Success: true,
					Stdout:  fmt.Sprintf("No active sessions found for user: %s", args[2]),
				}, nil
			}

			// Format output for multiple sessions
			var output strings.Builder
			output.WriteString(fmt.Sprintf("User: %s (%d session(s))\n", args[2], len(users)))
			for i, user := range users {
				output.WriteString(fmt.Sprintf("\nSession %d:\n", i+1))
				output.WriteString(fmt.Sprintf("  ID: %d\n", user.ID))
				output.WriteString(fmt.Sprintf("  State: %s\n", user.State))
				output.WriteString(fmt.Sprintf("  Device: %s\n", user.Device))
				output.WriteString(fmt.Sprintf("  Remote IP: %s\n", user.RemoteIP))
				output.WriteString(fmt.Sprintf("  VPN IPv4: %s\n", user.IPv4))
			}

			return &CommandResult{
				Success: true,
				Stdout:  output.String(),
			}, nil

		case "id":
			if len(args) < 3 {
				return &CommandResult{
					Success:  false,
					ErrorMsg: "show id requires connection ID",
				}, fmt.Errorf("show id requires connection ID")
			}

			conn, err := m.occtl.ShowID(ctx, args[2])
			if err != nil {
				return &CommandResult{
					Success:  false,
					ErrorMsg: err.Error(),
				}, err
			}

			return &CommandResult{
				Success: true,
				Stdout: fmt.Sprintf("Connection ID: %d\nUser: %s\nState: %s\nDevice: %s\nRemote IP: %s",
					conn.ID, conn.Username, conn.State, conn.Device, conn.RemoteIP),
			}, nil

		case "sessions":
			if len(args) < 3 {
				return &CommandResult{
					Success:  false,
					ErrorMsg: "show sessions requires 'all' or 'valid'",
				}, fmt.Errorf("show sessions requires 'all' or 'valid'")
			}

			switch args[2] {
			case "all":
				sessions, err := m.occtl.ShowSessionsAll(ctx)
				if err != nil {
					return &CommandResult{
						Success:  false,
						ErrorMsg: err.Error(),
					}, err
				}

				return &CommandResult{
					Success: true,
					Stdout:  fmt.Sprintf("Total sessions: %d", len(sessions)),
				}, nil

			case "valid":
				sessions, err := m.occtl.ShowSessionsValid(ctx)
				if err != nil {
					return &CommandResult{
						Success:  false,
						ErrorMsg: err.Error(),
					}, err
				}

				return &CommandResult{
					Success: true,
					Stdout:  fmt.Sprintf("Valid sessions: %d", len(sessions)),
				}, nil

			default:
				return &CommandResult{
					Success:  false,
					ErrorMsg: fmt.Sprintf("unknown sessions filter: %s", args[2]),
				}, fmt.Errorf("unknown sessions filter: %s", args[2])
			}

		case "session":
			if len(args) < 3 {
				return &CommandResult{
					Success:  false,
					ErrorMsg: "show session requires session ID",
				}, fmt.Errorf("show session requires session ID")
			}

			session, err := m.occtl.ShowSession(ctx, args[2])
			if err != nil {
				return &CommandResult{
					Success:  false,
					ErrorMsg: err.Error(),
				}, err
			}

			return &CommandResult{
				Success: true,
				Stdout: fmt.Sprintf("Session: %s\nUser: %s\nState: %s\nRemote IP: %s",
					session.Session, session.Username, session.State, session.RemoteIP),
			}, nil

		case "iroutes":
			iroutes, err := m.occtl.ShowIRoutes(ctx)
			if err != nil {
				return &CommandResult{
					Success:  false,
					ErrorMsg: err.Error(),
				}, err
			}

			return &CommandResult{
				Success: true,
				Stdout:  fmt.Sprintf("User routes: %d entries", len(iroutes)),
			}, nil

		case "ip":
			if len(args) < 3 {
				return &CommandResult{
					Success:  false,
					ErrorMsg: "show ip requires 'bans' or 'ban points'",
				}, fmt.Errorf("show ip requires 'bans' or 'ban points'")
			}

			switch args[2] {
			case "bans":
				bans, err := m.occtl.ShowIPBans(ctx)
				if err != nil {
					return &CommandResult{
						Success:  false,
						ErrorMsg: err.Error(),
					}, err
				}

				return &CommandResult{
					Success: true,
					Stdout:  fmt.Sprintf("Banned IPs: %d", len(bans)),
				}, nil

			case "ban":
				if len(args) >= 4 && args[3] == "points" {
					points, err := m.occtl.ShowIPBanPoints(ctx)
					if err != nil {
						return &CommandResult{
							Success:  false,
							ErrorMsg: err.Error(),
						}, err
					}

					return &CommandResult{
						Success: true,
						Stdout:  fmt.Sprintf("IPs with ban points: %d", len(points)),
					}, nil
				}

				return &CommandResult{
					Success:  false,
					ErrorMsg: "show ip ban requires 'points'",
				}, fmt.Errorf("show ip ban requires 'points'")

			default:
				return &CommandResult{
					Success:  false,
					ErrorMsg: fmt.Sprintf("unknown ip subcommand: %s", args[2]),
				}, fmt.Errorf("unknown ip subcommand: %s", args[2])
			}

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

	case "unban":
		if subcommand != "ip" {
			return &CommandResult{
				Success:  false,
				ErrorMsg: "unban requires 'ip' subcommand",
			}, fmt.Errorf("unban requires 'ip' subcommand")
		}

		if len(args) < 3 {
			return &CommandResult{
				Success:  false,
				ErrorMsg: "unban ip requires IP address",
			}, fmt.Errorf("unban ip requires IP address")
		}

		err := m.occtl.UnbanIP(ctx, args[2])
		if err != nil {
			return &CommandResult{
				Success:  false,
				ErrorMsg: err.Error(),
			}, err
		}

		return &CommandResult{
			Success: true,
			Stdout:  fmt.Sprintf("Unbanned IP: %s", args[2]),
		}, nil

	case "reload":
		err := m.occtl.Reload(ctx)
		if err != nil {
			return &CommandResult{
				Success:  false,
				ErrorMsg: err.Error(),
			}, err
		}

		return &CommandResult{
			Success: true,
			Stdout:  "Configuration reloaded successfully",
		}, nil

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
		regexp.MustCompile(`[;&|><\$\(\)\{\}\[\]]`),              // Shell metacharacters
		regexp.MustCompile("`"),                                  // Backtick command substitution
		regexp.MustCompile(`\\[;&|><\$\(\)\{\}\[\]` + "`" + `]`), // Escaped metacharacters
		regexp.MustCompile(`[\n\r]`),                             // Newline injection
		regexp.MustCompile(`[\x00-\x08\x0B\x0C\x0E-\x1F]`),       // Control characters (except \t and \n)
		regexp.MustCompile(`\.\./`),                              // Directory traversal
		regexp.MustCompile(`^-`),                                 // Flags starting with dash (potential flag injection)
	}

	for _, arg := range args {
		// Check for dangerous patterns
		for _, pattern := range dangerousPatterns {
			if pattern.MatchString(arg) {
				return fmt.Errorf("argument contains dangerous characters: %s", arg)
			}
		}

		// Check for null bytes (redundant with control chars but explicit)
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

// GetConfigReader returns the config reader (for direct access if needed)
func (m *Manager) GetConfigReader() *ConfigReader {
	return m.configReader
}

// ReadOcservConf reads the main ocserv configuration file
func (m *Manager) ReadOcservConf(ctx context.Context, path string) (*ConfigFile, error) {
	return m.configReader.ReadOcservConf(ctx, path)
}

// ReadUserConfig reads a per-user configuration file
func (m *Manager) ReadUserConfig(ctx context.Context, baseDir, username string) (*ConfigFile, error) {
	return m.configReader.ReadUserConfig(ctx, baseDir, username)
}

// ReadGroupConfig reads a per-group configuration file
func (m *Manager) ReadGroupConfig(ctx context.Context, baseDir, groupname string) (*ConfigFile, error) {
	return m.configReader.ReadGroupConfig(ctx, baseDir, groupname)
}

// ListUserConfigs lists all available per-user configuration files
func (m *Manager) ListUserConfigs(ctx context.Context, baseDir string) ([]string, error) {
	return m.configReader.ListUserConfigs(ctx, baseDir)
}

// ListGroupConfigs lists all available per-group configuration files
func (m *Manager) ListGroupConfigs(ctx context.Context, baseDir string) ([]string, error) {
	return m.configReader.ListGroupConfigs(ctx, baseDir)
}
