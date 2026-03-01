# E2E Test Cases

This directory contains end-to-end test cases for AgentsMesh, written in structured YAML format and executable by Claude Code or other automated testing tools.

## Directory Structure

```
e2e/
├── README.md                          # This file
├── billing/                           # Billing module
│   ├── subscription/                  # Subscription management
│   │   ├── TC-SUB-001-status-display.yaml
│   │   ├── TC-SUB-002-plans-dialog.yaml
│   │   ├── TC-SUB-003-cancel-at-period-end.yaml
│   │   ├── TC-SUB-004-cancel-immediately.yaml
│   │   ├── TC-SUB-005-reactivate.yaml
│   │   └── TC-SUB-006-plan-upgrade.yaml
│   ├── seats/                         # Seat management
│   │   ├── TC-SEAT-001-display.yaml
│   │   ├── TC-SEAT-002-add-dialog.yaml
│   │   └── TC-SEAT-003-based-plan-limit.yaml
│   ├── billing-cycle/                 # Billing cycle
│   │   ├── TC-CYCLE-001-display.yaml
│   │   ├── TC-CYCLE-002-monthly-to-yearly.yaml
│   │   └── TC-CYCLE-003-yearly-to-monthly.yaml
│   ├── promo-code/                    # Promo codes
│   │   ├── TC-PROMO-001-display.yaml
│   │   ├── TC-PROMO-002-valid-code.yaml
│   │   └── TC-PROMO-003-invalid-code.yaml
│   └── quota/                         # Quota checks
│       ├── TC-QUOTA-001-users.yaml
│       ├── TC-QUOTA-002-runners.yaml
│       └── TC-QUOTA-003-repositories.yaml
├── extensions/                        # Extensions module (Skills + MCP Servers)
│   ├── TC-EXT-001-full-capabilities-flow.yaml  # Full capabilities management E2E flow
│   ├── skills/                        # Skills tests
│   │   ├── TC-SKILL-001-capabilities-tab-display.yaml  # Capabilities Tab Skills display
│   │   ├── TC-SKILL-002-add-skill-dialog.yaml          # Add Skill dialog
│   │   ├── TC-SKILL-003-install-from-market.yaml       # Install from Marketplace
│   │   ├── TC-SKILL-004-install-from-github.yaml       # Install from GitHub URL
│   │   ├── TC-SKILL-005-toggle-and-uninstall.yaml      # Enable/disable and uninstall
│   │   ├── TC-SKILL-006-source-link.yaml               # Source link display
│   │   └── TC-SKILL-007-api-crud.yaml                  # API CRUD flow
│   ├── mcp/                           # MCP Servers tests
│   │   ├── TC-MCP-EXT-001-mcp-tab-display.yaml         # MCP Tab display
│   │   ├── TC-MCP-EXT-002-add-mcp-dialog.yaml          # Add MCP dialog
│   │   ├── TC-MCP-EXT-003-install-from-market.yaml     # Install from Market (no env vars)
│   │   ├── TC-MCP-EXT-004-install-from-market-with-env.yaml # Install from Market (with env vars)
│   │   ├── TC-MCP-EXT-005-install-custom.yaml          # Install custom MCP Server
│   │   ├── TC-MCP-EXT-006-edit-env-vars.yaml           # Edit environment variables
│   │   ├── TC-MCP-EXT-007-toggle-and-uninstall.yaml    # Enable/disable and uninstall
│   │   ├── TC-MCP-EXT-008-source-link.yaml             # Source link display
│   │   └── TC-MCP-EXT-009-api-crud.yaml                # API CRUD flow
│   └── settings/                      # Organization Settings Extensions management
│       ├── TC-EXTSET-001-extensions-page.yaml           # Extensions page display
│       ├── TC-EXTSET-002-skill-registries.yaml          # Skill Registries management
│       └── TC-EXTSET-003-mcp-templates.yaml             # MCP Templates browsing
└── runner/                            # Runner management module
    ├── list/                          # Runner list
    │   ├── TC-RUNNER-001-list-all.yaml       # List all Runners
    │   ├── TC-RUNNER-002-list-available.yaml # List available Runners
    │   └── TC-RUNNER-003-get-single.yaml     # Get single Runner
    ├── tokens/                        # Registration token management
    │   ├── TC-TOKEN-001-list.yaml            # List registration tokens
    │   ├── TC-TOKEN-002-create.yaml          # Create registration token
    │   ├── TC-TOKEN-003-revoke.yaml          # Revoke registration token
    │   └── TC-TOKEN-004-full-crud-flow.yaml  # Full CRUD flow
    ├── config/                        # Runner configuration
    │   ├── TC-CONFIG-001-update.yaml         # Update Runner configuration
    │   └── TC-CONFIG-002-disable-enable.yaml # Disable/enable Runner
    ├── delete/                        # Runner deletion
    │   └── TC-DELETE-001-basic.yaml          # Delete Runner
    ├── grpc-tokens/                   # gRPC registration tokens
    │   ├── TC-GRPC-001-list.yaml             # List gRPC tokens
    │   ├── TC-GRPC-002-generate.yaml         # Generate gRPC token
    │   ├── TC-GRPC-003-delete.yaml           # Delete gRPC token
    │   └── TC-GRPC-004-full-crud-flow.yaml   # Full CRUD flow
    ├── ui/                            # UI page tests
    │   ├── TC-UI-001-list-page.yaml          # Runner list page
    │   ├── TC-UI-002-add-runner-dialog.yaml  # Add Runner dialog
    │   ├── TC-UI-003-runner-config-dialog.yaml # Config dialog
    │   ├── TC-UI-004-delete-confirmation.yaml  # Delete confirmation
    │   └── TC-UI-005-full-management-flow.yaml # Full management flow
    ├── admin/                         # Admin Runner management
    │   ├── TC-ADMIN-001-list.yaml            # Admin list all Runners
    │   ├── TC-ADMIN-002-get-single.yaml      # Admin get single Runner
    │   ├── TC-ADMIN-003-disable-enable.yaml  # Admin disable/enable
    │   ├── TC-ADMIN-004-delete.yaml          # Admin delete
    │   └── TC-ADMIN-005-full-management-flow.yaml # Admin full flow
    └── registration/                  # Runner registration and Pod creation
        ├── TC-REG-001-multi-runner-registration.yaml  # Multi-runner registration full flow
        ├── TC-REG-002-runner-online-status.yaml       # Runner online status verification
        └── TC-REG-003-pod-creation-flow.yaml          # Pod creation full flow
```

