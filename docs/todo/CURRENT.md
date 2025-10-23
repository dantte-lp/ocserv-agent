# Current TODO - ocserv-agent

**Last Updated:** 2025-10-23
**Last Commit:** 9ee265d - devops: add automatic gofmt to local development workflow
**Status:** v0.4.0 IN PROGRESS - Unit tests (97.1% config) + DevOps improvements (automatic formatting)

## 🎉 v0.4.0: DevOps Improvements - COMPLETED!

**Local Development Workflow:**
- ✅ Automatic code formatting (scripts/quick-check.sh)
- ✅ Git hooks for pre-commit and pre-push (scripts/install-hooks.sh)
- ✅ Updated README.md with git hooks documentation

**Git Hooks:**
- ✅ pre-commit: Auto-formats Go code with gofmt before each commit
- ✅ pre-push: Runs quick-check.sh before each push
- ✅ One-time installation: `./scripts/install-hooks.sh`

**Benefits:**
- Eliminates CI formatting failures
- Consistent code style automatically
- Fast local checks (2-3 seconds)
- Optional (can skip with --no-verify)

## 🎉 v0.4.0: Unit Tests - IN PROGRESS!

**Test Coverage Achieved:**
- ✅ internal/config: 97.1% coverage (exceeds >80% target)
- ✅ internal/cert: 77.6% coverage (close to 80% target)
- ✅ internal/ocserv/config.go: 82-100% coverage (per function)

**Remaining Tests:**
- [ ] Unit tests for internal/grpc (server, handlers)
- [ ] Unit tests for internal/ocserv (manager, occtl, systemctl)
- [ ] Unit tests for internal/health
- [ ] Unit tests for internal/metrics
- [ ] Unit tests for internal/telemetry
- [ ] Achieve >80% overall test coverage

## 🎉 Phase 1: Core - COMPLETED!

All critical Phase 1 tasks done ✅

## 🎉 Phase 2: ocserv Integration - COMPLETED!

All critical Phase 2 tasks done ✅
- ✅ Systemctl wrapper (start, stop, restart, reload, status)
- ✅ Occtl wrapper (show users/status/stats, disconnect)
- ✅ Command validation and security (whitelist, sanitization, injection protection)
- ✅ ExecuteCommand RPC fully functional

## 🎉 Phase 3: occtl Commands - COMPLETED!

**All 16/16 occtl commands implemented:**
- ✅ Complete type definitions (occtl_types.go - 179 lines)
- ✅ All 16 occtl commands with JSON parsing
- ✅ Production-tested types (DTLS, compression, multiple sessions)
- ✅ Full occtl compatibility (100%)

**For v0.4.0+:**
- [ ] ShowEvents() streaming support (requires ServerStream RPC)
- [ ] ocpasswd wrapper
- [ ] UpdateConfig RPC
- [x] Unit tests for internal/config (97.1% coverage) ✅
- [ ] Unit tests for other packages (cert, grpc, ocserv) - targeting >80% overall

## 🎉 v0.2.1: CI/CD Infrastructure - COMPLETED!

**GitHub Actions Workflows (4 workflows):**
- ✅ CI Pipeline (ci.yml) - Tests, builds, coverage
- ✅ Lint Pipeline (lint.yml) - golangci-lint, markdown, YAML, Dockerfile
- ✅ Security Pipeline (security.yml) - gosec, CodeQL, Trivy, OSSF Scorecard
- ✅ Release Pipeline (release.yml) - SLSA Level 3, multi-arch builds

**Smart CI Optimization:**
- ✅ Path filtering - skip heavy checks for docs-only changes
- ✅ File-type filtering - each lint runs only for relevant files
- ✅ Resource optimization - docs PRs only run markdown lint

**Branch Protection:**
- ✅ PR workflow configured
- ✅ Branch protection rules (no force push, no delete)
- ✅ Required status checks (temporarily disabled for initial setup)

**Documentation:**
- ✅ CONTRIBUTING.md (339 lines) - complete development guide
- ✅ WORKFLOWS.md - CI/CD pipeline documentation
- ✅ README display fix (GitHub homepage)
- ✅ Platform updates (Linux + FreeBSD: amd64/x86_64, arm64/aarch64)

**Code Quality:**
- ✅ All Go code formatted with gofmt
- ✅ golangci-lint configuration (30+ linters)
- ✅ YAML and Markdown linting

**Dependencies:**
- ✅ golang.org/x/net 0.34.0 → 0.38.0 (security fix)
- ✅ Dependabot configuration (auto updates)

## 🎉 v0.3.0: Certificate Auto-Generation - COMPLETED!

