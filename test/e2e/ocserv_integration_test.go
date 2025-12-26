// +build e2e

package e2e_test

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const (
	// Paths в E2E окружении
	ocservSocketPath = "/var/run/ocserv/ocserv.sock"
	occtlPath        = "/usr/bin/occtl"
	configPerUserDir = "/etc/ocserv/config-per-user"
	testUsername     = "e2etest"
)

// OcservE2ETestSuite содержит E2E тесты для интеграции с ocserv
type OcservE2ETestSuite struct {
	suite.Suite
	ctx           context.Context
	socketPath    string
	occtlPath     string
	configUserDir string
}

// SetupSuite выполняется один раз перед всеми тестами
func (s *OcservE2ETestSuite) SetupSuite() {
	s.ctx = context.Background()
	s.socketPath = getEnvOrDefault("OCSERV_SOCKET_PATH", ocservSocketPath)
	s.occtlPath = getEnvOrDefault("OCCTL_PATH", occtlPath)
	s.configUserDir = getEnvOrDefault("CONFIG_PER_USER_DIR", configPerUserDir)

	// Проверка доступности unix socket
	s.T().Logf("Checking ocserv socket at: %s", s.socketPath)
	s.waitForSocket(s.socketPath, 30*time.Second)
}

// TearDownSuite выполняется один раз после всех тестов
func (s *OcservE2ETestSuite) TearDownSuite() {
	// Cleanup test user config if exists
	testUserConfig := filepath.Join(s.configUserDir, testUsername)
	if _, err := os.Stat(testUserConfig); err == nil {
		_ = os.Remove(testUserConfig)
	}
}

// SetupTest выполняется перед каждым тестом
func (s *OcservE2ETestSuite) SetupTest() {
	// Cleanup before each test
}

// TestOcctlSocketAccess проверяет доступ к unix socket ocserv
func (s *OcservE2ETestSuite) TestOcctlSocketAccess() {
	t := s.T()

	// Проверка существования socket файла
	info, err := os.Stat(s.socketPath)
	require.NoError(t, err, "Socket file should exist")
	require.NotNil(t, info, "Socket info should not be nil")

	// Проверка типа файла (должен быть socket)
	assert.Equal(t, os.ModeSocket, info.Mode()&os.ModeSocket,
		"File should be a unix socket")

	t.Logf("Socket file verified: %s (mode: %v)", s.socketPath, info.Mode())
}

// TestOcctlShowStatus проверяет выполнение команды "occtl show status"
func (s *OcservE2ETestSuite) TestOcctlShowStatus() {
	t := s.T()

	ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, s.occtlPath, "-s", s.socketPath, "show", "status")
	output, err := cmd.CombinedOutput()

	require.NoError(t, err, "occtl show status should succeed")
	assert.NotEmpty(t, output, "Output should not be empty")

	// Проверка наличия ожидаемых полей в выводе
	outputStr := string(output)
	assert.Contains(t, outputStr, "OpenConnect SSL VPN server",
		"Output should contain server info")

	t.Logf("occtl show status output:\n%s", outputStr)
}

// TestOcctlShowUsersJSON проверяет получение списка пользователей в JSON формате
func (s *OcservE2ETestSuite) TestOcctlShowUsersJSON() {
	t := s.T()

	ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, s.occtlPath,
		"-s", s.socketPath,
		"--json",
		"show", "users")
	output, err := cmd.CombinedOutput()

	require.NoError(t, err, "occtl show users should succeed")
	assert.NotEmpty(t, output, "Output should not be empty")

	// Парсинг JSON ответа
	var users []map[string]interface{}
	err = json.Unmarshal(output, &users)
	require.NoError(t, err, "Output should be valid JSON")

	t.Logf("Active users count: %d", len(users))
	if len(users) > 0 {
		t.Logf("First user: %+v", users[0])
	}
}

// TestOcctlShowSessionsJSON проверяет получение активных сессий
func (s *OcservE2ETestSuite) TestOcctlShowSessionsJSON() {
	t := s.T()

	ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, s.occtlPath,
		"-s", s.socketPath,
		"--json",
		"show", "sessions", "all")
	output, err := cmd.CombinedOutput()

	// Команда может вернуть ошибку, если нет активных сессий
	if err != nil {
		t.Logf("No active sessions (expected in test environment): %v", err)
		return
	}

	assert.NotEmpty(t, output, "Output should not be empty")

	// Парсинг JSON если есть сессии
	var sessions []map[string]interface{}
	err = json.Unmarshal(output, &sessions)
	require.NoError(t, err, "Output should be valid JSON")

	t.Logf("Active sessions count: %d", len(sessions))
}

