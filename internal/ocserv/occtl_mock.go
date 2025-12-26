package ocserv

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
)

// MockOcctlManager is a mock implementation of OcctlInterface for testing
type MockOcctlManager struct {
	mu sync.RWMutex

	// Mock data
	users         []User
	serverStatus  *ServerStatus
	serverStats   *ServerStats
	disconnected  []string // Track disconnected users
	reloadCalled  bool
	stopCalled    bool

	// Error injection
	showUsersErr     error
	disconnectErr    error
	getStatusErr     error
	getStatsErr      error
	reloadErr        error
	stopErr          error
}

// NewMockOcctlManager creates a new mock occtl manager
func NewMockOcctlManager() *MockOcctlManager {
	return &MockOcctlManager{
		users:        make([]User, 0),
		disconnected: make([]string, 0),
		serverStatus: &ServerStatus{
			Status:      "online",
			SecMod:      "certificate",
			Compression: "true",
			Uptime:      3600,
		},
		serverStats: &ServerStats{
			ActiveUsers:   0,
			TotalSessions: 0,
		},
	}
}

// ShowUsers returns the mock user list
func (m *MockOcctlManager) ShowUsers(ctx context.Context) ([]User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.showUsersErr != nil {
		return nil, m.showUsersErr
	}

	// Filter out disconnected users
	activeUsers := make([]User, 0)
	for _, user := range m.users {
		disconnected := false
		for _, username := range m.disconnected {
			if user.Username == username {
				disconnected = true
				break
			}
		}
		if !disconnected {
			activeUsers = append(activeUsers, user)
		}
	}

	return activeUsers, nil
}

// DisconnectUser disconnects a user by username
func (m *MockOcctlManager) DisconnectUser(ctx context.Context, username string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.disconnectErr != nil {
		return m.disconnectErr
	}

	// Check if user exists
	found := false
	for _, user := range m.users {
		if user.Username == username {
			found = true
			break
		}
	}

	if !found {
		return errors.Newf("user not found: %s", username)
	}

	// Add to disconnected list
	m.disconnected = append(m.disconnected, username)

	return nil
}

// DisconnectID disconnects a user by session ID
func (m *MockOcctlManager) DisconnectID(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.disconnectErr != nil {
		return m.disconnectErr
	}

	// Find user by ID (convert string to int for comparison)
	var username string
	for _, user := range m.users {
		if fmt.Sprintf("%d", user.ID) == id {
			username = user.Username
			break
		}
	}

	if username == "" {
		return errors.Newf("user not found with ID: %s", id)
	}

	// Add to disconnected list
	m.disconnected = append(m.disconnected, username)

	return nil
}

// ShowStatus returns mock server status
func (m *MockOcctlManager) ShowStatus(ctx context.Context) (*ServerStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.getStatusErr != nil {
		return nil, m.getStatusErr
	}

	return m.serverStatus, nil
}

// ShowStats returns mock server statistics
func (m *MockOcctlManager) ShowStats(ctx context.Context) (*ServerStats, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.getStatsErr != nil {
		return nil, m.getStatsErr
	}

	// Update active users count
	stats := *m.serverStats
	activeCount := 0
	for _, user := range m.users {
		disconnected := false
		for _, username := range m.disconnected {
			if user.Username == username {
				disconnected = true
				break
			}
		}
		if !disconnected {
			activeCount++
		}
	}
	stats.ActiveUsers = activeCount

	return &stats, nil
}

// Reload simulates server reload
func (m *MockOcctlManager) Reload(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.reloadErr != nil {
		return m.reloadErr
	}

	m.reloadCalled = true
	return nil
}

// === Mock helpers for testing ===

// AddUser adds a mock user to the list
func (m *MockOcctlManager) AddUser(user User) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.users = append(m.users, user)
}

// AddMockUser adds a mock user with default values
func (m *MockOcctlManager) AddMockUser(id int, username, vpnIP, clientIP string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	user := User{
		ID:             id,
		Username:       username,
		IPv4:           vpnIP,
		RemoteIP:       clientIP,
		Device:         "vpns0",
		State:          "connected",
		RX:             "1.5M",
		TX:             "2.3M",
		ConnectedAt:    time.Now().Format(time.RFC3339),
		RawConnectedAt: time.Now().Unix(),
		UserAgent:      "AnyConnect",
		Hostname:       fmt.Sprintf("client-%s", username),
		TLSCiphersuite: "TLS_AES_256_GCM_SHA384",
	}

	m.users = append(m.users, user)
}

// ClearUsers removes all mock users
func (m *MockOcctlManager) ClearUsers() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.users = make([]User, 0)
	m.disconnected = make([]string, 0)
}

// SetShowUsersError sets an error to be returned by ShowUsers
func (m *MockOcctlManager) SetShowUsersError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.showUsersErr = err
}

// SetDisconnectError sets an error to be returned by DisconnectUser
func (m *MockOcctlManager) SetDisconnectError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.disconnectErr = err
}

// GetDisconnectedUsers returns list of disconnected usernames
func (m *MockOcctlManager) GetDisconnectedUsers() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]string, len(m.disconnected))
	copy(result, m.disconnected)
	return result
}

// WasReloadCalled returns whether ReloadServer was called
func (m *MockOcctlManager) WasReloadCalled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.reloadCalled
}

// WasStopCalled returns whether StopServer was called
func (m *MockOcctlManager) WasStopCalled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.stopCalled
}
