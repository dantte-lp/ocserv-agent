# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**ocserv-agent** is a production-grade Go agent for remote management of OpenConnect VPN servers (ocserv) via gRPC with mTLS authentication. This is part of a distributed VPN infrastructure management system.

**Architecture:**
```
Control Server (ocserv-web-panel)
    ↓ gRPC + mTLS
Agent (this project)
    ↓ exec/shell
ocserv daemon
```

**Current Status:** BETA v0.6.0 - Production-tested deployment with 75-80% test coverage, all integration tests complete.

## Technology Stack

- **Go:** 1.25.0+ (toolchain 1.25.3)
- **gRPC:** google.golang.org/grpc v1.69.4
- **Protocol Buffers:** google.golang.org/protobuf v1.36.4
- **Logging:** github.com/rs/zerolog v1.33.0
- **Config:** gopkg.in/yaml.v3 v3.0.1
- **Container:** Podman (rootless, RHEL/OracleLinux best practice)
- **CI/CD:** GitHub Actions with self-hosted runners
- **Deployment:** Ansible automation

## Common Development Commands

### Building & Testing

**CRITICAL:** Always use Podman Compose - NEVER run `go build` or `go test` directly on the host!

```bash
# Quick pre-commit check (auto-formats code!)
./scripts/quick-check.sh

# Full build pipeline (recommended before push)
make build-all                # Run security + tests + build (all platforms)
make build-all-security       # Security scans only
make build-all-test           # Tests only
make build-all-build          # Multi-platform build only

# Development with hot reload
make compose-dev

# Run tests in containers
make compose-test

# Build multi-arch binaries (linux amd64/arm64, freebsd amd64/arm64)
make compose-build

# Security scanning
make security-check           # All scans
make security-gosec           # Gosec only
make security-govulncheck     # govulncheck only
make security-trivy           # Trivy only

# Stop all services
make compose-down

# Clean volumes
make compose-clean
```

### Testing Infrastructure

```bash
# Start mock ocserv socket server
make compose-mock-ocserv

# Ansible deployment environment
make compose-ansible
make ansible-shell           # Enter Ansible container

# View logs
make compose-logs

# Run specific test package
cd internal/grpc && go test -v
cd internal/ocserv && go test -v -run TestOcctl
```

### Git Hooks (Recommended)

```bash
# One-time setup - auto-formats code on commit
./scripts/install-hooks.sh

# Installed hooks:
# - pre-commit: Auto-format with gofmt
# - pre-push: Run quick-check.sh

# Skip hooks temporarily if needed:
git commit --no-verify
git push --no-verify
```

### Protocol Buffers

```bash
# Generate proto files (if modified pkg/proto/agent/v1/agent.proto)
# Note: This is included in compose-build, but for emergencies:
make local-proto
```

### Deployment

```bash
# Production deployment to remote server
cd deploy/ansible
ansible-playbook -i inventory/production deploy.yml

# See deploy/ansible/README.md for details
```

## Code Architecture

### High-Level Structure

```
cmd/agent/          - Main entrypoint, CLI commands
internal/
  ├── config/       - YAML config loading & validation
  ├── grpc/         - gRPC server, handlers, interceptors
  ├── ocserv/       - ocserv/occtl management, systemctl wrapper
  ├── cert/         - TLS certificate generation (bootstrap mode)
  ├── health/       - Health check implementation
  ├── metrics/      - Metrics collection
  └── telemetry/    - OpenTelemetry integration
pkg/proto/agent/v1/ - Protocol Buffers definitions
test/
  ├── fixtures/     - Test data (ocserv configs, occtl outputs)
  ├── mock-ocserv/  - Mock ocserv socket server (900+ lines)
  └── mock-server/  - Mock control server
deploy/
  ├── compose/      - Podman Compose configs (dev, test, build, security)
  ├── ansible/      - Automated deployment with backup/rollback
  └── systemd/      - systemd service unit
```

### Key Architectural Patterns

**1. Security-First Design**
- All commands validated against whitelist (`occtl`, `systemctl` only)
- Input validation prevents command injection (backticks, shell metacharacters, path traversal, control chars)
- mTLS authentication required (TLS 1.3 minimum)
- Certificate auto-generation for bootstrap mode (testing)
- 100% test coverage for `validateArguments()` with 29 injection test cases

**2. gRPC API Design**
- 5 RPC methods: AgentStream (bidirectional), ExecuteCommand, UpdateConfig, StreamLogs, HealthCheck
- Context propagation for cancellation and timeouts
- Structured logging with request_id correlation
- Graceful shutdown with timeout

