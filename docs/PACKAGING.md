# Package Building Guide

## Overview

ocserv-agent supports native package builds for multiple platforms:
- **RPM**: RHEL 8/9/10, Oracle Linux 8/9/10, Rocky Linux
- **DEB**: Debian 12/13, Ubuntu 24.04 LTS
- **FreeBSD**: amd64 and arm64 architectures

All packages are built using dedicated self-hosted GitHub Actions runners with platform-specific tooling.

## Build Infrastructure

### RPM Builds (Oracle Linux Runner)

**Runner:** `github-runner` (Oracle Linux 10)
**Labels:** `self-hosted`, `oracle-linux`, `rpm-build`, `mock`

**Tools:**
- `mock` - Clean chroot-based RPM builds
- `rpm-build` - RPM packaging tools
- `rpmdevtools` - RPM development utilities
- `rpmlint` - RPM quality checker

**Target Distributions:**
- Enterprise Linux 8 (RHEL 8, Oracle Linux 8, Rocky Linux 8)
- Enterprise Linux 9 (RHEL 9, Oracle Linux 9, Rocky Linux 9)
- Enterprise Linux 10 (RHEL 10, Oracle Linux 10)

### DEB Builds (Debian Runner)

**Runner:** `github-runner-debian` (Debian Trixie)
**Labels:** `self-hosted`, `debian`

**Tools:**
- `dpkg-dev` - Debian package development tools
- `debhelper` - Helper scripts for debian/rules
- `devscripts` - Package building scripts

**Target Distributions:**
- Debian 12 (Bookworm)
- Debian 13 (Trixie)
- Ubuntu 24.04 LTS (Noble Numbat)

### FreeBSD Builds (Cross-compilation)

**Runner:** Any (uses Go cross-compilation)
**Method:** GOOS=freebsd GOARCH={amd64,arm64}

**Supported:**
- FreeBSD 14.x (amd64, arm64)

## Package Structure

### RPM Package

#### File Layout
```
/usr/sbin/ocserv-agent                      # Main binary
/etc/ocserv-agent/config.yaml.example       # Config template
/etc/ocserv-agent/certs/                    # Certificates directory
/var/log/ocserv-agent/                      # Log directory
/var/backups/ocserv-agent/                  # Backup directory
/lib/systemd/system/ocserv-agent.service    # Systemd unit
```

#### User & Group
```
User:  ocserv-agent (system user)
Group: ocserv-agent (system group)
Home:  /etc/ocserv-agent
Shell: /sbin/nologin
```

#### Permissions
```
Binary:     /usr/sbin/ocserv-agent           (755 root:root)
Config:     /etc/ocserv-agent/               (750 ocserv-agent:ocserv-agent)
Config:     /etc/ocserv-agent/config.yaml    (640 ocserv-agent:ocserv-agent, noreplace)
Certs:      /etc/ocserv-agent/certs/         (750 ocserv-agent:ocserv-agent)
Logs:       /var/log/ocserv-agent/           (755 ocserv-agent:ocserv-agent)
Backups:    /var/backups/ocserv-agent/       (755 ocserv-agent:ocserv-agent)
```

#### SELinux Support (RHEL/Oracle Linux)

Automatic SELinux context configuration in `%post`:

```bash
# Binary context
semanage fcontext -a -t bin_t '/usr/sbin/ocserv-agent'
restorecon -v /usr/sbin/ocserv-agent

# Config context
semanage fcontext -a -t etc_t '/etc/ocserv-agent(/.*)?'
restorecon -Rv /etc/ocserv-agent

# Log context
semanage fcontext -a -t var_log_t '/var/log/ocserv-agent(/.*)?'
restorecon -Rv /var/log/ocserv-agent

# Network permissions
setsebool -P nis_enabled 1
```

#### RPM Spec Highlights

**%pre (before installation):**
- Creates `ocserv-agent` user and group

**%post (after installation):**
- Runs `%systemd_post` macro
- Sets SELinux contexts
- Sets directory ownership
- Configures permissions

**%preun (before uninstallation):**
- Runs `%systemd_preun` macro
- Stops service if removing

**%postun (after uninstallation):**
- Runs `%systemd_postun_with_restart` macro
- Removes user only on complete removal (not upgrade)

#### Dependencies
```
BuildRequires: golang >= 1.25
BuildRequires: protobuf-compiler
BuildRequires: systemd-rpm-macros

Requires: ocserv
```

### DEB Package

