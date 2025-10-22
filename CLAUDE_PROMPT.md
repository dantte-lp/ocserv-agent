# Промпт для Claude Code: Разработка ocserv-agent

Ты - expert Go разработчик, специализирующийся на системном программировании, gRPC и работе с Linux системами. Твоя задача - создать production-ready агент для управления OpenConnect VPN сервером (ocserv).

## Контекст проекта

**Репозиторий:** https://github.com/dantte-lp/ocserv-agent

**Цель:** Создать легковесный агент (Go application), который устанавливается на каждый сервер с ocserv и обеспечивает удалённое управление через gRPC с использованием mTLS.

**Архитектура:**
```
Control Server (ocserv-web-panel)
    ↓ gRPC + mTLS
Agent (этот проект)
    ↓ exec/shell
ocserv daemon
```

## Технический стек

- **Go:** 1.25.1
- **gRPC:** google.golang.org/grpc v1.69.4
- **Protocol Buffers:** google.golang.org/protobuf v1.36.3
- **Logging:** github.com/rs/zerolog v1.33.0
- **OpenTelemetry:** go.opentelemetry.io/otel v1.34.0
- **Config:** gopkg.in/yaml.v3 v3.0.1

**Базовый образ:** golang:1.25-trixie (для сборки), debian:trixie-slim (runtime)

## Требования к агенту

### 1. Функциональность

**Управление ocserv:**
- Получение статуса сервера (`systemctl status ocserv`)
- Start/Stop/Restart/Reload сервера
- Выполнение команд occtl:
  - `occtl show users` - список активных пользователей
  - `occtl show status` - статус сервера
  - `occtl show stats` - статистика
  - `occtl disconnect user <username>` - отключение пользователя
  - `occtl disconnect id <id>` - отключение по ID сессии

**Управление конфигурацией:**
- Чтение конфигурационных файлов:
  - `/etc/ocserv/ocserv.conf` - главный конфиг
  - `/etc/ocserv/config-per-group/*` - групповые конфиги
  - `/etc/ocserv/config-per-user/*` - пользовательские конфиги
- Обновление конфигурации (с валидацией перед применением)
- Backup конфигов перед изменениями
- Rollback к предыдущей версии

**Мониторинг:**
- Heartbeat каждые 10-15 секунд (статус, CPU, RAM, активные сессии)
- Streaming метрик (OpenTelemetry)
- Streaming логов ocserv в реальном времени

**Управление пользователями:**
- Создание/удаление пользователей через `ocpasswd`
- Блокировка/разблокировка пользователей
- Изменение паролей
- Управление группами пользователей

### 2. Безопасность

**mTLS (обязательно):**
- Client certificate authentication
- Проверка Common Name сервера
- TLS 1.3 minimum
- Cipher suites: TLS_AES_256_GCM_SHA384, TLS_CHACHA20_POLY1305_SHA256

**Execution Security:**
- Whitelist разрешённых команд (только occtl, systemctl для ocserv)
- Валидация всех аргументов (защита от command injection)
- Запуск под отдельным пользователем (не root, sudo для occtl)
- Capability-based security (CAP_NET_ADMIN)

**Audit:**
- Логирование всех выполненных команд
- Structured logging с контекстом (admin_id, command, args, result)

### 3. Надёжность

**Health Checks (3-tier):**
1. **Tier 1 - Heartbeat** (каждые 10-15 сек)
   - Базовый статус агента
   - Системные метрики (CPU, RAM)
   - Количество активных VPN сессий

2. **Tier 2 - Deep Check** (каждые 1-2 мин)
   - ocserv процесс работает (`systemctl is-active ocserv`)
   - Порт 443 listening (`ss -tlnp | grep :443`)
   - Конфигурация валидна

3. **Tier 3 - Application Check** (on-demand)
   - End-to-end VPN connection test
   - Запускается по запросу от control server

**Reconnection Logic:**
- Exponential backoff (1s, 2s, 4s, 8s, 16s, max 60s)
- Circuit breaker pattern (5 failed attempts → wait 5 min)
- Graceful degradation (кеширование команд при отключении)

**Error Handling:**
- Все ошибки логируются с полным context
- Panic recovery с stack trace
- Retry logic для transient errors

### 4. Production Ready

**Deployment:**
- Systemd service integration
- Automatic restart on crash
- Log rotation
- Resource limits (memory, CPU)

**Configuration:**
- YAML конфигурация (`/etc/ocserv-agent/config.yaml`)
- Environment variables override
- Config hot-reload (SIGHUP)

**Observability:**
- OpenTelemetry traces для всех gRPC calls
- Prometheus metrics endpoint
- Structured JSON logs

**Testing:**
- Unit tests (>80% coverage)
- Integration tests с mock ocserv
- gRPC interceptor tests

## gRPC Protocol Definition

Используй этот Protocol Buffers файл как основу и расширяй по мере необходимости:

```protobuf
syntax = "proto3";

package agent.v1;

option go_package = "github.com/dantte-lp/ocserv-agent/pkg/proto/agent/v1";

import "google/protobuf/timestamp.proto";

// AgentService - основной сервис агента
service AgentService {
  // Bidirectional streaming для heartbeat и команд
  rpc AgentStream(stream AgentMessage) returns (stream ServerMessage);
  
  // Выполнение команды
  rpc ExecuteCommand(CommandRequest) returns (CommandResponse);
  
  // Обновление конфигурации
  rpc UpdateConfig(ConfigUpdateRequest) returns (ConfigUpdateResponse);
  
  // Streaming логов
  rpc StreamLogs(LogStreamRequest) returns (stream LogEntry);
  
  // Health check
  rpc HealthCheck(HealthCheckRequest) returns (HealthCheckResponse);
}

// Сообщения от агента к серверу
message AgentMessage {
  string agent_id = 1;
  google.protobuf.Timestamp timestamp = 2;
  
  oneof payload {
    Heartbeat heartbeat = 10;
    MetricsReport metrics = 11;
    EventNotification event = 12;
  }
}

// Heartbeat от агента
message Heartbeat {
  AgentStatus status = 1;
  SystemMetrics system = 2;
  OcservStatus ocserv = 3;
}

enum AgentStatus {
  AGENT_STATUS_UNSPECIFIED = 0;
  AGENT_STATUS_HEALTHY = 1;
  AGENT_STATUS_DEGRADED = 2;
  AGENT_STATUS_UNHEALTHY = 3;
}

message SystemMetrics {
  double cpu_usage_percent = 1;
  double memory_usage_percent = 2;
  uint64 memory_total_bytes = 3;
  uint64 memory_used_bytes = 4;
  double load_average_1m = 5;
}

message OcservStatus {
  bool is_running = 1;
  string version = 2;
  uint32 active_sessions = 3;
  uint64 total_bytes_in = 4;
  uint64 total_bytes_out = 5;
}

// Сообщения от сервера к агенту
message ServerMessage {
  string request_id = 1;
  
  oneof payload {
    CommandInstruction command = 10;
    ConfigUpdate config_update = 11;
    ControlAction action = 12;
  }
}

// Запрос на выполнение команды
message CommandRequest {
  string request_id = 1;
  string command_type = 2;
  repeated string args = 3;
  int32 timeout_seconds = 4;
}

message CommandResponse {
  string request_id = 1;
  bool success = 2;
  string stdout = 3;
  string stderr = 4;
  int32 exit_code = 5;
  string error_message = 6;
}

// Обновление конфигурации
message ConfigUpdateRequest {
  string request_id = 1;
  ConfigType config_type = 2;
  string config_name = 3;  // имя файла или пользователя/группы
  string config_content = 4;
  bool validate_only = 5;  // только валидация, не применять
  bool create_backup = 6;
}

enum ConfigType {
  CONFIG_TYPE_UNSPECIFIED = 0;
  CONFIG_TYPE_MAIN = 1;          // ocserv.conf
  CONFIG_TYPE_PER_USER = 2;      // config-per-user/
  CONFIG_TYPE_PER_GROUP = 3;     // config-per-group/
}

message ConfigUpdateResponse {
  string request_id = 1;
  bool success = 2;
  string validation_result = 3;
  string backup_path = 4;
  string error_message = 5;
}

// Streaming логов
message LogStreamRequest {
  string log_source = 1;  // "ocserv", "agent", "system"
  google.protobuf.Timestamp start_time = 2;
  bool follow = 3;  // tail -f mode
}

message LogEntry {
  google.protobuf.Timestamp timestamp = 1;
  string level = 2;
  string source = 3;
  string message = 4;
  map<string, string> fields = 5;
}

// Health Check
message HealthCheckRequest {
  int32 tier = 1;  // 1, 2, или 3
}

message HealthCheckResponse {
  bool healthy = 1;
  string status_message = 2;
  map<string, string> checks = 3;
}
```

