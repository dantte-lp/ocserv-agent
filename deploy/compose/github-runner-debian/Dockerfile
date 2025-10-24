# ============================================================================
# GitHub Actions Self-Hosted Runner for ocserv-agent
# Base: Python 3.14 on Debian Trixie (Python уже установлен и оптимизирован)
# ============================================================================
FROM python:3.14-trixie

# ============================================================================
# Install base dependencies and Docker
# ============================================================================
RUN apt-get update && apt-get install -y \
        # Base tools
        curl \
        git \
        jq \
        sudo \
        ca-certificates \
        wget \
        zip \
        unzip \
        rsync \
        # Build tools
        build-essential \
        cmake \
        clang \
        llvm \
        # Autotools (для ocserv и других C проектов)
        autoconf \
        automake \
        libtool \
        m4 \
        # Дополнительные C/C++ инструменты
        gdb \
        valgrind \
        strace \
        # Docker for container actions
        docker.io \
        # Network tools
        iputils-ping \
        openssh-client \
        # Text editors
        vim \
        nano \
        # Development libraries
        libssl-dev \
        libcurl4-openssl-dev \
        libgnutls28-dev \
        pkg-config \
        # Библиотеки для ocserv и VPN разработки
        libreadline-dev \
        libpam0g-dev \
        liblz4-dev \
        libseccomp-dev \
        libwrap0-dev \
        libprotobuf-c-dev \
        libev-dev \
        libhttp-parser-dev \
        libtalloc-dev \
        # Linters
        shellcheck \
    && rm -rf /var/lib/apt/lists/*

# ============================================================================
# Install Poetry 2.2
# ============================================================================
RUN curl -sSL https://install.python-poetry.org | python3 - && \
    ln -s /root/.local/bin/poetry /usr/local/bin/poetry

# ============================================================================
# Install Node.js 25 (from NodeSource)
# ============================================================================
RUN curl -fsSL https://deb.nodesource.com/setup_25.x | bash - && \
    apt-get install -y nodejs && \
    npm install -g npm@latest yarn pnpm && \
    rm -rf /var/lib/apt/lists/*

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
# Install .NET SDK 8.0
# ============================================================================
RUN curl -fsSL https://packages.microsoft.com/keys/microsoft.asc | gpg --dearmor -o /usr/share/keyrings/microsoft-prod.gpg && \
    echo "deb [arch=amd64 signed-by=/usr/share/keyrings/microsoft-prod.gpg] https://packages.microsoft.com/debian/12/prod bookworm main" > /etc/apt/sources.list.d/microsoft-prod.list && \
    apt-get update && \
    apt-get install -y dotnet-sdk-8.0 && \
    rm -rf /var/lib/apt/lists/*

# ============================================================================
# Install GraalVM for JDK 25 (includes JDK + Native Image support)
# https://www.graalvm.org/latest/getting-started/linux/
# ============================================================================
ARG GRAALVM_VERSION="25.0.1"
RUN curl -fsSL -o graalvm.tar.gz \
        https://download.oracle.com/graalvm/25/latest/graalvm-jdk-25_linux-x64_bin.tar.gz && \
    mkdir -p /opt/graalvm && \
    tar -xzf graalvm.tar.gz -C /opt/graalvm --strip-components=1 && \
    rm graalvm.tar.gz

ENV GRAALVM_HOME=/opt/graalvm
ENV JAVA_HOME=/opt/graalvm
ENV PATH="${GRAALVM_HOME}/bin:${PATH}"

# Note: Native Image is included by default in GraalVM 25+
# https://www.graalvm.org/latest/reference-manual/native-image/

# Install Maven and Gradle
RUN apt-get update && apt-get install -y \
        maven \
        gradle \
    && rm -rf /var/lib/apt/lists/*

# ============================================================================
# Install static analysis and security tools
# ============================================================================

# Go tools (install globally in GOPATH)
RUN go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest && \
    go install honnef.co/go/tools/cmd/staticcheck@latest

# Python tools (CLI tools installed globally)
RUN pip3 install --no-cache-dir \
        pylint \
        flake8 \
        black \
        mypy \
        bandit \
        safety \
        isort \
        ruff

# C/C++ tools
RUN apt-get update && apt-get install -y \
        cppcheck \
        clang-tidy \
    && rm -rf /var/lib/apt/lists/*

# JavaScript/TypeScript tools (global npm packages)
RUN npm install -g \
    eslint \
    prettier \
    typescript \
    @typescript-eslint/parser \
    @typescript-eslint/eslint-plugin

# Java tools (download standalone versions)
ARG CHECKSTYLE_VERSION="12.1.0"
ARG PMD_VERSION="7.17.0"
ARG SPOTBUGS_VERSION="4.9.8"

RUN mkdir -p /opt/java-tools && \
    # Checkstyle
    curl -fsSL -o /opt/java-tools/checkstyle.jar \
        https://github.com/checkstyle/checkstyle/releases/download/checkstyle-${CHECKSTYLE_VERSION}/checkstyle-${CHECKSTYLE_VERSION}-all.jar && \
    # PMD
    curl -fsSL -o pmd.zip \
        https://github.com/pmd/pmd/releases/download/pmd_releases%2F${PMD_VERSION}/pmd-dist-${PMD_VERSION}-bin.zip && \
    unzip pmd.zip -d /opt/java-tools && \
    mv /opt/java-tools/pmd-bin-${PMD_VERSION} /opt/java-tools/pmd && \
    rm pmd.zip && \
    # SpotBugs
    curl -fsSL -o spotbugs.tgz \
        https://github.com/spotbugs/spotbugs/releases/download/${SPOTBUGS_VERSION}/spotbugs-${SPOTBUGS_VERSION}.tgz && \
    tar -xzf spotbugs.tgz -C /opt/java-tools && \
    mv /opt/java-tools/spotbugs-${SPOTBUGS_VERSION} /opt/java-tools/spotbugs && \
    rm spotbugs.tgz && \
    # Create wrapper scripts
    echo '#!/bin/bash\njava -jar /opt/java-tools/checkstyle.jar "$@"' > /usr/local/bin/checkstyle && \
    chmod +x /usr/local/bin/checkstyle

ENV PATH="/opt/java-tools/pmd/bin:/opt/java-tools/spotbugs/bin:${PATH}"

# Universal tools
RUN apt-get update && apt-get install -y \
        yamllint \
    && rm -rf /var/lib/apt/lists/*

# Install hadolint (Dockerfile linter)
RUN curl -fsSL -o /usr/local/bin/hadolint \
    https://github.com/hadolint/hadolint/releases/download/v2.14.0/hadolint-Linux-x86_64 && \
    chmod +x /usr/local/bin/hadolint

# Install trivy (container/dependency security scanner)
RUN curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | sh -s -- -b /usr/local/bin

# Install markdownlint-cli
RUN npm install -g markdownlint-cli

# ============================================================================
# Create runner user
# ============================================================================
RUN useradd -m -s /bin/bash runner && \
    usermod -aG sudo runner && \
    usermod -aG docker runner && \
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
