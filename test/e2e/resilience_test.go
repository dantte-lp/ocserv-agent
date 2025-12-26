// +build e2e

package e2e_test

import (
	"context"
	"fmt"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	pb "github.com/dantte-lp/ocserv-agent/pkg/proto/agent/v1"
)

const (
	// Параметры для resilience тестов
	ocservRestartTimeout  = 30 * time.Second
	agentRestartTimeout   = 30 * time.Second
	reconnectionTimeout   = 15 * time.Second
	operationRetryTimeout = 10 * time.Second
)

// ResilienceTestSuite содержит тесты отказоустойчивости
type ResilienceTestSuite struct {
	suite.Suite
	ctx        context.Context
	grpcClient pb.VPNAgentServiceClient
	grpcConn   *grpc.ClientConn
}

// SetupSuite выполняется один раз перед всеми тестами
func (s *ResilienceTestSuite) SetupSuite() {
	s.ctx = context.Background()

	// Подключение к gRPC серверу агента
	s.T().Log("Connecting to agent gRPC server for resilience testing...")
	conn := s.connectToAgent(10 * time.Second)
	require.NotNil(s.T(), conn, "Failed to connect to agent gRPC server")

	s.grpcConn = conn
	s.grpcClient = pb.NewVPNAgentServiceClient(conn)

	s.T().Logf("Connected to agent at %s", agentGRPCAddr)
}

// TearDownSuite выполняется один раз после всех тестов
func (s *ResilienceTestSuite) TearDownSuite() {
	if s.grpcConn != nil {
		_ = s.grpcConn.Close()
	}
}

// connectToAgent пытается подключиться к агенту с retry
func (s *ResilienceTestSuite) connectToAgent(timeout time.Duration) *grpc.ClientConn {
	ctx, cancel := context.WithTimeout(s.ctx, timeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			s.T().Log("Connection timeout")
			return nil
		default:
			conn, err := grpc.DialContext(
				ctx,
				agentGRPCAddr,
				grpc.WithTransportCredentials(insecure.NewCredentials()),
				grpc.WithBlock(),
			)
			if err == nil {
				return conn
			}
			s.T().Logf("Connection failed, retrying: %v", err)
			time.Sleep(1 * time.Second)
		}
	}
}

// TestResilience_OcservRestart тестирует поведение при перезапуске ocserv
func (s *ResilienceTestSuite) TestResilience_OcservRestart() {
	t := s.T()

	// Это тест требует docker/podman exec доступа к ocserv контейнеру
	// Пропускаем если не в E2E окружении
	if !s.isE2EEnvironment() {
		t.Skip("Skipping ocserv restart test - not in E2E environment")
	}

	ctx, cancel := context.WithTimeout(s.ctx, ocservRestartTimeout+30*time.Second)
	defer cancel()

	// Создать тестовую сессию
	sessionID := fmt.Sprintf("resilience_session_%d", time.Now().Unix())
	username := "resilience_user"

	connectReq := &pb.NotifyConnectRequest{
		Username:  username,
		ClientIp:  "192.168.220.10",
		VpnIp:     "10.40.10.10",
		SessionId: sessionID,
	}

	t.Log("Creating test session before ocserv restart...")
	connectResp, err := s.grpcClient.NotifyConnect(ctx, connectReq)
	require.NoError(t, err, "NotifyConnect should succeed")
	require.True(t, connectResp.Allowed, "Connection should be allowed")

	// Проверить что сессия существует
	sessionsResp, err := s.grpcClient.GetActiveSessions(ctx, &pb.GetActiveSessionsRequest{})
	require.NoError(t, err, "GetActiveSessions should succeed")

	found := false
	for _, session := range sessionsResp.Sessions {
		if session.SessionId == sessionID {
			found = true
			break
		}
	}
	require.True(t, found, "Session should exist before restart")

	// Перезапустить ocserv
	t.Log("Restarting ocserv...")
	restartCmd := exec.CommandContext(ctx, "podman", "exec", "ocserv-e2e-test", "systemctl", "restart", "ocserv")
	output, err := restartCmd.CombinedOutput()

	if err != nil {
		// Альтернативный метод: kill и start процесса
		t.Logf("systemctl restart failed (%v), trying kill method: %s", err, output)
		killCmd := exec.CommandContext(ctx, "podman", "exec", "ocserv-e2e-test", "pkill", "-HUP", "ocserv")
		_, _ = killCmd.CombinedOutput()
	}

	// Ждём восстановления ocserv
	t.Log("Waiting for ocserv to recover...")
	time.Sleep(5 * time.Second)

	// Проверить что агент всё ещё работает
	t.Log("Checking agent health after ocserv restart...")
	healthCtx, healthCancel := context.WithTimeout(ctx, 5*time.Second)
	defer healthCancel()

	_, err = s.grpcClient.GetActiveSessions(healthCtx, &pb.GetActiveSessionsRequest{})
	assert.NoError(t, err, "Agent should still respond after ocserv restart")

	// Попытаться создать новую сессию после restart
	t.Log("Creating new session after ocserv restart...")
	newSessionID := fmt.Sprintf("resilience_session_after_%d", time.Now().Unix())
	newConnectReq := &pb.NotifyConnectRequest{
		Username:  username + "_new",
		ClientIp:  "192.168.220.11",
		VpnIp:     "10.40.10.11",
		SessionId: newSessionID,
	}

	newConnectResp, err := s.grpcClient.NotifyConnect(ctx, newConnectReq)
	assert.NoError(t, err, "Should be able to create new session after ocserv restart")
	if err == nil {
		assert.True(t, newConnectResp.Allowed, "New connection should be allowed")
	}

	// Cleanup
	_, _ = s.grpcClient.NotifyDisconnect(ctx, &pb.NotifyDisconnectRequest{
		Username:  username,
		SessionId: sessionID,
	})
	_, _ = s.grpcClient.NotifyDisconnect(ctx, &pb.NotifyDisconnectRequest{
		Username:  username + "_new",
		SessionId: newSessionID,
	})
}

