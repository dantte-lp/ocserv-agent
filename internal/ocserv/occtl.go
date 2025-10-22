package ocserv

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// OcctlManager handles occtl operations
type OcctlManager struct {
	socketPath string
	sudoUser   string
	timeout    time.Duration
	logger     zerolog.Logger
}

// NewOcctlManager creates a new occtl manager
func NewOcctlManager(socketPath, sudoUser string, timeout time.Duration, logger zerolog.Logger) *OcctlManager {
	return &OcctlManager{
		socketPath: socketPath,
		sudoUser:   sudoUser,
		timeout:    timeout,
		logger:     logger,
	}
}

// User represents a connected VPN user
type User struct {
	ID          string
	Username    string
	GroupName   string
	IPAddress   string
	VPNIPv4     string
	VPNIPv6     string
	Device      string
	ConnectedAt time.Time
	Hostname    string
}

// ServerStatus represents ocserv status
type ServerStatus struct {
	Status      string
	SecMod      string
	Compression string
	Uptime      int64
}

// ServerStats represents ocserv statistics
type ServerStats struct {
	ActiveUsers     int
	TotalSessions   int64
	TotalBytesIn    uint64
	TotalBytesOut   uint64
	TLSDBSize       int
	TLSDBEntries    int
	IPLeaseDBSize   int
	IPLeaseDBEntries int
}

// ShowUsers retrieves list of connected users
func (m *OcctlManager) ShowUsers(ctx context.Context) ([]User, error) {
	m.logger.Debug().Msg("Getting connected users")

	stdout, stderr, err := m.execute(ctx, "show", "users")
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w (stderr: %s)", err, stderr)
	}

	// Parse output
	users := m.parseUsers(stdout)

	m.logger.Debug().Int("count", len(users)).Msg("Retrieved connected users")

	return users, nil
}

// ShowStatus retrieves server status
func (m *OcctlManager) ShowStatus(ctx context.Context) (*ServerStatus, error) {
	m.logger.Debug().Msg("Getting server status")

	stdout, stderr, err := m.execute(ctx, "show", "status")
	if err != nil {
		return nil, fmt.Errorf("failed to get status: %w (stderr: %s)", err, stderr)
	}

	// Parse output
	status := m.parseStatus(stdout)

	m.logger.Debug().
		Str("status", status.Status).
		Int64("uptime", status.Uptime).
		Msg("Retrieved server status")

	return status, nil
}

// ShowStats retrieves server statistics
func (m *OcctlManager) ShowStats(ctx context.Context) (*ServerStats, error) {
	m.logger.Debug().Msg("Getting server statistics")

	stdout, stderr, err := m.execute(ctx, "show", "stats")
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w (stderr: %s)", err, stderr)
	}

	// Parse output
	stats := m.parseStats(stdout)

	m.logger.Debug().
		Int("active_users", stats.ActiveUsers).
		Uint64("bytes_in", stats.TotalBytesIn).
		Uint64("bytes_out", stats.TotalBytesOut).
		Msg("Retrieved server statistics")

	return stats, nil
}

// DisconnectUser disconnects a user by username
func (m *OcctlManager) DisconnectUser(ctx context.Context, username string) error {
	m.logger.Info().Str("username", username).Msg("Disconnecting user")

	_, stderr, err := m.execute(ctx, "disconnect", "user", username)
	if err != nil {
		return fmt.Errorf("failed to disconnect user %s: %w (stderr: %s)", username, err, stderr)
	}

	m.logger.Info().Str("username", username).Msg("User disconnected")

	return nil
}

// DisconnectID disconnects a user by session ID
func (m *OcctlManager) DisconnectID(ctx context.Context, id string) error {
	m.logger.Info().Str("id", id).Msg("Disconnecting session")

	_, stderr, err := m.execute(ctx, "disconnect", "id", id)
	if err != nil {
		return fmt.Errorf("failed to disconnect session %s: %w (stderr: %s)", id, err, stderr)
	}

	m.logger.Info().Str("id", id).Msg("Session disconnected")

	return nil
}

