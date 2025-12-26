// +build e2e

package e2e_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/dantte-lp/ocserv-agent/pkg/proto/agent/v1"
)

const (
	// Путь к gRPC серверу агента в E2E окружении
	agentGRPCAddr = "localhost:9091"
	testUser1     = "fullflow_user1"
	testUser2     = "fullflow_user2"
)

// FullFlowE2ETestSuite содержит тесты полного цикла работы агента
type FullFlowE2ETestSuite struct {
	suite.Suite
	ctx           context.Context
	grpcClient    pb.VPNAgentServiceClient
	grpcConn      *grpc.ClientConn
	configUserDir string
}

// SetupSuite выполняется один раз перед всеми тестами
func (s *FullFlowE2ETestSuite) SetupSuite() {
	s.ctx = context.Background()
	s.configUserDir = getEnvOrDefault("CONFIG_PER_USER_DIR", "/etc/ocserv/config-per-user")

	// Подключение к gRPC серверу агента
	s.T().Log("Connecting to agent gRPC server...")
	conn, err := grpc.Dial(
		agentGRPCAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithTimeout(10*time.Second),
	)
	require.NoError(s.T(), err, "Failed to connect to agent gRPC server")
	s.grpcConn = conn
	s.grpcClient = pb.NewVPNAgentServiceClient(conn)

	s.T().Logf("Connected to agent at %s", agentGRPCAddr)
}

// TearDownSuite выполняется один раз после всех тестов
func (s *FullFlowE2ETestSuite) TearDownSuite() {
	if s.grpcConn != nil {
		_ = s.grpcConn.Close()
	}

	// Cleanup test configs
	for _, user := range []string{testUser1, testUser2} {
		configPath := filepath.Join(s.configUserDir, user)
		if _, err := os.Stat(configPath); err == nil {
			_ = os.Remove(configPath)
		}
	}
}

// TestFullFlow_ConnectSessionManagement тестирует полный цикл:
// 1. NotifyConnect - создание сессии
// 2. GetActiveSessions - проверка наличия сессии
// 3. UpdateUserRoutes - обновление маршрутов
// 4. NotifyDisconnect - удаление сессии
// 5. GetActiveSessions - проверка отсутствия сессии
func (s *FullFlowE2ETestSuite) TestFullFlow_ConnectSessionManagement() {
	t := s.T()
	ctx, cancel := context.WithTimeout(s.ctx, 60*time.Second)
	defer cancel()

	sessionID := fmt.Sprintf("session_%d", time.Now().Unix())
	username := testUser1
	clientIP := "192.168.100.50"
	vpnIP := "10.10.10.50"

	// ===== STEP 1: NotifyConnect =====
	t.Log("STEP 1: NotifyConnect - creating session")
	connectReq := &pb.NotifyConnectRequest{
		Username:  username,
		ClientIp:  clientIP,
		VpnIp:     vpnIP,
		SessionId: sessionID,
		DeviceId:  "device_e2e_test",
		Metadata: map[string]string{
			"user_agent": "OpenConnect",
			"protocol":   "ssl",
		},
	}

	connectResp, err := s.grpcClient.NotifyConnect(ctx, connectReq)
	require.NoError(t, err, "NotifyConnect should succeed")
	require.NotNil(t, connectResp, "Response should not be nil")
	assert.True(t, connectResp.Allowed, "Connection should be allowed")
	assert.False(t, connectResp.ShouldDisconnect, "Should not request disconnect")

	t.Logf("Connection allowed for session: %s", sessionID)

	// Даём время на обработку
	time.Sleep(500 * time.Millisecond)

	// ===== STEP 2: GetActiveSessions - check session exists =====
	t.Log("STEP 2: GetActiveSessions - verifying session exists")
	sessionsReq := &pb.GetActiveSessionsRequest{}
	sessionsResp, err := s.grpcClient.GetActiveSessions(ctx, sessionsReq)
	require.NoError(t, err, "GetActiveSessions should succeed")
	require.NotNil(t, sessionsResp, "Response should not be nil")

	// Проверка наличия нашей сессии
	var foundSession *pb.Session
	for _, session := range sessionsResp.Sessions {
		if session.SessionId == sessionID {
			foundSession = session
			break
		}
	}

	require.NotNil(t, foundSession, "Session should exist in active sessions")
	assert.Equal(t, username, foundSession.Username, "Username should match")
	assert.Equal(t, clientIP, foundSession.ClientIp, "Client IP should match")
	assert.Equal(t, vpnIP, foundSession.VpnIp, "VPN IP should match")
	assert.NotNil(t, foundSession.ConnectedAt, "ConnectedAt should be set")

	t.Logf("Session found: %s (connected at: %v)", sessionID, foundSession.ConnectedAt.AsTime())

	// ===== STEP 3: UpdateUserRoutes - generate per-user config =====
	t.Log("STEP 3: UpdateUserRoutes - generating per-user config")
	routesReq := &pb.UpdateUserRoutesRequest{
		Username: username,
		Routes: []string{
			"10.20.0.0/16",
			"192.168.50.0/24",
		},
		DnsServers: []string{
			"10.0.0.53",
			"8.8.8.8",
		},
		SplitDns: []string{
			"internal.corp.com",
		},
	}

	routesResp, err := s.grpcClient.UpdateUserRoutes(ctx, routesReq)
	require.NoError(t, err, "UpdateUserRoutes should succeed")
	require.NotNil(t, routesResp, "Response should not be nil")
	assert.True(t, routesResp.Success, "Route update should succeed")

	t.Log("User routes updated successfully")

	// Проверка созданного конфига (если configGenerator доступен)
	configPath := filepath.Join(s.configUserDir, username)
	if _, err := os.Stat(s.configUserDir); err == nil {
		// Directory exists, check if config was created
		if content, err := os.ReadFile(configPath); err == nil {
			t.Logf("Per-user config created:\n%s", string(content))
			assert.Contains(t, string(content), "10.20.0.0/16", "Config should contain route")
			assert.Contains(t, string(content), "10.0.0.53", "Config should contain DNS")
		}
	}

	// ===== STEP 4: NotifyDisconnect - remove session =====
	t.Log("STEP 4: NotifyDisconnect - removing session")
	disconnectReq := &pb.NotifyDisconnectRequest{
		Username:         username,
		SessionId:        sessionID,
		BytesIn:          1024000,
		BytesOut:         2048000,
		DisconnectReason: "user_request",
	}

	disconnectResp, err := s.grpcClient.NotifyDisconnect(ctx, disconnectReq)
	require.NoError(t, err, "NotifyDisconnect should succeed")
	require.NotNil(t, disconnectResp, "Response should not be nil")
	assert.True(t, disconnectResp.Success, "Disconnect should succeed")

	t.Logf("Session disconnected: %s", sessionID)

	// Даём время на обработку
	time.Sleep(500 * time.Millisecond)

	// ===== STEP 5: GetActiveSessions - verify session removed =====
	t.Log("STEP 5: GetActiveSessions - verifying session removed")
	sessionsResp2, err := s.grpcClient.GetActiveSessions(ctx, sessionsReq)
	require.NoError(t, err, "GetActiveSessions should succeed")

	// Проверка отсутствия сессии
	foundAfterDisconnect := false
	for _, session := range sessionsResp2.Sessions {
		if session.SessionId == sessionID {
			foundAfterDisconnect = true
			break
		}
	}

	assert.False(t, foundAfterDisconnect, "Session should be removed after disconnect")
	t.Log("Session successfully removed from store")

	// ===== STEP 6: Cleanup - verify config cleanup =====
	t.Log("STEP 6: Cleanup verification")
	// Config файл может оставаться или удаляться в зависимости от логики
	// Это опциональная проверка
}

