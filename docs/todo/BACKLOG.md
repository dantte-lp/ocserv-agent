# Backlog - ocserv-agent

**NOTE:** For detailed ocserv 1.3.0 compatibility roadmap, see `OCSERV_COMPATIBILITY.md`

## Phase 2: ocserv Integration (Week 2) - ✅ COMPLETED

- [x] systemctl wrapper implementation
- [x] occtl command execution with validation (partial - 5/16 commands)
- [x] Config file reading (main, per-user, per-group)
- [x] Command validation and security (whitelist, sanitization)
- [ ] Backup/rollback for config changes (moved to Phase 3)

## Phase 3: Streaming & Full ocserv Integration (Week 3-4)

See `OCSERV_COMPATIBILITY.md` for complete breakdown.

**High Priority:**
- [x] Complete all 16 occtl commands (16/16 DONE!)
- [ ] ocpasswd wrapper for user management
- [ ] UpdateConfig RPC with backup/rollback
- [ ] Bidirectional streaming (AgentStream)
- [ ] Unit tests (>80% coverage)

**Medium Priority:**
- [ ] Log streaming (StreamLogs RPC)
- [ ] Heartbeat with exponential backoff
- [ ] HealthCheck Tier 2 & 3
- [ ] Enhanced metrics (Prometheus)
- [ ] Reconnection logic with circuit breaker
- [ ] ocserv-fw firewall integration

## Phase 4: Production Ready (Week 4)

- [x] OpenTelemetry integration (basic setup in config)
- [x] Error handling and retry logic (circuit breaker, exponential backoff in config)
- [ ] Unit tests (>80% coverage)
- [ ] Integration tests with mock ocserv
- [x] Complete documentation (README, CERTIFICATES, LOCAL_TESTING, workflows)
- [ ] Performance testing
- [x] Security audit (OSSF Scorecard, Gosec, Trivy all passing)

## v0.3.0 BETA Completed (2025-10-23)

**Infrastructure & Tooling:**
- [x] Certificate auto-generation (bootstrap mode)
- [x] CLI commands (gencert, help, version)
- [x] Versioned archive packaging
- [x] FreeBSD support (amd64, arm64)
- [x] SLSA Level 3 provenance
- [x] Local testing infrastructure (quick-check, test-local)
- [x] Unified build pipeline (scripts/build-all.sh)
- [x] Multi-platform builds (4 platforms)
- [x] Security scanning (gosec, govulncheck, trivy)
- [x] MIT License
- [x] Complete documentation

## Future Enhancements (Low Priority)

**See `OCSERV_COMPATIBILITY.md` for detailed planning**

- [ ] Virtual hosts support
- [ ] RADIUS/Kerberos monitoring
- [ ] Certificate management helpers
- [ ] ocserv-script documentation (custom hooks)
- [ ] Support for multiple control servers (failover)
- [ ] Automated backup scheduling
- [ ] Advanced ocserv 1.3.0 features:
  - Camouflage mode configuration
  - HTTP security headers management
  - Network namespace configuration
