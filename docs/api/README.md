# AgentsMesh API Documentation

## Overview

AgentsMesh provides APIs for managing multi-agent AI development workspaces:

- **REST API**: For web/mobile clients (authentication, resources, management)
- **External API**: For third-party integrations (API key authentication)
- **gRPC + mTLS**: For Runner connections (bidirectional streaming, certificate-based authentication)
- **WebSocket**: For real-time terminal streaming and events

The platform supports multi-tenancy, OAuth authentication, and real-time terminal streaming.

## Base URL

- Production: `https://your-domain.com/api/v1`
- Development: `http://localhost:10000/api/v1` (via Traefik)

## Authentication

### JWT Authentication

Most endpoints require JWT authentication. Include the token in the Authorization header:

```
Authorization: Bearer <token>
```

### Obtaining a Token

#### Email/Password Login

```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "your-password"
}
```

Response:
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
  "expires_in": 3600,
  "user": {
    "id": 1,
    "email": "user@example.com",
    "username": "johndoe",
    "name": "John Doe"
  }
}
```

#### OAuth Login

Supported providers: GitHub, Google, GitLab, Gitee

```http
GET /api/v1/auth/oauth/{provider}?redirect=/dashboard
```

### Token Refresh

```http
POST /api/v1/auth/refresh
Content-Type: application/json

{
  "refresh_token": "eyJhbGciOiJIUzI1NiIs..."
}
```

### API Key Authentication (External API)

For external integrations, use API keys:

```
Authorization: Bearer <api_key>
```

API keys are scoped to an organization and can have fine-grained permissions.

## Multi-Tenancy

Organization-scoped endpoints require the organization slug in the URL path:

```
/api/v1/orgs/{slug}/...
```

---

## REST API Endpoints

### Authentication (`/api/v1/auth`)

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/auth/login` | Email/password login |
| POST | `/auth/register` | User registration |
| POST | `/auth/refresh` | Refresh JWT token |
| POST | `/auth/logout` | Logout and revoke token |
| POST | `/auth/verify-email` | Verify email address |
| POST | `/auth/resend-verification` | Resend verification email |
| POST | `/auth/forgot-password` | Request password reset |
| POST | `/auth/reset-password` | Reset password with token |
| GET | `/auth/oauth/github` | GitHub OAuth redirect |
| GET | `/auth/oauth/github/callback` | GitHub OAuth callback |
| GET | `/auth/oauth/google` | Google OAuth redirect |
| GET | `/auth/oauth/google/callback` | Google OAuth callback |
| GET | `/auth/oauth/gitlab` | GitLab OAuth redirect |
| GET | `/auth/oauth/gitlab/callback` | GitLab OAuth callback |
| GET | `/auth/oauth/gitee` | Gitee OAuth redirect |
| GET | `/auth/oauth/gitee/callback` | Gitee OAuth callback |

### Users (`/api/v1/users`)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/users/me` | Get current user profile |
| PUT | `/users/me` | Update profile |
| POST | `/users/me/password` | Change password |
| GET | `/users/me/organizations` | List user's organizations |
| GET | `/users/me/identities` | List OAuth identities |
| DELETE | `/users/me/identities/{provider}` | Remove OAuth identity |
| GET | `/users/me/agent-configs` | List user agent configs |
| GET | `/users/me/agent-configs/{slug}` | Get agent config |
| PUT | `/users/me/agent-configs/{slug}` | Set agent config |
| DELETE | `/users/me/agent-configs/{slug}` | Delete agent config |
| GET | `/users/me/agentpod/settings` | Get AgentPod settings |
| PUT | `/users/me/agentpod/settings` | Update AgentPod settings |
| GET | `/users/me/agentpod/providers` | List AI providers |
| POST | `/users/me/agentpod/providers` | Create AI provider |
| PUT | `/users/me/agentpod/providers/{id}` | Update AI provider |
| DELETE | `/users/me/agentpod/providers/{id}` | Delete AI provider |
| POST | `/users/me/agentpod/providers/{id}/default` | Set default provider |
| GET | `/users/search?q=` | Search users |

### Organizations (`/api/v1/orgs`)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/orgs` | List user's organizations |
| POST | `/orgs` | Create organization |
| GET | `/orgs/{slug}` | Get organization |
| PUT | `/orgs/{slug}` | Update organization |
| DELETE | `/orgs/{slug}` | Delete organization |
| GET | `/orgs/{slug}/members` | List members |
| POST | `/orgs/{slug}/members` | Invite member |
| PUT | `/orgs/{slug}/members/{user_id}` | Update member role |
| DELETE | `/orgs/{slug}/members/{user_id}` | Remove member |