// TestFullFlow_MultipleSessionsSameUser тестирует несколько сессий одного пользователя
func (s *FullFlowE2ETestSuite) TestFullFlow_MultipleSessionsSameUser() {
	t := s.T()
	ctx, cancel := context.WithTimeout(s.ctx, 60*time.Second)
	defer cancel()

	username := testUser1
	session1ID := fmt.Sprintf("session1_%d", time.Now().Unix())
	session2ID := fmt.Sprintf("session2_%d", time.Now().Unix())

	// Создать две сессии для одного пользователя
	for i, sessionID := range []string{session1ID, session2ID} {
		connectReq := &pb.NotifyConnectRequest{
			Username:  username,
			ClientIp:  fmt.Sprintf("192.168.100.%d", 60+i),
			VpnIp:     fmt.Sprintf("10.10.10.%d", 60+i),
			SessionId: sessionID,
			DeviceId:  fmt.Sprintf("device_%d", i+1),
		}

		connectResp, err := s.grpcClient.NotifyConnect(ctx, connectReq)
		require.NoError(t, err, "NotifyConnect should succeed for session %d", i+1)
		assert.True(t, connectResp.Allowed, "Connection should be allowed for session %d", i+1)

		t.Logf("Created session %d: %s", i+1, sessionID)
	}

	time.Sleep(500 * time.Millisecond)

	// Проверить обе сессии
	sessionsResp, err := s.grpcClient.GetActiveSessions(ctx, &pb.GetActiveSessionsRequest{})
	require.NoError(t, err, "GetActiveSessions should succeed")

	foundCount := 0
	for _, session := range sessionsResp.Sessions {
		if session.Username == username && (session.SessionId == session1ID || session.SessionId == session2ID) {
			foundCount++
		}
	}

	assert.GreaterOrEqual(t, foundCount, 2, "Should find at least 2 sessions for user")
	t.Logf("Found %d sessions for user %s", foundCount, username)

	// Отключить первую сессию
	disconnectReq := &pb.NotifyDisconnectRequest{
		Username:  username,
		SessionId: session1ID,
	}

	disconnectResp, err := s.grpcClient.NotifyDisconnect(ctx, disconnectReq)
	require.NoError(t, err, "NotifyDisconnect should succeed")
	assert.True(t, disconnectResp.Success, "Disconnect should succeed")

	time.Sleep(500 * time.Millisecond)

	// Проверить что осталась одна сессия
	sessionsResp2, err := s.grpcClient.GetActiveSessions(ctx, &pb.GetActiveSessionsRequest{})
	require.NoError(t, err, "GetActiveSessions should succeed")

	foundAfter := 0
	for _, session := range sessionsResp2.Sessions {
		if session.Username == username {
			foundAfter++
			assert.Equal(t, session2ID, session.SessionId, "Only second session should remain")
		}
	}

	assert.Equal(t, 1, foundAfter, "Should have exactly 1 session remaining")
	t.Log("First session removed, second session remains")

	// Cleanup второй сессии
	_, _ = s.grpcClient.NotifyDisconnect(ctx, &pb.NotifyDisconnectRequest{
		Username:  username,
		SessionId: session2ID,
	})
}

