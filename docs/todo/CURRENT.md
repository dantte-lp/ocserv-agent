# Current TODO - ocserv-agent

**Last Updated:** 2025-10-23
**Last Commit:** 0161ffc - fix(build): fix command order to ensure RAW binaries are created
**Status:** v0.3.0 BETA - Unified build pipeline working, all 12 artifacts created successfully

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
- [ ] Unit tests (>80% coverage)

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

## 🔴 Critical (Next Steps - v0.4.0)

### Based on ocserv 1.3.0 Compatibility Analysis

See: `docs/todo/OCSERV_COMPATIBILITY.md` for complete roadmap

**High Priority:**
- [x] **[FEATURE]** Complete missing occtl commands (16/16 done!)
  - ✅ show user [NAME], show id [ID]
  - ✅ show sessions (all/valid), show session [SID]
  - ✅ show ip bans, show ip ban points, unban ip
  - ✅ show iroutes
  - ✅ reload
  - [ ] show events (real-time streaming) - needs special implementation

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

- [ ] **[TEST]** Add unit tests for config package
- [ ] **[TEST]** Add unit tests for gRPC handlers
- [ ] **[TEST]** Add unit tests for ocserv manager
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

- **Tests:** 0% coverage (tests planned for v0.4.0+)
- **Documentation:** 100% complete
- **Release notes:** v0.3.0 BETA released
- **Phase 1:** COMPLETED (100%) ✅
- **Phase 2:** COMPLETED (100%) ✅
- **Phase 3:** COMPLETED (100%) ✅ - All occtl commands
- **v0.2.1:** COMPLETED (100%) ✅ - CI/CD infrastructure
- **v0.3.0:** COMPLETED (100%) ✅ - Certificate auto-generation
- **Current:** v0.3.0 BETA published on GitHub - ready for production testing
- **Next Phase:** v0.4.0 - Streaming, ocpasswd, UpdateConfig, Unit tests
