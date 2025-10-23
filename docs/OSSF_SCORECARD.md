# OSSF Scorecard Report

**Last Updated:** 2025-10-24
**Current Score:** 6.6/10
**Target Score:** 7.5+/10

## Overview

This document tracks OSSF (Open Source Security Foundation) Scorecard improvements for ocserv-agent project.

The OSSF Scorecard is an automated tool that assesses open source projects on various security practices and provides a score out of 10.

**Official Scorecard:** https://api.scorecard.dev/projects/github.com/dantte-lp/ocserv-agent

## Current Scores

| Check | Score | Status | Notes |
|-------|-------|--------|-------|
| Binary-Artifacts | 10/10 | ✅ | No binary artifacts found |
| Branch-Protection | -1/10 → 9/10 | ✅ FIXED | Enabled enforce_admins |
| CI-Tests | -1/10 | 🔄 IN PROGRESS | Workflow exists, needs PR history |
| CII-Best-Practices | ? | ℹ️ | Not applicable for new projects |
| Code-Review | 0/10 | 🔄 IN PROGRESS | Enabled, needs PR history |
| Contributors | 0/10 | ℹ️ INFO | Single developer, cannot be improved artificially |
| Dangerous-Workflow | 10/10 | ✅ | No dangerous patterns detected |
| Dependency-Update-Tool | 0/10 | 📋 PLANNED | Will add Dependabot |
| Fuzzing | 0/10 | 📋 FUTURE | Planned for v1.0 |
| License | 10/10 | ✅ | MIT license present |
| Maintained | 0/10 | ℹ️ INFO | New project, requires time (90+ days) |
| Packaging | -1/10 | ℹ️ | Not applicable |
| Pinned-Dependencies | 0/10 | 📋 PLANNED | 73 dependencies to pin |
| SAST | 0/10 | 📋 PLANNED | Will add gosec/CodeQL |
| Security-Policy | 10/10 | ✅ | SECURITY.md present |
| Signed-Releases | 8/10 | 🔄 | v0.2.0 missing provenance |
| Token-Permissions | 10/10 | ✅ | Workflows use minimal permissions |
| Vulnerabilities | 9/10 | ✅ DOCUMENTED | PYSEC-2020-220 is false positive |

**Legend:**
- ✅ PASSED - Score 8-10, no action needed
- ✅ FIXED - Recently fixed, awaiting rescan
- 🔄 IN PROGRESS - Currently being worked on
- 📋 PLANNED - Scheduled for implementation
- ℹ️ INFO - Cannot be improved (informational only)

## Completed Improvements (2025-10-24)

### 1. Branch Protection ✅
**Before:** -1/10 (API error)
**After:** 9-10/10 (estimated)

**Changes:**
- Enabled `enforce_admins: true` (admins cannot bypass rules)
- Configured required PR reviews (1 approval)
- Enabled dismiss stale reviews
- Required status check: "CI Success"
- Linear history enabled
- Conversation resolution required

### 2. Vulnerabilities ✅
**Status:** 9/10 (documented as false positive)

**Analysis:**
- PYSEC-2020-220 (CVE-2020-25635) affects Ansible aws_ssm plugin
- Project does not use aws_ssm connection plugin
- ansible 12.1.0 (Oct 2025) is 4+ years newer than CVE (Oct 2020)
- Debian security tracker confirms regular Ansible not affected

**Documentation:** See `deploy/ansible/pyproject.toml`

### 3. Code Review ✅
**Status:** Enabled, awaiting PR history

**Changes:**
- Branch protection enforces PR reviews
- Required approving review count: 1
- Dismiss stale reviews: enabled

**Note:** OSSF shows 0/10 because all 30 commits were direct to main before protection was enabled. Score will improve with new PRs.

## Planned Improvements

### High Priority

#### Pinned-Dependencies (0 → 10/10)
**Impact:** +1.0 point

**Scope:** 73 unpinned dependencies:
- GitHub Actions (49+ dependencies, 22 unique actions)
- Docker images (golang:1.25-trixie, debian:trixie-slim)
- Go install commands

**Plan:**
1. Pin all GitHub Actions to SHA hashes
2. Pin Docker base images to digests (@sha256:...)
3. Setup Dependabot for automated updates

**Estimated effort:** 4-6 hours

#### CI-Tests (-1 → 10/10)
**Impact:** +1.1 points

**Status:** Workflow exists, needs PR execution

**Plan:**
1. Create test PR to demonstrate CI works
2. Ensure "CI Success" check passes
3. OSSF will detect PR with successful tests

**Estimated effort:** 30 minutes

### Medium Priority

#### SAST (0 → 10/10)
**Impact:** +1.0 point

**Plan:**
1. Add gosec (Go security scanner)
2. Add CodeQL analysis
3. Integrate into CI workflow

**Estimated effort:** 2-3 hours

#### Signed-Releases (8 → 10/10)
**Impact:** +0.2 points

**Current:**
- v0.3.0+: SLSA provenance ✅
- v0.2.0: Missing provenance ⚠️

**Plan:**
- Option A: Re-release v0.2.0 with provenance
- Option B: Accept minor deduction

**Estimated effort:** 1 hour (if re-releasing)

### Low Priority / Future

#### Dependency-Update-Tool (0 → 10/10)
**Impact:** +1.0 point

**Plan:**
- Configure Dependabot for Go modules
- Configure Dependabot for GitHub Actions
- Configure Dependabot for Docker images

**Estimated effort:** 1-2 hours

#### Fuzzing (0 → 10/10)
**Impact:** +1.0 point

**Status:** Planned for v1.0

**Plan:**
- Integrate with OSS-Fuzz or similar
- Add fuzz tests for critical paths
- Continuous fuzzing in CI

**Estimated effort:** 8-12 hours

### Informational Only

#### Contributors (0/10)
Cannot be artificially improved. Requires organic growth of contributor base from multiple organizations.

#### Maintained (0/10)
Cannot be artificially improved. Requires sustained development activity over 90+ days.

#### CII-Best-Practices
Not applicable for new projects. May be pursued after project maturity.

## Expected Score After Improvements

**Current:** 6.6/10

**After High Priority fixes:**
```
Branch-Protection:     -1 → 9   (+1.0)
Code-Review:            0 → 10  (+1.0)
CI-Tests:              -1 → 10  (+1.1)
Pinned-Dependencies:    0 → 10  (+1.0)
```

**Estimated Total:** ~10.7/10 → **10.0/10** (maximum)

**Realistic:** ~9.5-10.0/10 ✅ (accounting for Contributors, Maintained, Fuzzing)

## Timeline

- **2025-10-24:** Branch protection fixed, vulnerabilities documented
- **2025-10-24:** Test PR created for CI demonstration
- **2025-10-25 (planned):** Pin dependencies via PR
- **2025-10-26 (planned):** Add SAST (gosec/CodeQL)
- **2025-11-01 (target):** Achieve 9.5+/10 score

## References

- [OSSF Scorecard Docs](https://github.com/ossf/scorecard)
- [OSSF Best Practices](https://bestpractices.coreinfrastructure.org/)
- [Scorecard Checks](https://github.com/ossf/scorecard/blob/main/docs/checks.md)
- [Project Scorecard API](https://api.scorecard.dev/projects/github.com/dantte-lp/ocserv-agent)

## Contributing

See [CONTRIBUTING.md](../.github/CONTRIBUTING.md) for development guidelines.

---

**Maintained by:** ocserv-agent team
**Last review:** 2025-10-24