#### File Layout
```
/usr/sbin/ocserv-agent                      # Main binary
/etc/ocserv-agent/config.yaml.example       # Config template
/etc/ocserv-agent/certs/                    # Certificates directory
/var/log/ocserv-agent/                      # Log directory
/var/backups/ocserv-agent/                  # Backup directory
/lib/systemd/system/ocserv-agent.service    # Systemd unit
/usr/share/doc/ocserv-agent/README.md       # Documentation
/usr/share/doc/ocserv-agent/LICENSE         # License
```

#### User & Group
```
User:  ocserv-agent (system user)
Group: ocserv-agent (system group)
Home:  /etc/ocserv-agent
Shell: /usr/sbin/nologin
```

#### Maintainer Scripts

**preinst (before installation):**
```bash
#!/bin/sh
set -e
if [ "$1" = "install" ] || [ "$1" = "upgrade" ]; then
    # Create user and group
    if ! getent group ocserv-agent >/dev/null; then
        addgroup --system ocserv-agent
    fi
    if ! getent passwd ocserv-agent >/dev/null; then
        adduser --system --home /etc/ocserv-agent --no-create-home \
            --ingroup ocserv-agent --disabled-password \
            --shell /usr/sbin/nologin \
            --gecos "ocserv-agent service user" ocserv-agent
    fi
fi
```

**postinst (after installation):**
```bash
#!/bin/sh
set -e
if [ "$1" = "configure" ]; then
    # Set ownership
    chown -R ocserv-agent:ocserv-agent /etc/ocserv-agent
    chown -R ocserv-agent:ocserv-agent /var/log/ocserv-agent
    chown -R ocserv-agent:ocserv-agent /var/backups/ocserv-agent

    # Set permissions
    chmod 750 /etc/ocserv-agent
    chmod 750 /etc/ocserv-agent/certs
    chmod 640 /etc/ocserv-agent/config.yaml.example

    # Reload systemd
    systemctl daemon-reload || true
fi
```

**prerm (before removal):**
```bash
#!/bin/sh
set -e
if [ "$1" = "remove" ]; then
    systemctl stop ocserv-agent.service 2>/dev/null || true
    systemctl disable ocserv-agent.service 2>/dev/null || true
fi
```

**postrm (after removal):**
```bash
#!/bin/sh
set -e
if [ "$1" = "purge" ]; then
    # Remove user and group only on purge
    if getent passwd ocserv-agent >/dev/null; then
        deluser --system ocserv-agent 2>/dev/null || true
    fi
    if getent group ocserv-agent >/dev/null; then
        delgroup --system ocserv-agent 2>/dev/null || true
    fi

    # Remove directories
    rm -rf /var/log/ocserv-agent || true
    rm -rf /var/backups/ocserv-agent || true
fi
systemctl daemon-reload || true
```

#### Dependencies
```
Depends: ocserv, adduser
```

### FreeBSD Package

#### File Layout
```
/usr/local/sbin/ocserv-agent                      # Main binary
/usr/local/etc/ocserv-agent/config.yaml.example   # Config template
/usr/local/etc/ocserv-agent/certs/                # Certificates directory
/usr/local/etc/rc.d/ocserv_agent                  # RC script
/usr/local/share/doc/ocserv-agent/README.md       # Documentation
/usr/local/share/doc/ocserv-agent/LICENSE         # License
```

Note: FreeBSD uses `/usr/local/` prefix (BSD standard), not `/usr/`

#### RC Script
```bash
#!/bin/sh

# PROVIDE: ocserv_agent
# REQUIRE: NETWORKING
# KEYWORD: shutdown

. /etc/rc.subr

name="ocserv_agent"
rcvar="ocserv_agent_enable"

command="/usr/local/sbin/ocserv-agent"
pidfile="/var/run/${name}.pid"

load_rc_config $name
: ${ocserv_agent_enable:="NO"}
: ${ocserv_agent_config:="/usr/local/etc/ocserv-agent/config.yaml"}

command_args="-config ${ocserv_agent_config}"

run_rc_command "$1"
```

#### Package Manifest
```
name: ocserv-agent
version: <version>
origin: net/ocserv-agent
comment: "gRPC management agent for OpenConnect VPN server"
desc: "ocserv-agent provides remote management for ocserv via gRPC with mTLS authentication"
www: https://github.com/dantte-lp/ocserv-agent
maintainer: noreply@github.com
prefix: /usr/local
arch: freebsd:14:amd64
licenses: [MIT]
```

## Build Process

### Manual Build (RPM)

```bash
# On Oracle Linux runner
cd /opt/projects/repositories/ocserv-agent

# Get version
VERSION="0.7.0"

# Generate protobuf
protoc --go_out=. --go-grpc_out=. \
  --go_opt=paths=source_relative \
  --go-grpc_opt=paths=source_relative \
  pkg/proto/agent/v1/agent.proto

# Build SRPM
rpmbuild -bs ~/rpmbuild/SPECS/ocserv-agent.spec

# Build RPM with mock
mock -r el9-x86_64 \
  --resultdir=./rpm-output \
  ~/rpmbuild/SRPMS/ocserv-agent-*.src.rpm
```

