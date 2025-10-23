# Release vX.Y.Z

**Release Date:** YYYY-MM-DD
**Type:** MAJOR/MINOR/PATCH - Brief description
**Status:** ALPHA/BETA/STABLE

## 📋 Summary

Brief summary of the main changes in this release (1-3 sentences).

## 🎯 Key Features

### Feature Name 1

**Problem Solved:** Description of the problem this feature addresses.

**Solution:** Brief explanation of how the feature solves the problem.

#### Feature Details
- Implementation detail 1
- Implementation detail 2
- Implementation detail 3

### Feature Name 2

**Problem Solved:** Description of the problem.

**Solution:** How it's solved.

## 📦 What's Changed

<details>
<summary><strong>Click to expand detailed changes</strong></summary>

### New Features

**Component Name**
- Feature description with implementation details
- Another feature
- Technical specifications

**Another Component**
- Feature A
- Feature B

### Documentation

**New Guides:**
- `docs/GUIDE_NAME.md` - Description
  - What's covered
  - Use cases
  - Key sections

**Updated:**
- Updated documentation topic 1
- Updated documentation topic 2

### Code Changes

**Files Added:**
- `path/to/new/file.go` - Description
- `docs/NEW_GUIDE.md` - Guide description

**Files Modified:**
- `path/to/modified/file.go` - Changes description
- `config.yaml.example` - Updated configuration options

### Bug Fixes

**Issue Type:**
- Fix description and impact
- Related issues/PRs

**Security:**
- Security fix description
- Impact and remediation

</details>

## 🚀 Usage Examples

<details>
<summary><strong>Click to expand usage examples</strong></summary>

### Quick Start

```bash
# Installation steps
sudo mkdir -p /etc/ocserv-agent
sudo cp ocserv-agent /etc/ocserv-agent/
sudo chmod +x /etc/ocserv-agent/ocserv-agent

# Configuration
cat > /etc/ocserv-agent/config.yaml <<EOF
# Configuration example
EOF

# Start
sudo ocserv-agent -config /etc/ocserv-agent/config.yaml
```

**Output:**
```
Expected output showing the feature working
```

### Advanced Usage

```bash
# Advanced example commands
command --with-options
```

### Download and Extract

```bash
# Download archive
wget https://github.com/dantte-lp/ocserv-agent/releases/download/vX.Y.Z/ocserv-agent-vX.Y.Z-linux-amd64.tar.gz

# Verify checksum
wget https://github.com/dantte-lp/ocserv-agent/releases/download/vX.Y.Z/ocserv-agent-vX.Y.Z-linux-amd64.tar.gz.sha256
sha256sum -c ocserv-agent-vX.Y.Z-linux-amd64.tar.gz.sha256

# Extract
tar -xzf ocserv-agent-vX.Y.Z-linux-amd64.tar.gz

# Install
sudo mkdir -p /etc/ocserv-agent
sudo mv ocserv-agent /etc/ocserv-agent/
sudo chmod +x /etc/ocserv-agent/ocserv-agent
```

</details>

## 🔒 Security

### Security Improvements

**Feature Name:**
- ✅ Security improvement 1
- ✅ Security improvement 2
- ⚠️ Known limitation

### Build Security

- **SLSA Level 3** provenance for all releases
- **SHA256 checksums** for verification
- **Multi-platform** reproducible builds
- **No embedded secrets** in binaries

## 📊 Statistics

### Code Changes
- **Files changed:** XX
- **Lines added:** ~X,XXX
- **Lines removed:** ~XXX
- **Net change:** +X,XXX lines

### Commits Since vX.Y-1.Z
- XX commits
- X features
- X improvements
- X bugfixes
- X documentation updates

## 🔍 Compatibility

**Breaking Changes:** None / Description of breaking changes

**New Requirements:**
- Go X.YZ+ for development (was X.YZ+)
- Other requirement changes

**Dependencies:**
- Go: X.YZ (toolchain: goX.YZ.Z)
- gRPC: vX.YZ.Z (updated from vX.YZ-1.Z)
- protobuf: vX.YZ.Z (no change)
- zerolog: vX.YZ.Z (no change)

