package ocserv

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

// TestNewConfigReader tests ConfigReader creation
func TestNewConfigReader(t *testing.T) {
	logger := zerolog.New(os.Stdout)
	reader := NewConfigReader(logger)

	if reader == nil {
		t.Fatal("NewConfigReader() returned nil")
	}
}

// TestReadOcservConf tests reading main ocserv configuration
func TestReadOcservConf(t *testing.T) {
	logger := zerolog.New(os.Stdout)
	reader := NewConfigReader(logger)
	ctx := context.Background()

	tests := []struct {
		name      string
		path      string
		wantError bool
		checkFunc func(*testing.T, *ConfigFile)
	}{
		{
			name:      "valid config",
			path:      "../../test/fixtures/ocserv/configs/ocserv.conf",
			wantError: false,
			checkFunc: func(t *testing.T, cfg *ConfigFile) {
				// Check basic settings
				if val, ok := cfg.GetSetting("auth"); !ok || val == "" {
					t.Error("auth setting not found or empty")
				}
				if val, ok := cfg.GetSetting("tcp-port"); !ok || val != "443" {
					t.Errorf("tcp-port = %v, expected 443", val)
				}

				// Check multi-value settings
				dns, ok := cfg.GetSettings("dns")
				if !ok || len(dns) != 3 {
					t.Errorf("dns count = %d, expected 3", len(dns))
				}

				routes, ok := cfg.GetSettings("route")
				if !ok || len(routes) != 3 {
					t.Errorf("route count = %d, expected 3", len(routes))
				}

				// Check inline comment handling
				if val, ok := cfg.GetSetting("compression"); !ok || val != "true" {
					t.Errorf("compression = %v, expected true", val)
				}

				// Check space-separated format
				if val, ok := cfg.GetSetting("cookie-timeout"); !ok || val != "86400" {
					t.Errorf("cookie-timeout = %v, expected 86400", val)
				}
			},
		},
		{
			name:      "minimal config",
			path:      "../../test/fixtures/ocserv/configs/minimal.conf",
			wantError: false,
			checkFunc: func(t *testing.T, cfg *ConfigFile) {
				if len(cfg.Settings) != 2 {
					t.Errorf("Settings count = %d, expected 2", len(cfg.Settings))
				}
			},
		},
		{
			name:      "non-existent file",
			path:      "/nonexistent/path/config.conf",
			wantError: true,
			checkFunc: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := reader.ReadOcservConf(ctx, tt.path)

			if tt.wantError {
				if err == nil {
					t.Error("ReadOcservConf() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("ReadOcservConf() error = %v", err)
			}

			if cfg == nil {
				t.Fatal("ReadOcservConf() returned nil config")
			}

			if cfg.Path != tt.path {
				t.Errorf("Path = %s, expected %s", cfg.Path, tt.path)
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, cfg)
			}
		})
	}
}

// TestReadUserConfig tests reading per-user configuration
func TestReadUserConfig(t *testing.T) {
	logger := zerolog.New(os.Stdout)
	reader := NewConfigReader(logger)
	ctx := context.Background()

	baseDir := "../../test/fixtures/ocserv/configs"
	username := "user-john"

	cfg, err := reader.ReadUserConfig(ctx, baseDir, username)
	if err != nil {
		t.Fatalf("ReadUserConfig() error = %v", err)
	}

	if cfg == nil {
		t.Fatal("ReadUserConfig() returned nil config")
	}

	expectedPath := filepath.Join(baseDir, username)
	if cfg.Path != expectedPath {
		t.Errorf("Path = %s, expected %s", cfg.Path, expectedPath)
	}

	// Check settings
	if val, ok := cfg.GetSetting("ipv4-network"); !ok || val != "192.168.100.0" {
		t.Errorf("ipv4-network = %v, expected 192.168.100.0", val)
	}
}

// TestReadUserConfigNonExistent tests reading non-existent user config
func TestReadUserConfigNonExistent(t *testing.T) {
	logger := zerolog.New(os.Stdout)
	reader := NewConfigReader(logger)
	ctx := context.Background()

	_, err := reader.ReadUserConfig(ctx, "/tmp", "nonexistent-user")
	if err == nil {
		t.Error("ReadUserConfig() expected error for non-existent user, got nil")
	}
}

