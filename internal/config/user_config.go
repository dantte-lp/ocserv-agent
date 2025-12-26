package config

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
)

// Generator генерирует per-user конфигурационные файлы для ocserv
type Generator struct {
	perUserDir  string
	perGroupDir string
	backupDir   string
	mu          sync.Mutex
}

// NewGenerator создает новый генератор конфигурационных файлов
func NewGenerator(perUserDir, perGroupDir, backupDir string) (*Generator, error) {
	// Проверка существования директорий
	if perUserDir != "" {
		if err := ensureDir(perUserDir); err != nil {
			return nil, errors.Wrapf(err, "failed to ensure per-user dir: %s", perUserDir)
		}
	}

	if perGroupDir != "" {
		if err := ensureDir(perGroupDir); err != nil {
			return nil, errors.Wrapf(err, "failed to ensure per-group dir: %s", perGroupDir)
		}
	}

	if backupDir != "" {
		if err := ensureDir(backupDir); err != nil {
			return nil, errors.Wrapf(err, "failed to ensure backup dir: %s", backupDir)
		}
	}

	return &Generator{
		perUserDir:  perUserDir,
		perGroupDir: perGroupDir,
		backupDir:   backupDir,
	}, nil
}

// UserConfig представляет конфигурацию пользователя
type UserConfig struct {
	Username           string            // Имя пользователя
	Routes             []string          // Маршруты для пользователя (CIDR формат)
	DNSServers         []string          // DNS серверы
	SplitDNS           []string          // Split DNS домены
	RestrictToRoutes   bool              // Ограничить пользователя только указанными маршрутами
	MaxSameClients     int               // Максимальное количество одновременных подключений
	CustomParams       map[string]string // Дополнительные параметры конфигурации
	NoRoute            bool              // Не отправлять маршруты клиенту
	ExplicitIPv4       string            // Явный IPv4 адрес для пользователя
	ExplicitIPv6       string            // Явный IPv6 адрес для пользователя
	RXPerSec           int               // Ограничение скорости приема (bytes/sec)
	TXPerSec           int               // Ограничение скорости передачи (bytes/sec)
	IdleTimeout        int               // Таймаут неактивности (секунды)
	MobileIdleTimeout  int               // Таймаут неактивности для мобильных (секунды)
	SessionTimeout     int               // Максимальная длительность сессии (секунды)
}

// GenerateUserConfig генерирует и сохраняет конфигурацию пользователя
func (g *Generator) GenerateUserConfig(config UserConfig) (string, error) {
	if config.Username == "" {
		return "", errors.New("username cannot be empty")
	}

	// Валидация маршрутов
	if err := validateRoutes(config.Routes); err != nil {
		return "", errors.Wrap(err, "invalid routes")
	}

	// Валидация DNS серверов
	if err := validateIPAddresses(config.DNSServers); err != nil {
		return "", errors.Wrap(err, "invalid DNS servers")
	}

	// Валидация явных IP адресов
	if config.ExplicitIPv4 != "" {
		if err := validateIPAddress(config.ExplicitIPv4); err != nil {
			return "", errors.Wrapf(err, "invalid explicit IPv4: %s", config.ExplicitIPv4)
		}
	}

	if config.ExplicitIPv6 != "" {
		if err := validateIPAddress(config.ExplicitIPv6); err != nil {
			return "", errors.Wrapf(err, "invalid explicit IPv6: %s", config.ExplicitIPv6)
		}
	}

	// Блокировка для thread-safety
	g.mu.Lock()
	defer g.mu.Unlock()

	// Путь к файлу конфигурации
	configPath := filepath.Join(g.perUserDir, config.Username)

	// Создать backup если файл существует
	if g.backupDir != "" {
		if err := g.backupExistingConfig(configPath, config.Username); err != nil {
			return "", errors.Wrap(err, "failed to backup existing config")
		}
	}

	// Генерация содержимого конфигурации
	content := g.generateConfigContent(config)

	// Atomic write: write to temp file, then rename
	tempPath := configPath + ".tmp"
	// #nosec G306 - config files need to be readable by ocserv
	if err := os.WriteFile(tempPath, []byte(content), 0644); err != nil {
		return "", errors.Wrapf(err, "failed to write temp config: %s", tempPath)
	}

	// Rename temp file to final path (atomic operation)
	if err := os.Rename(tempPath, configPath); err != nil {
		// Cleanup temp file on error
		_ = os.Remove(tempPath)
		return "", errors.Wrapf(err, "failed to rename config: %s -> %s", tempPath, configPath)
	}

	return configPath, nil
}