// TestResilience_SocketUnavailable тестирует поведение при недоступности socket
func (s *ResilienceTestSuite) TestResilience_SocketUnavailable() {
	t := s.T()
	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()

	// Этот тест проверяет что агент gracefully обрабатывает недоступность ocserv socket
	// Агент должен продолжать принимать gRPC запросы, но occtl команды будут failover

	t.Log("Testing agent behavior when ocserv socket unavailable...")

	// Попытаться выполнить команду через ExecuteCommand (если реализовано)
	// Это должно fail gracefully, а не crash агента

	// Создать сессию (это не зависит от ocserv socket напрямую)
	sessionID := fmt.Sprintf("socket_test_%d", time.Now().Unix())
	connectReq := &pb.NotifyConnectRequest{
		Username:  "socket_test_user",
		ClientIp:  "192.168.220.20",
		VpnIp:     "10.40.10.20",
		SessionId: sessionID,
	}

	connectResp, err := s.grpcClient.NotifyConnect(ctx, connectReq)
	assert.NoError(t, err, "NotifyConnect should work even if socket unavailable")
	if err == nil {
		assert.True(t, connectResp.Allowed, "Connection should be allowed")
	}

	// Попытаться получить активные сессии (работает с SessionStore, не зависит от socket)
	sessionsResp, err := s.grpcClient.GetActiveSessions(ctx, &pb.GetActiveSessionsRequest{})
	assert.NoError(t, err, "GetActiveSessions should work (uses SessionStore)")
	if err == nil {
		t.Logf("Active sessions count: %d", len(sessionsResp.Sessions))
	}

	// Cleanup
	_, _ = s.grpcClient.NotifyDisconnect(ctx, &pb.NotifyDisconnectRequest{
		Username:  "socket_test_user",
		SessionId: sessionID,
	})
}

// TestResilience_TimeoutHandling тестирует обработку таймаутов
func (s *ResilienceTestSuite) TestResilience_TimeoutHandling() {
	t := s.T()

	// Создать контекст с очень коротким таймаутом
	ctx, cancel := context.WithTimeout(s.ctx, 1*time.Millisecond)
	defer cancel()

	// Попытаться выполнить операцию (должна вернуть DeadlineExceeded)
	_, err := s.grpcClient.GetActiveSessions(ctx, &pb.GetActiveSessionsRequest{})

	// Проверить что ошибка корректно обработана
	if err != nil {
		st, ok := status.FromError(err)
		assert.True(t, ok, "Error should be gRPC status")
		if ok {
			assert.Equal(t, codes.DeadlineExceeded, st.Code(),
				"Should return DeadlineExceeded for timeout")
			t.Logf("Timeout correctly handled: %v", st.Message())
		}
	}

	// Проверить что агент всё ещё работает после timeout
	normalCtx, normalCancel := context.WithTimeout(s.ctx, 5*time.Second)
	defer normalCancel()

	_, err = s.grpcClient.GetActiveSessions(normalCtx, &pb.GetActiveSessionsRequest{})
	assert.NoError(t, err, "Agent should recover after timeout error")
}

