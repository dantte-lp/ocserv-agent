# Testing ocserv-agent v0.2.0 BETA

**Target Server:** 195.238.126.25 (production ocserv 1.3.0)

## Quick Start

### 1. Download Binary

```bash
# SSH to server
ssh root@195.238.126.25

# Create directory
mkdir -p /opt/ocserv-agent
cd /opt/ocserv-agent

# Download binary
wget https://github.com/dantte-lp/ocserv-agent/releases/download/v0.2.0/ocserv-agent-linux-amd64
chmod +x ocserv-agent-linux-amd64

# Verify
./ocserv-agent-linux-amd64 --version
```

### 2. Create Configuration

Create `/opt/ocserv-agent/config.yaml`:

```yaml
# Server Configuration
server:
  host: "0.0.0.0"
  port: 50051
  tls:
    enabled: true
    cert_file: "/path/to/server-cert.pem"
    key_file: "/path/to/server-key.pem"
    ca_file: "/path/to/ca-cert.pem"
    min_version: "1.3"
    server_name: "ocserv-agent.local"

# ocserv Configuration
ocserv:
  systemd_service: "ocserv"
  ctl_socket: "/var/run/ocserv/ocserv.sock"
  config_file: "/etc/ocserv/ocserv.conf"
  config_per_user_dir: "/etc/ocserv/config-per-user"
  config_per_group_dir: "/etc/ocserv/config-per-group"

# Security
security:
  sudo_user: ""  # Empty for root, or specify user
  max_command_timeout: "30s"
  allowed_commands:
    - systemctl
    - occtl

# Health Check
health:
  interval: "10s"
  timeout: "5s"

# Logging
logging:
  level: "info"
  format: "json"
  output: "stdout"
```

### 3. Generate Test Certificates (If Needed)

```bash
# Quick self-signed certs for testing
cd /opt/ocserv-agent

# CA
openssl req -x509 -newkey rsa:4096 -keyout ca-key.pem -out ca-cert.pem -days 365 -nodes -subj "/CN=Test CA"

# Server
openssl req -newkey rsa:4096 -keyout server-key.pem -out server-csr.pem -nodes -subj "/CN=ocserv-agent.local"
openssl x509 -req -in server-csr.pem -CA ca-cert.pem -CAkey ca-key.pem -out server-cert.pem -days 365 -CAcreateserial

# Client
openssl req -newkey rsa:4096 -keyout client-key.pem -out client-csr.pem -nodes -subj "/CN=test-client"
openssl x509 -req -in client-csr.pem -CA ca-cert.pem -CAkey ca-key.pem -out client-cert.pem -days 365 -CAcreateserial

# Update config.yaml paths
sed -i 's|/path/to/server-cert.pem|/opt/ocserv-agent/server-cert.pem|g' config.yaml
sed -i 's|/path/to/server-key.pem|/opt/ocserv-agent/server-key.pem|g' config.yaml
sed -i 's|/path/to/ca-cert.pem|/opt/ocserv-agent/ca-cert.pem|g' config.yaml
```

### 4. Run Agent

```bash
# Test run (foreground)
./ocserv-agent-linux-amd64 --config config.yaml

# Check logs
# Should see:
# {"level":"info","message":"Starting ocserv-agent","version":"v0.2.0"}
# {"level":"info","message":"gRPC server listening","address":"0.0.0.0:50051"}
```

## Testing Commands

### Using grpcurl

```bash
# Install grpcurl
wget https://github.com/fullstorydev/grpcurl/releases/download/v1.9.1/grpcurl_1.9.1_linux_amd64.tar.gz
tar -xzf grpcurl_1.9.1_linux_amd64.tar.gz
chmod +x grpcurl

# Test HealthCheck
./grpcurl -insecure \
  -cert client-cert.pem \
  -key client-key.pem \
  -cacert ca-cert.pem \
  -d '{}' \
  localhost:50051 \
  agent.v1.AgentService/HealthCheck

# Test ExecuteCommand - show users
./grpcurl -insecure \
  -cert client-cert.pem \
  -key client-key.pem \
  -cacert ca-cert.pem \
  -d '{"command":"occtl","args":["show","users"]}' \
  localhost:50051 \
  agent.v1.AgentService/ExecuteCommand

# Test ExecuteCommand - show status
./grpcurl -insecure \
  -cert client-cert.pem \
  -key client-key.pem \
  -cacert ca-cert.pem \
  -d '{"command":"occtl","args":["show","status"]}' \
  localhost:50051 \
  agent.v1.AgentService/ExecuteCommand
```

