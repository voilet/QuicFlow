# Dev Container for QUIC Flow
FROM mcr.microsoft.com/devcontainers/go:1.23-bookworm

# Install protobuf compiler and tools
RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
    && apt-get -y install --no-install-recommends \
        protobuf-compiler \
        net-tools \
        iputils-ping \
        curl \
        wget \
        jq \
        tree \
        htop \
        vim \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Install Go tools
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest \
    && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest \
    && go install github.com/go-delve/delve/cmd/dlv@latest

# Set Go environment
ENV GOPROXY=https://goproxy.cn,direct
ENV GO111MODULE=on
