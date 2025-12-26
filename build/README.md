# Build & E2E Testing

Директория содержит файлы для сборки и E2E тестирования ocserv-agent.

## Файлы

### Containerfile.e2e-ocserv
Dockerfile для создания E2E тестового окружения с OracleLinux 10 и ocserv 1.3.0.

**Особенности:**
- Базовый образ: OracleLinux 10
- ocserv из EPEL репозитория
- Self-signed TLS сертификаты
- Unix socket для occtl
- Healthcheck на доступность socket

### ocserv.conf.e2e
Минимальная конфигурация ocserv для E2E тестирования.

**Основные параметры:**
- Plain password аутентификация
- Socket: `/var/run/ocserv/ocserv.sock`
- Network: 192.168.99.0/24
- Config-per-user поддержка

### docker-compose.e2e.yaml
Docker Compose файл для запуска полного E2E окружения.

**Сервисы:**
- `ocserv-e2e`: ocserv сервер на OracleLinux 10
- `agent-e2e`: ocserv-agent для интеграционных тестов

### e2e-test.sh
Helper скрипт для управления E2E окружением.

**Команды:**
```bash
./e2e-test.sh build      # Сборка контейнеров
./e2e-test.sh start      # Запуск окружения
./e2e-test.sh test       # Запуск E2E тестов
./e2e-test.sh logs       # Просмотр логов
./e2e-test.sh status     # Проверка статуса
./e2e-test.sh cleanup    # Полная очистка
```

## Быстрый старт

```bash
# 1. Сборка
./e2e-test.sh build

# 2. Запуск
./e2e-test.sh start

# 3. Проверка
./e2e-test.sh status

# 4. Тесты
./e2e-test.sh test

# 5. Остановка
./e2e-test.sh stop
```

## Отладка

```bash
# Логи ocserv
./e2e-test.sh logs ocserv-e2e

# Логи agent
./e2e-test.sh logs agent-e2e

# Вход в контейнер
./e2e-test.sh exec ocserv-e2e-test
```

## Документация

Полная документация: [docs/tmp/E2E_TESTING_GUIDE.md](../docs/tmp/E2E_TESTING_GUIDE.md)
