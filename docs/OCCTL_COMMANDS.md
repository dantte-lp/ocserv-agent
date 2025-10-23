# occtl Commands Support

This document describes which occtl commands are supported by ocserv-agent and their current status.

## Command Execution via gRPC

All occtl commands are executed via the `ExecuteCommand` RPC with:

```bash
{
  "command_type": "occtl",
  "args": ["show", "users"]
}
```

The agent automatically uses `occtl -j` (JSON mode) when available and falls back to text parsing for commands that don't support JSON.

## Supported Commands

### ✅ Fully Working (JSON mode)

| Command | Status | Notes |
|---------|--------|-------|
| `show status` | ✅ Working | Returns server status and uptime |
| `show users` | ✅ Working | Returns all connected users (40+ fields per user) |
| `show ip bans` | ✅ Working | Returns list of banned IP addresses |
| `show ip ban points` | ✅ Working | Returns IPs with ban points |
| `show user [NAME]` | ✅ Working | Returns detailed user information |
| `show id [ID]` | ✅ Working | Returns user information by ID |

### ⚠️ Partial Support (occtl JSON bugs)

| Command | Status | Issue |
|---------|--------|-------|
| `show iroutes` | ⚠️ JSON parsing fails | occtl returns invalid JSON (duplicate keys, missing commas) |
| `show sessions all` | ⚠️ JSON parsing fails | occtl returns invalid JSON |
| `show sessions valid` | ⚠️ JSON parsing fails | occtl returns invalid JSON |
| `show session [SID]` | ⚠️ Not tested | May have same JSON issues |

**Note:** These commands have bugs in occtl 1.3.0 where the JSON output is malformed. The agent correctly identifies these as errors.

### ✅ Action Commands

| Command | Status | Notes |
|---------|--------|-------|
| `disconnect user [NAME]` | ✅ Working | Disconnects specified user |
| `disconnect id [ID]` | ✅ Working | Disconnects user by ID |
| `unban ip [IP]` | ✅ Working | Removes IP from ban list |
| `reload` | ✅ Working | Reloads server configuration |

### 🔴 Not Supported

| Command | Status | Reason |
|---------|--------|--------|
| `stop now` | 🔴 Blocked | Dangerous - would terminate ocserv |
| `show events` | 🔴 Not implemented | Requires streaming RPC (planned for v0.4.0) |

## systemctl Commands

The agent also supports systemctl commands for ocserv service management:

| Command | Status | Notes |
|---------|--------|-------|
| `systemctl status ocserv` | ✅ Working | Returns service status |
| `systemctl start ocserv` | ✅ Working | Starts ocserv service |
| `systemctl stop ocserv` | ✅ Working | Stops ocserv service |
| `systemctl restart ocserv` | ✅ Working | Restarts ocserv service |
| `systemctl reload ocserv` | ✅ Working | Reloads ocserv configuration |

## User Data Fields (JSON mode)

When using `show users` or `show user [NAME]`, the agent returns 40+ fields including:

### Connection Info
- ID, Username, Groupname, State
- VHost, Device, MTU
- Remote IP, Local Device IP
- Location

### Network Config
- IPv4, P-t-P IPv4
- IPv6, P-t-P IPv6
- DNS servers
- Routes, No-routes, iRoutes
- Split-DNS-Domains

### Traffic Stats
- RX, TX (bytes)
- Readable RX, TX (human format)
- Average RX, TX (rates)

### Security
- TLS ciphersuite (e.g., TLS1.3 + ECDHE + RSA-PSS + AES-256-GCM)
- DTLS cipher (e.g., DTLS1.2 + ECDHE-RSA + AES-256-GCM)
- CSTP compression, DTLS compression

### Session Info
- Connected at (timestamp and duration)
- Full session ID, Short session ID
- DPD timeout, KeepAlive
- Hostname, User-Agent

### Restrictions
- Restricted to routes
- Restricted to ports

## Examples

### List Connected Users

```bash
grpcurl -d '{"command_type": "occtl", "args": ["show", "users"]}' \
  localhost:9090 agent.v1.AgentService/ExecuteCommand
```

Response:
```json
{
  "success": true,
  "stdout": "Connected users: 3"
}
```

### Get User Details

```bash
grpcurl -d '{"command_type": "occtl", "args": ["show", "user", "testuser"]}' \
  localhost:9090 agent.v1.AgentService/ExecuteCommand
```

### Disconnect User