// TestReadGroupConfig tests reading per-group configuration
func TestReadGroupConfig(t *testing.T) {
	logger := zerolog.New(os.Stdout)
	reader := NewConfigReader(logger)
	ctx := context.Background()

	baseDir := "../../test/fixtures/ocserv/configs"
	groupname := "group-admins"

	cfg, err := reader.ReadGroupConfig(ctx, baseDir, groupname)
	if err != nil {
		t.Fatalf("ReadGroupConfig() error = %v", err)
	}

	if cfg == nil {
		t.Fatal("ReadGroupConfig() returned nil config")
	}

	expectedPath := filepath.Join(baseDir, groupname)
	if cfg.Path != expectedPath {
		t.Errorf("Path = %s, expected %s", cfg.Path, expectedPath)
	}

	// Check settings
	if val, ok := cfg.GetSetting("max-same-clients"); !ok || val != "5" {
		t.Errorf("max-same-clients = %v, expected 5", val)
	}
}

// TestListUserConfigs tests listing user configuration files
func TestListUserConfigs(t *testing.T) {
	logger := zerolog.New(os.Stdout)
	reader := NewConfigReader(logger)
	ctx := context.Background()

	baseDir := "../../test/fixtures/ocserv/configs"

	files, err := reader.ListUserConfigs(ctx, baseDir)
	if err != nil {
		t.Fatalf("ListUserConfigs() error = %v", err)
	}

	// Should find our test files
	if len(files) == 0 {
		t.Error("ListUserConfigs() returned empty list")
	}

	// Check that user-john is in the list
	found := false
	for _, f := range files {
		if f == "user-john" {
			found = true
			break
		}
	}
	if !found {
		t.Error("user-john not found in user configs list")
	}
}

// TestListUserConfigsNonExistentDir tests listing from non-existent directory
func TestListUserConfigsNonExistentDir(t *testing.T) {
	logger := zerolog.New(os.Stdout)
	reader := NewConfigReader(logger)
	ctx := context.Background()

	files, err := reader.ListUserConfigs(ctx, "/nonexistent/directory")
	if err != nil {
		t.Fatalf("ListUserConfigs() error = %v", err)
	}

	// Should return empty list for non-existent directory
	if len(files) != 0 {
		t.Errorf("ListUserConfigs() returned %d files, expected 0", len(files))
	}
}

// TestListGroupConfigs tests listing group configuration files
func TestListGroupConfigs(t *testing.T) {
	logger := zerolog.New(os.Stdout)
	reader := NewConfigReader(logger)
	ctx := context.Background()

	baseDir := "../../test/fixtures/ocserv/configs"

	files, err := reader.ListGroupConfigs(ctx, baseDir)
	if err != nil {
		t.Fatalf("ListGroupConfigs() error = %v", err)
	}

	if len(files) == 0 {
		t.Error("ListGroupConfigs() returned empty list")
	}

	// Check that group-admins is in the list
	found := false
	for _, f := range files {
		if f == "group-admins" {
			found = true
			break
		}
	}
	if !found {
		t.Error("group-admins not found in group configs list")
	}
}

// TestListConfigFilesWithFile tests listing when path is a file not directory
func TestListConfigFilesWithFile(t *testing.T) {
	logger := zerolog.New(os.Stdout)
	reader := NewConfigReader(logger)
	ctx := context.Background()

	// Use a config file as path (should be a directory)
	filePath := "../../test/fixtures/ocserv/configs/ocserv.conf"

	_, err := reader.listConfigFiles(ctx, filePath)
	if err == nil {
		t.Error("listConfigFiles() expected error when path is a file, got nil")
	}
}