// generateConfigContent генерирует INI-содержимое конфигурации
func (g *Generator) generateConfigContent(config UserConfig) string {
	var sb strings.Builder

	// Заголовок
	sb.WriteString(fmt.Sprintf("# Per-user configuration for %s\n", config.Username))
	sb.WriteString(fmt.Sprintf("# Generated at: %s\n", time.Now().Format(time.RFC3339)))
	sb.WriteString("\n")

	// Маршруты
	if config.NoRoute {
		sb.WriteString("# No routes sent to client\n")
		sb.WriteString("no-route = true\n")
	} else if len(config.Routes) > 0 {
		sb.WriteString("# Custom routes\n")
		for _, route := range config.Routes {
			sb.WriteString(fmt.Sprintf("route = %s\n", route))
		}
	}

	// DNS серверы
	if len(config.DNSServers) > 0 {
		sb.WriteString("\n# DNS servers\n")
		for _, dns := range config.DNSServers {
			sb.WriteString(fmt.Sprintf("dns = %s\n", dns))
		}
	}

	// Split DNS
	if len(config.SplitDNS) > 0 {
		sb.WriteString("\n# Split DNS domains\n")
		for _, domain := range config.SplitDNS {
			sb.WriteString(fmt.Sprintf("split-dns = %s\n", domain))
		}
	}

	// Ограничение на маршруты
	if config.RestrictToRoutes {
		sb.WriteString("\n# Restrict user to specified routes only\n")
		sb.WriteString("restrict-user-to-routes = true\n")
	}

	// Максимальное количество одновременных подключений
	if config.MaxSameClients > 0 {
		sb.WriteString(fmt.Sprintf("\n# Maximum simultaneous connections\n"))
		sb.WriteString(fmt.Sprintf("max-same-clients = %d\n", config.MaxSameClients))
	}

	// Явные IP адреса
	if config.ExplicitIPv4 != "" {
		sb.WriteString(fmt.Sprintf("\n# Explicit IPv4 address\n"))
		sb.WriteString(fmt.Sprintf("explicit-ipv4 = %s\n", config.ExplicitIPv4))
	}

	if config.ExplicitIPv6 != "" {
		sb.WriteString(fmt.Sprintf("\n# Explicit IPv6 address\n"))
		sb.WriteString(fmt.Sprintf("explicit-ipv6 = %s\n", config.ExplicitIPv6))
	}

	// Rate limiting
	if config.RXPerSec > 0 {
		sb.WriteString(fmt.Sprintf("\n# Download rate limit (bytes/sec)\n"))
		sb.WriteString(fmt.Sprintf("rx-per-sec = %d\n", config.RXPerSec))
	}

	if config.TXPerSec > 0 {
		sb.WriteString(fmt.Sprintf("# Upload rate limit (bytes/sec)\n"))
		sb.WriteString(fmt.Sprintf("tx-per-sec = %d\n", config.TXPerSec))
	}

	// Timeouts
	if config.IdleTimeout > 0 {
		sb.WriteString(fmt.Sprintf("\n# Idle timeout (seconds)\n"))
		sb.WriteString(fmt.Sprintf("idle-timeout = %d\n", config.IdleTimeout))
	}

	if config.MobileIdleTimeout > 0 {
		sb.WriteString(fmt.Sprintf("# Mobile idle timeout (seconds)\n"))
		sb.WriteString(fmt.Sprintf("mobile-idle-timeout = %d\n", config.MobileIdleTimeout))
	}

	if config.SessionTimeout > 0 {
		sb.WriteString(fmt.Sprintf("# Session timeout (seconds)\n"))
		sb.WriteString(fmt.Sprintf("session-timeout = %d\n", config.SessionTimeout))
	}

	// Дополнительные параметры
	if len(config.CustomParams) > 0 {
		sb.WriteString("\n# Custom parameters\n")
		for key, value := range config.CustomParams {
			sb.WriteString(fmt.Sprintf("%s = %s\n", key, value))
		}
	}

	return sb.String()
}

