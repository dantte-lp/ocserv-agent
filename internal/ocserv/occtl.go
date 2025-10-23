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
	ID                int         `json:"ID"`
	Username          string      `json:"Username"`
	Groupname         string      `json:"Groupname"`
	State             string      `json:"State"`
	VHost             string      `json:"vhost"`
	Device            string      `json:"Device"`
	MTU               string      `json:"MTU"`
	RemoteIP          string      `json:"Remote IP"`
	Location          string      `json:"Location"`
	LocalDeviceIP     string      `json:"Local Device IP"`
	IPv4              string      `json:"IPv4"`
	PtPIPv4           string      `json:"P-t-P IPv4"`
	IPv6              string      `json:"IPv6"`
	PtPIPv6           string      `json:"P-t-P IPv6"`
	UserAgent         string      `json:"User-Agent"`
	RX                string      `json:"RX"`
	TX                string      `json:"TX"`
	ReadableRX        string      `json:"_RX"`
	ReadableTX        string      `json:"_TX"`
	AverageRX         string      `json:"Average RX"`
	AverageTX         string      `json:"Average TX"`
	DPD               string      `json:"DPD"`
	KeepAlive         string      `json:"KeepAlive"`
	Hostname          string      `json:"Hostname"`
	ConnectedAt       string      `json:"Connected at"`
	ConnectedDuration string      `json:"_Connected at"`
	RawConnectedAt    int64       `json:"raw_connected_at"`
	FullSession       string      `json:"Full session"`
	Session           string      `json:"Session"`
	TLSCiphersuite    string      `json:"TLS ciphersuite"`
	DTLSCipher        string      `json:"DTLS cipher"`
	CSTPCompression   string      `json:"CSTP compression"`
	DTLSCompression   string      `json:"DTLS compression"`
	DNS               []string    `json:"DNS"`
	NBNS              []string    `json:"NBNS"`
	SplitDNSDomains   []string    `json:"Split-DNS-Domains"`
	Routes            interface{} `json:"Routes"` // Can be []string or string (e.g. "defaultroute")
	NoRoutes          []string    `json:"No-routes"`
	IRoutes           []string    `json:"iRoutes"`
	RestrictedRoutes  string      `json:"Restricted to routes"`
	RestrictedPorts   []string    `json:"Restricted to ports"`
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
	ActiveUsers      int
	TotalSessions    int64
	TotalBytesIn     uint64
	TotalBytesOut    uint64
	TLSDBSize        int
	TLSDBEntries     int
	IPLeaseDBSize    int
	IPLeaseDBEntries int
}

// ShowUsers retrieves list of connected users
func (m *OcctlManager) ShowUsers(ctx context.Context) ([]User, error) {
	m.logger.Debug().Msg("Getting connected users")

	// Use -j flag for JSON output
	stdout, stderr, err := m.executeJSON(ctx, "show", "users")
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w (stderr: %s)", err, stderr)
	}

	// Parse JSON output
	users, err := m.parseUsersJSON(stdout)
	if err != nil {
		return nil, fmt.Errorf("failed to parse users: %w", err)
	}

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

// ShowUser retrieves detailed information about a specific user
// Note: Returns array - multiple elements if user has multiple active sessions
func (m *OcctlManager) ShowUser(ctx context.Context, username string) ([]UserDetailed, error) {
	m.logger.Debug().Str("username", username).Msg("Getting user details")

	stdout, stderr, err := m.executeJSON(ctx, "show", "user", username)
	if err != nil {
		return nil, fmt.Errorf("failed to get user %s: %w (stderr: %s)", username, err, stderr)
	}

	// Parse JSON array (can contain multiple sessions for same user)
	var users []UserDetailed
	if err := json.Unmarshal([]byte(stdout), &users); err != nil {
		return nil, fmt.Errorf("failed to parse user details: %w", err)
	}

	m.logger.Debug().
		Str("username", username).
		Int("count", len(users)).
		Msg("Retrieved user details")

	return users, nil
}

