# AgentsMesh Deployment Guide

## Overview

AgentsMesh can be deployed using Docker Compose for development/staging or Kubernetes for production environments.

## Prerequisites

- Docker 20.10+ and Docker Compose v2
- Kubernetes 1.25+ (for production)
- PostgreSQL 15+
- Redis 7+

## Quick Start (Docker Compose)

### Development Environment

```bash
# Clone the repository
git clone https://github.com/agentsmesh/agentsmesh.git
cd agentsmesh

# Copy environment file
cp .env.example .env

# Start services
docker compose up -d

# Apply database migrations
docker compose exec backend ./migrate up

# Access the application
# Web: http://localhost:3000
# API: http://localhost:8080
```

### Production Environment

```bash
# Use production compose file
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d
```

## Kubernetes Deployment

### Using Kustomize

```bash
# Development
kubectl apply -k deploy/kubernetes/overlays/dev

# Staging
kubectl apply -k deploy/kubernetes/overlays/staging

# Production
kubectl apply -k deploy/kubernetes/overlays/production
```

### Using Helm (Self-hosted)

```bash
# Add Helm repository
helm repo add agentsmesh https://charts.agentsmesh.io
helm repo update

# Install
helm install agentsmesh agentsmesh/agentsmesh \
  --namespace agentsmesh \
  --create-namespace \
  -f values.yaml
```

## Configuration

### Environment Variables

#### Backend

| Variable | Description | Default |
|----------|-------------|---------|
| `SERVER_PORT` | HTTP server port | `8080` |
| `SERVER_DEBUG` | Enable debug mode | `false` |
| `DATABASE_URL` | PostgreSQL connection string | Required |
| `REDIS_URL` | Redis connection string | Required |
| `JWT_SECRET` | JWT signing secret | Required |
| `JWT_EXPIRY` | JWT expiry duration | `24h` |

#### OAuth Providers

| Variable | Description |
|----------|-------------|
| `OAUTH_GITHUB_CLIENT_ID` | GitHub OAuth App Client ID |
| `OAUTH_GITHUB_CLIENT_SECRET` | GitHub OAuth App Client Secret |
| `OAUTH_GOOGLE_CLIENT_ID` | Google OAuth Client ID |
| `OAUTH_GOOGLE_CLIENT_SECRET` | Google OAuth Client Secret |
| `OAUTH_GITLAB_CLIENT_ID` | GitLab OAuth App ID |
| `OAUTH_GITLAB_CLIENT_SECRET` | GitLab OAuth App Secret |
| `OAUTH_GITEE_CLIENT_ID` | Gitee OAuth App ID |
| `OAUTH_GITEE_CLIENT_SECRET` | Gitee OAuth App Secret |

#### Billing (Stripe)

| Variable | Description |
|----------|-------------|
| `STRIPE_SECRET_KEY` | Stripe secret key |
| `STRIPE_WEBHOOK_SECRET` | Stripe webhook signing secret |

### Database Migrations

```bash
# Run migrations
./backend/migrate up

# Rollback last migration
./backend/migrate down 1

# Check migration status
./backend/migrate status
```

## Architecture

```
                    ┌─────────────────┐
                    │   Load Balancer │
                    │  (nginx/ALB)    │
                    └────────┬────────┘
                             │
              ┌──────────────┼──────────────┐
              │              │              │
       ┌──────▼──────┐ ┌─────▼─────┐ ┌─────▼─────┐
       │    Web      │ │   API     │ │    WS     │
       │  (Next.js)  │ │  (Go)     │ │  (Go)     │
       └──────┬──────┘ └─────┬─────┘ └─────┬─────┘
              │              │              │
              └──────────────┼──────────────┘
                             │
              ┌──────────────┼──────────────┐
              │              │              │
       ┌──────▼──────┐ ┌─────▼─────┐ ┌─────▼─────┐
       │  PostgreSQL │ │   Redis   │ │  Runners  │
       │             │ │           │ │ (self-    │
       │             │ │           │ │  hosted)  │
       └─────────────┘ └───────────┘ └───────────┘
```

## Scaling

### Horizontal Pod Autoscaler

Production deployment includes HPA configuration:

```yaml
# Backend HPA
minReplicas: 3
maxReplicas: 10
metrics:
  - cpu: 70%
  - memory: 80%

# Web HPA
minReplicas: 3
maxReplicas: 8
metrics:
  - cpu: 70%
```

### Pod Disruption Budget

```yaml
# Ensure minimum availability during updates
minAvailable: 2
```

## Monitoring

### Health Checks

- `/health` - Liveness probe
- `/health/ready` - Readiness probe

### Metrics

Prometheus metrics are exposed at `/metrics` (when enabled).

### Logging

Structured JSON logs are output to stdout. Configure log level:

```bash
LOG_LEVEL=info  # debug, info, warn, error
```

## Backup & Recovery

### Database Backup

```bash
# Create backup
pg_dump -h localhost -U agentsmesh agentsmesh > backup.sql

# Restore backup
psql -h localhost -U agentsmesh agentsmesh < backup.sql
```

### Redis Backup

Enable Redis persistence (RDB/AOF) in your configuration.

## Security

### TLS/SSL

Production deployments should use TLS. Configure via:

1. **Load Balancer termination** - Recommended
2. **Ingress Controller** - Using cert-manager
3. **Direct TLS** - Configure `TLS_CERT_FILE` and `TLS_KEY_FILE`

### Network Policies

Apply Kubernetes network policies to restrict traffic:

```bash
kubectl apply -f deploy/kubernetes/network-policies.yaml
```

### Secrets Management

Use Kubernetes Secrets or external secret management (Vault, AWS Secrets Manager):

```bash
# Create secrets
kubectl create secret generic agentsmesh-secrets \
  --from-literal=jwt-secret=<secret> \
  --from-literal=database-url=<url>
```

## Troubleshooting

### Common Issues

1. **Database connection failed**
   - Check DATABASE_URL format
   - Verify network connectivity
   - Check PostgreSQL logs

2. **Runner not connecting**
   - Verify registration token
   - Check firewall rules
   - Ensure gRPC connectivity (port 9443)

3. **OAuth login failed**
   - Verify OAuth app credentials
   - Check callback URL configuration
   - Review OAuth provider logs

### Debug Mode

Enable debug mode for verbose logging:

```bash
SERVER_DEBUG=true
LOG_LEVEL=debug
```

### Logs

```bash
# Docker
docker compose logs -f backend

# Kubernetes
kubectl logs -f deployment/backend -n agentsmesh
```