### Code Agents (`/api/v1/orgs/{slug}/agents`)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/agents/types` | List agents |
| GET | `/agents/types/{slug}` | Get agent |
| POST | `/agents/custom` | Create custom agent |
| PUT | `/agents/custom/{id}` | Update custom agent |
| DELETE | `/agents/custom/{id}` | Delete custom agent |
| GET | `/agents/{slug}/config-schema` | Get config schema |

### Repositories (`/api/v1/orgs/{slug}/repositories`)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/repositories` | List repositories |
| POST | `/repositories` | Create repository |
| GET | `/repositories/{id}` | Get repository |
| PUT | `/repositories/{id}` | Update repository |
| DELETE | `/repositories/{id}` | Delete repository |
| GET | `/repositories/{id}/branches` | List branches |
| POST | `/repositories/{id}/sync-branches` | Sync branches from remote |
| POST | `/repositories/{id}/webhook` | Register webhook |
| DELETE | `/repositories/{id}/webhook` | Delete webhook |
| GET | `/repositories/{id}/webhook/status` | Get webhook status |
| GET | `/repositories/{id}/webhook/secret` | Get webhook secret |
| POST | `/repositories/{id}/webhook/configured` | Mark webhook configured |
| GET | `/repositories/{id}/merge-requests` | List merge requests |

### Extensions - Skills (`/api/v1/orgs/{slug}/repositories/{id}/skills`)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/repositories/{id}/skills` | List installed skills |
| POST | `/repositories/{id}/skills/install-from-market` | Install from market |
| POST | `/repositories/{id}/skills/install-from-github` | Install from GitHub |
| POST | `/repositories/{id}/skills/install-from-upload` | Install from upload |
| PUT | `/repositories/{id}/skills/{installId}` | Update skill |
| DELETE | `/repositories/{id}/skills/{installId}` | Uninstall skill |

### Extensions - MCP Servers (`/api/v1/orgs/{slug}/repositories/{id}/mcp-servers`)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/repositories/{id}/mcp-servers` | List installed MCP servers |
| POST | `/repositories/{id}/mcp-servers/install-from-market` | Install from market |
| POST | `/repositories/{id}/mcp-servers/install-custom` | Install custom MCP server |
| PUT | `/repositories/{id}/mcp-servers/{installId}` | Update MCP server |
| DELETE | `/repositories/{id}/mcp-servers/{installId}` | Uninstall MCP server |

### Skill Registries (`/api/v1/orgs/{slug}/skill-registries`)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/skill-registries` | List skill registries |
| POST | `/skill-registries` | Create skill registry |
| POST | `/skill-registries/{id}/sync` | Sync skill registry |
| DELETE | `/skill-registries/{id}` | Delete skill registry |
| PUT | `/skill-registries/{id}/toggle` | Toggle platform registry |
| GET | `/skill-registry-overrides` | List registry overrides |
| GET | `/market/skills` | List market skills |
| GET | `/market/mcp-servers` | List market MCP servers |

### Runners (`/api/v1/orgs/{slug}/runners`)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/runners` | List runners |
| GET | `/runners/{id}` | Get runner |
| GET | `/runners/available` | List available runners |
| PUT | `/runners/{id}` | Update runner |
| DELETE | `/runners/{id}` | Delete runner |
| GET | `/runners/{id}/pods` | List runner's pods |
| POST | `/runners/{id}/sandboxes/query` | Query sandbox info |

### Pods (`/api/v1/orgs/{slug}/pods`)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/pods` | List pods |
| POST | `/pods` | Create pod |
| GET | `/pods/{key}` | Get pod |
| POST | `/pods/{key}/terminate` | Terminate pod |
| GET | `/pods/{key}/connect` | Get connection info |
| GET | `/pods/{key}/terminal/connect` | Get Relay connection info |

### Channels (`/api/v1/orgs/{slug}/channels`)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/channels` | List channels |
| POST | `/channels` | Create channel |
| GET | `/channels/{id}` | Get channel |
| PUT | `/channels/{id}` | Update channel |
| POST | `/channels/{id}/archive` | Archive channel |
| POST | `/channels/{id}/unarchive` | Unarchive channel |
| GET | `/channels/{id}/messages` | List messages |
| POST | `/channels/{id}/messages` | Send message |
| GET | `/channels/{id}/document` | Get channel document |
| PUT | `/channels/{id}/document` | Update channel document |
| GET | `/channels/{id}/pods` | List channel pods |
| POST | `/channels/{id}/pods` | Join pod to channel |
| DELETE | `/channels/{id}/pods/{pod_key}` | Remove pod from channel |

