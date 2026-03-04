# Self-Hosted Deployment

Deploy AgentsMesh on your own infrastructure using Docker.

## Prerequisites

- Docker 20.10+ with Compose V2
- OpenSSL (for certificate generation)
- 4 GB+ RAM, 20 GB+ disk
- A server IP address or domain name
- Ports 80 (HTTP) and 9443 (gRPC) available

## Quick Start

```bash
git clone https://github.com/AgentsMesh/AgentsMesh.git
cd AgentsMesh/deploy/selfhost

# Install with your server IP
./selfhost.sh --host 192.168.1.100

# Or with a domain name and custom ports
./selfhost.sh --host agentsmesh.example.com --http-port 8080

# Pin a specific image version
./selfhost.sh --host 192.168.1.100 --version sha-abc1234
```

The script will:

1. Generate secrets and `.env` configuration
2. Generate SSL certificates for gRPC mTLS
3. Pull Docker images from Docker Hub
4. Start all services
5. Run database migrations
6. Import seed data (admin account + runner token)

## Access

| Service | URL |
|---------|-----|
| Web Console | `http://<HOST>:<HTTP_PORT>` |
| MinIO Console | `http://<HOST>:9001` |

**Default Admin Account:**

| Field | Value |
|-------|-------|
| Email | `admin@localhost.local` |
| Password | `Admin@123` |

> **Change the admin password immediately after first login.**

## Register a Runner

Runners execute AI agents on your machines. Install the runner:

```bash
# Install (macOS / Linux)
curl -fsSL https://agentsmesh.ai/install.sh | sh

# Register
agentsmesh-runner register \
  --server http://<HOST>:<HTTP_PORT> \
  --token selfhost-runner-token

# Start
agentsmesh-runner run
```

## Manual Setup

If you prefer manual installation over the script:

```bash
# 1. Create configuration
cp .env.example .env
# Edit .env — set SERVER_HOST and replace all __CHANGE_ME__ with random secrets

# 2. Generate SSL certificates for gRPC mTLS
mkdir -p ssl
# Generate CA
openssl ecparam -name prime256v1 -genkey -noout -out ssl/ca.key
openssl req -new -x509 -days 3650 -key ssl/ca.key -out ssl/ca.crt \
  -subj "/CN=AgentsMesh CA/O=AgentsMesh"
# Generate server cert (add your IP/domain to SAN)
openssl ecparam -name prime256v1 -genkey -noout -out ssl/server.key
# ... see selfhost.sh for full certificate generation steps

# 3. Pull and start services
docker compose pull
docker compose up -d

# 4. Run migrations
docker compose exec -T backend migrate -path /app/migrations \
  -database "postgres://agentsmesh:<DB_PASSWORD>@postgres:5432/agentsmesh?sslmode=disable" up

# 5. Import seed data
docker compose exec -T postgres psql -U agentsmesh -d agentsmesh < seed/seed.sql
```

## Configuration

All configuration is in `.env`. Key variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_HOST` | — | Server IP or domain (required) |
| `HTTP_PORT` | `80` | HTTP port |
| `GRPC_PORT` | `9443` | gRPC port for Runner connections |
| `VERSION` | `latest` | Docker image tag |
| `EMAIL_PROVIDER` | `console` | `console` (log only) or `smtp` |

## Operations

```bash
# View service status
docker compose ps

# View logs
docker compose logs -f              # All services
docker compose logs -f backend      # Backend only

# Restart a service
docker compose restart backend

# Stop all services
docker compose down

# Stop and remove all data
./selfhost.sh --clean
```

## Upgrade

```bash
# Pull new images
docker compose pull

# Recreate containers
docker compose up -d

# Run new migrations (if any)
docker compose exec -T backend migrate -path /app/migrations \
  -database "postgres://agentsmesh:${DB_PASSWORD}@postgres:5432/agentsmesh?sslmode=disable" up
```

## Backup & Restore

### Database

```bash
# Backup
docker compose exec -T postgres pg_dump -U agentsmesh agentsmesh > backup.sql

# Restore
docker compose exec -T postgres psql -U agentsmesh -d agentsmesh < backup.sql
```

### All Data (PostgreSQL + MinIO)

```bash
docker compose down

docker run --rm -v agentsmesh_postgres_data:/data -v $(pwd):/backup alpine \
  tar czf /backup/postgres_backup.tar.gz -C /data .

docker run --rm -v agentsmesh_minio_data:/data -v $(pwd):/backup alpine \
  tar czf /backup/minio_backup.tar.gz -C /data .

docker compose up -d
```

## Firewall

Ensure these ports are accessible:

| Port | Protocol | Purpose |
|------|----------|---------|
| 80 (or `HTTP_PORT`) | TCP | Web console + API |
| 9443 (or `GRPC_PORT`) | TCP | Runner gRPC + mTLS |

## Troubleshooting

**Services won't start:**

```bash
docker compose logs --tail=50
```

**Backend unhealthy:**

```bash
docker compose exec -T backend wget --spider http://localhost:8080/health
docker compose logs backend
```

**Runner can't connect:**

1. Check firewall allows port 9443
2. Verify SSL certificate includes the server host: `openssl x509 -in ssl/server.crt -text -noout | grep -A2 "Subject Alternative Name"`
3. Check backend gRPC logs: `docker compose logs backend | grep grpc`

**SSL certificate expired (1 year validity):**

```bash
rm -rf ssl/
./selfhost.sh --host <SERVER_HOST>
# Or just regenerate certs and restart:
# delete ssl/, rerun selfhost.sh which will regenerate them
```

## Architecture

```
                    ┌──────────────────────────────────────────┐
 Browser ──HTTP──── │ :80  Traefik                             │
                    │   ├── /api/*   → Backend (:8080)         │
                    │   ├── /relay/* → Relay   (:8090)  [WS]   │
                    │   └── /*       → Web     (:3000)         │
                    │                                          │
 Runner ──gRPC───── │ :9443 TLS passthrough → Backend (:9090)  │
                    │                                          │
                    │ PostgreSQL · Redis · MinIO                │
                    └──────────────────────────────────────────┘
```