// TestFullFlow_SessionExpiry тестирует истечение сессий (если TTL настроен)
func (s *FullFlowE2ETestSuite) TestFullFlow_SessionExpiry() {
	t := s.T()
	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()

	// Этот тест проверяет что сессии не истекают сразу
	// В production SessionStore имеет TTL 24h
	sessionID := fmt.Sprintf("session_expiry_%d", time.Now().Unix())

	connectReq := &pb.NotifyConnectRequest{
		Username:  testUser2,
		ClientIp:  "192.168.100.70",
		VpnIp:     "10.10.10.70",
		SessionId: sessionID,
	}

	connectResp, err := s.grpcClient.NotifyConnect(ctx, connectReq)
	require.NoError(t, err, "NotifyConnect should succeed")
	assert.True(t, connectResp.Allowed, "Connection should be allowed")

	// Проверить сессию сразу
	sessionsResp, err := s.grpcClient.GetActiveSessions(ctx, &pb.GetActiveSessionsRequest{})
	require.NoError(t, err, "GetActiveSessions should succeed")

	found := false
	for _, session := range sessionsResp.Sessions {
		if session.SessionId == sessionID {
			found = true
			break
		}
	}
	assert.True(t, found, "Session should exist immediately after connect")

	// Подождать несколько секунд
	t.Log("Waiting 5 seconds to verify session doesn't expire prematurely...")
	time.Sleep(5 * time.Second)

	// Проверить что сессия всё ещё активна
	sessionsResp2, err := s.grpcClient.GetActiveSessions(ctx, &pb.GetActiveSessionsRequest{})
	require.NoError(t, err, "GetActiveSessions should succeed after delay")

	foundAfterDelay := false
	for _, session := range sessionsResp2.Sessions {
		if session.SessionId == sessionID {
			foundAfterDelay = true
			break
		}
	}
	assert.True(t, foundAfterDelay, "Session should still exist after 5 seconds (TTL is 24h)")

	t.Log("Session persists correctly (TTL validation)")

	// Cleanup
	_, _ = s.grpcClient.NotifyDisconnect(ctx, &pb.NotifyDisconnectRequest{
		Username:  testUser2,
		SessionId: sessionID,
	})
}

// TestFullFlow_UpdateRoutesWithoutSession тестирует обновление маршрутов
// для пользователя без активной сессии
func (s *FullFlowE2ETestSuite) TestFullFlow_UpdateRoutesWithoutSession() {
	t := s.T()
	ctx, cancel := context.WithTimeout(s.ctx, 15*time.Second)
	defer cancel()

	username := "routes_only_user"

	// Попытаться обновить маршруты без активной сессии
	routesReq := &pb.UpdateUserRoutesRequest{
		Username: username,
		Routes: []string{
			"172.16.0.0/12",
		},
		DnsServers: []string{"1.1.1.1"},
	}

	routesResp, err := s.grpcClient.UpdateUserRoutes(ctx, routesReq)
	require.NoError(t, err, "UpdateUserRoutes should succeed even without active session")
	require.NotNil(t, routesResp, "Response should not be nil")
	assert.True(t, routesResp.Success, "Route update should succeed")

	t.Log("Routes updated successfully for user without active session")

	// Проверить что конфиг создан (если directory доступна)
	configPath := filepath.Join(s.configUserDir, username)
	if _, err := os.Stat(s.configUserDir); err == nil {
		if content, err := os.ReadFile(configPath); err == nil {
			t.Logf("Config created:\n%s", string(content))
			assert.Contains(t, string(content), "172.16.0.0/12", "Config should contain route")
		}
	}

	// Cleanup
	if _, err := os.Stat(configPath); err == nil {
		_ = os.Remove(configPath)
	}
}

// TestFullFlowE2E запускает full flow E2E тесты
func TestFullFlowE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E tests in short mode")
	}

	suite.Run(t, new(FullFlowE2ETestSuite))
}
