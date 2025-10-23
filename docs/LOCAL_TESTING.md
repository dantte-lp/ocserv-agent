# Local Testing Guide

Экономьте часы GitHub Actions, проводя тесты локально перед пушем!

## 🚀 Быстрая проверка (2-3 секунды)

Для быстрой проверки перед коммитом:

```bash
./scripts/quick-check.sh
```

Проверяет:
- ✅ Форматирование кода (gofmt)
- ✅ go vet
- ✅ Сборка проекта
- ✅ Базовые тесты

## 🔬 Полная проверка (как в CI)

Для полной проверки перед пушем в GitHub:

```bash
./scripts/test-local.sh
```

Проверяет всё, что проверяет GitHub Actions:
- ✅ Генерация protobuf кода
- ✅ Проверка зависимостей (go mod verify)
- ✅ Форматирование (gofmt)
- ✅ go vet
- ✅ go mod tidy
- ✅ Тесты с race detector и coverage
- ✅ Сборка для всех платформ (Linux/FreeBSD, amd64/arm64)
- ✅ Линтеры (golangci-lint, markdownlint, yamllint)
- ⚠️ Проверка безопасности (опционально, медленно)

## ⚙️ Настройка переменных

```bash
# Пропустить тесты
RUN_TESTS=false ./scripts/test-local.sh

# Пропустить линтеры
RUN_LINT=false ./scripts/test-local.sh

# Включить проверку безопасности (медленно)
RUN_SECURITY=true ./scripts/test-local.sh

# Пропустить сборку бинарников
RUN_BUILD=false ./scripts/test-local.sh

# Пропустить генерацию protobuf
SKIP_PROTO=true ./scripts/test-local.sh

# Комбинация
RUN_SECURITY=true RUN_BUILD=false ./scripts/test-local.sh
```

## 📦 Установка инструментов

### Обязательные (для полной проверки)

```bash
# Go tools
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Protobuf compiler
sudo apt-get install protobuf-compiler

# golangci-lint
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin
```

### Опциональные (для линтеров)

```bash
# Markdown lint
npm install -g markdownlint-cli

# YAML lint
pip install yamllint
```

### Для проверки безопасности

```bash
# gosec (static security analyzer)
go install github.com/securego/gosec/v2/cmd/gosec@latest

# govulncheck (known vulnerabilities)
go install golang.org/x/vuln/cmd/govulncheck@latest
```

## 🎯 Рекомендуемый workflow

### Перед каждым коммитом

```bash
./scripts/quick-check.sh
```

### Перед пушем в GitHub

```bash
./scripts/test-local.sh
```

Если всё прошло успешно - можно смело пушить!

### Перед релизом

```bash
RUN_SECURITY=true ./scripts/test-local.sh
```

## 🔒 Security Testing (локально)

Для запуска security сканирования локально используй Podman Compose:

### Все security тесты сразу

```bash
make security-check
# или
./scripts/security-check.sh
```

Запускает:
- ✅ **Gosec** - статический анализ безопасности Go кода
- ✅ **govulncheck** - проверка известных уязвимостей в зависимостях
- ✅ **Trivy** - сканирование уязвимостей в коде и зависимостях

### Отдельные тесты

```bash
# Только Gosec
make security-gosec

# Только govulncheck
make security-govulncheck

# Только Trivy
make security-trivy
```

### Результаты

Все результаты сохраняются в `deploy/compose/security-results/`:

```bash
# Просмотр findings
cat deploy/compose/security-results/gosec-fixed.sarif | jq '.runs[0].results[]'
cat deploy/compose/security-results/trivy.sarif | jq '.runs[0].results[]'
cat deploy/compose/security-results/govulncheck.json | jq

# Количество issues
jq '.runs[0].results | length' deploy/compose/security-results/gosec-fixed.sarif
jq '.runs[0].results | length' deploy/compose/security-results/trivy.sarif
```

### Почему локально?

1. **Быстрее** - результаты за 30-60 секунд vs 3-5 минут в GitHub Actions
2. **Бесплатно** - не тратятся минуты GitHub Actions
3. **До коммита** - находишь проблемы до пуша
4. **GitHub-compatible** - те же SARIF файлы, что и в CI

**Важно:** SARIF файлы из `gosec-fixed.sarif` содержат автоматическое исправление проблемного формата Gosec и готовы к загрузке в GitHub Security.

## 🔧 Pre-commit Hook (опционально)

Чтобы автоматически запускать quick-check перед каждым коммитом:

```bash
cat > .git/hooks/pre-commit <<'EOF'
#!/bin/bash
./scripts/quick-check.sh
EOF

chmod +x .git/hooks/pre-commit
```

Отключить на время:
```bash
git commit --no-verify
```

## 📊 Экономия GitHub Actions

**Пример:**
- 1 пуш = ~4-5 минут Actions (CI + Lint + Security)
- 10 пушей в день = 40-50 минут
- 30 дней = **1200-1500 минут в месяц**

С локальными тестами:
- Локальная проверка = 10-30 секунд
- Пушить только когда всё работает
- Экономия = **до 80% Actions minutes** 💰

## 🐛 Troubleshooting

### "protoc not found"

```bash
sudo apt-get install protobuf-compiler
```

### "golangci-lint not found"

```bash
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin
```

### "Tests fail locally but pass in CI"

Проверьте версию Go:
```bash
go version  # должно быть 1.25+
```

### "Build fails for FreeBSD"

Это нормально, если вы не на FreeBSD. CI соберёт правильно.
Можно пропустить: `RUN_BUILD=false ./scripts/test-local.sh`

## 📝 Что проверяется в CI

### CI Workflow (.github/workflows/ci.yml)
- ✅ Tests (race detector, coverage)
- ✅ Build (all platforms)
- ✅ Integration tests
- ✅ Code quality (gofmt, go vet, go mod tidy)

### Lint Workflow (.github/workflows/lint.yml)
- ✅ golangci-lint (30+ linters)
- ✅ Markdown lint
- ✅ YAML lint
- ✅ Dockerfile lint

### Security Workflow (.github/workflows/security.yml)
- ✅ gosec (static analysis)
- ✅ CodeQL (deep analysis)
- ✅ Trivy (container scanning)
- ✅ OSSF Scorecard

### Release Workflow (.github/workflows/release.yml)
- ✅ Multi-platform builds
- ✅ SHA256 checksums
- ✅ SLSA Level 3 provenance
- ✅ Container images
- ✅ GitHub Release creation

## 🎓 Best Practices

1. **Перед коммитом**: `./scripts/quick-check.sh` (быстро)
2. **Перед пушем**: `./scripts/test-local.sh` (полностью)
3. **Перед релизом**: `RUN_SECURITY=true ./scripts/test-local.sh` (всё + безопасность)
4. **В CI**: Автоматически при каждом пуше/PR

Это позволяет:
- 🚀 Быстрее разрабатывать (находить ошибки локально)
- 💰 Экономить GitHub Actions minutes
- ✅ Увереннее пушить (знаешь, что CI пройдёт)
- 🔒 Поддерживать качество кода

## 🔗 Связанные документы

- [Contributing Guide](../.github/CONTRIBUTING.md)
- [Workflows Documentation](../.github/WORKFLOWS.md)
- [CI Configuration](../.github/workflows/ci.yml)