### Manual Build (DEB)

```bash
# On Debian runner
cd /opt/projects/repositories/ocserv-agent

# Build binary
CGO_ENABLED=0 go build -trimpath \
  -ldflags="-s -w -X main.version=0.7.0" \
  -o ocserv-agent ./cmd/agent

# Create package structure
mkdir -p debian-build/{DEBIAN,usr/sbin,lib/systemd/system,etc/ocserv-agent/certs}

# Copy files
cp ocserv-agent debian-build/usr/sbin/
cp deploy/systemd/ocserv-agent.service debian-build/lib/systemd/system/
cp config.yaml.example debian-build/etc/ocserv-agent/

# Create control file
cat > debian-build/DEBIAN/control <<EOF
Package: ocserv-agent
Version: 0.7.0
Section: net
Priority: optional
Architecture: amd64
Depends: ocserv, adduser
Maintainer: ocserv-agent developers <noreply@github.com>
Description: gRPC management agent for OpenConnect VPN server
EOF

# Build package
dpkg-deb --build debian-build ocserv-agent_0.7.0_amd64.deb
```

### Manual Build (FreeBSD)

```bash
# Cross-compile
GOOS=freebsd GOARCH=amd64 CGO_ENABLED=0 go build -trimpath \
  -ldflags="-s -w -X main.version=0.7.0" \
  -o ocserv-agent-freebsd-amd64 ./cmd/agent

GOOS=freebsd GOARCH=arm64 CGO_ENABLED=0 go build -trimpath \
  -ldflags="-s -w -X main.version=0.7.0" \
  -o ocserv-agent-freebsd-arm64 ./cmd/agent

# Create tarball
tar -czf ocserv-agent-0.7.0-freebsd-amd64.tar.gz \
  ocserv-agent-freebsd-amd64 config.yaml.example README.md LICENSE
```

### Automated Build (GitHub Actions)

Packages are built automatically on Git tags:

```bash
# Create and push tag
git tag v0.7.0
git push origin v0.7.0

# GitHub Actions will:
# 1. Build RPMs for EL8, EL9, EL10
# 2. Build DEBs for Debian 12/13, Ubuntu 24.04
# 3. Build FreeBSD tarballs for amd64/arm64
# 4. Create GitHub Release with all packages
# 5. Generate SHA256 checksums
```

Workflow: `.github/workflows/package.yml`

## Installation

### RPM Installation

```bash
# RHEL/Oracle Linux/Rocky Linux
sudo dnf install ocserv-agent-0.7.0-1.el9.x86_64.rpm

# Or from URL
sudo dnf install https://github.com/dantte-lp/ocserv-agent/releases/download/v0.7.0/ocserv-agent-0.7.0-1.el9.x86_64.rpm
```

**First-time setup:**
```bash
# Configure
sudo cp /etc/ocserv-agent/config.yaml.example /etc/ocserv-agent/config.yaml
sudo vi /etc/ocserv-agent/config.yaml

# Setup certificates
sudo ocserv-agent generate-certs --output /etc/ocserv-agent/certs

# Enable and start
sudo systemctl enable ocserv-agent
sudo systemctl start ocserv-agent

# Check status
sudo systemctl status ocserv-agent
```

### DEB Installation

```bash
# Debian/Ubuntu
sudo apt install ./ocserv-agent_0.7.0_amd64.deb

# Or from URL
wget https://github.com/dantte-lp/ocserv-agent/releases/download/v0.7.0/ocserv-agent_0.7.0_amd64.deb
sudo dpkg -i ocserv-agent_0.7.0_amd64.deb
sudo apt-get install -f  # Install dependencies if missing
```

**First-time setup:** Same as RPM

### FreeBSD Installation

```bash
# Download and extract
fetch https://github.com/dantte-lp/ocserv-agent/releases/download/v0.7.0/ocserv-agent-0.7.0-freebsd-amd64.tar.gz
tar -xzf ocserv-agent-0.7.0-freebsd-amd64.tar.gz

# Install
sudo cp ocserv-agent-freebsd-amd64 /usr/local/sbin/ocserv-agent
sudo chmod +x /usr/local/sbin/ocserv-agent

# Install RC script (from pkg)
sudo cp ocserv_agent /usr/local/etc/rc.d/
sudo chmod +x /usr/local/etc/rc.d/ocserv_agent

# Configure
sudo mkdir -p /usr/local/etc/ocserv-agent/certs
sudo cp config.yaml.example /usr/local/etc/ocserv-agent/config.yaml
sudo vi /usr/local/etc/ocserv-agent/config.yaml

# Enable service
sudo sysrc ocserv_agent_enable="YES"
sudo service ocserv_agent start
```

