# ============================================================================
# GitHub Actions Self-Hosted Runner for ocserv-agent
# Base: Oracle Linux 10 (full image for GitHub Actions runner compatibility)
# ============================================================================
FROM oraclelinux:10

# ============================================================================
# Enable additional repositories
# ============================================================================
RUN dnf config-manager --set-enabled ol10_addons && \
    dnf config-manager --set-enabled ol10_appstream && \
    dnf config-manager --set-enabled ol10_codeready_builder && \
    dnf config-manager --set-enabled ol10_distro_builder

# ============================================================================
# Install base dependencies + EPEL
# ============================================================================
RUN dnf install -y \
        # Base tools
        curl \
        tar \
        gzip \
        bzip2 \
        xz \
        unzip \
        git \
        jq \
        sudo \
        # Build tools
        gcc \
        gcc-c++ \
        make \
        automake \
        autoconf \
        libtool \
        pkgconfig \
        # Libraries for building
        openssl-devel \
        zlib-devel \
        libffi-devel \
        readline-devel \
        sqlite-devel \
        bzip2-devel \
        xz-devel \
        # Network tools
        iputils \
        ca-certificates \
        # Podman for containerized builds
        podman \
    && dnf clean all

# Install EPEL and update packages
RUN dnf install -y oracle-epel-release-el10 && \
    dnf update -y --allowerasing && \
    dnf clean all

# ============================================================================
# Install Python 3.14.0 from source
# ============================================================================
RUN cd /tmp && \
    curl -fsSL -o Python-3.14.0.tgz https://www.python.org/ftp/python/3.14.0/Python-3.14.0.tgz && \
    tar -xzf Python-3.14.0.tgz && \
    cd Python-3.14.0 && \
    ./configure --enable-optimizations --with-ensurepip=install && \
    make -j$(nproc) && \
    make altinstall && \
    cd /tmp && \
    rm -rf Python-3.14.0* && \
    ln -sf /usr/local/bin/python3.14 /usr/local/bin/python3 && \
    ln -sf /usr/local/bin/pip3.14 /usr/local/bin/pip3

ENV PATH="/usr/local/bin:${PATH}"

# ============================================================================
# Install Poetry 2.2
# ============================================================================
RUN curl -sSL https://install.python-poetry.org | python3 - && \
    ln -s /root/.local/bin/poetry /usr/local/bin/poetry

# ============================================================================
# Install .NET SDK 8.0
# ============================================================================
RUN dnf install -y dotnet-sdk-8.0 && \
    dnf clean all

# ============================================================================
# Install Go 1.25.1
# ============================================================================
RUN curl -fsSL -o go1.25.1.linux-amd64.tar.gz https://go.dev/dl/go1.25.1.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf go1.25.1.linux-amd64.tar.gz && \
    rm go1.25.1.linux-amd64.tar.gz

ENV PATH="/usr/local/go/bin:${PATH}"
ENV GOPATH="/go"

# ============================================================================
# Install protoc (Protocol Buffers compiler)
# ============================================================================
ARG PROTOC_VERSION="33.0"
RUN curl -fsSL -o protoc-${PROTOC_VERSION}-linux-x86_64.zip \
        https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-linux-x86_64.zip && \
    unzip protoc-${PROTOC_VERSION}-linux-x86_64.zip -d /usr/local && \
    rm protoc-${PROTOC_VERSION}-linux-x86_64.zip

# Install protoc-gen-go and protoc-gen-go-grpc
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

ENV PATH="${GOPATH}/bin:${PATH}"

# ============================================================================
# Install Go static analysis and security tools
# ============================================================================
# SAST & Code Quality
RUN go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest && \
    go install honnef.co/go/tools/cmd/staticcheck@latest && \
    go install github.com/securego/gosec/v2/cmd/gosec@latest && \
    go install github.com/kisielk/errcheck@latest && \
    go install github.com/gordonklaus/ineffassign@latest && \
# Vulnerability & Dependency Scanning
    go install golang.org/x/vuln/cmd/govulncheck@latest && \
    go install github.com/google/osv-scanner/v2/cmd/osv-scanner@latest && \
    go install github.com/sonatype-nexus-community/nancy@latest && \
    go install github.com/google/go-licenses@latest

# ============================================================================
# Install mock (for RPM builds)
# ============================================================================
RUN dnf install -y mock && \
    dnf clean all

# ============================================================================
# Install additional build tools
# ============================================================================
RUN dnf install -y \
        # RPM building
        rpm-build \
        rpmdevtools \
        rpmlint \
        # Container tools
        buildah \
        skopeo \
        # Security scanning tools
        trivy \
        # Git tools
        git-lfs \
    && dnf clean all

