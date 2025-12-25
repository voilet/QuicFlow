#!/bin/bash
# Generate self-signed TLS certificates for development

set -e

CERT_DIR="certs"
DAYS=365

# Create certs directory if it doesn't exist
mkdir -p "$CERT_DIR"

echo "Generating self-signed TLS certificates for development..."
echo "⚠️  WARNING: These certificates are for development only!"
echo ""

# Generate server certificate
echo "Generating server certificate..."
openssl req -x509 -newkey rsa:4096 \
    -keyout "$CERT_DIR/server-key.pem" \
    -out "$CERT_DIR/server-cert.pem" \
    -days $DAYS -nodes \
    -subj "/CN=localhost/O=QUIC Backbone/OU=Development" \
    -addext "subjectAltName=DNS:localhost,IP:127.0.0.1"

echo "✓ Server certificate generated: $CERT_DIR/server-cert.pem"
echo "✓ Server key generated: $CERT_DIR/server-key.pem"
echo ""

# Generate client certificate (optional, for mutual TLS)
echo "Generating client certificate (optional)..."
openssl req -x509 -newkey rsa:4096 \
    -keyout "$CERT_DIR/client-key.pem" \
    -out "$CERT_DIR/client-cert.pem" \
    -days $DAYS -nodes \
    -subj "/CN=client/O=QUIC Backbone/OU=Development"

echo "✓ Client certificate generated: $CERT_DIR/client-cert.pem"
echo "✓ Client key generated: $CERT_DIR/client-key.pem"
echo ""

# Set permissions
chmod 600 "$CERT_DIR"/*.pem

echo "✅ Certificate generation complete!"
echo ""
echo "Certificate files are in the $CERT_DIR/ directory:"
ls -lh "$CERT_DIR/"
echo ""
echo "Note: Add '$CERT_DIR/' to your .gitignore to avoid committing secrets"
