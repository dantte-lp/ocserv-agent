# Security Tools Documentation

## Overview

This project implements a comprehensive multi-layered security scanning approach following OWASP, OSSF, and 2025 DevSecOps best practices.

## Architecture

```
┌─────────────┬──────────────────┬────────────────┬──────────────┐
│ Pre-commit  │ CI Build Time    │ Post-build     │ Runtime      │
├─────────────┼──────────────────┼────────────────┼──────────────┤
│ Local hooks │ • SAST           │ • SBOM         │ • Scorecard  │
│ (optional)  │ • CodeQL         │ • Binary scan  │ • Trivy      │
│             │ • gosec          │ • Grype        │ • Dependabot │
│             │ • semgrep        │                │              │
│             │ • govulncheck    │                │              │
│             │ • OSV-Scanner    │                │              │
│             │ • Nancy          │                │              │
│             │ • Secret scan    │                │              │
│             │ • License check  │                │              │
└─────────────┴──────────────────┴────────────────┴──────────────┘
```

## Security Tools Matrix

### SAST (Static Application Security Testing)

#### 1. Semgrep
**Version:** Latest (via pip)
**Purpose:** Multi-language SAST with 2000+ security rules
**Location:** `.github/workflows/security.yml`

**Features:**
- Go-specific security patterns
- Secret detection rules
- Security audit ruleset
- SARIF output for GitHub Security

**Configuration:**
```yaml
semgrep scan \
  --config auto \
  --config p/golang \
  --config p/security-audit \
  --config p/secrets \
  --sarif
```

**Rules Applied:**
- `p/golang` - Go best practices and security patterns
- `p/security-audit` - General security vulnerabilities
- `p/secrets` - Hardcoded credentials detection
- `auto` - Language-specific automatic detection

#### 2. gosec
**Version:** Latest (go install)
**Purpose:** Go-specific security scanner
**Location:** `.github/workflows/security.yml`

**Checks:**
- SQL injection
- Command injection
- Integer overflow
- Weak cryptography
- Insecure random
- Path traversal
- Unhandled errors

**Configuration:**
```bash
gosec -fmt=sarif -out=results.sarif ./...
```

#### 3. staticcheck
**Version:** Latest (go install)
**Purpose:** Advanced Go linter
**Location:** `.github/workflows/ci.yml`

**Features:**
- Bug detection
- Code simplification
- Performance issues
- Style violations

#### 4. errcheck
**Version:** Latest (go install)
**Purpose:** Unchecked error detection
**Location:** `.github/workflows/ci.yml`

**Purpose:**
- Ensures all errors are properly handled
- Prevents silent failures

#### 5. ineffassign
**Version:** Latest (go install)
**Purpose:** Dead code detection
**Location:** `.github/workflows/ci.yml`

**Purpose:**
- Detects ineffectual assignments
- Improves code quality

### Secret Detection

#### 1. Gitleaks
**Version:** 8.28.0
**Purpose:** Fast and lightweight secret scanner
**Location:** `.github/workflows/security.yml`

**Features:**
- Scans entire git history
- 1000+ secret patterns
- SARIF output
- Fast execution (<10 seconds)

**Configuration:**
```bash
gitleaks detect \
  --source . \
  --report-format sarif \
  --report-path gitleaks-results.sarif \
  --verbose
```

**Detects:**
- API keys (AWS, GitHub, Slack, etc.)
- Database credentials
- Private keys (RSA, SSH)
- OAuth tokens
- Generic secrets (passwords, tokens)

#### 2. TruffleHog
**Version:** 3.90.3
**Purpose:** Comprehensive secret scanner with verification
**Location:** `.github/workflows/security.yml`

**Features:**
- 800+ secret types
- **Verification** - validates secrets are active
- Git history scanning
- Entropy-based detection

**Configuration:**
```bash
trufflehog git file://. \
  --only-verified \
  --json
```

**Why Both?**
- Gitleaks: Fast, pattern-based (catches format violations)
- TruffleHog: Thorough, verifies credentials (catches active secrets)
- **Dual-tool approach ensures maximum coverage**

### Vulnerability Scanning

