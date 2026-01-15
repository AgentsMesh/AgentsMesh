# AgentMesh API Documentation

## Overview

AgentMesh provides a RESTful API for managing multi-agent AI development workspaces. The API supports multi-tenancy, OAuth authentication, and real-time WebSocket connections.

## Base URL

- Production: `https://api.agentmesh.io/api/v1`
- Staging: `https://staging-api.agentmesh.example.com/api/v1`
- Development: `http://localhost:8080/api/v1`

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

## Multi-Tenancy

Organization-scoped endpoints require the organization slug in the URL path:

```
/api/v1/organizations/{slug}/...
```

## API Endpoints

### Authentication

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/auth/login` | Email/password login |
| POST | `/auth/register` | User registration |
| POST | `/auth/refresh` | Refresh JWT token |
| POST | `/auth/logout` | Logout and revoke token |
| GET | `/auth/oauth/{provider}` | OAuth redirect |
| GET | `/auth/oauth/{provider}/callback` | OAuth callback |

### Users

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/users/me` | Get current user profile |
| PUT | `/users/me` | Update profile |
| POST | `/users/me/password` | Change password |
| GET | `/users/me/organizations` | List user's organizations |
| GET | `/users/me/identities` | List OAuth identities |
| DELETE | `/users/me/identities/{provider}` | Remove OAuth identity |
| GET | `/users/search?q=` | Search users |

### Organizations

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/organizations` | List user's organizations |
| POST | `/organizations` | Create organization |
| GET | `/organizations/{slug}` | Get organization |
| PUT | `/organizations/{slug}` | Update organization |
| DELETE | `/organizations/{slug}` | Delete organization |
| GET | `/organizations/{slug}/members` | List members |
| POST | `/organizations/{slug}/members` | Invite member |
| PUT | `/organizations/{slug}/members/{user_id}` | Update member role |
| DELETE | `/organizations/{slug}/members/{user_id}` | Remove member |

### Teams

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/organizations/{slug}/teams` | List teams |
| POST | `/organizations/{slug}/teams` | Create team |
| GET | `/organizations/{slug}/teams/{id}` | Get team |
| PUT | `/organizations/{slug}/teams/{id}` | Update team |
| DELETE | `/organizations/{slug}/teams/{id}` | Delete team |
| GET | `/organizations/{slug}/teams/{id}/members` | List team members |
| POST | `/organizations/{slug}/teams/{id}/members` | Add team member |
| DELETE | `/organizations/{slug}/teams/{id}/members/{user_id}` | Remove team member |

### Code Agents

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/organizations/{slug}/agents/types` | List agent types |
| GET | `/organizations/{slug}/agents/{type_id}/config-schema` | Get agent config schema |
| POST | `/organizations/{slug}/agents/custom` | Create custom agent |
| PUT | `/organizations/{slug}/agents/custom/{id}` | Update custom agent |
| DELETE | `/organizations/{slug}/agents/custom/{id}` | Delete custom agent |

### User Agent Configuration (Personal Settings)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/users/me/agents/credentials` | Get user credentials status |
| PUT | `/users/me/agents/credentials/{type_id}` | Set user credentials |
| DELETE | `/users/me/agents/credentials/{type_id}` | Delete user credentials |
| GET | `/users/me/agent-configs` | List user agent configs |
| GET | `/users/me/agent-configs/{type_id}` | Get user agent config |
| PUT | `/users/me/agent-configs/{type_id}` | Set user agent config |
| DELETE | `/users/me/agent-configs/{type_id}` | Delete user agent config |

### Git Providers

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/organizations/{slug}/git-providers` | List providers |
| POST | `/organizations/{slug}/git-providers` | Create provider |
| GET | `/organizations/{slug}/git-providers/{id}` | Get provider |
| PUT | `/organizations/{slug}/git-providers/{id}` | Update provider |
| DELETE | `/organizations/{slug}/git-providers/{id}` | Delete provider |
| POST | `/organizations/{slug}/git-providers/{id}/test` | Test connection |
| POST | `/organizations/{slug}/git-providers/{id}/sync` | Sync projects |

### Repositories

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/organizations/{slug}/repositories` | List repositories |
| POST | `/organizations/{slug}/repositories` | Create repository |
| GET | `/organizations/{slug}/repositories/{id}` | Get repository |
| PUT | `/organizations/{slug}/repositories/{id}` | Update repository |
| DELETE | `/organizations/{slug}/repositories/{id}` | Delete repository |
| GET | `/organizations/{slug}/repositories/{id}/branches` | List branches |
| POST | `/organizations/{slug}/repositories/{id}/sync-branches` | Sync branches |
| POST | `/organizations/{slug}/repositories/{id}/webhook` | Setup webhook |

