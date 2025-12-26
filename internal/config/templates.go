package config

import (
	"text/template"
	"time"
)

// templateFuncs returns template functions for config generation
var templateFuncs = template.FuncMap{
	"now": func() string {
		return time.Now().Format(time.RFC3339)
	},
}

// userConfigTemplate defines the template for per-user ocserv configuration
const userConfigTemplate = `# Auto-generated per-user configuration for ocserv
# User: {{.Username}}
# Generated: {{now}}
# WARNING: This file is managed by ocserv-agent. Manual changes will be overwritten.

{{if .Routes -}}
# Routes pushed to client
{{range .Routes -}}
route = {{.}}
{{end}}
{{- end}}

{{if .DNS -}}
# DNS servers
{{range .DNS -}}
dns = {{.}}
{{end}}
{{- end}}

{{if .RestrictUserToRoutes -}}
# Security: restrict user to pushed routes only
restrict-user-to-routes = true
{{- end}}

{{if .MaxSameClients -}}
# Maximum concurrent connections for this user
max-same-clients = {{.MaxSameClients}}
{{- end}}

{{if .CustomDirectives -}}
# Custom directives
{{range $key, $value := .CustomDirectives -}}
{{$key}} = {{$value}}
{{end}}
{{- end}}
`

// groupConfigTemplate defines the template for per-group ocserv configuration
const groupConfigTemplate = `# Auto-generated per-group configuration for ocserv
# Group: {{.GroupName}}
# Generated: {{now}}
# WARNING: This file is managed by ocserv-agent. Manual changes will be overwritten.

{{if .Routes -}}
# Routes pushed to group members
{{range .Routes -}}
route = {{.}}
{{end}}
{{- end}}

{{if .DNS -}}
# DNS servers
{{range .DNS -}}
dns = {{.}}
{{end}}
{{- end}}

{{if .SplitDNS -}}
# Split DNS domains
{{range .SplitDNS -}}
split-dns = {{.}}
{{end}}
{{- end}}

{{if .MaxSameClients -}}
# Maximum concurrent connections per user in this group
max-same-clients = {{.MaxSameClients}}
{{- end}}

{{if .RestrictToRoutes -}}
# Security: restrict group to pushed routes only
restrict-user-to-routes = true
{{- end}}

{{if .CustomDirectives -}}
# Custom directives
{{range $key, $value := .CustomDirectives -}}
{{$key}} = {{$value}}
{{end}}
{{- end}}
`

// DefaultUserConfig returns a default per-user configuration
func DefaultUserConfig(username string) *PerUserConfig {
	return &PerUserConfig{
		Username:             username,
		Routes:               []string{},
		DNS:                  []string{"8.8.8.8", "8.8.4.4"},
		RestrictUserToRoutes: true,
		MaxSameClients:       2,
		CustomDirectives:     make(map[string]string),
	}
}

// DefaultGroupConfig returns a default per-group configuration
func DefaultGroupConfig(groupName string) *PerGroupConfig {
	return &PerGroupConfig{
		GroupName:        groupName,
		Routes:           []string{},
		DNS:              []string{"8.8.8.8", "8.8.4.4"},
		SplitDNS:         []string{},
		MaxSameClients:   2,
		RestrictToRoutes: true,
		CustomDirectives: make(map[string]string),
	}
}

// CommonRoutes returns commonly used route configurations
type CommonRoutes struct{}

var Routes = CommonRoutes{}

// PrivateNetworks returns RFC1918 private network routes
func (CommonRoutes) PrivateNetworks() []string {
	return []string{
		"10.0.0.0/255.0.0.0",
		"172.16.0.0/255.240.0.0",
		"192.168.0.0/255.255.0.0",
	}
}

// FullTunnel returns a route that tunnels all traffic
func (CommonRoutes) FullTunnel() []string {
	return []string{
		"0.0.0.0/0.0.0.0",
	}
}

// NoRoute explicitly denies a route
func (CommonRoutes) NoRoute(cidr string) string {
	return "no-" + cidr
}

// CommonDNS returns commonly used DNS server configurations
type CommonDNS struct{}

var DNS = CommonDNS{}

// Google returns Google Public DNS servers
func (CommonDNS) Google() []string {
	return []string{"8.8.8.8", "8.8.4.4"}
}

// Cloudflare returns Cloudflare DNS servers
func (CommonDNS) Cloudflare() []string {
	return []string{"1.1.1.1", "1.0.0.1"}
}

// Quad9 returns Quad9 DNS servers
func (CommonDNS) Quad9() []string {
	return []string{"9.9.9.9", "149.112.112.112"}
}