## Структура проекта

```
ocserv-agent/
├── cmd/
│   └── agent/
│       └── main.go              # Entrypoint
├── internal/
│   ├── config/
│   │   ├── config.go            # Загрузка конфига
│   │   └── validation.go        # Валидация конфига
│   ├── grpc/
│   │   ├── server.go            # gRPC server logic
│   │   ├── handlers.go          # Обработчики RPC методов
│   │   ├── interceptors.go      # Auth, logging, metrics
│   │   └── stream.go            # Bidirectional streaming
│   ├── ocserv/
│   │   ├── manager.go           # Управление ocserv
│   │   ├── occtl.go             # Обёртка для occtl
│   │   ├── config.go            # Работа с конфигами
│   │   ├── users.go             # Управление пользователями
│   │   └── systemctl.go         # systemctl wrapper
│   ├── metrics/
│   │   ├── collector.go         # Сбор метрик
│   │   └── reporter.go          # Отправка в control server
│   ├── health/
│   │   ├── checker.go           # Health check logic
│   │   └── tiers.go             # 3-tier health checks
│   └── telemetry/
│       ├── otel.go              # OpenTelemetry setup
│       └── traces.go            # Tracing helpers
├── pkg/
│   └── proto/
│       └── agent/
│           └── v1/
│               ├── agent.proto
│               ├── agent.pb.go       # Generated
│               └── agent_grpc.pb.go  # Generated
├── deploy/
│   ├── systemd/
│   │   └── ocserv-agent.service
│   └── ansible/
│       └── install.yml
├── scripts/
│   ├── generate-certs.sh        # Генерация mTLS сертификатов
│   └── install.sh               # Установка агента
├── config.yaml.example          # Пример конфига
├── go.mod
├── go.sum
├── Makefile
├── Dockerfile
└── README.md
```

## Конфигурация агента

**`/etc/ocserv-agent/config.yaml`:**
```yaml
# Agent identification
agent_id: "server-01"
hostname: ""  # auto-detect if empty

# Control server connection
control_server:
  address: "control.example.com:9090"
  reconnect:
    initial_delay: 1s
    max_delay: 60s
    multiplier: 2
    max_attempts: 5
  circuit_breaker:
    failure_threshold: 5
    timeout: 5m

# TLS configuration
tls:
  enabled: true
  cert_file: "/etc/ocserv-agent/certs/agent.crt"
  key_file: "/etc/ocserv-agent/certs/agent.key"
  ca_file: "/etc/ocserv-agent/certs/ca.crt"
  server_name: "control-server"  # Expected CN in server cert
  min_version: "TLS1.3"

# ocserv configuration
ocserv:
  config_path: "/etc/ocserv/ocserv.conf"
  config_per_user_dir: "/etc/ocserv/config-per-user"
  config_per_group_dir: "/etc/ocserv/config-per-group"
  ctl_socket: "/var/run/occtl.socket"
  systemd_service: "ocserv"
  backup_dir: "/var/backups/ocserv-agent"

# Health checks
health:
  heartbeat_interval: 15s
  deep_check_interval: 2m
  metrics_interval: 30s

# Telemetry (OpenTelemetry)
telemetry:
  enabled: true
  endpoint: "http://uptrace:14318"
  service_name: "ocserv-agent"
  service_version: "1.0.0"
  sample_rate: 1.0

# Logging
logging:
  level: "info"  # debug, info, warn, error
  format: "json"
  output: "stdout"  # stdout, file
  file_path: "/var/log/ocserv-agent/agent.log"
  max_size_mb: 100
  max_backups: 3
  max_age_days: 30

# Security
security:
  allowed_commands:
    - "occtl"
    - "systemctl"
  sudo_user: "ocserv-agent"
  max_command_timeout: 300s
```

## Примеры использования

### 1. Запуск агента

```bash
# Development
go run cmd/agent/main.go --config config.yaml.example

# Production (systemd)
sudo systemctl start ocserv-agent
sudo journalctl -u ocserv-agent -f
```

### 2. Тестирование gRPC

```bash
# grpcurl для тестирования
grpcurl -cacert certs/ca.crt \
        -cert certs/admin.crt \
        -key certs/admin.key \
        -d '{"tier": 1}' \
        control.example.com:9090 \
        agent.v1.AgentService/HealthCheck
```

### 3. Генерация Proto

```bash
make proto

# Или вручную
protoc --go_out=. --go-grpc_out=. \
       --go_opt=paths=source_relative \
       --go-grpc_opt=paths=source_relative \
       pkg/proto/agent/v1/agent.proto
```

## Важные паттерны и практики

### 1. Context Propagation

```go
// Всегда передавай context через цепочку вызовов
func (s *Server) ExecuteCommand(ctx context.Context, req *pb.CommandRequest) (*pb.CommandResponse, error) {
    // Добавь tracing span
    ctx, span := s.tracer.Start(ctx, "ExecuteCommand")
    defer span.End()
    
    // Проверь cancellation
    if err := ctx.Err(); err != nil {
        return nil, status.Error(codes.Canceled, "context canceled")
    }
    
    // Передай контекст дальше
    return s.ocservManager.RunCommand(ctx, req.CommandType, req.Args)
}
```

### 2. Graceful Shutdown

```go
// main.go
func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    // Start gRPC server
    go func() {
        if err := grpcServer.Serve(lis); err != nil {
            log.Fatal().Err(err).Msg("gRPC server failed")
        }
    }()
    
    // Wait for interrupt
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
    <-sigCh
    
    log.Info().Msg("Shutting down gracefully...")
    
    // Graceful stop with timeout
    stopped := make(chan struct{})
    go func() {
        grpcServer.GracefulStop()
        close(stopped)
    }()
    
    select {
    case <-stopped:
        log.Info().Msg("Server stopped gracefully")
    case <-time.After(30 * time.Second):
        log.Warn().Msg("Forcing shutdown after timeout")
        grpcServer.Stop()
    }
}
```

### 3. Command Execution Security

