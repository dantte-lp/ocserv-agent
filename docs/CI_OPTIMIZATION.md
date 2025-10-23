# CI Optimization Guide

## 🎯 Overview

Our CI pipeline is optimized to skip expensive jobs when only documentation is changed. This saves GitHub Actions minutes and speeds up the merge process for docs-only PRs.

## 🔧 How It Works

### 1. Change Detection

The `detect-changes` job analyzes which files were modified:

```yaml
code: true/false  # Go files, protos, configs, workflows
docs: true/false  # Markdown, txt files, docs/ directory
```

### 2. Conditional Job Execution

Heavy jobs are skipped when only docs changed:

- ✅ **Always Run:** `detect-changes`, `ci-success`
- ⏭️ **Skipped for docs-only:** `test`, `build`, `integration`, `checks`

### 3. Summary Job

The `ci-success` job always runs and reports overall status:

- **Docs-only changes:** ✅ Success (heavy jobs skipped as expected)
- **Code changes:** ✅ Success only if all jobs passed
- **Job failures:** ❌ Failure with detailed status

## 📊 Performance Impact

### Before Optimization

**Docs-only PR:**
- ❌ Runs all jobs (4-5 minutes)
- 15+ jobs execute
- ~10 GB-minutes consumed

### After Optimization

**Docs-only PR:**
- ✅ Skips heavy jobs (<30 seconds)
- Only 2 jobs execute: `detect-changes` + `ci-success`
- ~0.5 GB-minutes consumed

**Savings:** ~95% for docs-only changes 💰

## 🔍 Code vs Docs Classification

### Code Changes (trigger full CI)

- `*.go` - Go source files
- `*.proto` - Protocol buffer definitions
- `go.mod`, `go.sum` - Dependencies
- `cmd/`, `internal/`, `pkg/` - Code directories
- `deploy/` - Deployment configs
- `.github/workflows/` - CI workflows

### Docs Changes (skip heavy jobs)

- `*.md` - Markdown files
- `*.txt` - Text files
- `docs/` - Documentation directory
- `LICENSE`, `.gitignore`

## 🚦 Branch Protection

Required status check: **"CI Success"**

This single check ensures:
- ✅ Docs-only PRs can merge quickly
- ✅ Code PRs must pass all tests
- ✅ Consistent merge requirements

Previous checks (`Test (Go 1.25)`, `Code Quality Checks`) are now optional but still run for code changes.

## 🧪 Testing Locally

Before pushing, use local testing to catch issues early:

```bash
# Quick check (2-3 seconds)
./scripts/quick-check.sh

# Full check (30-60 seconds)
./scripts/test-local.sh

# With security scans
RUN_SECURITY=true ./scripts/test-local.sh
```

See [LOCAL_TESTING.md](LOCAL_TESTING.md) for details.

## 📈 Monitoring

View CI performance:

```bash
# Check recent workflow runs
gh run list --limit 10

# View specific run details
gh run view <run-id>

# Check PR status
gh pr checks <pr-number>
```

## 🔄 Future Improvements

Potential optimizations:

- [ ] Cache protobuf compilation
- [ ] Parallel security scans
- [ ] Conditional linting (only changed files)
- [ ] Docker layer caching for builds

## 📚 References

- [GitHub Actions: Conditional Jobs](https://docs.github.com/en/actions/using-jobs/using-conditions-to-control-job-execution)
- [Workflow Optimization Best Practices](https://docs.github.com/en/actions/using-workflows/caching-dependencies-to-speed-up-workflows)
- [LOCAL_TESTING.md](LOCAL_TESTING.md) - Local testing guide
- [WORKFLOWS.md](../.github/WORKFLOWS.md) - CI/CD documentation
