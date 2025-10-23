# Integration Tests Implementation Plan

**Created:** 2025-10-23
**Target Release:** v0.6.0
**Current Coverage:** 51.2% → **Target:** 75-80%
**Estimated Time:** ~12 hours (15 tasks)

---

## 🚨 BLOCKERS

Blockers are tasks that prevent other tasks from starting. They must be resolved first.

**Current Blockers:**
- None! All blockers resolved ✅

**Resolved Blockers:**
- ✅ **BLOCKER #1:** Ansible environment setup - **RESOLVED** (2025-10-23)
  - Blocked: Tasks 1.2, 5.1, 5.2
  - Resolution time: 30 min (as estimated)
  - Commit: 97e05aa

---

## 📊 Progress Tracking

**Phase 1: Infrastructure Setup** [3/3] ✅✅✅ **COMPLETE!**
**Phase 2: Occtl Integration Tests** [0/4] ⬜⬜⬜⬜
**Phase 3: Systemctl Integration Tests** [0/3] ⬜⬜⬜
**Phase 4: gRPC End-to-End Tests** [0/3] ⬜⬜⬜
**Phase 5: Remote Server Testing** [0/2] ⬜⬜

**Total Progress:** 3/15 (20.0%)

---

## 🎯 Phase 1: Infrastructure Setup (3 tasks)

### ✅ Task 1.1: Create Ansible environment in podman-compose
**Status:** ✅ COMPLETED (2025-10-23) | **Priority:** HIGH | **Time:** 30 min
**BLOCKER #1** - RESOLVED ✅ | **Commit:** 97e05aa

**Objectives:**
- Create `deploy/compose/ansible.yml` with Python 3.14-slim-trixie
- Install Poetry 2.2 (official installer)
- Install Ansible 12.1.0 + ansible-core 2.19.3
- Setup volume mounts for playbooks and inventory
- Verify installation works

**Files to create:**
- `deploy/compose/ansible.yml`
- `deploy/ansible/pyproject.toml` (Poetry config)
- `deploy/ansible/ansible.cfg`

**Acceptance criteria:**
- ✅ `make compose-ansible` starts container
- ✅ `ansible --version` shows 12.1.0
- ✅ Poetry environment active
- ✅ .env file for credentials (not in git)
- ✅ RFC 5737 examples in documentation

**Dependencies:** None (this is a blocker for others)

**Results:**
- ✅ All acceptance criteria met
- ✅ Ansible 12.1.0 + ansible-core 2.19.3 installed
- ✅ Security: .env in .gitignore, RFC examples used
- ✅ Makefile targets: `make compose-ansible`, `make ansible-shell`
- ✅ Comprehensive README with safety measures

---