// ShowID retrieves detailed information about a specific connection ID
func (m *OcctlManager) ShowID(ctx context.Context, id string) (*UserDetailed, error) {
	m.logger.Debug().Str("id", id).Msg("Getting connection details")

	stdout, stderr, err := m.executeJSON(ctx, "show", "id", id)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection %s: %w (stderr: %s)", id, err, stderr)
	}

	// Parse JSON array (single element expected)
	var users []UserDetailed
	if err := json.Unmarshal([]byte(stdout), &users); err != nil {
		return nil, fmt.Errorf("failed to parse connection details: %w", err)
	}

	if len(users) == 0 {
		return nil, fmt.Errorf("no connection found with ID %s", id)
	}

	user := users[0]

	m.logger.Debug().
		Int("id", user.ID).
		Str("username", user.Username).
		Str("state", user.State).
		Msg("Retrieved connection details")

	return &user, nil
}

// ShowSessionsAll retrieves all sessions
func (m *OcctlManager) ShowSessionsAll(ctx context.Context) ([]SessionInfo, error) {
	m.logger.Debug().Msg("Getting all sessions")

	stdout, stderr, err := m.executeJSON(ctx, "show", "sessions", "all")
	if err != nil {
		return nil, fmt.Errorf("failed to get sessions: %w (stderr: %s)", err, stderr)
	}

	var sessions []SessionInfo
	if err := json.Unmarshal([]byte(stdout), &sessions); err != nil {
		return nil, fmt.Errorf("failed to parse sessions: %w", err)
	}

	m.logger.Debug().Int("count", len(sessions)).Msg("Retrieved all sessions")

	return sessions, nil
}

// ShowSessionsValid retrieves valid (reconnectable) sessions
func (m *OcctlManager) ShowSessionsValid(ctx context.Context) ([]SessionInfo, error) {
	m.logger.Debug().Msg("Getting valid sessions")

	stdout, stderr, err := m.executeJSON(ctx, "show", "sessions", "valid")
	if err != nil {
		return nil, fmt.Errorf("failed to get valid sessions: %w (stderr: %s)", err, stderr)
	}

	var sessions []SessionInfo
	if err := json.Unmarshal([]byte(stdout), &sessions); err != nil {
		return nil, fmt.Errorf("failed to parse valid sessions: %w", err)
	}

	m.logger.Debug().Int("count", len(sessions)).Msg("Retrieved valid sessions")

	return sessions, nil
}

// ShowSession retrieves information about a specific session ID
func (m *OcctlManager) ShowSession(ctx context.Context, sessionID string) (*SessionInfo, error) {
	m.logger.Debug().Str("session_id", sessionID).Msg("Getting session details")

	stdout, stderr, err := m.executeJSON(ctx, "show", "session", sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session %s: %w (stderr: %s)", sessionID, err, stderr)
	}

	var session SessionInfo
	if err := json.Unmarshal([]byte(stdout), &session); err != nil {
		return nil, fmt.Errorf("failed to parse session details: %w", err)
	}

	m.logger.Debug().
		Str("session", session.Session).
		Str("username", session.Username).
		Str("state", session.State).
		Msg("Retrieved session details")

	return &session, nil
}

// ShowStatusDetailed retrieves detailed server status with all metrics
func (m *OcctlManager) ShowStatusDetailed(ctx context.Context) (*ServerStatusDetailed, error) {
	m.logger.Debug().Msg("Getting detailed server status")

	stdout, stderr, err := m.executeJSON(ctx, "show", "status")
	if err != nil {
		return nil, fmt.Errorf("failed to get detailed status: %w (stderr: %s)", err, stderr)
	}

	var status ServerStatusDetailed
	if err := json.Unmarshal([]byte(stdout), &status); err != nil {
		return nil, fmt.Errorf("failed to parse detailed status: %w", err)
	}

	m.logger.Debug().
		Str("status", status.Status).
		Int("active_sessions", status.ActiveSessions).
		Int64("uptime", status.Uptime).
		Msg("Retrieved detailed server status")

	return &status, nil
}

// ShowUsersDetailed retrieves detailed list of connected users with all information
func (m *OcctlManager) ShowUsersDetailed(ctx context.Context) ([]UserDetailed, error) {
	m.logger.Debug().Msg("Getting detailed connected users")

	stdout, stderr, err := m.executeJSON(ctx, "show", "users")
	if err != nil {
		return nil, fmt.Errorf("failed to get detailed users: %w (stderr: %s)", err, stderr)
	}

	var users []UserDetailed
	if err := json.Unmarshal([]byte(stdout), &users); err != nil {
		return nil, fmt.Errorf("failed to parse detailed users: %w", err)
	}

	m.logger.Debug().Int("count", len(users)).Msg("Retrieved detailed connected users")

	return users, nil
}

