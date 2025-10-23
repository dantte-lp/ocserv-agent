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

- **Credentials:** Never commit `.env` or credentials to git
- **SSH Keys:** Use ed25519 keys with strong passphrases
- **Ansible Vault:** Encrypt sensitive variables in production
- **Least Privilege:** Create dedicated deployment user (not root)
- **Audit Logs:** All deployments are logged in `ansible.log`

## References

- [Ansible Documentation](https://docs.ansible.com/)
- [Poetry Documentation](https://python-poetry.org/docs/)
- [Ansible Best Practices](https://docs.ansible.com/ansible/latest/user_guide/playbooks_best_practices.html)