#### 1. govulncheck
**Version:** Latest (go install)
**Purpose:** Go-specific vulnerability scanner (official)
**Location:** `.github/workflows/security.yml`

**Features:**
- Uses official Go vulnerability database
- Analyzes call graph (not just dependencies)
- Zero false positives
- Go team maintained

**Database:**
- https://vuln.go.dev/

#### 2. OSV-Scanner
**Version:** 2.x (latest)
**Purpose:** Multi-ecosystem vulnerability scanner (Google)
**Location:** `.github/workflows/security.yml`

**Features:**
- Scans go.mod, package-lock.json, requirements.txt, etc.
- Uses OSV (Open Source Vulnerabilities) database
- Transitive dependency analysis
- Fast and accurate

**Databases:**
- Go: https://osv.dev/
- npm, PyPI, Maven, etc.

#### 3. Nancy
**Version:** Latest (go install)
**Purpose:** OSS Index dependency scanner for Go
**Location:** `.github/workflows/security.yml`

**Features:**
- Sonatype OSS Index integration
- Go module vulnerability detection
- Complementary to govulncheck

**Configuration:**
```bash
go list -json -deps ./... | nancy sleuth
```

#### 4. Grype
**Version:** 0.101.1
**Purpose:** Binary and container vulnerability scanner
**Location:** `.github/workflows/ci.yml` (post-build)

**Features:**
- Scans compiled binaries
- Container image scanning
- Multiple vulnerability databases
- DB Schema v6 with CISA KEV integration

**Configuration:**
```bash
grype file:ocserv-agent \
  --output sarif \
  --file grype-results.sarif
```

**Databases:**
- NVD (National Vulnerability Database)
- GitHub Security Advisory
- Alpine, Debian, RHEL, Ubuntu security trackers
- CISA KEV (Known Exploited Vulnerabilities)

### SBOM & Supply Chain

#### 1. Syft
**Version:** 1.34.2
**Purpose:** SBOM generation
**Location:** `.github/workflows/ci.yml`, `release.yml`

**Features:**
- Generates CycloneDX and SPDX formats
- Scans binaries, containers, filesystems
- License detection
- Fast scans (DB schema v6)

**Outputs:**
- `sbom-repo.json` - Source repository SBOM
- `sbom-{binary}.json` - Per-binary SBOM
- `sbom-spdx.json` - SPDX format for compliance

**Use Cases:**
- Supply chain transparency
- License compliance
- Vulnerability tracking
- EU Cyber Resilience Act (CRA) compliance

#### 2. Cosign
**Version:** 3.0.2
**Purpose:** Container signing and verification (Sigstore)
**Location:** `.github/workflows/release.yml`

**Features:**
- Keyless signing (OIDC)
- Sigstore public-good instance
- Container image verification
- Bundle format support (v3)

**Configuration:**
```bash
cosign sign --yes ghcr.io/dantte-lp/ocserv-agent:latest
```

**Verification:**
```bash
cosign verify \
  --certificate-identity-regexp='.*' \
  --certificate-oidc-issuer='https://token.actions.githubusercontent.com' \
  ghcr.io/dantte-lp/ocserv-agent:latest
```

### License Compliance

#### go-licenses
**Version:** Latest (go install)
**Purpose:** Go dependency license analysis
**Location:** `.github/workflows/security.yml`

**Features:**
- Lists all dependency licenses
- Detects copyleft licenses (GPL, AGPL, LGPL)
- Exports license reports

**Configuration:**
```bash
# Generate report
go-licenses report ./... > licenses-report.txt

# CSV export
go-licenses csv ./...

# Check for GPL
go-licenses csv ./... | grep -iE 'GPL|AGPL|LGPL'
```

**Use Cases:**
- License compliance checking
- Copyleft detection
- Legal audit trail

### Project Security Health

#### OSSF Scorecard
**Version:** 5.3.0
**Purpose:** Project security health metrics
**Location:** `.github/workflows/security.yml`

**Checks (25 total):**
- Binary-Artifacts
- Branch-Protection
- CI-Tests
- Code-Review
- Contributors
- Dangerous-Workflow
- Dependency-Update-Tool
- Fuzzing
- License
- Maintained
- Packaging
- Pinned-Dependencies
- SAST
- Security-Policy
- Signed-Releases
- Token-Permissions
- Vulnerabilities

