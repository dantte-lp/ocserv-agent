# Local Testing Guide

Save hours of GitHub Actions time by testing locally before pushing!

## 🎯 Full Pipeline (Recommended)

**New!** Unified build script to run the complete CI/CD pipeline locally:

```bash
# Run everything: security + tests + build
make build-all

# Or run individually:
make build-all-security  # Security scans (gosec, govulncheck, trivy)
make build-all-test      # Unit tests + linting
make build-all-build     # Multi-platform builds (4 platforms)
```

What gets executed:
- ✅ **Security scans**: gosec (with SARIF fix), govulncheck, trivy
- ✅ **Unit tests**: coverage report, race detector
- ✅ **Linting**: golangci-lint (30+ linters)
- ✅ **Multi-platform build**: Linux/FreeBSD × amd64/arm64
- ✅ **Artifacts**: tar.gz archives + SHA256 checksums

Results are saved to:
- `deploy/compose/security-results/` - SARIF and JSON reports
- `bin/` - binaries and checksums
- `coverage.out`, `coverage.html` - test coverage

**Benefits:**
- 🚀 One script runs everything
- 🐳 Everything runs in containers (isolated)
- 💰 Save GitHub Actions minutes
- 🔍 Early problem detection

## 🚀 Quick Check (2-3 seconds)

For a quick check before committing:

```bash
./scripts/quick-check.sh
```

Checks:
- ✅ Code formatting (gofmt)
- ✅ go vet
- ✅ Project build
- ✅ Basic tests

## 🔬 Full Check (Like CI)

For a complete check before pushing to GitHub:

```bash
./scripts/test-local.sh
```

Checks everything that GitHub Actions checks:
- ✅ Protobuf code generation
- ✅ Dependency verification (go mod verify)
- ✅ Formatting (gofmt)
- ✅ go vet
- ✅ go mod tidy
- ✅ Tests with race detector and coverage
- ✅ Build for all platforms (Linux/FreeBSD, amd64/arm64)
- ✅ Linters (golangci-lint, markdownlint, yamllint)
- ⚠️ Security checks (optional, slow)

## ⚙️ Environment Variables

```bash
# Skip tests
RUN_TESTS=false ./scripts/test-local.sh

# Skip linters
RUN_LINT=false ./scripts/test-local.sh

# Enable security checks (slow)
RUN_SECURITY=true ./scripts/test-local.sh

# Skip binary builds
RUN_BUILD=false ./scripts/test-local.sh

# Skip protobuf generation
SKIP_PROTO=true ./scripts/test-local.sh

# Combination
RUN_SECURITY=true RUN_BUILD=false ./scripts/test-local.sh
```

## 📦 Tool Installation

### Required (for full check)

```bash
# Go tools
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Protobuf compiler
sudo apt-get install protobuf-compiler

# golangci-lint
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin
```

### Optional (for linters)

```bash
# Markdown lint
npm install -g markdownlint-cli

# YAML lint
pip install yamllint
```

### For security checks

```bash
# gosec (static security analyzer)
go install github.com/securego/gosec/v2/cmd/gosec@latest

# govulncheck (known vulnerabilities)
go install golang.org/x/vuln/cmd/govulncheck@latest
```

## 🎯 Recommended Workflow

### Before each commit

```bash
./scripts/quick-check.sh
```

### Before pushing to GitHub

```bash
./scripts/test-local.sh
```

If everything passes - you're good to push!

### Before a release

```bash
RUN_SECURITY=true ./scripts/test-local.sh
```

## 🔒 Security Testing (Locally)

To run security scanning locally, use Podman Compose:

### All security tests at once

```bash
make security-check
# or
./scripts/security-check.sh
```

Runs:
- ✅ **Gosec** - Static security analysis for Go code
- ✅ **govulncheck** - Check for known vulnerabilities in dependencies
- ✅ **Trivy** - Vulnerability scanning for code and dependencies

