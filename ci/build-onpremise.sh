#!/bin/bash
# =============================================================================
# Build OnPremise Deployment Images
# =============================================================================
#
# Builds all Docker images for on-premise deployment and exports them as tar.
#
# Usage:
#   ./build-onpremise.sh [VERSION]
#
# Example:
#   ./build-onpremise.sh v1.0.0
#   ./build-onpremise.sh latest
#
# Output:
#   images/backend.tar
#   images/web.tar
#   images/web-admin.tar
#   images/relay.tar
#   images/postgres.tar
#   images/redis.tar
#   images/minio.tar
#   images/traefik.tar
#
# =============================================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="${SCRIPT_DIR}/.."
VERSION="${1:-latest}"
IMAGES_DIR="${PROJECT_ROOT}/images"

# Use public Docker Hub registry for base images
# Note: Dockerfile uses ${REGISTRY}/library/xxx, so we just need docker.io
REGISTRY="docker.io"

echo "=============================================="
echo "Building AgentsMesh OnPremise Images"
echo "=============================================="
echo ""
echo "Version: ${VERSION}"
echo "Registry: ${REGISTRY}"
echo "Output: ${IMAGES_DIR}"
echo ""

# Create output directory
mkdir -p "${IMAGES_DIR}"

# Change to project root
cd "${PROJECT_ROOT}"

# =============================================================================
# Build Application Images
# =============================================================================

echo "[1/8] Building backend image..."
docker build \
    -f ci/backend.Dockerfile \
    --build-arg REGISTRY="${REGISTRY}" \
    -t "agentsmesh/backend:${VERSION}" \
    .
echo "  Done."

echo "[2/8] Building web image (runtime env via docker-entrypoint.sh)..."
docker build \
    -f ci/web.Dockerfile \
    --build-arg REGISTRY="${REGISTRY}" \
    -t "agentsmesh/web:${VERSION}" \
    ./web
echo "  Done."

echo "[3/8] Building web-admin image (runtime env via docker-entrypoint.sh)..."
docker build \
    -f ci/web-admin.Dockerfile \
    --build-arg REGISTRY="${REGISTRY}" \
    -t "agentsmesh/web-admin:${VERSION}" \
    ./web-admin
echo "  Done."

echo "[4/8] Building relay image..."
docker build \
    -f ci/relay.Dockerfile \
    --build-arg REGISTRY="${REGISTRY}" \
    -t "agentsmesh/relay:${VERSION}" \
    .
echo "  Done."

# =============================================================================
# Pull Base Images
# =============================================================================

echo "[5/8] Pulling base images..."
docker pull postgres:16-alpine
docker pull redis:7-alpine
docker pull pgsty/minio:latest
docker pull traefik:v3.2
echo "  Done."

# =============================================================================
# Export Images
# =============================================================================

echo "[6/8] Exporting application images..."
docker save "agentsmesh/backend:${VERSION}" -o "${IMAGES_DIR}/backend.tar"
docker save "agentsmesh/web:${VERSION}" -o "${IMAGES_DIR}/web.tar"
docker save "agentsmesh/web-admin:${VERSION}" -o "${IMAGES_DIR}/web-admin.tar"
docker save "agentsmesh/relay:${VERSION}" -o "${IMAGES_DIR}/relay.tar"
echo "  Done."

echo "[7/8] Exporting base images..."
docker save postgres:16-alpine -o "${IMAGES_DIR}/postgres.tar"
docker save redis:7-alpine -o "${IMAGES_DIR}/redis.tar"
docker save pgsty/minio:latest -o "${IMAGES_DIR}/minio.tar"
docker save traefik:v3.2 -o "${IMAGES_DIR}/traefik.tar"
echo "  Done."

# =============================================================================
# Summary
# =============================================================================

echo "[8/8] Calculating image sizes..."
echo ""
echo "=============================================="
echo "Build Complete!"
echo "=============================================="
echo ""
echo "Images exported to ${IMAGES_DIR}:"
ls -lh "${IMAGES_DIR}"/*.tar
echo ""
TOTAL_SIZE=$(du -sh "${IMAGES_DIR}" | cut -f1)
echo "Total size: ${TOTAL_SIZE}"
echo ""
echo "Next steps:"
echo "  1. Run: ./pack-onpremise.sh ${VERSION}"
echo "  2. Copy to target server"
echo "  3. Run: ./scripts/install.sh --ip <SERVER_IP>"
echo ""
