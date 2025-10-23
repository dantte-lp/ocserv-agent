#!/bin/bash
# Test ocserv-agent gRPC API using grpcurl
# Usage: ./scripts/test-grpc.sh [server]

set -e

# Configuration
SERVER="${1:-195.238.126.25}"
SSH_USER="root"
SSH_PASS="lnwwPBE43PkuLKq0"
GRPC_HOST="localhost:9090"
CERT_DIR="/etc/ocserv-agent/certs"

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${BLUE}╔════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║       ocserv-agent gRPC API Test Suite        ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${YELLOW}Server: ${SERVER}${NC}"
echo -e "${YELLOW}gRPC endpoint: ${GRPC_HOST}${NC}"
echo ""

# Function to run grpcurl on remote server
grpc_call() {
    sshpass -p "${SSH_PASS}" ssh -o StrictHostKeyChecking=no "${SSH_USER}@${SERVER}" \
        "grpcurl -cacert ${CERT_DIR}/ca.crt -cert ${CERT_DIR}/agent.crt -key ${CERT_DIR}/agent.key $@"
}

echo -e "${GREEN}[1/6] Testing gRPC Reflection - List Services${NC}"
grpc_call "${GRPC_HOST}" list
echo ""

echo -e "${GREEN}[2/6] Testing gRPC Reflection - List Methods${NC}"
grpc_call "${GRPC_HOST}" list agent.v1.AgentService
echo ""

echo -e "${GREEN}[3/6] Testing HealthCheck RPC (Tier 1)${NC}"
grpc_call -d '{"tier": 1}' "${GRPC_HOST}" agent.v1.AgentService/HealthCheck
echo ""

echo -e "${GREEN}[4/6] Testing ExecuteCommand - occtl show status${NC}"
grpc_call -d '{"command_type": "occtl", "args": ["show", "status"]}' \
    "${GRPC_HOST}" agent.v1.AgentService/ExecuteCommand
echo ""

echo -e "${GREEN}[5/6] Testing ExecuteCommand - occtl show users${NC}"
grpc_call -d '{"command_type": "occtl", "args": ["show", "users"]}' \
    "${GRPC_HOST}" agent.v1.AgentService/ExecuteCommand
echo ""

echo -e "${GREEN}[6/6] Testing ExecuteCommand - systemctl status${NC}"
grpc_call -d '{"command_type": "systemctl", "args": ["status", "ocserv"]}' \
    "${GRPC_HOST}" agent.v1.AgentService/ExecuteCommand
echo ""

echo -e "${BLUE}╔════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║            All Tests Completed!                ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${YELLOW}Additional Commands:${NC}"
echo ""
echo -e "${GREEN}Describe message structure:${NC}"
echo -e "  grpc_call '${GRPC_HOST}' describe agent.v1.CommandRequest"
echo ""
echo -e "${GREEN}Describe service:${NC}"
echo -e "  grpc_call '${GRPC_HOST}' describe agent.v1.AgentService"
echo ""
echo -e "${GREEN}Check agent logs:${NC}"
echo -e "  ssh root@${SERVER} 'tail -f /tmp/ocserv-agent.log | jq .'"
echo ""
