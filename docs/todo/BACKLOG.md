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
- [ ] Complete all 16 occtl commands (currently 5/16)
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

- [ ] OpenTelemetry integration (traces, metrics)
- [ ] Error handling and retry logic
- [ ] Unit tests (>80% coverage)
- [ ] Integration tests with mock ocserv
- [ ] Complete documentation
- [ ] Performance testing
- [ ] Security audit

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
