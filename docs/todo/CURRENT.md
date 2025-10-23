# Current TODO - ocserv-agent

**Last Updated:** 2025-10-23
**Last Commit:** d577619 - feat(ocserv): implement all 11 missing occtl commands (Phase 3)

## 🎉 Phase 1: Core - COMPLETED!

All critical Phase 1 tasks done ✅

## 🎉 Phase 2: ocserv Integration - COMPLETED!

All critical Phase 2 tasks done ✅
- ✅ Systemctl wrapper (start, stop, restart, reload, status)
- ✅ Occtl wrapper (show users/status/stats, disconnect)
- ✅ Command validation and security (whitelist, sanitization, injection protection)
- ✅ ExecuteCommand RPC fully functional

## 🎉 Phase 3: Advanced occtl - IN PROGRESS!

**Completed:**
- ✅ Complete type definitions (occtl_types.go)
- ✅ All 11 missing occtl commands implemented
- ✅ JSON parsing with production-tested types
- ✅ Full occtl compatibility (16/16 commands)

**Remaining:**
- [ ] ShowEvents() streaming support
- [ ] ocpasswd wrapper
- [ ] UpdateConfig RPC
- [ ] Unit tests (>80% coverage)

## 🔴 Critical (Next Steps - Phase 3 Continued)

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
  - 1a97fe9: TODO update
  - a899a75: Config package
  - 110d823: gRPC server + HealthCheck + main
  - f6b7436: Phase 1 completion docs
  - 678b766: Exclude .claude
  - 55bac55: Release notes update
  - 56da3c5: Phase 2 ocserv integration ✅
  - 4c640d0: Phase 2 TODO update
  - cd338e6: Phase 2 release notes update
  - cf0a6b2: Config file reading ✅
  - 4777464: Docs update with commit hash
  - 2be3eaf: ocserv 1.3.0 reference config
  - 6f2a59a: Compatibility analysis roadmap ✅
  - 7af7cee: Config consolidation
  - 9c4dcd6: Production occtl examples ✅
  - 0ab84c6: v0.1.0 ALPHA release ✅
  - d577619: All 11 missing occtl commands ✅

- **Tests:** 0% coverage (tests pending Phase 3.5)
- **Documentation:** 90% complete
- **Release notes:** v0.1.0 ALPHA released ✅
- **Phase 1:** COMPLETED (100%) ✅
- **Phase 2:** COMPLETED (100%) ✅
- **Phase 3:** IN PROGRESS (60%)
  - ✅ occtl commands (16/16)
  - ⏳ ShowEvents streaming
  - ⏳ ocpasswd wrapper
  - ⏳ UpdateConfig RPC
  - ⏳ Unit tests
- **Next Milestone:** v0.2.0 BETA (Phase 3 completion)