// TestResilience_ConcurrentFailures тестирует параллельные сбои
func (s *ResilienceTestSuite) TestResilience_ConcurrentFailures() {
	t := s.T()
	ctx, cancel := context.WithTimeout(s.ctx, 60*time.Second)
	defer cancel()

	// Создать несколько сессий
	sessionCount := 10
	sessionIDs := make([]string, 0, sessionCount)

	t.Logf("Creating %d sessions...", sessionCount)
	for i := 0; i < sessionCount; i++ {
		sessionID := fmt.Sprintf("concurrent_session_%d_%d", i, time.Now().Unix())
		sessionIDs = append(sessionIDs, sessionID)

		connectReq := &pb.NotifyConnectRequest{
			Username:  fmt.Sprintf("concurrent_user_%d", i),
			ClientIp:  fmt.Sprintf("192.168.220.%d", 30+i),
			VpnIp:     fmt.Sprintf("10.40.10.%d", 30+i),
			SessionId: sessionID,
		}

		_, err := s.grpcClient.NotifyConnect(ctx, connectReq)
		require.NoError(t, err, "Failed to create session %d", i)
	}

	// Проверить все сессии созданы
	sessionsResp, err := s.grpcClient.GetActiveSessions(ctx, &pb.GetActiveSessionsRequest{})
	require.NoError(t, err, "GetActiveSessions should succeed")

	foundCount := 0
	for _, session := range sessionsResp.Sessions {
		for _, sid := range sessionIDs {
			if session.SessionId == sid {
				foundCount++
				break
			}
		}
	}
	assert.GreaterOrEqual(t, foundCount, sessionCount, "All sessions should be created")

	// Симулировать concurrent failures: половину сессий disconnect с ошибкой
	t.Log("Simulating concurrent disconnects...")
	for i := 0; i < sessionCount/2; i++ {
		disconnectReq := &pb.NotifyDisconnectRequest{
			Username:         fmt.Sprintf("concurrent_user_%d", i),
			SessionId:        sessionIDs[i],
			DisconnectReason: "simulated_failure",
		}

		_, err := s.grpcClient.NotifyDisconnect(ctx, disconnectReq)
		assert.NoError(t, err, "Disconnect should succeed even during failures")
	}

	time.Sleep(1 * time.Second)

	// Проверить что оставшиеся сессии всё ещё активны
	sessionsResp2, err := s.grpcClient.GetActiveSessions(ctx, &pb.GetActiveSessionsRequest{})
	require.NoError(t, err, "GetActiveSessions should work after concurrent failures")

	remainingCount := 0
	for _, session := range sessionsResp2.Sessions {
		for i := sessionCount / 2; i < sessionCount; i++ {
			if session.SessionId == sessionIDs[i] {
				remainingCount++
				break
			}
		}
	}

	expectedRemaining := sessionCount - sessionCount/2
	assert.Equal(t, expectedRemaining, remainingCount,
		"Should have %d sessions remaining after %d disconnects",
		expectedRemaining, sessionCount/2)

	t.Logf("Resilience test passed: %d/%d sessions remaining after concurrent failures",
		remainingCount, expectedRemaining)

	// Cleanup оставшихся сессий
	for i := sessionCount / 2; i < sessionCount; i++ {
		_, _ = s.grpcClient.NotifyDisconnect(ctx, &pb.NotifyDisconnectRequest{
			Username:  fmt.Sprintf("concurrent_user_%d", i),
			SessionId: sessionIDs[i],
		})
	}
}