### Tickets (`/api/v1/orgs/{slug}/tickets`)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/tickets` | List tickets |
| POST | `/tickets` | Create ticket |
| GET | `/tickets/active` | Get active tickets |
| GET | `/tickets/board` | Get kanban board |
| POST | `/tickets/batch-pods` | Batch get ticket pods |
| GET | `/tickets/{slug}` | Get ticket |
| PUT | `/tickets/{slug}` | Update ticket |
| DELETE | `/tickets/{slug}` | Delete ticket |
| PATCH | `/tickets/{slug}/status` | Update ticket status |
| POST | `/tickets/{slug}/assignees` | Add assignee |
| DELETE | `/tickets/{slug}/assignees/{user_id}` | Remove assignee |
| POST | `/tickets/{slug}/labels` | Add label |
| DELETE | `/tickets/{slug}/labels/{label_id}` | Remove label |
| GET | `/tickets/{slug}/merge-requests` | List merge requests |
| GET | `/tickets/{slug}/sub-tickets` | Get sub-tickets |
| GET | `/tickets/{slug}/relations` | List ticket relations |
| POST | `/tickets/{slug}/relations` | Create ticket relation |
| DELETE | `/tickets/{slug}/relations/{relation_id}` | Delete relation |
| GET | `/tickets/{slug}/commits` | List linked commits |
| POST | `/tickets/{slug}/commits` | Link commit |
| DELETE | `/tickets/{slug}/commits/{commit_id}` | Unlink commit |
| GET | `/tickets/{slug}/comments` | List comments |
| POST | `/tickets/{slug}/comments` | Create comment |
| PUT | `/tickets/{slug}/comments/{id}` | Update comment |
| DELETE | `/tickets/{slug}/comments/{id}` | Delete comment |
| GET | `/tickets/{slug}/pods` | Get ticket pods |
| POST | `/tickets/{slug}/pods` | Create pod for ticket |

### Labels (`/api/v1/orgs/{slug}/labels`)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/labels` | List labels |
| POST | `/labels` | Create label |
| PUT | `/labels/{id}` | Update label |
| DELETE | `/labels/{id}` | Delete label |

### Mesh (`/api/v1/orgs/{slug}/mesh`)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/mesh/topology` | Get mesh topology |

### Messages (`/api/v1/orgs/{slug}/messages`)

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/messages` | Send message |
| GET | `/messages` | Get messages |
| GET | `/messages/unread-count` | Get unread count |
| GET | `/messages/sent` | Get sent messages |
| POST | `/messages/mark-read` | Mark as read |
| POST | `/messages/mark-all-read` | Mark all as read |
| GET | `/messages/conversation/{correlation_id}` | Get conversation |
| GET | `/messages/dlq` | Get dead letter queue |
| POST | `/messages/dlq/{id}/replay` | Replay dead letter |
| GET | `/messages/{id}` | Get message |

### Pod Bindings (`/api/v1/orgs/{slug}/bindings`)

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/bindings` | Request binding |
| GET | `/bindings` | List bindings |
| POST | `/bindings/accept` | Accept binding |
| POST | `/bindings/reject` | Reject binding |
| POST | `/bindings/unbind` | Unbind |
| GET | `/bindings/pending` | Get pending bindings |
| GET | `/bindings/pods` | Get bound pods |
| GET | `/bindings/check/{target_pod}` | Check binding |
| POST | `/bindings/{id}/scopes` | Request scopes |
| POST | `/bindings/{id}/scopes/approve` | Approve scopes |

