package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGenerator(t *testing.T) {
	t.Run("creates generator with valid directories", func(t *testing.T) {
		tempDir := t.TempDir()
		perUserDir := filepath.Join(tempDir, "per-user")
		perGroupDir := filepath.Join(tempDir, "per-group")
		backupDir := filepath.Join(tempDir, "backups")

		gen, err := NewGenerator(perUserDir, perGroupDir, backupDir)

		require.NoError(t, err)
		assert.NotNil(t, gen)
		assert.Equal(t, perUserDir, gen.perUserDir)
		assert.Equal(t, perGroupDir, gen.perGroupDir)
		assert.Equal(t, backupDir, gen.backupDir)

		// Verify directories were created
		_, err = os.Stat(perUserDir)
		assert.NoError(t, err)
		_, err = os.Stat(perGroupDir)
		assert.NoError(t, err)
		_, err = os.Stat(backupDir)
		assert.NoError(t, err)
	})

	t.Run("handles empty directory paths", func(t *testing.T) {
		gen, err := NewGenerator("", "", "")

		require.NoError(t, err)
		assert.NotNil(t, gen)
	})
}

func TestGenerateUserConfig(t *testing.T) {
	tempDir := t.TempDir()
	gen, err := NewGenerator(
		filepath.Join(tempDir, "per-user"),
		filepath.Join(tempDir, "per-group"),
		filepath.Join(tempDir, "backups"),
	)
	require.NoError(t, err)

	t.Run("generates basic config successfully", func(t *testing.T) {
		config := UserConfig{
			Username: "john.doe",
			Routes:   []string{"10.0.0.0/8", "192.168.0.0/16"},
			DNSServers: []string{"8.8.8.8", "1.1.1.1"},
		}

		configPath, err := gen.GenerateUserConfig(config)

		require.NoError(t, err)
		assert.NotEmpty(t, configPath)

		// Verify file exists
		_, err = os.Stat(configPath)
		assert.NoError(t, err)

		// Read and verify content
		content, err := os.ReadFile(configPath)
		require.NoError(t, err)

		contentStr := string(content)
		assert.Contains(t, contentStr, "john.doe")
		assert.Contains(t, contentStr, "route = 10.0.0.0/8")
		assert.Contains(t, contentStr, "route = 192.168.0.0/16")
		assert.Contains(t, contentStr, "dns = 8.8.8.8")
		assert.Contains(t, contentStr, "dns = 1.1.1.1")
	})

	t.Run("generates config with split DNS", func(t *testing.T) {
		config := UserConfig{
			Username: "jane.smith",
			SplitDNS: []string{"internal.company.com", "vpn.example.org"},
		}

		configPath, err := gen.GenerateUserConfig(config)

		require.NoError(t, err)

		content, err := os.ReadFile(configPath)
		require.NoError(t, err)

		contentStr := string(content)
		assert.Contains(t, contentStr, "split-dns = internal.company.com")
		assert.Contains(t, contentStr, "split-dns = vpn.example.org")
	})

	t.Run("generates config with restrictions", func(t *testing.T) {
		config := UserConfig{
			Username:         "restricted.user",
			Routes:           []string{"172.16.0.0/12"},
			RestrictToRoutes: true,
			MaxSameClients:   2,
		}

		configPath, err := gen.GenerateUserConfig(config)

		require.NoError(t, err)

		content, err := os.ReadFile(configPath)
		require.NoError(t, err)

		contentStr := string(content)
		assert.Contains(t, contentStr, "restrict-user-to-routes = true")
		assert.Contains(t, contentStr, "max-same-clients = 2")
	})

	t.Run("generates config with explicit IPs", func(t *testing.T) {
		config := UserConfig{
			Username:     "static.ip.user",
			ExplicitIPv4: "10.10.10.100",
			ExplicitIPv6: "fd00::100",
		}

		configPath, err := gen.GenerateUserConfig(config)

		require.NoError(t, err)

		content, err := os.ReadFile(configPath)
		require.NoError(t, err)

		contentStr := string(content)
		assert.Contains(t, contentStr, "explicit-ipv4 = 10.10.10.100")
		assert.Contains(t, contentStr, "explicit-ipv6 = fd00::100")
	})

	t.Run("generates config with rate limits", func(t *testing.T) {
		config := UserConfig{
			Username: "rate.limited",
			RXPerSec: 1048576,  // 1 MB/s
			TXPerSec: 524288,   // 512 KB/s
		}

		configPath, err := gen.GenerateUserConfig(config)

		require.NoError(t, err)

		content, err := os.ReadFile(configPath)
		require.NoError(t, err)

		contentStr := string(content)
		assert.Contains(t, contentStr, "rx-per-sec = 1048576")
		assert.Contains(t, contentStr, "tx-per-sec = 524288")
	})

	t.Run("generates config with timeouts", func(t *testing.T) {
		config := UserConfig{
			Username:          "timeout.user",
			IdleTimeout:       300,
			MobileIdleTimeout: 600,
			SessionTimeout:    86400,
		}

		configPath, err := gen.GenerateUserConfig(config)

		require.NoError(t, err)

		content, err := os.ReadFile(configPath)
		require.NoError(t, err)

		contentStr := string(content)
		assert.Contains(t, contentStr, "idle-timeout = 300")
		assert.Contains(t, contentStr, "mobile-idle-timeout = 600")
		assert.Contains(t, contentStr, "session-timeout = 86400")
	})

	t.Run("generates config with custom params", func(t *testing.T) {
		config := UserConfig{
			Username: "custom.user",
			CustomParams: map[string]string{
				"banner": "Welcome to VPN",
				"mtu":    "1400",
			},
		}

		configPath, err := gen.GenerateUserConfig(config)

		require.NoError(t, err)

		content, err := os.ReadFile(configPath)
		require.NoError(t, err)

		contentStr := string(content)
		assert.Contains(t, contentStr, "banner = Welcome to VPN")
		assert.Contains(t, contentStr, "mtu = 1400")
	})

	t.Run("generates config with no-route", func(t *testing.T) {
		config := UserConfig{
			Username: "no.route.user",
			NoRoute:  true,
		}

		configPath, err := gen.GenerateUserConfig(config)

		require.NoError(t, err)

		content, err := os.ReadFile(configPath)
		require.NoError(t, err)

		contentStr := string(content)
		assert.Contains(t, contentStr, "no-route = true")
	})

	t.Run("creates backup when updating existing config", func(t *testing.T) {
		config := UserConfig{
			Username: "backup.test",
			Routes:   []string{"10.0.0.0/8"},
		}

		// Create initial config
		_, err := gen.GenerateUserConfig(config)
		require.NoError(t, err)

		// Update config
		config.Routes = []string{"192.168.0.0/16"}
		_, err = gen.GenerateUserConfig(config)
		require.NoError(t, err)

		// Verify backup exists
		backupFiles, err := os.ReadDir(gen.backupDir)
		require.NoError(t, err)
		assert.NotEmpty(t, backupFiles)

		// Verify backup contains old config
		var backupFound bool
		for _, file := range backupFiles {
			if strings.HasPrefix(file.Name(), "backup.test.") {
				backupFound = true
				break
			}
		}
		assert.True(t, backupFound, "backup file should exist")
	})

	t.Run("returns error for empty username", func(t *testing.T) {
		config := UserConfig{
			Username: "",
		}

		_, err := gen.GenerateUserConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "username cannot be empty")
	})

	t.Run("returns error for invalid routes", func(t *testing.T) {
		config := UserConfig{
			Username: "invalid.routes",
			Routes:   []string{"invalid-route"},
		}

		_, err := gen.GenerateUserConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid routes")
	})

	t.Run("returns error for invalid DNS", func(t *testing.T) {
		config := UserConfig{
			Username:   "invalid.dns",
			DNSServers: []string{"invalid-dns"},
		}

		_, err := gen.GenerateUserConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid DNS servers")
	})

	t.Run("returns error for invalid explicit IPv4", func(t *testing.T) {
		config := UserConfig{
			Username:     "invalid.ipv4",
			ExplicitIPv4: "999.999.999.999",
		}

		_, err := gen.GenerateUserConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid explicit IPv4")
	})
}

