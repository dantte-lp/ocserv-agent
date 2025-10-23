#!/bin/bash
# Local CI/CD Testing Script
# Run all checks locally before pushing to GitHub to save Actions hours

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
RUN_TESTS=${RUN_TESTS:-true}
RUN_LINT=${RUN_LINT:-true}
RUN_SECURITY=${RUN_SECURITY:-false}  # Security checks are slow, disabled by default
RUN_BUILD=${RUN_BUILD:-true}
SKIP_PROTO=${SKIP_PROTO:-false}  # Skip protobuf generation if already done

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}ocserv-agent Local CI Test Runner${NC}"
echo -e "${BLUE}========================================${NC}"
echo

# Function to print section headers
print_section() {
    echo
    echo -e "${BLUE}>>> $1${NC}"
    echo
}

# Function to print success
print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

# Function to print error
print_error() {
    echo -e "${RED}✗ $1${NC}"
}

# Function to print warning
print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

# Check if in project root
if [ ! -f "go.mod" ]; then
    print_error "Must run from project root directory"
    exit 1
fi

# 1. Generate Protobuf (if needed)
if [ "$SKIP_PROTO" != "true" ]; then
    print_section "Generating Protobuf Code"

    # Check if protoc is installed
    if ! command -v protoc &> /dev/null; then
        print_warning "protoc not found, skipping proto generation"
        print_warning "Install with: sudo apt-get install protobuf-compiler"
    else
        # Check if plugins are installed
        if ! command -v protoc-gen-go &> /dev/null; then
            print_warning "protoc-gen-go not found, installing..."
            go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
        fi
        if ! command -v protoc-gen-go-grpc &> /dev/null; then
            print_warning "protoc-gen-go-grpc not found, installing..."
            go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
        fi

        protoc --go_out=. --go-grpc_out=. \
            --go_opt=paths=source_relative \
            --go-grpc_opt=paths=source_relative \
            pkg/proto/agent/v1/agent.proto

        print_success "Protobuf code generated"
    fi
else
    print_warning "Skipping protobuf generation (SKIP_PROTO=true)"
fi

# 2. Download dependencies
print_section "Downloading Dependencies"
go mod download
print_success "Dependencies downloaded"

# 3. Verify dependencies
print_section "Verifying Dependencies"
go mod verify
print_success "Dependencies verified"

# 4. Check formatting
print_section "Checking Code Formatting"
UNFORMATTED=$(gofmt -s -l . | grep -v '^vendor/' | grep -v '.pb.go$' || true)
if [ -n "$UNFORMATTED" ]; then
    print_error "Code is not formatted:"
    echo "$UNFORMATTED"
    echo
    print_warning "Run: gofmt -s -w ."
    exit 1
fi
print_success "Code formatting OK"

# 5. Run go vet
print_section "Running go vet"
go vet ./...
print_success "go vet passed"

# 6. Check go mod tidy
print_section "Checking go.mod and go.sum"
cp go.mod go.mod.backup
cp go.sum go.sum.backup
go mod tidy
if ! diff -u go.mod.backup go.mod > /dev/null 2>&1; then
    print_error "go.mod is not tidy"
    print_warning "Run: go mod tidy"
    mv go.mod.backup go.mod
    mv go.sum.backup go.sum
    exit 1
fi
if ! diff -u go.sum.backup go.sum > /dev/null 2>&1; then
    print_error "go.sum is not tidy"
    print_warning "Run: go mod tidy"
    mv go.mod.backup go.mod
    mv go.sum.backup go.sum
    exit 1
fi
rm go.mod.backup go.sum.backup
print_success "go.mod and go.sum are tidy"

# 7. Run tests
if [ "$RUN_TESTS" == "true" ]; then
    print_section "Running Tests"

    # Run tests with race detector and coverage
    go test -v -race -coverprofile=coverage.out -covermode=atomic ./...

    # Generate coverage report
    go tool cover -func=coverage.out | tail -1

    print_success "Tests passed"
