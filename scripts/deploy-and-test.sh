#!/bin/bash
# Deploy and test ocserv-agent on production server
# Usage: SERVER=<ip> SSH_PASS=<password> ./scripts/deploy-and-test.sh [version]
#    or: ./scripts/deploy-and-test.sh [version] [server] [password]

set -e

# Configuration from environment or arguments
VERSION="${1:-v0.3.0-21-gcb1f848}"
SERVER="${2:-${SERVER:-localhost}}"
SSH_PASS="${3:-${SSH_PASS}}"
SSH_USER="${SSH_USER:-root}"
INSTALL_DIR="/etc/ocserv-agent"
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

# Check if SSH_PASS is set
if [ -z "$SSH_PASS" ]; then
    echo -e "${RED}✗ SSH_PASS environment variable not set${NC}"
    echo -e "${YELLOW}Usage:${NC}"
    echo -e "  SERVER=<ip> SSH_PASS=<password> $0 [version]"
    echo -e "  or: $0 [version] [server] [password]"
    exit 1
fi

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
ssh_cmd "systemctl stop ocserv-agent 2>/dev/null || pkill ocserv-agent || true"
sleep 2
echo -e "${GREEN}✓ Service stopped${NC}"
echo ""

echo -e "${BLUE}[4/7] Extracting and installing new binary...${NC}"
ssh_cmd "cd /tmp && tar -xzf ${ARCHIVE} && mv ocserv-agent ${INSTALL_DIR}/ocserv-agent && chmod +x ${INSTALL_DIR}/ocserv-agent"
echo -e "${GREEN}✓ Binary installed${NC}"
echo ""

echo -e "${BLUE}[5/7] Starting ocserv-agent...${NC}"
ssh_cmd "cd ${INSTALL_DIR} && nohup ./ocserv-agent -config config.yaml > /tmp/ocserv-agent.log 2>&1 &"
sleep 3
echo -e "${GREEN}✓ Agent started${NC}"
echo ""

echo -e "${BLUE}[6/7] Checking agent status...${NC}"
ssh_cmd "ps aux | grep ocserv-agent | grep -v grep || echo 'Agent not running!'"
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
echo -e "  - Review service logs: ssh root@${SERVER} 'tail -f /tmp/ocserv-agent.log | jq .'"
echo -e "  - Test other RPCs with grpcurl"
echo -e "  - Run test suite: SERVER=${SERVER} SSH_PASS=\$SSH_PASS ./scripts/test-grpc.sh"
echo ""
