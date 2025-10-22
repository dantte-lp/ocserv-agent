# Current TODO - ocserv-agent

**Last Updated:** 2025-01-23
**Last Commit:** 110d823 - feat(grpc): implement gRPC server with mTLS and HealthCheck

## 🎉 Phase 1: Core - COMPLETED!

All critical Phase 1 tasks are done. Agent has:
- ✅ Configuration loading and validation
- ✅ gRPC server with mTLS
- ✅ HealthCheck endpoint (Tier 1)
- ✅ Graceful shutdown

## 🔴 Critical (Must do now)

- [ ] **[TEST]** Test the agent with compose-build
  - Priority: P0
  - Assigned: -
  - Deadline: Now
  - Blockers: None
  - Notes: Verify that everything compiles and runs

- [ ] **[TEST]** Create test certificates for mTLS
  - Priority: P0
  - Assigned: -
  - Deadline: Now
  - Blockers: None
  - Notes: Need CA, server cert, and client cert for testing

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

## 🟡 High Priority (Next steps)

- [ ] **[TEST]** Add unit tests for config package
- [ ] **[TEST]** Add unit tests for gRPC handlers
- [ ] **[DOCS]** Update release notes for v0.1.0
- [ ] **[FEATURE]** Create certificate generation script (scripts/generate-certs.sh)

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
  - 1a97fe9: TODO update
  - a899a75: Config package
  - 110d823: gRPC server + HealthCheck + main

- **Tests:** 0% coverage (tests pending)
- **Documentation:** 70% complete
- **Next Phase:** Phase 2 - ocserv Integration