else
    print_warning "Skipping tests (RUN_TESTS=false)"
fi

# 8. Build for multiple platforms
if [ "$RUN_BUILD" == "true" ]; then
    print_section "Building for Multiple Platforms"

    PLATFORMS=(
        "linux/amd64"
        "linux/arm64"
        "freebsd/amd64"
        "freebsd/arm64"
    )

    mkdir -p bin

    for PLATFORM in "${PLATFORMS[@]}"; do
        GOOS=${PLATFORM%/*}
        GOARCH=${PLATFORM#*/}

        OUTPUT="bin/ocserv-agent-${GOOS}-${GOARCH}"

        echo "Building ${GOOS}/${GOARCH}..."

        CGO_ENABLED=0 GOOS=$GOOS GOARCH=$GOARCH \
            go build -trimpath -ldflags="-s -w" -o "$OUTPUT" ./cmd/agent

        print_success "Built ${OUTPUT}"
    done

    echo
    echo "Built binaries:"
    ls -lh bin/
else
    print_warning "Skipping builds (RUN_BUILD=false)"
fi

# 9. Run linters
if [ "$RUN_LINT" == "true" ]; then
    print_section "Running Linters"

    # Check if golangci-lint is installed
    if ! command -v golangci-lint &> /dev/null; then
        print_warning "golangci-lint not found"
        print_warning "Install with: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b \$(go env GOPATH)/bin"
    else
        golangci-lint run --timeout=5m
        print_success "golangci-lint passed"
    fi

    # Check markdown files
    if command -v markdownlint &> /dev/null; then
        echo "Checking markdown files..."
        markdownlint '**/*.md' --ignore node_modules --ignore vendor || print_warning "Markdown lint warnings"
    else
        print_warning "markdownlint not found (npm install -g markdownlint-cli)"
    fi

    # Check YAML files
    if command -v yamllint &> /dev/null; then
        echo "Checking YAML files..."
        yamllint . || print_warning "YAML lint warnings"
    else
        print_warning "yamllint not found (pip install yamllint)"
    fi
else
    print_warning "Skipping linters (RUN_LINT=false)"
fi

# 10. Run security checks
if [ "$RUN_SECURITY" == "true" ]; then
    print_section "Running Security Checks"

    # Check if gosec is installed
    if ! command -v gosec &> /dev/null; then
        print_warning "gosec not found"
        print_warning "Install with: go install github.com/securego/gosec/v2/cmd/gosec@latest"
    else
        gosec -fmt sarif -out gosec.sarif ./... || print_warning "gosec found issues"
        print_success "gosec scan completed (check gosec.sarif)"
    fi

    # Check for known vulnerabilities
    if ! command -v govulncheck &> /dev/null; then
        print_warning "govulncheck not found"
        print_warning "Install with: go install golang.org/x/vuln/cmd/govulncheck@latest"
    else
        govulncheck ./...
        print_success "No known vulnerabilities"
    fi
else
    print_warning "Skipping security checks (RUN_SECURITY=false)"
fi

# Summary
echo
echo -e "${BLUE}========================================${NC}"
echo -e "${GREEN}✓ All checks passed!${NC}"
echo -e "${BLUE}========================================${NC}"
echo
echo "Safe to push to GitHub!"
echo
echo "Environment variables:"
echo "  RUN_TESTS=${RUN_TESTS}      - Run Go tests"
echo "  RUN_LINT=${RUN_LINT}        - Run linters"
echo "  RUN_SECURITY=${RUN_SECURITY}    - Run security scans (slow)"
echo "  RUN_BUILD=${RUN_BUILD}       - Build binaries"
echo "  SKIP_PROTO=${SKIP_PROTO}     - Skip protobuf generation"
echo
echo "Example: RUN_SECURITY=true ./scripts/test-local.sh"