**Configuration:**
```bash
scorecard --repo=github.com/dantte-lp/ocserv-agent \
  --format=sarif \
  --output-file=results.sarif
```

**Current Score:** 6.6/10
**Target Score:** 9.5+/10

See [OSSF_SCORECARD_IMPROVEMENTS.md](OSSF_SCORECARD_IMPROVEMENTS.md) for improvement plan.

### Container Security

#### Trivy
**Version:** Latest (system package)
**Purpose:** Comprehensive container and filesystem scanner
**Location:** `.github/workflows/security.yml`

**Features:**
- OS package vulnerabilities
- Language-specific vulnerabilities (Go, npm, pip, etc.)
- Misconfigurations (Dockerfile, Kubernetes, Terraform)
- Secret scanning
- License detection

**Scan Types:**
- Container images
- Filesystem
- Git repositories
- Kubernetes clusters

## Tool Installation

All security tools are pre-installed in GitHub runners:

### Debian Runner (`github-runner-debian`)
```dockerfile
# Go tools
RUN go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest && \
    go install honnef.co/go/tools/cmd/staticcheck@latest && \
    go install github.com/securego/gosec/v2/cmd/gosec@latest && \
    go install github.com/kisielk/errcheck@latest && \
    go install github.com/gordonklaus/ineffassign@latest && \
    go install golang.org/x/vuln/cmd/govulncheck@latest && \
    go install github.com/google/osv-scanner/v2/cmd/osv-scanner@latest && \
    go install github.com/sonatype-nexus-community/nancy@latest && \
    go install github.com/google/go-licenses@latest

# SBOM and scanning tools
ARG SYFT_VERSION="v1.34.2"
RUN curl -sSfL https://raw.githubusercontent.com/anchore/syft/main/install.sh | sh -s -- -b /usr/local/bin ${SYFT_VERSION}

ARG GRYPE_VERSION="v0.101.1"
RUN curl -sSfL https://raw.githubusercontent.com/anchore/grype/main/install.sh | sh -s -- -b /usr/local/bin ${GRYPE_VERSION}

ARG COSIGN_VERSION="v3.0.2"
RUN curl -fsSL -o /usr/local/bin/cosign \
    https://github.com/sigstore/cosign/releases/download/${COSIGN_VERSION}/cosign-linux-amd64 && \
    chmod +x /usr/local/bin/cosign

# Secret scanners
ARG GITLEAKS_VERSION="8.28.0"
RUN curl -fsSL -o gitleaks.tar.gz \
    https://github.com/gitleaks/gitleaks/releases/download/v${GITLEAKS_VERSION}/gitleaks_${GITLEAKS_VERSION}_linux_x64.tar.gz && \
    tar -xzf gitleaks.tar.gz -C /usr/local/bin gitleaks

ARG TRUFFLEHOG_VERSION="3.90.3"
RUN curl -fsSL -o trufflehog.tar.gz \
    https://github.com/trufflesecurity/trufflehog/releases/download/v${TRUFFLEHOG_VERSION}/trufflehog_${TRUFFLEHOG_VERSION}_linux_amd64.tar.gz && \
    tar -xzf trufflehog.tar.gz -C /usr/local/bin trufflehog

# SAST
RUN pip3 install --no-cache-dir semgrep

# Scorecard
ARG SCORECARD_VERSION="5.3.0"
RUN curl -fsSL -o scorecard.tar.gz \
    https://github.com/ossf/scorecard/releases/download/v${SCORECARD_VERSION}/scorecard_${SCORECARD_VERSION}_linux_amd64.tar.gz && \
    tar -xzf scorecard.tar.gz -C /usr/local/bin scorecard
```

### Oracle Linux Runner (`github-runner`)
Same tools installed for RPM package building.

## Workflow Integration

### security.yml
All security scans run in parallel for fast feedback:

