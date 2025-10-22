# occtl Production Output Examples

**Source:** Production ocserv 1.3.0 server (Oracle Linux 10)
**Date:** 2025-10-23
**Purpose:** Reference examples for parsing occtl command output

This directory contains real output from a production ocserv deployment, useful for:
- Testing occtl output parsing in OcctlManager
- Understanding JSON structure for each command
- Reference for implementing missing commands
- Validation of data types and field names

---

## Available Examples

### help
**Command:** `occtl`
**Format:** Plain text
**Description:** Complete list of available occtl commands and options

**Commands listed:**
- disconnect user/id
- unban ip
- reload
- show status/users/ip bans/ip ban points/iroutes
- show sessions (all/valid/specific)
- show user/id
- show events
- stop now

---

### occtl -j show users
**Format:** JSON array
**Description:** List all connected users with detailed connection information

**Key fields:**
```json
{
  "ID": 835235,
  "Username": "lpa",
  "Groupname": "(none)",
  "State": "connected",
  "vhost": "default",
  "Device": "vpns0",
  "MTU": "1402",
  "Remote IP": "90.156.164.225",
  "Location": "unknown",
  "Local Device IP": "195.238.126.25",
  "IPv4": "10.0.16.23",
  "P-t-P IPv4": "10.0.16.1",
  "IPv6": "fc00::1:8651",
  "P-t-P IPv6": "fc00::1:8601",
  "User-Agent": "AnyConnect AppleSSLVPN_Darwin_ARM (iPhone) 5.1.11.347",
  "RX": "0",
  "TX": "96",
  "_RX": "0 bytes",
  "_TX": "96 bytes",
  "Average RX": "0 bytes/s",
  "Average TX": "32 bytes/s",
  "DPD": "90",
  "KeepAlive": "32400",
  "Hostname": "localhost",
  "Connected at": "2025-10-23 02:32",
  "_Connected at": "    3s",
  "raw_connected_at": 1761175942,
  "Full session": "0/zuQ1RjBWv5J/hneJun8+sesWs=",
  "Session": "0/zuQ1",
  "TLS ciphersuite": "(TLS1.3)-(ECDHE-SECP256R1)-(RSA-PSS-RSAE-SHA256)-(AES-256-GCM)",
  "DNS": ["10.0.16.1", "fc00::1:8601"],
  "NBNS": [],
  "Split-DNS-Domains": [],
  "Routes": "defaultroute",
  "No-routes": [],
  "iRoutes": [],
  "Restricted to routes": "False",
  "Restricted to ports": []
}
```

**Note:** Real production data shows iPhone AnyConnect client connection

---

### occtl -j show status
**Format:** JSON object
**Description:** Server statistics and operational status

**Key fields:**
```json
{
  "Status": "online",
  "Server PID": 802,
  "Sec-mod PID": 821,
  "Sec-mod instance count": 1,
  "Up since": "2025-09-12 14:37",
  "_Up since": "40days",
  "raw_up_since": 1757677078,
  "uptime": 3498723,
  "Active sessions": 0,
  "Total sessions": 44,
  "Total authentication failures": 10,
  "IPs in ban list": 0,
  "Last stats reset": "2025-10-20 20:40",
  "_Last stats reset": " 2days",
  "raw_last_stats_reset": 1760982020,
  "Sessions handled": 4,
  "Timed out sessions": 0,
  "Timed out (idle) sessions": 0,
  "Closed due to error sessions": 2,
  "Authentication failures": 0,
  "Average auth time": "    0s",
  "raw_avg_auth_time": 0,
  "Max auth time": "    5s",
  "raw_max_auth_time": 5,
  "Average session time": " 3h:43m",
  "raw_avg_session_time": 13380,
  "Max session time": " 1h:32m",
  "raw_max_session_time": 5520,
  "Min MTU": 1324,
  "Max MTU": 1402,
  "RX": "110.0 MB",
  "raw_rx": 110013000,
  "TX": "1.8 GB",
  "raw_tx": 1827434000
}
```

**Note:** Server has been running for 40 days with 44 total sessions

---

### occtl -j show user
**Format:** JSON object
**Description:** Detailed information about specific user (includes all show users fields)

---

### occtl -j show id
**Format:** JSON object
**Description:** Detailed information about specific connection ID

**Also available:** `occtl show id` (plain text format)

---

### occtl -j show sessions all
**Format:** JSON array
**Description:** All session identifiers and their states

**Key fields:**
```json
{
  "Session": "0/zuQ1",
  "Full session": "0/zuQ1RjBWv5J/hneJun8+sesWs=",
  "Created": "2025-10-23 02:30",
  "State": "authenticated",
  "Username": "lpa",
  "Groupname": "(none)",
  "vhost": "default",
  "User-Agent": "AnyConnect AppleSSLVPN_Darwin_ARM (iPhone) 5.1.11.347",
  "Remote IP": "90.156.164.225",
  "Location": "unknown",
  "session_is_open": 1,
  "tls_auth_ok": 0,
  "in_use": 1
}
```

