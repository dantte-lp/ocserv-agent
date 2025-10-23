# ocserv-agent Development Roadmap

**Last Updated:** 2025-10-23
**Current Version:** v0.5.0 BETA
**Status:** Active Development

---

## 🎯 Project Vision

Build a production-ready, secure, and well-tested agent for managing OpenConnect VPN servers (ocserv) via gRPC API with comprehensive monitoring, configuration management, and security features.

## 📊 Project Phases

### Phase 1: Foundation ✅ COMPLETED (v0.1.0 - v0.3.1)

**Core Infrastructure:**
- ✅ Project structure and build system
- ✅ gRPC API with mTLS authentication
- ✅ Configuration management (YAML + env overrides)
- ✅ Ocserv integration (systemctl + occtl)
- ✅ All 16 occtl commands implemented
- ✅ Certificate auto-generation
- ✅ CI/CD pipelines (GitHub Actions)
- ✅ SLSA Level 3 provenance
- ✅ Multi-platform builds (Linux/FreeBSD, amd64/arm64)

**Status:** Production-tested with real VPN users ✅

---

## 🚀 Recent Releases

### v0.5.0 BETA: Test Coverage Expansion & Security Fixes ✅ (October 2025)

**Achievements:**
- ✅ **CRITICAL:** Fixed 4 command injection vulnerabilities (29 test cases)
- ✅ internal/grpc: 0% → **87.6%** coverage (exceeded >80% target!)
- ✅ internal/ocserv: 15.8% → 23.1% coverage
- ✅ Overall internal: ~40% → **51.2%** (+11.2%)
- ✅ 1,600+ new lines of test code
- ✅ Test infrastructure: TLS certificate helpers, security validation
- ✅ validateArguments: 100% coverage (security-first testing)

**Security Fixes:**
- ✅ Backtick command substitution (HIGH severity)
- ✅ Escaped metacharacter injection (MEDIUM severity)
- ✅ Newline injection (MEDIUM severity)
- ✅ Control character injection (LOW severity)

**Test Infrastructure:**
- ✅ TLS certificate helper (createTestCerts)
- ✅ Mock stream implementations
- ✅ Security validation test suite
- ✅ Interceptor testing (100% coverage)

### v0.4.0 BETA: Test Foundation & DevOps Improvements ✅ (October 2025)

**What's New:**

**Unit Tests:**
- ✅ internal/config: 97.1% coverage (config loading, validation, env overrides)
- ✅ internal/cert: 77.6% coverage (certificate generation, PEM operations)
- ✅ internal/ocserv/config.go: 82-100% coverage (config file parsing)
- ✅ Test fixtures infrastructure (8 fixture files)
- ✅ 2,225 lines of test code

**DevOps Improvements:**
- ✅ Automatic code formatting (scripts/quick-check.sh)
- ✅ Git hooks (pre-commit: auto-format, pre-push: checks)
- ✅ One-time setup script (scripts/install-hooks.sh)
- ✅ Eliminates CI formatting failures

**Security:**
- ✅ Branch protection with admin bypass
- ✅ Required PR reviews (1 approval)
- ✅ CI path filtering (skip heavy jobs for docs-only changes)

---

## 🔮 Upcoming Releases

### v0.6.0: Integration Tests & Coverage Expansion 🚧 IN PROGRESS (Target: January 2026)

**Status:** 40% Complete (6/15 tasks) ⚡

**Integration Tests - IN PROGRESS:**
- ✅ **Phase 1: Infrastructure Setup** [3/3] COMPLETE!
  - ✅ Ansible environment in podman-compose (v0.3.0)
  - ✅ Ansible playbooks for remote deployment
  - ✅ Mock ocserv Unix socket server (900+ lines, 14 fixtures)

