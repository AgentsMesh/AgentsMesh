# AgentsMesh Self-Hosted Deployment Guide

This guide covers deploying AgentsMesh on your own infrastructure for private/enterprise use.

## Overview

AgentsMesh can be fully self-hosted, giving you complete control over your data and infrastructure. The self-hosted version includes all features of the cloud version.

## System Requirements

### Minimum Requirements

| Component | Specification |
|-----------|--------------|
| CPU | 4 cores |
| RAM | 8 GB |
| Storage | 50 GB SSD |
| OS | Linux (Ubuntu 22.04+, RHEL 8+) |

### Recommended for Production

| Component | Specification |
|-----------|--------------|
| CPU | 8+ cores |
| RAM | 32 GB |
| Storage | 200 GB SSD |
| Network | 1 Gbps |

## Deployment Options

### Option 1: Docker Compose (Simple)

Best for small teams or evaluation.

```bash
# Download docker-compose files
curl -O https://raw.githubusercontent.com/agentsmesh/agentsmesh/main/deploy/docker-compose.yml
curl -O https://raw.githubusercontent.com/agentsmesh/agentsmesh/main/deploy/docker-compose.prod.yml
curl -O https://raw.githubusercontent.com/agentsmesh/agentsmesh/main/.env.example

# Configure environment
cp .env.example .env
vim .env  # Edit configuration

# Start services
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d
```

### Option 2: Kubernetes + Helm (Production)

Best for larger deployments with high availability requirements.

```bash
# Add Helm repository
helm repo add agentsmesh https://charts.agentsmesh.io
helm repo update

# Create namespace
kubectl create namespace agentsmesh

# Create secrets
kubectl create secret generic agentsmesh-secrets \
  --namespace agentsmesh \
  --from-literal=jwt-secret=$(openssl rand -hex 32) \
  --from-literal=database-password=$(openssl rand -hex 16)

# Install with custom values
helm install agentsmesh agentsmesh/agentsmesh \
  --namespace agentsmesh \
  -f values.yaml
```

## Configuration

### values.yaml Example

```yaml
# AgentsMesh Helm Values

global:
  domain: agentsmesh.company.com

backend:
  replicas: 3
  resources:
    requests:
      memory: "512Mi"
      cpu: "500m"
    limits:
      memory: "1Gi"
      cpu: "1000m"

  config:
    logLevel: info
    debug: false

web:
  replicas: 2
  resources:
    requests:
      memory: "256Mi"
      cpu: "200m"

postgresql:
  enabled: true  # Use built-in PostgreSQL
  # Or use external:
  # enabled: false
  # external:
  #   host: your-db-host
  #   port: 5432
  #   database: agentsmesh
  #   username: agentsmesh
  #   existingSecret: db-credentials

redis:
  enabled: true  # Use built-in Redis
  # Or use external Redis

ingress:
  enabled: true
  className: nginx
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
  tls:
    enabled: true

oauth:
  github:
    enabled: true
    # clientId and clientSecret from secret
  gitlab:
    enabled: true
    baseUrl: https://gitlab.company.com  # Self-hosted GitLab
```

## Runner Setup

Runners are self-hosted agents that execute AI pods. Each organization deploys their own runners.

### Installing a Runner

1. **Generate Registration Token**

   In the web UI: Settings → Runners → Create Token

2. **Deploy Runner**

   ```bash
   # Docker
   docker run -d \
     --name agentsmesh-runner \
     -e REGISTRATION_TOKEN=<token> \
     -e BACKEND_URL=https://agentsmesh.company.com \
     -e NODE_ID=runner-01 \
     -v /var/run/docker.sock:/var/run/docker.sock \
     agentsmesh/runner:latest

   # Or using docker-compose
   docker compose -f runner-compose.yml up -d
   ```

3. **Verify Registration**

   The runner should appear in Settings → Runners with "Online" status.

### Runner Configuration

```yaml
# runner-compose.yml
version: '3.8'

services:
  runner:
    image: agentsmesh/runner:latest
    container_name: agentsmesh-runner
    restart: unless-stopped
    environment:
      - REGISTRATION_TOKEN=${REGISTRATION_TOKEN}
      - BACKEND_URL=${BACKEND_URL}
      - NODE_ID=${HOSTNAME}
      - MAX_CONCURRENT_PODS=5
      - WORKSPACE_BASE=/workspaces
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - runner-workspaces:/workspaces

volumes:
  runner-workspaces:
```

