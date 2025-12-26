package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
	"time"

	"github.com/cockroachdb/errors"
)

// PerUserConfig represents per-user ocserv configuration
type PerUserConfig struct {
	Username string
	Routes   []string
	DNS      []string
	// Security settings
	RestrictUserToRoutes bool
	MaxSameClients       int
	// Custom directives
	CustomDirectives map[string]string
}

// PerGroupConfig represents per-group ocserv configuration
type PerGroupConfig struct {
	GroupName        string
	Routes           []string
	DNS              []string
	SplitDNS         []string
	MaxSameClients   int
	RestrictToRoutes bool
	CustomDirectives map[string]string
}

// Generator generates per-user and per-group ocserv configuration files
type Generator struct {
	perUserDir  string
	perGroupDir string
	backupDir   string
	templates   *Templates
}

// NewGenerator creates a new configuration generator
func NewGenerator(perUserDir, perGroupDir, backupDir string) (*Generator, error) {
	if perUserDir == "" {
		return nil, errors.New("per-user directory is required")
	}

	// Ensure directories exist
	for _, dir := range []string{perUserDir, perGroupDir, backupDir} {
		if dir != "" {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return nil, errors.Wrapf(err, "create directory %s", dir)
			}
		}
	}

	templates, err := NewTemplates()
	if err != nil {
		return nil, errors.Wrap(err, "initialize templates")
	}

	return &Generator{
		perUserDir:  perUserDir,
		perGroupDir: perGroupDir,
		backupDir:   backupDir,
		templates:   templates,
	}, nil
}

// GenerateUserConfig generates a per-user configuration file
func (g *Generator) GenerateUserConfig(cfg *PerUserConfig) error {
	if cfg.Username == "" {
		return errors.New("username is required")
	}

	// Validate routes
	if err := ValidateRoutes(cfg.Routes); err != nil {
		return errors.Wrap(err, "invalid routes")
	}

	// Validate DNS servers
	if err := ValidateDNSServers(cfg.DNS); err != nil {
		return errors.Wrap(err, "invalid DNS servers")
	}

	// Generate config content
	content, err := g.templates.RenderUserConfig(cfg)
	if err != nil {
		return errors.Wrap(err, "render user config")
	}

	// Determine file path
	configPath := filepath.Join(g.perUserDir, cfg.Username)

	// Backup existing config if it exists
	if err := g.backupConfig(configPath); err != nil {
		return errors.Wrap(err, "backup existing config")
	}

	// Write new config
	if err := os.WriteFile(configPath, content, 0644); err != nil {
		return errors.Wrapf(err, "write config to %s", configPath)
	}

	return nil
}

// GenerateGroupConfig generates a per-group configuration file
func (g *Generator) GenerateGroupConfig(cfg *PerGroupConfig) error {
	if cfg.GroupName == "" {
		return errors.New("group name is required")
	}
	if g.perGroupDir == "" {
		return errors.New("per-group directory not configured")
	}

	// Validate routes
	if err := ValidateRoutes(cfg.Routes); err != nil {
		return errors.Wrap(err, "invalid routes")
	}

	// Validate DNS servers
	if err := ValidateDNSServers(cfg.DNS); err != nil {
		return errors.Wrap(err, "invalid DNS servers")
	}

	// Generate config content
	content, err := g.templates.RenderGroupConfig(cfg)
	if err != nil {
		return errors.Wrap(err, "render group config")
	}

	// Determine file path
	configPath := filepath.Join(g.perGroupDir, cfg.GroupName)

	// Backup existing config if it exists
	if err := g.backupConfig(configPath); err != nil {
		return errors.Wrap(err, "backup existing config")
	}

	// Write new config
	if err := os.WriteFile(configPath, content, 0644); err != nil {
		return errors.Wrapf(err, "write config to %s", configPath)
	}

	return nil
}

// DeleteUserConfig deletes a per-user configuration file
func (g *Generator) DeleteUserConfig(username string) error {
	if username == "" {
		return errors.New("username is required")
	}

	configPath := filepath.Join(g.perUserDir, username)

	// Backup before deleting
	if err := g.backupConfig(configPath); err != nil {
		return errors.Wrap(err, "backup config before deletion")
	}

	// Delete config file
	if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
		return errors.Wrapf(err, "delete config %s", configPath)
	}

	return nil
}

// DeleteGroupConfig deletes a per-group configuration file
func (g *Generator) DeleteGroupConfig(groupName string) error {
	if groupName == "" {
		return errors.New("group name is required")
	}
	if g.perGroupDir == "" {
		return errors.New("per-group directory not configured")
	}

	configPath := filepath.Join(g.perGroupDir, groupName)

	// Backup before deleting
	if err := g.backupConfig(configPath); err != nil {
		return errors.Wrap(err, "backup config before deletion")
	}

	// Delete config file
	if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
		return errors.Wrapf(err, "delete config %s", configPath)
	}

	return nil
}

// ListUserConfigs returns a list of all per-user config files
func (g *Generator) ListUserConfigs() ([]string, error) {
	entries, err := os.ReadDir(g.perUserDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, errors.Wrapf(err, "read directory %s", g.perUserDir)
	}

	usernames := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			usernames = append(usernames, entry.Name())
		}
	}

	return usernames, nil
}

// backupConfig creates a backup of the config file if it exists
func (g *Generator) backupConfig(configPath string) error {
	// Skip if backup directory is not configured
	if g.backupDir == "" {
		return nil
	}

	// Check if config exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil
	}

	// Read existing config
	content, err := os.ReadFile(configPath)
	if err != nil {
		return errors.Wrapf(err, "read config %s", configPath)
	}

	// Generate backup filename with timestamp
	timestamp := time.Now().Format("20060102-150405")
	filename := filepath.Base(configPath)
	backupPath := filepath.Join(g.backupDir, fmt.Sprintf("%s.%s.bak", filename, timestamp))

	// Write backup
	if err := os.WriteFile(backupPath, content, 0644); err != nil {
		return errors.Wrapf(err, "write backup to %s", backupPath)
	}

	return nil
}

// Templates manages ocserv config templates
type Templates struct {
	userTemplate  *template.Template
	groupTemplate *template.Template
}

// NewTemplates creates and parses config templates
func NewTemplates() (*Templates, error) {
	userTpl, err := template.New("user").Funcs(templateFuncs).Parse(userConfigTemplate)
	if err != nil {
		return nil, errors.Wrap(err, "parse user template")
	}

	groupTpl, err := template.New("group").Funcs(templateFuncs).Parse(groupConfigTemplate)
	if err != nil {
		return nil, errors.Wrap(err, "parse group template")
	}

	return &Templates{
		userTemplate:  userTpl,
		groupTemplate: groupTpl,
	}, nil
}

// RenderUserConfig renders a per-user config template
func (t *Templates) RenderUserConfig(cfg *PerUserConfig) ([]byte, error) {
	var buf bytes.Buffer
	if err := t.userTemplate.Execute(&buf, cfg); err != nil {
		return nil, errors.Wrap(err, "execute user template")
	}
	return buf.Bytes(), nil
}

// RenderGroupConfig renders a per-group config template
func (t *Templates) RenderGroupConfig(cfg *PerGroupConfig) ([]byte, error) {
	var buf bytes.Buffer
	if err := t.groupTemplate.Execute(&buf, cfg); err != nil {
		return nil, errors.Wrap(err, "execute group template")
	}
	return buf.Bytes(), nil
}
