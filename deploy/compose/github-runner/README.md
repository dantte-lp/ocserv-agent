# GitHub Actions Self-Hosted Runner

Containerized GitHub Actions runner for ocserv-agent project on Oracle Linux 10.

## Quick Start

### 1. Get Registration Token

```bash
make runner-token
```

Or manually:
```bash
gh api --method POST \
  /repos/dantte-lp/ocserv-agent/actions/runners/registration-token \
  --jq '.token'
```

### 2. Start Runner

```bash
RUNNER_TOKEN="YOUR_TOKEN_HERE" make runner-up
```

### 3. Check Status

```bash
make runner-logs
```

### 4. Stop Runner

```bash
make runner-down
```

## Installed Tools & Versions

### Languages & Runtimes
- **Python 3.14.0** - Built from source with optimizations
- **Poetry 2.2** - Python dependency management
- **Go 1.25.1** - With protoc-gen-go and protoc-gen-go-grpc
- **.NET SDK 8.0** - For GitHub Actions runner
- **Protocol Buffers 29.3** - protoc compiler

### Build Tools
- **GCC/G++** - C/C++ compilers
- **Make, Automake, Autoconf** - Build automation
- **RPM tools** - rpm-build, rpmdevtools, rpmlint
- **Mock** - RPM package building in chroot

### Container Tools
- **Podman** - Container runtime
- **Buildah** - Container image builder
- **Skopeo** - Container image management

### Security & Analysis
- **Trivy** - Container and dependency scanner

### Version Control
- **Git** - With git-lfs support

## Features

- ✅ Oracle Linux 10 base (latest stable, .NET 8.0 support)
- ✅ Full ocserv-agent build environment
- ✅ Ansible deployment environment (Python 3.14 + Poetry)
- ✅ Protocol Buffers code generation
- ✅ RPM package building with mock
- ✅ Container image building (Podman/Buildah)
- ✅ Automatic runner registration/deregistration
- ✅ Graceful shutdown with cleanup
- ✅ Persistent work directory
- ✅ Project mounted at /workspace

## Makefile Targets

Add to main Makefile:

```makefile
# GitHub Actions Runner
.PHONY: runner-up runner-down runner-logs runner-token

runner-token:
	@gh api --method POST /repos/dantte-lp/ocserv-agent/actions/runners/registration-token --jq '.token'

runner-up:
	@echo "🏃 Starting GitHub Actions runner..."
	cd deploy/compose && podman-compose -f github-runner.yml up -d
	@echo "✅ Runner started. Check logs: make runner-logs"

runner-down:
	@echo "🛑 Stopping GitHub Actions runner..."
	cd deploy/compose && podman-compose -f github-runner.yml down

runner-logs:
	@podman logs -f ocserv-agent-github-runner
```

## Workflow Configuration

Update workflows to use self-hosted runner:

```yaml
jobs:
  test:
    runs-on: self-hosted
    # Or specific labels:
    # runs-on: [self-hosted, linux, x64, podman]
```

## Troubleshooting

### Runner not appearing in GitHub

1. Check logs: `podman logs ocserv-agent-github-runner`
2. Verify token is valid (expires after 1 hour)
3. Get new token and restart

### Runner stuck

```bash
podman-compose -f github-runner.yml down
podman-compose -f github-runner.yml up -d
```

### Clean restart

```bash
podman-compose -f github-runner.yml down -v  # Remove volumes
podman-compose -f github-runner.yml up -d
```

## Security Notes

- Runner has sudo access (for package installation)
- Runs in isolated container
- `/workspace` mounted read-write for builds
- No Docker socket mounted by default (add if needed)

## References

- [GitHub Actions Runner](https://github.com/actions/runner)
- [Self-Hosted Runners](https://docs.github.com/en/actions/hosting-your-own-runners)
