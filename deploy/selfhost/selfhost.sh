#!/bin/bash
# =============================================================================
# AgentsMesh Self-Hosted Deployment Script
# =============================================================================
#
# One-command setup for self-hosted AgentsMesh using Docker Hub images.
#
# Usage:
#   ./selfhost.sh --host 192.168.1.100
#   ./selfhost.sh --host agentsmesh.example.com --http-port 8080
#   ./selfhost.sh --host 192.168.1.100 --version sha-abc1234
#   ./selfhost.sh --clean
#
# =============================================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="${SCRIPT_DIR}/.env"
SSL_DIR="${SCRIPT_DIR}/ssl"
SEED_FILE="${SCRIPT_DIR}/seed/seed.sql"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

info()    { echo -e "${BLUE}[INFO]${NC} $1"; }
success() { echo -e "${GREEN}[ OK ]${NC} $1"; }
warn()    { echo -e "${YELLOW}[WARN]${NC} $1"; }
error()   { echo -e "${RED}[ERROR]${NC} $1"; }

# Default values
SERVER_HOST=""
HTTP_PORT=80
GRPC_PORT=9443
VERSION="latest"
CLEAN=false

# =============================================================================
# Parse arguments
# =============================================================================
while [[ $# -gt 0 ]]; do
    case $1 in
        --host)       SERVER_HOST="$2"; shift 2 ;;
        --http-port)  HTTP_PORT="$2";   shift 2 ;;
        --grpc-port)  GRPC_PORT="$2";   shift 2 ;;
        --version)    VERSION="$2";     shift 2 ;;
        --clean)      CLEAN=true;       shift   ;;
        -h|--help)
            echo "Usage: $0 --host <IP_OR_DOMAIN> [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --host        Server IP or domain name (required)"
            echo "  --http-port   HTTP port (default: 80)"
            echo "  --grpc-port   gRPC port for Runner connections (default: 9443)"
            echo "  --version     Docker image tag (default: latest)"
            echo "  --clean       Stop services and remove all data"
            echo ""
            echo "Examples:"
            echo "  $0 --host 192.168.1.100"
            echo "  $0 --host agentsmesh.example.com --http-port 8080"
            echo "  $0 --host 10.0.0.5 --version sha-abc1234"
            exit 0
            ;;
        *) error "Unknown option: $1"; exit 1 ;;
    esac
done

cd "${SCRIPT_DIR}"

# =============================================================================
# Clean mode
# =============================================================================
if [ "${CLEAN}" = true ]; then
    warn "Stopping services and removing all data..."
    docker compose down -v 2>/dev/null || true
    rm -rf "${SSL_DIR}" "${ENV_FILE}"
    success "Cleanup complete."
    exit 0
fi

# Validate
if [ -z "${SERVER_HOST}" ]; then
    error "--host is required"
    echo "Usage: $0 --host <IP_OR_DOMAIN>"
    echo "Run '$0 --help' for more options."
    exit 1
fi

echo ""
echo "=============================================="
echo "  AgentsMesh Self-Hosted Deployment"
echo "=============================================="
echo ""
echo "  Host:    ${SERVER_HOST}"
echo "  HTTP:    ${HTTP_PORT}"
echo "  gRPC:    ${GRPC_PORT}"
echo "  Version: ${VERSION}"
echo ""

# =============================================================================
# Step 1: Check prerequisites
# =============================================================================
info "[1/6] Checking prerequisites..."

if ! command -v docker &> /dev/null; then
    error "Docker is not installed. Install: https://docs.docker.com/engine/install/"
    exit 1
fi

if ! docker info &> /dev/null; then
    error "Docker daemon is not running."
    exit 1
fi

if ! command -v openssl &> /dev/null; then
    error "OpenSSL is not installed."
    exit 1
fi

success "Docker $(docker version --format '{{.Server.Version}}' 2>/dev/null || echo 'OK')"

# =============================================================================
# Step 2: Generate .env
# =============================================================================
info "[2/6] Generating configuration..."

generate_secret() {
    openssl rand -base64 32 | tr -dc 'a-zA-Z0-9' | head -c 32
}

if [ -f "${ENV_FILE}" ]; then
    warn "Found existing .env — preserving secrets (updating host/ports)"
    # shellcheck disable=SC1090
    source "${ENV_FILE}"
else
    info "Generating new secrets..."
    DB_PASSWORD=$(generate_secret)
    JWT_SECRET=$(generate_secret)
    INTERNAL_API_SECRET=$(generate_secret)
    MINIO_ROOT_PASSWORD=$(generate_secret)
fi

cat > "${ENV_FILE}" << EOF
# AgentsMesh Self-Hosted Configuration
# Generated: $(date -Iseconds 2>/dev/null || date)

VERSION=${VERSION}
SERVER_HOST=${SERVER_HOST}
HTTP_PORT=${HTTP_PORT}
GRPC_PORT=${GRPC_PORT}

COMPOSE_PROJECT_NAME=agentsmesh

DB_PASSWORD=${DB_PASSWORD}
JWT_SECRET=${JWT_SECRET}
INTERNAL_API_SECRET=${INTERNAL_API_SECRET}
MINIO_ROOT_PASSWORD=${MINIO_ROOT_PASSWORD}

