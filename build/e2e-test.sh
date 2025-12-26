#!/bin/bash
# E2E Testing Helper Script for ocserv-agent
# Manages E2E testing environment with OracleLinux 10 and ocserv

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
COMPOSE_FILE="$SCRIPT_DIR/docker-compose.e2e.yaml"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if podman-compose is installed
check_requirements() {
    log_info "Checking requirements..."

    if ! command -v podman &> /dev/null; then
        log_error "podman is not installed"
        exit 1
    fi

    if ! command -v podman-compose &> /dev/null; then
        log_error "podman-compose is not installed"
        exit 1
    fi

    log_success "All requirements satisfied"
}

# Build E2E containers
build_containers() {
    log_info "Building E2E containers..."

    cd "$PROJECT_ROOT"
    podman-compose -f "$COMPOSE_FILE" build

    log_success "Containers built successfully"
}

# Start E2E environment
start_environment() {
    log_info "Starting E2E environment..."

    cd "$PROJECT_ROOT"
    podman-compose -f "$COMPOSE_FILE" up -d

    log_info "Waiting for containers to be healthy..."
    sleep 10

    # Wait for ocserv socket
    local max_wait=60
    local elapsed=0

    while [ $elapsed -lt $max_wait ]; do
        if podman exec ocserv-e2e-test test -S /var/run/ocserv/ocserv.sock 2>/dev/null; then
            log_success "ocserv socket is ready"
            break
        fi

        log_info "Waiting for ocserv socket... ($elapsed/$max_wait)"
        sleep 5
        elapsed=$((elapsed + 5))
    done

    if [ $elapsed -ge $max_wait ]; then
        log_error "Timeout waiting for ocserv socket"
        return 1
    fi

    log_success "E2E environment started successfully"
}

# Stop E2E environment
stop_environment() {
    log_info "Stopping E2E environment..."

    cd "$PROJECT_ROOT"
    podman-compose -f "$COMPOSE_FILE" down

    log_success "E2E environment stopped"
}

# Show logs
show_logs() {
    local service="${1:-}"

    cd "$PROJECT_ROOT"

    if [ -n "$service" ]; then
        podman-compose -f "$COMPOSE_FILE" logs -f "$service"
    else
        podman-compose -f "$COMPOSE_FILE" logs -f
    fi
}

# Run E2E tests
run_tests() {
    log_info "Running E2E tests..."

    # Ensure environment is running
    if ! podman ps | grep -q ocserv-e2e-test; then
        log_warning "E2E environment not running, starting..."
        start_environment
    fi

    # Run tests inside ocserv container
    log_info "Executing E2E tests..."

    cd "$PROJECT_ROOT"

    # Copy test binary to container
    podman exec ocserv-e2e-test /bin/bash -c "
        export OCSERV_SOCKET_PATH=/var/run/ocserv/ocserv.sock
        export OCCTL_PATH=/usr/bin/occtl
        export CONFIG_PER_USER_DIR=/etc/ocserv/config-per-user

        # Check socket
        ls -la /var/run/ocserv/

        # Test occtl command
        occtl -s /var/run/ocserv/ocserv.sock show status
    "

    local exit_code=$?

    if [ $exit_code -eq 0 ]; then
        log_success "E2E tests passed"
    else
        log_error "E2E tests failed with exit code: $exit_code"
        return $exit_code
    fi
}

# Exec into container
exec_container() {
    local container="${1:-ocserv-e2e-test}"
    local shell="${2:-/bin/bash}"

    log_info "Executing shell in container: $container"
    podman exec -it "$container" "$shell"
}

# Show status
show_status() {
    log_info "E2E Environment Status:"
    echo

    cd "$PROJECT_ROOT"
    podman-compose -f "$COMPOSE_FILE" ps

    echo
    log_info "Socket status:"
    if podman exec ocserv-e2e-test test -S /var/run/ocserv/ocserv.sock 2>/dev/null; then
        log_success "ocserv socket exists"
        podman exec ocserv-e2e-test ls -la /var/run/ocserv/ocserv.sock
    else
        log_error "ocserv socket not found"
    fi

    echo
    log_info "ocserv status:"
    podman exec ocserv-e2e-test occtl -s /var/run/ocserv/ocserv.sock show status 2>/dev/null || \
        log_error "Failed to get ocserv status"
}

# Cleanup
cleanup() {
    log_info "Cleaning up E2E environment..."

    cd "$PROJECT_ROOT"
    podman-compose -f "$COMPOSE_FILE" down -v

    # Remove dangling images
    log_info "Removing dangling images..."
    podman image prune -f

    log_success "Cleanup completed"
}

# Show help
show_help() {
    cat << EOF
E2E Testing Helper for ocserv-agent

Usage: $0 [command]

Commands:
    build       Build E2E containers
    start       Start E2E environment
    stop        Stop E2E environment
    restart     Restart E2E environment
    test        Run E2E tests
    logs        Show logs (optionally specify service)
    exec        Execute shell in container
    status      Show environment status
    cleanup     Stop and remove all containers and volumes
    help        Show this help message

Examples:
    $0 start                    # Start E2E environment
    $0 test                     # Run E2E tests
    $0 logs ocserv-e2e         # Show ocserv logs
    $0 exec ocserv-e2e-test    # Execute shell in ocserv container
    $0 status                   # Show status
    $0 cleanup                  # Full cleanup

EOF
}

# Main command dispatcher
main() {
    case "${1:-}" in
        build)
            check_requirements
            build_containers
            ;;
        start)
            check_requirements
            start_environment
            ;;
        stop)
            stop_environment
            ;;
        restart)
            stop_environment
            start_environment
            ;;
        test)
            check_requirements
            run_tests
            ;;
        logs)
            show_logs "${2:-}"
            ;;
        exec)
            exec_container "${2:-ocserv-e2e-test}" "${3:-/bin/bash}"
            ;;
        status)
            show_status
            ;;
        cleanup)
            cleanup
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            log_error "Unknown command: ${1:-}"
            echo
            show_help
            exit 1
            ;;
    esac
}

# Run main
main "$@"