```bash
grpcurl -d '{"command_type": "occtl", "args": ["disconnect", "user", "testuser"]}' \
  localhost:9090 agent.v1.AgentService/ExecuteCommand
```

### Check Server Status

```bash
grpcurl -d '{"command_type": "occtl", "args": ["show", "status"]}' \
  localhost:9090 agent.v1.AgentService/ExecuteCommand
```

### Restart ocserv

```bash
grpcurl -d '{"command_type": "systemctl", "args": ["restart", "ocserv"]}' \
  localhost:9090 agent.v1.AgentService/ExecuteCommand
```

## Known Issues

### occtl JSON Bugs (Production Tested: 2025-10-23)

Some occtl commands return invalid JSON. These are **upstream bugs in occtl 1.3.0**, not in ocserv-agent.

#### 1. `show iroutes` - COMPLETELY BROKEN

**Issue:** Invalid JSON structure - completely unparseable

**Example output:**
```json
  {
    "ID":  844960,
    "Username":  "lpa",
    "Device":  "vpns1",
    "IP":  "10.0.16.23",
    "iRoutes": [],
    "IP":  "10.0.16.23"    // ❌ DUPLICATE KEY!
    "ID":  843724,          // ❌ MISSING COMMA!
    "Username":  "win2k25",
    ...
  }
```

**Problems:**
- ❌ Duplicate "IP" key in each object
- ❌ Missing commas between objects
- ❌ Missing array wrapper `[` at start
- ❌ Would fail any JSON parser

**Status:** Cannot be parsed by ocserv-agent or any JSON library.

#### 2. `show sessions all` - MINOR BUG

**Issue:** Trailing commas in JSON objects (not allowed in strict JSON)

**Example output:**
```json
[
  {
    "Session": "Al/uNv",
    "Username": "win2k25",
    "Remote IP": "90.156.162.214",
    "in_use": 1,  // ⚠️ TRAILING COMMA
  },
  ...
]
```

**Status:** Some parsers may accept this, but it's technically invalid JSON.

#### 3. `show sessions valid` - MINOR BUG

**Issue:** Same as `show sessions all` - trailing commas in objects.

**Status:** Same parsing issues as above.

---

**Tested on:** Production server with 3 active VPN users (ocserv 1.3.0, occtl 1.3.0)
**Date:** 2025-10-23

**Related Upstream Issues:**
- [GitLab #220](https://gitlab.com/openconnect/ocserv/-/issues/220) - Invalid JSON structure with trailing commas
- [GitLab #517](https://gitlab.com/openconnect/ocserv/-/issues/517) - occtl generates invalid JSON with `--debug` parameter
- [GitLab #20](https://gitlab.com/openconnect/ocserv/-/issues/20) - occtl: show iroutes command

**Previous Fixes (but still broken in 1.3.0):**
- v1.2.1 - Fixed duplicate key in `occtl --json show users` output
- v1.2.0 - Fixed JSON output with `--debug` flag (#517)
- v0.10.7 - Fixed several cases of invalid JSON output

**Note:** Despite previous fixes in v1.2.x, JSON bugs persist in ocserv/occtl 1.3.0. These appear to be regressions or new issues.

**Workaround:** For `show iroutes`, use text mode output (without `-j` flag). For `show sessions`, some lenient JSON parsers may work, but strict parsers will fail on trailing commas.

**Recommendation:** Consider reporting these issues to the [ocserv GitLab issue tracker](https://gitlab.com/openconnect/ocserv/-/issues).

## Command Validation

For security, the agent validates all commands against a whitelist:

**Allowed command types:**
- `occtl` - ocserv control commands
- `systemctl` - systemd service management

**Validation includes:**
- Command whitelist checking
- Argument sanitization
- Protection against command injection
- Shell metacharacter filtering
- Directory traversal prevention

## Future Enhancements (v0.4.0+)

- [ ] `show events` - Real-time streaming support via ServerStream RPC
- [ ] Custom parsers for commands with invalid JSON
- [ ] Structured response types (not just stdout string)
- [ ] Typed user/session objects in proto definitions
- [ ] Batch command execution

## See Also

- [gRPC Testing Guide](GRPC_TESTING.md) - How to test commands with grpcurl
- [ocserv Compatibility](todo/OCSERV_COMPATIBILITY.md) - Complete ocserv 1.3.0 feature analysis
- [Production Testing](TESTING_PROD.md) - Production testing procedures
