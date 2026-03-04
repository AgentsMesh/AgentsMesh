#!/bin/bash
# =============================================================================
# AgentsMesh OnPremise Installation Script
# =============================================================================
#
# One-click installation for on-premise deployment.
#
# Usage:
#   ./install.sh --ip 192.168.1.100
#   ./install.sh --ip 192.168.1.100 --http-port 8080 --grpc-port 9443
#
# Options:
#   --ip          Server IP address (required)
#   --http-port   HTTP port (default: 80)
#   --grpc-port   gRPC port (default: 9443)
#   --skip-load   Skip loading images (use existing images)
#   --clean       Clean up everything and start fresh
#
# =============================================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEPLOY_DIR="${SCRIPT_DIR}/.."

# Default values
SERVER_IP=""
HTTP_PORT=80
GRPC_PORT=9443
VERSION="v1.0.0"
SKIP_LOAD=false
CLEAN=false

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --ip)
            SERVER_IP="$2"
            shift 2
            ;;
        --http-port)
            HTTP_PORT="$2"
            shift 2
            ;;
        --grpc-port)
            GRPC_PORT="$2"
            shift 2
            ;;
        --version)
            VERSION="$2"
            shift 2
            ;;
        --skip-load)
            SKIP_LOAD=true
            shift
            ;;
        --clean)
            CLEAN=true
            shift
            ;;
        -h|--help)
            echo "Usage: $0 --ip <SERVER_IP> [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --ip          Server IP address (required)"
            echo "  --http-port   HTTP port (default: 80)"
            echo "  --grpc-port   gRPC port (default: 9443)"
            echo "  --version     Image version tag (default: v1.0.0)"
            echo "  --skip-load   Skip loading images"
            echo "  --clean       Clean up and start fresh"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Validate required arguments
if [ -z "${SERVER_IP}" ]; then
    echo "Error: --ip is required"
    echo "Usage: $0 --ip <SERVER_IP>"
    exit 1
fi