- ✅ **Phase 2: Occtl Integration Tests** [3/4] 75% COMPLETE!
  - ✅ Task 2.1: Test infrastructure (10 tests) - mock helpers, fixtures, utilities
  - ✅ Task 2.2: ShowUsers and basic commands (24 tests) - ShowUsers(5), ShowStatus/Stats(7), errors(13)
  - ✅ Task 2.3: User management commands (30 tests) - ShowUser/ID(9), Disconnect(11), edge cases(10)
  - ⬜ Task 2.4: IP management commands (pending) - ShowIPBans, UnbanIP, Reload

- ⬜ **Phase 3: Systemctl Integration Tests** [0/3]
- ⬜ **Phase 4: gRPC End-to-End Tests** [0/3]
- ⬜ **Phase 5: Remote Server Testing** [0/2]

**Current Achievements:**
- ✅ **64 integration tests** (10 + 24 + 30)
- ✅ **~70% coverage** for occtl.go (target: 75-80%)
- ✅ Mock ocserv running in podman-compose
- ✅ Comprehensive edge cases: Unicode, special chars, long strings, concurrent operations
- ✅ Ansible automation tested on production server

**Coverage Goal:** 51.2% → 75-80% overall

**OSSF Scorecard Improvements (Target: 7.5+/10):**

**Phase 1: Quick Wins**
- [x] Branch protection rules (require PR, dismiss stale reviews) ✅ v0.4.0
- [ ] Restrict GitHub workflow token permissions
- [ ] Create .github/CODEOWNERS
- [ ] Setup GPG commit signing
- **Impact:** +2.0 points (5.9 → 7.9)

**Phase 2: Dependency Pinning**
- [ ] Pin all GitHub Actions to SHA hashes (49+ dependencies)
- [ ] Pin Docker base images to digests
- [ ] Automate pinning with Dependabot
- **Impact:** +1.0 point (Pinned-Dependencies: 0 → 10)

**Security Features:**
- [ ] Rate limiting for gRPC API
- [ ] Audit logging for sensitive operations
- [ ] Security scanning in CI (gosec, trivy)
- [ ] Vulnerability management process

### v0.7.0: Advanced Features (Target: February 2026)

**Configuration Management:**
- [ ] UpdateConfig RPC with backup/rollback
  - Main config updates (ocserv.conf)
  - Per-user config updates
  - Per-group config updates
  - Atomic configuration changes
  - Rollback on failure
- [ ] Configuration validation API
- [ ] Configuration diff/compare
- [ ] Configuration templates

**User Management:**
- [ ] ocpasswd wrapper implementation
  - User add/delete/lock/unlock
  - Password hashing (SHA-512/MD5)
  - Group assignment
  - Batch operations
- [ ] User lifecycle management
- [ ] Password policy enforcement

**Streaming:**
- [ ] ShowEvents() streaming (ServerStream RPC)
  - Real-time event notifications
  - Filtered event streams
  - Event history replay
- [ ] StreamLogs RPC implementation
  - Real-time log streaming
  - Log filtering and search
  - Multiple log sources

### v0.8.0: Monitoring & Observability (Target: March 2026)

**Metrics & Monitoring:**
- [ ] Prometheus metrics export
  - Connection metrics
  - Bandwidth usage
  - Server health
  - Command execution stats
- [ ] Grafana dashboards
- [ ] Alerting rules
- [ ] Custom metric collectors

**Telemetry:**
- [ ] OpenTelemetry tracing
  - Request tracing
  - Performance profiling
  - Distributed tracing
- [ ] Health check improvements
  - Tier 2: Deep health checks
  - Tier 3: End-to-end tests
  - Custom health checks

**Logging:**
- [ ] Structured logging enhancements
- [ ] Log aggregation support
- [ ] Log retention policies
- [ ] Debug mode with verbose logging

### v0.9.0: Performance & Scalability (Target: April 2026)

**Performance:**
- [ ] Connection pooling for occtl
- [ ] Caching layer for frequently-accessed data
- [ ] Request batching
- [ ] Concurrent command execution
- [ ] Resource usage optimization

**Scalability:**
- [ ] Support for multiple ocserv instances
- [ ] Load balancing
- [ ] High availability mode
- [ ] Clustering support