**Certificate Management (internal/cert):**
- ✅ Self-signed certificate generation (ECDSA P-256)
- ✅ Auto-generate on first run (bootstrap mode)
- ✅ CLI commands: gencert, help, version
- ✅ SHA256 fingerprint calculation
- ✅ Proper permissions (0644 certs, 0600 keys)

**Config Auto-Generate:**
- ✅ `auto_generate: true` option in TLS config
- ✅ Bootstrap certificates on config load
- ✅ Conditional validation (skip if auto_generate)
- ✅ Informative console output with warnings

**Build Improvements:**
- ✅ Versioned tar.gz archives (ocserv-agent-{version}-{os}-{arch}.tar.gz)
- ✅ FreeBSD support (amd64, arm64)
- ✅ SHA256 checksums for all archives
- ✅ SLSA Level 3 provenance

**Documentation:**
- ✅ docs/CERTIFICATES.md - Complete certificate guide
- ✅ TESTING_PROD.md - Production testing guide
- ✅ Sanitized sensitive data (RFC examples)

**Bug Fixes:**
- ✅ Go 1.24 covdata tool issue (toolchain directive)
- ✅ CI test matrix (Go 1.25 only)

**Status:** BETA - Published with all platforms (Linux + FreeBSD, amd64 + arm64)

**Release Assets:**
- ✅ 4 platform binaries with SHA256 checksums
- ✅ SLSA Level 3 provenance attestation
- ✅ Marked as pre-release (BETA status)
- ✅ Complete release notes and documentation

**Post-Release Fixes:**
- ✅ SLSA workflow job dependency ordering (68185df)
- ✅ Container build protobuf include paths - libprotobuf-dev (4b65e05)
- ✅ Documentation cleanup - removed CLAUDE_PROMPT.md from repo (ad75891)
- ✅ TODO documentation updates with v0.3.0 status (903797d)
- ✅ Marked v0.3.0 as BETA pre-release (89897c1)
- ✅ Local testing scripts for CI/CD (01ebe67, 597eb62)
- ✅ OSSF Scorecard permission error fix (be3c5c0)
- ✅ Security workflow paths-ignore fix (b8aeb6e)
- ✅ Gosec SARIF format fix with jq processing (241c28b)
- ✅ Local security testing infrastructure (podman-compose) (241c28b)
- ✅ Documentation updates and MIT license (5f0d2a7)
- ✅ Binary installation path fix - /etc/ocserv-agent (18fd5c8)
- ✅ Configuration validation logging (f6f077d)
- ✅ Unified build pipeline script (09d3c50)
- ✅ Documentation updates for unified pipeline (8500b69)
- ✅ Fix VERSION variable expansion in docker-compose (4a83924)
- ✅ Fix command order for RAW binaries (0161ffc)
- ✅ gRPC reflection support for grpcurl testing (cb1f848)
- ✅ Production deployment and testing scripts (deploy-and-test.sh, test-grpc.sh)
- ✅ gRPC testing documentation (GRPC_TESTING.md)

## 🎉 v0.3.1: Critical Bugfixes + Documentation - COMPLETED!

**Critical Bugfix - occtl JSON Parsing:**
- ✅ **FIXED:** User count showing 0 when users connected
- ✅ Switched from text parsing to JSON mode (`occtl -j`)
- ✅ Added 40+ JSON fields per user (vs 6 in text mode)
- ✅ Fixed Routes field polymorphism (string vs []string)
- ✅ Production-tested with 3 real VPN users ✅
- ✅ Commit: 4fd990f

**Security Improvements:**
- ✅ Removed hardcoded credentials from repository (3c2d96a)
- ✅ Sanitized deployment scripts to use environment variables
- ✅ Created SECURITY.md vulnerability disclosure policy (37310dc)
- ✅ OSSF Scorecard: 4.9/10 → 5.9/10 (+1.0)

**Documentation (5 new/updated documents):**
- ✅ docs/OCCTL_COMMANDS.md - Complete command reference (8837ee6)
  - 13/16 working commands with examples
  - 40+ user data fields documentation
  - Known issues with occtl 1.3.0 JSON bugs
- ✅ docs/GRPC_TESTING.md - gRPC testing guide (801b32d)
  - grpcurl testing instructions
  - Production deployment procedures
- ✅ docs/OSSF_SCORECARD_IMPROVEMENTS.md - Security roadmap (37310dc)
  - Current: 4.9/10, target: 7.5+/10
  - 4-phase improvement plan
- ✅ docs/todo/OCSERV_COMPATIBILITY.md - Updated status (37310dc)
  - Real production results: 13/16 working
  - Documented occtl bugs
  - Score: 40/100 → 36/100 (realistic)