```go
func (m *Manager) RunCommand(ctx context.Context, cmdType string, args []string) error {
    // Whitelist check
    if !isAllowedCommand(cmdType) {
        return fmt.Errorf("command not allowed: %s", cmdType)
    }
    
    // Argument validation
    for _, arg := range args {
        if !isValidArgument(arg) {
            return fmt.Errorf("invalid argument: %s", arg)
        }
    }
    
    // Set timeout from context
    ctx, cancel := context.WithTimeout(ctx, m.config.MaxCommandTimeout)
    defer cancel()
    
    // Execute with sudo if needed
    var cmd *exec.Cmd
    if m.config.SudoUser != "" {
        cmd = exec.CommandContext(ctx, "sudo", "-u", m.config.SudoUser, cmdType)
        cmd.Args = append(cmd.Args, args...)
    } else {
        cmd = exec.CommandContext(ctx, cmdType, args...)
    }
    
    // Capture output
    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr
    
    // Run and log
    err := cmd.Run()
    m.logger.Info().
        Str("command", cmdType).
        Strs("args", args).
        Int("exit_code", cmd.ProcessState.ExitCode()).
        Err(err).
        Msg("Command executed")
    
    return err
}
```

### 4. Exponential Backoff

```go
func (c *Client) connectWithBackoff(ctx context.Context) error {
    delay := c.config.InitialDelay
    attempts := 0
    
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
        }
        
        conn, err := c.dial()
        if err == nil {
            c.conn = conn
            c.logger.Info().Msg("Connected to control server")
            return nil
        }
        
        attempts++
        if attempts >= c.config.MaxAttempts {
            return fmt.Errorf("max reconnect attempts exceeded")
        }
        
        c.logger.Warn().
            Err(err).
            Int("attempt", attempts).
            Dur("delay", delay).
            Msg("Connection failed, retrying...")
        
        time.Sleep(delay)
        delay = time.Duration(float64(delay) * c.config.Multiplier)
        if delay > c.config.MaxDelay {
            delay = c.config.MaxDelay
        }
    }
}
```

## Тестирование

### Unit Tests

```go
// internal/ocserv/manager_test.go
func TestManager_GetStatus(t *testing.T) {
    tests := []struct {
        name    string
        mockOut string
        mockErr error
        want    *OcservStatus
        wantErr bool
    }{
        {
            name: "success",
            mockOut: "active (running)",
            mockErr: nil,
            want: &OcservStatus{IsRunning: true},
            wantErr: false,
        },
        {
            name: "not running",
            mockOut: "inactive (dead)",
            want: &OcservStatus{IsRunning: false},
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Setup mock
            // Test logic
            // Assertions
        })
    }
}
```

### Integration Tests

```go
// integration_test.go
func TestGRPC_HealthCheck(t *testing.T) {
    // Start test server
    lis := bufconn.Listen(1024 * 1024)
    s := grpc.NewServer()
    pb.RegisterAgentServiceServer(s, &server{})
    
    go s.Serve(lis)
    defer s.Stop()
    
    // Create client
    conn, _ := grpc.DialContext(ctx, "",
        grpc.WithContextDialer(bufDialer(lis)),
        grpc.WithTransportCredentials(insecure.NewCredentials()))
    defer conn.Close()
    
    client := pb.NewAgentServiceClient(conn)
    
    // Test
    resp, err := client.HealthCheck(ctx, &pb.HealthCheckRequest{Tier: 1})
    assert.NoError(t, err)
    assert.True(t, resp.Healthy)
}
```

## Makefile

```makefile
.PHONY: all build test proto clean install

VERSION ?= $(shell git describe --tags --always --dirty)
LDFLAGS := -X main.version=$(VERSION) -s -w

all: proto test build

proto:
	@echo "Generating protobuf code..."
	protoc --go_out=. --go-grpc_out=. \
		--go_opt=paths=source_relative \
		--go-grpc_opt=paths=source_relative \
		pkg/proto/agent/v1/agent.proto

build:
	@echo "Building agent..."
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/ocserv-agent ./cmd/agent

test:
	@echo "Running tests..."
	go test -v -race -cover ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

lint:
	golangci-lint run ./...

install: build
	sudo cp bin/ocserv-agent /usr/local/bin/
	sudo mkdir -p /etc/ocserv-agent/certs
	sudo cp config.yaml.example /etc/ocserv-agent/config.yaml
	sudo cp deploy/systemd/ocserv-agent.service /etc/systemd/system/
	sudo systemctl daemon-reload

clean:
	rm -rf bin/ coverage.out
```

## Dockerfile

```dockerfile
# Build stage
FROM golang:1.25-trixie AS builder

WORKDIR /build

# Dependencies
COPY go.mod go.sum ./
RUN go mod download

# Build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.version=$(git describe --tags --always)" \
    -o ocserv-agent ./cmd/agent

# Runtime stage
FROM debian:trixie-slim

# Install dependencies
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        ca-certificates \
        ocserv \
        sudo && \
    rm -rf /var/lib/apt/lists/*

# Create user
RUN useradd -r -s /bin/false ocserv-agent

# Copy binary
COPY --from=builder /build/ocserv-agent /usr/local/bin/

# Config directory
RUN mkdir -p /etc/ocserv-agent/certs
VOLUME /etc/ocserv-agent

USER ocserv-agent
EXPOSE 9090

ENTRYPOINT ["/usr/local/bin/ocserv-agent"]
CMD ["--config", "/etc/ocserv-agent/config.yaml"]
```

## Приоритеты разработки

### Phase 1: Core (Week 1)
1. ✅ Базовая структура проекта
2. ✅ Proto definitions
3. ✅ Config loading
4. ✅ gRPC server setup с mTLS
5. ✅ Basic health check

### Phase 2: ocserv Integration (Week 2)
1. ✅ systemctl wrapper
2. ✅ occtl command execution
3. ✅ Config file reading
4. ✅ Command validation и security

### Phase 3: Streaming (Week 3)
1. ✅ Bidirectional streaming
2. ✅ Heartbeat implementation
3. ✅ Log streaming
4. ✅ Reconnection logic

### Phase 4: Production Ready (Week 4)
1. ✅ OpenTelemetry integration
2. ✅ Error handling и retry logic
3. ✅ Unit tests (>80% coverage)
4. ✅ Integration tests
5. ✅ Documentation

## Критерии готовности

- [ ] Все gRPC методы реализованы
- [ ] mTLS работает корректно
- [ ] Heartbeat стабильно отправляется каждые 15 секунд
- [ ] Reconnection с exponential backoff работает
- [ ] Все команды occtl выполняются
- [ ] Config updates применяются с backup
- [ ] Health checks всех 3 уровней работают
- [ ] OpenTelemetry traces отправляются
- [ ] Unit tests покрывают >80% кода
- [ ] Integration tests проходят
- [ ] Systemd service корректно запускается
- [ ] Graceful shutdown работает
- [ ] Логи структурированные (JSON)
- [ ] Нет race conditions (go test -race)
- [ ] golangci-lint проходит без ошибок
- [ ] README.md с примерами использования
- [ ] Dockerfile работает

## Стиль кода