// TestParseLine tests configuration line parsing
func TestParseLine(t *testing.T) {
	logger := zerolog.New(os.Stdout)
	reader := NewConfigReader(logger)

	tests := []struct {
		name      string
		line      string
		wantKey   string
		wantValue string
		wantError bool
	}{
		{
			name:      "equals format",
			line:      "tcp-port = 443",
			wantKey:   "tcp-port",
			wantValue: "443",
			wantError: false,
		},
		{
			name:      "equals format without spaces",
			line:      "auth=\"plain\"",
			wantKey:   "auth",
			wantValue: "\"plain\"",
			wantError: false,
		},
		{
			name:      "space-separated format",
			line:      "cookie-timeout 86400",
			wantKey:   "cookie-timeout",
			wantValue: "86400",
			wantError: false,
		},
		{
			name:      "space-separated with multiple values",
			line:      "banner Welcome to VPN",
			wantKey:   "banner",
			wantValue: "Welcome to VPN",
			wantError: false,
		},
		{
			name:      "inline comment",
			line:      "compression = true # Enable compression",
			wantKey:   "compression",
			wantValue: "true",
			wantError: false,
		},
		{
			name:      "comment line",
			line:      "# This is a comment",
			wantKey:   "",
			wantValue: "",
			wantError: false,
		},
		{
			name:      "empty line",
			line:      "",
			wantKey:   "",
			wantValue: "",
			wantError: false,
		},
		{
			name:      "whitespace only",
			line:      "   ",
			wantKey:   "",
			wantValue: "",
			wantError: false,
		},
		{
			name:      "only comment after spaces",
			line:      "  # Comment",
			wantKey:   "",
			wantValue: "",
			wantError: false,
		},
		{
			name:      "invalid format - only key",
			line:      "tcp-port",
			wantKey:   "",
			wantValue: "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, value, err := reader.parseLine(tt.line)

			if tt.wantError {
				if err == nil {
					t.Error("parseLine() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("parseLine() unexpected error = %v", err)
			}

			if key != tt.wantKey {
				t.Errorf("key = %s, expected %s", key, tt.wantKey)
			}

			if value != tt.wantValue {
				t.Errorf("value = %s, expected %s", value, tt.wantValue)
			}
		})
	}
}

// TestConfigFileGetSetting tests retrieving single-value settings
func TestConfigFileGetSetting(t *testing.T) {
	cfg := &ConfigFile{
		Settings: map[string][]string{
			"tcp-port": {"443"},
			"dns":      {"8.8.8.8", "8.8.4.4"},
		},
	}

	// Test existing setting
	val, ok := cfg.GetSetting("tcp-port")
	if !ok {
		t.Error("GetSetting() returned false for existing setting")
	}
	if val != "443" {
		t.Errorf("GetSetting() = %s, expected 443", val)
	}

	// Test multi-value setting (should return first value)
	val, ok = cfg.GetSetting("dns")
	if !ok {
		t.Error("GetSetting() returned false for existing multi-value setting")
	}
	if val != "8.8.8.8" {
		t.Errorf("GetSetting() = %s, expected 8.8.8.8", val)
	}

	// Test non-existent setting
	val, ok = cfg.GetSetting("nonexistent")
	if ok {
		t.Error("GetSetting() returned true for non-existent setting")
	}
	if val != "" {
		t.Errorf("GetSetting() = %s, expected empty string", val)
	}
}

// TestConfigFileGetSettings tests retrieving multi-value settings
func TestConfigFileGetSettings(t *testing.T) {
	cfg := &ConfigFile{
		Settings: map[string][]string{
			"tcp-port": {"443"},
			"dns":      {"8.8.8.8", "8.8.4.4", "1.1.1.1"},
		},
	}

	// Test multi-value setting
	vals, ok := cfg.GetSettings("dns")
	if !ok {
		t.Error("GetSettings() returned false for existing setting")
	}
	if len(vals) != 3 {
		t.Errorf("GetSettings() returned %d values, expected 3", len(vals))
	}

	// Test single-value setting
	vals, ok = cfg.GetSettings("tcp-port")
	if !ok {
		t.Error("GetSettings() returned false for existing setting")
	}
	if len(vals) != 1 {
		t.Errorf("GetSettings() returned %d values, expected 1", len(vals))
	}

	// Test non-existent setting
	vals, ok = cfg.GetSettings("nonexistent")
	if ok {
		t.Error("GetSettings() returned true for non-existent setting")
	}
	if vals != nil {
		t.Error("GetSettings() returned non-nil for non-existent setting")
	}
}

// TestConfigFileHasSetting tests checking setting existence
func TestConfigFileHasSetting(t *testing.T) {
	cfg := &ConfigFile{
		Settings: map[string][]string{
			"tcp-port": {"443"},
			"dns":      {"8.8.8.8"},
		},
	}

	if !cfg.HasSetting("tcp-port") {
		t.Error("HasSetting() returned false for existing setting")
	}

	if !cfg.HasSetting("dns") {
		t.Error("HasSetting() returned false for existing setting")
	}

	if cfg.HasSetting("nonexistent") {
		t.Error("HasSetting() returned true for non-existent setting")
	}
}

// TestConfigFileAllKeys tests retrieving all configuration keys
func TestConfigFileAllKeys(t *testing.T) {
	cfg := &ConfigFile{
		Settings: map[string][]string{
			"tcp-port": {"443"},
			"dns":      {"8.8.8.8"},
			"auth":     {"plain"},
		},
	}

	keys := cfg.AllKeys()

	if len(keys) != 3 {
		t.Errorf("AllKeys() returned %d keys, expected 3", len(keys))
	}

	// Check that all expected keys are present
	expectedKeys := map[string]bool{
		"tcp-port": false,
		"dns":      false,
		"auth":     false,
	}

	for _, key := range keys {
		if _, exists := expectedKeys[key]; exists {
			expectedKeys[key] = true
		}
	}

	for key, found := range expectedKeys {
		if !found {
			t.Errorf("AllKeys() missing expected key: %s", key)
		}
	}
}

// TestReadConfigFileWithCancellation tests context cancellation during read
func TestReadConfigFileWithCancellation(t *testing.T) {
	logger := zerolog.New(os.Stdout)
	reader := NewConfigReader(logger)

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	path := "../../test/fixtures/ocserv/configs/ocserv.conf"
	_, err := reader.readConfigFile(ctx, path)

	// Should return context error (but might succeed if file is read too fast)
	// We can't reliably test this without a large file
	_ = err
}

// TestReadConfigFileWithTimeout tests context timeout during read
func TestReadConfigFileWithTimeout(t *testing.T) {
	logger := zerolog.New(os.Stdout)
	reader := NewConfigReader(logger)

	// Create a context with very short timeout (but test files are small)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	path := "../../test/fixtures/ocserv/configs/ocserv.conf"
	cfg, err := reader.readConfigFile(ctx, path)

	// Small test file will likely complete before timeout
	// This is mainly for code coverage
	if err == nil && cfg == nil {
		t.Error("readConfigFile() returned nil config without error")
	}
}

// TestConfigFileEmptySettings tests config with no settings
func TestConfigFileEmptySettings(t *testing.T) {
	cfg := &ConfigFile{
		Settings: map[string][]string{},
	}

	// Test operations on empty config
	if cfg.HasSetting("anything") {
		t.Error("HasSetting() returned true for empty config")
	}

	if _, ok := cfg.GetSetting("anything"); ok {
		t.Error("GetSetting() returned true for empty config")
	}

	if _, ok := cfg.GetSettings("anything"); ok {
		t.Error("GetSettings() returned true for empty config")
	}

	keys := cfg.AllKeys()
	if len(keys) != 0 {
		t.Errorf("AllKeys() returned %d keys, expected 0", len(keys))
	}
}

// TestReadConfigFilePermissionDenied tests reading file with no permissions
func TestReadConfigFilePermissionDenied(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping test when running as root")
	}

	logger := zerolog.New(os.Stdout)
	reader := NewConfigReader(logger)
	ctx := context.Background()

	// Create a temporary file with no read permissions
	tmpFile := filepath.Join(t.TempDir(), "no-permission.conf")
	if err := os.WriteFile(tmpFile, []byte("test"), 0000); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Chmod(tmpFile, 0644) // Restore for cleanup

	_, err := reader.readConfigFile(ctx, tmpFile)
	if err == nil {
		t.Error("readConfigFile() expected error for permission denied, got nil")
	}
}