- ✅ SECURITY.md - Security policy (37310dc)
  - Vulnerability disclosure process
  - Response timeline (48h initial)

**Testing Results:**
- ✅ Tested 10+ occtl commands on production
- ✅ Verified with 3 connected VPN users
- ✅ Identified 3 upstream occtl bugs (iroutes, sessions)
- ✅ All core commands working correctly

**Status:** BETA - Ready for production with full documentation

## 🔴 Critical (Next Steps - v0.4.0)

### OSSF Scorecard Improvements (HIGH PRIORITY)

See: `docs/OSSF_SCORECARD_IMPROVEMENTS.md` for complete plan

**Phase 1 - Quick Wins (Target: 6.5/10):**
- [ ] **[SECURITY]** Setup branch protection rules
  - Require pull requests for all changes
  - Require 1 approval before merge
  - Dismiss stale reviews
  - Linear history enforcement
  - **Impact:** Code-Review: 0 → 10 (+1.0 point)

- [ ] **[SECURITY]** Restrict GitHub workflow token permissions
  - Set minimal permissions per workflow
  - Explicit permissions for each job
  - Remove unnecessary write access
  - **Impact:** Token-Permissions: 0 → 10 (+1.0 point)

- [ ] **[SECURITY]** Setup GPG commit signing
  - Generate GPG key
  - Configure git signing
  - Add key to GitHub
  - Sign all commits going forward

- [ ] **[SECURITY]** Create .github/CODEOWNERS
  - Define code owners
  - Automatic review requests

**Phase 2 - Dependency Pinning (Target: 7.5/10):**
- [ ] **[SECURITY]** Pin all GitHub Actions to SHA hashes (49+ dependencies)
  - actions/checkout@v4 → @sha
  - actions/setup-go@v5 → @sha
  - golangci/golangci-lint-action@v4 → @sha
  - ... (22 unique actions total)
  - **Impact:** Pinned-Dependencies: 0 → 10 (+1.0 point)

- [ ] **[SECURITY]** Pin Docker base images to digests
  - golang:1.25-alpine → @sha256:...
  - Update all compose files

### ocserv Features (MEDIUM PRIORITY)

See: `docs/todo/OCSERV_COMPATIBILITY.md` for complete roadmap

**High Priority:**
- [x] **[FEATURE]** Complete missing occtl commands (13/16 working!)
  - ✅ show user [NAME], show id [ID]
  - ✅ show users, status, stats, ip bans
  - ✅ disconnect, unban, reload
  - ⚠️ show iroutes, sessions (occtl bugs)
  - [ ] show events (real-time streaming) - needs ServerStream RPC

- [ ] **[FEATURE]** Implement ocpasswd wrapper
  - User management (add, delete, lock, unlock)
  - Password hashing (SHA-512/MD5)
  - Group assignment
  - Integration with UpdateConfig RPC

- [ ] **[FEATURE]** Implement UpdateConfig RPC
  - Main config updates (ocserv.conf)
  - Per-user config updates
  - Per-group config updates
  - Backup/restore mechanism
  - Validation and rollback

- [ ] **[FEATURE]** Implement AgentStream RPC (bidirectional streaming)
  - Heartbeat with exponential backoff
  - Real-time event notifications
  - Command execution via stream
  - Metrics reporting

## 🟡 High Priority (This week - Phase 1: Core)

- [x] **[SETUP]** Create project directory structure
  - ✅ Completed: 2025-01-23
  - Commit: cc69c82
  - All directories created

- [x] **[SETUP]** Create go.mod with dependencies
  - ✅ Completed: 2025-01-23
  - Commit: cc69c82
  - gRPC v1.69.4, protobuf v1.36.3, zerolog v1.33.0

- [x] **[SETUP]** Create proto definitions
  - ✅ Completed: 2025-01-23
  - Commit: cc69c82
  - File: pkg/proto/agent/v1/agent.proto

- [x] **[SETUP]** Create Podman Compose configuration
  - ✅ Completed: 2025-01-23
  - Commit: cc69c82
  - Dev, test, build compose files created

- [x] **[SETUP]** Create Dockerfile, systemd service, README
  - ✅ Completed: 2025-01-23
  - Commit: cc69c82
  - Multi-stage Dockerfile, hardened systemd service, comprehensive README

- [x] **[FEATURE]** Implement internal/config package
  - ✅ Completed: 2025-01-23
  - Commit: a899a75
  - YAML loading, validation, env overrides, defaults

- [x] **[FEATURE]** Generate protobuf code
  - ✅ Completed: 2025-01-23
  - Via Podman Compose proto-gen service