### Billing (`/api/v1/orgs/{slug}/billing`)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/billing/overview` | Get billing overview |
| GET | `/billing/subscription` | Get subscription |
| POST | `/billing/subscription` | Create subscription |
| PUT | `/billing/subscription` | Update subscription |
| DELETE | `/billing/subscription` | Cancel subscription |
| GET | `/billing/plans` | List plans |
| GET | `/billing/plans/prices` | List plan prices |
| GET | `/billing/plans/{name}/prices` | Get plan prices |
| GET | `/billing/plans/{name}/all-prices` | Get all currency prices |
| GET | `/billing/usage` | Get usage |
| GET | `/billing/usage/history` | Get usage history |
| POST | `/billing/quota` | Set custom quota |
| GET | `/billing/quota/check` | Check quota |
| POST | `/billing/checkout` | Create checkout |
| GET | `/billing/checkout/{order_no}` | Get checkout status |
| POST | `/billing/subscription/cancel` | Request cancellation |
| POST | `/billing/subscription/reactivate` | Reactivate subscription |
| POST | `/billing/subscription/change-cycle` | Change billing cycle |
| POST | `/billing/subscription/downgrade` | Downgrade subscription |
| PUT | `/billing/subscription/auto-renew` | Update auto-renew |
| GET | `/billing/seats` | Get seat usage |
| POST | `/billing/seats/purchase` | Purchase seats |
| GET | `/billing/invoices` | List invoices |
| POST | `/billing/customer-portal` | Get customer portal URL |
| POST | `/billing/stripe/customer` | Create Stripe customer |
| GET | `/billing/deployment` | Get deployment info |
| GET | `/billing/promo-codes` | List promo codes |

### Files (`/api/v1/orgs/{slug}/files`)

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/files/presign` | Get presigned upload URL |

### API Keys (`/api/v1/orgs/{slug}/api-keys`)

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api-keys` | Create API key |
| GET | `/api-keys` | List API keys |
| GET | `/api-keys/{id}` | Get API key |
| PUT | `/api-keys/{id}` | Update API key |
| DELETE | `/api-keys/{id}` | Delete API key |
| POST | `/api-keys/{id}/revoke` | Revoke API key |

### Invitations (`/api/v1/invitations`)

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/invitations/accept` | Accept invitation |
| POST | `/invitations/reject` | Reject invitation |
| GET | `/invitations/pending` | Get pending invitations |
| GET | `/invitations/sent` | Get sent invitations |

### Autopilot (`/api/v1/orgs/{slug}/autopilot`)

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/autopilot/controllers` | Create controller |
| GET | `/autopilot/controllers` | List controllers |
| GET | `/autopilot/controllers/{id}` | Get controller |

### License (`/api/v1/license`)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/license/status` | Get license status |
| POST | `/license/activate` | Activate license |

---

## External API (`/api/v1/ext/orgs/{slug}`)

External API uses API key authentication and is designed for third-party integrations.

### Pods

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/pods` | List pods |
| GET | `/pods/{key}` | Get pod |
| POST | `/pods` | Create pod |
| POST | `/pods/{key}/terminate` | Terminate pod |

### Tickets

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/tickets` | List tickets |
| GET | `/tickets/board` | Get kanban board |
| GET | `/tickets/{slug}` | Get ticket |
| POST | `/tickets` | Create ticket |
| PUT | `/tickets/{slug}` | Update ticket |
| PATCH | `/tickets/{slug}/status` | Update status |
| DELETE | `/tickets/{slug}` | Delete ticket |

### Channels

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/channels` | List channels |
| GET | `/channels/{id}` | Get channel |
| POST | `/channels` | Create channel |
| PUT | `/channels/{id}` | Update channel |
| POST | `/channels/{id}/messages` | Send message |
| GET | `/channels/{id}/messages` | List messages |

### Runners

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/runners` | List runners |
| GET | `/runners/{id}` | Get runner |
| GET | `/runners/available` | List available runners |
| GET | `/runners/{id}/pods` | List runner's pods |

### Repositories

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/repositories` | List repositories |
| GET | `/repositories/{id}` | Get repository |
| GET | `/repositories/{id}/branches` | List branches |
| GET | `/repositories/{id}/merge-requests` | List merge requests |

---

## Admin API (`/api/v1/admin`)

Admin API requires system administrator privileges (`is_system_admin = true`).

### Authentication

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/admin/login` | Admin login |
| GET | `/admin/me` | Get current admin |

### Dashboard

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/admin/dashboard/stats` | Get system statistics |

### User Management

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/admin/users` | List users |
| GET | `/admin/users/{id}` | Get user |
| PUT | `/admin/users/{id}` | Update user |
| POST | `/admin/users/{id}/disable` | Disable user |
| POST | `/admin/users/{id}/enable` | Enable user |
| POST | `/admin/users/{id}/grant-admin` | Grant admin privileges |
| POST | `/admin/users/{id}/revoke-admin` | Revoke admin privileges |

### Organization Management

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/admin/organizations` | List organizations |
| GET | `/admin/organizations/{id}` | Get organization |
| GET | `/admin/organizations/{id}/members` | Get members |
| DELETE | `/admin/organizations/{id}` | Delete organization |

### Runner Management

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/admin/runners` | List runners |
| GET | `/admin/runners/{id}` | Get runner |
| POST | `/admin/runners/{id}/disable` | Disable runner |
| POST | `/admin/runners/{id}/enable` | Enable runner |
| DELETE | `/admin/runners/{id}` | Delete runner |

