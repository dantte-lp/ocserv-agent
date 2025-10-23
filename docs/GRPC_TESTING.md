# gRPC API Testing Guide

This guide explains how to test the ocserv-agent gRPC API using grpcurl.

## Prerequisites

- grpcurl installed on the server
- Valid mTLS certificates (CA, client cert, client key)
- Agent running with gRPC reflection enabled

## Quick Start

Use the automated test script:

```bash
./scripts/test-grpc.sh [server-ip]
```

Default server: `195.238.126.25`

## Manual Testing

### 1. List Available Services

```bash
grpcurl -cacert /etc/ocserv-agent/certs/ca.crt \
  -cert /etc/ocserv-agent/certs/agent.crt \
  -key /etc/ocserv-agent/certs/agent.key \
  localhost:9090 list
```

Expected output:
```
agent.v1.AgentService
grpc.reflection.v1.ServerReflection
grpc.reflection.v1alpha.ServerReflection
```

### 2. List Service Methods

```bash
grpcurl -cacert /etc/ocserv-agent/certs/ca.crt \
  -cert /etc/ocserv-agent/certs/agent.crt \
  -key /etc/ocserv-agent/certs/agent.key \
  localhost:9090 list agent.v1.AgentService
```

Expected output:
```
agent.v1.AgentService.AgentStream
agent.v1.AgentService.ExecuteCommand
agent.v1.AgentService.HealthCheck
agent.v1.AgentService.StreamLogs
agent.v1.AgentService.UpdateConfig
```

### 3. Test HealthCheck RPC

**Tier 1 (Basic):**
```bash
grpcurl -cacert /etc/ocserv-agent/certs/ca.crt \
  -cert /etc/ocserv-agent/certs/agent.crt \
  -key /etc/ocserv-agent/certs/agent.key \
  -d '{"tier": 1}' \
  localhost:9090 agent.v1.AgentService/HealthCheck
```

Expected response:
```json
{
  "healthy": true,
  "statusMessage": "OK",
  "checks": {
    "agent": "running",
    "config": "loaded"
  },
  "timestamp": "2025-10-23T05:18:15.783617992Z"
}
```

### 4. Test ExecuteCommand RPC

**occtl show status:**
```bash
grpcurl -cacert /etc/ocserv-agent/certs/ca.crt \
  -cert /etc/ocserv-agent/certs/agent.crt \
  -key /etc/ocserv-agent/certs/agent.key \
  -d '{"command_type": "occtl", "args": ["show", "status"]}' \
  localhost:9090 agent.v1.AgentService/ExecuteCommand
```

**occtl show users:**
```bash
grpcurl -cacert /etc/ocserv-agent/certs/ca.crt \
  -cert /etc/ocserv-agent/certs/agent.crt \
  -key /etc/ocserv-agent/certs/agent.key \
  -d '{"command_type": "occtl", "args": ["show", "users"]}' \
  localhost:9090 agent.v1.AgentService/ExecuteCommand
```

**systemctl status:**
```bash
grpcurl -cacert /etc/ocserv-agent/certs/ca.crt \
  -cert /etc/ocserv-agent/certs/agent.crt \
  -key /etc/ocserv-agent/certs/agent.key \
  -d '{"command_type": "systemctl", "args": ["status", "ocserv"]}' \
  localhost:9090 agent.v1.AgentService/ExecuteCommand
```

## Message Introspection

### Describe Message Structure

**CommandRequest:**
```bash
grpcurl -cacert /etc/ocserv-agent/certs/ca.crt \
  -cert /etc/ocserv-agent/certs/agent.crt \
  -key /etc/ocserv-agent/certs/agent.key \
  localhost:9090 describe agent.v1.CommandRequest
```

Output:
```
message CommandRequest {
  string request_id = 1;
  string command_type = 2;
  repeated string args = 3;
  int32 timeout_seconds = 4;
}
```

**CommandResponse:**
```bash
grpcurl -cacert /etc/ocserv-agent/certs/ca.crt \
  -cert /etc/ocserv-agent/certs/agent.crt \
  -key /etc/ocserv-agent/certs/agent.key \
  localhost:9090 describe agent.v1.CommandResponse
```

### Describe Service

```bash
grpcurl -cacert /etc/ocserv-agent/certs/ca.crt \
  -cert /etc/ocserv-agent/certs/agent.crt \
  -key /etc/ocserv-agent/certs/agent.key \
  localhost:9090 describe agent.v1.AgentService
```

## Debugging

### Check Agent Logs

**Raw logs:**
```bash
tail -f /tmp/ocserv-agent.log
```

**Formatted JSON logs:**
```bash
tail -f /tmp/ocserv-agent.log | jq .
```

**Filter by log level:**
```bash
tail -f /tmp/ocserv-agent.log | jq 'select(.level == "debug")'
```

**Filter by RPC method:**
```bash
tail -f /tmp/ocserv-agent.log | jq 'select(.method != null)'
```

### Common Issues

**Error: server does not support the reflection API**
- Solution: Ensure agent is built with reflection support (v0.3.0-21+ or later)
- Check: `./ocserv-agent -version`

**Error: connection refused**
- Check if agent is running: `ps aux | grep ocserv-agent`
- Check port binding: `netstat -tlnp | grep 9090`

**Error: certificate verification failed**
- Verify certificate paths
- Check certificate validity: `openssl x509 -in /etc/ocserv-agent/certs/agent.crt -noout -dates`
- Ensure CA matches server certificate

## Remote Testing

### From Control Server

If testing from a different machine:

```bash
grpcurl -cacert ca.crt \
  -cert client.crt \
  -key client.key \
  <agent-hostname>:9090 list
```

Replace `localhost:9090` with the agent's hostname or IP address.

### Using Deployment Script

Deploy new version and test:

```bash
./scripts/deploy-and-test.sh [version]
```

Example:
```bash
./scripts/deploy-and-test.sh v0.3.0-21-gcb1f848
```

## Production Testing Checklist

- [ ] gRPC reflection working (list services)
- [ ] HealthCheck RPC responds correctly
- [ ] ExecuteCommand works for occtl commands
- [ ] ExecuteCommand works for systemctl commands
- [ ] Debug logging enabled and working
- [ ] mTLS certificate validation working
- [ ] All RPC calls logged with correct level
- [ ] Performance acceptable (response time < 1s)

## See Also

- [Certificate Management](CERTIFICATES.md)
- [Local Testing](LOCAL_TESTING.md)
- [Production Testing](TESTING_PROD.md)
- [gRPC API Documentation](../pkg/proto/agent/v1/agent.proto)