## Test Case Format

```yaml
id: TC-XXX-001
name: Test case name
description: Test case description
priority: critical | high | medium | low
must_execute: true  # 🚨 Must be set to true for UI tests
module: billing/subscription
tags:
  - ui          # Mark as UI test
  - mcp-required  # Mark as requiring MCP Chrome DevTools

preconditions:
  - Precondition description

setup:
  sql: |
    -- Optional database initialization SQL

steps:
  - action: Action description
    expected: Expected result
    verification:
      type: ui | api | database
      details: Verification details

cleanup:
  - sql: |
      -- Cleanup SQL
```

### 🚨 UI Test Enforcement Rules

UI tests (`verification.type: ui`) are the core of E2E testing and **must not be skipped**:

- `priority: critical` - UI tests must be set to the highest priority
- `must_execute: true` - Mark as required execution
- `tags: [ui, mcp-required]` - Mark as requiring MCP Chrome DevTools

**When executing UI tests:**
1. Must use MCP Chrome DevTools tools
2. Do not substitute API calls for browser verification
3. If MCP is unavailable, report the issue rather than skipping the test

## Running Tests

### Using Claude Code

```
Please execute the e2e/billing/subscription/TC-SUB-001-status-display.yaml test case
```

Or execute an entire module:

```
Please execute all test cases under the e2e/billing/subscription/ directory
```

### Verification Types

| Type | Description | Example |
|------|-------------|---------|
| `ui` | Browser snapshot verification | Check page elements, text, button states |
| `api` | API call verification | curl requests, validate status codes and responses |
| `database` | Database query verification | psql SQL execution, validate data state |

## Test Data

| Data | Value |
|------|-------|
| Test user email | dev@agentsmesh.local |
| Test user password | devpass123 |
| Admin user email | admin@agentsmesh.local |
| Admin user password | adminpass123 |
| Test organization slug | dev-org |
| Default subscription plan | pro |
| Billing page path | /dev-org/settings?scope=organization&tab=billing |
| Runner management page path | /dev-org/runners |

## Runner Module Test Coverage

Runner E2E tests cover the following functionality:

### API Tests

| Endpoint | Test Case | Description |
|----------|-----------|-------------|
| `GET /orgs/:slug/runners` | TC-RUNNER-001 | List all Runners in organization |
| `GET /orgs/:slug/runners/available` | TC-RUNNER-002 | List available Runners |
| `GET /orgs/:slug/runners/:id` | TC-RUNNER-003 | Get single Runner |
| `PUT /orgs/:slug/runners/:id` | TC-CONFIG-001/002 | Update Runner config, disable/enable |
| `DELETE /orgs/:slug/runners/:id` | TC-DELETE-001 | Delete Runner |
| `GET /orgs/:slug/runners/tokens` | TC-TOKEN-001 | List registration tokens |
| `POST /orgs/:slug/runners/tokens` | TC-TOKEN-002 | Create registration token |
| `DELETE /orgs/:slug/runners/tokens/:id` | TC-TOKEN-003 | Revoke registration token |
| `GET /orgs/:slug/runners/grpc/tokens` | TC-GRPC-001 | List gRPC tokens |
| `POST /orgs/:slug/runners/grpc/tokens` | TC-GRPC-002 | Generate gRPC token |
| `DELETE /orgs/:slug/runners/grpc/tokens/:id` | TC-GRPC-003 | Delete gRPC token |

