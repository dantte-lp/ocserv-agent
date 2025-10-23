# Dockerfile for mock-ocserv Unix socket server
# Multi-stage build for minimal final image
# Base images: golang:1.25-trixie (build), debian:trixie-slim (runtime)

FROM golang:1.25-trixie AS builder

WORKDIR /build

# Copy go.mod from project root
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy mock-ocserv source
COPY test/mock-ocserv/*.go ./

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o mock-ocserv .

# Final stage - minimal Debian runtime
FROM debian:trixie-slim

# Install ca-certificates (standard for all project images)
RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/mock-ocserv /app/mock-ocserv

# Create socket directory
RUN mkdir -p /var/run

# Default socket path
ENV SOCKET_PATH=/var/run/occtl.socket
ENV FIXTURES_PATH=/fixtures/ocserv/occtl

# Expose volume for fixtures
VOLUME /fixtures

# Run mock server
ENTRYPOINT ["/app/mock-ocserv"]
CMD ["-socket", "/var/run/occtl.socket", "-fixtures", "/fixtures/ocserv/occtl"]
