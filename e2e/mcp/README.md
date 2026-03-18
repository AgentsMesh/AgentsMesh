# MCP E2E Tests

## Overview

This directory contains end-to-end tests for the Model Context Protocol (MCP) collaboration features.

## MCP Tools Coverage

The `TC-MCP-001-full-collaboration-scenario.yaml` test covers all 25 MCP tools:

### Discovery Tools (3)

| Tool | Description | Test Steps |
|------|-------------|------------|
| `list_available_pods` | List available pods for collaboration | step-11 |
| `list_runners` | List available runners | step-2 |
| `list_repositories` | List configured repositories | step-3 |

### Binding Tools (6)

| Tool | Description | Test Steps |
|------|-------------|------------|
| `bind_pod` | Request binding with another pod | step-12, step-14 |
| `accept_binding` | Accept binding request | auto-accept |
| `reject_binding` | Reject binding request | step-15 |
| `get_bindings` | Get binding list | step-13 |
| `get_bound_pods` | Get list of bound pods | step-16, step-41 |
| `unbind_pod` | Unbind from a pod | step-40 |

### Channel Tools (7)

| Tool | Description | Test Steps |
|------|-------------|------------|
| `create_channel` | Create collaboration channel | step-22 |
| `search_channels` | Search channels | step-23 |
| `get_channel` | Get channel details | step-24 |
| `send_channel_message` | Send message to channel | step-27, step-28, step-29 |
| `get_channel_messages` | Get channel messages | step-30 |
| `get_channel_document` | Get shared document | step-32 |
| `update_channel_document` | Update shared document | step-31, step-33 |

### Pod Interaction Tools (2)

| Tool | Description | Test Steps |
|------|-------------|------------|
| `get_pod_snapshot` | Get a snapshot of another pod's terminal | step-17, step-21, step-42 |
| `send_pod_input` | Send text and/or key presses to terminal | step-18, step-19 |

### Ticket Tools (4)

| Tool | Description | Test Steps |
|------|-------------|------------|
| `create_ticket` | Create a ticket | step-34 |
| `search_tickets` | Search tickets | step-35 |
| `get_ticket` | Get ticket details | step-36 |
| `update_ticket` | Update ticket | step-37, step-38 |

### Pod Tools (1)

| Tool | Description | Test Steps |
|------|-------------|------------|
| `create_pod` | Create a new pod | step-5, step-6, step-7 |

### Loop Tools (2)

| Tool | Description | Test Steps |
|------|-------------|------------|
| `list_loops` | List automated loops in the organization | - |
| `trigger_loop` | Manually trigger a loop run | - |

## Test Scenario Flow

```
Phase 1: Authentication
    └── Login and get token

Phase 2: Discovery
    ├── List runners
    ├── List repositories
    └── Get agent types

Phase 3: Pod Creation
    ├── Create Pod A (Controller)
    ├── Create Pod B (Collaborator)
    ├── Create Pod C (Reject Test)
    └── Wait for all pods to be running

Phase 4: Binding
    ├── Pod A binds to Pod B (accept)
    ├── Pod A binds to Pod C (reject)
    └── Verify binding states

Phase 5: Terminal Control
    ├── Pod A gets Pod B terminal snapshot
    ├── Pod A sends text and Enter key to Pod B
    └── Verify command execution

Phase 6: Channel Collaboration
    ├── Create channel
    ├── Both pods join channel
    ├── Exchange messages with @mentions
    ├── Update shared document
    └── Verify document sync

Phase 7: Ticket Management
    ├── Create ticket
    ├── Search and get ticket
    └── Update ticket status (backlog → in_progress → done)

Phase 8: Unbinding
    ├── Pod A unbinds from Pod B
    └── Verify permission revoked (get_pod_snapshot fails with 403)

Phase 9: Cleanup
    ├── Archive channel
    ├── Terminate all pods
    └── Verify final state
```

## Running the Test

```bash
# Using the E2E test runner
/e2e e2e/mcp/TC-MCP-001-full-collaboration-scenario.yaml
```

## Prerequisites

- Development environment running (`cd deploy/dev && docker compose up -d`)
- Test user exists (`dev@agentsmesh.local / devpass123`)
- At least one online Runner with MCP enabled