**3. ocserv Integration**
- Executes `occtl` commands via shell (with validation)
- Parses JSON output from occtl (13/16 commands working, 3 upstream bugs)
- Wraps `systemctl` for service control
- Reads config files (main, per-user, per-group)

**4. Testing Strategy**
- Unit tests: 75-80% coverage (internal packages)
- Integration tests: 119 tests (82 occtl + 11 systemctl + 26 gRPC)
- Mock ocserv server with 14 production fixtures
- TLS certificate helpers for testing
- Table-driven tests throughout

**5. Error Handling**
- All errors wrapped with context: `fmt.Errorf("operation failed: %w", err)`
- Structured logging with zerolog: `log.Error().Err(err).Str("key", val).Msg("message")`
- gRPC status codes: `status.Error(codes.InvalidArgument, "message")`
- Panic recovery in gRPC interceptors

### Critical Implementation Details

**Command Execution Flow:**
```go
1. Client calls ExecuteCommand RPC
2. Handler validates command type against whitelist
3. validateArguments() checks each argument for injection
4. exec.CommandContext() with timeout from config
5. Capture stdout/stderr, log execution
6. Return CommandResponse with exit code
```

**Security Validation (internal/ocserv/manager.go):**
```go
// validateArguments checks for command injection patterns
// 100% test coverage - do NOT modify without tests!
func validateArguments(args []string) error {
    for _, arg := range args {
        // Check for shell metacharacters, backticks, newlines, etc.
        if containsDangerousChars(arg) {
            return ErrInvalidArgument
        }
    }
    return nil
}
```

**TLS Certificate Bootstrap (internal/cert/):**
- Auto-generates self-signed CA + agent cert on first run if `tls.auto_generate: true`
- Production mode: Use existing certs or run `ocserv-agent gencert`
- See docs/CERTIFICATES.md for details

**Configuration Hierarchy:**
1. Default values (hardcoded)
2. YAML config file (`/etc/ocserv-agent/config.yaml`)
3. Environment variables (override)
4. Command-line flags (highest priority)

### Important Files to Know

**Core Implementation:**
- `cmd/agent/main.go` - Entrypoint, graceful shutdown, signal handling
- `internal/grpc/server.go` - gRPC server setup with mTLS
- `internal/grpc/handlers.go` - RPC method implementations
- `internal/ocserv/manager.go` - occtl command execution (SECURITY CRITICAL)
- `internal/ocserv/systemctl.go` - systemctl wrapper
- `internal/ocserv/config.go` - Config file parser
- `internal/config/config.go` - YAML config loading
- `internal/cert/generator.go` - Certificate generation

**Testing Infrastructure:**
- `internal/ocserv/testutil/mock.go` - Mock ocserv for tests
- `internal/ocserv/testutil/fixtures.go` - Test data loader
- `test/fixtures/ocserv/occtl/*.json` - Real occtl output samples
- `test/mock-ocserv/server.go` - Unix socket server emulating ocserv

**Configuration:**
- `config.yaml.example` - Full config reference with comments
- `pkg/proto/agent/v1/agent.proto` - gRPC API specification

### Known Issues & Upstream Bugs

