#!/bin/bash
# Quick local check before commit
# Fast checks only - formatting, vet, basic tests

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

echo "🔍 Quick local check..."

# Format check and auto-fix
echo -n "Formatting... "
UNFORMATTED=$(gofmt -s -l . | grep -v '^vendor/' | grep -v '.pb.go$')
if [ -n "$UNFORMATTED" ]; then
    echo -e "${RED}FIXING${NC}"
    echo "Auto-formatting files:"
    echo "$UNFORMATTED"
    gofmt -s -w . > /dev/null 2>&1
    echo -e "${GREEN}FIXED${NC} (files formatted automatically)"
else
    echo -e "${GREEN}OK${NC}"
fi

# Vet check
echo -n "go vet... "
if ! go vet ./... > /dev/null 2>&1; then
    echo -e "${RED}FAIL${NC}"
    go vet ./...
    exit 1
fi
echo -e "${GREEN}OK${NC}"

# Build check
echo -n "Build... "
if ! go build -o /tmp/ocserv-agent ./cmd/agent > /dev/null 2>&1; then
    echo -e "${RED}FAIL${NC}"
    go build ./cmd/agent
    exit 1
fi
rm -f /tmp/ocserv-agent
echo -e "${GREEN}OK${NC}"

# Basic tests
echo -n "Tests... "
if ! go test ./... > /dev/null 2>&1; then
    echo -e "${RED}FAIL${NC}"
    go test ./...
    exit 1
fi
echo -e "${GREEN}OK${NC}"

echo
echo -e "${GREEN}✓ All quick checks passed!${NC}"
echo
echo "For full CI checks, run: ./scripts/test-local.sh"
