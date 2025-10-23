# Build stage
FROM golang:1.25-trixie AS builder

WORKDIR /build

# Install protoc and protobuf development files for proto generation
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        protobuf-compiler \
        libprotobuf-dev && \
    rm -rf /var/lib/apt/lists/*

# Install Go proto plugins
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Generate protobuf code
RUN protoc -I. -I/usr/include --go_out=. --go-grpc_out=. \
    --go_opt=paths=source_relative \
    --go-grpc_opt=paths=source_relative \
    pkg/proto/agent/v1/agent.proto

# Build
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o ocserv-agent ./cmd/agent

# Runtime stage
FROM debian:trixie-slim

# Install dependencies
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        ca-certificates \
        ocserv \
        sudo \
        iproute2 \
        procps && \
    rm -rf /var/lib/apt/lists/*

# Create user
RUN useradd -r -s /bin/false -u 1000 ocserv-agent && \
    echo "ocserv-agent ALL=(ALL) NOPASSWD: /usr/bin/systemctl, /usr/sbin/occtl" >> /etc/sudoers

# Copy binary
COPY --from=builder /build/ocserv-agent /usr/local/bin/

# Config directory
RUN mkdir -p /etc/ocserv-agent/certs && \
    chown -R ocserv-agent:ocserv-agent /etc/ocserv-agent

# Copy example config
COPY config.yaml.example /etc/ocserv-agent/config.yaml

VOLUME /etc/ocserv-agent

USER ocserv-agent
EXPOSE 9090

ENTRYPOINT ["/usr/local/bin/ocserv-agent"]
CMD ["--config", "/etc/ocserv-agent/config.yaml"]