## Git Provider Integration

### Self-Hosted GitLab

```yaml
oauth:
  gitlab:
    enabled: true
    baseUrl: https://gitlab.company.com
    clientId: <from-gitlab-app>
    clientSecret: <from-gitlab-app>
```

1. Create GitLab Application:
   - Go to Admin → Applications
   - Name: AgentsMesh
   - Redirect URI: `https://agentsmesh.company.com/api/v1/auth/oauth/gitlab/callback`
   - Scopes: `api`, `read_user`, `read_repository`

### GitHub Enterprise

```yaml
oauth:
  github:
    enabled: true
    baseUrl: https://github.company.com  # For GHE
    clientId: <from-github-app>
    clientSecret: <from-github-app>
```

## SSL/TLS Configuration

### Using cert-manager (Kubernetes)

```yaml
ingress:
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
  tls:
    enabled: true
    secretName: agentsmesh-tls
```

### Using Custom Certificates

```yaml
ingress:
  tls:
    enabled: true
    secretName: custom-tls-secret

# Create secret with your certificates
kubectl create secret tls custom-tls-secret \
  --cert=path/to/cert.pem \
  --key=path/to/key.pem \
  -n agentsmesh
```

## LDAP/SSO Integration

### SAML 2.0

```yaml
auth:
  saml:
    enabled: true
    entityId: https://agentsmesh.company.com
    ssoUrl: https://idp.company.com/saml/sso
    certificate: |
      -----BEGIN CERTIFICATE-----
      ...
      -----END CERTIFICATE-----
    attributeMapping:
      email: emailAddress
      name: displayName
      groups: memberOf
```

### LDAP

```yaml
auth:
  ldap:
    enabled: true
    host: ldap.company.com
    port: 636
    useTLS: true
    baseDN: dc=company,dc=com
    bindDN: cn=agentsmesh,ou=services,dc=company,dc=com
    userFilter: (uid={0})
    groupFilter: (member={0})
```

## Backup & Disaster Recovery

### Database Backup

```bash
# Automated backup with cron
0 2 * * * pg_dump -h localhost -U agentsmesh agentsmesh | gzip > /backups/agentsmesh-$(date +\%Y\%m\%d).sql.gz

# Restore
gunzip -c backup.sql.gz | psql -h localhost -U agentsmesh agentsmesh
```

### Full System Backup

Using Velero for Kubernetes:

```bash
# Install Velero
velero install --provider aws --bucket <bucket> ...

# Create backup
velero backup create agentsmesh-backup --include-namespaces agentsmesh

# Restore
velero restore create --from-backup agentsmesh-backup
```

## Monitoring & Observability

### Prometheus + Grafana

```yaml
monitoring:
  prometheus:
    enabled: true
    serviceMonitor:
      enabled: true

  grafana:
    enabled: true
    dashboards:
      - agentsmesh-overview
      - agentsmesh-pods
```

### Log Aggregation

Configure structured logging with ELK or Loki:

```yaml
backend:
  config:
    logFormat: json
    logLevel: info
```

## Security Hardening

### Network Policies

```yaml
networkPolicies:
  enabled: true
  # Restrict backend to only accept traffic from web and runners
  backend:
    ingress:
      - from:
          - podSelector:
              matchLabels:
                app: web
          - podSelector:
              matchLabels:
                app: runner
```

### Pod Security Standards

```yaml
podSecurityContext:
  runAsNonRoot: true
  runAsUser: 1000
  fsGroup: 1000

securityContext:
  allowPrivilegeEscalation: false
  readOnlyRootFilesystem: true
  capabilities:
    drop:
      - ALL
```

## Upgrades

### Minor Upgrades

```bash
# Helm
helm upgrade agentsmesh agentsmesh/agentsmesh -n agentsmesh -f values.yaml

# Docker Compose
docker compose pull
docker compose up -d
```

### Major Upgrades

1. Review release notes for breaking changes
2. Backup database
3. Apply database migrations
4. Update configuration if needed
5. Deploy new version

## Support

For enterprise support and consulting:
- Email: enterprise@agentsmesh.io
- Documentation: https://docs.agentsmesh.io/self-hosted
