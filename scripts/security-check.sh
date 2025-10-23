#!/bin/bash
# Local Security Testing Script
# Runs all security checks locally using Podman Compose

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
COMPOSE_FILE="deploy/compose/security.yml"
RESULTS_DIR="deploy/compose/security-results"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}ocserv-agent Security Testing Suite${NC}"
echo -e "${BLUE}========================================${NC}"
echo

# Check if in project root
if [ ! -f "go.mod" ]; then
    echo -e "${RED}Error: Must run from project root directory${NC}"
    exit 1
fi

# Check if podman-compose is installed
if ! command -v podman-compose &> /dev/null; then
    echo -e "${RED}Error: podman-compose not found${NC}"
    echo "Install with: pip install podman-compose"
    exit 1
fi

# Create results directory
echo -e "${BLUE}>>> Preparing results directory${NC}"
mkdir -p "$RESULTS_DIR"
echo "Results will be saved to: $RESULTS_DIR"
echo

# Clean up old results
if [ -n "$(ls -A $RESULTS_DIR 2>/dev/null)" ]; then
    echo -e "${YELLOW}Cleaning old results...${NC}"
    rm -f "$RESULTS_DIR"/*
fi

# Run security tests
echo -e "${BLUE}>>> Running security tests${NC}"
echo

if [ "$1" == "gosec" ]; then
    echo "Running Gosec only..."
    podman-compose -f "$COMPOSE_FILE" up gosec-fixed
elif [ "$1" == "govulncheck" ]; then
    echo "Running govulncheck only..."
    podman-compose -f "$COMPOSE_FILE" up govulncheck
elif [ "$1" == "trivy" ]; then
    echo "Running Trivy only..."
    podman-compose -f "$COMPOSE_FILE" up trivy
else
    echo "Running all security tests..."
    podman-compose -f "$COMPOSE_FILE" up
fi

# Clean up containers
echo
echo -e "${BLUE}>>> Cleaning up containers${NC}"
podman-compose -f "$COMPOSE_FILE" down

# Display results
echo
echo -e "${BLUE}========================================${NC}"
echo -e "${GREEN}Security Scan Results${NC}"
echo -e "${BLUE}========================================${NC}"
echo

if [ -f "$RESULTS_DIR/gosec-fixed.sarif" ]; then
    echo -e "${GREEN}✓ Gosec SARIF (GitHub-compatible)${NC}"
    GOSEC_ISSUES=$(jq '.runs[0].results | length' "$RESULTS_DIR/gosec-fixed.sarif")
    echo "  Found $GOSEC_ISSUES security issues"
    echo "  File: $RESULTS_DIR/gosec-fixed.sarif"
    echo
fi

if [ -f "$RESULTS_DIR/govulncheck.json" ]; then
    echo -e "${GREEN}✓ govulncheck${NC}"
    if grep -q '"vulns":null' "$RESULTS_DIR/govulncheck.json" 2>/dev/null; then
        echo "  No known vulnerabilities found"
    else
        echo "  Check file for vulnerabilities: $RESULTS_DIR/govulncheck.json"
    fi
    echo
fi

if [ -f "$RESULTS_DIR/trivy.sarif" ]; then
    echo -e "${GREEN}✓ Trivy SARIF${NC}"
    TRIVY_ISSUES=$(jq '.runs[0].results | length' "$RESULTS_DIR/trivy.sarif")
    echo "  Found $TRIVY_ISSUES vulnerabilities"
    echo "  File: $RESULTS_DIR/trivy.sarif"
    echo
fi

# Summary
echo -e "${BLUE}========================================${NC}"
echo -e "${GREEN}All security scans completed!${NC}"
echo -e "${BLUE}========================================${NC}"
echo
echo "Results directory: $RESULTS_DIR"
echo
echo "Commands to view results:"
echo "  Gosec findings:   cat $RESULTS_DIR/gosec-fixed.sarif | jq '.runs[0].results[]'"
echo "  Trivy findings:   cat $RESULTS_DIR/trivy.sarif | jq '.runs[0].results[]'"
echo "  govulncheck:      cat $RESULTS_DIR/govulncheck.json | jq"
echo
echo "To run specific test:"
echo "  ./scripts/security-check.sh gosec"
echo "  ./scripts/security-check.sh govulncheck"
echo "  ./scripts/security-check.sh trivy"
echo
