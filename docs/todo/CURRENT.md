# Current TODO - ocserv-agent

**Last Updated:** 2025-10-23
**Current Version:** v0.5.0 BETA
**Status:** Planning v0.6.0 (Target: January 2026)

---

## ✅ v0.5.0 BETA - COMPLETED! (October 2025)

**Released:** [v0.5.0](https://github.com/dantte-lp/ocserv-agent/releases/tag/v0.5.0)

**Key Achievements:**
- ✅ **CRITICAL:** Fixed 4 command injection vulnerabilities (29 test cases)
- ✅ internal/grpc: 0% → **87.6%** coverage (exceeded >80% target!)
- ✅ Overall coverage: ~40% → **51.2%** (+11.2%)
- ✅ 1,600+ new lines of test code
- ✅ validateArguments: 100% coverage

**See:** [Release Notes v0.5.0](../releases/v0.5.0.md) for full details.

---

## 🚀 v0.6.0: Security Hardening & Integration Tests (Target: January 2026)

### Integration Tests (HIGH PRIORITY) - IN PROGRESS

**📋 Detailed Plan:** [INTEGRATION_TESTS_PLAN.md](INTEGRATION_TESTS_PLAN.md) (15 tasks, ~12 hours)

**Progress:** 7/15 tasks (46.7%) ⚡ **MAJOR UPDATE!**
- Phase 1: Infrastructure Setup [3/3] ✅✅✅ **COMPLETE!**
- Phase 2: Occtl Integration Tests [4/4] ✅✅✅✅ **COMPLETE!** 🎉
  - ✅ Task 2.1: Test infrastructure (10 tests)
  - ✅ Task 2.2: ShowUsers and basic commands (24 tests)
  - ✅ Task 2.3: User management commands (30 tests)
  - ✅ Task 2.4: IP management commands (18 tests)
- Phase 3: Systemctl Integration Tests [0/3]
- Phase 4: gRPC End-to-End Tests [0/3]
- Phase 5: Remote Server Testing [0/2]

**Current Status:**
- ✅ **64 integration tests** created (10 + 24 + 30)
- ✅ **Coverage:** ~70% for occtl.go (target: 75-80% by end of Phase 2)
- ✅ **Test files:** 7 new integration test files
- ✅ **Mock ocserv:** Running in podman-compose with 14 fixtures
- ✅ No blockers!

**Recent Achievements (2025-10-23):**
- ✅ **Phase 2 COMPLETE!** All occtl commands tested (82 tests)
- ✅ Task 2.4 completed: 18 tests for ShowIPBans, ShowIPBanPoints, UnbanIP, Reload
- ✅ New fixtures: IP bans, ban points, unban, reload
- ✅ Fixed compilation errors in previous test files
- ✅ Edge cases: special chars, Unicode, long strings, concurrent operations
- ✅ Test infrastructure: mock socket helpers, fixture validation, test utilities

**Coverage progression:**
- v0.5.0: 51.2% overall, 23.1% internal/ocserv
- v0.6.0 (current): ~90% occtl.go, 82 integration tests ✅

**Remote Server (195.238.126.25):**
- Configuration: Use `.env` file (see `.env.example` for RFC 5737 template)
- Current setup: OracleLinux 9.6 + ocserv 1.3 (active) + 3 active VPN users
- Agent: v0.3.0-24-groutes (installed, service inactive)
- **CRITICAL:** Do NOT break existing VPN service
- **Status:** ✅ Verified via Ansible playbook (2025-10-23)

### OSSF Scorecard Improvements (HIGH PRIORITY)

**Current Score:** 5.9/10 | **Target:** 7.5+/10

**Phase 1: Quick Wins (+2.0 points)**
- [x] Branch protection rules ✅ (v0.4.0)
- [ ] Restrict GitHub workflow token permissions
  - Set minimal permissions per workflow
  - Explicit permissions for each job
  - **Impact:** Token-Permissions: 0 → 10
- [ ] Create `.github/CODEOWNERS`
  - Define code owners
  - Automatic review requests
- [ ] Setup GPG commit signing
  - Generate GPG key
  - Configure git signing
  - Sign all commits going forward

**Phase 2: Dependency Pinning (+1.0 point)**
- [ ] Pin all GitHub Actions to SHA hashes
  - 49+ action dependencies
  - 22 unique actions
  - **Impact:** Pinned-Dependencies: 0 → 10
- [ ] Pin Docker base images to digests
  - `golang:1.25-alpine@sha256:...`
  - Update all compose files

### Security Features (MEDIUM PRIORITY)

- [ ] Rate limiting for gRPC API
- [ ] Audit logging for sensitive operations
- [ ] Security scanning in CI (gosec, trivy)
- [ ] Vulnerability management process

---

## 🔮 v0.7.0 Planning (Target: February 2026)

**See:** [ROADMAP.md](../../ROADMAP.md) for detailed v0.7.0+ plans.

**Key Features:**
- [ ] UpdateConfig RPC with backup/rollback
- [ ] ocpasswd wrapper (user management)
- [ ] ShowEvents() streaming (ServerStream RPC)
- [ ] StreamLogs RPC implementation

---

## 📊 Current Metrics

### Test Coverage
- **internal/cert:** 77.6% ✅
- **internal/config:** 97.1% ✅
- **internal/grpc:** 87.6% ✅ (was 0%, major achievement!)
- **internal/ocserv:** 23.1% 🔴 (manager 100%, occtl/systemctl need integration tests)
- **Total (internal):** 51.2% 🟡 (target: >80%)

### Security
- **OSSF Scorecard:** 5.9/10 (target: 7.5+/10)
- **Vulnerabilities:** 0 critical ✅
- **Command injection protection:** 100% coverage ✅

### Documentation
- **Release notes:** 5 versions documented
- **User guides:** 8 comprehensive docs
- **Test coverage:** 3,800+ lines of test code

---

## 📚 Related Documentation

- **[ROADMAP.md](../../ROADMAP.md)** - Long-term project roadmap (v0.5.0-v1.0.0)
- **[Release Notes](../releases/)** - Detailed release history
- **[OCSERV_COMPATIBILITY.md](OCSERV_COMPATIBILITY.md)** - ocserv feature coverage
- **[CONTRIBUTING.md](../../.github/CONTRIBUTING.md)** - Development guidelines

---

**Note:** This document tracks current and upcoming work. For completed work, see release notes. For long-term plans, see ROADMAP.md.
