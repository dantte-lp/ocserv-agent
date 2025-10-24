# GitHub Wiki Migration Plan

## Overview

Migrate user-facing documentation from `/docs` to GitHub Wiki for better discoverability and user experience, while keeping technical/security documentation in the repository.

## Why GitHub Wiki?

### Advantages
- ✅ **Better UX** - Sidebar navigation, search, history
- ✅ **Discoverability** - Easier for users to find guides
- ✅ **Editing** - Non-developers can contribute via web UI
- ✅ **Organization** - Natural hierarchical structure
- ✅ **Accessibility** - Public without cloning repository
- ✅ **SEO** - Better indexing for user guides

### Repository vs Wiki Decision Matrix

| Content Type | Location | Reason |
|--------------|----------|--------|
| **Installation guides** | Wiki | User-facing, frequently accessed |
| **Configuration guides** | Wiki | User-facing, OS-specific |
| **Troubleshooting** | Wiki | User-facing, searchable |
| **FAQ** | Wiki | User-facing |
| **Architecture overview** | Wiki | High-level, user-friendly |
| **API documentation** | Repo | Technical, versioned with code |
| **Security documentation** | Repo | Audit requirements, immutable |
| **Development guides** | Repo | Developer-specific, versioned |
| **CI/CD documentation** | Repo | Technical, tightly coupled with code |
| **Contributing guidelines** | Repo | Pull request workflow context |
| **Release notes** | Repo | Version control, changelog |

## Proposed Wiki Structure

### Home
- Project overview
- Quick start guide
- Links to key resources
- Latest release information

### User Documentation

#### 1. Installation
- **Installation Overview** (`docs/LOCAL_TESTING.md` → Wiki)
  - System requirements
  - Supported platforms
- **Installing from Packages**
  - RPM installation (RHEL/Oracle/Rocky)
  - DEB installation (Debian/Ubuntu)
  - FreeBSD installation
- **Installing from Source**
  - Prerequisites
  - Build instructions
  - Manual installation
- **Docker/Container Installation**
  - Docker Compose setup
  - Podman setup

#### 2. Configuration
- **Configuration Guide** (extract from README)
  - config.yaml reference
  - Environment variables
  - Command-line options
- **Certificate Management** (`docs/CERTIFICATES.md` → Wiki)
  - Generating certificates
  - Certificate rotation
  - TLS configuration
- **Platform-Specific Configuration**
  - SELinux configuration (RHEL/Oracle)
  - AppArmor (Ubuntu/Debian)
  - FreeBSD rc.conf

#### 3. Usage & Operations
- **Getting Started**
  - First-time setup
  - Basic operations
  - Testing connectivity
- **gRPC API Guide** (`docs/GRPC_TESTING.md` → Wiki)
  - Available RPCs
  - grpcurl examples
  - Authentication
- **occtl Commands** (`docs/OCCTL_COMMANDS.md` → Wiki)
  - Command reference
  - Examples
  - Best practices

#### 4. Troubleshooting
- **Common Issues** (new)
  - Connection problems
  - Permission errors
  - Certificate issues
- **Platform-Specific Issues**
  - SELinux denials
  - Systemd problems
  - Network configuration
- **Debugging Guide** (new)
  - Log analysis
  - Verbose mode
  - Debug tools

#### 5. Architecture & Design
- **Architecture Overview** (new)
  - High-level design
  - Component diagram
  - Data flow
- **Security Architecture** (from `SECURITY_TOOLS.md`)
  - Authentication
  - Authorization
  - TLS/mTLS
  - Security layers

### Developer Documentation (stays in repo)

Keep in `/docs`:
- `SECURITY_TOOLS.md` - Security audit trail
- `PACKAGING.md` - Build documentation
- `OSSF_SCORECARD_IMPROVEMENTS.md` - Security compliance
- `CI_OPTIMIZATION.md` - CI/CD technical details
- `.github/WORKFLOWS.md` - Workflow documentation
- `todo/CURRENT.md` - Development tracking
- `releases/*.md` - Version control

Move to `.github/`:
- `CONTRIBUTING.md` (already there)
- `CODEOWNERS` (already there)

## Migration Plan

### Phase 1: Wiki Setup & Structure
**Effort:** 2 hours

1. **Enable GitHub Wiki**
   ```bash
   # Settings → Features → Wikis (enable)
   ```

2. **Clone wiki repository**
   ```bash
   git clone https://github.com/dantte-lp/ocserv-agent.wiki.git
   cd ocserv-agent.wiki
   ```

3. **Create initial structure**
   ```
   Home.md
   Installation/
     ├── Overview.md
     ├── From-Packages.md
     ├── From-Source.md
     └── Docker-Setup.md
   Configuration/
     ├── Overview.md
     ├── Certificates.md
     └── Platform-Specific.md
   Usage/
     ├── Getting-Started.md
     ├── gRPC-API.md
     └── occtl-Commands.md
   Troubleshooting/
     ├── Common-Issues.md
     └── Debugging.md
   Architecture/
     ├── Overview.md
     └── Security.md
   ```

### Phase 2: Content Migration
**Effort:** 4-6 hours

#### 2.1 High Priority (User-facing)
1. **Home page** (new, 30 min)
   - Project description
   - Quick start
   - Navigation guide

2. **Installation guides** (2 hours)
   - Extract from `docs/PACKAGING.md` → Wiki/Installation/
   - Platform-specific guides
   - Prerequisites and dependencies

3. **Certificate management** (30 min)
   - `docs/CERTIFICATES.md` → Wiki/Configuration/Certificates.md
   - Minor formatting updates