### ✅ Task 1.2: Create Ansible playbooks for remote server setup
**Status:** ✅ COMPLETED (2025-10-23) | **Priority:** HIGH | **Time:** 45 min
**Was blocked by:** Task 1.1 (BLOCKER #1) - RESOLVED ✅ | **Commits:** 8a6a96e, f797893

**Objectives:**
- Setup test user with certificate authentication (security best practice)
- Install dependencies if needed (ocserv already installed)
- Configure firewall rules for testing
- Setup logging for test runs

**Files to create:**
- `deploy/ansible/inventory/production.yml` (uses ${REMOTE_HOST} from .env)
- `deploy/ansible/playbooks/setup-test-user.yml`
- `deploy/ansible/playbooks/verify-ocserv.yml`
- `deploy/ansible/playbooks/deploy-agent.yml`
- `deploy/ansible/roles/test-user/tasks/main.yml`
- `.env.example` (RFC 5737 example: 192.0.2.1)

**Server details:**
- Host: Configured via `.env` file (REMOTE_HOST, see `.env.example`)
- User: Configured via `.env` file (REMOTE_USER, REMOTE_PASSWORD)
- Current: ocserv 1.3, ocserv-agent v0.3.0-24-groutes
- **CRITICAL:** Do NOT break existing setup!

**Acceptance criteria:**
- ✅ Playbook creates test user with cert auth
- ✅ Test user has sudo privileges
- ✅ Can SSH to server as test user
- ✅ Existing ocserv still works
- ✅ Deployment playbook with backup/rollback
- ✅ Verify playbook for ocserv status

**Dependencies:** Task 1.1 (completed)

**Results:**
- ✅ All playbooks created with comprehensive safety measures
- ✅ Inventory with .env integration (no secrets in git)
- ✅ test-user role: SSH cert auth (ed25519) + sudo
- ✅ Confirmation prompts before destructive actions
- ✅ Backup procedures: timestamped backups before deploy
- ✅ Rollback playbook: restore from backup
- ✅ VPN users monitoring: before/after comparison
- ✅ 4 playbooks: setup-test-user, verify-ocserv, deploy-agent, rollback-agent

**Testing Results (2025-10-23):**
- ✅ Ansible container starts successfully
- ✅ Python 3.14.0 + Poetry 2.2.0 + Ansible 12.1.0
- ✅ verify-ocserv.yml tested on production server
- ✅ Server verified: OracleLinux 9.6, ocserv 1.3 active
- ✅ Current agent: v0.3.0-24-groutes (inactive service)
- ✅ 3 active VPN users confirmed
- ✅ Ready for deployment: Yes

**Issues Fixed:**
- Fixed missing system dependencies (curl, git, openssh-client, sshpass)
- Removed ansible-lint due to dependency conflict with Python 3.14
- Added poetry.lock for reproducible builds

---

### ✅ Task 1.3: Create mock ocserv Unix socket server
**Status:** ✅ COMPLETED (2025-10-23) | **Priority:** HIGH | **Time:** 1 hour | **Commit:** 9bb62c5

**Objectives:**
- Create Go program simulating occtl Unix socket
- Use existing test fixtures from `test/fixtures/ocserv/occtl/`
- Support all commands: show users, status, stats, disconnect, etc.
- Run in test environment (no real ocserv needed)

**Files created:**
- ✅ `test/mock-ocserv/main.go` (server setup, signal handling)
- ✅ `test/mock-ocserv/handler.go` (connection handling, command execution)
- ✅ `test/mock-ocserv/command.go` (JSON and plain text command parser)
- ✅ `test/mock-ocserv/fixtures.go` (fixture loading and caching)
- ✅ `test/mock-ocserv/README.md` (comprehensive documentation)

**Features implemented:**
- ✅ Listen on Unix socket (configurable path, default: /tmp/occtl-test.socket)
- ✅ Parse occtl JSON protocol: `{"command": ["show", "-j", "users"]}`
- ✅ Parse plain text format: `show -j users` (for testing)
- ✅ Return realistic responses from 14 production fixtures
- ✅ Log all requests with -verbose flag
- ✅ Graceful shutdown on SIGINT/SIGTERM
- ✅ Concurrent connection handling
- ✅ Command-line flags: -socket, -fixtures, -verbose

**Testing results:**
- ✅ Compiles successfully (Go 1.25)
- ✅ Loads 14 fixtures from test/fixtures/ocserv/occtl
- ✅ Starts and listens on Unix socket
- ✅ Handles SIGTERM gracefully
- ✅ Socket permissions set to 0666 (like real ocserv)

**Supported commands (13 total):**
- `show -j users` - List all connected users
- `show -j user <name>` - User details
- `show -j id <id>` - Connection by ID
- `show -j status` - Server statistics
- `show -j sessions all/valid` - Session management
- `show -j session <id>` - Session details
- `show -j cookies all/valid` - Cookie management
- `show -j iroutes` - User routes
- `show -j events` - Event stream
- `show -j ip ban points` - IP bans
- `show id <id>` - Plain text format

**Dependencies:** None

**Next steps:**
- Integrate into podman-compose for CI/CD
- Add integration tests using mock server
- Test with real ocserv-agent OcctlManager

---

## 🧪 Phase 2: Occtl Integration Tests (4 tasks)

### Task 2.1: Setup test infrastructure for occtl
**Status:** PENDING | **Priority:** HIGH | **Time:** 30 min

**Objectives:**
- Create test helpers for starting/stopping mock socket
- Setup test fixtures validation
- Cleanup functions for resources
- Test utilities for common operations

**Files to create:**
- `internal/ocserv/occtl_integration_test.go`
- `internal/ocserv/testutil/socket_helper.go`
- `internal/ocserv/testutil/fixtures.go`

**Acceptance criteria:**
- ✅ Can start mock socket in tests
- ✅ Automatic cleanup after tests
- ✅ Test fixtures load correctly
- ✅ Parallel test support

**Dependencies:** Task 1.3

---

### Task 2.2: Test ShowUsers and basic commands
**Status:** PENDING | **Priority:** HIGH | **Time:** 45 min

**Objectives:**
- Test `ShowUsers()` with real JSON parsing
- Test `ShowStatus()` parsing
- Test `ShowStats()` parsing
- Error scenarios (socket not available, timeout, invalid JSON)

**Coverage target:** occtl.go 0% → 40%

**Test cases:**
- ShowUsers with 0, 1, 3+ users
- ShowStatus with different states
- ShowStats with various numbers
- Timeout handling
- Socket connection errors
- JSON parsing errors

**Acceptance criteria:**
- ✅ All test cases pass
- ✅ Coverage reaches 40%+
- ✅ Error handling tested
- ✅ No flaky tests

**Dependencies:** Task 2.1

---

### Task 2.3: Test user management commands
**Status:** PENDING | **Priority:** MEDIUM | **Time:** 45 min

**Objectives:**
- Test `ShowUser(username)` with valid/invalid users
- Test `ShowUserByID(id)` with valid/invalid IDs
- Test `DisconnectUser(username)`
- Test `DisconnectUserByID(id)`
- Edge cases (user not found, already disconnected)

**Coverage target:** occtl.go 40% → 70%

**Test cases:**
- ShowUser with existing user
- ShowUser with non-existent user
- ShowUserByID with valid ID
- ShowUserByID with invalid ID
- DisconnectUser success/failure
- DisconnectUserByID success/failure

**Acceptance criteria:**
- ✅ All test cases pass
- ✅ Coverage reaches 70%+
- ✅ Error messages validated
- ✅ Edge cases covered

**Dependencies:** Task 2.2

---

### Task 2.4: Test IP management commands
**Status:** PENDING | **Priority:** MEDIUM | **Time:** 30 min

**Objectives:**
- Test `ShowIPBans()` with banned/no banned IPs
- Test `ShowIPBanPoints()` with various points
- Test `UnbanIP(ip)` success/failure
- Test `Reload()` command

**Coverage target:** occtl.go 70% → 90%

**Test cases:**
- ShowIPBans with empty list
- ShowIPBans with multiple bans
- ShowIPBanPoints with 0 points
- ShowIPBanPoints with various IPs
- UnbanIP with banned IP
- UnbanIP with non-banned IP
- Reload success

**Acceptance criteria:**
- ✅ All test cases pass
- ✅ Coverage reaches 90%+
- ✅ IP validation tested
- ✅ Reload command works

**Dependencies:** Task 2.3

---

## ⚙️ Phase 3: Systemctl Integration Tests (3 tasks)

### Task 3.1: Setup systemctl test infrastructure
**Status:** PENDING | **Priority:** HIGH | **Time:** 30 min

**Objectives:**
- Create mock systemd service for testing (or use user-level systemd)
- Test helpers for service management
- Cleanup after tests
- Handle platforms without systemd

**Files to create:**
- `internal/ocserv/systemctl_integration_test.go`
- `internal/ocserv/testutil/systemd_helper.go`
- `test/fixtures/systemd/mock-service.service`

**Acceptance criteria:**
- ✅ Can create test service
- ✅ Cleanup removes test service
- ✅ Tests skip on non-systemd systems
- ✅ Parallel test safe

**Dependencies:** None

---

### Task 3.2: Test service lifecycle commands
**Status:** PENDING | **Priority:** HIGH | **Time:** 45 min

**Objectives:**
- Test `Start()` command
- Test `Stop()` command
- Test `Restart()` command
- Test `Reload()` command
- Error scenarios (service not found, permission denied)

**Coverage target:** systemctl.go 0% → 60%

**Test cases:**
- Start stopped service
- Stop running service
- Restart running service
- Reload with reload support
- Service not found error
- Permission denied error
- Timeout handling

**Acceptance criteria:**
- ✅ All test cases pass
- ✅ Coverage reaches 60%+
- ✅ Error handling tested
- ✅ State transitions validated

**Dependencies:** Task 3.1

---

### Task 3.3: Test service status commands
**Status:** PENDING | **Priority:** MEDIUM | **Time:** 30 min

**Objectives:**
- Test `Status()` parsing (systemctl show output)
- Test `IsActive()` check
- Test `IsEnabled()` check
- Various service states (running, dead, failed)

**Coverage target:** systemctl.go 60% → 85%

**Test cases:**
- Status for running service
- Status for stopped service
- Status for failed service
- Status for non-existent service
- IsActive true/false
- IsEnabled true/false
- Status field parsing

**Acceptance criteria:**
- ✅ All test cases pass
- ✅ Coverage reaches 85%+
- ✅ All status fields parsed
- ✅ Edge cases covered

**Dependencies:** Task 3.2

---

## 🌐 Phase 4: gRPC End-to-End Tests (3 tasks)

### Task 4.1: Create gRPC integration test framework
**Status:** PENDING | **Priority:** HIGH | **Time:** 1 hour

**Objectives:**
- Real gRPC server startup in tests
- Test client with mTLS authentication
- Port allocation helper (avoid conflicts)
- Graceful shutdown testing
- Integration with mock ocserv socket

**Files to create:**
- `internal/grpc/integration_test.go`
- `internal/grpc/testutil/server_helper.go`
- `internal/grpc/testutil/client_helper.go`
- `internal/grpc/testutil/port_allocator.go`

**Features:**
- Start real gRPC server on random port
- Generate test certificates (use internal/cert)
- Create authenticated test client
- Automatic cleanup

**Acceptance criteria:**
- ✅ Can start real gRPC server
- ✅ mTLS connection works
- ✅ Port conflicts avoided
- ✅ Clean shutdown tested

**Dependencies:** Task 1.3 (mock ocserv), Task 3.1 (systemctl)

---

### Task 4.2: Test ExecuteCommand with real execution
**Status:** PENDING | **Priority:** HIGH | **Time:** 45 min

**Objectives:**
- Test ExecuteCommand RPC with real occtl commands (via mock socket)
- Test ExecuteCommand RPC with real systemctl commands (via test service)
- Error scenarios (command not allowed, invalid args, timeout)
- Request ID propagation

**Coverage target:** handlers.go 64.7% → 85%

**Test cases:**
- ExecuteCommand occtl show users
- ExecuteCommand occtl disconnect
- ExecuteCommand systemctl status
- ExecuteCommand systemctl restart
- Command not in whitelist
- Invalid arguments (injection attempts)
- Timeout scenario
- Request ID in logs

**Acceptance criteria:**
- ✅ All test cases pass
- ✅ Coverage reaches 85%+
- ✅ Real commands execute
- ✅ Security validation works

**Dependencies:** Task 4.1

---

### Task 4.3: Test Server.Serve with real listener
**Status:** PENDING | **Priority:** MEDIUM | **Time:** 30 min

**Objectives:**
- Test `Serve()` method with real network listener
- Test connection acceptance
- Test graceful shutdown (Stop, GracefulStop)
- Test listener errors

**Coverage target:** server.go Serve 0% → 100%

**Test cases:**
- Serve starts and accepts connections
- Client can connect and call RPCs
- GracefulStop waits for requests
- Stop immediately closes
- Listener error handling
- Multiple concurrent connections

**Acceptance criteria:**
- ✅ All test cases pass
- ✅ Serve coverage 100%
- ✅ Shutdown behavior validated
- ✅ No connection leaks

**Dependencies:** Task 4.1

---

## 🚀 Phase 5: Remote Server Testing (2 tasks)

### Task 5.1: Deploy to test server via Ansible
**Status:** BLOCKED | **Priority:** MEDIUM | **Time:** 45 min
**Blocked by:** Task 1.1, Task 1.2

**Objectives:**
- Deploy new agent version to remote server (configured via .env)
- Backup old agent (v0.3.0-24-groutes)
- Update configuration
- Restart agent service
- Verify no disruption to existing VPN users

**Steps:**
1. Backup current agent binary
2. Backup current config
3. Stop old agent
4. Deploy new agent binary
5. Update config (if needed)
6. Start new agent
7. Verify connectivity
8. Rollback procedure if fails

**CRITICAL Safety measures:**
- ✅ Backup before changes
- ✅ Rollback script ready
- ✅ Monitor existing VPN connections
- ✅ Test on non-production first (if possible)

**Acceptance criteria:**
- ✅ New agent deployed successfully
- ✅ Existing VPN users unaffected
- ✅ gRPC API responsive
- ✅ Can rollback if needed

**Dependencies:** Task 1.2

---

### Task 5.2: End-to-end production tests
**Status:** BLOCKED | **Priority:** HIGH | **Time:** 1 hour
**Blocked by:** Task 5.1

**Objectives:**
- Test all gRPC commands on real server
- Verify with real VPN users (ocserv 1.3)
- Performance validation
- Error scenario testing
- Collect metrics and logs

**Test scenarios:**
- HealthCheck (all 3 tiers if implemented)
- ExecuteCommand occtl show users (real users)
- ExecuteCommand occtl show status
- ExecuteCommand systemctl status ocserv
- DisconnectUser (test user only!)
- Configuration reading
- Error handling (invalid commands)

**Metrics to collect:**
- Response times
- Memory usage
- CPU usage
- Network bandwidth
- Error rates

**Acceptance criteria:**
- ✅ All commands work on production
- ✅ Response times acceptable (<100ms p95)
- ✅ No impact on VPN performance
- ✅ Error handling works correctly
- ✅ Logs are useful for debugging

**Dependencies:** Task 5.1

---

## 📈 Expected Results

### Coverage Improvements
- **internal/ocserv/occtl.go:** 0% → 90%
- **internal/ocserv/systemctl.go:** 0% → 85%
- **internal/grpc/server.go (Serve):** 0% → 100%
- **internal/grpc/handlers.go:** 64.7% → 85%
- **Overall internal packages:** 51.2% → **75-80%**

### Quality Metrics
- ✅ Integration tests run in CI
- ✅ Real command execution tested
- ✅ Production deployment validated
- ✅ No regression in existing functionality
- ✅ Comprehensive error scenario coverage

---

## 🔄 Workflow for Each Task

1. **Before starting:**
   - Mark task as IN PROGRESS in this file
   - Update todo list with TodoWrite tool
   - Check for blockers

2. **During implementation:**
   - Write code following best practices
   - Write tests for new functionality
   - Run tests locally: `go test ./...`
   - Run pre-commit checks: `scripts/quick-check.sh`

3. **After completion:**
   - Run full test suite locally
   - Verify coverage improvement: `go test -cover ./...`
   - Update this plan (mark task COMPLETED)
   - Commit with descriptive message
   - Update CURRENT.md if milestone reached

4. **If blocked:**
   - Mark blocker in this file
   - Document what's needed to unblock
   - Escalate to user if cannot resolve

---

## 📝 Notes

### Remote Server Safety
- **Configuration:** Set REMOTE_HOST, REMOTE_USER, REMOTE_PASSWORD in `.env` file
- **Example (RFC 5737):** See `.env.example` for template
- **Existing setup:** ocserv 1.3 + old agent v0.3.0-24-groutes
- **Active users:** Real VPN users connected
- **CRITICAL:** Do NOT break existing VPN service
- **Strategy:** Backup → Deploy → Test → Rollback if needed

### Testing Strategy
- **Local tests:** Mock ocserv socket + mock systemd service
- **Integration tests:** Real server, real ocserv, real users
- **Safety:** Always test locally first, then deploy carefully

### Timeline
- **Estimated total:** ~12 hours
- **Blockers:** Task 1.1 must be done first (30 min)
- **Can parallelize:** Some tasks can run in parallel after blockers cleared
- **Target completion:** 1-2 days of focused work

---

**Last Updated:** 2025-10-23
