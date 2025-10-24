# Docker/Podman Compose Environments

This directory contains Docker Compose / Podman Compose configurations for various development, testing, and deployment scenarios.

## Available Environments

### Development

**docker-compose.dev.yml** - Hot-reload development environment
```bash
make compose-dev
# Or manually:
podman-compose -f deploy/compose/docker-compose.dev.yml up
```

**Features**:
- Hot reload with Air
- Mock control server for testing
- Mock ocserv for VPN testing
- Redis for caching

### Testing

**docker-compose.test.yml** - Integration testing environment
```bash
make compose-test
```

**Features**:
- Automated test execution
- Mock ocserv with test fixtures
- Coverage reporting

### Building

**docker-compose.build.yml** - Multi-architecture build environment
```bash
make compose-build
```

**Features**:
- Cross-compilation for multiple architectures
- RPM/DEB package creation
- Binary artifact generation

### Security Scanning

**security.yml** - Security scanning and vulnerability analysis
```bash
make compose-security
```

**Features**:
- Multiple security scanners (Trivy, govulncheck, etc.)
- SBOM generation
- Vulnerability reporting

### Deployment

**ansible.yml** - Ansible-based deployment automation
```bash
make compose-ansible
```

**Features**:
- Automated deployment to remote servers
- Configuration management
- Health checks

### Testing Infrastructure

**mock-ocserv.yml** - Mock OpenConnect server for testing
```bash
podman-compose -f deploy/compose/mock-ocserv.yml up
```

**Features**:
- Simulates ocserv behavior
- Test fixtures for integration tests
- No actual VPN server required

## GitHub Actions Self-Hosted Runners

**⚠️ Moved to separate repository**: The GitHub Actions self-hosted runners have been moved to a dedicated repository for better organization and reusability.

**New Location**: [self-hosted-runners](https://github.com/dantte-lp/self-hosted-runners) (or `/opt/projects/repositories/self-hosted-runners`)

**Why moved?**:
- Better separation of concerns (app vs infrastructure)
- Reusable across multiple projects
- Modern systemd quadlets instead of docker-compose
- RHEL 9+ best practices with Podman pods

**Migration**:
If you previously used `github-runner.yml` or `github-runner-debian.yml`:
1. Clone the new repository: https://github.com/dantte-lp/self-hosted-runners
2. Follow the setup guide: `docs/SETUP.md`
3. Runners now use systemd quadlets for auto-start on boot

## Environment Variables

All compose files source environment variables from `.env` file in this directory.

**Create from example**:
```bash
cp deploy/compose/.env.example deploy/compose/.env
vi deploy/compose/.env  # Edit as needed
```

## Common Tasks

### Start Development Environment
```bash
make compose-dev
```

### Run Tests
```bash
make compose-test
```

### Build Packages
```bash
make compose-build
```

### Security Scan
```bash
make compose-security
```

### Deploy to Server
```bash
make compose-ansible
```

## Cleanup

Remove all containers, volumes, and networks:
```bash
# Using make
make compose-down

# Or manually
podman-compose -f deploy/compose/docker-compose.dev.yml down -v
podman-compose -f deploy/compose/docker-compose.test.yml down -v
```

## Troubleshooting

### Podman Socket Issues

If you get socket connection errors:
```bash
# Enable Podman socket
sudo systemctl enable --now podman.socket

# Verify
ls -la /run/podman/podman.sock
```

### SELinux Permission Issues

If containers can't access volumes:
```bash
# Relabel volumes
sudo chcon -R -t container_file_t /opt/projects/repositories/ocserv-agent

# Or use :z flag in volume mounts (already configured)
```

### Port Conflicts

If ports are already in use:
```bash
# Check what's using the port
sudo ss -tulpn | grep :9090

# Stop conflicting service or change port in compose file
```

## Related Documentation

- [Local Testing Guide](../docs/LOCAL_TESTING.md)
- [CI/CD Workflows](../.github/WORKFLOWS.md)
- [Self-Hosted Runners](https://github.com/dantte-lp/self-hosted-runners)
- [Makefile Reference](../../Makefile)