# Validate IP format (basic check)
if ! [[ "${SERVER_IP}" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo "Error: Invalid IP address format: ${SERVER_IP}"
    exit 1
fi

# Detect Docker socket path (macOS vs Linux)
detect_docker_socket() {
    if [ -S "/var/run/docker.sock" ]; then
        echo "/var/run/docker.sock"
    elif [ -S "${HOME}/.docker/run/docker.sock" ]; then
        echo "${HOME}/.docker/run/docker.sock"
    else
        echo "/var/run/docker.sock"  # fallback
    fi
}
DOCKER_SOCKET=$(detect_docker_socket)

echo "=============================================="
echo "AgentsMesh OnPremise Installation"
echo "=============================================="
echo ""
echo "Configuration:"
echo "  Server IP:     ${SERVER_IP}"
echo "  HTTP Port:     ${HTTP_PORT}"
echo "  gRPC Port:     ${GRPC_PORT}"
echo "  Version:       ${VERSION}"
echo "  Docker Socket: ${DOCKER_SOCKET}"
echo ""

# Change to deploy directory
cd "${DEPLOY_DIR}"

# =============================================================================
# Step 0: Clean (if requested)
# =============================================================================
if [ "${CLEAN}" = true ]; then
    echo "[Step 0/7] Cleaning up existing deployment..."
    docker compose down -v 2>/dev/null || true
    rm -rf ssl .env
    echo "  Cleaned."
    echo ""
fi

# =============================================================================
# Step 1: Check Docker
# =============================================================================
echo "[Step 1/7] Checking Docker environment..."

if ! command -v docker &> /dev/null; then
    echo "  Error: Docker is not installed."
    echo "  Please install Docker first: https://docs.docker.com/engine/install/"
    exit 1
fi

if ! docker info &> /dev/null; then
    echo "  Error: Docker daemon is not running."
    echo "  Please start Docker and try again."
    exit 1
fi

DOCKER_VERSION=$(docker version --format '{{.Server.Version}}' 2>/dev/null || echo "unknown")
echo "  Docker version: ${DOCKER_VERSION}"
echo ""

# =============================================================================
# Step 2: Load Docker Images
# =============================================================================
if [ "${SKIP_LOAD}" = false ] && [ -d "${DEPLOY_DIR}/images" ]; then
    echo "[Step 2/7] Loading Docker images..."
    "${SCRIPT_DIR}/load-images.sh" "${DEPLOY_DIR}/images"
    echo ""
else
    echo "[Step 2/7] Skipping image loading (--skip-load or no images directory)"
    echo ""
fi

# =============================================================================
# Step 3: Generate Secrets and Create .env
# =============================================================================
echo "[Step 3/7] Generating configuration..."

# Generate random secrets
generate_secret() {
    openssl rand -base64 32 | tr -dc 'a-zA-Z0-9' | head -c 32
}

# Check if .env already exists (upgrade scenario)
if [ -f "${DEPLOY_DIR}/.env" ]; then
    echo "  Found existing .env, preserving secrets for upgrade..."
    # Source existing .env to preserve secrets
    source "${DEPLOY_DIR}/.env"
    # Keep existing secrets, only update SERVER_IP and ports if provided
    DB_PASSWORD="${DB_PASSWORD}"
    JWT_SECRET="${JWT_SECRET}"
    INTERNAL_API_SECRET="${INTERNAL_API_SECRET}"
    MINIO_ROOT_PASSWORD="${MINIO_ROOT_PASSWORD}"
else
    echo "  First installation, generating new secrets..."
    DB_PASSWORD=$(generate_secret)
    JWT_SECRET=$(generate_secret)
    INTERNAL_API_SECRET=$(generate_secret)
    MINIO_ROOT_PASSWORD=$(generate_secret)
fi

# Create .env file
cat > "${DEPLOY_DIR}/.env" << EOF
# =============================================================================
# AgentsMesh OnPremise Configuration
# Generated: $(date -Iseconds)
# =============================================================================

# Version (must match image tags)
VERSION=${VERSION:-v1.0.0}

# Server Configuration
SERVER_IP=${SERVER_IP}
HTTP_PORT=${HTTP_PORT}
GRPC_PORT=${GRPC_PORT}

# Docker Socket (auto-detected)
DOCKER_SOCKET=${DOCKER_SOCKET}

# Docker Compose
COMPOSE_PROJECT_NAME=agentsmesh

# Database
DB_PASSWORD=${DB_PASSWORD}

# Authentication
JWT_SECRET=${JWT_SECRET}
INTERNAL_API_SECRET=${INTERNAL_API_SECRET}

# Storage
MINIO_ROOT_PASSWORD=${MINIO_ROOT_PASSWORD}

# External Ports (for debugging)
POSTGRES_PORT=5432
REDIS_PORT=6379
MINIO_API_PORT=9000
MINIO_CONSOLE_PORT=9001

# Deployment
DEPLOYMENT_TYPE=onpremise
EMAIL_PROVIDER=console
EOF

echo "  Generated .env with secure secrets"
echo ""

# =============================================================================
# Step 4: Generate SSL Certificates
# =============================================================================
echo "[Step 4/7] Generating SSL certificates..."
"${SCRIPT_DIR}/generate-certs.sh" "${SERVER_IP}"
echo ""

# =============================================================================
# Step 5: Start Services
# =============================================================================
echo "[Step 5/7] Starting services..."
docker compose up -d

echo "  Waiting for services to become healthy..."
sleep 10

# Wait for backend to be healthy (max 120 seconds)
TIMEOUT=120
ELAPSED=0
while [ $ELAPSED -lt $TIMEOUT ]; do
    if docker compose exec -T backend wget --no-verbose --tries=1 --spider http://localhost:8080/health 2>/dev/null; then
        echo "  Backend is healthy."
        break
    fi
    sleep 5
    ELAPSED=$((ELAPSED + 5))
    echo "  Waiting for backend... (${ELAPSED}s)"
done

if [ $ELAPSED -ge $TIMEOUT ]; then
    echo "  Warning: Backend health check timed out after ${TIMEOUT}s"
    echo "  Continuing anyway..."
fi
echo ""

# =============================================================================
# Step 6: Run Database Migrations
# =============================================================================
echo "[Step 6/7] Running database migrations..."

# Get database connection string
DB_URL="postgres://agentsmesh:${DB_PASSWORD}@postgres:5432/agentsmesh?sslmode=disable"

# Run migrations using migrate tool in backend container
docker compose exec -T backend migrate -path /app/migrations -database "${DB_URL}" up

echo "  Migrations completed."
echo ""

# =============================================================================
# Step 7: Import Seed Data
# =============================================================================
echo "[Step 7/7] Importing initial data..."

# Copy and execute seed SQL
docker compose exec -T postgres psql -U agentsmesh -d agentsmesh < "${DEPLOY_DIR}/seed/onpremise-seed.sql"

echo "  Seed data imported."
echo ""

# =============================================================================
# Complete
# =============================================================================
echo "=============================================="
echo "Installation Complete!"
echo "=============================================="
echo ""
echo "Access URLs:"
echo "  Frontend:      http://${SERVER_IP}:${HTTP_PORT}"
echo "  Admin Console: http://${SERVER_IP}:3001"
echo "  MinIO Console: http://${SERVER_IP}:9001"
echo ""
echo "Admin Account:"
echo "  Email:    admin@localhost.local"
echo "  Password: Admin@123"
echo ""
echo "Runner Registration:"
echo "  curl -fsSL https://agentsmesh.ai/install.sh | sh"
echo "  agentsmesh-runner register --server http://${SERVER_IP}:${HTTP_PORT} --token onpremise-runner-token"
echo "  agentsmesh-runner run"
echo ""
echo "Useful Commands:"
echo "  View logs:    docker compose logs -f"
echo "  Stop:         docker compose down"
echo "  Start:        docker compose up -d"
echo "  Clean:        docker compose down -v"
echo ""
