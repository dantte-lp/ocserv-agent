# Руководство по эксплуатации ocserv-agent

![Version](https://img.shields.io/badge/version-0.7.0-blue)
![Status](https://img.shields.io/badge/status-production--ready-green)

Операционное руководство для администраторов ocserv-agent в production окружении.

---

## Содержание

- [Обзор](#обзор)
- [Мониторинг и метрики](#мониторинг-и-метрики)
- [Логирование](#логирование)
- [Troubleshooting](#troubleshooting)
- [Процедуры обслуживания](#процедуры-обслуживания)
- [Резервное копирование](#резервное-копирование)
- [Аварийное восстановление](#аварийное-восстановление)
- [Безопасность](#безопасность)
- [Обновление версий](#обновление-версий)

---

## Обзор

### Архитектура системы

```
┌──────────────────┐
│  ocserv-portal   │ (Portal)
│  (gRPC Client)   │
└────────┬─────────┘
         │ mTLS
         ↓
┌──────────────────┐
│  ocserv-agent    │ (Agent)
│  - gRPC Server   │
│  - IPC Handler   │
│  - Portal Client │
│  - Circuit Break │
│  - Cache Layer   │
└────────┬─────────┘
         │ Unix Socket
         ↓
┌──────────────────┐
│  ocserv daemon   │ (VPN Server)
└──────────────────┘
```

### Основные компоненты

| Компонент | Назначение | Критичность |
|-----------|------------|-------------|
| **gRPC Server** | Управляющий API | Критический |
| **IPC Handler** | Связь с ocserv через Unix socket | Критический |
| **Portal Client** | Авторизация через portal | Критический |
| **Circuit Breaker** | Защита от сбоев portal | Высокая |
| **Decision Cache** | Кеш авторизационных решений | Высокая |
| **Metrics Exporter** | Экспорт метрик в Prometheus | Средняя |

---

## Мониторинг и метрики

### Prometheus Metrics

#### Доступ к метрикам

```bash
# Локальный просмотр метрик
curl http://localhost:9090/metrics

# Фильтрация метрик agent
curl http://localhost:9090/metrics | grep ocserv_agent

# Проверка доступности endpoint
curl -I http://localhost:9090/metrics
```

#### Бизнес-метрики

##### VPN сессии

```promql
# Текущее количество активных VPN сессий
ocserv_agent_active_sessions

# Текущее количество подключенных пользователей
ocserv_agent_connected_users

# Пример запроса: средняя загрузка за 5 минут
avg_over_time(ocserv_agent_active_sessions[5m])

# Пиковая нагрузка за час
max_over_time(ocserv_agent_active_sessions[1h])
```

##### Выполнение команд

```promql
# Общее количество выполненных команд
ocserv_agent_commands_total

# Rate выполнения команд (команд в секунду)
rate(ocserv_agent_commands_total[5m])

# Количество ошибок
ocserv_agent_command_errors_total

# Error rate
rate(ocserv_agent_command_errors_total[5m]) / rate(ocserv_agent_commands_total[5m])

# Длительность команд (квантили)
histogram_quantile(0.95, rate(ocserv_agent_command_duration_seconds_bucket[5m]))
histogram_quantile(0.99, rate(ocserv_agent_command_duration_seconds_bucket[5m]))
```

#### gRPC метрики

```promql
# Общее количество gRPC запросов
grpc_server_requests_total

# Rate запросов
rate(grpc_server_requests_total[5m])

# Успешность запросов
rate(grpc_server_requests_total{status="ok"}[5m]) / rate(grpc_server_requests_total[5m])

# Длительность обработки (95-й перцентиль)
histogram_quantile(0.95, rate(grpc_server_request_duration_seconds_bucket[5m]))

# Медленные запросы (> 1s)
grpc_server_request_duration_seconds_bucket{le="1.0"}
```

#### Portal Integration метрики

```promql
# Circuit Breaker состояние (0=closed, 1=open, 2=half-open)
ocserv_agent_circuit_breaker_state

# Количество запросов к portal
ocserv_agent_portal_requests_total

# Ошибки при обращении к portal
ocserv_agent_portal_errors_total

# Cache hit rate
rate(ocserv_agent_cache_hits_total[5m]) / rate(ocserv_agent_cache_requests_total[5m])

# Использование stale данных
ocserv_agent_cache_stale_hits_total
```

#### Runtime метрики (Go)

```promql
# Количество goroutines
go_goroutines

# Использование памяти (MB)
go_memory_alloc_bytes / 1024 / 1024

# Системная память (MB)
go_memory_sys_bytes / 1024 / 1024

# Garbage Collection метрики
rate(go_gc_duration_seconds_sum[5m])
```

### Health Checks

#### Проверка доступности

```bash
# HTTP health check (если реализован)
curl http://localhost:8080/health

# gRPC health check (через grpcurl)
grpcurl -plaintext localhost:8080 grpc.health.v1.Health/Check

# Systemd статус
systemctl status ocserv-agent

# Проверка процесса
ps aux | grep ocserv-agent
pgrep -a ocserv-agent
```

#### Проверка компонентов

```bash
# 1. Проверка gRPC server
grpcurl -plaintext localhost:8080 list

# 2. Проверка ocserv socket
ls -la /var/run/ocserv/ocserv.sock
test -S /var/run/ocserv/ocserv.sock && echo "Socket OK" || echo "Socket FAIL"

# 3. Проверка доступности portal
timeout 5 bash -c "</dev/tcp/portal.example.com/8080" && echo "Portal reachable" || echo "Portal unreachable"

# 4. Проверка TLS сертификатов
openssl x509 -in /etc/ocserv-agent/mtls/client.crt -noout -enddate
openssl x509 -in /etc/ocserv-agent/mtls/client.crt -noout -checkend 2592000  # 30 дней
```

### Рекомендуемые алерты

#### Критические (P1 - немедленная реакция)

```yaml
# Agent недоступен
- alert: OcservAgentDown
  expr: up{job="ocserv-agent"} == 0
  for: 1m
  labels:
    severity: critical
  annotations:
    summary: "ocserv-agent недоступен"

# Portal недоступен > 5 минут
- alert: PortalUnavailable
  expr: ocserv_agent_circuit_breaker_state == 1
  for: 5m
  labels:
    severity: critical
  annotations:
    summary: "Circuit Breaker открыт - portal недоступен"

# ocserv daemon остановлен
- alert: OcservDaemonDown
  expr: ocserv_agent_active_sessions == 0 AND ocserv_agent_command_errors_total > 0
  for: 2m
  labels:
    severity: critical
  annotations:
    summary: "ocserv daemon возможно остановлен"
```

#### Высокий приоритет (P2 - реакция в течение часа)

```yaml
# Высокий error rate
- alert: HighErrorRate
  expr: rate(ocserv_agent_command_errors_total[5m]) / rate(ocserv_agent_commands_total[5m]) > 0.1
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "Высокий процент ошибок (>10%)"

# Низкий cache hit rate
- alert: LowCacheHitRate
  expr: rate(ocserv_agent_cache_hits_total[5m]) / rate(ocserv_agent_cache_requests_total[5m]) < 0.5
  for: 10m
  labels:
    severity: warning
  annotations:
    summary: "Низкий cache hit rate (<50%)"

# Медленные запросы
- alert: SlowGRPCRequests
  expr: histogram_quantile(0.95, rate(grpc_server_request_duration_seconds_bucket[5m])) > 5
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "95-й перцентиль gRPC запросов >5s"
```

#### Средний приоритет (P3 - мониторинг)

```yaml
# Высокое использование памяти
- alert: HighMemoryUsage
  expr: go_memory_alloc_bytes / 1024 / 1024 > 400
  for: 10m
  labels:
    severity: info
  annotations:
    summary: "Использование памяти >400MB"

# TLS сертификат истекает через 30 дней
- alert: TLSCertificateExpiringSoon
  expr: (ocserv_agent_tls_cert_expiry_seconds - time()) / 86400 < 30
  for: 1h
  labels:
    severity: info
  annotations:
    summary: "TLS сертификат истекает через <30 дней"
```

---

## Логирование

### Структура логов

Agent использует структурированное логирование (JSON) через `slog` и `zerolog`.

#### Уровни логирования

| Уровень | Использование |
|---------|---------------|
| **DEBUG** | Детальная отладочная информация |
| **INFO** | Стандартные операции (default) |
| **WARN** | Предупреждения (не критичные) |
| **ERROR** | Ошибки (требуют внимания) |

#### Просмотр логов

```bash
# Systemd журнал
journalctl -u ocserv-agent -f

# Только ошибки
journalctl -u ocserv-agent -p err -f

# За последний час
journalctl -u ocserv-agent --since "1 hour ago"

# С временными метками
journalctl -u ocserv-agent -o short-iso -f

# Экспорт в файл
journalctl -u ocserv-agent --since today > agent-logs-$(date +%Y%m%d).log
```

#### Анализ логов

```bash
# Поиск ошибок
journalctl -u ocserv-agent | grep -i error

# Подсчет ошибок за день
journalctl -u ocserv-agent --since today | grep -c ERROR

# Поиск circuit breaker событий
journalctl -u ocserv-agent | grep circuit_breaker

# Поиск медленных запросов
journalctl -u ocserv-agent | grep "duration" | awk '$NF > 1000'
```

#### Типичные лог-сообщения

**Успешный запуск:**
```json
{
  "level": "info",
  "time": "2025-12-27T10:00:00Z",
  "msg": "ocserv-agent starting",
  "version": "0.7.0",
  "config": "/etc/ocserv-agent/config.yaml"
}
```

**Circuit breaker открыт:**
```json
{
  "level": "warn",
  "time": "2025-12-27T10:05:00Z",
  "msg": "circuit breaker opened",
  "component": "portal_client",
  "failures": 5
}
```

**Cache miss:**
```json
{
  "level": "debug",
  "time": "2025-12-27T10:10:00Z",
  "msg": "cache miss, fetching from portal",
  "username": "john.doe",
  "ip": "10.0.1.50"
}
```

### Изменение уровня логирования

#### Runtime изменение (через конфиг)

```bash
# 1. Отредактировать конфиг
vim /etc/ocserv-agent/config.yaml

# Изменить:
logging:
  level: debug  # info -> debug

# 2. Перезагрузить конфиг (если поддерживается)
systemctl reload ocserv-agent

# Или перезапустить
systemctl restart ocserv-agent
```

#### Environment variable

```bash
# Установить переменную окружения
systemctl edit ocserv-agent

# Добавить:
[Service]
Environment="LOG_LEVEL=debug"

# Применить
systemctl daemon-reload
systemctl restart ocserv-agent
```

---

## Troubleshooting

### Проблема: Agent не стартует

#### Симптомы
- `systemctl status ocserv-agent` показывает failed
- Нет слушающего порта на 8080

#### Диагностика

```bash
# 1. Проверить системный статус
systemctl status ocserv-agent

# 2. Просмотреть логи запуска
journalctl -u ocserv-agent -n 50 --no-pager

# 3. Проверить конфигурацию
/usr/local/bin/ocserv-agent --config /etc/ocserv-agent/config.yaml --validate

# 4. Проверить права доступа
ls -la /etc/ocserv-agent/config.yaml
ls -la /etc/ocserv-agent/mtls/

# 5. Проверить зависимости
systemctl status ocserv.service
test -S /var/run/ocserv/ocserv.sock
```

#### Решения

**A. Ошибка в конфигурации:**
```bash
# Проверить YAML синтаксис
yamllint /etc/ocserv-agent/config.yaml

# Восстановить из backup
cp /var/backups/ocserv-agent/config.yaml.backup /etc/ocserv-agent/config.yaml

# Использовать минимальный конфиг
cat > /etc/ocserv-agent/config.yaml <<EOF
telemetry:
  prometheus:
    enabled: true
    address: ":9090"
portal:
  address: "portal.example.com:8080"
  insecure: true
ocserv:
  ctl_socket: /var/run/ocserv/ocserv.sock
EOF
```

**B. Проблемы с TLS сертификатами:**
```bash
# Проверить валидность сертификатов
openssl x509 -in /etc/ocserv-agent/mtls/client.crt -noout -text
openssl verify -CAfile /etc/ocserv-agent/mtls/ca.crt /etc/ocserv-agent/mtls/client.crt

# Проверить соответствие cert и key
openssl x509 -noout -modulus -in /etc/ocserv-agent/mtls/client.crt | openssl md5
openssl rsa -noout -modulus -in /etc/ocserv-agent/mtls/client.key | openssl md5

# Временно отключить TLS для теста
# В config.yaml:
portal:
  insecure: true  # ТОЛЬКО ДЛЯ ТЕСТА!
```

**C. Порт уже занят:**
```bash
# Проверить занятость порта
netstat -tlnp | grep 8080
lsof -i :8080

# Остановить конфликтующий процесс
kill <PID>

# Или изменить порт в конфиге
```

---

### Проблема: Circuit Breaker постоянно открыт

#### Симптомы
- Метрика `ocserv_agent_circuit_breaker_state == 1`
- Логи: "circuit breaker opened"
- Пользователи получают stale decisions или отказы

#### Диагностика

```bash
# 1. Проверить метрики circuit breaker
curl localhost:9090/metrics | grep circuit_breaker

# 2. Проверить логи portal запросов
journalctl -u ocserv-agent -f | grep portal

# 3. Проверить доступность portal
curl -v https://portal.example.com:8080/health
grpcurl -vv portal.example.com:8080 list

# 4. Проверить mTLS подключение
openssl s_client -connect portal.example.com:8080 \
  -cert /etc/ocserv-agent/mtls/client.crt \
  -key /etc/ocserv-agent/mtls/client.key \
  -CAfile /etc/ocserv-agent/mtls/ca.crt

# 5. Проверить сетевую связность
ping portal.example.com
traceroute portal.example.com
telnet portal.example.com 8080
```

#### Решения

**A. Portal недоступен:**
```bash
# Проверить статус portal
ssh portal.example.com "systemctl status ocserv-portal"

# Проверить firewall
ssh portal.example.com "firewall-cmd --list-ports"

# Временно переключить на fail_mode: stale
# В config.yaml:
resilience:
  fail_mode: stale  # Использовать закешированные данные
```

**B. TLS/mTLS проблемы:**
```bash
# Проверить CA сертификат
curl -v --cacert /etc/ocserv-agent/mtls/ca.crt https://portal.example.com:8080

# Обновить сертификаты
# (см. раздел "Ротация сертификатов")

# Временно включить insecure (ТОЛЬКО ДЛЯ ТЕСТА!)
portal:
  insecure: true
```

**C. Тайминг проблемы:**
```bash
# Увеличить timeout в конфиге
portal:
  timeout: 30s  # было 10s

# Увеличить failure threshold
resilience:
  circuit_breaker:
    failure_threshold: 10  # было 5
    timeout: 120s  # было 60s

# Перезапустить agent
systemctl restart ocserv-agent
```

---

### Проблема: Высокий error rate команд

#### Симптомы
- Метрика `ocserv_agent_command_errors_total` растет
- Логи показывают ошибки выполнения occtl команд

#### Диагностика

```bash
# 1. Проверить метрики ошибок
curl localhost:9090/metrics | grep command_errors

# 2. Проверить логи с ошибками
journalctl -u ocserv-agent -p err -n 100

# 3. Проверить статус ocserv
systemctl status ocserv
systemctl is-active ocserv

# 4. Проверить доступность socket
ls -la /var/run/ocserv/ocserv.sock
stat /var/run/ocserv/ocserv.sock

# 5. Проверить права доступа
sudo -u ocserv-agent test -r /var/run/ocserv/ocserv.sock && echo OK || echo FAIL

# 6. Попробовать вручную выполнить команду
sudo -u ocserv-agent occtl show status
```

#### Решения

**A. ocserv daemon остановлен:**
```bash
# Проверить и запустить
systemctl status ocserv
systemctl start ocserv

# Проверить автозапуск
systemctl enable ocserv

# Проверить зависимости в systemd
systemctl edit ocserv-agent
# Добавить:
[Unit]
Requires=ocserv.service
After=ocserv.service
```

**B. Проблемы с правами доступа:**
```bash
# Проверить группу пользователя
id ocserv-agent

# Добавить в группу ocserv (если нужно)
usermod -a -G ocserv ocserv-agent

# Проверить permissions на socket
chmod 660 /var/run/ocserv/ocserv.sock
chgrp ocserv /var/run/ocserv/ocserv.sock

# В ocserv.conf:
socket-file-prefix = /var/run/ocserv/
socket-file-permissions = 660
```

**C. Socket timeout:**
```bash
# Увеличить timeout в конфиге agent
ipc:
  socket_path: /var/run/ocserv/ocserv.sock
  timeout: 30s  # было 10s

# Перезапустить agent
systemctl restart ocserv-agent
```

---

### Проблема: Низкий cache hit rate

#### Симптомы
- Cache hit rate < 50%
- Высокая нагрузка на portal
- Медленные авторизации

#### Диагностика

```bash
# Проверить cache метрики
curl localhost:9090/metrics | grep cache

# Вычислить hit rate
echo "Cache hit rate:"
curl -s localhost:9090/metrics | awk '
/ocserv_agent_cache_hits_total/ {hits=$2}
/ocserv_agent_cache_requests_total/ {reqs=$2}
END {print hits/reqs*100 "%"}
'

# Проверить TTL конфигурацию
grep -A 5 "cache:" /etc/ocserv-agent/config.yaml
```

#### Решения

**A. Увеличить TTL:**
```bash
# Отредактировать конфиг
vim /etc/ocserv-agent/config.yaml

resilience:
  cache:
    ttl: 15m        # было 5m
    stale_ttl: 4h   # было 1h

# Применить
systemctl restart ocserv-agent
```

**B. Проверить паттерны использования:**
```bash
# Анализ логов авторизации
journalctl -u ocserv-agent | grep CheckPolicy | \
  awk '{print $NF}' | sort | uniq -c | sort -rn | head

# Если много уникальных пользователей - это нормально
# Если одни и те же пользователи - проверить cache
```

---

### Проблема: Высокое использование памяти

#### Симптомы
- `go_memory_alloc_bytes > 400MB`
- Systemd OOM killer убивает процесс

#### Диагностика

```bash
# 1. Проверить текущее использование
curl localhost:9090/metrics | grep go_memory

# 2. Проверить systemd статистику
systemctl status ocserv-agent | grep Memory

# 3. Проверить через top
top -p $(pgrep ocserv-agent)

# 4. Получить memory profile (если pprof включен)
curl http://localhost:6060/debug/pprof/heap > heap.prof
go tool pprof -http=:8081 heap.prof
```

#### Решения

**A. Увеличить лимит памяти:**
```bash
# Отредактировать systemd unit
systemctl edit ocserv-agent

[Service]
MemoryLimit=1G
MemoryHigh=768M

# Применить
systemctl daemon-reload
systemctl restart ocserv-agent
```

**B. Оптимизировать cache:**
```bash
# Уменьшить размер cache (если реализовано)
resilience:
  cache:
    max_size: 10000  # ограничить количество записей

# Уменьшить TTL
resilience:
  cache:
    ttl: 3m
    stale_ttl: 30m
```

**C. Перезапуск при высокой памяти (workaround):**
```bash
# Добавить watchdog
systemctl edit ocserv-agent

[Service]
WatchdogSec=60s
```

---

## Процедуры обслуживания

### Перезапуск сервиса

#### Graceful restart

```bash
# 1. Проверить текущее состояние
systemctl status ocserv-agent
curl localhost:9090/metrics | grep active_sessions

# 2. Graceful stop (SIGTERM)
systemctl stop ocserv-agent

# 3. Проверить, что остановился
systemctl is-active ocserv-agent

# 4. Запустить снова
systemctl start ocserv-agent

# 5. Проверить логи запуска
journalctl -u ocserv-agent -f
```

#### Restart без downtime (если поддерживается reload)

```bash
# Reload конфигурации (SIGHUP)
systemctl reload ocserv-agent

# Или restart
systemctl restart ocserv-agent
```

### Обновление конфигурации

```bash
# 1. Создать backup
cp /etc/ocserv-agent/config.yaml \
   /var/backups/ocserv-agent/config.yaml.$(date +%Y%m%d_%H%M%S)

# 2. Проверить синтаксис нового конфига
yamllint /etc/ocserv-agent/config.yaml

# 3. Валидация конфига (если поддерживается)
/usr/local/bin/ocserv-agent --config /etc/ocserv-agent/config.yaml --validate

# 4. Применить (reload или restart)
systemctl reload ocserv-agent

# 5. Проверить логи
journalctl -u ocserv-agent -f -n 50

# 6. Проверить метрики
curl localhost:9090/metrics | head -20
```

### Ротация сертификатов

#### Подготовка новых сертификатов

```bash
# 1. Создать директорию для новых сертификатов
mkdir -p /etc/ocserv-agent/mtls-new

# 2. Получить новые сертификаты (из portal PKI или вручную)
# Пример с cfssl:
cfssl gencert \
  -ca=/path/to/ca.pem \
  -ca-key=/path/to/ca-key.pem \
  -config=/path/to/ca-config.json \
  -profile=client \
  agent-csr.json | cfssljson -bare /etc/ocserv-agent/mtls-new/client

# 3. Проверить новые сертификаты
openssl x509 -in /etc/ocserv-agent/mtls-new/client.crt -noout -text
openssl verify -CAfile /path/to/ca.crt /etc/ocserv-agent/mtls-new/client.crt
```

#### Применение новых сертификатов

```bash
# 1. Backup старых сертификатов
cp -r /etc/ocserv-agent/mtls /etc/ocserv-agent/mtls.backup.$(date +%Y%m%d)

# 2. Atomic замена
mv /etc/ocserv-agent/mtls-new /etc/ocserv-agent/mtls-temp
mv /etc/ocserv-agent/mtls /etc/ocserv-agent/mtls-old
mv /etc/ocserv-agent/mtls-temp /etc/ocserv-agent/mtls

# 3. Проверить права
chown -R ocserv-agent:ocserv-agent /etc/ocserv-agent/mtls
chmod 600 /etc/ocserv-agent/mtls/*.key
chmod 644 /etc/ocserv-agent/mtls/*.crt

# 4. Рестарт agent
systemctl restart ocserv-agent

# 5. Проверить подключение к portal
journalctl -u ocserv-agent -f | grep portal

# 6. Проверить circuit breaker
curl localhost:9090/metrics | grep circuit_breaker_state
# Должно быть 0 (closed)

# 7. Если всё работает - удалить старые
rm -rf /etc/ocserv-agent/mtls-old
```

#### Rollback при проблемах

```bash
# Если что-то пошло не так:
systemctl stop ocserv-agent
mv /etc/ocserv-agent/mtls /etc/ocserv-agent/mtls-failed
mv /etc/ocserv-agent/mtls-old /etc/ocserv-agent/mtls
systemctl start ocserv-agent
```

---

## Резервное копирование

### Что бэкапить

| Компонент | Путь | Частота | Критичность |
|-----------|------|---------|-------------|
| Конфигурация | `/etc/ocserv-agent/` | Перед изменениями | Критично |
| TLS сертификаты | `/etc/ocserv-agent/mtls/` | Еженедельно | Критично |
| Per-user configs | `/etc/ocserv/config-per-user/` | Ежедневно | Высокая |
| Логи | `journalctl export` | Ежедневно | Средняя |

### Скрипт автоматического backup

```bash
#!/bin/bash
# /usr/local/bin/ocserv-agent-backup.sh

BACKUP_DIR="/var/backups/ocserv-agent"
DATE=$(date +%Y%m%d_%H%M%S)
RETENTION_DAYS=30

# Создать директорию
mkdir -p "$BACKUP_DIR"

# Backup конфигурации
tar -czf "$BACKUP_DIR/config-$DATE.tar.gz" \
  /etc/ocserv-agent/ \
  --exclude='*.log'

# Backup per-user configs
tar -czf "$BACKUP_DIR/per-user-configs-$DATE.tar.gz" \
  /etc/ocserv/config-per-user/

# Backup TLS сертификатов
tar -czf "$BACKUP_DIR/mtls-certs-$DATE.tar.gz" \
  /etc/ocserv-agent/mtls/

# Экспорт логов за последние 24 часа
journalctl -u ocserv-agent --since "24 hours ago" \
  --output=export > "$BACKUP_DIR/logs-$DATE.journal"

# Очистка старых backup'ов
find "$BACKUP_DIR" -name "*.tar.gz" -mtime +$RETENTION_DAYS -delete
find "$BACKUP_DIR" -name "*.journal" -mtime +$RETENTION_DAYS -delete

# Проверка успешности
if [ $? -eq 0 ]; then
  echo "Backup успешно создан: $BACKUP_DIR"
  exit 0
else
  echo "Ошибка при создании backup"
  exit 1
fi
```

### Настройка cron

```bash
# Установка cron задачи
cat > /etc/cron.d/ocserv-agent-backup <<EOF
# Ежедневный backup в 02:00
0 2 * * * root /usr/local/bin/ocserv-agent-backup.sh >> /var/log/ocserv-agent-backup.log 2>&1
EOF

# Проверка
chmod 644 /etc/cron.d/ocserv-agent-backup
systemctl restart cron
```

### Восстановление из backup

```bash
# 1. Остановить agent
systemctl stop ocserv-agent

# 2. Восстановить конфигурацию
cd /var/backups/ocserv-agent
tar -xzf config-YYYYMMDD_HHMMSS.tar.gz -C /

# 3. Восстановить per-user configs
tar -xzf per-user-configs-YYYYMMDD_HHMMSS.tar.gz -C /

# 4. Восстановить сертификаты
tar -xzf mtls-certs-YYYYMMDD_HHMMSS.tar.gz -C /

# 5. Проверить права доступа
chown -R ocserv-agent:ocserv-agent /etc/ocserv-agent
chmod 600 /etc/ocserv-agent/mtls/*.key

# 6. Запустить agent
systemctl start ocserv-agent

# 7. Проверить логи
journalctl -u ocserv-agent -f
```

---

## Аварийное восстановление

### Сценарий: Полная потеря сервера

#### Шаг 1: Подготовка нового сервера

```bash
# 1. Установить ОС (например, Rocky Linux 9)
# 2. Установить зависимости
dnf install -y ocserv

# 3. Установить ocserv-agent binary
curl -L https://github.com/dantte-lp/ocserv-agent/releases/download/v0.7.0/ocserv-agent-linux-amd64 \
  -o /usr/local/bin/ocserv-agent
chmod +x /usr/local/bin/ocserv-agent

# 4. Создать пользователя
useradd -r -s /sbin/nologin ocserv-agent

# 5. Создать директории
mkdir -p /etc/ocserv-agent/{mtls,}
mkdir -p /var/lib/ocserv-agent
mkdir -p /var/log/ocserv-agent
chown -R ocserv-agent:ocserv-agent /var/lib/ocserv-agent /var/log/ocserv-agent
```

#### Шаг 2: Восстановление конфигурации

```bash
# 1. Скопировать backup с удаленного хранилища
scp backup-server:/backups/ocserv-agent/config-latest.tar.gz /tmp/

# 2. Распаковать
tar -xzf /tmp/config-latest.tar.gz -C /

# 3. Восстановить сертификаты
scp backup-server:/backups/ocserv-agent/mtls-certs-latest.tar.gz /tmp/
tar -xzf /tmp/mtls-certs-latest.tar.gz -C /

# 4. Проверить права
chown -R ocserv-agent:ocserv-agent /etc/ocserv-agent
chmod 600 /etc/ocserv-agent/mtls/*.key
```

#### Шаг 3: Запуск сервисов

```bash
# 1. Скопировать systemd unit
curl -L https://raw.githubusercontent.com/dantte-lp/ocserv-agent/main/deploy/systemd/ocserv-agent.service \
  -o /etc/systemd/system/ocserv-agent.service

# 2. Reload systemd
systemctl daemon-reload

# 3. Запустить ocserv
systemctl start ocserv
systemctl enable ocserv

# 4. Запустить ocserv-agent
systemctl start ocserv-agent
systemctl enable ocserv-agent

# 5. Проверить статус
systemctl status ocserv
systemctl status ocserv-agent
```

#### Шаг 4: Проверка работоспособности

```bash
# 1. Проверить метрики
curl http://localhost:9090/metrics

# 2. Проверить подключение к portal
journalctl -u ocserv-agent -f | grep portal

# 3. Тестовая VPN сессия
# (подключиться тестовым пользователем)

# 4. Проверить алерты
# (в Grafana/Prometheus)
```

---

## Безопасность

### Регулярные проверки безопасности

#### Еженедельно

```bash
# 1. Проверка сроков действия сертификатов
openssl x509 -in /etc/ocserv-agent/mtls/client.crt -noout -enddate

# 2. Проверка обновлений безопасности
dnf check-update | grep security

# 3. Проверка логов на подозрительную активность
journalctl -u ocserv-agent --since "7 days ago" | grep -i "error\|fail\|denied"

# 4. Проверка открытых портов
netstat -tlnp | grep ocserv-agent
```

#### Ежемесячно

```bash
# 1. Обновление зависимостей
dnf update -y

# 2. Аудит файловой системы
find /etc/ocserv-agent -type f -ls
find /var/lib/ocserv-agent -type f -ls

# 3. Проверка прав доступа
ls -laR /etc/ocserv-agent
ls -laR /var/lib/ocserv-agent

# 4. Ротация логов
journalctl --rotate
journalctl --vacuum-time=30d
```

### Security Hardening Checklist

- [ ] Включен mTLS между agent и portal
- [ ] TLS минимум версии 1.3
- [ ] Сертификаты не истекают в ближайшие 30 дней
- [ ] Systemd security directives включены (см. systemd unit)
- [ ] Firewall настроен (только необходимые порты)
- [ ] SELinux/AppArmor активирован
- [ ] Логи отправляются в централизованную систему
- [ ] Metrics защищены (firewall или authentication)
- [ ] Regular backups настроены
- [ ] Обновления безопасности применяются регулярно

---

## Обновление версий

### Процедура обновления agent

#### Pre-update checklist

```bash
# 1. Проверить changelog
curl https://github.com/dantte-lp/ocserv-agent/releases/tag/v0.7.1

# 2. Создать полный backup
/usr/local/bin/ocserv-agent-backup.sh

# 3. Проверить текущую версию
/usr/local/bin/ocserv-agent --version

# 4. Скачать новую версию
curl -L https://github.com/dantte-lp/ocserv-agent/releases/download/v0.7.1/ocserv-agent-linux-amd64 \
  -o /tmp/ocserv-agent-new
chmod +x /tmp/ocserv-agent-new

# 5. Проверить новую версию
/tmp/ocserv-agent-new --version
```

#### Обновление

```bash
# 1. Остановить agent
systemctl stop ocserv-agent

# 2. Backup текущего binary
cp /usr/local/bin/ocserv-agent /usr/local/bin/ocserv-agent.backup

# 3. Заменить binary
mv /tmp/ocserv-agent-new /usr/local/bin/ocserv-agent

# 4. Проверить config compatibility (если есть breaking changes)
/usr/local/bin/ocserv-agent --config /etc/ocserv-agent/config.yaml --validate

# 5. Запустить новую версию
systemctl start ocserv-agent

# 6. Проверить логи
journalctl -u ocserv-agent -f -n 100

# 7. Проверить метрики
curl localhost:9090/metrics | head -20

# 8. Smoke test
# (провести базовые проверки)
```

#### Rollback при проблемах

```bash
# Если новая версия не работает:
systemctl stop ocserv-agent
mv /usr/local/bin/ocserv-agent.backup /usr/local/bin/ocserv-agent
systemctl start ocserv-agent
journalctl -u ocserv-agent -f
```

---

## Приложения

### Полезные команды

```bash
# Статус agent
systemctl status ocserv-agent

# Логи в реальном времени
journalctl -u ocserv-agent -f

# Метрики
curl localhost:9090/metrics

# Проверка конфига
/usr/local/bin/ocserv-agent --config /etc/ocserv-agent/config.yaml --validate

# Перезапуск
systemctl restart ocserv-agent

# Просмотр конфигурации
cat /etc/ocserv-agent/config.yaml

# Проверка версии
/usr/local/bin/ocserv-agent --version
```

### Контакты

| Роль | Контакт |
|------|---------|
| **Техподдержка** | support@example.com |
| **Экстренная связь** | +998-XX-XXX-XXXX |
| **GitHub Issues** | https://github.com/dantte-lp/ocserv-agent/issues |
| **Документация** | https://github.com/dantte-lp/ocserv-agent |

---

**Версия документа:** 1.0.0
**Дата обновления:** 2025-12-27
**Статус:** Production Ready