- [x] **[FEATURE]** Implement gRPC server with mTLS
  - ✅ Completed: 2025-01-23
  - Commit: 110d823
  - TLS 1.3, client cert auth, interceptors

- [x] **[FEATURE]** Implement HealthCheck endpoint (Tier 1)
  - ✅ Completed: 2025-01-23
  - Commit: 110d823
  - Basic heartbeat working

- [x] **[FEATURE]** Create cmd/agent/main.go entrypoint
  - ✅ Completed: 2025-01-23
  - Commit: 110d823
  - Graceful shutdown with SIGTERM/SIGINT handling

## 🟡 High Priority (Phase 2 - Completed Tasks)

- [x] **[FEATURE]** Implement systemctl wrapper
  - ✅ Completed: 2025-01-23
  - Commit: 56da3c5
  - internal/ocserv/systemctl.go

- [x] **[FEATURE]** Implement occtl wrapper
  - ✅ Completed: 2025-01-23
  - Commit: 56da3c5
  - internal/ocserv/occtl.go

- [x] **[FEATURE]** Implement command validation and security
  - ✅ Completed: 2025-01-23
  - Commit: 56da3c5
  - internal/ocserv/manager.go

- [x] **[FEATURE]** Update ExecuteCommand RPC handler
  - ✅ Completed: 2025-01-23
  - Commit: 56da3c5
  - Full integration with ocserv manager

- [x] **[DOCS]** Update release notes for v0.1.0
  - ✅ Completed: 2025-01-23
  - All features, commits, and statistics updated

## 🟢 Medium Priority (Recently Completed)