func TestDeleteUserConfig(t *testing.T) {
	tempDir := t.TempDir()
	gen, err := NewGenerator(
		filepath.Join(tempDir, "per-user"),
		filepath.Join(tempDir, "per-group"),
		filepath.Join(tempDir, "backups"),
	)
	require.NoError(t, err)

	t.Run("deletes existing config", func(t *testing.T) {
		config := UserConfig{
			Username: "delete.test",
			Routes:   []string{"10.0.0.0/8"},
		}

		configPath, err := gen.GenerateUserConfig(config)
		require.NoError(t, err)

		// Verify config exists
		_, err = os.Stat(configPath)
		require.NoError(t, err)

		// Delete config
		err = gen.DeleteUserConfig("delete.test")
		require.NoError(t, err)

		// Verify config deleted
		_, err = os.Stat(configPath)
		assert.True(t, os.IsNotExist(err))

		// Verify backup exists
		backupFiles, err := os.ReadDir(gen.backupDir)
		require.NoError(t, err)
		assert.NotEmpty(t, backupFiles)
	})

	t.Run("handles non-existent config", func(t *testing.T) {
		err := gen.DeleteUserConfig("non.existent")
		assert.NoError(t, err) // Should not error
	})

	t.Run("returns error for empty username", func(t *testing.T) {
		err := gen.DeleteUserConfig("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "username cannot be empty")
	})
}

func TestUserConfigExists(t *testing.T) {
	tempDir := t.TempDir()
	gen, err := NewGenerator(
		filepath.Join(tempDir, "per-user"),
		filepath.Join(tempDir, "per-group"),
		filepath.Join(tempDir, "backups"),
	)
	require.NoError(t, err)

	t.Run("returns true for existing config", func(t *testing.T) {
		config := UserConfig{
			Username: "exists.test",
			Routes:   []string{"10.0.0.0/8"},
		}

		_, err := gen.GenerateUserConfig(config)
		require.NoError(t, err)

		exists := gen.UserConfigExists("exists.test")
		assert.True(t, exists)
	})

	t.Run("returns false for non-existent config", func(t *testing.T) {
		exists := gen.UserConfigExists("non.existent")
		assert.False(t, exists)
	})
}

func TestValidateRoutes(t *testing.T) {
	tests := []struct {
		name    string
		routes  []string
		wantErr bool
	}{
		{
			name:    "valid routes",
			routes:  []string{"10.0.0.0/8", "192.168.0.0/16", "172.16.0.0/12"},
			wantErr: false,
		},
		{
			name:    "valid single route",
			routes:  []string{"192.168.1.0/24"},
			wantErr: false,
		},
		{
			name:    "empty routes",
			routes:  []string{},
			wantErr: false,
		},
		{
			name:    "invalid CIDR format",
			routes:  []string{"10.0.0.0"},
			wantErr: true,
		},
		{
			name:    "invalid IP address",
			routes:  []string{"999.999.999.999/8"},
			wantErr: true,
		},
		{
			name:    "invalid prefix length",
			routes:  []string{"10.0.0.0/99"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRoutes(tt.routes)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateIPAddresses(t *testing.T) {
	tests := []struct {
		name    string
		ips     []string
		wantErr bool
	}{
		{
			name:    "valid IPv4 addresses",
			ips:     []string{"8.8.8.8", "1.1.1.1", "192.168.1.1"},
			wantErr: false,
		},
		{
			name:    "valid IPv6 addresses",
			ips:     []string{"2001:4860:4860::8888", "fd00::1"},
			wantErr: false,
		},
		{
			name:    "mixed IPv4 and IPv6",
			ips:     []string{"8.8.8.8", "2001:4860:4860::8888"},
			wantErr: false,
		},
		{
			name:    "empty list",
			ips:     []string{},
			wantErr: false,
		},
		{
			name:    "invalid IP address",
			ips:     []string{"invalid-ip"},
			wantErr: true,
		},
		{
			name:    "invalid IPv4",
			ips:     []string{"999.999.999.999"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateIPAddresses(tt.ips)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetUserConfigPath(t *testing.T) {
	tempDir := t.TempDir()
	gen, err := NewGenerator(
		filepath.Join(tempDir, "per-user"),
		"",
		"",
	)
	require.NoError(t, err)

	path := gen.GetUserConfigPath("test.user")
	expected := filepath.Join(tempDir, "per-user", "test.user")
	assert.Equal(t, expected, path)
}

func TestGeneratorThreadSafety(t *testing.T) {
	tempDir := t.TempDir()
	gen, err := NewGenerator(
		filepath.Join(tempDir, "per-user"),
		filepath.Join(tempDir, "per-group"),
		filepath.Join(tempDir, "backups"),
	)
	require.NoError(t, err)

	// Test concurrent writes
	const numGoroutines = 10
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			config := UserConfig{
				Username: "concurrent.user",
				Routes:   []string{"10.0.0.0/8"},
			}

			_, err := gen.GenerateUserConfig(config)
			assert.NoError(t, err)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify config exists
	assert.True(t, gen.UserConfigExists("concurrent.user"))
}