// backupExistingConfig создает backup существующего конфигурационного файла
func (g *Generator) backupExistingConfig(configPath, username string) error {
	// Проверка существования файла
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil // Файл не существует, backup не нужен
	}

	// Создание имени backup файла с timestamp
	timestamp := time.Now().Format("20060102-150405")
	backupName := fmt.Sprintf("%s.%s.backup", username, timestamp)
	backupPath := filepath.Join(g.backupDir, backupName)

	// Чтение существующего файла
	content, err := os.ReadFile(configPath)
	if err != nil {
		return errors.Wrapf(err, "failed to read existing config: %s", configPath)
	}

	// Сохранение backup
	// #nosec G306 - backup files need to be readable
	if err := os.WriteFile(backupPath, content, 0644); err != nil {
		return errors.Wrapf(err, "failed to write backup: %s", backupPath)
	}

	return nil
}

// DeleteUserConfig удаляет конфигурацию пользователя
func (g *Generator) DeleteUserConfig(username string) error {
	if username == "" {
		return errors.New("username cannot be empty")
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	configPath := filepath.Join(g.perUserDir, username)

	// Создать backup перед удалением
	if g.backupDir != "" {
		if err := g.backupExistingConfig(configPath, username); err != nil {
			return errors.Wrap(err, "failed to backup before deletion")
		}
	}

	// Удаление файла
	if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
		return errors.Wrapf(err, "failed to delete config: %s", configPath)
	}

	return nil
}

// GetUserConfigPath возвращает путь к конфигурации пользователя
func (g *Generator) GetUserConfigPath(username string) string {
	return filepath.Join(g.perUserDir, username)
}

// UserConfigExists проверяет существование конфигурации пользователя
func (g *Generator) UserConfigExists(username string) bool {
	configPath := g.GetUserConfigPath(username)
	_, err := os.Stat(configPath)
	return err == nil
}

// validateRoutes проверяет корректность маршрутов (CIDR формат)
func validateRoutes(routes []string) error {
	for i, route := range routes {
		if _, _, err := net.ParseCIDR(route); err != nil {
			return errors.Wrapf(err, "invalid CIDR format at index %d: %s", i, route)
		}
	}
	return nil
}

// validateIPAddresses проверяет корректность IP адресов
func validateIPAddresses(ips []string) error {
	for i, ip := range ips {
		if err := validateIPAddress(ip); err != nil {
			return errors.Wrapf(err, "invalid IP address at index %d: %s", i, ip)
		}
	}
	return nil
}

// validateIPAddress проверяет корректность IP адреса
func validateIPAddress(ip string) error {
	if net.ParseIP(ip) == nil {
		return errors.Newf("invalid IP address: %s", ip)
	}
	return nil
}

// ensureDir создает директорию если она не существует
func ensureDir(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Создать директорию
			// #nosec G301 - directories need appropriate permissions
			if err := os.MkdirAll(path, 0755); err != nil {
				return errors.Wrapf(err, "failed to create directory: %s", path)
			}
			return nil
		}
		return errors.Wrapf(err, "failed to stat directory: %s", path)
	}

	if !info.IsDir() {
		return errors.Newf("path exists but is not a directory: %s", path)
	}

	return nil
}