- [x] **[FEATURE]** Implement config file reading (internal/ocserv/config.go)
  - ✅ Completed: 2025-10-23
  - Commit: cf0a6b2
  - Read ocserv.conf
  - Read config-per-user/*
  - Read config-per-group/*

- [x] **[RESEARCH]** Production occtl output examples
  - ✅ Completed: 2025-10-23
  - Commit: pending
  - Real output from production ocserv 1.3.0 server
  - All major commands: show users, status, sessions, iroutes, events
  - JSON and plain text formats
  - Complete documentation in test/fixtures/ocserv/occtl/README.md
  - Ready for OcctlManager enhancement implementation

## 🟢 Medium Priority (Testing & Polish)

- [x] **[TEST]** Add unit tests for config package
  - ✅ Completed: 2025-10-23
  - Commit: 83e3f05
  - Coverage: 97.1% (exceeds >80% target)
  - Files: config_test.go (347 lines), validation_test.go (579 lines)
  - Test fixtures: 4 YAML files (valid, minimal, invalid scenarios)
- [x] **[TEST]** Add unit tests for cert package
  - ✅ Completed: 2025-10-23
  - Commit: a6dee4c
  - Coverage: 77.6% (close to 80% target)
  - Files: generator_test.go (678 lines)
  - Certificate generation, PEM operations, fingerprints
- [x] **[TEST]** Add unit tests for ocserv/config.go
  - ✅ Completed: 2025-10-23
  - Commit: 36b4678
  - Coverage: 82-100% for all functions
  - Files: config_test.go (621 lines)
  - Test fixtures: 4 ocserv config files
- [ ] **[TEST]** Add unit tests for gRPC handlers
- [ ] **[TEST]** Add unit tests for remaining ocserv files (manager, occtl, systemctl)
- [ ] **[FEATURE]** Create certificate generation script (scripts/generate-certs.sh)
- [ ] **[TEST]** Test the agent with compose-build
- [ ] **[TEST]** Create test certificates for mTLS

## 🔵 Low Priority (Phase 3+)

- [ ] **[FEATURE]** Bidirectional streaming (AgentStream)
- [ ] **[FEATURE]** Heartbeat implementation with metrics
- [ ] **[FEATURE]** Log streaming (StreamLogs)
- [ ] **[FEATURE]** Config updates with backup (UpdateConfig)
- [ ] **[FEATURE]** HealthCheck Tier 2 (deep check)
- [ ] **[FEATURE]** HealthCheck Tier 3 (end-to-end test)
- [ ] **[FEATURE]** User management (ocpasswd wrapper)

## 📋 Code Review Needed

None yet

## 🐛 Known Issues

None yet

## 📊 Progress

- **Phase 1 Core: 9/9 (100%) ✅ COMPLETED!**
  - ✅ Project structure
  - ✅ Dependencies
  - ✅ Proto definitions
  - ✅ Compose infrastructure
  - ✅ Documentation
  - ✅ Config package
  - ✅ gRPC server
  - ✅ HealthCheck
  - ✅ Main entrypoint

- **Commits:**
  - cc69c82: Initial setup
  - a899a75: Config package
  - 110d823: gRPC server + HealthCheck + main
  - 56da3c5: Phase 2 ocserv integration ✅
  - cf0a6b2: Config file reading ✅
  - 6f2a59a: Compatibility analysis roadmap ✅
  - 9c4dcd6: Production occtl examples ✅
  - 0ab84c6: v0.1.0 ALPHA release ✅
  - d577619: All 11 missing occtl commands ✅
  - 66600a3: Phase 3 progress docs
  - 9c6942a: New fields and multiple sessions
  - b11bb9e: JSON parsing fix ✅
  - 778145b: v0.2.0 BETA release ✅
  - ee9fbe3: Build infrastructure (go.sum)
  - 4bc5b19: GitHub Actions workflows ✅
  - a6bfd55: Code formatting (gofmt) ✅
  - a25e925: README display fix ✅
  - 612e212: Contributing guide ✅
  - 22f38cc: Platform updates ✅
  - b4ac820: Security fix (golang.org/x/net) ✅
  - 07d02ed: v0.2.1 release notes ✅
  - 208021b: Certificate auto-generation ✅
  - 520a42b: Versioned archive packaging ✅
  - 2d50a1c: Sanitize sensitive data ✅
  - a710481: Fix Go toolchain issue ✅
  - 084a0b5: v0.3.0 release notes ✅
  - 68185df: Fix release workflow and Docker build ✅
  - 4b65e05: Add libprotobuf-dev for proto types ✅
  - ad75891: Remove CLAUDE_PROMPT.md from repo ✅
  - 903797d: Update TODO docs with v0.3.0 status ✅
  - 89897c1: Mark v0.3.0 as BETA pre-release ✅
  - 01ebe67: Create local testing scripts ✅
  - 597eb62: Update README with local testing section ✅
  - be3c5c0: Fix OSSF Scorecard permissions ✅
  - b8aeb6e: Fix security workflow paths-ignore ✅
  - 241c28b: Gosec SARIF fix + local security testing ✅
  - 5f0d2a7: Documentation updates and MIT license ✅
  - 18fd5c8: Binary installation path fix ✅
  - f6f077d: Configuration validation logging ✅
  - 09d3c50: Unified build pipeline script ✅
  - 8500b69: Documentation updates for unified pipeline ✅
  - 4a83924: Fix VERSION variable expansion ✅
  - 0161ffc: Fix command order for RAW binaries ✅
  - 7f0a18c: TODO updates ✅
  - 783984f: DONE.md and BACKLOG.md updates ✅
  - c0efd50: CURRENT.md updates ✅
  - cb1f848: gRPC reflection support ✅
  - 801b32d: gRPC testing guide and deployment scripts ✅
  - 3c2d96a: Remove hardcoded credentials ✅
  - 4fd990f: **Fix occtl JSON parsing (CRITICAL)** ✅
  - 8837ee6: Add OCCTL_COMMANDS.md reference ✅
  - 37310dc: Update compatibility + add security docs ✅
  - 83e3f05: Add comprehensive unit tests for internal/config (97.1% coverage) ✅
  - a6dee4c: Add unit tests for internal/cert (77.6% coverage) ✅
  - 36b4678: Add unit tests for internal/ocserv/config.go (82-100% coverage) ✅

- **Tests:**
  - internal/config: 97.1% coverage ✅
  - internal/cert: 77.6% coverage ✅
  - internal/ocserv/config.go: 82-100% coverage ✅
  - internal/ocserv (overall): 15.8% (other files pending)
  - Overall project: Moving from 0% toward >80% target
  - Target for v0.4.0: >80% overall coverage
- **Documentation:** 100% complete + 5 new comprehensive guides
- **Release notes:** v0.3.1 BETA completed, v0.4.0 in progress
- **Phase 1:** COMPLETED (100%) ✅
- **Phase 2:** COMPLETED (100%) ✅
- **Phase 3:** COMPLETED (100%) ✅ - occtl commands working
- **v0.2.1:** COMPLETED (100%) ✅ - CI/CD infrastructure
- **v0.3.0:** COMPLETED (100%) ✅ - Certificate auto-generation
- **v0.3.1:** COMPLETED (100%) ✅ - Critical bugfixes + Documentation
- **v0.4.0:** IN PROGRESS - Unit tests implementation (internal/config ✅ 97.1%)
- **Current:** v0.4.0 development - Unit test infrastructure established
- **Next Steps:** Unit tests for cert/grpc/ocserv packages, OSSF improvements