## Upgrading

### RPM Upgrade
```bash
sudo dnf upgrade ocserv-agent-0.7.0-1.el9.x86_64.rpm

# Config files with %config(noreplace) are preserved
# Service is restarted automatically
```

### DEB Upgrade
```bash
sudo apt install ./ocserv-agent_0.7.0_amd64.deb

# Or
sudo dpkg -i ocserv-agent_0.7.0_amd64.deb
```

### FreeBSD Upgrade
```bash
# Stop service
sudo service ocserv_agent stop

# Replace binary
sudo cp ocserv-agent-freebsd-amd64 /usr/local/sbin/ocserv-agent

# Start service
sudo service ocserv_agent start
```

## Uninstallation

### RPM Uninstall
```bash
# Remove package (keeps config)
sudo dnf remove ocserv-agent

# Purge package (removes everything)
sudo dnf remove ocserv-agent
sudo rm -rf /etc/ocserv-agent
sudo rm -rf /var/log/ocserv-agent
sudo rm -rf /var/backups/ocserv-agent
```

### DEB Uninstall
```bash
# Remove package (keeps config)
sudo apt remove ocserv-agent

# Purge package (removes config and user)
sudo apt purge ocserv-agent
```

### FreeBSD Uninstall
```bash
# Stop and disable service
sudo service ocserv_agent stop
sudo sysrc ocserv_agent_enable="NO"

# Remove files
sudo rm /usr/local/sbin/ocserv-agent
sudo rm /usr/local/etc/rc.d/ocserv_agent
sudo rm -rf /usr/local/etc/ocserv-agent
```

## Troubleshooting

### SELinux Issues (RHEL/Oracle Linux)

**Check SELinux status:**
```bash
getenforce  # Should show "Enforcing"
```

**View denials:**
```bash
sudo ausearch -m avc -ts recent
```

**Check file contexts:**
```bash
ls -Z /usr/sbin/ocserv-agent
ls -Z /etc/ocserv-agent
```

**Restore contexts:**
```bash
sudo restorecon -Rv /usr/sbin/ocserv-agent
sudo restorecon -Rv /etc/ocserv-agent
```

**Check booleans:**
```bash
getsebool nis_enabled  # Should be "on"
```

### Permission Issues

**Check ownership:**
```bash
ls -la /etc/ocserv-agent
ls -la /var/log/ocserv-agent
```

**Fix ownership:**
```bash
sudo chown -R ocserv-agent:ocserv-agent /etc/ocserv-agent
sudo chown -R ocserv-agent:ocserv-agent /var/log/ocserv-agent
```

### Service Issues

**Check service status:**
```bash
sudo systemctl status ocserv-agent
```

**View logs:**
```bash
sudo journalctl -u ocserv-agent -f
```

**Test binary:**
```bash
/usr/sbin/ocserv-agent --version
/usr/sbin/ocserv-agent --help
```

## Package Verification

### RPM Verification

**Verify package integrity:**
```bash
rpm -V ocserv-agent
```

**List package files:**
```bash
rpm -ql ocserv-agent
```

**Show package info:**
```bash
rpm -qi ocserv-agent
```

### DEB Verification

**List package files:**
```bash
dpkg -L ocserv-agent
```

**Show package info:**
```bash
dpkg -s ocserv-agent
```

**Verify package:**
```bash
debsums ocserv-agent
```

## Version Compatibility

| Package Version | Go Version | OS Compatibility |
|----------------|------------|------------------|
| 0.7.0+ | 1.25+ | RHEL/OL 8/9/10, Debian 12/13, Ubuntu 24.04, FreeBSD 14 |
| 0.6.0 | 1.23+ | RHEL/OL 8/9, Debian 12, Ubuntu 22.04, FreeBSD 13 |
| 0.5.0 | 1.22+ | RHEL/OL 8/9, Debian 11/12, Ubuntu 20.04/22.04 |

## References

- [RPM Packaging Guide](https://rpm-packaging-guide.github.io/)
- [Debian Policy Manual](https://www.debian.org/doc/debian-policy/)
- [FreeBSD Porter's Handbook](https://docs.freebsd.org/en/books/porters-handbook/)
- [FHS (Filesystem Hierarchy Standard)](https://refspecs.linuxfoundation.org/FHS_3.0/fhs/index.html)
- [systemd Service Units](https://www.freedesktop.org/software/systemd/man/systemd.service.html)
- [SELinux User's and Administrator's Guide](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/9/html/using_selinux/index)
