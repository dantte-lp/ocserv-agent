# Current TODO - ocserv-agent

**Last Updated:** 2025-01-23 (auto-generated)

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
  - All directories created

- [x] **[SETUP]** Create go.mod with dependencies
  - ✅ Completed: 2025-01-23
  - gRPC v1.69.4, protobuf v1.36.3, zerolog v1.33.0

- [x] **[SETUP]** Create proto definitions
  - ✅ Completed: 2025-01-23
  - File: pkg/proto/agent/v1/agent.proto

- [x] **[SETUP]** Create Podman Compose configuration
  - ✅ Completed: 2025-01-23
  - Dev, test, build compose files created

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

- [ ] **[DOCS]** Complete README.md with usage examples
- [ ] **[INFRA]** Create systemd service file
- [ ] **[INFRA]** Complete Dockerfile multi-stage build
- [ ] **[TEST]** Add unit tests for config package
- [ ] **[TEST]** Add unit tests for gRPC handlers

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

- Phase 1 Core Setup: 6/10 (60%)
- Tests: 0% coverage
- Documentation: 30% complete
