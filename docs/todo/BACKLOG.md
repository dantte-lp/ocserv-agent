# Backlog - ocserv-agent

## Phase 2: ocserv Integration (Week 2)

- [ ] systemctl wrapper implementation
- [ ] occtl command execution with validation
- [ ] Config file reading (main, per-user, per-group)
- [ ] Command validation and security (whitelist, sanitization)
- [ ] Backup/rollback for config changes

## Phase 3: Streaming (Week 3)

- [ ] Bidirectional streaming implementation
- [ ] Heartbeat with exponential backoff
- [ ] Log streaming (tail -f mode)
- [ ] Reconnection logic with circuit breaker
- [ ] Metrics collection and reporting

## Phase 4: Production Ready (Week 4)

- [ ] OpenTelemetry integration (traces, metrics)
- [ ] Error handling and retry logic
- [ ] Unit tests (>80% coverage)
- [ ] Integration tests with mock ocserv
- [ ] Complete documentation
- [ ] Performance testing
- [ ] Security audit

## Future Enhancements

### ocserv 1.3.0 Utilities Support
- [ ] **ocpasswd wrapper** - User password management
  - Add password entries (username:groups:hash)
  - Update passwords
  - Delete users
  - Lock/unlock accounts
  - Integration with UpdateConfig RPC
- [ ] **ocserv-genkey wrapper** - Certificate/key generation
  - Generate server keys
  - Generate client certificates
  - CA management

### Additional Features
- [ ] Support for multiple control servers (failover)
- [ ] Configuration hot-reload on SIGHUP
- [ ] Prometheus metrics endpoint (/metrics)
- [ ] Rate limiting for gRPC calls
- [ ] Certificate rotation with zero downtime
- [ ] Automated backup scheduling
- [ ] Config validation before apply
- [ ] Rollback on failed config updates
- [ ] Support for ocserv 1.3.0 new features:
  - Camouflage mode configuration
  - HTTP security headers
  - Ban system management
  - Network namespace configuration
