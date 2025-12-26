# AGILE Plan - ocserv-agent

![Status](https://img.shields.io/badge/status-active-green)
![Sprint](https://img.shields.io/badge/sprint-post--merge-blue)
![Updated](https://img.shields.io/badge/updated-2025--12--26-green)

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
| **Версия** | 0.7.0-dev (после PR #36 merge) |
| **Завершено фаз** | 4 / 7 |
| **Coverage** | 75-80% |
| **golangci-lint** | 0 errors ✅ |
| **Tests** | 119 passed |
| **Proto sync** | ✅ Синхронизировано с portal |

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
| Sprint 13: gRPC Client | Phase 5: Integration | 🔄 Planned |
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

### 🔄 Phase 5: Advanced Integration (IN PROGRESS)

**Даты:** 2025-12-26 - 2025-12-29 (4 дня)
**Статус:** 🔄 In Progress (Day 2 Complete ✅)
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

- [x] Все методы VPNAgentService реализованы (базовые версии) ✅
- [x] Per-user config генерируется корректно ✅ (Day 2 complete)
- [x] Session tracking работает ✅ (Day 2 complete)
- [x] golangci-lint: 0 errors ✅
- [x] gosec HIGH: 0 issues ✅
- [x] Coverage: 85%+ для новых компонентов ✅ (Day 2)
- [ ] Integration tests проходят (Day 3)

#### Связь с Portal

**Portal Sprint 9** (VPN Agent gRPC Server) → **Agent Phase 5**

Portal реализует gRPC server для авторизации, Agent реализует gRPC server для управления.

---

### 🔄 Phase 6: E2E Integration Testing (PLANNED)

**Даты:** 2025-12-30 - 2026-01-02 (4 дня)
**Статус:** 🔄 Planned
**Приоритет:** HIGH (синхронизация с Portal Sprint 14)

#### Цели

End-to-end тестирование полного стека: Portal ↔ Agent ↔ ocserv.

#### Задачи

##### 6.1: Test Environment

- [ ] **deploy/compose.e2e.yaml** — E2E test stack
  - [ ] ocserv-portal (backend)
  - [ ] PostgreSQL для portal
  - [ ] ocserv-agent
  - [ ] Mock ocserv server (или real ocserv)
  - [ ] step-ca (для cert generation)

##### 6.2: E2E Test Scenarios

- [ ] **test/e2e/full_flow_test.go** — Полный lifecycle
  ```
  1. Portal выдаёт сертификат пользователю
  2. User подключается к ocserv
  3. connect-script → agent IPC → portal CheckPolicy
  4. Portal authorize → agent → ocserv tunnel established
  5. Portal calls DisconnectUser → agent → ocserv disconnect
  6. agent ReportSessionUpdate → portal
  ```

- [ ] **Resilience scenarios**
  - [ ] Portal unavailable → fail mode stale → cached decision
  - [ ] Circuit breaker opens → cached decisions used
  - [ ] Portal recovers → circuit closes → fresh decisions

- [ ] **Load testing**
  - [ ] 100 concurrent connections
  - [ ] CheckPolicy latency < 100ms
  - [ ] Session sync latency < 200ms

##### 6.3: QA Automation

- [ ] **qa_runner/e2e_tests.py** — E2E test runner
  - [ ] Запуск compose.e2e.yaml
  - [ ] Выполнение test scenarios
  - [ ] Сбор метрик (latency, throughput)
  - [ ] Генерация HTML отчёта

##### 6.4: Documentation

- [ ] **docs/E2E_TESTING.md** — E2E testing guide
  - [ ] Как запустить E2E тесты
  - [ ] Архитектура test stack
  - [ ] Troubleshooting guide
  - [ ] Performance benchmarks

#### Acceptance Criteria

- [ ] Full flow E2E test проходит
- [ ] Resilience scenarios работают
- [ ] Load testing targets достигнуты
- [ ] Документация полная
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

    section Completed (1-4)
    Phase 1: IPC + Portal      :done, p1, 2025-12-23, 1d
    Phase 2: Portal Integration :done, p2, 2025-12-24, 1d
    Phase 3: Session Sync       :done, p3, 2025-12-25, 1d
    Phase 4: Resilience         :done, p4, 2025-12-26, 1d

    section Planned (5-7)
    Phase 5: Advanced Integration :p5, 2025-12-27, 3d
    Phase 6: E2E Testing         :p6, 2025-12-30, 4d
    Phase 7: Production Hardening :p7, 2026-01-03, 5d
```

### Milestones

- ✅ **Phase 1-4 Complete** - 2025-12-26 (Foundation + Resilience)
- 🎯 **Phase 5 Complete** - 2025-12-29 (Advanced Integration)
- 🎯 **Phase 6 Complete** - 2026-01-02 (E2E Tests)
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
| Версия плана | 1.0 |
| Создан | 2025-12-26 |
| Обновлен | 2025-12-26 |
| Ответственный | Development Team |
| Статус | Phase 4 Complete, Phase 5 Planned |
| Синхронизация | ocserv-portal AGILE_PLAN.md ✅ |

---

> **Примечание:** План синхронизирован с ocserv-portal roadmap. Фазы 5-7 соответствуют Portal Sprints 13-15.