4. **gRPC & occtl guides** (1 hour)
   - `docs/GRPC_TESTING.md` → Wiki/Usage/gRPC-API.md
   - `docs/OCCTL_COMMANDS.md` → Wiki/Usage/occtl-Commands.md
   - Add examples and best practices

#### 2.2 Medium Priority (Guides)
5. **Configuration guide** (1.5 hours)
   - Extract from README
   - Create comprehensive config reference
   - Platform-specific sections

6. **Troubleshooting** (1 hour)
   - New content based on common issues
   - Platform-specific problems
   - Debug techniques

#### 2.3 Low Priority (Architecture)
7. **Architecture overview** (1 hour)
   - High-level design
   - Simplified from technical docs
   - Diagrams (optional)

### Phase 3: Repository Cleanup
**Effort:** 1 hour

1. **Update README.md**
   - Add prominent link to Wiki
   - Simplified quick start
   - Point to Wiki for detailed guides

2. **Add deprecation notices**
   ```markdown
   # docs/CERTIFICATES.md
   > **Note:** This guide has moved to the [Wiki](https://github.com/dantte-lp/ocserv-agent/wiki/Configuration/Certificates).
   > This file will be removed in v0.8.0.
   ```

3. **Update navigation**
   - Update links in remaining docs
   - Update `.github/WORKFLOWS.md`

### Phase 4: Maintenance & Monitoring
**Effort:** Ongoing

1. **Set up wiki protection** (if available)
2. **Monitor for vandalism** (GitHub wikis are public editable)
3. **Establish update process**
   - Wiki updates with each release
   - Documentation review in PR checklist

## Migration Checklist

### Before Migration
- [ ] Enable GitHub Wiki in repository settings
- [ ] Clone wiki repository locally
- [ ] Create wiki structure (directories, template pages)
- [ ] Set up navigation sidebar

### Content Migration
- [ ] Home page (new)
- [ ] Installation → From Packages (from PACKAGING.md)
- [ ] Installation → From Source (from README)
- [ ] Installation → Docker Setup (from compose files)
- [ ] Configuration → Overview (new)
- [ ] Configuration → Certificates (from CERTIFICATES.md)
- [ ] Configuration → Platform-Specific (from PACKAGING.md)
- [ ] Usage → Getting Started (new)
- [ ] Usage → gRPC API (from GRPC_TESTING.md)
- [ ] Usage → occtl Commands (from OCCTL_COMMANDS.md)
- [ ] Troubleshooting → Common Issues (new)
- [ ] Troubleshooting → Debugging (new)
- [ ] Architecture → Overview (new)
- [ ] Architecture → Security (from SECURITY_TOOLS.md extract)

### Repository Updates
- [ ] Update README.md with Wiki link
- [ ] Add deprecation notices to migrated docs
- [ ] Update internal documentation links
- [ ] Remove migrated files (in v0.8.0)

### Quality Assurance
- [ ] All Wiki links work
- [ ] Navigation is intuitive
- [ ] Search finds relevant content
- [ ] Mobile-friendly formatting
- [ ] Code blocks render correctly
- [ ] Images display properly

## Files to Keep in Repository

### Must Stay (Security/Compliance)
- `docs/SECURITY_TOOLS.md` - Audit trail
- `docs/PACKAGING.md` - Build documentation
- `docs/OSSF_SCORECARD_IMPROVEMENTS.md` - Compliance tracking
- `docs/CI_OPTIMIZATION.md` - CI/CD technical details
- `.github/WORKFLOWS.md` - Workflow documentation

### Must Stay (Version Control)
- `docs/releases/*.md` - Release notes
- `docs/todo/*.md` - Development tracking
- `CHANGELOG.md` - Version history

### Must Stay (Development)
- `.github/CONTRIBUTING.md` - PR workflow
- `.github/CODEOWNERS` - Code review
- `docs/LOCAL_TESTING.md` - Developer testing

## Benefits After Migration

### For Users
- ✅ Easier to find installation guides
- ✅ Better navigation and search
- ✅ Platform-specific instructions clearly separated
- ✅ No need to browse repository structure

### For Developers
- ✅ Cleaner `/docs` directory
- ✅ Clear separation of concerns
- ✅ Less documentation in code reviews
- ✅ Focus on technical docs

### For Project
- ✅ Professional documentation structure
- ✅ Better SEO and discoverability
- ✅ Easier community contributions
- ✅ Reduced repository size

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Wiki vandalism | Monitor changes, revert if needed |
| Link rot (external links to old docs) | Keep deprecation notices, redirects in README |
| Search engines cache old docs | Add robots.txt, update sitemap |
| Contributors edit wrong location | Clear docs in CONTRIBUTING.md |
| Wiki and repo get out of sync | Update process in release checklist |

## Timeline

**Total Effort:** 8-10 hours

- **Week 1:** Phase 1 (Setup) + Phase 2.1 (High Priority) - 4.5 hours
- **Week 2:** Phase 2.2-2.3 (Medium/Low Priority) - 3 hours
- **Week 3:** Phase 3 (Cleanup) + testing - 1.5 hours
- **Ongoing:** Phase 4 (Maintenance)

**Target Completion:** End of November 2025

## References

- [GitHub Wiki Documentation](https://docs.github.com/en/communities/documenting-your-project-with-wikis)
- [Best Practices for Documentation](https://www.writethedocs.org/guide/)
- [Documentation as Code](https://www.writethedocs.org/guide/docs-as-code/)

## Decision

**Recommendation:** ✅ **Proceed with Wiki migration**

The benefits significantly outweigh the risks, especially for user-facing documentation. The phased approach allows gradual migration with minimal disruption.

**Next Steps:**
1. Enable Wiki in repository settings
2. Create initial structure (Phase 1)
3. Migrate high-priority content (Phase 2.1)
4. Evaluate and continue based on user feedback