## Test Scenarios

### 1. Basic Functionality

**Test all 16 occtl commands:**

```bash
# User management
occtl show users
occtl show user lpa
occtl show id 836625

# Session management
occtl show sessions all
occtl show sessions valid
occtl show session DN4npe

# Server status
occtl show status
occtl show stats

# Routes and network
occtl show iroutes

# Security
occtl show ip bans
occtl show ip ban points
occtl unban ip 1.2.3.4

# Control
occtl reload
```

### 2. Multi-Session User

**Test user with multiple active sessions:**

```bash
# Connect same user from 2 devices
# iPhone + macOS for example

# Then query
occtl -j show user lpa

# Should return array with 2 elements
```

### 3. Performance

```bash
# Measure response time
time occtl show users
time occtl show status
time occtl show sessions all
```

### 4. Error Handling

```bash
# Non-existent user
occtl show user nonexistent

# Invalid ID
occtl show id 999999

# Unauthorized command (should fail)
occtl show events  # Not implemented yet
```

## Expected Results

### HealthCheck
```json
{
  "status": "healthy",
  "timestamp": "2025-10-23T03:00:00Z",
  "message": "Agent is operational"
}
```

### ExecuteCommand (show users)
```json
{
  "success": true,
  "stdout": "Connected users: 2",
  "stderr": "",
  "exitCode": 0
}
```

### ExecuteCommand (show status)
```json
{
  "success": true,
  "stdout": "Status: online\nSec-mod: ...\nCompression: ...\nUptime: ...",
  "stderr": "",
  "exitCode": 0
}
```

## Monitoring

### Check Agent Logs

```bash
# If running with systemd
journalctl -u ocserv-agent -f

# If running foreground
# Logs to stdout
```

### Check ocserv Status

```bash
# Verify ocserv is running
systemctl status ocserv

# Check socket
ls -l /var/run/ocserv/ocserv.sock

# Test occtl directly
occtl -j show status
```

## Troubleshooting

### Agent Won't Start

```bash
# Check config syntax
cat config.yaml

# Check certificate paths
ls -l /opt/ocserv-agent/*.pem

# Check port not in use
netstat -tlnp | grep 50051

# Check permissions
ls -l ocserv-agent-linux-amd64
# Should be executable
```

### mTLS Connection Failed

```bash
# Verify certificates
openssl verify -CAfile ca-cert.pem server-cert.pem
openssl verify -CAfile ca-cert.pem client-cert.pem

# Check certificate details
openssl x509 -in server-cert.pem -text -noout
```

### Command Execution Failed

```bash
# Test occtl directly
occtl -j show users

# Check ocserv socket permissions
ls -l /var/run/ocserv/ocserv.sock

# Check if running as correct user
whoami
# If agent runs as non-root, may need sudo_user config
```

### Permission Denied

```bash
# Run agent as root
sudo ./ocserv-agent-linux-amd64 --config config.yaml

# Or configure sudo_user in config
security:
  sudo_user: "root"
```

## Production Deployment

### systemd Service

Create `/etc/systemd/system/ocserv-agent.service`:

```ini
[Unit]
Description=ocserv Management Agent
After=network.target ocserv.service
Requires=ocserv.service

[Service]
Type=simple
User=root
Group=root
ExecStart=/opt/ocserv-agent/ocserv-agent-linux-amd64 --config /opt/ocserv-agent/config.yaml
Restart=on-failure
RestartSec=5s

# Security
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/run/ocserv

# Limits
LimitNOFILE=65536
MemoryLimit=256M
CPUQuota=50%

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
systemctl daemon-reload
systemctl enable ocserv-agent
systemctl start ocserv-agent
systemctl status ocserv-agent
```

## Cleanup

```bash
# Stop agent
killall ocserv-agent-linux-amd64

# Remove files
rm -rf /opt/ocserv-agent

# Or keep for future use
```

## Reporting Issues

If you encounter issues:

1. Collect logs: `journalctl -u ocserv-agent -n 100 > agent-logs.txt`
2. Check config: `cat config.yaml`
3. Test occtl: `occtl -j show status`
4. Report at: https://github.com/dantte-lp/ocserv-agent/issues

Include:
- OS version: `cat /etc/os-release`
- ocserv version: `ocserv --version`
- Agent version: `./ocserv-agent-linux-amd64 --version`
- Error message
- Steps to reproduce

---

**Testing Server:** 195.238.126.25
**ocserv version:** 1.3.0
**OS:** Oracle Linux 10
**Release:** v0.2.0 BETA
