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
# Функция: создать mock-server для тестов
# ═══════════════════════════════════════════════
create_mock_server() {
    local mock_dir="$PROJECT_ROOT/test/mock-server"
    mkdir -p "$mock_dir"

    if [ -f "$mock_dir/main.go" ]; then
        echo -e "${YELLOW}⚠️  mock-server/main.go already exists, skipping${NC}"
        return
    fi

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

    if [ -f "$mock_dir/run-mock.sh" ]; then
        echo -e "${YELLOW}⚠️  mock-ocserv/run-mock.sh already exists, skipping${NC}"
        return
    fi

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
create_mock_server
create_mock_ocserv

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
