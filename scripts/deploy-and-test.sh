#!/bin/bash
# Deploy and test ocserv-agent on production server
# Usage: ./scripts/deploy-and-test.sh [version]

set -e

# Configuration
SERVER="195.238.126.25"
SSH_USER="root"
SSH_PASS="lnwwPBE43PkuLKq0"
INSTALL_DIR="/etc/ocserv-agent"
VERSION="${1:-v0.3.0-21-gcb1f848}"
ARCHIVE="ocserv-agent-${VERSION}-linux-amd64.tar.gz"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}╔════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║   ocserv-agent Deployment & Testing Script    ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════╝${NC}"
echo ""

# Check if archive exists
if [ ! -f "bin/${ARCHIVE}" ]; then
    echo -e "${RED}✗ Archive not found: bin/${ARCHIVE}${NC}"
    echo -e "${YELLOW}Available versions:${NC}"
    ls -1 bin/*.tar.gz 2>/dev/null || echo "No archives found"
    exit 1
fi

echo -e "${GREEN}✓ Found archive: bin/${ARCHIVE}${NC}"
echo ""

# Function to run SSH command
ssh_cmd() {
    sshpass -p "${SSH_PASS}" ssh -o StrictHostKeyChecking=no "${SSH_USER}@${SERVER}" "$@"
}

# Function to copy file via SCP
scp_copy() {
    sshpass -p "${SSH_PASS}" scp -o StrictHostKeyChecking=no "$1" "${SSH_USER}@${SERVER}:$2"
}

echo -e "${BLUE}[1/7] Checking server connectivity...${NC}"
if ssh_cmd "echo 'Server is reachable'" > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Server is reachable${NC}"
else
    echo -e "${RED}✗ Cannot connect to server${NC}"
    exit 1
fi
echo ""

echo -e "${BLUE}[2/7] Copying archive to server...${NC}"
scp_copy "bin/${ARCHIVE}" "/tmp/${ARCHIVE}"
echo -e "${GREEN}✓ Archive copied${NC}"
echo ""

echo -e "${BLUE}[3/7] Stopping ocserv-agent service...${NC}"
ssh_cmd "systemctl stop ocserv-agent || true"
sleep 2
echo -e "${GREEN}✓ Service stopped${NC}"
echo ""

echo -e "${BLUE}[4/7] Extracting and installing new binary...${NC}"
ssh_cmd "cd /tmp && tar -xzf ${ARCHIVE} && mv ocserv-agent ${INSTALL_DIR}/ocserv-agent && chmod +x ${INSTALL_DIR}/ocserv-agent"
echo -e "${GREEN}✓ Binary installed${NC}"
echo ""

echo -e "${BLUE}[5/7] Starting ocserv-agent service...${NC}"
ssh_cmd "systemctl start ocserv-agent"
sleep 3
echo -e "${GREEN}✓ Service started${NC}"
echo ""

echo -e "${BLUE}[6/7] Checking service status...${NC}"
ssh_cmd "systemctl status ocserv-agent --no-pager -n 10 || true"
echo ""

echo -e "${BLUE}[7/7] Testing gRPC reflection API...${NC}"
echo -e "${YELLOW}Listing available services:${NC}"
ssh_cmd "grpcurl -cacert ${INSTALL_DIR}/certs/ca.crt -cert ${INSTALL_DIR}/certs/agent.crt -key ${INSTALL_DIR}/certs/agent.key localhost:9090 list" || {
    echo -e "${RED}✗ gRPC reflection test failed${NC}"
    echo -e "${YELLOW}Checking if grpcurl is installed...${NC}"
    ssh_cmd "which grpcurl || echo 'grpcurl not found'"
    exit 1
}
echo ""

echo -e "${GREEN}✓ Listing AgentService methods:${NC}"
ssh_cmd "grpcurl -cacert ${INSTALL_DIR}/certs/ca.crt -cert ${INSTALL_DIR}/certs/agent.crt -key ${INSTALL_DIR}/certs/agent.key localhost:9090 list agent.v1.AgentService" || true
echo ""

echo -e "${GREEN}✓ Testing HealthCheck RPC:${NC}"
ssh_cmd "grpcurl -cacert ${INSTALL_DIR}/certs/ca.crt -cert ${INSTALL_DIR}/certs/agent.crt -key ${INSTALL_DIR}/certs/agent.key -d '{\"tier\": 1}' localhost:9090 agent.v1.AgentService/HealthCheck" || true
echo ""

echo -e "${BLUE}╔════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║          Deployment Complete!                  ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${YELLOW}Version deployed: ${VERSION}${NC}"
echo -e "${YELLOW}Server: ${SERVER}${NC}"
echo ""
echo -e "${GREEN}Next steps:${NC}"
echo -e "  - Review service logs: ssh root@${SERVER} 'journalctl -u ocserv-agent -f'"
echo -e "  - Test other RPCs with grpcurl"
echo -e "  - Monitor performance and debug logs"
echo ""