**ocserv Compatibility (13/16 working):**
- ✅ Working: `show users`, `show status`, `show stats`, `disconnect user/id`, `reload`, etc.
- ❌ Broken (upstream bugs):
  - `show iroutes` - Invalid JSON output (we contributed fix to upstream: issue #661)
  - `show sessions all/valid` - Trailing commas regression (reported: issue #669)

**Security:**
- 4 command injection vulnerabilities fixed in v0.5.0 (backtick, escaped chars, newlines, control chars)
- OSSF Scorecard: 5.9/10 (roadmap to 7.5+/10 in future releases)

## Development Workflow

### Standard Workflow (Follow This!)

```bash
# 1. Create feature branch
git checkout -b feature/my-feature

# 2. Make changes
vim internal/grpc/handlers.go

# 3. Run quick check (auto-formats!)
./scripts/quick-check.sh

# 4. If tests pass, commit
git add internal/grpc/handlers.go
git commit -m "feat(grpc): add new RPC method"

# 5. Before push, run full pipeline
make build-all

# 6. Push and create PR
git push origin feature/my-feature
gh pr create --base main --fill
```

### Commit Message Format (Conventional Commits)

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation only
- `test`: Adding tests
- `refactor`: Code refactoring
- `chore`: Dependency updates, build tasks
- `security`: Security fixes (use this for CVEs!)

**Examples:**
```bash
feat(grpc): implement bidirectional streaming
fix(ocserv): handle missing config file gracefully
docs(readme): add installation instructions
test(manager): add unit tests for RunCommand
security(validation): fix command injection in validateArguments
```

### Code Style & Standards

**Go Standards:**
- Follow [Effective Go](https://go.dev/doc/effective_go)
- Use `gofmt -s` and `goimports` (automatic via git hooks)
- golangci-lint must pass (runs in CI)
- All exported functions must have godoc comments
- Table-driven tests for all new code

**Logging:**
```go
// Use structured logging with zerolog
log.Info().
    Str("command", cmdType).
    Strs("args", args).
    Int("exit_code", exitCode).
    Msg("Command executed")

log.Error().
    Err(err).
    Str("request_id", reqID).
    Msg("Failed to execute command")
```

**Error Handling:**
```go
// Wrap errors with context
if err != nil {
    return fmt.Errorf("failed to connect to ocserv: %w", err)
}

// gRPC status codes
return status.Error(codes.InvalidArgument, "invalid command type")
```

**Context Propagation:**
```go
// Always pass context through call chain
func (s *Server) ExecuteCommand(ctx context.Context, req *pb.CommandRequest) (*pb.CommandResponse, error) {
    // Add tracing span
    ctx, span := s.tracer.Start(ctx, "ExecuteCommand")
    defer span.End()

    // Check cancellation
    if err := ctx.Err(); err != nil {
        return nil, status.Error(codes.Canceled, "context canceled")
    }

    // Pass context to next layer
    return s.manager.RunCommand(ctx, req.CommandType, req.Args)
}
```

### Testing Requirements

**Before Committing:**
- Run `./scripts/quick-check.sh` (fast, auto-formats)
- All tests must pass
- No new golangci-lint warnings

**Before Pushing:**
- Run `make build-all` (full pipeline)
- Security scans must pass
- Coverage should not decrease

**When Adding New Features:**
- Write unit tests (aim for >80% coverage)
- Add integration tests if touching gRPC/ocserv
- Update documentation (README.md, GRPC_TESTING.md, etc.)
- Add examples to config.yaml.example if new config options

**Test File Organization:**
```
internal/grpc/
  ├── handlers.go              # Implementation
  ├── handlers_test.go         # Unit tests
  └── integration_test.go      # Integration tests
```

### Security Considerations

**CRITICAL - Command Injection Prevention:**
- NEVER modify `validateArguments()` without adding tests
- ALL user input MUST be validated
- Use whitelist approach (only `occtl`, `systemctl`)
- No shell metacharacters: `;`, `|`, `&`, `$`, backticks, newlines, etc.

**TLS/mTLS:**
- TLS 1.3 minimum
- Client certificate verification required in production
- Bootstrap mode (auto-generate) for testing only
- See docs/CERTIFICATES.md for production setup

**Secrets Management:**
- NO hardcoded credentials in code
- NO secrets in git history
- Use environment variables or secure config
- Sanitize logs (mask sensitive data)

**Dependency Management:**
- Run `govulncheck` before releases
- Keep dependencies updated
- Review security advisories

## Documentation

### User-Facing Docs
- `README.md` - Project overview, quick start, features
- `ROADMAP.md` - Development roadmap (strategic)
- `docs/CERTIFICATES.md` - TLS/mTLS setup guide
- `docs/GRPC_TESTING.md` - Test gRPC API with grpcurl
- `docs/LOCAL_TESTING.md` - Development and CI testing
- `config.yaml.example` - Full configuration reference

### Developer Docs
- `CLAUDE_PROMPT.md` - Detailed Russian-language development guide (original spec)
- `docs/todo/CURRENT.md` - Current tasks (tactical, regularly updated)
- `docs/releases/` - Release notes for each version
- `.github/CONTRIBUTING.md` - Contribution guidelines
- `.github/WORKFLOWS.md` - CI/CD pipeline documentation

### API Documentation
- `pkg/proto/agent/v1/agent.proto` - gRPC API specification
- Generated docs at runtime via gRPC reflection

## Deployment

### Production Deployment

**Recommended: Ansible (Automated)**
```bash
cd deploy/ansible
cp inventory/production.example inventory/production
# Edit inventory with your servers
ansible-playbook -i inventory/production deploy.yml
```

**Features:**
- Zero-downtime deployment
- Automatic backup before update
- Rollback on failure
- Health check validation
- Systemd service management

See `deploy/ansible/README.md` for details.

### Configuration Files

**Production Config:** `/etc/ocserv-agent/config.yaml`

**Key Settings:**
```yaml
agent_id: "unique-server-id"

control_server:
  address: "control.example.com:9090"

tls:
  enabled: true
  auto_generate: false  # Use real certs in production!
  cert_file: "/etc/ocserv-agent/certs/agent.crt"
  key_file: "/etc/ocserv-agent/certs/agent.key"
  ca_file: "/etc/ocserv-agent/certs/ca.crt"

ocserv:
  config_path: "/etc/ocserv/ocserv.conf"
  ctl_socket: "/var/run/occtl.socket"
  systemd_service: "ocserv"

security:
  allowed_commands: ["occtl", "systemctl"]
  max_command_timeout: 300s

logging:
  level: "info"
  format: "json"
```

### Monitoring

**Health Checks (3-Tier):**
1. Tier 1 (every 10-15s): Basic status, CPU, RAM, active sessions
2. Tier 2 (every 1-2m): Process status, port listening, config valid
3. Tier 3 (on-demand): End-to-end VPN connection test

**Logs:**
```bash
# systemd journal
sudo journalctl -u ocserv-agent -f

# Log files (if configured)
sudo tail -f /var/log/ocserv-agent/agent.log
```

**Metrics:**
- System metrics: CPU, memory, load
- ocserv metrics: active sessions, bandwidth
- OpenTelemetry traces (if enabled)

## CI/CD

### GitHub Actions Workflows

**On Push/PR:**
- Test (Go 1.25)
- Code Quality Checks
- golangci-lint
- Security Scan (gosec, govulncheck, CodeQL)

**On Release Tag:**
- Build multi-arch binaries
- Create GitHub release
- Publish container images

**Path Filtering:**
- Docs-only changes skip expensive jobs
- Optimized for fast feedback

### Self-Hosted Runners

**Moved to separate repo:** https://github.com/dantte-lp/self-hosted-runners

Uses systemd quadlets (RHEL 9+ best practice) instead of compose.

## Common Pitfalls & Solutions

### "Permission denied" when running agent
```bash
# Agent needs access to /var/run/occtl.socket
sudo usermod -aG ocserv ocserv-agent
# Or run as root (not recommended)
```

### "Command not allowed" errors
```bash
# Check security.allowed_commands in config.yaml
# Only occtl and systemctl are allowed by default
```

### Test failures with "socket not found"
```bash
# Start mock ocserv first
make compose-mock-ocserv
# Then run tests
make compose-test
```

### gRPC connection refused
```bash
# Check if agent is running
sudo systemctl status ocserv-agent

# Check TLS certificates
ls -la /etc/ocserv-agent/certs/

# Enable auto-generate for testing
# Set tls.auto_generate: true in config
```

### Podman permission issues (SELinux)
```bash
# Relabel project directory
sudo chcon -R -t container_file_t /opt/projects/repositories/ocserv-agent

# Or use :z flag in volume mounts (already configured in compose files)
```

## Project Conventions

### Version Numbering (Semantic Versioning)
- MAJOR.MINOR.PATCH (e.g., 0.6.0)
- v0.x.x = Beta (breaking changes allowed)
- v1.0.0 = Stable API contract
- Pre-release: v0.6.0-beta.1

### Release Process
1. Update version in `cmd/agent/main.go`
2. Update CHANGELOG.md
3. Create release notes in `docs/releases/vX.Y.Z.md`
4. Tag: `git tag -a v0.6.0 -m "Release v0.6.0"`
5. Push: `git push origin v0.6.0`
6. GitHub Actions builds and publishes

### Branch Strategy
- `main` - Stable, production-ready
- `develop` - Integration branch (if using)
- `feature/*` - Feature branches
- `fix/*` - Bug fixes
- `security/*` - Security patches (priority merge)

### PR Requirements
- At least 1 approving review
- All CI checks must pass
- No merge conflicts
- Commit messages follow Conventional Commits

## Quick Reference Card

```bash
# Development
make compose-dev              # Start dev env with hot reload
./scripts/quick-check.sh      # Fast pre-commit check
make build-all                # Full pipeline (before push)

# Testing
make compose-test             # All tests in containers
make security-check           # Security scans
make compose-mock-ocserv      # Start mock ocserv

# Building
make compose-build            # Multi-arch binaries
make local-proto              # Regen proto (emergency only)

# Deployment
make compose-ansible          # Ansible environment
ansible-playbook deploy.yml   # Deploy to production

# Cleanup
make compose-down             # Stop all services
make compose-clean            # Clean volumes
make clean                    # Clean build artifacts

# Git hooks
./scripts/install-hooks.sh    # One-time setup
git commit --no-verify        # Skip hooks if needed
```

## Contact & Resources

- **Repository:** https://github.com/dantte-lp/ocserv-agent
- **Issues:** https://github.com/dantte-lp/ocserv-agent/issues
- **Security:** See SECURITY.md for vulnerability disclosure
- **Documentation:** All docs in `docs/` directory
- **Upstream:** https://ocserv.gitlab.io/www/index.html (OpenConnect VPN)