---

### occtl -j show sessions valid
**Format:** JSON array
**Description:** Sessions valid for reconnection (cookie-based)

**Structure:** Same as "show sessions all" but filtered

---

### occtl -j show cookies all
**Format:** JSON array
**Description:** All session cookies (alias for sessions)

**Note:** Cookies and sessions are the same in ocserv terminology

---

### occtl -j show cookies valid
**Format:** JSON array
**Description:** Valid reconnection cookies (alias for valid sessions)

---

### occtl -j show iroutes
**Format:** JSON object
**Description:** User-provided routes (routes pushed from client to server)

**Key fields:**
```json
{
  "ID": 835257,
  "Username": "lpa",
  "vhost": "default",
  "Device": "vpns0",
  "IP": "10.0.16.23",
  "iRoutes": []
}
```

**Note:** In this example, no iRoutes are configured (empty array)

---

### occtl -j show events
**Format:** JSON array (streaming)
**Description:** Real-time events of connecting/disconnecting users

**Structure:** Array of event objects with timestamps and connection details

**Note:** This is a streaming command - continues to output events in real-time

---

## Data Type Reference

### Common Fields

**IDs:**
- `ID`: Integer (connection ID)
- `Server PID`, `Sec-mod PID`: Integer

**Strings:**
- `Username`, `Groupname`, `State`, `vhost`, `Device`
- `Remote IP`, `Local Device IP`, `IPv4`, `IPv6`
- `User-Agent`, `Hostname`, `TLS ciphersuite`

**Timestamps:**
- `Connected at`: String (human-readable)
- `_Connected at`: String (relative time, e.g., "3s", "2days")
- `raw_connected_at`: Integer (Unix timestamp)

**Traffic:**
- `RX`, `TX`: String (formatted with units, e.g., "110.0 MB")
- `raw_rx`, `raw_tx`: Integer (bytes)
- `Average RX/TX`: String with rate

**Arrays:**
- `DNS`, `NBNS`, `Split-DNS-Domains`: String arrays
- `Routes`, `No-routes`, `iRoutes`: String arrays
- `Restricted to ports`: Array

**Booleans (as strings):**
- `Restricted to routes`: "True" or "False"

**Session flags (integers):**
- `session_is_open`, `tls_auth_ok`, `in_use`: 0 or 1

---

## Implementation Notes

### For OcctlManager Enhancement

When implementing the missing commands, use these examples to:

1. **Define Go structs** matching JSON structure:
```go
type User struct {
    ID           int      `json:"ID"`
    Username     string   `json:"Username"`
    Groupname    string   `json:"Groupname"`
    State        string   `json:"State"`
    Vhost        string   `json:"vhost"`
    Device       string   `json:"Device"`
    MTU          string   `json:"MTU"`
    RemoteIP     string   `json:"Remote IP"`
    IPv4         string   `json:"IPv4"`
    IPv6         string   `json:"IPv6"`
    DNS          []string `json:"DNS"`
    // ... more fields
}
```

2. **Parse JSON output** with proper error handling:
```go
func (m *OcctlManager) ShowUser(ctx context.Context, username string) (*User, error) {
    output, err := m.runCommand(ctx, "show", "user", username, "-j")
    if err != nil {
        return nil, err
    }

    var user User
    if err := json.Unmarshal([]byte(output), &user); err != nil {
        return nil, fmt.Errorf("failed to parse user: %w", err)
    }

    return &user, nil
}
```

3. **Handle edge cases:**
   - Empty arrays vs null
   - Fields that can be missing
   - Multiple data formats (raw vs formatted)
   - Boolean as string ("True"/"False") vs integer (0/1)

4. **Streaming commands** (show events):
   - Use `ServerStream` for real-time updates
   - Parse line-by-line JSON objects
   - Handle context cancellation

---

## Production Insights

### Server Configuration
- **Platform:** Oracle Linux 10
- **ocserv version:** 1.3.0
- **GnuTLS:** 3.8.9
- **Uptime:** 40+ days (stable)
- **Client:** AnyConnect iPhone app

### Traffic Patterns
- Total RX: 110.0 MB
- Total TX: 1.8 GB
- Shows typical VPN usage (more TX than RX)
- 44 sessions over 40 days = ~1 session/day

### Authentication
- 10 total authentication failures
- PAM authentication method
- No current IP bans

### Performance
- Avg auth time: 0s (very fast)
- Max auth time: 5s
- Avg session time: 3h 43m
- Max session time: 1h 32m

---

## Testing Recommendations

1. **Unit Tests:**
   - Parse all example files
   - Verify struct field mapping
   - Test error handling for malformed JSON

2. **Integration Tests:**
   - Mock occtl responses using these files
   - Test all command variations
   - Verify JSON vs plain text parsing

3. **Validation:**
   - Compare parsed data with expected values
   - Test edge cases (empty arrays, missing fields)
   - Verify data type conversions

---

**Last Updated:** 2025-10-23
**Maintainer:** ocserv-agent team
