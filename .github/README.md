# GitHub Actions Workflows

This directory contains automated CI/CD workflows for ocserv-agent.

## 📋 Workflows

### 🚀 Release (`release.yml`)

**Trigger:** Push tag `v*.*.*`

**Features:**
- **SLSA Level 3** provenance generation for secure builds
- Multi-architecture binary builds (amd64, arm64, armv7)
- Automated GitHub Release creation with release notes
- Container image build and push to GitHub Container Registry
- SHA256 checksums for all artifacts

**Usage:**
```bash
git tag -a v0.3.0 -m "Release v0.3.0"
git push origin v0.3.0
```

**Artifacts:**
- `ocserv-agent-linux-amd64` - Linux x86_64 binary
- `ocserv-agent-linux-arm64` - Linux ARM64 binary
- `ocserv-agent-linux-arm-v7` - Linux ARMv7 binary
- `*.sha256` - SHA256 checksums
- `*.intoto.jsonl` - SLSA provenance

**SLSA Verification:**
```bash
# Install slsa-verifier
go install github.com/slsa-framework/slsa-verifier/v2/cli/slsa-verifier@latest

# Verify binary
slsa-verifier verify-artifact \
  --provenance-path ocserv-agent-*.intoto.jsonl \
  --source-uri github.com/dantte-lp/ocserv-agent \
  ocserv-agent-linux-amd64
```

---

### 🔄 CI (`ci.yml`)

**Trigger:** Push/PR to `main` or `develop`

**Jobs:**
1. **Test** - Run tests on Go 1.24 and 1.25
   - Unit tests with race detection
   - Coverage reporting to Codecov
   - Dependency verification

2. **Build** - Build for multiple platforms
   - Linux (amd64, arm64)
   - Darwin/macOS (amd64, arm64)
   - Windows (amd64)

3. **Integration** - Integration tests (when available)

4. **Checks** - Code quality checks
   - Go formatting (`gofmt`)
   - Go vet
   - `go mod tidy` verification

**Requirements:**
- All tests must pass
- Code must be formatted
- No vet warnings

---

### 🧹 Lint (`lint.yml`)

**Trigger:** Push/PR to `main` or `develop`

**Linters:**
1. **golangci-lint** - Go code linter
   - 30+ linters enabled
   - Configuration: `.golangci.yml`
   - Security checks (gosec)

2. **Markdown** - Markdown file linting
   - Checks README, docs

3. **YAML** - YAML file linting
   - Configuration: `.yamllint.yml`

4. **Dockerfile** - Dockerfile linting (hadolint)

**Fix issues:**
```bash
# Run golangci-lint locally
golangci-lint run

# Auto-fix some issues
golangci-lint run --fix
```

---

### 🔒 Security (`security.yml`)

**Trigger:**
- Push/PR to `main` or `develop`
- Weekly schedule (Monday 00:00 UTC)

**Scanners:**
1. **gosec** - Go security scanner
   - Detects common security issues
   - Results uploaded to GitHub Security

2. **govulncheck** - Go vulnerability check
   - Scans for known vulnerabilities in dependencies

3. **CodeQL** - GitHub's semantic code analysis
   - Deep security analysis
   - Dataflow analysis

4. **Trivy** - Container and filesystem scanner
   - Vulnerability scanning
   - License scanning

5. **OSSF Scorecard** - Security scorecard
   - Rates security posture
   - Best practices check

**View results:**
- GitHub Security → Code scanning alerts

---

### 📦 Dependabot (`dependabot.yml`)

**Automatic dependency updates for:**
- Go modules (weekly, Monday 09:00)
- GitHub Actions (weekly, Monday 09:00)
- Docker base images (weekly, Monday 09:00)

**Features:**
- Grouped updates for related packages (gRPC, OTEL)
- Auto-labeling
- Auto-assignment to maintainer

**Settings:**
- Max 10 PRs for Go modules
- Max 5 PRs for Actions/Docker

---

## 🛡️ Security Best Practices

### SLSA Build Provenance

All releases include SLSA Level 3 provenance:
- Tamper-proof build information
- Verifiable build process
- Supply chain security

### Code Scanning

Multiple security scanners:
- **gosec** - Go-specific vulnerabilities
- **CodeQL** - Dataflow analysis
- **Trivy** - Known vulnerabilities
- **govulncheck** - Go vulnerability database

### Dependency Management

Automated updates via Dependabot:
- Security patches
- Version updates
- Grouped related updates

---

## 📊 Badges

Add to README.md:

```markdown
[![CI](https://github.com/dantte-lp/ocserv-agent/actions/workflows/ci.yml/badge.svg)](https://github.com/dantte-lp/ocserv-agent/actions/workflows/ci.yml)
[![Lint](https://github.com/dantte-lp/ocserv-agent/actions/workflows/lint.yml/badge.svg)](https://github.com/dantte-lp/ocserv-agent/actions/workflows/lint.yml)
[![Security](https://github.com/dantte-lp/ocserv-agent/actions/workflows/security.yml/badge.svg)](https://github.com/dantte-lp/ocserv-agent/actions/workflows/security.yml)
[![OSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/dantte-lp/ocserv-agent/badge)](https://securityscorecards.dev/viewer/?uri=github.com/dantte-lp/ocserv-agent)
[![codecov](https://codecov.io/gh/dantte-lp/ocserv-agent/branch/main/graph/badge.svg)](https://codecov.io/gh/dantte-lp/ocserv-agent)
```

---

## 🔧 Local Development

### Run linters locally

```bash
# golangci-lint
golangci-lint run

# YAML lint
yamllint .

# Markdown lint
markdownlint '**/*.md'

# Dockerfile lint
hadolint Dockerfile
```

### Run tests

```bash
# Unit tests
go test -v -race -coverprofile=coverage.out ./...

# View coverage
go tool cover -html=coverage.out
```

### Run security scanners

```bash
# gosec
gosec ./...

# govulncheck
govulncheck ./...

# Trivy
trivy fs .
```

---

## 📝 Workflow Configuration

### Secrets Required

**Optional (for enhanced features):**
- `CODECOV_TOKEN` - Codecov coverage reporting

All other secrets are automatically provided by GitHub:
- `GITHUB_TOKEN` - Automatic
- `secrets.GITHUB_TOKEN` - Automatic

### Permissions

Workflows use minimal required permissions:
- `contents: read` - Read repository
- `contents: write` - Create releases (release.yml only)
- `security-events: write` - Upload security results
- `id-token: write` - SLSA provenance

---

## 🚦 Status Checks

Required checks for PRs:
- ✅ All CI tests pass
- ✅ Code is formatted
- ✅ No lint errors
- ✅ No security vulnerabilities (high/critical)
- ✅ OSSF Scorecard passes

---

## 📚 References

- [SLSA Framework](https://slsa.dev/)
- [golangci-lint](https://golangci-lint.run/)
- [CodeQL](https://codeql.github.com/)
- [Trivy](https://github.com/aquasecurity/trivy)
- [OSSF Scorecard](https://github.com/ossf/scorecard)
- [Dependabot](https://docs.github.com/en/code-security/dependabot)