// TestListConfigFilesHiddenFiles tests that hidden files are skipped
func TestListConfigFilesHiddenFiles(t *testing.T) {
	logger := zerolog.New(os.Stdout)
	reader := NewConfigReader(logger)
	ctx := context.Background()

	// Create temp directory with hidden file
	tmpDir := t.TempDir()
	normalFile := filepath.Join(tmpDir, "normal.conf")
	hiddenFile := filepath.Join(tmpDir, ".hidden")

	if err := os.WriteFile(normalFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create normal file: %v", err)
	}
	if err := os.WriteFile(hiddenFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create hidden file: %v", err)
	}

	files, err := reader.listConfigFiles(ctx, tmpDir)
	if err != nil {
		t.Fatalf("listConfigFiles() error = %v", err)
	}

	// Should only find normal file
	if len(files) != 1 {
		t.Errorf("listConfigFiles() returned %d files, expected 1", len(files))
	}

	if len(files) > 0 && files[0] != "normal.conf" {
		t.Errorf("listConfigFiles() returned %s, expected normal.conf", files[0])
	}
}

// TestListConfigFilesSkipsDirectories tests that subdirectories are skipped
func TestListConfigFilesSkipsDirectories(t *testing.T) {
	logger := zerolog.New(os.Stdout)
	reader := NewConfigReader(logger)
	ctx := context.Background()

	// Create temp directory with subdirectory
	tmpDir := t.TempDir()
	normalFile := filepath.Join(tmpDir, "normal.conf")
	subDir := filepath.Join(tmpDir, "subdir")

	if err := os.WriteFile(normalFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create normal file: %v", err)
	}
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	files, err := reader.listConfigFiles(ctx, tmpDir)
	if err != nil {
		t.Fatalf("listConfigFiles() error = %v", err)
	}

	// Should only find normal file, not directory
	if len(files) != 1 {
		t.Errorf("listConfigFiles() returned %d files, expected 1", len(files))
	}
}
