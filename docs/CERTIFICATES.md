# Certificate Management Guide

This guide explains how to manage TLS certificates for ocserv-agent.

## 📋 Table of Contents

- [Overview](#overview)
- [Bootstrap Mode (Self-Signed)](#bootstrap-mode-self-signed)
- [Production Mode (CA-Signed)](#production-mode-ca-signed)
- [CLI Commands](#cli-commands)
- [Auto-Generation](#auto-generation)
- [Security Considerations](#security-considerations)

## Overview

ocserv-agent supports two certificate modes:

1. **Bootstrap Mode** - Self-signed certificates for autonomous operation
2. **Production Mode** - CA-signed certificates for control server integration

## Bootstrap Mode (Self-Signed)

### Automatic Generation

The simplest way is to enable `auto_generate` in your config:

```yaml
tls:
  enabled: true
  auto_generate: true  # Automatically generate if missing
  cert_file: "/etc/ocserv-agent/certs/agent.crt"
  key_file: "/etc/ocserv-agent/certs/agent.key"
  ca_file: "/etc/ocserv-agent/certs/ca.crt"
  min_version: "TLS1.3"
```

When you start the agent, it will:
1. Check if certificates exist
2. If missing, generate self-signed CA + certificate + key
3. Display certificate fingerprints for verification
4. Start normally

**Example output:**

```
🔐 Generated self-signed certificates for bootstrap mode
   CA Fingerprint:   SHA256:a1:b2:c3:d4:...
   Cert Fingerprint: SHA256:e5:f6:g7:h8:...
   Subject:          ocserv-agent-vpn-server-01
   Valid:            2025-10-23 - 2026-10-23
   Location:         /etc/ocserv-agent/certs

⚠️  These are self-signed certificates for autonomous operation.
   To connect to a control server, replace with CA-signed certificates:
   - Use: ocserv-agent gencert --ca /path/to/server-ca.crt
```

### Manual Generation

You can also generate certificates manually:

```bash
# Generate in default location
sudo ocserv-agent gencert

# Generate in custom location
sudo ocserv-agent gencert -output /custom/path

# Generate with specific hostname
sudo ocserv-agent gencert -hostname vpn.example.com -output /etc/ocserv-agent/certs
```

## Production Mode (CA-Signed)

For production use with a control server, you need CA-signed certificates.

### Step 1: Disable Auto-Generation

Update your config:

```yaml
tls:
  enabled: true
  auto_generate: false  # Disable auto-gen for production
  cert_file: "/etc/ocserv-agent/certs/agent.crt"
  key_file: "/etc/ocserv-agent/certs/agent.key"
  ca_file: "/etc/ocserv-agent/certs/ca.crt"
  server_name: "control-server"  # Expected CN in server cert
  min_version: "TLS1.3"
```

### Step 2: Obtain CA-Signed Certificates

**Option A: Use control server's CA**

1. Get CA certificate from control server
2. Generate CSR on agent
3. Sign CSR with control server's CA
4. Copy signed cert + CA to agent

**Option B: Replace self-signed certs** (not yet implemented)

```bash
# Future feature - generate cert signed by external CA
sudo ocserv-agent gencert --ca /path/to/server-ca.crt -output /etc/ocserv-agent/certs
```

### Step 3: Verify Certificate

```bash
# Check certificate details
openssl x509 -in /etc/ocserv-agent/certs/agent.crt -text -noout

# Verify certificate is signed by CA
openssl verify -CAfile /etc/ocserv-agent/certs/ca.crt \
  /etc/ocserv-agent/certs/agent.crt
```

## CLI Commands

### gencert - Generate Certificates

Generate self-signed certificates for bootstrap mode.

**Usage:**

```bash
ocserv-agent gencert [flags]
```

**Flags:**

- `-output <dir>` - Output directory (default: `/etc/ocserv-agent/certs`)
- `-hostname <name>` - Hostname for certificate (default: auto-detect)
- `-self-signed` - Generate self-signed certs (default: `true`)
- `-ca <path>` - Path to CA cert for signing (not implemented)

**Examples:**

```bash
# Generate with defaults
sudo ocserv-agent gencert

# Custom output directory
sudo ocserv-agent gencert -output /var/lib/ocserv-agent/certs

# Specific hostname
sudo ocserv-agent gencert -hostname vpn01.example.com
```

**Output:**

```
🔐 Generating self-signed certificates...
   Hostname:        vpn-server-01
   Output dir:      /etc/ocserv-agent/certs

✅ Certificates generated successfully!

Certificate Information:
   CA Fingerprint:   SHA256:a1:b2:c3:...
   Cert Fingerprint: SHA256:e5:f6:g7:...
   Subject:          ocserv-agent-vpn-server-01
   Valid From:       2025-10-23 12:00:00 UTC
   Valid Until:      2026-10-23 12:00:00 UTC

Files created:
   /etc/ocserv-agent/certs/ca.crt       - CA certificate
   /etc/ocserv-agent/certs/agent.crt    - Agent certificate
   /etc/ocserv-agent/certs/agent.key    - Agent private key

⚠️  These are self-signed certificates for autonomous operation.
   To connect to a control server, you'll need CA-signed certificates.
```

## Auto-Generation

### How It Works

When `auto_generate: true` in config:

1. **On startup**, agent checks if all 3 files exist:
   - `cert_file`
   - `key_file`
   - `ca_file`

2. **If any missing**, agent generates:
   - Self-signed CA certificate
   - Agent certificate signed by CA
   - Agent private key

3. **Files are created** with proper permissions:
   - CA cert: `0644` (world-readable)
   - Agent cert: `0644` (world-readable)
   - Private key: `0600` (owner-only)

4. **Certificate properties:**
   - Algorithm: ECDSA P-256
   - Validity: 1 year
   - Subject: `ocserv-agent-<hostname>`
   - SAN: `hostname`, `localhost`

### When to Use

**Use auto-generation when:**
- ✅ Testing or development
- ✅ Autonomous agent (no control server)
- ✅ First-time setup
- ✅ Quick deployment

**Don't use auto-generation when:**
- ❌ Production with control server
- ❌ Strict security requirements
- ❌ Need custom certificate attributes
- ❌ Compliance requirements

## Security Considerations

### Self-Signed Certificates

**Pros:**
- ✅ Easy setup - no CA infrastructure needed
- ✅ Still provides TLS 1.3 encryption
- ✅ Protects against passive eavesdropping
- ✅ Good for autonomous operation

**Cons:**
- ❌ No chain of trust
- ❌ Cannot verify agent identity
- ❌ MITM possible without fingerprint verification
- ❌ Not suitable for multi-agent management

### CA-Signed Certificates

**Pros:**
- ✅ Chain of trust to control server CA
- ✅ Agent identity verification
- ✅ MITM protection
- ✅ Proper for production deployments

**Cons:**
- ❌ Requires CA infrastructure
- ❌ More complex setup
- ❌ Certificate lifecycle management

### Best Practices

1. **Use auto-generation for initial setup**
   ```yaml
   auto_generate: true  # First deployment
   ```

2. **Verify CA fingerprint** when bootstrapping
   - Note the SHA256 fingerprint from first run
   - Verify via secure channel (SSH, console, etc.)

3. **Replace with CA-signed certs** for production
   ```yaml
   auto_generate: false  # Production with control server
   ```

4. **Rotate certificates regularly**
   - Self-signed: valid for 1 year
   - Re-generate before expiry

5. **Protect private keys**
   - Stored with `0600` permissions
   - Never share or copy over insecure channels

6. **Monitor expiration**
   ```bash
   # Check cert expiry
   openssl x509 -in /etc/ocserv-agent/certs/agent.crt -noout -enddate
   ```

## Troubleshooting

### "cert_file not found" Error

**Problem:** Agent fails to start with certificate not found error

**Solution:**

1. Enable auto-generation:
   ```yaml
   tls:
     auto_generate: true
   ```

2. Or generate manually:
   ```bash
   sudo ocserv-agent gencert -output /etc/ocserv-agent/certs
   ```

### "Permission Denied" on Cert Files

**Problem:** Agent cannot read certificate files

**Solution:**

```bash
# Fix permissions
sudo chown root:root /etc/ocserv-agent/certs/*
sudo chmod 644 /etc/ocserv-agent/certs/*.crt
sudo chmod 600 /etc/ocserv-agent/certs/*.key
```

### Certificates Expired

**Problem:** Certificates are past their validity period

**Solution:**

```bash
# Remove old certs
sudo rm -f /etc/ocserv-agent/certs/*

# Regenerate
sudo ocserv-agent gencert -output /etc/ocserv-agent/certs

# Or let auto-generate create new ones
# (with auto_generate: true in config)
sudo systemctl restart ocserv-agent
```

### "TLS Handshake Failed"

**Problem:** Agent cannot connect to control server

**Possible causes:**

1. **Mismatched CA** - Agent's CA doesn't match server's CA
   ```bash
   # Verify CA matches server's CA
   openssl x509 -in /etc/ocserv-agent/certs/ca.crt -text -noout
   ```

2. **Wrong server_name** - Certificate CN doesn't match
   ```yaml
   tls:
     server_name: "control-server"  # Must match server cert CN
   ```

3. **Expired certificates**
   ```bash
   # Check expiry
   openssl x509 -in /etc/ocserv-agent/certs/agent.crt -noout -dates
   ```

## Example Workflows

### Workflow 1: First-Time Setup (Auto)

```bash
# 1. Install agent
sudo cp ocserv-agent /usr/local/bin/

# 2. Create config with auto-generation
cat > /etc/ocserv-agent/config.yaml <<EOF
agent_id: "server-01"
control_server:
  address: "localhost:9090"
tls:
  enabled: true
  auto_generate: true  # Auto-generate on first run
  cert_file: "/etc/ocserv-agent/certs/agent.crt"
  key_file: "/etc/ocserv-agent/certs/agent.key"
  ca_file: "/etc/ocserv-agent/certs/ca.crt"
# ... other config
EOF

# 3. Start agent (certs auto-generated)
sudo ocserv-agent -config /etc/ocserv-agent/config.yaml
```

### Workflow 2: First-Time Setup (Manual)

```bash
# 1. Install agent
sudo cp ocserv-agent /usr/local/bin/

# 2. Generate certificates
sudo ocserv-agent gencert -output /etc/ocserv-agent/certs

# 3. Create config (auto-gen disabled)
cat > /etc/ocserv-agent/config.yaml <<EOF
agent_id: "server-01"
control_server:
  address: "localhost:9090"
tls:
  enabled: true
  auto_generate: false  # Certs already exist
  cert_file: "/etc/ocserv-agent/certs/agent.crt"
  key_file: "/etc/ocserv-agent/certs/agent.key"
  ca_file: "/etc/ocserv-agent/certs/ca.crt"
# ... other config
EOF

# 4. Start agent
sudo ocserv-agent -config /etc/ocserv-agent/config.yaml
```

### Workflow 3: Migrate to Production CA

```bash
# 1. Running with self-signed certs
# auto_generate: true in config

# 2. Get CA cert from control server
scp controlserver:/etc/control-server/ca.crt /tmp/

# 3. Generate CSR (manual process for now)
# TODO: Will be automated in future version

# 4. Replace certs
sudo cp /path/to/ca-signed-agent.crt /etc/ocserv-agent/certs/agent.crt
sudo cp /path/to/ca-signed-agent.key /etc/ocserv-agent/certs/agent.key
sudo cp /tmp/ca.crt /etc/ocserv-agent/certs/ca.crt

# 5. Update config
# Change auto_generate: false
# Add server_name

# 6. Restart agent
sudo systemctl restart ocserv-agent
```

## Future Enhancements

The following features are planned for future releases:

- [ ] CA-signed certificate generation (`gencert --ca`)
- [ ] Certificate renewal automation
- [ ] CSR generation command
- [ ] Certificate rotation without restart
- [ ] ACME/Let's Encrypt integration
- [ ] Hardware security module (HSM) support
- [ ] Certificate revocation checking (CRL/OCSP)

## References

- [TLS 1.3 RFC](https://datatracker.ietf.org/doc/html/rfc8446)
- [X.509 Certificate Format](https://datatracker.ietf.org/doc/html/rfc5280)
- [mTLS Best Practices](https://www.cloudflare.com/learning/access-management/what-is-mutual-tls/)
