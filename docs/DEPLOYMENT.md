# Руководство по развертыванию ocserv-agent

![Version](https://img.shields.io/badge/version-0.7.0-blue)
![Status](https://img.shields.io/badge/status-production--ready-green)

Комплексное руководство по развертыванию ocserv-agent в production окружении.

---

## Содержание

- [Обзор](#обзор)
- [Требования](#требования)
- [Варианты развертывания](#варианты-развертывания)
  - [Docker/Podman](#dockerpodman)
  - [Kubernetes с Helm](#kubernetes-с-helm)
  - [Systemd (bare metal)](#systemd-bare-metal)
- [Конфигурация](#конфигурация)
- [Безопасность](#безопасность)
- [Мониторинг](#мониторинг)
- [Troubleshooting](#troubleshooting)

---

## Обзор

ocserv-agent — это gRPC агент для удаленного управления OpenConnect VPN серверами. Он обеспечивает:

- Централизованное управление через ocserv-portal
- Авторизацию пользователей в реальном времени
- Управление VPN сессиями
- Метрики и мониторинг
- Resilience (Circuit Breaker, Cache, Fail modes)

**Архитектура:**

```
Portal (ocserv-portal)
    ↓ gRPC + mTLS
ocserv-agent
    ↓ Unix Socket
ocserv daemon
```

---

## Требования

### Системные требования

| Компонент | Минимум | Рекомендуется |
|-----------|---------|---------------|
| CPU | 1 core | 2 cores |
| RAM | 256 MB | 512 MB |
| Disk | 100 MB | 1 GB |
| OS | Linux kernel 3.10+ | Linux kernel 5.x+ |

### Зависимости

**Обязательные:**
- ocserv 1.1.0+ (должен быть установлен и запущен)
- Unix socket: `/var/run/ocserv/ocserv.sock` (настраивается)

**Опциональные:**
- TLS сертификаты (для gRPC server)
- mTLS сертификаты (для portal integration)
- Prometheus/VictoriaMetrics (для метрик)

### Сетевые порты

| Порт | Протокол | Назначение | Обязательный |
|------|----------|------------|--------------|
| 8080 | gRPC | gRPC API server | Да |
| 9090 | HTTP | Prometheus metrics | Нет (рекомендуется) |

---

## Варианты развертывания

### Docker/Podman

#### Быстрый старт

```bash
# 1. Скачать образ
podman pull ghcr.io/dantte-lp/ocserv-agent:0.7.0

# 2. Создать конфигурацию
cat > config.yaml <<EOF
telemetry:
  service_name: ocserv-agent
  prometheus:
    enabled: true
    address: ":9090"

portal:
  address: "portal.example.com:8080"
  tls_cert: /etc/ocserv-agent/mtls/client.crt
  tls_key: /etc/ocserv-agent/mtls/client.key
  tls_ca: /etc/ocserv-agent/mtls/ca.crt
  timeout: 10s

ocserv:
  ctl_socket: /var/run/ocserv/ocserv.sock

resilience:
  fail_mode: stale
  circuit_breaker:
    failure_threshold: 5
    timeout: 60s
  cache:
    ttl: 5m
    stale_ttl: 1h
EOF

# 3. Запустить контейнер
podman run -d \
  --name ocserv-agent \
  -v $(pwd)/config.yaml:/etc/ocserv-agent/config.yaml:ro \
  -v /var/run/ocserv:/var/run/ocserv:ro \
  -v /path/to/mtls-certs:/etc/ocserv-agent/mtls:ro \
  -p 8080:8080 \
  -p 9090:9090 \
  --restart unless-stopped \
  ghcr.io/dantte-lp/ocserv-agent:0.7.0
```

#### Docker Compose

```yaml
# docker-compose.yaml
version: '3.8'

services:
  ocserv-agent:
    image: ghcr.io/dantte-lp/ocserv-agent:0.7.0
    container_name: ocserv-agent
    restart: unless-stopped

    ports:
      - "8080:8080"  # gRPC
      - "9090:9090"  # Metrics

    volumes:
      - ./config.yaml:/etc/ocserv-agent/config.yaml:ro
      - /var/run/ocserv:/var/run/ocserv:ro
      - ./mtls-certs:/etc/ocserv-agent/mtls:ro

    environment:
      - LOG_LEVEL=info
      - CONFIG_PATH=/etc/ocserv-agent/config.yaml

    healthcheck:
      test: ["/usr/local/bin/ocserv-agent", "healthcheck"]
      interval: 30s
      timeout: 5s
      retries: 3
      start_period: 10s

    networks:
      - vpn-network

networks:
  vpn-network:
    driver: bridge
```

Запуск:

```bash
docker-compose up -d
docker-compose logs -f ocserv-agent
```

#### Production сборка

```bash
# Сборка production образа
cd /path/to/ocserv-agent
podman build \
  -f build/Containerfile.production \
  -t ocserv-agent:0.7.0 \
  --build-arg VERSION=0.7.0 \
  --build-arg COMMIT=$(git rev-parse --short HEAD) \
  --build-arg BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ") \
  .

# Проверка образа
podman images ocserv-agent
podman run --rm ocserv-agent:0.7.0 --version
```

---

### Kubernetes с Helm

#### Установка Helm Chart

```bash
# 1. Добавить Helm репозиторий (если есть)
helm repo add ocserv-agent https://charts.example.com/ocserv-agent
helm repo update

# Или использовать локальный chart
cd deploy/helm/ocserv-agent

# 2. Создать namespace
kubectl create namespace vpn-system

# 3. Создать secrets для TLS и mTLS
kubectl create secret tls ocserv-agent-tls \
  --cert=/path/to/tls/server.crt \
  --key=/path/to/tls/server.key \
  -n vpn-system

kubectl create secret generic ocserv-agent-mtls \
  --from-file=client.crt=/path/to/mtls/client.crt \
  --from-file=client.key=/path/to/mtls/client.key \
  --from-file=ca.crt=/path/to/mtls/ca.crt \
  -n vpn-system

# 4. Создать values файл
cat > values-production.yaml <<EOF
replicaCount: 2

image:
  repository: ghcr.io/dantte-lp/ocserv-agent
  tag: "0.7.0"
  pullPolicy: IfNotPresent

resources:
  requests:
    cpu: 100m
    memory: 128Mi
  limits:
    cpu: 500m
    memory: 256Mi

config:
  logLevel: info
  logFormat: json

  portal:
    endpoint: "ocserv-portal.vpn-system.svc.cluster.local:8080"
    mtls:
      enabled: true
    timeout: 10s

  circuitBreaker:
    failMode: stale
    maxFailures: 5
    timeout: 60s

tls:
  existingSecret: "ocserv-agent-tls"

mtls:
  existingSecret: "ocserv-agent-mtls"

metrics:
  enabled: true
  serviceMonitor:
    enabled: true
    interval: 30s

autoscaling:
  enabled: true
  minReplicas: 2
  maxReplicas: 5
  targetCPUUtilizationPercentage: 80

podDisruptionBudget:
  enabled: true
  minAvailable: 1
EOF

# 5. Установить chart
helm install ocserv-agent . \
  -n vpn-system \
  -f values-production.yaml

# 6. Проверить статус
kubectl get pods -n vpn-system
kubectl logs -n vpn-system -l app.kubernetes.io/name=ocserv-agent
```

#### Обновление

```bash
# Обновить до новой версии
helm upgrade ocserv-agent . \
  -n vpn-system \
  -f values-production.yaml \
  --set image.tag=0.8.0

# Откат
helm rollback ocserv-agent -n vpn-system
```

#### Удаление

```bash
helm uninstall ocserv-agent -n vpn-system
kubectl delete namespace vpn-system
```

---

### Systemd (bare metal)

#### Установка

```bash
# 1. Скачать бинарник
VERSION=0.7.0
ARCH=amd64  # или arm64
wget https://github.com/dantte-lp/ocserv-agent/releases/download/v${VERSION}/ocserv-agent-${VERSION}-linux-${ARCH}.tar.gz
tar -xzf ocserv-agent-${VERSION}-linux-${ARCH}.tar.gz
sudo mv ocserv-agent /usr/local/bin/
sudo chmod +x /usr/local/bin/ocserv-agent

# 2. Создать пользователя
sudo useradd -r -s /bin/false ocserv-agent

# 3. Создать директории
sudo mkdir -p /etc/ocserv-agent
sudo mkdir -p /var/log/ocserv-agent
sudo mkdir -p /var/run/ocserv-agent
sudo chown -R ocserv-agent:ocserv-agent /var/log/ocserv-agent /var/run/ocserv-agent

# 4. Создать конфигурацию
sudo cat > /etc/ocserv-agent/config.yaml <<EOF
telemetry:
  service_name: ocserv-agent
  service_version: "0.7.0"
  environment: production

  prometheus:
    enabled: true
    address: ":9090"

logging:
  level: info
  format: json

grpc:
  address: ":8080"
  tls:
    enabled: true
    cert_path: /etc/ocserv-agent/tls/server.crt
    key_path: /etc/ocserv-agent/tls/server.key

portal:
  address: "portal.example.com:8080"
  tls_cert: /etc/ocserv-agent/mtls/client.crt
  tls_key: /etc/ocserv-agent/mtls/client.key
  tls_ca: /etc/ocserv-agent/mtls/ca.crt
  timeout: 10s

ocserv:
  ctl_socket: /var/run/ocserv/ocserv.sock

resilience:
  fail_mode: stale
  circuit_breaker:
    failure_threshold: 5
    timeout: 60s
  cache:
    ttl: 5m
    stale_ttl: 1h
EOF

sudo chown root:ocserv-agent /etc/ocserv-agent/config.yaml
sudo chmod 640 /etc/ocserv-agent/config.yaml

# 5. Создать systemd unit
sudo cat > /etc/systemd/system/ocserv-agent.service <<EOF
[Unit]
Description=OpenConnect VPN Server Agent
Documentation=https://github.com/dantte-lp/ocserv-agent
After=network.target ocserv.service
Requires=ocserv.service

[Service]
Type=simple
User=ocserv-agent
Group=ocserv-agent

ExecStart=/usr/local/bin/ocserv-agent -config /etc/ocserv-agent/config.yaml
ExecReload=/bin/kill -HUP \$MAINPID

Restart=on-failure
RestartSec=10s

# Security settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/log/ocserv-agent /var/run/ocserv-agent
ReadOnlyPaths=/var/run/ocserv

# Resource limits
LimitNOFILE=65536
LimitNPROC=512

[Install]
WantedBy=multi-user.target
EOF

# 6. Запустить сервис
sudo systemctl daemon-reload
sudo systemctl enable ocserv-agent
sudo systemctl start ocserv-agent

# 7. Проверить статус
sudo systemctl status ocserv-agent
sudo journalctl -u ocserv-agent -f
```

---

## Конфигурация

### Полная структура config.yaml

```yaml
# Telemetry и мониторинг
telemetry:
  service_name: ocserv-agent
  service_version: "0.7.0"
  environment: production

  # Prometheus метрики
  prometheus:
    enabled: true
    address: ":9090"

  # OTLP экспорт (опционально)
  otlp:
    enabled: false
    endpoint: "otel-collector:4317"
    insecure: false

# Логирование
logging:
  level: info      # debug, info, warn, error
  format: json     # json, text
  output: stdout   # stdout, file
  file_path: /var/log/ocserv-agent/agent.log

# gRPC Server
grpc:
  address: ":8080"
  tls:
    enabled: true
    cert_path: /etc/ocserv-agent/tls/server.crt
    key_path: /etc/ocserv-agent/tls/server.key

# Portal Integration
portal:
  address: "portal.example.com:8080"
  tls_cert: /etc/ocserv-agent/mtls/client.crt
  tls_key: /etc/ocserv-agent/mtls/client.key
  tls_ca: /etc/ocserv-agent/mtls/ca.crt
  timeout: 10s
  insecure: false

# IPC Configuration
ipc:
  socket_path: /var/run/ocserv-agent/ipc.sock
  timeout: 5s

# Ocserv Integration
ocserv:
  ctl_socket: /var/run/ocserv/ocserv.sock
  systemd_service: ocserv.service
  config_path: /etc/ocserv/ocserv.conf

# Resilience Settings
resilience:
  fail_mode: stale  # open, close, stale

  circuit_breaker:
    max_requests: 10
    interval: 10s
    timeout: 60s
    failure_threshold: 5

  cache:
    ttl: 5m
    stale_ttl: 1h
    max_size: 10000

# Security
security:
  sudo_user: ""
  max_command_timeout: 30s
```

### Переменные окружения

```bash
# Приоритет: ENV > config.yaml

# Logging
export LOG_LEVEL=info
export LOG_FORMAT=json

# Portal
export PORTAL_ADDRESS=portal.example.com:8080
export PORTAL_TLS_CERT=/path/to/client.crt
export PORTAL_TLS_KEY=/path/to/client.key
export PORTAL_TLS_CA=/path/to/ca.crt

# Metrics
export PROMETHEUS_ENABLED=true
export PROMETHEUS_ADDRESS=:9090
```

---

## Безопасность

### TLS Сертификаты

#### Генерация self-signed (для тестирования)

```bash
# Генерация через встроенную команду
ocserv-agent gencert \
  -output /etc/ocserv-agent/tls \
  -hostname agent.example.com

# Или вручную через openssl
openssl req -x509 -newkey rsa:4096 \
  -keyout /etc/ocserv-agent/tls/server.key \
  -out /etc/ocserv-agent/tls/server.crt \
  -days 365 -nodes \
  -subj "/CN=agent.example.com"
```

#### Production сертификаты

Используйте корпоративный PKI или Let's Encrypt:

```bash
# cert-manager в Kubernetes
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: ocserv-agent-tls
  namespace: vpn-system
spec:
  secretName: ocserv-agent-tls
  issuerRef:
    name: letsencrypt-prod
    kind: ClusterIssuer
  dnsNames:
  - agent.example.com
```

### mTLS для Portal

```bash
# 1. Получить сертификаты от portal PKI
# 2. Разместить в /etc/ocserv-agent/mtls/
#    - client.crt
#    - client.key
#    - ca.crt

# 3. Установить права доступа
sudo chown -R ocserv-agent:ocserv-agent /etc/ocserv-agent/mtls
sudo chmod 600 /etc/ocserv-agent/mtls/client.key
sudo chmod 644 /etc/ocserv-agent/mtls/client.crt
sudo chmod 644 /etc/ocserv-agent/mtls/ca.crt
```

### Firewall

```bash
# Разрешить только необходимые порты
sudo firewall-cmd --permanent --add-port=8080/tcp  # gRPC
sudo firewall-cmd --permanent --add-port=9090/tcp  # Metrics (только внутренняя сеть!)
sudo firewall-cmd --reload

# Ограничить доступ к metrics
sudo firewall-cmd --permanent --zone=internal --add-source=10.0.0.0/8
sudo firewall-cmd --reload
```

---

## Мониторинг

### Prometheus Metrics

Доступные метрики на `/metrics` (порт 9090):

```prometheus
# Agent метрики
ocserv_agent_up                              # Статус агента (1 = работает)
ocserv_agent_commands_total                  # Количество команд
ocserv_agent_command_duration_seconds        # Длительность команд
ocserv_agent_command_errors_total            # Ошибки команд
ocserv_agent_active_sessions                 # Активные VPN сессии
ocserv_agent_connected_users                 # Подключенные пользователи

# gRPC метрики
grpc_server_requests_total                   # gRPC запросы
grpc_server_request_duration_seconds         # Длительность gRPC

# Circuit Breaker
ocserv_agent_circuit_breaker_state           # Состояние CB (0=closed, 1=open, 2=half-open)
ocserv_agent_circuit_breaker_failures_total  # Количество сбоев

# Cache
ocserv_agent_cache_hits_total                # Cache hits
ocserv_agent_cache_misses_total              # Cache misses
ocserv_agent_cache_size                      # Размер кеша

# Go runtime
go_goroutines                                # Количество goroutines
go_memory_alloc_bytes                        # Выделенная память
go_memory_sys_bytes                          # Память от OS
```

### Prometheus конфигурация

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'ocserv-agent'
    static_configs:
      - targets:
        - 'agent1.example.com:9090'
        - 'agent2.example.com:9090'
    scrape_interval: 30s
    scrape_timeout: 10s
```

### Grafana Dashboard

Импортируйте готовый dashboard (ID: TBD) или создайте свой:

```json
{
  "dashboard": {
    "title": "ocserv-agent Monitoring",
    "panels": [
      {
        "title": "Active VPN Sessions",
        "targets": [
          {
            "expr": "ocserv_agent_active_sessions"
          }
        ]
      },
      {
        "title": "Circuit Breaker State",
        "targets": [
          {
            "expr": "ocserv_agent_circuit_breaker_state"
          }
        ]
      }
    ]
  }
}
```

### Health Checks

```bash
# Kubernetes liveness/readiness
livenessProbe:
  exec:
    command: ["/usr/local/bin/ocserv-agent", "healthcheck"]
  initialDelaySeconds: 10
  periodSeconds: 30

# HTTP health endpoint (если добавлен)
curl http://localhost:9090/health
```

---

## Troubleshooting

### Проблема: Agent не запускается

**Симптомы:**
```
systemctl status ocserv-agent
● ocserv-agent.service - failed
```

**Диагностика:**
```bash
# 1. Проверить логи
sudo journalctl -u ocserv-agent -n 50

# 2. Проверить конфигурацию
ocserv-agent -config /etc/ocserv-agent/config.yaml --validate

# 3. Проверить права доступа
ls -la /etc/ocserv-agent/
ls -la /var/run/ocserv/ocserv.sock

# 4. Проверить сертификаты
openssl x509 -in /etc/ocserv-agent/tls/server.crt -text -noout
```

**Решение:**
- Исправить синтаксис config.yaml
- Добавить пользователя в группу ocserv: `sudo usermod -aG ocserv ocserv-agent`
- Проверить сертификаты и пути к ним

---

### Проблема: Portal недоступен

**Симптомы:**
```
circuit_breaker_state = 1 (open)
command_errors_total растет
```

**Диагностика:**
```bash
# 1. Проверить connectivity
curl -v --insecure https://portal.example.com:8080

# 2. Проверить mTLS сертификаты
openssl s_client -connect portal.example.com:8080 \
  -cert /etc/ocserv-agent/mtls/client.crt \
  -key /etc/ocserv-agent/mtls/client.key \
  -CAfile /etc/ocserv-agent/mtls/ca.crt

# 3. Проверить метрики
curl http://localhost:9090/metrics | grep circuit_breaker

# 4. Проверить логи
journalctl -u ocserv-agent | grep -i portal
```

**Решение:**
- Проверить сетевую доступность portal
- Обновить mTLS сертификаты (возможно истекли)
- Настроить `fail_mode: stale` для работы без portal

---

### Проблема: Ocserv socket недоступен

**Симптомы:**
```
ERROR: failed to connect to occtl socket
```

**Диагностика:**
```bash
# 1. Проверить socket
ls -la /var/run/ocserv/ocserv.sock
srw-rw---- 1 root ocserv 0 Dec 27 10:00 /var/run/ocserv/ocserv.sock

# 2. Проверить ocserv
systemctl status ocserv
occtl show status

# 3. Проверить группы пользователя
groups ocserv-agent
```

**Решение:**
```bash
# Добавить пользователя в группу ocserv
sudo usermod -aG ocserv ocserv-agent

# Перезапустить agent
sudo systemctl restart ocserv-agent
```

---

### Проблема: Высокое потребление памяти

**Симптомы:**
```
go_memory_alloc_bytes > 500MB
```

**Диагностика:**
```bash
# 1. Проверить метрики
curl http://localhost:9090/metrics | grep go_memory

# 2. Проверить cache size
curl http://localhost:9090/metrics | grep cache_size

# 3. Профилировать (если pprof включен)
go tool pprof http://localhost:6060/debug/pprof/heap
```

**Решение:**
- Уменьшить `cache.max_size` в config.yaml
- Уменьшить `cache.stale_ttl`
- Увеличить лимит памяти в systemd/k8s

---

## Контакты и поддержка

- GitHub Issues: https://github.com/dantte-lp/ocserv-agent/issues
- Email: devops@paymart.uz
- Документация: https://github.com/dantte-lp/ocserv-agent/docs

---

**Версия документа:** 1.0.0
**Дата обновления:** 2025-12-27
**Автор:** DevOps Team
