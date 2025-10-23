# ocserv-agent

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/dantte-lp/ocserv-agent?include_prereleases)](https://github.com/dantte-lp/ocserv-agent/releases)

[![CI](https://github.com/dantte-lp/ocserv-agent/actions/workflows/ci.yml/badge.svg)](https://github.com/dantte-lp/ocserv-agent/actions/workflows/ci.yml)
[![Lint](https://github.com/dantte-lp/ocserv-agent/actions/workflows/lint.yml/badge.svg)](https://github.com/dantte-lp/ocserv-agent/actions/workflows/lint.yml)
[![Security](https://github.com/dantte-lp/ocserv-agent/actions/workflows/security.yml/badge.svg)](https://github.com/dantte-lp/ocserv-agent/actions/workflows/security.yml)
[![OSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/dantte-lp/ocserv-agent/badge)](https://securityscorecards.dev/viewer/?uri=github.com/dantte-lp/ocserv-agent)

**ocserv-agent** - A lightweight Go agent for remote management of OpenConnect VPN servers (ocserv) via gRPC with mTLS authentication.

## 📋 Overview

ocserv-agent is a production-ready agent that runs on each ocserv instance and provides secure remote management capabilities through a gRPC API. It enables centralized control of distributed VPN infrastructure.

### Architecture

```
Control Server (ocserv-web-panel)
    ↓ gRPC + mTLS
Agent (this project)
    ↓ exec/shell
ocserv daemon
```

### Key Features

- **🔐 Secure Communication**: mTLS authentication, TLS 1.3 minimum
- **📊 Real-time Monitoring**: Heartbeat, metrics streaming, log streaming
- **⚙️ Configuration Management**: Remote config updates with backup/rollback
- **👥 User Management**: ocpasswd wrapper, user lifecycle management
- **🔄 High Availability**: Exponential backoff, circuit breaker, graceful degradation
- **📈 Observability**: OpenTelemetry traces, Prometheus metrics, structured logging
- **🐳 Container-First**: Podman Compose based development and testing

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

#### 🚀 Quick Local Check (Before Commit)

```bash
# Fast checks in 2-3 seconds
./scripts/quick-check.sh
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

- [GitHub Actions Workflows](.github/WORKFLOWS.md) - CI/CD pipeline documentation
- [TODO Management](docs/todo/CURRENT.md) - Current tasks and progress
- [Release Notes](docs/releases/v0.1.0.md) - Version history and changes
- [Configuration Guide](config.yaml.example) - All configuration options

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

- **mTLS**: Client certificate authentication required
- **Command Whitelist**: Only approved commands (occtl, systemctl)
- **Input Validation**: Protection against command injection
- **Audit Logging**: All commands logged with context
- **Capability-Based**: Runs with minimal privileges (CAP_NET_ADMIN only)

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
curl -L https://github.com/dantte-lp/ocserv-agent/releases/download/v0.1.0/ocserv-agent-linux-amd64 -o ocserv-agent
chmod +x ocserv-agent
sudo mv ocserv-agent /usr/local/bin/

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

## 📬 Contact

- GitHub: [@dantte-lp](https://github.com/dantte-lp)
- Issues: [GitHub Issues](https://github.com/dantte-lp/ocserv-agent/issues)

## 🗺️ Roadmap

### Phase 1: Core (Week 1) - In Progress
- [x] Project structure and setup
- [x] Proto definitions
- [x] Podman Compose configuration
- [ ] Config loading
- [ ] gRPC server with mTLS
- [ ] Basic health check

### Phase 2: ocserv Integration (Week 2)
- [ ] systemctl wrapper
- [ ] occtl command execution
- [ ] Config file management
- [ ] Command validation

### Phase 3: Streaming (Week 3)
- [ ] Bidirectional streaming
- [ ] Heartbeat implementation
- [ ] Log streaming
- [ ] Reconnection logic

### Phase 4: Production Ready (Week 4)
- [ ] OpenTelemetry integration
- [ ] Error handling & retry
- [ ] Unit tests (>80% coverage)
- [ ] Integration tests
- [ ] Documentation

See [TODO](docs/todo/CURRENT.md) for detailed task list.
