.PHONY: all build test proto-gen lint clean install-tools help

# Variables
BINARY_NAME=quic-backbone
BINARY_SERVER=quic-server
BINARY_CLIENT=quic-client
BINARY_CTL=quic-ctl
GO=go
PROTOC=protoc
PROTOC_GEN_GO=$(shell go env GOPATH)/bin/protoc-gen-go
GOLANGCI_LINT=golangci-lint

# Build directories
BUILD_DIR=bin
PROTO_DIR=pkg/protocol
PROTO_SRC_DIR=specs/001-quic-backbone-network/contracts/protobuf

all: lint test build

## build: Build server, client, and CLI binaries
build:
	@echo "Building binaries..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build -o $(BUILD_DIR)/$(BINARY_SERVER) ./cmd/server
	$(GO) build -o $(BUILD_DIR)/$(BINARY_CLIENT) ./cmd/client
	$(GO) build -o $(BUILD_DIR)/$(BINARY_CTL) ./cmd/ctl
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_SERVER), $(BUILD_DIR)/$(BINARY_CLIENT), $(BUILD_DIR)/$(BINARY_CTL)"

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
	./$(BUILD_DIR)/$(BINARY_SERVER)

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
