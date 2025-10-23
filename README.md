# ocserv-agent

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/dantte-lp/ocserv-agent?include_prereleases)](https://github.com/dantte-lp/ocserv-agent/releases)
[![Status](https://img.shields.io/badge/status-BETA-yellow.svg)](https://github.com/dantte-lp/ocserv-agent/releases/tag/v0.5.0)
[![Test Coverage](https://img.shields.io/badge/coverage-51.2%25%20(all_internal)-green)](https://github.com/dantte-lp/ocserv-agent/blob/main/docs/releases/v0.5.0.md)
[![grpc Coverage](https://img.shields.io/badge/grpc-87.6%25-brightgreen)](https://github.com/dantte-lp/ocserv-agent/blob/main/docs/releases/v0.5.0.md)

[![CI](https://github.com/dantte-lp/ocserv-agent/actions/workflows/ci.yml/badge.svg)](https://github.com/dantte-lp/ocserv-agent/actions/workflows/ci.yml)
[![Lint](https://github.com/dantte-lp/ocserv-agent/actions/workflows/lint.yml/badge.svg)](https://github.com/dantte-lp/ocserv-agent/actions/workflows/lint.yml)
[![Security](https://github.com/dantte-lp/ocserv-agent/actions/workflows/security.yml/badge.svg)](https://github.com/dantte-lp/ocserv-agent/actions/workflows/security.yml)
[![OSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/dantte-lp/ocserv-agent/badge)](https://securityscorecards.dev/viewer/?uri=github.com/dantte-lp/ocserv-agent)

**ocserv-agent** - A lightweight Go agent for remote management of OpenConnect VPN servers (ocserv) via gRPC with mTLS authentication.

> **Status:** BETA (v0.5.0) - Production-tested with real VPN users. Core features complete, test coverage expanded (51.2% overall, 87.6% grpc), critical security vulnerabilities fixed. See [ROADMAP.md](ROADMAP.md) for future plans.

## 📋 Overview

ocserv-agent is a **production-tested BETA** agent that runs on each ocserv instance and provides secure remote management capabilities through a gRPC API. It enables centralized control of distributed VPN infrastructure.

**Current Release:** [v0.5.0 BETA](https://github.com/dantte-lp/ocserv-agent/releases/tag/v0.5.0) (October 2025)
- ✅ **CRITICAL security fixes:** Fixed 4 command injection vulnerabilities (29 test cases)
- ✅ **Test coverage expansion:** internal/grpc 0% → 87.6%, overall 40% → 51.2%
- ✅ 1,600+ new lines of test code (total: 3,800+ lines)
- ✅ Test infrastructure: TLS certificate helpers, security validation tests
- ✅ Security-first testing: validateArguments 100% coverage

**Previous Release:** [v0.3.1 BETA](https://github.com/dantte-lp/ocserv-agent/releases/tag/v0.3.1) (October 2025)
- Production-tested with 3 active VPN users
- 13/16 occtl commands working (3 broken due to upstream ocserv bugs)
- Security hardening: mTLS, command validation, audit logging

### Architecture

```
Control Server (ocserv-web-panel)
    ↓ gRPC + mTLS
Agent (this project)
    ↓ exec/shell
ocserv daemon
```

### Key Features

- **🔐 Secure Communication**: mTLS authentication, TLS 1.3 minimum, client certificate verification
- **📊 ocserv Control**: Execute occtl/systemctl commands remotely (13/16 working, 3 upstream bugs)
- **⚙️ Configuration Management**: Read ocserv configs (main, per-user, per-group)
- **🔒 Security Hardening**: Command whitelist, input validation, command injection protection
- **📝 Comprehensive Docs**: 800+ lines of documentation, ROADMAP, release notes
- **🏗️ Production Ready**: Certificate auto-generation, systemd service, multi-platform builds
- **🐳 Container-First**: Podman Compose based development and testing
- **🤝 Open Source**: MIT license, OSSF security best practices, upstream contributions
- **✅ Test Coverage**: 97.1% config, 77.6% cert, 82-100% ocserv/config (2,225 lines of tests)
- **⚙️ DevOps**: Automatic formatting, git hooks, CI optimizations

### What's Working (v0.5.0)

✅ **Core Features:**
- gRPC server with mTLS authentication
- ExecuteCommand RPC (occtl/systemctl commands)
- HealthCheck RPC (Tier 1 - heartbeat)
- Config file reading (main, per-user, per-group)
- Command validation and security
- Certificate auto-generation (bootstrap mode)

✅ **occtl Commands (13/16):**
- `show users`, `show user [NAME]`, `show id [ID]`
- `show status`, `show stats`
- `show ip bans`, `show ip ban points`, `unban ip [IP]`
- `disconnect user [NAME]`, `disconnect id [ID]`
- `reload`

⚠️ **Known Issues (3/16 - upstream ocserv bugs):**
- `show iroutes` - invalid JSON (we [contributed fix](https://gitlab.com/openconnect/ocserv/-/issues/661#note_2839397707) upstream)
- `show sessions all/valid` - trailing commas (we [reported regression](https://gitlab.com/openconnect/ocserv/-/issues/669))

## 🚀 Quick Start

### Prerequisites

- Go 1.25+
- Podman and podman-compose
- protobuf-compiler (for proto generation)

### Development Setup

```bash
# Clone repository
git clone https://github.com/dantte-lp/ocserv-agent.git
cd ocserv-agent

# Setup compose environment
make setup-compose

# Start development server with hot reload
make compose-dev
```

### Running Tests

#### 🚀 Full Build Pipeline (Recommended)

```bash
# Run everything: security + tests + multi-platform build
make build-all

# Or run specific stages:
make build-all-security  # Security scans only
make build-all-test      # Tests only
make build-all-build     # Multi-platform build only
```

This runs the complete CI/CD pipeline locally:
- Security scans (gosec, govulncheck, trivy)
- Unit tests with coverage
- Linting (golangci-lint)
- Multi-platform builds (Linux/FreeBSD, amd64/arm64)

#### 🏃 Quick Local Check (Before Commit)

```bash
# Fast checks in 2-3 seconds (auto-formats code!)
./scripts/quick-check.sh
```

**Features:**
- ✅ Auto-formats Go code with `gofmt -s -w`
- ✅ Runs `go vet` for common mistakes
- ✅ Builds the project
- ✅ Runs all unit tests

#### 🪝 Git Hooks (Automatic Formatting)

Install git hooks to automatically format code before commits:

```bash
# One-time setup
./scripts/install-hooks.sh
```

**Installed hooks:**
- **pre-commit**: Auto-formats Go code with `gofmt` before each commit
- **pre-push**: Runs `quick-check.sh` before each push

**Skip hooks temporarily:**
```bash
git commit --no-verify  # Skip pre-commit hook
git push --no-verify    # Skip pre-push hook
```

#### 🔬 Full Local CI (Before Push)

```bash
# Run all CI checks locally (saves GitHub Actions minutes!)
./scripts/test-local.sh
```

See [LOCAL_TESTING.md](docs/LOCAL_TESTING.md) for details.

#### 🐳 Container Tests

```bash
# Run all tests in containers
make compose-test

# View logs
make compose-logs
```

### Building

```bash
# Build multi-arch binaries
make compose-build

# Binaries will be in bin/:
# - bin/ocserv-agent-linux-amd64
# - bin/ocserv-agent-linux-arm64
```

## 📖 Documentation

### User Guides
- **[Release Notes v0.5.0](docs/releases/v0.5.0.md)** - Latest release: Test coverage & security fixes
- **[Project Roadmap](ROADMAP.md)** - Development roadmap and timeline
- **[occtl Commands Reference](docs/OCCTL_COMMANDS.md)** - Complete command guide with examples
- **[gRPC Testing Guide](docs/GRPC_TESTING.md)** - Test API with grpcurl
- **[Certificate Management](docs/CERTIFICATES.md)** - TLS/mTLS setup (bootstrap + production)
- [Configuration Guide](config.yaml.example) - All configuration options

### Developer Guides
- [Local Testing Guide](docs/LOCAL_TESTING.md) - Development and CI testing
- [GitHub Actions Workflows](.github/WORKFLOWS.md) - CI/CD pipeline
- [Contributing Guide](.github/CONTRIBUTING.md) - Development workflow
- [TODO Management](docs/todo/CURRENT.md) - Current tasks and progress (tactical)

### Security
- **[Security Policy](SECURITY.md)** - Vulnerability disclosure process
- **[OSSF Scorecard Improvements](docs/OSSF_SCORECARD_IMPROVEMENTS.md)** - Security roadmap (4.9 → 7.5+/10)
- [ocserv Compatibility](docs/todo/OCSERV_COMPATIBILITY.md) - Feature coverage analysis

### Releases
- [v0.5.0 Release Notes](docs/releases/v0.5.0.md) - Test coverage expansion & security fixes (Oct 2025) ✅
- [v0.4.0 Release Notes](docs/releases/v0.4.0.md) - Test foundation & DevOps (Oct 2025)
- [v0.3.1 Release Notes](docs/releases/v0.3.1.md) - Critical bugfixes + documentation (Oct 2025)
- [All Releases](https://github.com/dantte-lp/ocserv-agent/releases) - Full release history
- [v0.3.0 Release Notes](docs/releases/v0.3.0.md) - Certificate auto-generation (Oct 2025)

## 🔧 Configuration

Configuration is done via YAML file:

```yaml
# /etc/ocserv-agent/config.yaml
agent_id: "server-01"

control_server:
  address: "control.example.com:9090"

tls:
  enabled: true
  cert_file: "/etc/ocserv-agent/certs/agent.crt"
  key_file: "/etc/ocserv-agent/certs/agent.key"
  ca_file: "/etc/ocserv-agent/certs/ca.crt"

ocserv:
  config_path: "/etc/ocserv/ocserv.conf"
  systemd_service: "ocserv"
```

See [config.yaml.example](config.yaml.example) for all options.

## 🛠️ Development Workflow

This project uses **Podman Compose** for all development and testing:

```bash
# Start development (hot reload)
make compose-dev

# Run tests
make compose-test

# Build binaries
make compose-build

# Stop all services
make compose-down
```

**Important**: Do NOT use `go build` or `go test` directly on the host. Always use Podman Compose targets for consistency and reproducibility.

## 🏗️ Project Structure

```
ocserv-agent/
├── cmd/
│   └── agent/          # Main entrypoint
├── internal/
│   ├── config/         # Configuration loading
│   ├── grpc/           # gRPC server
│   ├── ocserv/         # ocserv management
│   ├── health/         # Health checks
│   ├── metrics/        # Metrics collection
│   └── telemetry/      # OpenTelemetry
├── pkg/
│   └── proto/          # Protocol Buffers
├── deploy/
│   ├── compose/        # Podman Compose files
│   ├── systemd/        # systemd service
│   └── scripts/        # Deployment scripts
├── docs/
│   ├── todo/           # Task management
│   └── releases/       # Release notes
└── test/
    ├── mock-server/    # Mock control server
    └── mock-ocserv/    # Mock ocserv
```

## 🔒 Security

- **mTLS**: Client certificate authentication required (TLS 1.3 minimum)
- **Command Whitelist**: Only approved commands (occtl, systemctl)
- **Input Validation**: Protection against command injection, shell metacharacters, path traversal
- **Audit Logging**: All commands logged with context (user, timestamp, result)
- **Capability-Based**: Runs with minimal privileges (CAP_NET_ADMIN only)
- **Security Scanning**: gosec, govulncheck, trivy, CodeQL
- **OSSF Scorecard**: 5.9/10 (improving to 7.5+/10)
- **Vulnerability Disclosure**: [SECURITY.md](SECURITY.md) with 48h response time

### Recent Security Improvements (v0.3.1 - v0.5.0)

- ✅ **v0.5.0: CRITICAL command injection fixes** (backtick, escaped chars, newlines, control chars)
- ✅ **v0.5.0: Security validation 100% coverage** (29 injection test cases)
- ✅ SECURITY.md vulnerability disclosure policy created
- ✅ Removed hardcoded credentials from repository
- ✅ Sanitized all deployment scripts
- ✅ OSSF Scorecard improved from 4.9/10 to 5.9/10
- ✅ Branch protection with required PR reviews (v0.4.0)
- ✅ Admin bypass for emergency hotfixes (v0.4.0)
- 📋 Roadmap to 7.5+/10 in v0.6.0 (GPG signing, dependency pinning, token permissions)

## 📊 Monitoring

### Health Checks (3-Tier)

1. **Tier 1 - Heartbeat** (every 10-15s): Basic status, CPU, RAM, active sessions
2. **Tier 2 - Deep Check** (every 1-2m): Process status, port listening, config validation
3. **Tier 3 - Application Check** (on-demand): End-to-end VPN connection test

### Metrics

- System metrics (CPU, memory, load)
- ocserv metrics (sessions, bandwidth)
- Custom application metrics
- OpenTelemetry traces for all gRPC calls

## 🔄 API

The agent provides the following gRPC services:

- `AgentStream`: Bidirectional streaming for heartbeat and commands
- `ExecuteCommand`: Execute occtl/systemctl commands
- `UpdateConfig`: Update ocserv configuration with backup
- `StreamLogs`: Stream ocserv logs in real-time
- `HealthCheck`: Multi-tier health checks

See [agent.proto](pkg/proto/agent/v1/agent.proto) for full API specification.

## 📦 Installation

### From Binary

```bash
# Download latest release
wget https://github.com/dantte-lp/ocserv-agent/releases/download/v0.5.0/ocserv-agent-v0.5.0-linux-amd64.tar.gz
tar -xzf ocserv-agent-v0.5.0-linux-amd64.tar.gz

# Install to /etc/ocserv-agent
sudo mkdir -p /etc/ocserv-agent
sudo mv ocserv-agent /etc/ocserv-agent/
sudo chmod +x /etc/ocserv-agent/ocserv-agent

# Install systemd service
sudo cp deploy/systemd/ocserv-agent.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now ocserv-agent
```

### From Source

```bash
git clone https://github.com/dantte-lp/ocserv-agent.git
cd ocserv-agent
make compose-build
sudo make install
```

### Docker

```bash
podman pull ghcr.io/dantte-lp/ocserv-agent:latest
podman run -d \
  --name ocserv-agent \
  -v /etc/ocserv-agent:/etc/ocserv-agent:z \
  ghcr.io/dantte-lp/ocserv-agent:latest
```

## 🧪 Testing

```bash
# Unit tests
make compose-test

# Integration tests
cd test/integration
go test -v ./...

# Test with grpcurl
grpcurl -cacert certs/ca.crt \
  -cert certs/admin.crt \
  -key certs/admin.key \
  -d '{"tier": 1}' \
  localhost:9090 \
  agent.v1.AgentService/HealthCheck
```

## 🤝 Contributing

We welcome contributions! Please follow our development workflow:

1. Read the [Contributing Guide](.github/CONTRIBUTING.md) for detailed instructions
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes and ensure CI passes
4. Create a Pull Request

**Required checks before merge:**
- ✅ Test (Go 1.25)
- ✅ Code Quality Checks
- ✅ golangci-lint

See [CONTRIBUTING.md](.github/CONTRIBUTING.md) for complete workflow documentation

### Commit Message Format

```
<type>(<scope>): <subject>

<body>

<footer>
```

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [ocserv](https://ocserv.gitlab.io/www/index.html) - OpenConnect VPN server
- [gRPC](https://grpc.io/) - High-performance RPC framework
- [zerolog](https://github.com/rs/zerolog) - Zero allocation JSON logger

### Upstream Contributions

We actively contribute bug reports and fixes to the ocserv project:

- **[Issue #661](https://gitlab.com/openconnect/ocserv/-/issues/661#note_2839397707)** - Root cause analysis for `show iroutes` invalid JSON
  - Identified 3 bugs in `src/occtl/unix.c:1018-1045`
  - Provided proposed fix with code examples

- **[Issue #669](https://gitlab.com/openconnect/ocserv/-/issues/669)** - Reported regression of #220 in ocserv 1.3.0
  - Trailing commas in `show sessions` commands
  - Production-tested and documented

## 📬 Contact

- GitHub: [@dantte-lp](https://github.com/dantte-lp)
- Issues: [GitHub Issues](https://github.com/dantte-lp/ocserv-agent/issues)

## 🗺️ Roadmap

> **See [ROADMAP.md](ROADMAP.md) for detailed project roadmap and timeline.**

### ✅ v0.4.0 BETA (Completed - October 2025)

**Test Foundation & DevOps:**
- ✅ internal/config: 97.1% coverage ([PR #14](https://github.com/dantte-lp/ocserv-agent/pull/14))
- ✅ internal/cert: 77.6% coverage (certificate generation tests)
- ✅ internal/ocserv/config.go: 82-100% coverage (config parser tests)
- ✅ Test infrastructure with 8 fixture files
- ✅ 2,225 lines of test code

**DevOps Improvements:**
- ✅ Automatic code formatting (scripts/quick-check.sh)
- ✅ Git hooks (pre-commit: auto-format, pre-push: checks) ([PR #16](https://github.com/dantte-lp/ocserv-agent/pull/16))
- ✅ One-time setup: `./scripts/install-hooks.sh`
- ✅ CI path filtering (skip expensive jobs for docs-only changes)

**Security:**
- ✅ Branch protection with required reviews
- ✅ Admin bypass for hotfixes
- ✅ OSSF Scorecard: 5.9/10

### ✅ v0.5.0 BETA (Completed - October 2025)

**Test Coverage Expansion & Security Fixes:**
- ✅ **CRITICAL:** Fixed 4 command injection vulnerabilities (29 test cases)
- ✅ internal/grpc: 0% → 87.6% coverage (exceeded >80% target!)
- ✅ internal/ocserv: 15.8% → 23.1% coverage
- ✅ Overall internal: ~40% → 51.2% (+11.2%)
- ✅ 1,600+ new lines of test code
- ✅ Test infrastructure: TLS certificate helpers, security validation
- ✅ validateArguments: 100% coverage (security-first testing)

### 🔮 Next: v0.6.0 (Security Hardening & Integration Tests)

**Target:** January 2026

**Goals:**
- OSSF Scorecard: 7.5+/10 (GPG signing, dependency pinning, token permissions)
- Integration tests with mock ocserv
- Rate limiting for gRPC API
- Security scanning in CI (gosec, trivy)

### 🔜 Future (v0.6.0+)

- [ ] Bidirectional streaming (AgentStream RPC)
- [ ] Enhanced metrics (Prometheus exporter)
- [ ] Heartbeat with exponential backoff
- [ ] ocserv-fw firewall integration
- [ ] Virtual hosts support
- [ ] RADIUS/Kerberos monitoring

See [CURRENT.md](docs/todo/CURRENT.md) for current tasks and [ROADMAP.md](ROADMAP.md) for long-term project roadmap.
