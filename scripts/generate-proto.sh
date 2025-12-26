#!/bin/bash
# generate-proto.sh - Генерация Go кода из proto файлов
set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

echo "=== Генерация Protocol Buffers кода ==="
echo "Рабочая директория: $PROJECT_ROOT"

# Проверка наличия protoc
if ! command -v protoc &> /dev/null; then
    echo "❌ ОШИБКА: protoc не установлен!"
    echo "Установите: apt-get install protobuf-compiler"
    exit 1
fi

# Проверка наличия Go плагинов
if ! command -v protoc-gen-go &> /dev/null; then
    echo "❌ ОШИБКА: protoc-gen-go не установлен!"
    echo "Установите: go install google.golang.org/protobuf/cmd/protoc-gen-go@latest"
    exit 1
fi

if ! command -v protoc-gen-go-grpc &> /dev/null; then
    echo "❌ ОШИБКА: protoc-gen-go-grpc не установлен!"
    echo "Установите: go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest"
    exit 1
fi

echo "✅ Все зависимости установлены"
echo ""

# Генерация agent proto
echo "📦 Генерация agent/v1/agent.proto..."
protoc -I. -I/usr/include --go_out=. --go-grpc_out=. \
    --go_opt=paths=source_relative \
    --go-grpc_opt=paths=source_relative \
    pkg/proto/agent/v1/agent.proto

echo "✅ agent.proto сгенерирован"
echo ""

# Генерация VPN proto файлов
echo "📦 Генерация vpn/v1/auth.proto..."
protoc -I. -I/usr/include --go_out=. --go-grpc_out=. \
    --go_opt=paths=source_relative \
    --go-grpc_opt=paths=source_relative \
    pkg/proto/vpn/v1/auth.proto

echo "✅ auth.proto сгенерирован"
echo ""

echo "📦 Генерация vpn/v1/events.proto..."
protoc -I. -I/usr/include --go_out=. --go-grpc_out=. \
    --go_opt=paths=source_relative \
    --go-grpc_opt=paths=source_relative \
    pkg/proto/vpn/v1/events.proto

echo "✅ events.proto сгенерирован"
echo ""

echo "📦 Генерация vpn/v1/config.proto..."
protoc -I. -I/usr/include --go_out=. --go-grpc_out=. \
    --go_opt=paths=source_relative \
    --go-grpc_opt=paths=source_relative \
    pkg/proto/vpn/v1/config.proto

echo "✅ config.proto сгенерирован"
echo ""

# Подсчет сгенерированных файлов
GENERATED_COUNT=$(find pkg/proto -name "*.pb.go" | wc -l)
echo "=== Генерация завершена ==="
echo "Сгенерировано файлов: $GENERATED_COUNT"
echo ""

# Список сгенерированных файлов
echo "Сгенерированные файлы:"
find pkg/proto -name "*.pb.go" -o -name "*_grpc.pb.go" | sort

exit 0
