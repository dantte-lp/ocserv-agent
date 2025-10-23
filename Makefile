.PHONY: all build test proto clean install help
.PHONY: compose-dev compose-test compose-build compose-down compose-logs compose-clean
.PHONY: local-build local-test local-proto setup-compose
.PHONY: security-check security-gosec security-govulncheck security-trivy

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -X main.version=$(VERSION) -s -w

# ═══════════════════════════════════════════════
# PRIMARY TARGETS - ВСЕГДА используй Podman Compose!
# ═══════════════════════════════════════════════

help:
	@echo "ocserv-agent Makefile"
	@echo ""
	@echo "📦 Recommended (Podman Compose):"
	@echo "  make compose-dev     - Start development with hot reload"
	@echo "  make compose-test    - Run all tests in containers"
	@echo "  make compose-build   - Build binaries (multi-arch)"
	@echo "  make compose-down    - Stop all services"
	@echo "  make compose-logs    - View logs"
	@echo "  make compose-clean   - Clean volumes"
	@echo ""
	@echo "🔒 Security Testing (Podman Compose):"
	@echo "  make security-check       - Run all security scans"
	@echo "  make security-gosec       - Run Gosec only"
	@echo "  make security-govulncheck - Run govulncheck only"
	@echo "  make security-trivy       - Run Trivy only"
	@echo ""
	@echo "🔧 Setup:"
	@echo "  make setup-compose   - Generate compose configuration"
	@echo ""
	@echo "⚠️  Local (emergency only):"
	@echo "  make local-proto     - Generate protobuf locally"
	@echo "  make local-build     - Build locally"
	@echo "  make local-test      - Test locally"

all: proto test build

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

# Proto generation
proto:
	@echo "⚠️  Use 'make compose-build' instead (includes proto generation)!"
	@exit 1

# ═══════════════════════════════════════════════
# Podman Compose targets
# ═══════════════════════════════════════════════

compose-dev:
	@echo "🚀 Starting development environment..."
	cd deploy/compose && podman-compose -f docker-compose.dev.yml up

compose-test:
	@echo "🧪 Running tests in containers..."
	cd deploy/compose && podman-compose -f docker-compose.test.yml up --abort-on-container-exit
	cd deploy/compose && podman-compose -f docker-compose.test.yml down

compose-build:
	@echo "🔨 Building binaries in containers..."
	cd deploy/compose && VERSION=$(VERSION) podman-compose -f docker-compose.build.yml up
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
	podman volume rm ocserv-agent_go-cache ocserv-agent_go-build-cache ocserv-agent_go-test-cache || true

# ═══════════════════════════════════════════════
# Setup
# ═══════════════════════════════════════════════

setup-compose:
	@./deploy/scripts/generate-compose.sh

# ═══════════════════════════════════════════════
# EMERGENCY: Local build (только для отладки!)
# ═══════════════════════════════════════════════

local-proto:
	@echo "⚠️  WARNING: Generating proto locally (not in container)"
	@echo "This should only be used for emergency debugging!"
	@sleep 2
	@echo "Generating protobuf code..."
	protoc --go_out=. --go-grpc_out=. \
		--go_opt=paths=source_relative \
		--go-grpc_opt=paths=source_relative \
		pkg/proto/agent/v1/agent.proto

local-build:
	@echo "⚠️  WARNING: Building locally (not in container)"
	@echo "This should only be used for emergency debugging!"
	@sleep 2
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/ocserv-agent ./cmd/agent

local-test:
	@echo "⚠️  WARNING: Testing locally (not in container)"
	@sleep 2
	go test -v -race -cover ./...

# ═══════════════════════════════════════════════
# Utility targets
# ═══════════════════════════════════════════════

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

lint:
	golangci-lint run ./...

install: build
	sudo mkdir -p /etc/ocserv-agent/certs
	sudo cp bin/ocserv-agent /etc/ocserv-agent/ocserv-agent
	sudo chmod +x /etc/ocserv-agent/ocserv-agent
	sudo cp config.yaml.example /etc/ocserv-agent/config.yaml
	sudo cp deploy/systemd/ocserv-agent.service /etc/systemd/system/
	sudo systemctl daemon-reload

clean:
	rm -rf bin/ coverage.out coverage.html tmp/
	find . -name "*.pb.go" -delete

# ═══════════════════════════════════════════════
# Security Testing
# ═══════════════════════════════════════════════

security-check:
	@echo "🔒 Running all security scans..."
	@./scripts/security-check.sh

security-gosec:
	@echo "🔒 Running Gosec security scanner..."
	@./scripts/security-check.sh gosec

security-govulncheck:
	@echo "🔒 Running govulncheck..."
	@./scripts/security-check.sh govulncheck

security-trivy:
	@echo "🔒 Running Trivy scanner..."
	@./scripts/security-check.sh trivy
