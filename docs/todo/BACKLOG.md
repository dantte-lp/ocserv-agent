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

- [ ] Support for multiple control servers
- [ ] Configuration hot-reload on SIGHUP
- [ ] Prometheus metrics endpoint
- [ ] Rate limiting for gRPC calls
- [ ] User management (ocpasswd wrapper)
- [ ] Certificate rotation
- [ ] Automated backup scheduling
