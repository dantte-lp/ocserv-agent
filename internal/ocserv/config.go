package ocserv

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog"
)

// ConfigFile represents a parsed ocserv configuration file
type ConfigFile struct {
	// Path is the file path that was read
	Path string

	// Settings is a map of configuration keys to their values
	// Keys may have multiple values (e.g., routes, dns)
	Settings map[string][]string

	// RawLines contains the raw lines from the file (for preservation)
	RawLines []string
}

// ConfigReader handles reading ocserv configuration files
type ConfigReader struct {
	logger zerolog.Logger
}

// NewConfigReader creates a new ConfigReader instance
func NewConfigReader(logger zerolog.Logger) *ConfigReader {
	return &ConfigReader{
		logger: logger,
	}
}

// ReadOcservConf reads the main ocserv configuration file
func (r *ConfigReader) ReadOcservConf(ctx context.Context, path string) (*ConfigFile, error) {
	r.logger.Debug().
		Str("path", path).
		Msg("Reading ocserv.conf")

	return r.readConfigFile(ctx, path)
}

// ReadUserConfig reads a per-user configuration file
func (r *ConfigReader) ReadUserConfig(ctx context.Context, baseDir, username string) (*ConfigFile, error) {
	path := filepath.Join(baseDir, username)

	r.logger.Debug().
		Str("path", path).
		Str("username", username).
		Msg("Reading per-user config")

	return r.readConfigFile(ctx, path)
}

// ReadGroupConfig reads a per-group configuration file
func (r *ConfigReader) ReadGroupConfig(ctx context.Context, baseDir, groupname string) (*ConfigFile, error) {
	path := filepath.Join(baseDir, groupname)

	r.logger.Debug().
		Str("path", path).
		Str("groupname", groupname).
		Msg("Reading per-group config")

	return r.readConfigFile(ctx, path)
}

// ListUserConfigs lists all available per-user configuration files
func (r *ConfigReader) ListUserConfigs(ctx context.Context, baseDir string) ([]string, error) {
	r.logger.Debug().
		Str("baseDir", baseDir).
		Msg("Listing per-user configs")

	return r.listConfigFiles(ctx, baseDir)
}

// ListGroupConfigs lists all available per-group configuration files
func (r *ConfigReader) ListGroupConfigs(ctx context.Context, baseDir string) ([]string, error) {
	r.logger.Debug().
		Str("baseDir", baseDir).
		Msg("Listing per-group configs")

	return r.listConfigFiles(ctx, baseDir)
}

// readConfigFile reads and parses an ocserv configuration file
func (r *ConfigReader) readConfigFile(ctx context.Context, path string) (*ConfigFile, error) {
	// Check if file exists
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config file not found: %s", path)
		}
		return nil, fmt.Errorf("failed to stat config file: %w", err)
	}

	// Open file
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	cfg := &ConfigFile{
		Path:     path,
		Settings: make(map[string][]string),
		RawLines: make([]string, 0),
	}

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		cfg.RawLines = append(cfg.RawLines, line)

		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Parse line
		key, value, err := r.parseLine(line)
		if err != nil {
			r.logger.Warn().
				Err(err).
				Str("path", path).
				Int("line", lineNum).
				Str("content", line).
				Msg("Failed to parse line, skipping")
			continue
		}

		// Skip empty lines and comments
		if key == "" {
			continue
		}

		// Add to settings (support multi-value keys)
		cfg.Settings[key] = append(cfg.Settings[key], value)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	r.logger.Info().
		Str("path", path).
		Int("settings_count", len(cfg.Settings)).
		Int("lines", lineNum).
		Msg("Successfully read config file")

	return cfg, nil
}

// parseLine parses a single line from an ocserv config file
// Returns: (key, value, error)
func (r *ConfigReader) parseLine(line string) (string, string, error) {
	// Trim whitespace
	line = strings.TrimSpace(line)

	// Skip empty lines
	if line == "" {
		return "", "", nil
	}

	// Skip comments
	if strings.HasPrefix(line, "#") {
		return "", "", nil
	}

	// Handle inline comments - remove everything after #
	if idx := strings.Index(line, "#"); idx != -1 {
		line = strings.TrimSpace(line[:idx])
		if line == "" {
			return "", "", nil
		}
	}

	// Split on = or whitespace
	var key, value string

	if strings.Contains(line, "=") {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return "", "", fmt.Errorf("invalid line format: %s", line)
		}
		key = strings.TrimSpace(parts[0])
		value = strings.TrimSpace(parts[1])
	} else {
		// Some configs use space-separated format
		parts := strings.Fields(line)
		if len(parts) < 2 {
			return "", "", fmt.Errorf("invalid line format: %s", line)
		}
		key = parts[0]
		value = strings.Join(parts[1:], " ")
	}

	// Validate key
	if key == "" {
		return "", "", fmt.Errorf("empty key in line: %s", line)
	}

	return key, value, nil
}

// listConfigFiles lists all files in a directory (non-recursive)
func (r *ConfigReader) listConfigFiles(ctx context.Context, baseDir string) ([]string, error) {
	// Check if directory exists
	info, err := os.Stat(baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil // Return empty list if dir doesn't exist
		}
		return nil, fmt.Errorf("failed to stat directory: %w", err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory: %s", baseDir)
	}

	// Read directory entries
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Skip directories and hidden files
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		files = append(files, entry.Name())
	}

	return files, nil
}

// GetSetting retrieves a single-value setting from the config
func (cfg *ConfigFile) GetSetting(key string) (string, bool) {
	values, ok := cfg.Settings[key]
	if !ok || len(values) == 0 {
		return "", false
	}
	return values[0], true
}

// GetSettings retrieves all values for a multi-value setting
func (cfg *ConfigFile) GetSettings(key string) ([]string, bool) {
	values, ok := cfg.Settings[key]
	if !ok {
		return nil, false
	}
	return values, true
}

// HasSetting checks if a setting exists
func (cfg *ConfigFile) HasSetting(key string) bool {
	_, ok := cfg.Settings[key]
	return ok
}

// AllKeys returns all configuration keys
func (cfg *ConfigFile) AllKeys() []string {
	keys := make([]string, 0, len(cfg.Settings))
	for key := range cfg.Settings {
		keys = append(keys, key)
	}
	return keys
}