# ============================================================================
# Create runner user
# ============================================================================
RUN useradd -m -s /bin/bash runner && \
    usermod -aG wheel runner && \
    usermod -aG mock runner && \
    echo "runner ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers

# ============================================================================
# Set working directory
# ============================================================================
WORKDIR /home/runner

# ============================================================================
# Download and install GitHub Actions runner
# ============================================================================
ARG RUNNER_VERSION="2.329.0"
RUN curl -o actions-runner-linux-x64-${RUNNER_VERSION}.tar.gz \
    -L https://github.com/actions/runner/releases/download/v${RUNNER_VERSION}/actions-runner-linux-x64-${RUNNER_VERSION}.tar.gz && \
    tar xzf actions-runner-linux-x64-${RUNNER_VERSION}.tar.gz && \
    rm actions-runner-linux-x64-${RUNNER_VERSION}.tar.gz

# Install runner dependencies
RUN cd /home/runner && ./bin/installdependencies.sh

# Install Docker CE (actual Docker instead of podman)
RUN dnf config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo && \
    dnf install -y docker-ce docker-ce-cli containerd.io && \
    dnf clean all

# ============================================================================
# Install security scanning and supply chain tools (OSSF/SLSA/SBOM)
# ============================================================================

# Install Syft (SBOM generation tool from Anchore)
ARG SYFT_VERSION="v1.34.2"
RUN curl -sSfL https://raw.githubusercontent.com/anchore/syft/main/install.sh | sh -s -- -b /usr/local/bin ${SYFT_VERSION}

# Install Grype (vulnerability scanner for containers and filesystems)
ARG GRYPE_VERSION="v0.101.1"
RUN curl -sSfL https://raw.githubusercontent.com/anchore/grype/main/install.sh | sh -s -- -b /usr/local/bin ${GRYPE_VERSION}

# Install Cosign (container signing and verification from Sigstore)
ARG COSIGN_VERSION="v3.0.2"
RUN curl -fsSL -o /usr/local/bin/cosign \
        https://github.com/sigstore/cosign/releases/download/${COSIGN_VERSION}/cosign-linux-amd64 && \
    chmod +x /usr/local/bin/cosign

# Install Gitleaks (secrets detection - fast and lightweight)
ARG GITLEAKS_VERSION="8.28.0"
RUN curl -fsSL -o gitleaks.tar.gz \
        https://github.com/gitleaks/gitleaks/releases/download/v${GITLEAKS_VERSION}/gitleaks_${GITLEAKS_VERSION}_linux_x64.tar.gz && \
    tar -xzf gitleaks.tar.gz -C /usr/local/bin gitleaks && \
    rm gitleaks.tar.gz && \
    chmod +x /usr/local/bin/gitleaks

# Install TruffleHog (secrets detection - comprehensive with verification)
ARG TRUFFLEHOG_VERSION="3.90.3"
RUN curl -fsSL -o trufflehog.tar.gz \
        https://github.com/trufflesecurity/trufflehog/releases/download/v${TRUFFLEHOG_VERSION}/trufflehog_${TRUFFLEHOG_VERSION}_linux_amd64.tar.gz && \
    tar -xzf trufflehog.tar.gz -C /usr/local/bin trufflehog && \
    rm trufflehog.tar.gz && \
    chmod +x /usr/local/bin/trufflehog

# Install OSSF Scorecard v5.3.0 (project security health metrics)
ARG SCORECARD_VERSION="5.3.0"
RUN curl -fsSL -o scorecard.tar.gz \
        https://github.com/ossf/scorecard/releases/download/v${SCORECARD_VERSION}/scorecard_${SCORECARD_VERSION}_linux_amd64.tar.gz && \
    tar -xzf scorecard.tar.gz -C /usr/local/bin scorecard && \
    rm scorecard.tar.gz && \
    chmod +x /usr/local/bin/scorecard

# Install Semgrep (multi-language SAST tool)
RUN pip3 install --no-cache-dir semgrep

# Fix GOPATH permissions for go install commands
RUN mkdir -p /go/pkg/mod/cache && \
    chown -R runner:runner /go && \
    chmod -R 755 /go

# Create work directory with correct permissions
RUN mkdir -p /home/runner/_work && \
    chown -R runner:runner /home/runner && \
    chmod -R 755 /home/runner/_work

# ============================================================================
# Configure runner environment
# ============================================================================
USER runner

# Ensure Poetry is in runner's PATH
RUN echo 'export PATH="/root/.local/bin:$PATH"' >> ~/.bashrc

# ============================================================================
# Copy and configure entrypoint
# ============================================================================
USER root
COPY entrypoint.sh /home/runner/entrypoint.sh
RUN chmod +x /home/runner/entrypoint.sh

USER runner

ENTRYPOINT ["/home/runner/entrypoint.sh"]
