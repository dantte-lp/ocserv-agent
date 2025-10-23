# Production Server Testing - Certificate Auto-Generation

## 📋 Quick Start

Test the new certificate auto-generation feature on production server `195.238.126.25`.

## ✅ What's New

The agent can now automatically generate self-signed certificates on first run, solving the "cert_file not found" error.

## 🚀 Testing Steps

### Step 1: Download Latest Binary

```bash
# SSH to production server
ssh root@195.238.126.25

# Download latest build from this repository
cd /opt/projects/repositories/ocserv-agent
git pull origin main

# Build fresh binary
podman-compose -f deploy/compose/docker-compose.build.yml up build-linux-amd64

# Copy to test location
cp bin/ocserv-agent-linux-amd64 /tmp/ocserv-agent
chmod +x /tmp/ocserv-agent
```

### Step 2: Test CLI Commands

```bash
# Test version
/tmp/ocserv-agent version

# Test help
/tmp/ocserv-agent help

# Test manual cert generation
/tmp/ocserv-agent gencert -output /tmp/test-certs -hostname cn02-lt-vno

# Verify certs created
ls -lh /tmp/test-certs/
# Should show:
# - ca.crt
# - agent.crt
# - agent.key

# Check certificate details
openssl x509 -in /tmp/test-certs/agent.crt -text -noout | grep -A 2 Subject
openssl x509 -in /tmp/test-certs/agent.crt -noout -dates
```

### Step 3: Test Auto-Generation on Startup

```bash
# Create test config with auto-generation
cat > /tmp/test-config.yaml <<'EOF'
agent_id: "cn02-lt-vno-test"
hostname: "cn02-lt-vno"

control_server:
  address: "localhost:9090"

tls:
  enabled: true
  auto_generate: true  # Enable auto-generation
  cert_file: "/tmp/auto-certs/agent.crt"
  key_file: "/tmp/auto-certs/agent.key"
  ca_file: "/tmp/auto-certs/ca.crt"
  min_version: "TLS1.3"

ocserv:
  config_path: "/etc/ocserv/ocserv.conf"
  config_per_user_dir: "/etc/ocserv/config-per-user"
  config_per_group_dir: "/etc/ocserv/config-per-group"
  ctl_socket: "/var/run/occtl.socket"
  systemd_service: "ocserv"
  backup_dir: "/var/backups/ocserv-agent"

health:
  heartbeat_interval: 15s
  deep_check_interval: 2m
  metrics_interval: 30s

telemetry:
  enabled: false

logging:
  level: "info"
  format: "text"
  output: "stdout"

security:
  allowed_commands:
    - "occtl"
    - "systemctl"
  sudo_user: "ocserv-agent"
  max_command_timeout: 300s
EOF

# Run agent (should auto-generate certs and start)
/tmp/ocserv-agent -config /tmp/test-config.yaml

# Expected output:
# 🔐 Generated self-signed certificates for bootstrap mode
#    CA Fingerprint:   SHA256:xx:xx:xx:...
#    Cert Fingerprint: SHA256:yy:yy:yy:...
#    Subject:          ocserv-agent-cn02-lt-vno
#    Valid:            2025-10-23 - 2026-10-23
#    Location:         /tmp/auto-certs
#
# ⚠️  These are self-signed certificates for autonomous operation.
#    To connect to a control server, replace with CA-signed certificates:
#    - Use: ocserv-agent gencert --ca /path/to/server-ca.crt
#
# {"level":"info","time":"2025-10-23T...","message":"Starting ocserv-agent"}
```

Press Ctrl+C to stop after verifying it starts successfully.

### Step 4: Verify Auto-Generated Certificates

```bash
# Check that certs were created
ls -lh /tmp/auto-certs/

# Verify permissions
stat -c "%a %n" /tmp/auto-certs/*
# Expected:
# 644 /tmp/auto-certs/ca.crt
# 644 /tmp/auto-certs/agent.crt
# 600 /tmp/auto-certs/agent.key

# Verify certificate chain
openssl verify -CAfile /tmp/auto-certs/ca.crt /tmp/auto-certs/agent.crt
# Expected: /tmp/auto-certs/agent.crt: OK

# Check certificate details
openssl x509 -in /tmp/auto-certs/agent.crt -text -noout | grep -E "(Subject:|Issuer:|Not Before|Not After)"
```

### Step 5: Test Without Auto-Generation (Should Fail)

```bash
# Modify config to disable auto-generation
cat > /tmp/test-config-no-auto.yaml <<'EOF'
agent_id: "cn02-lt-vno-test"
hostname: "cn02-lt-vno"

control_server:
  address: "localhost:9090"

tls:
  enabled: true
  auto_generate: false  # Disable auto-generation
  cert_file: "/tmp/nonexistent/agent.crt"
  key_file: "/tmp/nonexistent/agent.key"
  ca_file: "/tmp/nonexistent/ca.crt"
  min_version: "TLS1.3"

# ... rest same as above
ocserv:
  config_path: "/etc/ocserv/ocserv.conf"
  config_per_user_dir: "/etc/ocserv/config-per-user"
  config_per_group_dir: "/etc/ocserv/config-per-group"
  ctl_socket: "/var/run/occtl.socket"
  systemd_service: "ocserv"
  backup_dir: "/var/backups/ocserv-agent"

health:
  heartbeat_interval: 15s
  deep_check_interval: 2m
  metrics_interval: 30s

telemetry:
  enabled: false

logging:
  level: "info"
  format: "text"
  output: "stdout"

security:
  allowed_commands:
    - "occtl"
    - "systemctl"
  sudo_user: "ocserv-agent"
  max_command_timeout: 300s
EOF

# Run agent (should fail with cert not found)
/tmp/ocserv-agent -config /tmp/test-config-no-auto.yaml

# Expected output:
# Failed to load configuration: config validation failed: tls: cert_file not found: /tmp/nonexistent/agent.crt
# key_file not found: /tmp/nonexistent/agent.key
# ca_file not found: /tmp/nonexistent/ca.crt
```

