# AGILE Plan - ocserv-agent

![Status](https://img.shields.io/badge/status-active-green)
![Sprint](https://img.shields.io/badge/phase-6__day__2-blue)
![Updated](https://img.shields.io/badge/updated-2025--12--27-green)

> **Описание:** Agile план разработки ocserv-agent в синхронизации с ocserv-portal roadmap.

## Содержание

- [Overview](#overview)
- [Текущий статус](#текущий-статус)
- [Синхронизация с Portal](#синхронизация-с-portal)
- [Фазы разработки](#фазы-разработки)
- [Timeline](#timeline)
- [Команды для разработки](#команды-для-разработки)

---

## Overview

### Проект

**ocserv-agent** — gRPC-агент для удалённого управления OpenConnect VPN серверами, интегрированный с ocserv-portal для централизованной авторизации и управления политиками.

### Текущий статус

| Метрика | Значение |
|---------|----------|
| **Версия** | 0.7.0-dev (Phase 6 Day 2) |
| **Завершено фаз** | 5 / 7 ✅ (Phase 6 Day 2 ✅) |
| **Coverage** | 75-80% |
| **golangci-lint** | 0 errors ✅ |
| **Tests** | 273 + 14 E2E = 287 |
| **Proto sync** | ✅ Синхронизировано с portal |
| **Последнее обновление** | 2025-12-27 |

### Ключевые компоненты

| Компонент | Статус | Версия |
|-----------|--------|--------|
| **gRPC Server** | ✅ Production Ready | v0.6.0 |
| **occtl Wrapper** | ✅ Production Ready | 13/16 команд |
| **IPC Handler** | ✅ Implemented | Unix socket |
| **Portal Client** | ✅ Implemented | gRPC client |
| **Circuit Breaker** | ✅ Implemented | Phase 4 |
| **Decision Cache** | ✅ Implemented | TTL-based |
| **Resilience** | ✅ Implemented | fail_mode: stale |

---

## Синхронизация с Portal

### Portal Roadmap Overview

На основе `/opt/project/repositories/ocserv-portal/docs/AGILE_PLAN.md`:

**Portal Sprints:**
- Sprint 7-10: AD + PKI + gRPC Server (Foundation)
- Sprint 11-13: Advanced Features + gRPC Client
- Sprint 14-15: E2E Testing + Production Hardening

**Критические зависимости для agent:**
1. Portal Sprint 9 — VPN Agent Service (gRPC Server)
2. Portal Sprint 13 — gRPC Client Pool
3. Portal Sprint 14 — E2E Integration Testing

### Соответствие фаз

| Portal Phase | Agent Phase | Статус |
|-------------|-------------|--------|
| Sprint 7-8: AD + PKI | Phase 1: IPC + Portal | ✅ Complete |
| Sprint 9: gRPC Server | Phase 2: Portal Client | ✅ Complete |
| Sprint 10: Cert API | Phase 3: Session Sync | ✅ Complete |
| Sprint 11-12: Advanced | Phase 4: Resilience | ✅ Complete |
| Sprint 13: gRPC Client | Phase 5: Integration | ✅ Complete (2025-12-26) |
| Sprint 14: E2E Testing | Phase 6: E2E Tests | 🔄 Planned |
| Sprint 15: Hardening | Phase 7: Production | 🔄 Planned |

---

## Фазы разработки

### ✅ Phase 1: IPC + Portal Communication (COMPLETED)

**Даты:** 2025-12-23 - 2025-12-24
**Статус:** ✅ COMPLETED

**Результаты:**
- Unix socket IPC handler
- Portal gRPC client (CheckPolicy)
- Proto файлы vpn/v1/auth.proto
- Integration в connect-script workflow

---

### ✅ Phase 2: Portal Integration (COMPLETED)

**Даты:** 2025-12-24 - 2025-12-25
**Статус:** ✅ COMPLETED

**Результаты:**
- gRPC client с mTLS
- CheckPolicy RPC integration
- Error handling с cockroachdb/errors
- OTEL metrics для portal requests

---

### ✅ Phase 3: Session Sync (COMPLETED)

**Даты:** 2025-12-25
**Статус:** ✅ COMPLETED

**Результаты:**
- ReportSessionUpdate RPC
- vpn/v1/events.proto
- Session lifecycle tracking
- Periodic sync goroutine

---

### ✅ Phase 4: Resilience (COMPLETED)

**Даты:** 2025-12-26
**Статус:** ✅ COMPLETED

**Результаты:**
- Circuit Breaker pattern
- Decision Cache (TTL + stale)
- Fail mode policies (open/close/stale)
- OTEL metrics expansion

**Документация:** `docs/tmp/PHASE-4-IMPLEMENTATION-REPORT.md`

---

### ✅ Phase 5: Advanced Integration (COMPLETED)

**Даты:** 2025-12-26 (1 день)
**Дата завершения:** 2025-12-26
**Статус:** ✅ COMPLETED
**Приоритет:** HIGH (синхронизация с Portal Sprint 13)

#### Цели

Полная интеграция с portal gRPC client pool, поддержка всех методов VPN Agent Service.

#### Задачи

##### 5.1: Proto Expansion

**Синхронизация с Portal Sprint 9 requirements:**

- [x] **Расширить agent/v1/agent.proto** ✅ (commit 50815a1)
  - [x] Добавить VPNAgentService из AGILE_PLAN.md:
    ```protobuf
    service VPNAgentService {
        rpc NotifyConnect(NotifyConnectRequest) returns (NotifyConnectResponse);
        rpc NotifyDisconnect(NotifyDisconnectRequest) returns (NotifyDisconnectResponse);
        rpc GetActiveSessions(GetActiveSessionsRequest) returns (GetActiveSessionsResponse);
        rpc DisconnectUser(DisconnectUserRequest) returns (DisconnectUserResponse);
        rpc UpdateUserRoutes(UpdateUserRoutesRequest) returns (UpdateUserRoutesResponse);
    }
    ```
  - [x] Определить message types для всех RPC
  - [x] Обновить `make proto-gen` и сгенерировать Go код

##### 5.2: gRPC Server Extension

- [x] **internal/grpc/vpn_service.go** — Новый VPN service ✅ (commit 50815a1)
  - [x] Реализовать NotifyConnect handler
  - [x] Реализовать NotifyDisconnect handler
  - [x] Реализовать GetActiveSessions (обёртка над occtl show users)
  - [x] Реализовать DisconnectUser (обёртка над occtl disconnect user)
  - [x] Реализовать UpdateUserRoutes (генерация per-user config)

##### 5.3: Per-User Config Management

- [x] **internal/config/user_config.go** — Per-user config ✅ (commit 29a3edb)
  - [x] `GeneratePerUserConfig(username, routes, dns)` → INI файл
  - [x] Atomic file write (write → rename)
  - [x] Валидация routes (CIDR format)
  - [x] Template-based generation
  - [x] Backup старых конфигов
  - [x] Thread-safe операции (sync.Mutex)
  - [x] Rate limiting support (RX/TX per sec)
  - [x] Timeout configuration (idle, mobile, session)

**Пример per-user config:**
```ini
# /etc/ocserv/config-per-user/john.doe
route = 10.0.0.0/255.0.0.0
route = 192.168.0.0/255.255.0.0
dns = 10.0.0.53
split-dns = internal.company.com
restrict-user-to-routes = true
max-same-clients = 2
```

##### 5.4: Session Tracking Database

- [x] **internal/storage/session_store.go** — In-memory session store ✅ (commit 29a3edb)
  - [x] `Add(session)` — при connect
  - [x] `Remove(sessionID)` — при disconnect
  - [x] `List()` — для gRPC GetActiveSessions
  - [x] `ListByUsername(username)` — фильтр по пользователю
  - [x] `Update(sessionID, updateFn)` — обновление сессий
  - [x] `GetStats()` — агрегированная статистика
  - [x] TTL + automatic cleanup goroutine
  - [x] Thread-safe (sync.RWMutex)
  - [ ] Sync с occtl для reconciliation (Phase 5 Day 3)

##### 5.5: Testing

- [x] **internal/grpc/vpn_service_test.go** — Unit tests (базовые) ✅ (commit 50815a1)
  - [x] Test NotifyConnect basic flow
  - [x] Test NotifyDisconnect basic flow
  - [x] Test DisconnectUser validation
  - [x] Test UpdateUserRoutes validation
  - [ ] Mock occtl для GetActiveSessions (Phase 5 Day 3)
  - [ ] Mock config generator для UpdateUserRoutes (Phase 5 Day 3)

- [x] **internal/config/user_config_test.go** — Unit tests ✅ (commit 29a3edb)
  - [x] Generator creation и directory validation
  - [x] Config generation (basic, split-dns, restrictions, IPs, rate limits, timeouts)
  - [x] Validation tests (CIDR, IP addresses, empty fields)
  - [x] Backup mechanism testing
  - [x] Thread-safety testing (concurrent writes)
  - [x] 19 comprehensive test cases

- [x] **internal/storage/session_store_test.go** — Unit tests ✅ (commit 29a3edb)
  - [x] CRUD operations (Add, Get, Update, Remove)
  - [x] List operations (List, ListByUsername, Count, CountByUsername)
  - [x] Stats aggregation (GetStats, UpdateStats)
  - [x] TTL expiration testing
  - [x] Background cleanup goroutine testing
  - [x] Thread-safety testing (100 concurrent goroutines)
  - [x] 20 comprehensive test cases

- [x] **parseBytes() helper** — Implementation ✅ (commit 29a3edb)
  - [x] Парсинг human-readable sizes (K, M, G, T)
  - [x] Поддержка decimal values (1.5M, 3.2G)
  - [x] Error handling для invalid formats

- [ ] **Integration test** с mock portal (Phase 5 Day 3)
  - [ ] Full flow: NotifyConnect → CheckPolicy → session stored
  - [ ] Routes update propagation
  - [ ] Disconnect user workflow

#### Acceptance Criteria

- [x] Все методы VPNAgentService реализованы ✅
- [x] Per-user config генерируется корректно ✅
- [x] Session tracking работает ✅
- [x] golangci-lint: 0 errors ✅
- [x] gosec HIGH: 0 issues ✅
- [x] Coverage: 85%+ для новых компонентов ✅
- [x] Unit tests написаны и проходят ✅
- [x] PR #37 смержен в main ✅ (2025-12-26)

#### Связь с Portal

**Portal Sprint 9** (VPN Agent gRPC Server) → **Agent Phase 5**

Portal реализует gRPC server для авторизации, Agent реализует gRPC server для управления.

---

### 🔄 Phase 6: E2E Integration Testing (IN PROGRESS)

**Даты:** 2025-12-26 - 2025-12-27
**Статус:** 🔄 IN PROGRESS (Day 2 COMPLETED ✅)
**Приоритет:** HIGH (синхронизация с Portal Sprint 14)

#### Цели

End-to-end тестирование с реальным ocserv на OracleLinux 10.

#### Задачи

##### 6.1: Test Environment

- [x] **build/Containerfile.e2e-ocserv** — OracleLinux 10 + ocserv ✅
  - [x] OracleLinux 10 базовый образ
  - [x] EPEL репозиторий для ocserv 1.3.0
  - [x] Self-signed TLS сертификаты
  - [x] Unix socket configuration
  - [x] Healthcheck на socket доступность

- [x] **build/ocserv.conf.e2e** — Minimal ocserv config ✅
  - [x] Plain password аутентификация
  - [x] Unix socket: `/var/run/ocserv/ocserv.sock`
  - [x] Network: 192.168.99.0/24
  - [x] Config-per-user поддержка

- [x] **build/docker-compose.e2e.yaml** — E2E stack ✅
  - [x] ocserv-e2e service (OracleLinux 10)
  - [x] agent-e2e service
  - [x] Shared unix socket volume
  - [x] Network isolation

- [x] **build/e2e-test.sh** — Helper script ✅
  - [x] `build` — Сборка контейнеров
  - [x] `start` — Запуск окружения
  - [x] `test` — Запуск E2E тестов
  - [x] `logs` — Просмотр логов
  - [x] `status` — Проверка статуса
  - [x] `cleanup` — Полная очистка

##### 6.2: E2E Test Scenarios

- [x] **test/e2e/ocserv_integration_test.go** — ocserv integration tests ✅
  - [x] TestOcctlSocketAccess — проверка доступа к unix socket
  - [x] TestOcctlShowStatus — выполнение `occtl show status`
  - [x] TestOcctlShowUsersJSON — получение списка пользователей в JSON
  - [x] TestOcctlShowSessionsJSON — получение активных сессий
  - [x] TestConfigPerUserDirectory — проверка директории config-per-user
  - [x] TestGenerateUserConfig — создание пользовательской конфигурации
  - [x] TestOcctlReload — перезагрузка конфигурации ocserv
  - [x] TestOcctlCommandValidation — валидация команд occtl
  - [x] TestOcservProcessRunning — проверка запущенного процесса

- [x] **test/e2e/full_flow_test.go** — Полный lifecycle ✅ (Phase 6 Day 2)
  - [x] TestFullFlow_ConnectSessionManagement — полный цикл сессии
  - [x] TestFullFlow_MultipleSessionsSameUser — множественные сессии
  - [x] TestFullFlow_SessionExpiry — проверка TTL сессий
  - [x] TestFullFlow_UpdateRoutesWithoutSession — обновление без сессии
  - [x] 5 тест-кейсов, ~457 строк кода ✅

- [x] **test/e2e/resilience_test.go** — Resilience scenarios ✅ (Phase 6 Day 2)
  - [x] TestResilience_OcservRestart — перезапуск ocserv
  - [x] TestResilience_SocketUnavailable — недоступность socket
  - [x] TestResilience_TimeoutHandling — обработка таймаутов
  - [x] TestResilience_ConcurrentFailures — параллельные сбои
  - [x] TestResilience_GracefulDegradation — graceful degradation
  - [x] TestResilience_InvalidInput — некорректные данные
  - [x] 6 тест-кейсов, ~525 строк кода ✅

- [x] **test/e2e/load_test.go** — Load testing ✅ (Phase 6 Day 2)
  - [x] TestLoad_ConcurrentConnections — 100 одновременных подключений
  - [x] TestLoad_HighFrequencyUpdates — частые обновления маршрутов
  - [x] TestLoad_SessionQueryPerformance — производительность запросов
  - [x] Метрики: latency (p50, p95, p99), memory, goroutines, throughput
  - [x] 3 теста, ~465 строк кода ✅

##### 6.3: QA Automation

- [ ] **qa_runner/e2e_tests.py** — E2E test runner
  - [ ] Запуск compose.e2e.yaml
  - [ ] Выполнение test scenarios
  - [ ] Сбор метрик (latency, throughput)
  - [ ] Генерация HTML отчёта

##### 6.3: Documentation

- [x] **docs/tmp/E2E_TESTING_GUIDE.md** — E2E testing guide ✅
  - [x] Как запустить E2E тесты
  - [x] Архитектура test stack
  - [x] Troubleshooting guide
  - [x] Известные проблемы

- [x] **build/README.md** — Build & E2E helper docs ✅
  - [x] Описание файлов
  - [x] Быстрый старт
  - [x] Команды отладки

##### 6.4: QA Automation (Planned)

- [ ] **qa_runner/e2e_tests.py** — E2E test runner (Phase 6 Day 2)
  - [ ] Запуск compose.e2e.yaml
  - [ ] Выполнение test scenarios
  - [ ] Сбор метрик (latency, throughput)
  - [ ] Генерация HTML отчёта

#### Acceptance Criteria

**Day 1 (2025-12-26):**
- [x] E2E окружение с OracleLinux 10 создано ✅
- [x] ocserv 1.3.0 установлен и работает ✅
- [x] Unix socket communication протестирован ✅
- [x] E2E integration tests написаны (9 тестов) ✅
- [x] Документация создана ✅
- [x] Helper скрипты работают ✅

**Day 2 (2025-12-27) ✅ COMPLETED:**
- [x] Full flow E2E test реализован (5 тестов) ✅
- [x] Resilience scenarios реализованы (6 тестов) ✅
- [x] Load testing реализован (3 теста с метриками) ✅
- [x] Всего добавлено 14 новых тест-кейсов ✅
- [x] ~1447 строк тестового кода ✅
- [x] Port conflict исправлен (9091 вместо 9090) ✅
- [x] Документация обновлена ✅

**Day 3 (Planned):**
- [ ] Запуск всех E2E тестов в контейнере
- [ ] Сбор и анализ метрик производительности
- [ ] Генерация финального QA отчёта
- [ ] CI/CD pipeline интегрирован

#### Связь с Portal

**Portal Sprint 14** (E2E Integration & Testing) ↔ **Agent Phase 6**

Совместное тестирование integration stack.

---

### 🔄 Phase 7: Production Hardening (PLANNED)

**Даты:** 2026-01-03 - 2026-01-07 (5 дней)
**Статус:** 🔄 Planned
**Приоритет:** CRITICAL (синхронизация с Portal Sprint 15)

#### Цели

Подготовка к production deployment: мониторинг, алерты, операционные процедуры.

#### Задачи

##### 7.1: Observability

- [ ] **Prometheus Metrics expansion**
  ```
  # Agent-specific
  ocserv_agent_active_sessions{server_id}
  ocserv_agent_portal_requests_total{method,status}
  ocserv_agent_portal_request_duration_seconds{method}
  ocserv_agent_circuit_breaker_state{service}
  ocserv_agent_cache_size
  ocserv_agent_cache_hit_ratio

  # ocserv metrics
  ocserv_total_sessions
  ocserv_bytes_in_total
  ocserv_bytes_out_total
  ocserv_disconnect_total{reason}
  ```

- [ ] **Grafana Dashboards**
  - [ ] Agent health dashboard
  - [ ] VPN sessions dashboard
  - [ ] Portal integration dashboard
  - [ ] Circuit breaker dashboard

- [ ] **Alertmanager Rules**
  - [ ] Portal unavailable > 5min
  - [ ] Circuit breaker open > 10min
  - [ ] Cache hit ratio < 50%
  - [ ] ocserv daemon down

##### 7.2: Logging

- [ ] **Structured Logging** (zerolog)
  - [ ] JSON format для production
  - [ ] Context propagation (trace IDs)
  - [ ] Sensitive data redaction (passwords, tokens)
  - [ ] Log rotation config

- [ ] **VictoriaLogs integration**
  - [ ] OTLP logs exporter
  - [ ] Correlation с traces
  - [ ] Retention policies

##### 7.3: Deployment

- [ ] **Production Containerfile**
  - [ ] Multi-stage build
  - [ ] Distroless base image
  - [ ] Non-root user
  - [ ] Health checks

- [ ] **systemd Service**
  - [ ] ocserv-agent.service
  - [ ] Auto-restart on failure
  - [ ] Resource limits (CPU, memory)
  - [ ] Dependencies (ocserv.service, network.target)

- [ ] **Ansible Playbook**
  - [ ] Automated deployment
  - [ ] Config management
  - [ ] Certificate deployment
  - [ ] Health check verification

##### 7.4: Operations Runbook

- [ ] **Portal Integration Issues**
  ```bash
  # Симптом: Circuit breaker always open
  # Диагностика:
  journalctl -u ocserv-agent -f | grep "circuit_breaker"
  curl localhost:9090/metrics | grep circuit_breaker_state

  # Решение:
  1. Проверить доступность portal: curl -v https://portal:8080/health
  2. Проверить TLS сертификаты: openssl s_client -connect portal:8080
  3. Перезапустить agent: systemctl restart ocserv-agent
  ```

- [ ] **Session Sync Issues**
  - [ ] Проверка occtl доступности
  - [ ] Reconciliation процедуры
  - [ ] Manual session cleanup

- [ ] **Certificate Issues**
  - [ ] mTLS troubleshooting
  - [ ] Certificate rotation
  - [ ] CA verification

##### 7.5: Documentation

- [ ] **OPERATIONS.md** — Operations guide
  - [ ] Deployment procedures
  - [ ] Monitoring setup
  - [ ] Troubleshooting guide
  - [ ] Disaster recovery

- [ ] **SECURITY.md** — Security best practices
  - [ ] mTLS configuration
  - [ ] Secret management
  - [ ] Vulnerability management
  - [ ] Incident response

#### Acceptance Criteria

- [ ] Metrics экспортируются в Prometheus
- [ ] Dashboards показывают актуальные данные
- [ ] Alerts срабатывают корректно
- [ ] Deployment автоматизирован
- [ ] Runbook полный и актуальный
- [ ] Security audit пройден

#### Связь с Portal

**Portal Sprint 15** (Production Hardening) ↔ **Agent Phase 7**

Совместная подготовка к production.

---

## Timeline

```mermaid
gantt
    title ocserv-agent Development Timeline
    dateFormat  YYYY-MM-DD

    section Completed (1-5)
    Phase 1: IPC + Portal      :done, p1, 2025-12-23, 1d
    Phase 2: Portal Integration :done, p2, 2025-12-24, 1d
    Phase 3: Session Sync       :done, p3, 2025-12-25, 1d
    Phase 4: Resilience         :done, p4, 2025-12-26, 1d
    Phase 5: Advanced Integration :done, p5, 2025-12-26, 1d

    section Planned (6-7)
    Phase 6: E2E Testing         :p6, 2025-12-27, 4d
    Phase 7: Production Hardening :p7, 2026-01-03, 5d
```

### Milestones

- ✅ **Phase 1-5 Complete** - 2025-12-26 (Foundation + Integration)
- 🎯 **Phase 6 Complete** - 2025-12-31 (E2E Tests)
- 🎯 **Phase 7 Complete** - 2026-01-07 (Production Ready)
- 🚀 **Production Release** - 2026-01-10

### Critical Path

```
Phase 5 (Integration) → Phase 6 (E2E Tests) → Phase 7 (Production)
    ↓ (sync with Portal Sprint 13)
    ↓ (sync with Portal Sprint 14)
    ↓ (sync with Portal Sprint 15)
```

---

## Команды для разработки

### Development (Container-First)

```bash
# Запуск dev окружения
make compose-dev

# С hot reload
podman run --rm -v $(pwd):/app -p 8080:8080 ocserv-agent-qa

# Logs
make compose-logs
```

### Testing

```bash
# Unit tests (в контейнере)
make compose-test

# QA automation
python3 -m qa_runner.runner --container ocserv-agent-qa

# E2E tests (Phase 6)
make e2e-test

# Load testing (Phase 6)
make load-test
```

### Linting

```bash
# golangci-lint (в контейнере)
make compose-lint

# Security scan
make compose-security

# Full pipeline
make build-all
```

### Proto

```bash
# Генерация Go кода из proto
make proto-gen

# Проверка proto файлов
buf lint pkg/proto
```

### Deployment (Phase 7)

```bash
# Build production image
make build-production

# Deploy via Ansible
ansible-playbook deploy/ansible/deploy.yml

# systemd управление
systemctl start ocserv-agent
systemctl status ocserv-agent
journalctl -u ocserv-agent -f
```

---

## Синхронизация с Portal

### Критические точки синхронизации

| Дата | Portal | Agent | Действие |
|------|--------|-------|----------|
| 2025-12-29 | Sprint 9 complete | Phase 5 start | Proto sync, VPN service |
| 2026-01-02 | Sprint 13 complete | Phase 6 start | gRPC client pool, E2E tests |
| 2026-01-07 | Sprint 15 complete | Phase 7 complete | Production ready |

### Коммуникация

- **Daily sync**: Проверка proto синхронизации
- **Weekly review**: Обзор integration points
- **Milestone meetings**: Перед началом каждой Phase

---

## Связанная документация

### ocserv-agent

- [FINAL-INTEGRATION-PLAN-2025-12-26.md](/opt/project/repositories/ocserv-agent/docs/tmp/architecture/FINAL-INTEGRATION-PLAN-2025-12-26.md)
- [PHASE-4-IMPLEMENTATION-REPORT.md](/opt/project/repositories/ocserv-agent/docs/tmp/PHASE-4-IMPLEMENTATION-REPORT.md)
- [README.md](/opt/project/repositories/ocserv-agent/README.md)

### ocserv-portal

- [AGILE_PLAN.md](/opt/project/repositories/ocserv-portal/docs/AGILE_PLAN.md)
- [AGENT_INTEGRATION.md](/opt/project/repositories/ocserv-portal/docs/AGENT_INTEGRATION.md)

### Workspace

- [CLAUDE.md](/opt/project/repositories/CLAUDE.md)

---

**Метаданные:**

| Параметр | Значение |
|----------|----------|
| Проект | ocserv-agent |
| Версия плана | 1.1 |
| Создан | 2025-12-26 |
| Обновлен | 2025-12-26 (Phase 5 Complete) |
| Ответственный | Development Team |
| Статус | Phase 5 Complete, Phase 6 Planned |
| Синхронизация | ocserv-portal AGILE_PLAN.md ✅ |

---

> **Примечание:** План синхронизирован с ocserv-portal roadmap. Фазы 5-7 соответствуют Portal Sprints 13-15.
