# Systemd Testing Approach

## Overview

This document explains the testing strategy for systemctl integration in ocserv-agent.

## Test Levels

### 1. Unit Tests (Phase 3 - Current)

**Location:** `internal/ocserv/systemctl_test.go`

**Approach:** Unit-style tests that verify:
- SystemctlManager creation and configuration
- Method calls don't panic
- Timeout handling
- ServiceStatus struct functionality

**Why:** These tests run without requiring:
- Real systemd daemon
- Root privileges
- Complex setup

**Coverage:** Basic functionality, API surface, error handling

### 2. Integration Tests (Phase 5 - Future)

**Location:** Will be in remote server test suite

**Approach:** Real integration tests on production server via Ansible:
- Start/stop/restart actual ocserv service
- Verify service state transitions
- Test reload functionality
- Check status reporting

**Why:** Requires:
- Real systemd daemon (system-wide)
- Proper permissions (ocserv-agent user)
- Production environment

**Coverage:** End-to-end systemctl operations

## Systemd Helper Utilities

**File:** `testutil/systemd.go`

**Purpose:** Helper for creating user-level test services

**Status:** Created but not used in current unit tests

**Future Use:** May be useful for:
- Local development with user systemd
- Container-based integration tests
- Phase 5 remote testing

## Test Service

**File:** `test/fixtures/systemd/ocserv-agent-test.service`

**Type:** Simple oneshot service (ExecStart=/bin/true)

**Purpose:** Minimal service for testing systemd operations

**Usage:** Reserved for future integration tests

## Why This Approach?

**Problem:** Systemctl requires either:
1. User-level systemd (--user flag) - not in production code
2. System-wide systemd + root access - not available in compose
3. Real server environment - only in Phase 5

**Solution:**
- Phase 3: Unit tests (what we can test now)
- Phase 5: Integration tests (what we need real environment for)

**Benefits:**
- ✅ Tests run in CI/CD
- ✅ No special permissions needed
- ✅ Fast execution
- ✅ Real integration tested on actual server

## Running Tests

```bash
# Unit tests (Phase 3)
go test ./internal/ocserv -run TestSystemctl -v

# Integration tests (Phase 5)
# Will be run via Ansible on remote server
make test-remote  # (future)
```

## Related

- **Phase 2:** Occtl integration tests (socket-based, works in compose)
- **Phase 3:** Systemctl unit tests (current)
- **Phase 5:** Remote server integration tests (future)