### Admin API Tests

| Endpoint | Test Case | Description |
|----------|-----------|-------------|
| `GET /api/v1/admin/runners` | TC-ADMIN-001 | Admin list all Runners |
| `GET /api/v1/admin/runners/:id` | TC-ADMIN-002 | Admin get single Runner |
| `POST /api/v1/admin/runners/:id/disable` | TC-ADMIN-003 | Admin disable Runner |
| `POST /api/v1/admin/runners/:id/enable` | TC-ADMIN-003 | Admin enable Runner |
| `DELETE /api/v1/admin/runners/:id` | TC-ADMIN-004 | Admin delete Runner |

### UI Tests

| Page/Feature | Test Case | Description |
|--------------|-----------|-------------|
| Runner list page | TC-UI-001 | Page display and status statistics |
| Add Runner dialog | TC-UI-002 | Registration command and token generation |
| Runner config dialog | TC-UI-003 | Config editing and saving |
| Delete confirmation dialog | TC-UI-004 | Delete confirmation flow |
| Full management flow | TC-UI-005 | End-to-end management operations |

### Multi-Runner Registration & Pod Creation Tests

| Test Case | Description | Verification Type |
|-----------|-------------|-------------------|
| TC-REG-001 | Multi-runner registration full flow | UI + Docker + API + DB |
| TC-REG-002 | Runner online status verification | API + DB |
| TC-REG-003 | Pod creation full flow | UI + API + DB |

#### TC-REG-001 Test Flow

1. **Generate registration tokens via UI** - Generate multiple gRPC registration tokens on the Runner management page
2. **Start Docker Runners** - Start multiple Runner containers with tokens and register
3. **Verify Runners online** - Confirm multiple Runners are simultaneously showing "online" status
4. **Create Pod** - Create a Pod from one Runner
5. **Verify Pod running** - Confirm Pod enters running state with terminal available
6. **Cleanup resources** - Stop containers and clean up database

#### Execution Requirements

- Docker environment required
- MCP Chrome DevTools required (UI verification)
- Runner containers must be able to access the backend and nginx services (same Docker network)

## Extensions Module Test Coverage

Extensions E2E tests cover the full capabilities management functionality for Skills and MCP Servers.

### Test Data

| Data | Value |
|------|-------|
| Extensions settings page path | /dev-org/settings?scope=organization&tab=extensions |
| Repository page path | /dev-org/repositories → click Demo WebApp → Extensions Tab |
| MCP Market seed data | jira, postgres, slack, github, filesystem, memory |

### Skills UI Tests

| Page/Feature | Test Case | Description |
|--------------|-----------|-------------|
| Capabilities Tab display | TC-SKILL-001 | Skills sub-tab, org/user scope sections, empty state |
| Add Skill dialog | TC-SKILL-002 | Three installation method tabs (Marketplace/GitHub/Upload) |
| Marketplace installation | TC-SKILL-003 | Search, install, list update, installed indicator |
| GitHub URL installation | TC-SKILL-004 | Fill URL/Branch/Path to import |
| Enable/disable and uninstall | TC-SKILL-005 | Switch toggle, confirmation dialog, uninstall flow |
| Source link | TC-SKILL-006 | External link icon, source_url link |
| API CRUD | TC-SKILL-007 | Skills API full CRUD flow |

### MCP Servers UI Tests

| Page/Feature | Test Case | Description |
|--------------|-----------|-------------|
| MCP Tab display | TC-MCP-EXT-001 | MCP sub-tab, org/user scope sections, empty state |
| Add MCP dialog | TC-MCP-EXT-002 | Market Templates/Custom Tab, template list, search |
| Market install (no env) | TC-MCP-EXT-003 | Select Filesystem template and install directly |
| Market install (with env) | TC-MCP-EXT-004 | Select Jira template, fill required env vars, Change switch |
| Custom install | TC-MCP-EXT-005 | stdio type custom config, add environment variables |
| Edit env vars | TC-MCP-EXT-006 | Settings gear button, schema mode, free edit mode |
| Enable/disable and uninstall | TC-MCP-EXT-007 | Switch toggle, confirmation dialog, uninstall flow |
| Source link | TC-MCP-EXT-008 | Market tag, repository_url external link |
| API CRUD | TC-MCP-EXT-009 | MCP Server API full CRUD flow |

### Organization Settings Tests

| Page/Feature | Test Case | Description |
|--------------|-----------|-------------|
| Extensions page | TC-EXTSET-001 | Page title, dual tab display |
| Skill Registries | TC-EXTSET-002 | Platform/org registry, add dialog, auth options |
| MCP Templates | TC-EXTSET-003 | Template list, search, categories, count statistics |

### End-to-End Flow Tests

| Test Case | Description | Verification Type |
|-----------|-------------|-------------------|
| TC-EXT-001 | Full capabilities management flow (Settings → Repo → Install → Edit → Toggle → Uninstall) | UI + API |
