# Test Fixtures - ocserv Configuration Files

This directory contains sample ocserv configuration files for testing the ConfigReader.

## Files

### ocserv/ocserv.conf
**Source:** OpenConnect VPN Server 1.3.0 on Oracle Linux 10
**Date:** 2025-10-23
**Description:** Official production configuration from fresh ocserv 1.3.0 installation.

This is the complete default configuration from ocserv 1.3.0 with all 764 lines and comprehensive documentation. Use this as the authoritative reference for:
- All supported configuration directives
- Default values and recommendations
- Detailed comments explaining each option
- Real-world production settings

**Key features in 1.3.0:**
- HTTP security headers support
- Camouflage mode (hide VPN as web server)
- Ban system with score-based blocking
- Network namespace support
- Load balancer connection draining
- Virtual host support

**GnuTLS version:** 3.8.9 (compiled with 3.8.8)

### config-per-user/
Per-user configuration examples:
- **testuser** - Standard user with custom routes and DNS
- **admin** - Admin user with full tunnel and extended timeouts

### config-per-group/
Per-group configuration examples:
- **developers** - Developer group with internal network access
- **remote-workers** - Remote workers with corporate network only

## Available ocserv 1.3.0 Utilities

After installation, the following utilities are available:

- `occtl` - Control utility for managing running server (show users, disconnect, etc.)
- `ocpasswd` - Password management utility (not yet wrapped by ocserv-agent)
- `ocserv` - Main VPN server daemon
- `ocserv-genkey` - Key generation utility
- `ocserv-script` - Connect/disconnect script hook
- `ocserv-worker` - Worker process

## Testing ConfigReader

All files in this directory can be used to test the ConfigReader implementation in `internal/ocserv/config.go`.

Example usage:
```go
reader := ocserv.NewConfigReader(logger)

// Read main config
cfg, err := reader.ReadOcservConf(ctx, "test/fixtures/ocserv_13.conf")

// Read per-user config
userCfg, err := reader.ReadUserConfig(ctx, "test/fixtures/ocserv/config-per-user", "testuser")

// List available user configs
users, err := reader.ListUserConfigs(ctx, "test/fixtures/ocserv/config-per-user")
```