- Используй `gofmt` и `goimports`
- Следуй [Effective Go](https://go.dev/doc/effective_go)
- Используй `context.Context` везде
- Structured logging с zerolog
- Errors wrapping с `fmt.Errorf("%w", err)`
- Комментарии для всех exported функций
- Table-driven tests

## Git Workflow и Development Practices

### Политика коммитов

**КРИТИЧЕСКИ ВАЖНО:** Делай коммит после **КАЖДОГО** логического изменения, даже если это маленькое изменение.

#### Правила коммитов:

1. **Один коммит = одно изменение**
   - Добавил новую функцию → commit
   - Исправил баг → commit
   - Обновил документацию → commit
   - Добавил тест → commit

2. **Формат commit message (Conventional Commits):**
   ```
   <type>(<scope>): <subject>

   <body>

   <footer>
   ```

   **Types:**
   - `feat`: Новая функциональность
   - `fix`: Исправление бага
   - `docs`: Только документация
   - `style`: Форматирование (без изменения логики)
   - `refactor`: Рефакторинг кода
   - `test`: Добавление тестов
   - `chore`: Обновление зависимостей, build tasks

   **Примеры:**
   ```bash
   feat(grpc): implement HealthCheck endpoint
   fix(ocserv): handle missing config file gracefully
   docs(readme): add installation instructions
   test(manager): add unit tests for RunCommand
   refactor(config): extract validation logic
   chore(deps): update go.mod dependencies
   ```

3. **Commit body (опционально, но рекомендуется):**
   ```
   feat(grpc): implement bidirectional streaming

   Add AgentStream RPC method for heartbeat and command
   execution. Supports graceful reconnection with exponential
   backoff.

   Closes #12
   ```

4. **Breaking changes:**
   ```
   feat(proto)!: change heartbeat interval field type

   BREAKING CHANGE: heartbeat_interval changed from int32 to
   google.protobuf.Duration for better precision
   ```

#### Workflow

```bash
# 1. Сделал изменение
vim internal/grpc/server.go

# 2. Проверь что работает
go test ./internal/grpc/
go build ./cmd/agent

# 3. Commit сразу
git add internal/grpc/server.go
git commit -m "feat(grpc): add mTLS configuration"

# 4. Следующее изменение
vim internal/grpc/interceptors.go
git add internal/grpc/interceptors.go
git commit -m "feat(grpc): add logging interceptor"

# НЕ делай:
# git add .
# git commit -m "add grpc stuff"  ❌
```

### Release Notes

Веди детальные release notes для каждой версии в директории `docs/releases/`.

#### Структура:

```
docs/
└── releases/
    ├── v0.1.0.md
    ├── v0.2.0.md
    ├── v1.0.0.md
    └── TEMPLATE.md
```

#### Шаблон Release Notes

**`docs/releases/TEMPLATE.md`:**
```markdown
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
make build
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
```

#### Процесс создания Release Notes

```bash
# 1. Перед началом работы над новой версией
cp docs/releases/TEMPLATE.md docs/releases/v0.2.0.md

# 2. Во время разработки добавляй записи
# После каждого важного коммита обновляй release notes

# 3. Перед релизом
# Заполни все секции
# Проверь статистику: git diff v0.1.0...HEAD --stat
# Добавь contributors: git log v0.1.0..HEAD --format="%aN" | sort -u

# 4. Commit release notes
git add docs/releases/v0.2.0.md
git commit -m "docs(release): add v0.2.0 release notes"
```

### TODO Management

Веди активный TODO list в директории `docs/todo/`.

#### Структура:

```
docs/
└── todo/
    ├── CURRENT.md          # Текущие задачи
    ├── BACKLOG.md          # Будущие задачи
    ├── DONE.md             # Завершённые задачи
    └── archive/
        ├── 2025-01.md      # Архив по месяцам
        └── 2025-02.md
```

#### Формат TODO

**`docs/todo/CURRENT.md`:**
```markdown
# Current TODO - ocserv-agent

**Last Updated:** 2025-01-15 14:30 UTC

## 🔴 Critical (Must do now)

- [ ] **[BUG]** Fix memory leak in streaming (#45)
  - Priority: P0
  - Assigned: -
  - Deadline: 2025-01-16
  - Blockers: None
  - Notes: Occurs after 24h of continuous streaming

- [ ] **[SECURITY]** Implement rate limiting for gRPC (#47)
  - Priority: P0
  - Assigned: -
  - Deadline: 2025-01-17
  - Blockers: None

## 🟡 High Priority (This week)

- [ ] **[FEATURE]** Add config hot-reload on SIGHUP (#23)
  - Priority: P1
  - Estimated: 4h
  - Dependencies: None
  - Branch: feature/config-reload

- [x] **[FEATURE]** Implement health check tier 2 (#34)
  - ✅ Completed: 2025-01-15
  - Commit: abc1234
  - PR: #35

## 🟢 Medium Priority (This month)

- [ ] **[IMPROVEMENT]** Optimize memory usage in log streaming
- [ ] **[DOCS]** Add troubleshooting guide
- [ ] **[TEST]** Add integration tests for mTLS

## 🔵 Low Priority (Backlog)

- [ ] **[FEATURE]** Support for multiple control servers
- [ ] **[DOCS]** Add architecture diagrams

## 📋 Code Review Needed

- [ ] PR #42 - Add prometheus metrics endpoint
- [ ] PR #43 - Refactor config loading

## 🐛 Known Issues

- Issue #50: Occasional connection drop after 1 hour (investigating)
- Issue #51: High CPU usage with 100+ concurrent connections

## 📊 Progress

- Features: 12/20 (60%)
- Tests: 85% coverage
- Documentation: 70% complete
```

#### TODO Update Process

**ПОСЛЕ КАЖДОГО КОММИТА:**

```bash
# 1. Сделал коммит
git commit -m "feat(grpc): add health check tier 2"

# 2. НЕМЕДЛЕННО обновить TODO
vim docs/todo/CURRENT.md

# Изменить:
# - [ ] **[FEATURE]** Implement health check tier 2 (#34)
# На:
# - [x] **[FEATURE]** Implement health check tier 2 (#34)
#   - ✅ Completed: 2025-01-15
#   - Commit: abc1234

# 3. Commit TODO update
git add docs/todo/CURRENT.md
git commit -m "docs(todo): mark health check tier 2 as done"

# 4. Проверить прогресс
./scripts/check-todo.sh
```

#### Автоматизация проверки TODO

**`scripts/check-todo.sh`:**
```bash
#!/bin/bash

TODO_FILE="docs/todo/CURRENT.md"

# Подсчёт задач
TOTAL=$(grep -c "^- \[" "$TODO_FILE")
DONE=$(grep -c "^- \[x\]" "$TODO_FILE")
TODO=$((TOTAL - DONE))

# Критические задачи
CRITICAL=$(grep -A 5 "🔴 Critical" "$TODO_FILE" | grep -c "^- \[ \]")

echo "📊 TODO Status"
echo "━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Total Tasks:     $TOTAL"
echo "Completed:       $DONE"
echo "Remaining:       $TODO"
echo "Critical:        $CRITICAL"
echo ""

if [ $CRITICAL -gt 0 ]; then
    echo "⚠️  WARNING: $CRITICAL critical tasks remaining!"
    grep -A 5 "🔴 Critical" "$TODO_FILE" | grep "^- \[ \]"
    exit 1
fi

if [ $TODO -eq 0 ]; then
    echo "✅ All tasks completed!"
fi

exit 0
```

### Code Review Process

После каждого существенного изменения проводи **self-review**:

#### Self-Review Checklist

**`docs/SELF_REVIEW_CHECKLIST.md`:**
```markdown
# Self-Review Checklist

Перед тем как закрыть задачу, проверь:

## ✅ Code Quality

- [ ] Код следует Go best practices
- [ ] Нет commented code
- [ ] Нет TODO комментариев (или они задокументированы в docs/todo/)
- [ ] Все exported функции имеют godoc комментарии
- [ ] Нет magic numbers (используются константы)
- [ ] Error handling корректен (wrapped errors)
- [ ] Context propagation правильный

## ✅ Testing

- [ ] Unit tests добавлены/обновлены
- [ ] Tests проходят: `go test ./...`
- [ ] Race detector: `go test -race ./...`
- [ ] Coverage не упал: `go test -cover ./...`
- [ ] Integration tests обновлены (если нужно)

## ✅ Security

- [ ] Нет hardcoded secrets
- [ ] Input validation добавлена
- [ ] SQL injection защита (если применимо)
- [ ] Command injection защита
- [ ] Sensitive data логируется правильно (masked)

## ✅ Performance

- [ ] Нет memory leaks
- [ ] Goroutines завершаются правильно
- [ ] Context cancellation обрабатывается
- [ ] Resources закрываются (defer)

## ✅ Documentation

- [ ] README обновлён (если нужно)
- [ ] API documentation обновлена
- [ ] CHANGELOG.md обновлён
- [ ] Release notes обновлены
- [ ] TODO list обновлён

## ✅ Build & Deploy

- [ ] Код компилируется: `make build`
- [ ] Linter проходит: `make lint`
- [ ] Dockerfile работает
- [ ] systemd service конфиг обновлён (если нужно)

## ✅ Git

- [ ] Commit message следует Conventional Commits
- [ ] Branch от latest main/develop
- [ ] No merge conflicts
- [ ] Squash если много мелких коммитов
```

**Использование:**

```bash
# После завершения задачи
./scripts/self-review.sh

# Скрипт проверит все пункты автоматически
```

### Политика версионирования (Semantic Versioning)

Следуем **Semantic Versioning 2.0.0** (semver.org):

```
MAJOR.MINOR.PATCH[-PRERELEASE][+BUILD]

Пример: 1.2.3-beta.1+20250115
```

#### Правила версионирования

**MAJOR (X.0.0)** - Breaking changes
- Несовместимые изменения в API
- Изменения в proto definitions (breaking)
- Изменения в config формате (breaking)
- Удаление функциональности

**Примеры MAJOR:**
```
v1.0.0 → v2.0.0
- Изменили gRPC API (удалили методы)
- Изменили формат config.yaml (несовместимо)
- Требуется Go 1.26+ вместо 1.25+
```

**MINOR (x.Y.0)** - Новая функциональность (backward compatible)
- Добавление новых gRPC методов
- Новые опции в конфиге (с defaults)
- Новые features

**Примеры MINOR:**
```
v1.0.0 → v1.1.0
- Добавили новый RPC метод StreamMetrics
- Добавили поддержку TOTP authentication
- Добавили новый config параметр (опциональный)
```

**PATCH (x.y.Z)** - Bug fixes (backward compatible)
- Исправления багов
- Security patches
- Performance improvements (без изменения API)
- Documentation updates

**Примеры PATCH:**
```
v1.1.0 → v1.1.1
- Исправили memory leak
- Исправили race condition
- Обновили зависимости (security)
```

#### Pre-release Versions

**Форматы:**
- `v1.0.0-alpha.1` - Ранняя альфа (нестабильно)
- `v1.0.0-beta.1` - Бета (feature complete, но могут быть баги)
- `v1.0.0-rc.1` - Release candidate (готово к релизу)

**Правила:**
```
v0.1.0-alpha.1  → v0.1.0-alpha.2  (фиксы в альфе)
v0.1.0-alpha.2  → v0.1.0-beta.1   (альфа → бета)
v0.1.0-beta.3   → v0.1.0-rc.1     (бета → RC)
v0.1.0-rc.2     → v0.1.0          (RC → stable)
```

#### Version 0.x.x (Development Phase)

В фазе разработки (v0.x.x):
- Breaking changes могут быть в MINOR версиях
- API не гарантирует стабильность
- Используется до достижения production-ready

```
v0.1.0  - Initial implementation
v0.2.0  - Added mTLS (breaking: changed config)
v0.3.0  - Added streaming (compatible)
v0.9.0  - Feature complete (RC candidate)
v1.0.0  - Production release (стабильное API)
```

#### Version Lifecycle

```
Development:  v0.1.0 → v0.9.0
   ↓
Stable:       v1.0.0 → v1.9.0
   ↓
Next Gen:     v2.0.0 → v2.9.0
```

**Maintenance:**
- Latest: v2.3.0 (активная разработка)
- Previous: v1.9.5 (security fixes только)
- Legacy: v0.9.8 (не поддерживается)

#### Процесс создания релиза

**1. Pre-release preparation:**
```bash
# 1. Update version in code
vim cmd/agent/main.go
# const version = "1.2.0"

# 2. Update CHANGELOG
vim CHANGELOG.md

# 3. Update release notes
vim docs/releases/v1.2.0.md

# 4. Commit
git add .
git commit -m "chore(release): prepare v1.2.0"

# 5. Tag
git tag -a v1.2.0 -m "Release v1.2.0"
```

**2. Build & Test:**
```bash
# Build all targets
make build-all

# Run full test suite
make test-all

# Security scan
govulncheck ./...

# Lint
golangci-lint run
```

**3. Create GitHub Release:**
```bash
# Push tag
git push origin v1.2.0

# Create release with binaries
gh release create v1.2.0 \
  --title "v1.2.0 - Feature Name" \
  --notes-file docs/releases/v1.2.0.md \
  bin/ocserv-agent-linux-amd64 \
  bin/ocserv-agent-linux-arm64
```

**4. Post-release:**
```bash
# Update main branch
git checkout main
git merge develop

# Start next version
git checkout develop
vim cmd/agent/main.go  # version = "1.3.0-dev"
git commit -m "chore: start v1.3.0 development"
```

#### Version Tagging Strategy

```bash
# Lightweight tag (не рекомендуется для релизов)
git tag v1.0.0

# Annotated tag (правильный способ)
git tag -a v1.0.0 -m "Release v1.0.0: Initial production release"

# Signed tag (для production релизов)
git tag -s v1.0.0 -m "Release v1.0.0"

# Push tags
git push origin v1.0.0
# или все теги:
git push origin --tags
```

#### go.mod Versioning

Версия в `go.mod` должна соответствовать git tag:

```go
module github.com/dantte-lp/ocserv-agent

go 1.25

// v1.0.0 и выше используются как:
// require github.com/dantte-lp/ocserv-agent v1.2.0

// v2+ требует /v2 suffix:
// module github.com/dantte-lp/ocserv-agent/v2
```

### Итоговый Workflow

```bash
# ═══════════════════════════════════════════════
# ЕЖЕДНЕВНАЯ РАБОТА
# ═══════════════════════════════════════════════

# 1. Утро: Проверить TODO
cat docs/todo/CURRENT.md
./scripts/check-todo.sh

# 2. Взять задачу
# - [ ] Implement feature X

# 3. Создать branch
git checkout -b feature/feature-x

# 4. Реализовать
vim internal/feature/feature.go

# 5. Тест
go test ./internal/feature/

# 6. COMMIT
git add internal/feature/feature.go
git commit -m "feat(feature): add feature X implementation"

# 7. Обновить TODO
vim docs/todo/CURRENT.md
# - [x] Implement feature X
git add docs/todo/CURRENT.md
git commit -m "docs(todo): mark feature X as done"

# 8. Self-review
./scripts/self-review.sh

# 9. Push
git push origin feature/feature-x

# 10. Create PR
gh pr create --base develop --fill

# ═══════════════════════════════════════════════
# ПЕРЕД РЕЛИЗОМ
# ═══════════════════════════════════════════════

# 1. Проверить TODO
./scripts/check-todo.sh
# Все критичные задачи должны быть закрыты

# 2. Обновить версию
vim cmd/agent/main.go

# 3. Release notes
cp docs/releases/TEMPLATE.md docs/releases/v1.2.0.md
vim docs/releases/v1.2.0.md

# 4. CHANGELOG
vim CHANGELOG.md

# 5. Commit
git add .
git commit -m "chore(release): prepare v1.2.0"

# 6. Tag
git tag -a v1.2.0 -m "Release v1.2.0"

# 7. Build & Test
make build-all
make test-all

# 8. Push
git push origin main
git push origin v1.2.0

# 9. GitHub Release
gh release create v1.2.0 \
  --notes-file docs/releases/v1.2.0.md \
  bin/*

# 10. Archive TODO
mv docs/todo/CURRENT.md docs/todo/archive/2025-01.md
cp docs/todo/TEMPLATE.md docs/todo/CURRENT.md
```

### Podman-Compose для сборки и тестирования

**ОБЯЗАТЕЛЬНО:** Все сборки и тесты должны выполняться в **podman-compose**, а не на хост-системе.

#### Зачем?

1. **Консистентность окружения** - одинаковые зависимости у всех разработчиков
2. **Изоляция** - не засоряем хост-систему
3. **Воспроизводимость** - гарантия что работает везде
4. **CI/CD готовность** - локально = как в production

#### Структура compose файлов

```
deploy/
├── compose/
│   ├── docker-compose.dev.yml      # Development окружение
│   ├── docker-compose.test.yml     # Testing окружение
│   ├── docker-compose.build.yml    # Build окружение
│   └── .env.example                # Environment variables
└── scripts/
    └── generate-compose.sh         # Генерация compose файлов
```

#### Development Compose

**`deploy/compose/docker-compose.dev.yml`:**
```yaml
version: '3.8'

services:
  # ═══════════════════════════════════════════════
  # Development Agent (Hot Reload)
  # ═══════════════════════════════════════════════
  agent-dev:
    image: golang:1.25-trixie
    container_name: ocserv-agent-dev
    working_dir: /workspace
    volumes:
      - ../../:/workspace:z
      - go-cache:/go/pkg
      - go-build-cache:/root/.cache/go-build
    environment:
      - CGO_ENABLED=0
      - GOOS=linux
      - GOARCH=amd64
    command: |
      sh -c '
        echo "📦 Installing Air for hot reload..."
        go install github.com/air-verse/air@latest
        echo "🔄 Starting development server with hot reload..."
        air -c .air.toml
      '
    ports:
      - "9090:9090"
      - "2345:2345"  # Delve debugger
    networks:
      - agent-net
    restart: unless-stopped

  # ═══════════════════════════════════════════════
  # Mock Control Server (для тестирования агента)
  # ═══════════════════════════════════════════════
  mock-control-server:
    image: golang:1.25-trixie
    container_name: mock-control-server
    working_dir: /workspace
    volumes:
      - ../../:/workspace:z
    command: |
      sh -c '
        cd test/mock-server
        go run main.go
      '
    ports:
      - "9091:9091"
    networks:
      - agent-net
    depends_on:
      - agent-dev

  # ═══════════════════════════════════════════════
  # Mock ocserv (для тестирования без реального VPN)
  # ═══════════════════════════════════════════════
  mock-ocserv:
    image: debian:trixie-slim
    container_name: mock-ocserv
    volumes:
      - ../../test/mock-ocserv:/opt/mock:z
    command: |
      sh -c '
        apt-get update && apt-get install -y iproute2 procps
        /opt/mock/run-mock.sh
      '
    networks:
      - agent-net
    cap_add:
      - NET_ADMIN

  # ═══════════════════════════════════════════════
  # Redis (для кеша, если нужен)
  # ═══════════════════════════════════════════════
  redis:
    image: redis:7-alpine
    container_name: agent-redis
    ports:
      - "6379:6379"
    networks:
      - agent-net
    volumes:
      - redis-data:/data

networks:
  agent-net:
    driver: bridge

volumes:
  go-cache:
  go-build-cache:
  redis-data:
```

#### Test Compose

**`deploy/compose/docker-compose.test.yml`:**
```yaml
version: '3.8'

services:
  # ═══════════════════════════════════════════════
  # Test Runner
  # ═══════════════════════════════════════════════
  test:
    image: golang:1.25-trixie
    container_name: ocserv-agent-test
    working_dir: /workspace
    volumes:
      - ../../:/workspace:z
      - go-test-cache:/go/pkg
    environment:
      - CGO_ENABLED=0
      - GOCOVERDIR=/workspace/coverage
    command: |
      sh -c '
        echo "🧪 Running tests..."
        
        # Unit tests
        echo "▶ Unit tests"
        go test -v -race -coverprofile=coverage.out ./...
        
        # Coverage report
        echo "▶ Coverage report"
        go tool cover -func=coverage.out
        go tool cover -html=coverage.out -o coverage.html
        
        # Integration tests
        echo "▶ Integration tests"
        go test -v -tags=integration ./test/integration/...
        
        echo "✅ All tests passed!"
      '
    networks:
      - test-net
    depends_on:
      - mock-control-server
      - mock-ocserv

  # ═══════════════════════════════════════════════
  # Lint & Static Analysis
  # ═══════════════════════════════════════════════
  lint:
    image: golangci/golangci-lint:v1.62-alpine
    container_name: ocserv-agent-lint
    working_dir: /workspace
    volumes:
      - ../../:/workspace:z
      - golangci-cache:/root/.cache
    command: |
      sh -c '
        echo "🔍 Running linters..."
        golangci-lint run --timeout 5m ./...
        echo "✅ Linting passed!"
      '

  # ═══════════════════════════════════════════════
  # Security Scan
  # ═══════════════════════════════════════════════
  security:
    image: golang:1.25-trixie
    container_name: ocserv-agent-security
    working_dir: /workspace
    volumes:
      - ../../:/workspace:z
    command: |
      sh -c '
        echo "🔒 Running security scans..."
        
        # govulncheck
        go install golang.org/x/vuln/cmd/govulncheck@latest
        govulncheck ./...
        
        # gosec
        go install github.com/securego/gosec/v2/cmd/gosec@latest
        gosec -fmt=json -out=security-report.json ./...
        
        echo "✅ Security scan completed!"
      '

  # Mock services для тестов
  mock-control-server:
    image: golang:1.25-trixie
    working_dir: /workspace
    volumes:
      - ../../:/workspace:z
    command: sh -c 'cd test/mock-server && go run main.go'
    networks:
      - test-net

  mock-ocserv:
    image: debian:trixie-slim
    volumes:
      - ../../test/mock-ocserv:/opt/mock:z
    command: sh -c 'apt-get update && apt-get install -y iproute2 && /opt/mock/run-mock.sh'
    networks:
      - test-net
    cap_add:
      - NET_ADMIN

networks:
  test-net:
    driver: bridge

volumes:
  go-test-cache:
  golangci-cache:
```

#### Build Compose

**`deploy/compose/docker-compose.build.yml`:**
```yaml
version: '3.8'

services:
  # ═══════════════════════════════════════════════
  # Multi-arch Build
  # ═══════════════════════════════════════════════
  build-linux-amd64:
    image: golang:1.25-trixie
    container_name: build-amd64
    working_dir: /workspace
    volumes:
      - ../../:/workspace:z
      - go-build-cache:/root/.cache/go-build
    environment:
      - CGO_ENABLED=0
      - GOOS=linux
      - GOARCH=amd64
      - VERSION=${VERSION:-dev}
    command: |
      sh -c '
        echo "🔨 Building for linux/amd64..."
        go build -ldflags="-s -w -X main.version=${VERSION}" \
          -o bin/ocserv-agent-linux-amd64 \
          ./cmd/agent
        echo "✅ Built: bin/ocserv-agent-linux-amd64"
      '

  build-linux-arm64:
    image: golang:1.25-trixie
    container_name: build-arm64
    working_dir: /workspace
    volumes:
      - ../../:/workspace:z
      - go-build-cache:/root/.cache/go-build
    environment:
      - CGO_ENABLED=0
      - GOOS=linux
      - GOARCH=arm64
      - VERSION=${VERSION:-dev}
    command: |
      sh -c '
        echo "🔨 Building for linux/arm64..."
        go build -ldflags="-s -w -X main.version=${VERSION}" \
          -o bin/ocserv-agent-linux-arm64 \
          ./cmd/agent
        echo "✅ Built: bin/ocserv-agent-linux-arm64"
      '

  # ═══════════════════════════════════════════════
  # Build Production Docker Image
  # ═══════════════════════════════════════════════
  build-image:
    image: quay.io/podman/stable
    container_name: build-image
    privileged: true
    volumes:
      - ../../:/workspace:z
      - /var/run/docker.sock:/var/run/docker.sock
    working_dir: /workspace
    environment:
      - VERSION=${VERSION:-latest}
    command: |
      sh -c '
        echo "🐳 Building Docker image..."
        podman build \
          --tag ocserv-agent:${VERSION} \
          --tag ocserv-agent:latest \
          -f Dockerfile .
        
        echo "✅ Image built: ocserv-agent:${VERSION}"
        podman images | grep ocserv-agent
      '

volumes:
  go-build-cache:
```

#### Скрипт генерации compose файлов

**`deploy/scripts/generate-compose.sh`:**
```bash
#!/bin/bash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
COMPOSE_DIR="$PROJECT_ROOT/deploy/compose"

echo "🔧 Generating Podman Compose configurations..."

# Цвета для вывода
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# ═══════════════════════════════════════════════
# Функция: создать .env файл
# ═══════════════════════════════════════════════
create_env_file() {
    local env_file="$COMPOSE_DIR/.env"
    
    if [ -f "$env_file" ]; then
        echo -e "${YELLOW}⚠️  .env already exists, skipping${NC}"
        return
    fi
    
    cat > "$env_file" << 'EOF'
# Podman Compose Environment Variables

# Version
VERSION=dev

# Agent Configuration
AGENT_LOG_LEVEL=debug
AGENT_HEARTBEAT_INTERVAL=15s

# gRPC
GRPC_PORT=9090

# Control Server (for testing)
CONTROL_SERVER_HOST=mock-control-server
CONTROL_SERVER_PORT=9091

# Redis
REDIS_HOST=redis
REDIS_PORT=6379

# Paths
WORKSPACE_DIR=../../
CONFIG_DIR=/etc/ocserv-agent
CERTS_DIR=/etc/ocserv-agent/certs

# Build settings
CGO_ENABLED=0
GOOS=linux
GOARCH=amd64
EOF

    echo -e "${GREEN}✅ Created $env_file${NC}"
}

# ═══════════════════════════════════════════════
# Функция: создать .air.toml для hot reload
# ═══════════════════════════════════════════════
create_air_config() {
    local air_config="$PROJECT_ROOT/.air.toml"
    
    if [ -f "$air_config" ]; then
        echo -e "${YELLOW}⚠️  .air.toml already exists, skipping${NC}"
        return
    fi
    
    cat > "$air_config" << 'EOF'
# Air configuration for hot reload

root = "."
testdata_dir = "testdata"
tmp_dir = "tmp"

[build]
  args_bin = ["--config", "config.yaml.example"]
  bin = "./tmp/ocserv-agent"
  cmd = "go build -o ./tmp/ocserv-agent ./cmd/agent"
  delay = 1000
  exclude_dir = ["tmp", "vendor", "test", "docs"]
  exclude_file = []
  exclude_regex = ["_test.go"]
  exclude_unchanged = false
  follow_symlink = false
  full_bin = ""
  include_dir = []
  include_ext = ["go", "tpl", "tmpl", "html", "yaml", "yml"]
  include_file = []
  kill_delay = "0s"
  log = "build-errors.log"
  poll = false
  poll_interval = 0
  rerun = false
  rerun_delay = 500
  send_interrupt = false
  stop_on_error = false

[color]
  app = ""
  build = "yellow"
  main = "magenta"
  runner = "green"
  watcher = "cyan"

[log]
  main_only = false
  time = false

[misc]
  clean_on_exit = false

[screen]
  clear_on_rebuild = false
  keep_scroll = true
EOF

    echo -e "${GREEN}✅ Created $air_config${NC}"
}

# ═══════════════════════════════════════════════
# Функция: создать mock-server для тестов
# ═══════════════════════════════════════════════
create_mock_server() {
    local mock_dir="$PROJECT_ROOT/test/mock-server"
    mkdir -p "$mock_dir"
    
    cat > "$mock_dir/main.go" << 'EOF'
package main

import (
    "log"
    "net"
    
    "google.golang.org/grpc"
)

type mockServer struct {
    // TODO: implement proto interface
}

func main() {
    lis, err := net.Listen("tcp", ":9091")
    if err != nil {
        log.Fatalf("failed to listen: %v", err)
    }
    
    s := grpc.NewServer()
    // TODO: register service
    
    log.Println("Mock control server listening on :9091")
    if err := s.Serve(lis); err != nil {
        log.Fatalf("failed to serve: %v", err)
    }
}
EOF

    echo -e "${GREEN}✅ Created mock server${NC}"
}

# ═══════════════════════════════════════════════
# Функция: создать mock ocserv
# ═══════════════════════════════════════════════
create_mock_ocserv() {
    local mock_dir="$PROJECT_ROOT/test/mock-ocserv"
    mkdir -p "$mock_dir"
    
    cat > "$mock_dir/run-mock.sh" << 'EOF'
#!/bin/bash
# Mock ocserv для тестирования

echo "🔧 Starting mock ocserv..."

# Создать fake socket
mkdir -p /var/run
touch /var/run/occtl.socket

# Имитация occtl команд
while true; do
    if [ -p /tmp/occtl-pipe ]; then
        read cmd < /tmp/occtl-pipe
        case $cmd in
            "show users")
                echo '{"users": []}'
                ;;
            "show status")
                echo '{"status": "running", "uptime": 12345}'
                ;;
            *)
                echo '{"error": "unknown command"}'
                ;;
        esac
    fi
    sleep 1
done
EOF

    chmod +x "$mock_dir/run-mock.sh"
    echo -e "${GREEN}✅ Created mock ocserv${NC}"
}

# ═══════════════════════════════════════════════
# Главная логика
# ═══════════════════════════════════════════════

echo ""
echo "📁 Project root: $PROJECT_ROOT"
echo "📁 Compose dir: $COMPOSE_DIR"
echo ""

# Создать директории
mkdir -p "$COMPOSE_DIR"
mkdir -p "$PROJECT_ROOT/test/mock-server"
mkdir -p "$PROJECT_ROOT/test/mock-ocserv"
mkdir -p "$PROJECT_ROOT/tmp"

# Создать файлы
create_env_file
create_air_config
create_mock_server
create_mock_ocserv

# Создать алиасы в Makefile
cat >> "$PROJECT_ROOT/Makefile" << 'EOF'

# ═══════════════════════════════════════════════
# Podman Compose targets
# ═══════════════════════════════════════════════

.PHONY: compose-dev compose-test compose-build compose-down compose-logs

compose-dev:
	@echo "🚀 Starting development environment..."
	cd deploy/compose && podman-compose -f docker-compose.dev.yml up

compose-test:
	@echo "🧪 Running tests in containers..."
	cd deploy/compose && podman-compose -f docker-compose.test.yml up --abort-on-container-exit
	cd deploy/compose && podman-compose -f docker-compose.test.yml down

compose-build:
	@echo "🔨 Building binaries in containers..."
	cd deploy/compose && VERSION=${VERSION:-dev} podman-compose -f docker-compose.build.yml up
	cd deploy/compose && podman-compose -f docker-compose.build.yml down

compose-down:
	@echo "🛑 Stopping all compose services..."
	cd deploy/compose && podman-compose -f docker-compose.dev.yml down || true
	cd deploy/compose && podman-compose -f docker-compose.test.yml down || true
	cd deploy/compose && podman-compose -f docker-compose.build.yml down || true

compose-logs:
	cd deploy/compose && podman-compose -f docker-compose.dev.yml logs -f

compose-clean:
	@echo "🧹 Cleaning compose volumes..."
	podman volume rm ocserv-agent_go-cache ocserv-agent_go-build-cache || true
EOF

echo ""
echo -e "${GREEN}✅ Podman Compose configuration generated!${NC}"
echo ""
echo "Usage:"
echo "  make compose-dev    - Start development with hot reload"
echo "  make compose-test   - Run all tests in containers"
echo "  make compose-build  - Build binaries (multi-arch)"
echo "  make compose-down   - Stop all services"
echo "  make compose-logs   - View logs"
echo ""
```

#### Интеграция в Workflow

**ОБНОВЛЁННЫЙ workflow после каждого коммита:**

```bash
# ═══════════════════════════════════════════════
# ШАГ 1: Реализовать изменение
# ═══════════════════════════════════════════════
vim internal/grpc/server.go

# ═══════════════════════════════════════════════
# ШАГ 2: Тест в контейнере (ОБЯЗАТЕЛЬНО!)
# ═══════════════════════════════════════════════
make compose-test

# Если тесты прошли:

# ═══════════════════════════════════════════════
# ШАГ 3: Сборка в контейнере (проверка компиляции)
# ═══════════════════════════════════════════════
make compose-build

# ═══════════════════════════════════════════════
# ШАГ 4: Commit ТОЛЬКО если всё собралось и протестировалось
# ═══════════════════════════════════════════════
git add internal/grpc/server.go
git commit -m "feat(grpc): add server implementation"

# ═══════════════════════════════════════════════
# ШАГ 5: Обновить TODO
# ═══════════════════════════════════════════════
vim docs/todo/CURRENT.md
git add docs/todo/CURRENT.md
git commit -m "docs(todo): mark server implementation as done"

# ═══════════════════════════════════════════════
# ШАГ 6: Self-review в контейнере
# ═══════════════════════════════════════════════
make compose-test  # Ещё раз для уверенности
./scripts/self-review.sh
```

#### Обновлённый Makefile

**Добавить в начало `Makefile`:**

```makefile
# ═══════════════════════════════════════════════
# PRIMARY TARGETS - ВСЕГДА используй Podman Compose!
# ═══════════════════════════════════════════════

.PHONY: dev test build

# Development
dev:
	@echo "⚠️  Use 'make compose-dev' instead!"
	@echo "Running outside containers is not recommended."
	@exit 1

# Testing
test:
	@echo "⚠️  Use 'make compose-test' instead!"
	@exit 1

# Building
build:
	@echo "⚠️  Use 'make compose-build' instead!"
	@exit 1

# Генерация compose конфигурации
.PHONY: setup-compose
setup-compose:
	@./deploy/scripts/generate-compose.sh

# ═══════════════════════════════════════════════
# EMERGENCY: Local build (только для отладки!)
# ═══════════════════════════════════════════════

.PHONY: local-build local-test

local-build:
	@echo "⚠️  WARNING: Building locally (not in container)"
	@echo "This should only be used for emergency debugging!"
	@sleep 2
	go build -o bin/ocserv-agent ./cmd/agent

local-test:
	@echo "⚠️  WARNING: Testing locally (not in container)"
	@sleep 2
	go test -v ./...
```

#### Self-Review Checklist Update

**Добавить в `docs/SELF_REVIEW_CHECKLIST.md`:**

```markdown
## ✅ Container Build & Test

- [ ] **Тесты прошли в контейнере:** `make compose-test`
- [ ] **Сборка прошла в контейнере:** `make compose-build`
- [ ] **НЕ использовал `go build` на хосте** (только через compose)
- [ ] **НЕ использовал `go test` на хосте** (только через compose)
- [ ] **Coverage не упал** (проверь в test output)
- [ ] **Все архитектуры собираются** (amd64 + arm64)
```

#### CI/CD Integration

**`.github/workflows/ci.yml`:**
```yaml
name: CI

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Install Podman Compose
        run: |
          pip3 install podman-compose
      
      - name: Run tests in containers
        run: make compose-test
      
      - name: Upload coverage
        uses: codecov/codecov-action@v4
        with:
          file: ./coverage.out

  build:
    runs-on: ubuntu-latest
    needs: test
    steps:
      - uses: actions/checkout@v4
      
      - name: Install Podman Compose
        run: pip3 install podman-compose
      
      - name: Build multi-arch
        run: VERSION=${{ github.ref_name }} make compose-build
      
      - name: Upload artifacts
        uses: actions/upload-artifact@v4
        with:
          name: binaries
          path: bin/*
```

### Summary: Critical Rules

1. ✅ **Commit после КАЖДОГО изменения**
2. ✅ **TODO обновляется СРАЗУ после коммита**
3. ✅ **Self-review перед закрытием задачи**
4. ✅ **Release notes пишутся ВО ВРЕМЯ разработки**
5. ✅ **Semantic Versioning строго**
6. ✅ **Conventional Commits всегда**
7. ✅ **Всегда собирай и тестируй в Podman Compose** (НЕ на хосте!)
8. ✅ **Генерируй compose файлы:** `make setup-compose`

## Начни с

1. Создай базовую структуру проекта
2. Реализуй proto definitions
3. Настрой gRPC server с mTLS
4. Реализуй простой HealthCheck endpoint
5. Добавь базовый heartbeat
6. Постепенно добавляй функционал по приоритетам выше

**ПОМНИ:** После каждого шага → commit → обновить TODO → self-review

Готов начать? Создай базовую структуру проекта, сделай первый коммит, и обнови TODO!