```yaml
jobs:
  codeql:        # GitHub CodeQL
  gosec:         # Go security scanner
  semgrep:       # Multi-language SAST
  govulncheck:   # Go vulnerabilities
  nancy:         # OSS Index scanner
  osv-scanner:   # Multi-ecosystem scanner
  trivy:         # Container security
  scorecard:     # Project health
  gitleaks:      # Secret detection
  trufflehog:    # Secret verification
  license:       # License compliance
```

### ci.yml
Post-build security:

```yaml
jobs:
  build:         # Compile binaries
  sbom:          # Generate SBOMs
  grype-scan:    # Scan binaries
```

## Security Standards Compliance

| Standard | Status | Tools Used |
|----------|--------|-----------|
| **OWASP Top 10** | ✅ | semgrep, gosec, CodeQL |
| **OSSF Scorecard** | 🟡 6.6/10 | scorecard, all tools |
| **SLSA Build L3** | ✅ | SBOM (syft), Provenance, Cosign |
| **OSPS Baseline L3** | ✅ | Full stack |
| **EU CRA** | ✅ | SBOM (CycloneDX, SPDX) |
| **NIST SSDF** | ✅ | Multi-layer scanning |

## Performance

Typical execution times on self-hosted runner:

| Tool | Duration | When |
|------|----------|------|
| Gitleaks | ~5s | Every commit |
| TruffleHog | ~15s | Every commit |
| gosec | ~10s | Every commit |
| semgrep | ~30s | Every commit |
| govulncheck | ~8s | Every commit |
| OSV-Scanner | ~12s | Every commit |
| Nancy | ~5s | Every commit |
| Syft | ~3s/binary | Post-build |
| Grype | ~15s/binary | Post-build |
| Trivy | ~20s | Every commit |
| Scorecard | ~45s | Every commit |
| CodeQL | ~2min | Every commit |

**Total parallel execution:** ~2 minutes (all security checks)

## Best Practices

### 1. Layered Defense
- **Multiple tools** for same vulnerability class
- **Different approaches**: pattern-based + ML-based + signature-based
- **Complementary coverage**: Gitleaks (fast) + TruffleHog (thorough)

### 2. Shift-Left Security
- **Pre-commit**: Optional local hooks
- **CI Build Time**: Fast feedback (<5 min)
- **Post-build**: Binary analysis before release
- **Runtime**: Continuous monitoring

### 3. Supply Chain Security
- **SBOM** generation for all artifacts
- **Dependency pinning** (upcoming)
- **Container signing** with Cosign
- **SLSA provenance** generation

### 4. Zero False Positive Policy
- **govulncheck**: Analyzes actual usage (not just presence)
- **SARIF reports**: Integrated with GitHub Security
- **Verified secrets**: TruffleHog validates credentials

### 5. Compliance
- **SBOM formats**: CycloneDX (industry) + SPDX (legal)
- **License tracking**: Automated copyleft detection
- **Audit trail**: All scans logged and archived

## Troubleshooting

### Common Issues

#### 1. Gitleaks False Positives
```bash
# Add to .gitleaksignore
<commit_sha>:<rule_id>
```

#### 2. TruffleHog Slow Scan
```bash
# Use --since to scan recent commits only
trufflehog git file://. --since=1w
```

#### 3. Grype Database Updates
```bash
# Update vulnerability database
grype db update
```

#### 4. Semgrep Custom Rules
Create `.semgrep.yml`:
```yaml
rules:
  - id: custom-rule
    pattern: dangerous_function()
    message: Don't use dangerous_function
    severity: WARNING
```

## References

- [OWASP SAST Tools](https://owasp.org/www-community/Source_Code_Analysis_Tools)
- [OSSF Scorecard](https://github.com/ossf/scorecard)
- [Sigstore Cosign](https://docs.sigstore.dev/cosign/overview/)
- [SLSA Framework](https://slsa.dev/)
- [SBOM Guide (CISA)](https://www.cisa.gov/sbom)
- [Gitleaks Documentation](https://github.com/gitleaks/gitleaks)
- [TruffleHog Documentation](https://github.com/trufflesecurity/trufflehog)
- [Semgrep Rules](https://semgrep.dev/r)
- [Go Vulnerability Database](https://vuln.go.dev/)
