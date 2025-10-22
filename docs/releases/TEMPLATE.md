# Release vX.Y.Z

**Release Date:** YYYY-MM-DD
**Git Tag:** vX.Y.Z
**Go Version:** 1.25.1

## 🎯 Highlights

Краткое описание главных изменений этого релиза (1-3 предложения).

## ✨ New Features

- **[Feature Name]** - Описание новой функциональности
  - Детали реализации
  - PR: #123
  - Commit: abc1234

## 🐛 Bug Fixes

- **[Bug Description]** - Как было исправлено
  - Issue: #456
  - Commit: def5678

## 🔧 Improvements

- **[Improvement]** - Описание улучшения
  - Performance impact: +15% faster
  - Commit: ghi9012

## 🔒 Security

- **[Security Issue]** - Описание и исправление
  - Severity: High/Medium/Low
  - CVE: CVE-2025-XXXXX (если применимо)

## 📚 Documentation

- Updated README with new configuration options
- Added troubleshooting guide
- API documentation improvements

## ⚠️ Breaking Changes

- **[Breaking Change]** - Описание изменения
  - Migration guide: [link to doc]
  - Affected: Users of feature X

## 🔄 Dependencies

### Updated
- google.golang.org/grpc: v1.69.3 → v1.69.4
- github.com/rs/zerolog: v1.32.0 → v1.33.0

### Added
- github.com/new/package v1.0.0

### Removed
- github.com/old/package (replaced by Y)

## 📊 Statistics

- Commits: 47
- Files Changed: 23
- Contributors: 3
- Test Coverage: 82% → 85%
- Lines Added: +1,234
- Lines Deleted: -567

## 🙏 Contributors

- @username1 - Feature implementation
- @username2 - Bug fixes
- @username3 - Documentation

## 📦 Installation

### Binary
```bash
curl -L https://github.com/dantte-lp/ocserv-agent/releases/download/vX.Y.Z/ocserv-agent-linux-amd64 -o ocserv-agent
chmod +x ocserv-agent
```

### From Source
```bash
git clone https://github.com/dantte-lp/ocserv-agent
cd ocserv-agent
git checkout vX.Y.Z
make compose-build
```

### Docker
```bash
podman pull ghcr.io/dantte-lp/ocserv-agent:vX.Y.Z
```

## 🧪 Testing

All tests pass on:
- ✅ Ubuntu 22.04, 24.04
- ✅ Debian 12 (Bookworm), 13 (Trixie)
- ✅ RHEL 9
- ✅ ocserv 1.1.0, 1.2.0, 1.3.0

## 📝 Notes

Дополнительные заметки о релизе, известные проблемы, планы на будущее.

## 🔗 Links

- [Full Changelog](https://github.com/dantte-lp/ocserv-agent/compare/vX.Y-1.Z...vX.Y.Z)
- [Milestone](https://github.com/dantte-lp/ocserv-agent/milestone/N)
- [Documentation](https://github.com/dantte-lp/ocserv-agent/tree/vX.Y.Z/docs)