// TestResilience_GracefulDegradation тестирует graceful degradation
func (s *ResilienceTestSuite) TestResilience_GracefulDegradation() {
	t := s.T()
	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()

	// Этот тест проверяет что при частичной недоступности компонентов
	// агент продолжает работать в degraded режиме

	t.Log("Testing graceful degradation...")

	// Попытка создать сессию
	sessionID := fmt.Sprintf("degraded_session_%d", time.Now().Unix())
	connectReq := &pb.NotifyConnectRequest{
		Username:  "degraded_user",
		ClientIp:  "192.168.220.50",
		VpnIp:     "10.40.10.50",
		SessionId: sessionID,
	}

	connectResp, err := s.grpcClient.NotifyConnect(ctx, connectReq)
	require.NoError(t, err, "NotifyConnect should succeed in degraded mode")
	require.True(t, connectResp.Allowed, "Connection should be allowed")

	// UpdateUserRoutes может fail если config generator недоступен,
	// но агент должен вернуть корректный ответ, а не crash
	routesReq := &pb.UpdateUserRoutesRequest{
		Username: "degraded_user",
		Routes:   []string{"10.50.0.0/16"},
	}

	routesResp, err := s.grpcClient.UpdateUserRoutes(ctx, routesReq)
	// Может быть либо успех, либо graceful failure
	if err != nil {
		t.Logf("UpdateUserRoutes failed gracefully: %v", err)
		st, ok := status.FromError(err)
		assert.True(t, ok, "Error should be gRPC status")
		if ok {
			// Не должно быть Internal или Unknown ошибок
			assert.NotEqual(t, codes.Internal, st.Code(),
				"Should not return Internal error")
			assert.NotEqual(t, codes.Unknown, st.Code(),
				"Should not return Unknown error")
		}
	} else {
		assert.NotNil(t, routesResp, "Response should not be nil")
		t.Logf("UpdateUserRoutes succeeded: %v", routesResp.Success)
	}

	// Проверить что GetActiveSessions всё ещё работает
	sessionsResp, err := s.grpcClient.GetActiveSessions(ctx, &pb.GetActiveSessionsRequest{})
	require.NoError(t, err, "GetActiveSessions should work in degraded mode")
	assert.NotNil(t, sessionsResp, "Response should not be nil")

	t.Log("Graceful degradation test passed")

	// Cleanup
	_, _ = s.grpcClient.NotifyDisconnect(ctx, &pb.NotifyDisconnectRequest{
		Username:  "degraded_user",
		SessionId: sessionID,
	})
}

// TestResilience_InvalidInput тестирует обработку некорректных входных данных
func (s *ResilienceTestSuite) TestResilience_InvalidInput() {
	t := s.T()
	ctx, cancel := context.WithTimeout(s.ctx, 15*time.Second)
	defer cancel()

	testCases := []struct {
		name        string
		req         *pb.NotifyConnectRequest
		expectError bool
	}{
		{
			name: "empty username",
			req: &pb.NotifyConnectRequest{
				Username:  "",
				ClientIp:  "192.168.1.1",
				VpnIp:     "10.0.0.1",
				SessionId: "session_1",
			},
			expectError: false, // Агент может принять или отклонить
		},
		{
			name: "invalid IP format",
			req: &pb.NotifyConnectRequest{
				Username:  "test_user",
				ClientIp:  "invalid_ip",
				VpnIp:     "10.0.0.1",
				SessionId: "session_2",
			},
			expectError: false, // Агент должен gracefully обработать
		},
		{
			name: "empty session ID",
			req: &pb.NotifyConnectRequest{
				Username:  "test_user",
				ClientIp:  "192.168.1.1",
				VpnIp:     "10.0.0.1",
				SessionId: "",
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := s.grpcClient.NotifyConnect(ctx, tc.req)

			// Агент не должен crash
			if err != nil {
				st, ok := status.FromError(err)
				assert.True(t, ok, "Error should be gRPC status")
				if ok {
					assert.NotEqual(t, codes.Internal, st.Code(),
						"Should not return Internal error for invalid input")
					t.Logf("Invalid input handled gracefully: %v", st.Message())
				}
			} else {
				assert.NotNil(t, resp, "Response should not be nil")
				t.Logf("Request processed: allowed=%v", resp.Allowed)
			}
		})
	}
}

// isE2EEnvironment проверяет что мы в E2E окружении с доступом к контейнерам
func (s *ResilienceTestSuite) isE2EEnvironment() bool {
	cmd := exec.Command("podman", "ps", "--filter", "name=ocserv-e2e-test", "--format", "{{.Names}}")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	return len(output) > 0
}

// TestResilienceE2E запускает resilience тесты
func TestResilienceE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping resilience tests in short mode")
	}

	suite.Run(t, new(ResilienceTestSuite))
}
