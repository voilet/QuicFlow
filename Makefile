.PHONY: all build build-loadtest build-linux-amd64 test proto-gen lint clean install-tools help

# Variables
BINARY_NAME=quic-backbone
BINARY_SERVER=quic-server
BINARY_CLIENT=quic-client
BINARY_CTL=quic-ctl
BINARY_LOADTEST=quic-loadtest
GO=go
PROTOC=protoc
PROTOC_GEN_GO=$(shell go env GOPATH)/bin/protoc-gen-go
GOLANGCI_LINT=golangci-lint

# Build directories
BUILD_DIR=bin
PROTO_DIR=pkg/protocol
PROTO_SRC_DIR=specs/001-quic-backbone-network/contracts/protobuf

# Version info (injected at compile time)
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date -u '+%Y-%m-%d_%H:%M:%S')
VERSION_PKG = github.com/voilet/quic-flow/pkg/version
LDFLAGS = -X $(VERSION_PKG).Version=$(VERSION) \
          -X $(VERSION_PKG).GitCommit=$(GIT_COMMIT) \
          -X $(VERSION_PKG).BuildTime=$(BUILD_TIME)

all: lint test build

## build: Build server, client, CLI, and loadtest binaries
build:
	@echo "Building binaries..."
	@echo "Version: $(VERSION), Commit: $(GIT_COMMIT), Time: $(BUILD_TIME)"
	@mkdir -p $(BUILD_DIR)
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_SERVER) ./cmd/server
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_CLIENT) ./cmd/client
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_CTL) ./cmd/ctl
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_LOADTEST) ./cmd/loadtest
	@echo "Build complete: $(BUILD_DIR)/"

## build-loadtest: Build loadtest tool only
build-loadtest:
	@echo "Building loadtest tool..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_LOADTEST) ./cmd/loadtest
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_LOADTEST)"

## build-linux-amd64: Build for Linux x86_64
build-linux-amd64:
	@echo "Building for Linux x86_64..."
	@echo "Version: $(VERSION), Commit: $(GIT_COMMIT), Time: $(BUILD_TIME)"
	@mkdir -p $(BUILD_DIR)/linux-amd64
	GOOS=linux GOARCH=amd64 $(GO) build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/linux-amd64/$(BINARY_SERVER) ./cmd/server
	GOOS=linux GOARCH=amd64 $(GO) build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/linux-amd64/$(BINARY_CLIENT) ./cmd/client
	GOOS=linux GOARCH=amd64 $(GO) build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/linux-amd64/$(BINARY_CTL) ./cmd/ctl
	GOOS=linux GOARCH=amd64 $(GO) build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/linux-amd64/$(BINARY_LOADTEST) ./cmd/loadtest
	@echo "Build complete: $(BUILD_DIR)/linux-amd64/"

## test: Run tests
test:
	@echo "Running tests..."
	$(GO) test -v -race -coverprofile=coverage.txt -covermode=atomic ./...
	@echo "Tests complete"

## test-unit: Run unit tests only
test-unit:
	@echo "Running unit tests..."
	$(GO) test -v -race ./tests/unit/...

## test-integration: Run integration tests only
test-integration:
	@echo "Running integration tests..."
	$(GO) test -v -race ./tests/integration/...

## benchmark: Run performance benchmarks
benchmark:
	@echo "Running benchmarks..."
	$(GO) test -bench=. -benchmem ./tests/benchmark/...

## proto-gen: Generate Go code from protobuf definitions
proto-gen:
	@echo "Generating protobuf code..."
	@mkdir -p $(PROTO_DIR)
	$(PROTOC) --go_out=. --go_opt=paths=source_relative \
		$(PROTO_SRC_DIR)/frame.proto \
		$(PROTO_SRC_DIR)/message.proto \
		$(PROTO_SRC_DIR)/types.proto
	@echo "Protobuf generation complete"

## lint: Run linters
lint:
	@echo "Running linters..."
	$(GOLANGCI_LINT) run ./...
	@echo "Linting complete"

## fmt: Format code
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...
	goimports -w -local github.com/voilet/QuicFlow .

## clean: Remove build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.txt coverage.html
	@rm -f *.prof *.pprof
	@echo "Clean complete"

## install-tools: Install development tools
install-tools:
	@echo "Installing tools..."
	$(GO) install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@echo "Tools installed"

## certs: Generate development TLS certificates
certs:
	@echo "Generating development certificates..."
	@./scripts/gen-certs.sh
	@echo "Certificates generated"

## run-server: Run the server
run-server: build
	@echo "Starting server..."
	./$(BUILD_DIR)/$(BINARY_SERVER) --config config/server.yaml

## run-client: Run the client
run-client: build
	@echo "Starting client..."
	./$(BUILD_DIR)/$(BINARY_CLIENT)

## tidy: Tidy go modules
tidy:
	@echo "Tidying go modules..."
	$(GO) mod tidy
	@echo "Tidy complete"

## help: Show this help message
help:
	@echo "Available targets:"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'