### Individual tests

```bash
# Only Gosec
make security-gosec

# Only govulncheck
make security-govulncheck

# Only Trivy
make security-trivy
```

### Results

All results are saved to `deploy/compose/security-results/`:

```bash
# View findings
cat deploy/compose/security-results/gosec-fixed.sarif | jq '.runs[0].results[]'
cat deploy/compose/security-results/trivy.sarif | jq '.runs[0].results[]'
cat deploy/compose/security-results/govulncheck.json | jq

# Count issues
jq '.runs[0].results | length' deploy/compose/security-results/gosec-fixed.sarif
jq '.runs[0].results | length' deploy/compose/security-results/trivy.sarif
```

### Why locally?

1. **Faster** - results in 30-60 seconds vs 3-5 minutes in GitHub Actions
2. **Free** - doesn't consume GitHub Actions minutes
3. **Pre-commit** - find issues before pushing
4. **GitHub-compatible** - same SARIF files as CI

**Important:** SARIF files in `gosec-fixed.sarif` contain automatic fixes for Gosec's problematic format and are ready for upload to GitHub Security.

## 🔧 Pre-commit Hook (Optional)

To automatically run quick-check before each commit:

```bash
cat > .git/hooks/pre-commit <<'EOF'
#!/bin/bash
./scripts/quick-check.sh
EOF

chmod +x .git/hooks/pre-commit
```

Temporarily disable:
```bash
git commit --no-verify
```

## 📊 GitHub Actions Savings

**Example:**
- 1 push = ~4-5 minutes Actions (CI + Lint + Security)
- 10 pushes per day = 40-50 minutes
- 30 days = **1200-1500 minutes per month**

With local testing:
- Local check = 10-30 seconds
- Push only when everything works
- Savings = **up to 80% Actions minutes** 💰

## 🐛 Troubleshooting

### "protoc not found"

```bash
sudo apt-get install protobuf-compiler
```

### "golangci-lint not found"

```bash
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin
```

### "Tests fail locally but pass in CI"

Check Go version:
```bash
go version  # should be 1.25+
```

### "Build fails for FreeBSD"

This is normal if you're not on FreeBSD. CI will build it correctly.
You can skip: `RUN_BUILD=false ./scripts/test-local.sh`

## 📝 What Gets Checked in CI

### CI Workflow (.github/workflows/ci.yml)
- ✅ Tests (race detector, coverage)
- ✅ Build (all platforms)
- ✅ Integration tests
- ✅ Code quality (gofmt, go vet, go mod tidy)

### Lint Workflow (.github/workflows/lint.yml)
- ✅ golangci-lint (30+ linters)
- ✅ Markdown lint
- ✅ YAML lint
- ✅ Dockerfile lint

### Security Workflow (.github/workflows/security.yml)
- ✅ gosec (static analysis)
- ✅ CodeQL (deep analysis)
- ✅ Trivy (container scanning)
- ✅ OSSF Scorecard

### Release Workflow (.github/workflows/release.yml)
- ✅ Multi-platform builds
- ✅ SHA256 checksums
- ✅ SLSA Level 3 provenance
- ✅ Container images
- ✅ GitHub Release creation

## 🎓 Best Practices

1. **Before committing**: `./scripts/quick-check.sh` (fast)
2. **Before pushing**: `./scripts/test-local.sh` (complete)
3. **Before release**: `RUN_SECURITY=true ./scripts/test-local.sh` (everything + security)
4. **In CI**: Automatically on every push/PR

This enables:
- 🚀 Faster development (find errors locally)
- 💰 Save GitHub Actions minutes
- ✅ Confident pushing (you know CI will pass)
- 🔒 Maintain code quality

## 🔗 Related Documents

- [Contributing Guide](../.github/CONTRIBUTING.md)
- [Workflows Documentation](../.github/WORKFLOWS.md)
- [CI Configuration](../.github/workflows/ci.yml)
