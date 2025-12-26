# QA Testing Infrastructure

## Quick Start

### Run All QA Checks

```bash
# 1. Start QA container
podman-compose -f compose.qa.yaml up -d

# 2. Run automated QA report
python3 scripts/qa_report.py --container ocserv-agent-qa

# 3. View report
cat docs/tmp/reports/$(date +%Y-%m-%d)_go-qa-report.md
```

### Individual Checks

```bash
# Vulnerability check (matches CI)
podman exec ocserv-agent-qa govulncheck ./...

# All tests with race detector
podman exec ocserv-agent-qa go test -v -race -coverprofile=coverage.out ./...

# Linting
podman exec ocserv-agent-qa golangci-lint run ./...

# Static analysis
podman exec ocserv-agent-qa staticcheck ./...

# Security scan
podman exec ocserv-agent-qa gosec ./...

# Format check
podman exec ocserv-agent-qa gofmt -s -l .
```

## Infrastructure

### Files

- `deploy/Containerfile.dev-go` - Docker image with Go 1.25 + testing tools
- `compose.qa.yaml` - Podman Compose configuration
- `scripts/qa_report.py` - Automated QA runner
- `docs/tmp/reports/` - Generated markdown reports

### Installed Tools

| Tool | Version | Purpose |
|------|---------|---------|
| Go | 1.25 | Compiler |
| golangci-lint | v2.7.2 | Linting |
| govulncheck | latest | Vulnerability scanning |
| gosec | latest | Security scanning |
| staticcheck | latest | Static analysis |
| protoc | system | Protocol Buffers compiler |

### Container Details

- Name: `ocserv-agent-qa`
- Base: `golang:1.25-trixie`
- Volumes: Source code, go mod cache, build cache
- Command: Keeps running for interactive testing

## CI/CD Integration

All local checks match GitHub Actions workflows:

- `.github/workflows/security.yml` → `govulncheck`
- `.github/workflows/lint.yml` → `golangci-lint`
- `.github/workflows/ci.yml` → `go test -race`

## Cleanup

```bash
# Stop container
podman-compose -f compose.qa.yaml down

# Remove images and volumes
podman-compose -f compose.qa.yaml down --rmi all -v
```

## Troubleshooting

### Container not running

```bash
podman-compose -f compose.qa.yaml up -d
podman ps --filter name=ocserv-agent-qa
```

### protobuf errors

```bash
podman exec ocserv-agent-qa protoc --go_out=. --go-grpc_out=. \
  --go_opt=paths=source_relative \
  --go-grpc_opt=paths=source_relative \
  pkg/proto/agent/v1/agent.proto
```

### Coverage report

```bash
podman exec ocserv-agent-qa go test -coverprofile=coverage.out ./...
podman exec ocserv-agent-qa go tool cover -html=coverage.out -o /tmp/coverage.html
podman cp ocserv-agent-qa:/tmp/coverage.html .
```

---

**Maintained by:** Claude Code Agent
**Last updated:** 2025-12-26