### Subscription Management

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/admin/subscriptions` | List subscriptions |
| GET | `/admin/subscriptions/{id}` | Get subscription |
| PUT | `/admin/subscriptions/{id}` | Update subscription |

### Promo Code Management

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/admin/promo-codes` | List promo codes |
| POST | `/admin/promo-codes` | Create promo code |
| PUT | `/admin/promo-codes/{id}` | Update promo code |
| DELETE | `/admin/promo-codes/{id}` | Delete promo code |

### Relay Management

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/admin/relays` | List relays |
| GET | `/admin/relays/{id}` | Get relay |
| DELETE | `/admin/relays/{id}` | Delete relay |

### Skill Registry Management

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/admin/skill-registries` | List skill registries |
| POST | `/admin/skill-registries` | Create skill registry |
| DELETE | `/admin/skill-registries/{id}` | Delete skill registry |

### Audit Logs

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/admin/audit-logs` | List audit logs |

---

## Public Endpoints (No Authentication)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check |
| GET | `/health/ready` | Readiness check |
| GET | `/config/deployment` | Get deployment info |
| GET | `/config/pricing` | Get public pricing info |

## Webhooks

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/webhooks/github` | GitHub webhook |
| POST | `/webhooks/gitlab` | GitLab webhook |
| POST | `/webhooks/gitee` | Gitee webhook |
| POST | `/webhooks/stripe` | Stripe payment webhook |
| POST | `/webhooks/alipay` | Alipay payment webhook |
| POST | `/webhooks/wechat` | WeChat Pay webhook |

---

## gRPC API (Runner Communication)

Runners connect to the backend via gRPC with mTLS (mutual TLS) authentication.

### Endpoint

- Production: `grpcs://your-domain.com:9443`
- Development: `grpcs://localhost:9443`

### Authentication

Runners authenticate using client certificates issued by the AgentsMesh PKI:

1. **Registration**: Runner obtains a certificate via `RegisterWithToken` or browser-based authorization
2. **Connection**: Runner presents certificate during TLS handshake
3. **Validation**: Backend validates certificate and extracts Runner identity from CN

### Service Definition

```protobuf
service RunnerService {
  // Bidirectional streaming for Runner ↔ Backend communication
  rpc Connect(stream RunnerMessage) returns (stream ServerMessage);

  // Certificate registration (before mTLS setup)
  rpc RegisterWithToken(RegisterWithTokenRequest) returns (RegisterWithTokenResponse);
  rpc GetAuthStatus(GetAuthStatusRequest) returns (GetAuthStatusResponse);
}
```

### Message Types

**Server → Runner:**
- `CreatePodCommand`: Create a new pod with specified agent
- `TerminatePodCommand`: Stop and cleanup a pod
- `TerminalInputCommand`: Send input to pod's terminal
- `TerminalResizeCommand`: Resize terminal dimensions
- `PromptCommand`: Send prompt to agent

**Runner → Server:**
- `PodCreatedEvent`: Pod successfully created
- `PodTerminatedEvent`: Pod terminated (with exit code)
- `TerminalOutputEvent`: Terminal output data
- `AgentStatusEvent`: Agent state change
- `HeartbeatData`: Periodic health check

---

## WebSocket Endpoints

### Terminal WebSocket (via Relay)

```
ws://{domain}/relay/terminal/{pod_key}?token=<jwt>
```

Connect to a pod's terminal for real-time input/output via Relay service.

### Events WebSocket

```
ws://{domain}/api/v1/orgs/{slug}/ws/events
```

Subscribe to real-time events (pod status changes, channel messages, etc.).

---

## Error Responses

All errors return JSON with the following structure:

```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable error description"
  }
}
```

Common HTTP status codes:

| Code | Description |
|------|-------------|
| `400` | Bad Request - validation error |
| `401` | Unauthorized - missing or invalid token |
| `402` | Payment Required - quota exceeded |
| `403` | Forbidden - insufficient permissions |
| `404` | Not Found |
| `409` | Conflict - duplicate resource |
| `429` | Too Many Requests - rate limited |
| `500` | Internal Server Error |

## Rate Limiting

API requests are rate-limited. When exceeded, you'll receive:

```
HTTP/1.1 429 Too Many Requests
Retry-After: 60
```
