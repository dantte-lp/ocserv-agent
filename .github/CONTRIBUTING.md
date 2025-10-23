# Contributing to ocserv-agent

## Development Workflow

### 1. Branch Protection Rules

The `main` branch is protected with the following rules:

**Required Status Checks:**
- ✅ Test (Go 1.25) - Must pass
- ✅ Code Quality Checks - Must pass (gofmt, go vet, go mod tidy)
- ✅ golangci-lint - Must pass

**Other Checks (informational only):**
- Test (Go 1.24)
- Build checks (linux/amd64, linux/arm64, darwin, windows)
- Markdown lint
- YAML lint
- Security scans (gosec, govulncheck, CodeQL, Trivy, OSSF Scorecard)

**Protection Rules:**
- ❌ No force pushes to main
- ❌ No deletion of main branch
- ✅ All conversations must be resolved before merge
- ⚠️ Admins can bypass (use with caution)

### 2. Development Process

#### Step 1: Create Feature Branch

```bash
# Update main branch
git checkout main
git pull origin main

# Create feature branch
git checkout -b feature/my-feature
# or
git checkout -b fix/bug-description
```

#### Step 2: Make Changes

```bash
# Make your changes
vim internal/ocserv/manager.go

# Format code (REQUIRED)
podman run --rm -v $(pwd):/workspace:z -w /workspace golang:1.25 gofmt -s -w .

# Run tests locally (optional but recommended)
make compose-test
```

#### Step 3: Commit Changes

Follow [Conventional Commits](https://www.conventionalcommits.org/) format:

```bash
git add -A
git commit -m "feat(ocserv): add new occtl command support

Implement support for 'occtl show sessions' command.

- Add ShowSessions method
- Add SessionInfo struct
- Update manager to handle new command
- Add tests

Closes #123"
```

**Commit Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code formatting (no logic changes)
- `refactor`: Code refactoring
- `test`: Adding/updating tests
- `chore`: Maintenance tasks
- `ci`: CI/CD changes

#### Step 4: Push to GitHub

```bash
# Push feature branch
git push -u origin feature/my-feature
```

#### Step 5: Create Pull Request

```bash
# Using gh CLI
gh pr create --title "feat: add new feature" --body "Description..."

# Or via GitHub web interface
# https://github.com/dantte-lp/ocserv-agent/compare
```

**PR Template:**

```markdown
## Summary

Brief description of changes.

## Changes

- Change 1
- Change 2
- Change 3

## Testing

- [ ] Unit tests pass
- [ ] Integration tests pass (if applicable)
- [ ] Manually tested on [environment]

## Checklist

- [ ] Code formatted with gofmt
- [ ] Tests added/updated
- [ ] Documentation updated
- [ ] All conversations resolved
```

#### Step 6: Wait for CI

GitHub Actions will automatically run:
1. **Test (Go 1.25)** - ⚠️ REQUIRED
2. **Code Quality Checks** - ⚠️ REQUIRED
3. **golangci-lint** - ⚠️ REQUIRED
4. All other checks (informational)

**Check status:**
```bash
gh pr checks
```

#### Step 7: Merge PR

Once all **required** checks pass:

```bash
# Squash and merge (recommended for feature PRs)
gh pr merge --squash --delete-branch

# Or regular merge (for multi-commit features)
gh pr merge --merge --delete-branch

# Or rebase (for clean history)
gh pr merge --rebase --delete-branch
```

### 3. Quick Fixes

For simple changes (typos, docs, formatting):

```bash
git checkout -b fix/quick-fix
# make changes
git commit -m "docs: fix typo in README"
git push -u origin fix/quick-fix
gh pr create --fill
gh pr merge --squash --delete-branch
```

### 4. Handling CI Failures

#### Code Quality Checks Failed

```bash
# Format code
podman run --rm -v $(pwd):/workspace:z -w /workspace golang:1.25 gofmt -s -w .

# Check formatting
podman run --rm -v $(pwd):/workspace:z -w /workspace golang:1.25 gofmt -l .

# Run go vet
podman run --rm -v $(pwd):/workspace:z -w /workspace golang:1.25 go vet ./...

# Tidy dependencies
podman run --rm -v $(pwd):/workspace:z -w /workspace golang:1.25 go mod tidy

# Commit and push
git add -A
git commit -m "style: fix formatting"
git push
```

#### Tests Failed

```bash
# Run tests locally
make compose-test

# Run specific test
podman-compose -f deploy/compose/test.yml run --rm test go test -v ./internal/ocserv/...

# Fix tests and push
git add -A
git commit -m "test: fix failing tests"
git push
```

#### golangci-lint Failed

```bash
# Run locally
podman run --rm -v $(pwd):/workspace:z -w /workspace golangci/golangci-lint:latest golangci-lint run

# Auto-fix some issues
podman run --rm -v $(pwd):/workspace:z -w /workspace golangci/golangci-lint:latest golangci-lint run --fix

# Commit fixes
git add -A
git commit -m "style: fix linter issues"
git push
```

### 5. Emergency Hotfixes

For critical production issues:

```bash
# Create hotfix from main
git checkout main
git pull origin main
git checkout -b hotfix/critical-bug

# Make minimal fix
# ... fix code ...

# Fast-track PR
git commit -m "fix: critical security issue in auth"
git push -u origin hotfix/critical-bug
gh pr create --title "HOTFIX: critical security issue" --body "Emergency fix for CVE-2025-XXXX"

# Admin can bypass checks if needed (use with extreme caution)
# Regular process: wait for CI and merge
gh pr merge --squash --delete-branch
```

### 6. Dependabot PRs

Dependabot automatically creates PRs for dependency updates.

**Review Process:**

```bash
# View PR
gh pr view <number>

# Check what changed
gh pr diff <number>

# If safe, merge
gh pr merge <number> --squash --delete-branch
```

**Auto-merge for patch updates:**
```bash
gh pr review <number> --approve
gh pr merge <number> --auto --squash --delete-branch
```

### 7. Tips

**Useful Commands:**

```bash
# Check branch protection
gh api /repos/dantte-lp/ocserv-agent/branches/main/protection | jq

# List all PRs
gh pr list

# View PR checks
gh pr checks <number>

# View PR status
gh pr view <number>

# Rerun failed checks
gh pr checks <number> --required

# Update branch with latest main
git checkout feature/my-feature
git fetch origin
git rebase origin/main
git push --force-with-lease
```

**Pre-commit Checks:**

Create `.git/hooks/pre-commit`:
```bash
#!/bin/bash
# Format code before commit
podman run --rm -v $(pwd):/workspace:z -w /workspace golang:1.25 gofmt -s -w .
git add -A
```

### 8. Release Process

See [Release Workflow](.github/WORKFLOWS.md#release-workflow) for creating releases.

## Questions?

- GitHub Issues: https://github.com/dantte-lp/ocserv-agent/issues
- Workflow Docs: [.github/WORKFLOWS.md](.github/WORKFLOWS.md)
- Main README: [README.md](../README.md)
