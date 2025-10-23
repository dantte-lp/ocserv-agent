#!/bin/bash
#
# Build All - Unified script for testing and building
#
# This script runs the complete CI/CD pipeline locally:
# 1. Security scans (gosec, govulncheck, trivy)
# 2. Unit tests and linting
# 3. Multi-platform builds (Linux/FreeBSD, amd64/arm64)
#
# Usage:
#   ./scripts/build-all.sh              # Run everything
#   ./scripts/build-all.sh security     # Security only
#   ./scripts/build-all.sh test         # Tests only
#   ./scripts/build-all.sh build        # Build only
#

set -e  # Exit on error

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Directories
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
COMPOSE_DIR="$PROJECT_ROOT/deploy/compose"
SECURITY_RESULTS_DIR="$COMPOSE_DIR/security-results"

# Version
VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo "dev")}"

# Functions
log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
}

separator() {
    echo ""
    echo "═══════════════════════════════════════════════════════════════"
    echo ""
}

# Security scan
run_security() {
    separator
    log_info "Running security scans..."
    echo ""

    # Create results directory
    mkdir -p "$SECURITY_RESULTS_DIR"

    # Run security compose
    cd "$COMPOSE_DIR"

    log_info "Starting security containers (gosec, govulncheck, trivy)..."
    podman-compose -f security.yml up --abort-on-container-exit

    # Cleanup
    podman-compose -f security.yml down

    separator
    log_success "Security scans completed!"
    log_info "Results saved to: $SECURITY_RESULTS_DIR/"

    # Show summary
    if [ -f "$SECURITY_RESULTS_DIR/gosec-fixed.sarif" ]; then
        log_info "Gosec SARIF: security-results/gosec-fixed.sarif"
    fi
    if [ -f "$SECURITY_RESULTS_DIR/trivy.sarif" ]; then
        log_info "Trivy SARIF: security-results/trivy.sarif"
    fi
    if [ -f "$SECURITY_RESULTS_DIR/govulncheck.json" ]; then
        log_info "govulncheck JSON: security-results/govulncheck.json"
    fi
}

# Unit tests
run_tests() {
    separator
    log_info "Running unit tests and linting..."
    echo ""

    cd "$COMPOSE_DIR"

    log_info "Starting test containers..."
    podman-compose -f docker-compose.test.yml up --abort-on-container-exit

    # Check test exit code
    TEST_EXIT=$?

    # Cleanup
    podman-compose -f docker-compose.test.yml down

    if [ $TEST_EXIT -ne 0 ]; then
        log_error "Tests failed!"
        return 1
    fi

    separator
    log_success "All tests passed!"

    # Show coverage if exists
    if [ -f "$PROJECT_ROOT/coverage.out" ]; then
        log_info "Coverage report: coverage.html"
    fi
}

# Multi-platform build
run_build() {
    separator
    log_info "Building for all platforms..."
    echo ""
    log_info "Version: $VERSION"
    log_info "Platforms: Linux/FreeBSD (amd64, arm64)"
    echo ""

    cd "$COMPOSE_DIR"

    log_info "Starting build containers..."
    VERSION="$VERSION" podman-compose -f docker-compose.build.yml up --abort-on-container-exit

    # Cleanup
    podman-compose -f docker-compose.build.yml down

    separator
    log_success "Build completed!"

    # Show artifacts
    if [ -d "$PROJECT_ROOT/bin" ]; then
        log_info "Artifacts in bin/:"
        ls -lh "$PROJECT_ROOT/bin/" | grep -E "\.tar\.gz$|\.sha256$" || true
    fi
}

# Show final summary
show_summary() {
    separator
    echo -e "${GREEN}🎉 All stages completed successfully!${NC}"
    echo ""
    echo "Summary:"
    echo "  ✅ Security scans passed"
    echo "  ✅ Tests passed"
    echo "  ✅ Multi-platform build successful"
    echo ""
    echo "Artifacts:"

    if [ -d "$PROJECT_ROOT/bin" ]; then
        echo ""
        echo "Binaries (bin/):"
        ls -1 "$PROJECT_ROOT/bin/" | grep -E "\.tar\.gz$" | sed 's/^/    - /' || true
        echo ""
        echo "Checksums (bin/):"
        ls -1 "$PROJECT_ROOT/bin/" | grep -E "\.sha256$" | sed 's/^/    - /' || true
    fi

    if [ -d "$SECURITY_RESULTS_DIR" ]; then
        echo ""
        echo "Security reports (security-results/):"
        ls -1 "$SECURITY_RESULTS_DIR/" | sed 's/^/    - /' || true
    fi

    echo ""
    log_info "Ready to commit and push!"
    separator
}

# Main
main() {
    cd "$PROJECT_ROOT"

    echo ""
    echo "╔═══════════════════════════════════════════════════════════════╗"
    echo "║                 ocserv-agent Build Pipeline                  ║"
    echo "╚═══════════════════════════════════════════════════════════════╝"
    echo ""
    log_info "Version: $VERSION"
    log_info "Working directory: $PROJECT_ROOT"
    echo ""

    # Parse command
    COMMAND="${1:-all}"

    case "$COMMAND" in
        security)
            run_security
            ;;
        test)
            run_tests
            ;;
        build)
            run_build
            ;;
        all)
            # Run all stages
            run_security || { log_error "Security scans failed!"; exit 1; }
            run_tests || { log_error "Tests failed!"; exit 1; }
            run_build || { log_error "Build failed!"; exit 1; }
            show_summary
            ;;
        *)
            log_error "Unknown command: $COMMAND"
            echo ""
            echo "Usage: $0 [security|test|build|all]"
            echo ""
            echo "Commands:"
            echo "  security  - Run security scans only"
            echo "  test      - Run unit tests and linting only"
            echo "  build     - Run multi-platform build only"
            echo "  all       - Run everything (default)"
            exit 1
            ;;
    esac
}

# Trap errors
trap 'log_error "Build pipeline failed at line $LINENO"' ERR

# Run
main "$@"