// TestConfigPerUserDirectory проверяет наличие директории config-per-user
func (s *OcservE2ETestSuite) TestConfigPerUserDirectory() {
	t := s.T()

	info, err := os.Stat(s.configUserDir)
	require.NoError(t, err, "Config-per-user directory should exist")
	require.True(t, info.IsDir(), "Should be a directory")

	t.Logf("Config-per-user directory verified: %s", s.configUserDir)
}

// TestGenerateUserConfig проверяет создание пользовательской конфигурации
func (s *OcservE2ETestSuite) TestGenerateUserConfig() {
	t := s.T()

	// Создаём тестовую конфигурацию пользователя
	testUserConfig := filepath.Join(s.configUserDir, testUsername)
	testConfig := `# E2E test user config
route = 10.10.0.0/24
dns = 1.1.1.1
`

	err := os.WriteFile(testUserConfig, []byte(testConfig), 0644)
	require.NoError(t, err, "Should create user config file")

	// Проверка созданного файла
	content, err := os.ReadFile(testUserConfig)
	require.NoError(t, err, "Should read user config file")
	assert.Contains(t, string(content), "route = 10.10.0.0/24",
		"Config should contain route")

	// Cleanup
	err = os.Remove(testUserConfig)
	require.NoError(t, err, "Should remove test config")

	t.Logf("User config test completed: %s", testUserConfig)
}

// TestOcctlReload проверяет перезагрузку конфигурации ocserv
func (s *OcservE2ETestSuite) TestOcctlReload() {
	t := s.T()

	ctx, cancel := context.WithTimeout(s.ctx, 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, s.occtlPath,
		"-s", s.socketPath,
		"reload")
	output, err := cmd.CombinedOutput()

	require.NoError(t, err, "occtl reload should succeed")

	outputStr := string(output)
	t.Logf("occtl reload output: %s", outputStr)

	// После reload даём время на обработку
	time.Sleep(2 * time.Second)

	// Проверяем, что сервер всё ещё работает
	s.TestOcctlShowStatus()
}

// TestOcctlCommandValidation проверяет валидацию команд
func (s *OcservE2ETestSuite) TestOcctlCommandValidation() {
	t := s.T()

	testCases := []struct {
		name        string
		args        []string
		shouldError bool
	}{
		{
			name:        "valid show users",
			args:        []string{"show", "users"},
			shouldError: false,
		},
		{
			name:        "valid show status",
			args:        []string{"show", "status"},
			shouldError: false,
		},
		{
			name:        "invalid command",
			args:        []string{"invalid", "command"},
			shouldError: true,
		},
		{
			name:        "empty args",
			args:        []string{},
			shouldError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
			defer cancel()

			cmdArgs := append([]string{"-s", s.socketPath}, tc.args...)
			cmd := exec.CommandContext(ctx, s.occtlPath, cmdArgs...)
			_, err := cmd.CombinedOutput()

			if tc.shouldError {
				assert.Error(t, err, "Command should fail")
			} else {
				assert.NoError(t, err, "Command should succeed")
			}
		})
	}
}

// TestOcservProcessRunning проверяет, что процесс ocserv запущен
func (s *OcservE2ETestSuite) TestOcservProcessRunning() {
	t := s.T()

	cmd := exec.Command("pgrep", "-x", "ocserv")
	output, err := cmd.CombinedOutput()

	require.NoError(t, err, "ocserv process should be running")
	assert.NotEmpty(t, output, "Should find ocserv PID")

	pid := strings.TrimSpace(string(output))
	t.Logf("ocserv running with PID: %s", pid)
}

// Helper functions

// waitForSocket ждёт появления unix socket с таймаутом
func (s *OcservE2ETestSuite) waitForSocket(socketPath string, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if _, err := os.Stat(socketPath); err == nil {
				s.T().Logf("Socket found: %s", socketPath)
				return
			}

			if time.Now().After(deadline) {
				s.T().Fatalf("Socket not found after %v: %s", timeout, socketPath)
			}

			s.T().Logf("Waiting for socket: %s", socketPath)
		}
	}
}

// getEnvOrDefault возвращает значение env переменной или дефолт
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// TestOcservE2E запускает все E2E тесты
func TestOcservE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E tests in short mode")
	}

	suite.Run(t, new(OcservE2ETestSuite))
}