**Benchmarks:**
- [ ] Performance benchmarks
- [ ] Load testing suite
- [ ] Stress testing
- [ ] Performance regression tests

### v1.0.0: Production Release (Target: May 2026)

**Requirements for v1.0.0:**
- ✅ >80% test coverage
- ✅ OSSF Scorecard >7.5/10
- ✅ Production-tested for 6+ months
- ✅ Comprehensive documentation
- ✅ Security audit
- ✅ Performance benchmarks
- ✅ Zero critical bugs
- ✅ Stable API (backward compatibility guaranteed)

**Final Tasks:**
- [ ] Security audit by external team
- [ ] Performance optimization
- [ ] Documentation review and polish
- [ ] Migration guides
- [ ] Production deployment playbooks
- [ ] SLA commitments
- [ ] Long-term support plan

---

## 📋 Backlog (Future Considerations)

### Features Under Consideration

**API Enhancements:**
- [ ] REST API alongside gRPC
- [ ] GraphQL API
- [ ] WebSocket support for real-time updates
- [ ] API versioning strategy

**Management Features:**
- [ ] Web UI for management
- [ ] CLI tool for administration
- [ ] Backup/restore automation
- [ ] Configuration migration tools
- [ ] Bulk user import/export

**Integration:**
- [ ] LDAP/Active Directory integration
- [ ] RADIUS authentication
- [ ] OAuth2/OIDC support
- [ ] External CA integration
- [ ] Kubernetes operator

**Monitoring:**
- [ ] Datadog integration
- [ ] New Relic integration
- [ ] CloudWatch integration
- [ ] Custom webhook notifications

**Documentation:**
- [ ] Video tutorials
- [ ] Interactive API documentation
- [ ] Architecture decision records (ADRs)
- [ ] Case studies and best practices

---

## 🎓 Learning & Improvement

### Continuous Improvement Areas

**Code Quality:**
- Maintain >80% test coverage
- Regular dependency updates
- Code review culture
- Linting and static analysis

**Security:**
- Regular security audits
- Vulnerability scanning
- Dependency security updates
- Security training

**Documentation:**
- Keep docs up-to-date
- User feedback incorporation
- API documentation generation
- Example code maintenance

**Community:**
- Issue triage and response
- PR reviews and merging
- Community feedback
- Open source best practices

---

## 📈 Success Metrics

### Key Performance Indicators (KPIs)

**Quality Metrics:**
- Test coverage: >80% ✅ (Target for v0.5.0)
- OSSF Scorecard: >7.5/10 (Target for v0.6.0)
- Bug resolution time: <7 days
- Zero critical security vulnerabilities

**Community Metrics:**
- GitHub stars
- Contributors
- Issue response time: <48 hours
- Documentation quality score

**Production Metrics:**
- Uptime: >99.9%
- API response time: <100ms p95
- Error rate: <0.1%
- Resource usage: <50MB RAM

---

## 🤝 Contributing

We welcome contributions! See:
- [CONTRIBUTING.md](.github/CONTRIBUTING.md) - Development guidelines
- [TODO Management](docs/todo/CURRENT.md) - Current priorities
- [Issues](https://github.com/dantte-lp/ocserv-agent/issues) - Bug reports and feature requests

---

## 📚 References

### Documentation
- [README.md](README.md) - Project overview
- [docs/releases/](docs/releases/) - Release notes
- [docs/todo/CURRENT.md](docs/todo/CURRENT.md) - Current development status
- [SECURITY.md](SECURITY.md) - Security policy

### External Resources
- [ocserv Documentation](https://ocserv.gitlab.io/www/)
- [gRPC Documentation](https://grpc.io/docs/)
- [OSSF Scorecard](https://github.com/ossf/scorecard)
- [SLSA Framework](https://slsa.dev/)

---

**Note:** This roadmap is subject to change based on user feedback, community needs, and project priorities. Dates are approximate targets, not commitments.
