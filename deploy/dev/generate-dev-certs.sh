#!/bin/bash
# Generate self-signed certificates for development gRPC + mTLS
#
# This script creates:
# - ca.crt/ca.key: Self-signed CA certificate
# - server.crt/server.key: Server certificate signed by CA
#
# Usage:
#   ./generate-dev-certs.sh
#
# Output directory: ./ssl/

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SSL_DIR="${SCRIPT_DIR}/ssl"

# Create SSL directory
mkdir -p "${SSL_DIR}"

echo "Generating development certificates in ${SSL_DIR}..."

# Check if certificates already exist
if [ -f "${SSL_DIR}/ca.crt" ] && [ -f "${SSL_DIR}/server.crt" ]; then
    echo "Certificates already exist. To regenerate, delete ${SSL_DIR} first."
    exit 0
fi

# Generate CA private key (ECDSA P-256, PEM format)
echo "Generating CA private key..."
openssl genpkey -algorithm EC -pkeyopt ec_paramgen_curve:prime256v1 -out "${SSL_DIR}/ca.key"

# Generate CA certificate (10 years validity)
echo "Generating CA certificate..."
openssl req -new -x509 -days 3650 -key "${SSL_DIR}/ca.key" -out "${SSL_DIR}/ca.crt" \
    -subj "/CN=AgentsMesh Dev CA/O=AgentsMesh/OU=Development"

# Generate server private key (ECDSA P-256, PEM format)
echo "Generating server private key..."
openssl genpkey -algorithm EC -pkeyopt ec_paramgen_curve:prime256v1 -out "${SSL_DIR}/server.key"

# Generate server CSR
echo "Generating server CSR..."
openssl req -new -key "${SSL_DIR}/server.key" -out "${SSL_DIR}/server.csr" \
    -subj "/CN=localhost/O=AgentsMesh/OU=Backend"

# Create server certificate extensions config
cat > "${SSL_DIR}/server_ext.cnf" << EOF
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = localhost
DNS.2 = backend
DNS.3 = nginx
DNS.4 = *.local
IP.1 = 127.0.0.1
IP.2 = ::1
EOF

# Sign server certificate with CA (1 year validity)
echo "Signing server certificate with CA..."
openssl x509 -req -days 365 -in "${SSL_DIR}/server.csr" \
    -CA "${SSL_DIR}/ca.crt" -CAkey "${SSL_DIR}/ca.key" -CAcreateserial \
    -out "${SSL_DIR}/server.crt" -extfile "${SSL_DIR}/server_ext.cnf"

# Clean up temporary files
rm -f "${SSL_DIR}/server.csr" "${SSL_DIR}/server_ext.cnf" "${SSL_DIR}/ca.srl"

# Set permissions
chmod 600 "${SSL_DIR}/ca.key" "${SSL_DIR}/server.key"
chmod 644 "${SSL_DIR}/ca.crt" "${SSL_DIR}/server.crt"

echo ""
echo "✅ Development certificates generated successfully!"
echo ""
echo "Files created:"
echo "  ${SSL_DIR}/ca.crt     - CA certificate (Backend + Nginx + Runner)"
echo "  ${SSL_DIR}/ca.key     - CA private key (Backend only, for signing)"
echo "  ${SSL_DIR}/server.crt - Server certificate (Nginx TLS termination)"
echo "  ${SSL_DIR}/server.key - Server private key (Nginx TLS termination)"
echo ""
echo "gRPC + mTLS is auto-enabled when CA files are configured."
echo "The docker-compose.yml is already configured to mount these certificates."
echo ""
echo "To use:"
echo "  docker compose up -d"
echo "  # Runners can connect via grpcs://localhost:9443"