// execute runs an occtl command and captures output
func (m *OcctlManager) execute(ctx context.Context, args ...string) (string, string, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()

	// Build command
	var cmd *exec.Cmd
	if m.sudoUser != "" {
		// Run with sudo
		cmdArgs := []string{"sudo", "-n", "occtl"}
		if m.socketPath != "" {
			cmdArgs = append(cmdArgs, "-s", m.socketPath)
		}
		cmdArgs = append(cmdArgs, args...)
		cmd = exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)
	} else {
		cmdArgs := []string{"occtl"}
		if m.socketPath != "" {
			cmdArgs = append(cmdArgs, "-s", m.socketPath)
		}
		cmdArgs = append(cmdArgs, args...)
		cmd = exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)
	}

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute
	m.logger.Debug().
		Str("command", "occtl").
		Strs("args", args).
		Msg("Executing occtl command")

	err := cmd.Run()

	stdoutStr := stdout.String()
	stderrStr := stderr.String()

	if err != nil {
		m.logger.Warn().
			Err(err).
			Str("stdout", stdoutStr).
			Str("stderr", stderrStr).
			Strs("args", args).
			Msg("occtl command failed")
	}

	return stdoutStr, stderrStr, err
}

// parseUsers parses 'occtl show users' output
func (m *OcctlManager) parseUsers(output string) []User {
	var users []User

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "id") {
			// Skip empty lines and header
			continue
		}

		// Example format:
		// id: 1
		//     username: testuser
		//     groupname: users
		//     ip: 192.168.1.100
		//     vpn-ipv4: 10.10.10.2
		//     device: vpns0

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		key := strings.TrimSuffix(fields[0], ":")
		value := strings.Join(fields[1:], " ")

		// This is a simplified parser - real implementation would need
		// to properly track which user we're building
		// For now, create a basic user object
		if key == "id" {
			user := User{ID: value}
			users = append(users, user)
		}
	}

	return users
}

// parseStatus parses 'occtl show status' output
func (m *OcctlManager) parseStatus(output string) *ServerStatus {
	status := &ServerStatus{}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.Contains(line, ":") {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "Status":
			status.Status = value
		case "Sec-mod":
			status.SecMod = value
		case "Compression":
			status.Compression = value
		case "Uptime":
			// Parse uptime (e.g., "1234 seconds")
			if fields := strings.Fields(value); len(fields) > 0 {
				uptime, _ := strconv.ParseInt(fields[0], 10, 64)
				status.Uptime = uptime
			}
		}
	}

	return status
}

// parseStats parses 'occtl show stats' output
func (m *OcctlManager) parseStats(output string) *ServerStats {
	stats := &ServerStats{}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.Contains(line, ":") {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "Active users":
			stats.ActiveUsers, _ = strconv.Atoi(value)
		case "Total sessions":
			stats.TotalSessions, _ = strconv.ParseInt(value, 10, 64)
		case "Total bytes in":
			stats.TotalBytesIn, _ = strconv.ParseUint(value, 10, 64)
		case "Total bytes out":
			stats.TotalBytesOut, _ = strconv.ParseUint(value, 10, 64)
		case "TLS-DB size":
			stats.TLSDBSize, _ = strconv.Atoi(value)
		case "TLS-DB entries":
			stats.TLSDBEntries, _ = strconv.Atoi(value)
		case "IP-lease-DB size":
			stats.IPLeaseDBSize, _ = strconv.Atoi(value)
		case "IP-lease-DB entries":
			stats.IPLeaseDBEntries, _ = strconv.Atoi(value)
		}
	}

	return stats
}

// MarshalJSON for ServerStats to handle large numbers
func (s *ServerStats) MarshalJSON() ([]byte, error) {
	type Alias ServerStats
	return json.Marshal(&struct {
		*Alias
		TotalBytesInStr  string `json:"total_bytes_in_str"`
		TotalBytesOutStr string `json:"total_bytes_out_str"`
	}{
		Alias:            (*Alias)(s),
		TotalBytesInStr:  fmt.Sprintf("%d", s.TotalBytesIn),
		TotalBytesOutStr: fmt.Sprintf("%d", s.TotalBytesOut),
	})
}
