#!/bin/bash
set -e

echo "=== QUIC Flow Dev Container Setup ==="

# Go dependencies
echo "Installing Go dependencies..."
go mod download

# Generate certificates if not exist
if [ ! -f "certs/server-cert.pem" ]; then
    echo "Generating TLS certificates..."
    mkdir -p certs
    if [ -f "scripts/gen-certs.sh" ]; then
        bash scripts/gen-certs.sh
    else
        # Generate self-signed certs
        openssl req -x509 -newkey rsa:4096 -keyout certs/server-key.pem -out certs/server-cert.pem -days 365 -nodes -subj "/CN=localhost"
    fi
fi

# Generate protobuf files
echo "Generating protobuf files..."
if [ -d "pkg/protocol" ] && ls pkg/protocol/*.proto 1> /dev/null 2>&1; then
    protoc --go_out=. --go_opt=paths=source_relative pkg/protocol/*.proto
fi

# Web dependencies
if [ -d "web" ]; then
    echo "Installing web dependencies..."
    cd web
    npm install
    cd ..
fi

# Build binaries
echo "Building binaries..."
go build -o bin/quic-server ./cmd/server
go build -o bin/quic-client ./cmd/client

echo "=== Setup Complete ==="
echo ""
echo "Quick start:"
echo "  Server: ./bin/quic-server"
echo "  Client: ./bin/quic-client"
echo "  Web UI: cd web && npm run dev"