### Runners

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/organizations/{slug}/runners` | List runners |
| GET | `/organizations/{slug}/runners/{id}` | Get runner |
| DELETE | `/organizations/{slug}/runners/{id}` | Delete runner |
| GET | `/organizations/{slug}/runners/tokens` | List registration tokens |
| POST | `/organizations/{slug}/runners/tokens` | Create registration token |
| DELETE | `/organizations/{slug}/runners/tokens/{id}` | Revoke token |
| POST | `/runners/register` | Register runner (public) |
| POST | `/runners/heartbeat` | Runner heartbeat (public) |

### Pods

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/organizations/{slug}/pods` | List pods |
| POST | `/organizations/{slug}/pods` | Create pod |
| GET | `/organizations/{slug}/pods/{key}` | Get pod |
| POST | `/organizations/{slug}/pods/{key}/terminate` | Terminate pod |
| GET | `/organizations/{slug}/pods/{key}/connect` | Get connection info |
| POST | `/organizations/{slug}/pods/{key}/send-prompt` | Send prompt |

### Channels

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/organizations/{slug}/channels` | List channels |
| POST | `/organizations/{slug}/channels` | Create channel |
| GET | `/organizations/{slug}/channels/{id}` | Get channel |
| PUT | `/organizations/{slug}/channels/{id}` | Update channel |
| POST | `/organizations/{slug}/channels/{id}/archive` | Archive channel |
| POST | `/organizations/{slug}/channels/{id}/unarchive` | Unarchive channel |
| GET | `/organizations/{slug}/channels/{id}/messages` | List messages |
| POST | `/organizations/{slug}/channels/{id}/messages` | Send message |
| POST | `/organizations/{slug}/channels/{id}/pods` | Join pod |
| DELETE | `/organizations/{slug}/channels/{id}/pods/{key}` | Leave pod |

### Tickets

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/organizations/{slug}/tickets` | List tickets |
| POST | `/organizations/{slug}/tickets` | Create ticket |
| GET | `/organizations/{slug}/tickets/{identifier}` | Get ticket |
| PUT | `/organizations/{slug}/tickets/{identifier}` | Update ticket |
| DELETE | `/organizations/{slug}/tickets/{identifier}` | Delete ticket |
| POST | `/organizations/{slug}/tickets/{identifier}/assignees` | Add assignee |
| DELETE | `/organizations/{slug}/tickets/{identifier}/assignees/{user_id}` | Remove assignee |
| POST | `/organizations/{slug}/tickets/{identifier}/labels` | Add label |
| DELETE | `/organizations/{slug}/tickets/{identifier}/labels/{label_id}` | Remove label |
| GET | `/organizations/{slug}/tickets/{identifier}/merge-requests` | List MRs |

### Labels

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/organizations/{slug}/labels` | List labels |
| POST | `/organizations/{slug}/labels` | Create label |
| PUT | `/organizations/{slug}/labels/{id}` | Update label |
| DELETE | `/organizations/{slug}/labels/{id}` | Delete label |

### Billing

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/organizations/{slug}/billing/overview` | Get billing overview |
| GET | `/organizations/{slug}/billing/subscription` | Get subscription |
| POST | `/organizations/{slug}/billing/subscription` | Create subscription |
| PUT | `/organizations/{slug}/billing/subscription` | Update subscription |
| DELETE | `/organizations/{slug}/billing/subscription` | Cancel subscription |
| GET | `/organizations/{slug}/billing/plans` | List plans |
| GET | `/organizations/{slug}/billing/usage` | Get usage |
| GET | `/organizations/{slug}/billing/usage/history` | Get usage history |
| POST | `/organizations/{slug}/billing/quota` | Set custom quota |
| GET | `/organizations/{slug}/billing/quota/check` | Check quota |

## WebSocket Endpoints

### Terminal WebSocket

```
ws://localhost:8080/ws/terminal/{pod_key}
```

Connect to a pod's terminal. Requires JWT authentication via query parameter:

```
ws://localhost:8080/ws/terminal/{pod_key}?token=<jwt>
```

### Events WebSocket

```
ws://localhost:8080/ws/events
```

Subscribe to real-time events. Supports filtering by event type.

## Error Responses

All errors return JSON with the following structure:

```json
{
  "error": "Error message"
}
```

Common HTTP status codes:
- `400` - Bad Request (validation error)
- `401` - Unauthorized (missing or invalid token)
- `403` - Forbidden (insufficient permissions)
- `404` - Not Found
- `409` - Conflict (duplicate resource)
- `500` - Internal Server Error

## Rate Limiting

API requests are rate-limited. When exceeded, you'll receive:

```
HTTP/1.1 429 Too Many Requests
Retry-After: 60
```