MINIO_API_PORT=9000
MINIO_CONSOLE_PORT=9001
EMAIL_PROVIDER=console
EOF

success "Configuration saved to .env"

# =============================================================================
# Step 3: Generate SSL certificates (for gRPC mTLS)
# =============================================================================
info "[3/6] Generating SSL certificates..."

if [ -f "${SSL_DIR}/ca.crt" ] && [ -f "${SSL_DIR}/server.crt" ]; then
    success "Certificates already exist (delete ssl/ to regenerate)"
else
    mkdir -p "${SSL_DIR}"

    # CA key + cert (10 years)
    openssl ecparam -name prime256v1 -genkey -noout -out "${SSL_DIR}/ca.key" 2>/dev/null
    openssl req -new -x509 -days 3650 -key "${SSL_DIR}/ca.key" -out "${SSL_DIR}/ca.crt" \
        -subj "/CN=AgentsMesh CA/O=AgentsMesh" 2>/dev/null

    # Server key + CSR
    openssl ecparam -name prime256v1 -genkey -noout -out "${SSL_DIR}/server.key" 2>/dev/null
    openssl req -new -key "${SSL_DIR}/server.key" -out "${SSL_DIR}/server.csr" \
        -subj "/CN=agentsmesh-backend/O=AgentsMesh" 2>/dev/null

    # SAN config
    cat > "${SSL_DIR}/server_ext.cnf" << EXTEOF
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = localhost
DNS.2 = backend
DNS.3 = traefik
IP.1 = 127.0.0.1
IP.2 = ::1
EXTEOF

    # Add server host to SAN (IP or domain)
    if [[ "${SERVER_HOST}" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        echo "IP.3 = ${SERVER_HOST}" >> "${SSL_DIR}/server_ext.cnf"
    else
        echo "DNS.4 = ${SERVER_HOST}" >> "${SSL_DIR}/server_ext.cnf"
    fi

    # Sign server cert (1 year)
    openssl x509 -req -days 365 -in "${SSL_DIR}/server.csr" \
        -CA "${SSL_DIR}/ca.crt" -CAkey "${SSL_DIR}/ca.key" -CAcreateserial \
        -out "${SSL_DIR}/server.crt" -extfile "${SSL_DIR}/server_ext.cnf" 2>/dev/null

    # Cleanup temp files
    rm -f "${SSL_DIR}/server.csr" "${SSL_DIR}/server_ext.cnf" "${SSL_DIR}/ca.srl"
    chmod 600 "${SSL_DIR}/ca.key" "${SSL_DIR}/server.key"
    chmod 644 "${SSL_DIR}/ca.crt" "${SSL_DIR}/server.crt"

    success "Certificates generated in ssl/"
fi

# =============================================================================
# Step 4: Pull images and start services
# =============================================================================
info "[4/6] Pulling images and starting services..."

docker compose pull --quiet
docker compose up -d

info "Waiting for backend to be ready..."
TIMEOUT=120
ELAPSED=0
while [ $ELAPSED -lt $TIMEOUT ]; do
    if docker compose exec -T backend wget --no-verbose --tries=1 --spider http://localhost:8080/health 2>/dev/null; then
        break
    fi
    sleep 5
    ELAPSED=$((ELAPSED + 5))
done

if [ $ELAPSED -ge $TIMEOUT ]; then
    warn "Backend health check timed out (${TIMEOUT}s). Check: docker compose logs backend"
else
    success "All services running"
fi

# =============================================================================
# Step 5: Run database migrations
# =============================================================================
info "[5/6] Running database migrations..."

DB_URL="postgres://agentsmesh:${DB_PASSWORD}@postgres:5432/agentsmesh?sslmode=disable"
docker compose exec -T backend migrate -path /app/migrations -database "${DB_URL}" up 2>&1 | tail -5

success "Migrations complete"

# =============================================================================
# Step 6: Import seed data
# =============================================================================
info "[6/6] Importing seed data..."

docker compose exec -T postgres psql -U agentsmesh -d agentsmesh < "${SEED_FILE}" 2>&1 | grep -v "^$" | tail -5

success "Seed data imported"

# =============================================================================
# Done
# =============================================================================
echo ""
echo "=============================================="
echo -e "  ${GREEN}Installation Complete!${NC}"
echo "=============================================="
echo ""
echo "  Web Console:   http://${SERVER_HOST}:${HTTP_PORT}"
echo "  MinIO Console: http://${SERVER_HOST}:9001"
echo ""
echo "  Admin Account:"
echo "    Email:    admin@localhost.local"
echo "    Password: Admin@123"
echo ""
echo "  Register a Runner:"
echo "    curl -fsSL https://agentsmesh.ai/install.sh | sh"
echo "    agentsmesh-runner register \\"
echo "      --server http://${SERVER_HOST}:${HTTP_PORT} \\"
echo "      --token selfhost-runner-token"
echo "    agentsmesh-runner run"
echo ""
echo -e "  ${YELLOW}Change the admin password after first login!${NC}"
echo ""
echo "  Useful commands:"
echo "    docker compose logs -f       # View logs"
echo "    docker compose ps            # Service status"
echo "    docker compose down          # Stop services"
echo "    ./selfhost.sh --clean        # Remove everything"
echo ""
