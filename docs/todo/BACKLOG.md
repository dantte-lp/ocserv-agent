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
- [x] occtl commands with JSON mode (13/16 working!)
  - ✅ 13 fully working commands
  - ⚠️ 3 with upstream occtl bugs (iroutes, sessions)
  - [ ] 2 not implemented (show events, stop now)
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

## v0.3.1 BETA Completed (2025-10-23)

**Critical Bugfixes:**
- [x] Fixed occtl JSON parsing (user count was showing 0)
- [x] Switched to `occtl -j` mode with 40+ fields per user
- [x] Fixed Routes field polymorphism (string vs []string)
- [x] Production-tested with 3 real VPN users

**Security Improvements:**
- [x] SECURITY.md vulnerability disclosure policy
- [x] Removed hardcoded credentials from repository
- [x] Sanitized all deployment scripts
- [x] OSSF Scorecard: 4.9/10 → 5.9/10 (+1.0)

**Documentation (5 new guides):**
- [x] docs/OCCTL_COMMANDS.md - Complete command reference
- [x] docs/GRPC_TESTING.md - gRPC testing procedures
- [x] docs/OSSF_SCORECARD_IMPROVEMENTS.md - Security roadmap
- [x] Updated OCSERV_COMPATIBILITY.md with production results
- [x] docs/releases/v0.3.1.md - Full release notes

**Features:**
- [x] gRPC reflection support for service discovery
- [x] Production deployment scripts (deploy-and-test.sh, test-grpc.sh)
- [x] Comprehensive testing of all occtl commands

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

## v0.4.0 Planning - OSSF Scorecard & Security Improvements

**See `docs/OSSF_SCORECARD_IMPROVEMENTS.md` for detailed plan**

**Phase 1 - Quick Wins (Target: 6.5/10):**
- [ ] Setup branch protection rules (require PR, code review)
- [ ] Restrict GitHub workflow token permissions
- [ ] Setup GPG commit signing
- [ ] Create .github/CODEOWNERS file

**Phase 2 - Dependency Pinning (Target: 7.5/10):**
- [ ] Pin all 22 GitHub Actions to SHA hashes
- [ ] Pin Docker base images to digests
- [ ] Update all compose files

**Phase 3 - Signing & Provenance (Target: 8.0/10):**
- [ ] GPG sign all release artifacts
- [ ] Cosign for container images
- [ ] Improve release signing consistency

## Future Enhancements (Low Priority)

**See `OCSERV_COMPATIBILITY.md` for detailed planning**

- [ ] Virtual hosts support
- [ ] RADIUS/Kerberos monitoring
- [ ] Certificate management helpers
- [ ] ocserv-script documentation (custom hooks)
- [ ] Support for multiple control servers (failover)
- [ ] Automated backup scheduling
- [ ] Fuzzing integration (OSS-Fuzz, go-fuzz)
- [ ] CII Best Practices certification
- [ ] Advanced ocserv 1.3.0 features:
  - Camouflage mode configuration
  - HTTP security headers management
  - Network namespace configuration