// ShowIRoutes retrieves user-provided routes for all connected users
func (m *OcctlManager) ShowIRoutes(ctx context.Context) ([]IRoute, error) {
	m.logger.Debug().Msg("Getting user routes")

	stdout, stderr, err := m.executeJSON(ctx, "show", "iroutes")
	if err != nil {
		return nil, fmt.Errorf("failed to get iroutes: %w (stderr: %s)", err, stderr)
	}

	var iroutes []IRoute
	if err := json.Unmarshal([]byte(stdout), &iroutes); err != nil {
		return nil, fmt.Errorf("failed to parse iroutes: %w", err)
	}

	m.logger.Debug().Int("count", len(iroutes)).Msg("Retrieved user routes")

	return iroutes, nil
}

// ShowIPBans retrieves list of banned IP addresses
func (m *OcctlManager) ShowIPBans(ctx context.Context) ([]IPBan, error) {
	m.logger.Debug().Msg("Getting banned IPs")

	stdout, stderr, err := m.executeJSON(ctx, "show", "ip", "bans")
	if err != nil {
		return nil, fmt.Errorf("failed to get IP bans: %w (stderr: %s)", err, stderr)
	}

	var bans []IPBan
	if err := json.Unmarshal([]byte(stdout), &bans); err != nil {
		return nil, fmt.Errorf("failed to parse IP bans: %w", err)
	}

	m.logger.Debug().Int("count", len(bans)).Msg("Retrieved banned IPs")

	return bans, nil
}

// ShowIPBanPoints retrieves IPs with accumulated violation points
func (m *OcctlManager) ShowIPBanPoints(ctx context.Context) ([]IPBanPoints, error) {
	m.logger.Debug().Msg("Getting IP ban points")

	stdout, stderr, err := m.executeJSON(ctx, "show", "ip", "ban", "points")
	if err != nil {
		return nil, fmt.Errorf("failed to get IP ban points: %w (stderr: %s)", err, stderr)
	}

	var points []IPBanPoints
	if err := json.Unmarshal([]byte(stdout), &points); err != nil {
		return nil, fmt.Errorf("failed to parse IP ban points: %w", err)
	}

	m.logger.Debug().Int("count", len(points)).Msg("Retrieved IP ban points")

	return points, nil
}

// UnbanIP removes an IP address from the ban list
func (m *OcctlManager) UnbanIP(ctx context.Context, ip string) error {
	m.logger.Info().Str("ip", ip).Msg("Unbanning IP address")

	_, stderr, err := m.execute(ctx, "unban", "ip", ip)
	if err != nil {
		return fmt.Errorf("failed to unban IP %s: %w (stderr: %s)", ip, err, stderr)
	}

	m.logger.Info().Str("ip", ip).Msg("IP address unbanned")

	return nil
}

// Reload sends reload signal to ocserv (handled via systemctl in Manager)
func (m *OcctlManager) Reload(ctx context.Context) error {
	m.logger.Info().Msg("Reloading ocserv configuration")

	_, stderr, err := m.execute(ctx, "reload")
	if err != nil {
		return fmt.Errorf("failed to reload: %w (stderr: %s)", err, stderr)
	}

	m.logger.Info().Msg("ocserv configuration reloaded")

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

// executeJSON runs an occtl command with JSON output flag (-j) and captures output
func (m *OcctlManager) executeJSON(ctx context.Context, args ...string) (string, string, error) {
	// Add -j flag for JSON output
	jsonArgs := append([]string{"-j"}, args...)
	return m.execute(ctx, jsonArgs...)
}

// parseUsers is deprecated, use parseUsersJSON instead
// This function is kept for backwards compatibility but is not used
func (m *OcctlManager) parseUsers(output string) []User {
	// Deprecated: This function parses text output, but we now use JSON
	// Return empty list as this should not be called
	m.logger.Warn().Msg("parseUsers (text mode) is deprecated, use JSON mode")
	return []User{}
}

// parseUsersJSON parses 'occtl -j show users' JSON output
func (m *OcctlManager) parseUsersJSON(output string) ([]User, error) {
	var users []User

	if err := json.Unmarshal([]byte(output), &users); err != nil {
		return nil, fmt.Errorf("failed to unmarshal users JSON: %w", err)
	}

	return users, nil
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
