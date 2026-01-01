# Dev Container for QUIC Flow - openEuler 22.03 LTS
FROM openeuler/openeuler:22.03-lts

# Install basic tools
RUN dnf install -y \
        gcc \
        make \
        git \
        curl \
        wget \
        unzip \
        xz \
        net-tools \
        iputils \
        jq \
        tree \
        htop \
        vim \
        sudo \
        openssh-clients \
        which \
    && dnf clean all

# Install Go 1.23
ENV GO_VERSION=1.23.4
RUN wget -q https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz \
    && tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz \
    && rm go${GO_VERSION}.linux-amd64.tar.gz

ENV PATH=$PATH:/usr/local/go/bin:/root/go/bin
ENV GOPATH=/root/go
ENV GOPROXY=https://goproxy.cn,direct
ENV GO111MODULE=on

# Install protobuf compiler
ENV PROTOC_VERSION=25.1
RUN wget -q https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-linux-x86_64.zip \
    && unzip -q protoc-${PROTOC_VERSION}-linux-x86_64.zip -d /usr/local \
    && rm protoc-${PROTOC_VERSION}-linux-x86_64.zip

# Install Go tools
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest \
    && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest \
    && go install github.com/go-delve/delve/cmd/dlv@latest

# Install Node.js 20
ENV NODE_VERSION=20.19.6
RUN curl -fsSL https://nodejs.org/dist/v${NODE_VERSION}/node-v${NODE_VERSION}-linux-x64.tar.xz | tar -xJ -C /usr/local --strip-components=1 \
    && npm install -g npm@latest

# Create vscode user for devcontainer
RUN useradd -m -s /bin/bash vscode \
    && echo "vscode ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers

# Set PATH for vscode user
RUN echo 'export PATH=$PATH:/usr/local/go/bin:/home/vscode/go/bin' >> /home/vscode/.bashrc \
    && echo 'export GOPATH=/home/vscode/go' >> /home/vscode/.bashrc \
    && echo 'export GOPROXY=https://goproxy.cn,direct' >> /home/vscode/.bashrc

WORKDIR /workspace
