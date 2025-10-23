# Ansible Automation for ocserv-agent

Ansible playbooks and roles for deploying and managing ocserv-agent on remote servers.

## Setup

### 1. Install Dependencies

The Ansible environment runs in a Podman container:

```bash
# Start Ansible container
podman-compose -f deploy/compose/ansible.yml up -d

# Verify installation
podman exec ocserv-agent-ansible poetry run ansible --version

# Verify collections installed
podman exec ocserv-agent-ansible poetry run ansible-galaxy collection list
```

**Installed Collections:**
- `ansible.posix` (>= 1.5.0) - firewalld, authorized_key, mount, sysctl
- `community.general` (>= 8.0.0) - systemd_info, various utilities
- `community.docker` (>= 3.4.0) - Docker/Podman support

### 2. Configure Credentials

Copy `.env.example` to `.env` and fill in your server details:

```bash
cp .env.example .env
```

Edit `.env`:

```bash
# Example (replace with your actual server)
REMOTE_HOST=203.0.113.10
REMOTE_USER=root
REMOTE_PASSWORD=your_secure_password
```

**Security Notes:**
- `.env` file is in `.gitignore` and will NOT be committed to git
- Use SSH keys instead of passwords (recommended)
- For production, use Ansible Vault for sensitive data

### 3. Configure SSH Key Authentication (Recommended)

Instead of using passwords, configure SSH key authentication:

```bash
# Generate SSH key if you don't have one
ssh-keygen -t ed25519 -C "ocserv-agent-deploy"

# Copy public key to remote server
ssh-copy-id -i ~/.ssh/id_ed25519.pub root@${REMOTE_HOST}

# Leave REMOTE_PASSWORD empty in .env
```

## Usage

### Run Playbooks

**Before deployment, verify ocserv status:**

```bash
# Verify ocserv installation and status
podman exec ocserv-agent-ansible poetry run ansible-playbook playbooks/verify-ocserv.yml
```

**Setup deployment user (one-time):**

```bash
# Setup test user with certificate authentication
podman exec ocserv-agent-ansible poetry run ansible-playbook playbooks/setup-test-user.yml
```

**Deploy new agent version:**

```bash
# Build agent first
make compose-build

# Deploy to production
podman exec ocserv-agent-ansible poetry run ansible-playbook playbooks/deploy-agent.yml
```

### TLS Configuration

The deployed agent requires TLS certificates. Choose one of these options:

**Option 1: Auto-Generated Certificates (Testing/Initial Setup)**

The default config uses auto-generation:

```yaml
tls:
  enabled: true
  auto_generate: true  # Agent will generate self-signed certs on first run
```

On first start, the agent will:
1. Generate self-signed CA certificate
2. Generate agent certificate signed by CA
3. Create private key
4. Display fingerprints for verification

**Option 2: Manual Certificate Generation**

Generate certificates before starting the agent:

```bash
# On remote server (via SSH or Ansible)
sudo /usr/sbin/ocserv-agent gencert -output /etc/ocserv-agent/certs

# Update config to disable auto-generation
tls:
  auto_generate: false
```

**Option 3: CA-Signed Certificates (Production)**

For production with a control server:

1. Obtain CA certificate from control server
2. Generate and sign agent certificate with control server's CA
3. Copy certificates to `/etc/ocserv-agent/certs/`:
   - `agent.crt` - Agent certificate
   - `agent.key` - Private key (permissions: 0600)
   - `ca.crt` - CA certificate

4. Update config:

```yaml
tls:
  enabled: true
  auto_generate: false  # Use existing certificates
  cert_file: "/etc/ocserv-agent/certs/agent.crt"
  key_file: "/etc/ocserv-agent/certs/agent.key"
  ca_file: "/etc/ocserv-agent/certs/ca.crt"
  server_name: "control-server"  # Expected CN in server cert
  min_version: "TLS1.3"
```

📚 **See [docs/CERTIFICATES.md](../../docs/CERTIFICATES.md) for complete certificate management guide.**

**Rollback if needed:**

```bash
# Rollback to previous version
podman exec ocserv-agent-ansible poetry run ansible-playbook playbooks/rollback-agent.yml
```

### Interactive Shell

```bash
# Enter Ansible container
podman exec -it ocserv-agent-ansible bash

# Inside container:
poetry run ansible --version
poetry run ansible-playbook playbooks/example.yml
```

## Directory Structure

```
deploy/ansible/
├── ansible.cfg          # Ansible configuration
├── pyproject.toml       # Poetry dependencies (Ansible 12.1.0)
├── inventory/           # Server inventories
│   └── production.yml   # Production servers
├── playbooks/           # Ansible playbooks
│   ├── setup-test-user.yml
│   ├── verify-ocserv.yml
│   └── deploy-agent.yml
└── roles/               # Ansible roles
    └── test-user/
        └── tasks/
            └── main.yml
```

## Safety Measures

**CRITICAL: Production Server Safety**

The target server has:
- **Active VPN users** connected to ocserv
- **Production ocserv 1.3** running
- **Existing agent** (v0.3.0-24-groutes)

**Before deploying:**
1. ✅ Backup current agent binary
2. ✅ Backup current configuration
3. ✅ Test on staging environment first (if available)
4. ✅ Have rollback procedure ready
5. ✅ Monitor VPN connections during deployment

**Rollback Procedure:**
```bash
# Restore previous version
podman exec ocserv-agent-ansible poetry run ansible-playbook playbooks/rollback-agent.yml
```

## Development

### Lint Playbooks

```bash
podman exec ocserv-agent-ansible poetry run ansible-lint playbooks/
```

### Dry Run (Check Mode)

```bash
podman exec ocserv-agent-ansible poetry run ansible-playbook playbooks/deploy-agent.yml --check
```

## Troubleshooting

### Connection Issues

```bash
# Test SSH connection
ssh -i ~/.ssh/id_ed25519 ${REMOTE_USER}@${REMOTE_HOST}

# Test Ansible connectivity
podman exec ocserv-agent-ansible poetry run ansible all -m ping
```

### View Logs

```bash
# Ansible logs are in ansible.log
podman exec ocserv-agent-ansible tail -f ansible.log
```

## Security

**Credentials:**
- **Never commit** `.env` or credentials to git
- Use **SSH keys** (ed25519) with strong passphrases
- Use **Ansible Vault** to encrypt sensitive variables in production
- Create dedicated deployment user (not root) with minimal privileges

**TLS Certificates:**
- **Development:** Auto-generated self-signed certs are OK for initial setup
- **Production:** Use CA-signed certificates from your control server
- **Never share** private keys or copy over insecure channels
- **Verify fingerprints** when using self-signed certificates
- **Monitor expiration:** Self-signed certs are valid for 1 year

**Audit & Monitoring:**
- All deployments are logged in `ansible.log`
- Monitor agent status: `systemctl status ocserv-agent`
- Check agent logs: `journalctl -u ocserv-agent -f`
- Verify VPN service unchanged: `systemctl status ocserv`

## References

- [Ansible Documentation](https://docs.ansible.com/)
- [Poetry Documentation](https://python-poetry.org/docs/)
- [Ansible Best Practices](https://docs.ansible.com/ansible/latest/user_guide/playbooks_best_practices.html)