**Binary Compatibility:**
- ✅ Fully compatible with vX.Y-1.Z
- ✅ Config file compatible (new optional fields)
- ✅/⚠️ API changes if any

## 🚀 Deployment

<details>
<summary><strong>Click to expand deployment guide</strong></summary>

### Upgrade from vX.Y-1.Z

```bash
# 1. Download new version
wget https://github.com/dantte-lp/ocserv-agent/releases/download/vX.Y.Z/ocserv-agent-vX.Y.Z-linux-amd64.tar.gz
tar -xzf ocserv-agent-vX.Y.Z-linux-amd64.tar.gz

# 2. Backup current binary
sudo cp /etc/ocserv-agent/ocserv-agent /etc/ocserv-agent/ocserv-agent.vX.Y-1.Z

# 3. Install new version
sudo cp ocserv-agent /etc/ocserv-agent/
sudo chmod +x /etc/ocserv-agent/ocserv-agent

# 4. Update configuration if needed
# Review config changes in config.yaml.example

# 5. Restart service
sudo systemctl restart ocserv-agent
```

### Fresh Install

```bash
# Extract and install
tar -xzf ocserv-agent-vX.Y.Z-linux-amd64.tar.gz
sudo mkdir -p /etc/ocserv-agent
sudo mv ocserv-agent /etc/ocserv-agent/
sudo chmod +x /etc/ocserv-agent/ocserv-agent

# Create config
sudo cp config.yaml.example /etc/ocserv-agent/config.yaml
# Edit config as needed

# Start agent
sudo ocserv-agent -config /etc/ocserv-agent/config.yaml
```

</details>

## 📝 Full Changelog

<details>
<summary><strong>Click to expand full changelog</strong></summary>

### Features
- Feature description with commit hash (#abc1234)
- Another feature (#def5678)

### Bug Fixes
- Fix description (#ghi9012)

### Documentation
- Documentation update (#jkl3456)

### Build
- Build system update (#mno7890)

</details>

## 🐛 Known Issues

List any known issues or limitations in this release.

## 🔮 Next Steps

See [TODO](../todo/CURRENT.md) for upcoming features.

**Planned for vX.Y+1.Z:**
- Feature 1
- Feature 2
- Feature 3

## 📚 References

- [Guide 1](../GUIDE1.md)
- [Guide 2](../GUIDE2.md)
- [Contributing Guide](../../.github/CONTRIBUTING.md)
- [Workflows Documentation](../../.github/WORKFLOWS.md)
- [Previous Release (vX.Y-1.Z)](vX.Y-1.Z.md)

## 🔗 Downloads

<details>
<summary><strong>Click to expand download links and verification</strong></summary>

**Release Assets:**
- `ocserv-agent-vX.Y.Z-linux-amd64.tar.gz` - Linux x86_64
- `ocserv-agent-vX.Y.Z-linux-arm64.tar.gz` - Linux ARM64/aarch64
- `ocserv-agent-vX.Y.Z-freebsd-amd64.tar.gz` - FreeBSD x86_64
- `ocserv-agent-vX.Y.Z-freebsd-arm64.tar.gz` - FreeBSD ARM64/aarch64
- `*.sha256` - SHA256 checksums
- `*.intoto.jsonl` - SLSA provenance

**SLSA Verification:**
```bash
# Install slsa-verifier
go install github.com/slsa-framework/slsa-verifier/v2/cli/slsa-verifier@latest

# Verify binary
slsa-verifier verify-artifact \
  --provenance-path ocserv-agent-vX.Y.Z-linux-amd64.intoto.jsonl \
  --source-uri github.com/dantte-lp/ocserv-agent \
  ocserv-agent-vX.Y.Z-linux-amd64.tar.gz
```

</details>

---

**Full Diff:** [vX.Y-1.Z...vX.Y.Z](https://github.com/dantte-lp/ocserv-agent/compare/vX.Y-1.Z...vX.Y.Z)

**GitHub Release:** [vX.Y.Z](https://github.com/dantte-lp/ocserv-agent/releases/tag/vX.Y.Z)
