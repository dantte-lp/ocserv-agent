# Current TODO - ocserv-agent

**Last Updated:** 2025-01-23
**Last Commit:** cc69c82 - chore(setup): initial project structure

## 🔴 Critical (Must do now)

- [ ] **[FEATURE]** Implement internal/config package for configuration loading
  - Priority: P0
  - Assigned: -
  - Deadline: Phase 1
  - Blockers: None
  - Notes: YAML loading, validation, environment variable override

- [ ] **[FEATURE]** Implement basic gRPC server with mTLS
  - Priority: P0
  - Assigned: -
  - Deadline: Phase 1
  - Blockers: internal/config must be done first
  - Notes: TLS 1.3, client cert authentication

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

- [ ] **[FEATURE]** Implement HealthCheck endpoint (Tier 1)
  - Priority: P1
  - Estimated: 2h
  - Dependencies: gRPC server

- [ ] **[FEATURE]** Create cmd/agent/main.go entrypoint
  - Priority: P1
  - Estimated: 3h
  - Dependencies: config, gRPC server
  - Notes: Graceful shutdown with SIGTERM/SIGINT handling

## 🟢 Medium Priority (Phase 1)

- [ ] **[TEST]** Add unit tests for config package
- [ ] **[TEST]** Add unit tests for gRPC handlers
- [ ] **[FEATURE]** Generate protobuf code (make compose-build with proto-gen)

## 🔵 Low Priority (Phase 2+)

- [ ] **[FEATURE]** Implement ocserv manager (systemctl wrapper)
- [ ] **[FEATURE]** Implement occtl command execution
- [ ] **[FEATURE]** Config file reading and management
- [ ] **[FEATURE]** Bidirectional streaming
- [ ] **[FEATURE]** Heartbeat implementation

## 📋 Code Review Needed

None yet

## 🐛 Known Issues

None yet

## 📊 Progress

- Phase 1 Core Setup: 5/9 (55%)
  - ✅ Project structure
  - ✅ Dependencies
  - ✅ Proto definitions
  - ✅ Compose infrastructure
  - ✅ Documentation
  - ⏳ Config package
  - ⏳ gRPC server
  - ⏳ HealthCheck
  - ⏳ Main entrypoint
- Tests: 0% coverage
- Documentation: 60% complete
- First commit: ✅ cc69c82