### Step 6: Test Real Deployment to /etc/ocserv-agent

```bash
# Create production directory
sudo mkdir -p /etc/ocserv-agent/certs
sudo mkdir -p /var/backups/ocserv-agent

# Copy binary to production location
sudo cp /tmp/ocserv-agent /usr/local/bin/ocserv-agent-linux-amd64
sudo chmod +x /usr/local/bin/ocserv-agent-linux-amd64

# Create production config with auto-generation
sudo tee /etc/ocserv-agent/config.yaml <<'EOF'
agent_id: "cn02-lt-vno"
hostname: "cn02-lt-vno"

control_server:
  address: "localhost:9090"

tls:
  enabled: true
  auto_generate: true  # Enable for first run
  cert_file: "/etc/ocserv-agent/certs/agent.crt"
  key_file: "/etc/ocserv-agent/certs/agent.key"
  ca_file: "/etc/ocserv-agent/certs/ca.crt"
  min_version: "TLS1.3"

ocserv:
  config_path: "/etc/ocserv/ocserv.conf"
  config_per_user_dir: "/etc/ocserv/config-per-user"
  config_per_group_dir: "/etc/ocserv/config-per-group"
  ctl_socket: "/var/run/occtl.socket"
  systemd_service: "ocserv"
  backup_dir: "/var/backups/ocserv-agent"

health:
  heartbeat_interval: 15s
  deep_check_interval: 2m
  metrics_interval: 30s

telemetry:
  enabled: false

logging:
  level: "info"
  format: "json"
  output: "stdout"

security:
  allowed_commands:
    - "occtl"
    - "systemctl"
  sudo_user: "root"
  max_command_timeout: 300s
EOF

# Test run
sudo /usr/local/bin/ocserv-agent-linux-amd64 -config /etc/ocserv-agent/config.yaml

# Should auto-generate certs and start successfully
# Press Ctrl+C after verifying

# Verify certs were created
sudo ls -lh /etc/ocserv-agent/certs/
```

## ✅ Success Criteria

All tests should pass:

- [ ] `ocserv-agent version` shows version
- [ ] `ocserv-agent help` shows usage
- [ ] `ocserv-agent gencert` creates certificates
- [ ] Auto-generation creates certs on first run
- [ ] Auto-generated certs have correct permissions (644/600)
- [ ] Certificate chain validates
- [ ] Agent starts with auto-generated certs
- [ ] Agent fails appropriately when auto_generate=false and certs missing
- [ ] Production deployment to /etc/ocserv-agent works

## 🐛 Troubleshooting

### Permission Denied on /etc/ocserv-agent

```bash
# Check ownership
sudo ls -ld /etc/ocserv-agent/certs/

# Fix if needed
sudo chown -R root:root /etc/ocserv-agent
sudo chmod 755 /etc/ocserv-agent
sudo chmod 755 /etc/ocserv-agent/certs
```

### Certs Not Generated

```bash
# Check auto_generate is true
grep auto_generate /etc/ocserv-agent/config.yaml

# Check directory is writable
sudo test -w /etc/ocserv-agent/certs && echo "Writable" || echo "Not writable"

# Try manual generation
sudo /usr/local/bin/ocserv-agent-linux-amd64 gencert -output /etc/ocserv-agent/certs
```

### Agent Won't Start

```bash
# Check detailed error
sudo /usr/local/bin/ocserv-agent-linux-amd64 -config /etc/ocserv-agent/config.yaml 2>&1 | tee /tmp/agent-error.log

# Validate config syntax
cat /etc/ocserv-agent/config.yaml | grep -E "(auto_generate|cert_file|key_file|ca_file)"

# Check SELinux (if applicable)
getenforce
# If enforcing, may need to adjust contexts
```

## 📝 Report Results

After testing, report:

1. Which tests passed/failed
2. Any error messages
3. Certificate fingerprints generated
4. Screenshot or log of successful startup

## 🔄 Cleanup

```bash
# Remove test files
rm -rf /tmp/test-certs /tmp/auto-certs
rm /tmp/test-config*.yaml
rm /tmp/ocserv-agent

# Keep production deployment if successful
# Or remove to test again:
# sudo rm -rf /etc/ocserv-agent
# sudo rm /usr/local/bin/ocserv-agent-linux-amd64
```

## 📚 Next Steps

After successful testing:

1. Update systemd service to use new binary
2. Restart service with systemctl
3. Monitor logs for any issues
4. Plan migration to CA-signed certs for production control server integration

## 🔗 References

- [Certificate Management Guide](docs/CERTIFICATES.md)
- [Configuration Example](config.yaml.example)
