# Proto Migration Runbook (Per-Service)

Step-by-step for a specialist agent migrating **one service** from REST + hand-written DTOs to Connect-RPC + `.proto`. Companion to ADR / conventions / watch list.

This is the **only** document a specialist agent needs at the keyboard.

---

## Inputs

The orchestrator dispatches with:

- `{service_name}` — domain slug, lowercase (e.g., `extension`, `pod`, `billing`).
- `{rest_handler_paths}` — Go files under `backend/internal/api/rest/v1/` for this service.
- `{rust_dto_path}` — `clients/core/crates/types/src/<service>.rs`.
- `{rust_api_client_path}` — `clients/core/crates/api-client/src/modules/<service>.rs`.
- `{rust_wasm_service_path}` — `clients/core/crates/wasm/src/service_<service>.rs`.
- `{ts_types_path}` — `clients/web/src/lib/api/<service>Types.ts` (may not exist).
- `{historical_drift_prs}` — comma-separated PR numbers that previously touched this DTO.

---

## Step 1 — Predicate phase (read-only)

**Goal**: enumerate every field the service uses today across all four layers, reconcile drift, decide the `.proto` field set **before** writing any code.

### 1.1 Endpoints

```bash
grep -nE '(GET|POST|PUT|PATCH|DELETE).*"[^"]+"' backend/internal/api/rest/v1/<service>*.go
```

Record `{verb, path, handler, middleware}` per endpoint.

### 1.2 Response shapes

```bash
grep -nA 6 'c.JSON' backend/internal/api/rest/v1/<service>*.go
```

Each list endpoint maps to `{items, total, limit, offset}`. Each single-entity endpoint maps to the entity directly.

### 1.3 Rust DTO + api-client + wasm

```bash
cat clients/core/crates/types/src/<service>.rs
cat clients/core/crates/api-client/src/modules/<service>.rs
cat clients/core/crates/wasm/src/service_<service>.rs
```

Note every field, `Option<>`, `#[serde(...)]` annotation, every `get/post_resource` wrapper-key argument, every wasm-bindgen export name.

### 1.4 TS consumers

```bash
grep -rn "<DtoName>\|<entity>Service\|use<Service>" clients/web/src/ | grep -v ".test." | head -30
```

For each call site: what fields read, what shape destructured.

### 1.5 Three-way diff (drift signal)

Compare the field sets from §1.2 (Go handler), §1.3 (Rust DTO), §1.4 (TS reader). **Anything in two sources but not the third = drift bug — fix in this PR**. Note in PR description.

### 1.6 Historical PRs

```bash
for n in <historical_drift_prs>; do gh pr view $n --json title,body | jq -r '.title, .body' | head -60; done
```

Look for `#[serde(alias = "...")]` or other workarounds — remove them, align on `.proto`.

---

## Step 2 — Write the `.proto`

Create `proto/<service>/v1/<service>.proto`. **Follow conventions.md religiously** (§1 package, §2 service/method, §3 snake_case fields, §4 stable tags, §5 `optional`, §6 string timestamps, §7 oneof avoid, §8 list envelope, §9 entity-direct).

### Template

```protobuf
syntax = "proto3";

package proto.<service>.v1;

option go_package = "github.com/anthropics/agentsmesh/proto/gen/go/<service>/v1;<service>v1";

service <Service>Service {
  rpc List<Service>s(List<Service>sRequest) returns (List<Service>sResponse);
  rpc Get<Service>(Get<Service>Request) returns (<Service>);
  rpc Create<Service>(Create<Service>Request) returns (<Service>);
  rpc Update<Service>(Update<Service>Request) returns (<Service>);
  rpc Delete<Service>(Delete<Service>Request) returns (Delete<Service>Response);
}

message <Service> {
  int64 id = 1;
  string name = 2;
  optional string description = 3;
  string created_at = 4;            // RFC 3339
  optional string updated_at = 5;
  // every field from §1.3 reconciled
}

message List<Service>sRequest {
  optional int32 offset = 1;
  optional int32 limit = 2;
}

message List<Service>sResponse {
  repeated <Service> items = 1;
  int64 total = 2;
  int32 limit = 3;
  int32 offset = 4;
}

message Get<Service>Request    { int64 id = 1; }
message Create<Service>Request { string name = 1; optional string description = 2; }
message Update<Service>Request { int64 id = 1; optional string name = 2; optional string description = 3; }
message Delete<Service>Request { int64 id = 1; }
message Delete<Service>Response {}
```

### BUILD.bazel

```python
load("@rules_go//proto:def.bzl", "go_proto_library")
load("@rules_proto//proto:defs.bzl", "proto_library")

proto_library(
    name = "<service>_proto",
    srcs = ["<service>.proto"],
    visibility = ["//visibility:public"],
)

go_proto_library(
    name = "<service>_go_proto",
    compilers = ["@rules_go//proto:go_proto"],
    importpath = "github.com/anthropics/agentsmesh/proto/gen/go/<service>/v1",
    proto = ":<service>_proto",
    visibility = ["//visibility:public"],
)
```

### Verify

```bash
bazel build //proto/<service>/v1:<service>_proto //proto/<service>/v1:<service>_go_proto
```

---

## Step 3 — Go Connect handler

### 3.1 Handler — `backend/internal/api/connect/<service>/<service>.go`

```go
package <service>connect

import (
    "context"
    "errors"
    "net/http"
    "time"

    "connectrpc.com/connect"
    authinterceptor "github.com/anthropics/agentsmesh/backend/internal/api/connect/interceptor"
    <service>v1 "github.com/anthropics/agentsmesh/proto/gen/go/<service>/v1"
    <service>svc "github.com/anthropics/agentsmesh/backend/internal/service/<service>"
)

const ServiceName = "proto.<service>.v1.<Service>Service"

const (
    List<Service>sProcedure  = "/" + ServiceName + "/List<Service>s"
    Get<Service>Procedure    = "/" + ServiceName + "/Get<Service>"
    Create<Service>Procedure = "/" + ServiceName + "/Create<Service>"
    Update<Service>Procedure = "/" + ServiceName + "/Update<Service>"
    Delete<Service>Procedure = "/" + ServiceName + "/Delete<Service>"
)

type Server struct{ svc *<service>svc.Service }

func New(svc *<service>svc.Service) *Server { return &Server{svc: svc} }

func (s *Server) List<Service>s(
    ctx context.Context, req *connect.Request[<service>v1.List<Service>sRequest],
) (*connect.Response[<service>v1.List<Service>sResponse], error) {
    userID, err := authinterceptor.UserID(ctx)
    if err != nil { return nil, connect.NewError(connect.CodeUnauthenticated, err) }

    limit, offset := int(req.Msg.GetLimit()), int(req.Msg.GetOffset())
    if limit == 0 { limit = 20 }

    items, total, err := s.svc.List(ctx, userID, limit, offset)
    if err != nil { return nil, connect.NewError(connect.CodeInternal, err) }

    out := make([]*<service>v1.<Service>, 0, len(items))
    for _, it := range items { out = append(out, toProto(it)) }
    return connect.NewResponse(&<service>v1.List<Service>sResponse{
        Items: out, Total: int64(total), Limit: int32(limit), Offset: int32(offset),
    }), nil
}

func (s *Server) Create<Service>(
    ctx context.Context, req *connect.Request[<service>v1.Create<Service>Request],
) (*connect.Response[<service>v1.<Service>], error) {
    userID, err := authinterceptor.UserID(ctx)
    if err != nil { return nil, connect.NewError(connect.CodeUnauthenticated, err) }
    entity, err := s.svc.Create(ctx, userID, &<service>svc.CreateRequest{
        Name: req.Msg.GetName(), Description: req.Msg.GetDescription(),
    })
    if err != nil {
        if errors.Is(err, <service>svc.ErrAlreadyExists) { return nil, connect.NewError(connect.CodeAlreadyExists, err) }
        return nil, connect.NewError(connect.CodeInternal, err)
    }
    return connect.NewResponse(toProto(entity)), nil
}

// Get/Update/Delete similar — see conventions §10 for full error-code mapping.

func toProto(it *<service>svc.Entity) *<service>v1.<Service> {
    return &<service>v1.<Service>{
        Id: it.ID, Name: it.Name,
        CreatedAt: it.CreatedAt.Format(time.RFC3339),
        // map every field from the .proto Message
    }
}

func Mount(mux *http.ServeMux, s *Server) {
    opts := []connect.HandlerOption{connect.WithInterceptors(authinterceptor.AuthInterceptor())}
    mux.Handle(List<Service>sProcedure,  connect.NewUnaryHandler(List<Service>sProcedure,  s.List<Service>s,  opts...))
    mux.Handle(Get<Service>Procedure,    connect.NewUnaryHandler(Get<Service>Procedure,    s.Get<Service>,    opts...))
    mux.Handle(Create<Service>Procedure, connect.NewUnaryHandler(Create<Service>Procedure, s.Create<Service>, opts...))
    mux.Handle(Update<Service>Procedure, connect.NewUnaryHandler(Update<Service>Procedure, s.Update<Service>, opts...))
    mux.Handle(Delete<Service>Procedure, connect.NewUnaryHandler(Delete<Service>Procedure, s.Delete<Service>, opts...))
}
```

### 3.2 BUILD + mount in `connect_init.go`

`backend/internal/api/connect/<service>/BUILD.bazel`:

```python
load("@rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "<service>",
    srcs = ["<service>.go"],
    importpath = "github.com/anthropics/agentsmesh/backend/internal/api/connect/<service>",
    visibility = ["//backend:__subpackages__"],
    deps = [
        "//proto/<service>/v1:<service>_go_proto",
        "//backend/internal/api/connect/interceptor",
        "//backend/internal/service/<service>",
        "@com_connectrpc_connect//:connect",
    ],
)

go_test(name = "<service>_test", srcs = ["<service>_test.go"], embed = [":<service>"], deps = [
    "@com_connectrpc_connect//:connect", "@com_github_stretchr_testify//require",
])
```

Edit `backend/cmd/server/connect_init.go` to mount the service onto `connectMux` via `<service>connect.Mount(connectMux, <service>connect.New(deps.<Service>Service))`.

### 3.3 Keep REST mounted (dual-track)

**Do not delete** `backend/internal/api/rest/v1/<service>*.go`. Removal is a separate cleanup PR after the wasm + TS clients are confirmed in production on the Connect lane.

### 3.4 Verify

```bash
bazel build //backend/internal/api/connect/<service>:<service> //backend/cmd/server:server
bazel test  //backend/internal/api/connect/<service>:<service>_test
```

---

## Step 4 — Rust DTO + api-client

### 4.1 DTO — `clients/core/crates/types/src/<service>.rs` (REPLACE)

```rust
use prost::Message;
use serde::{Deserialize, Serialize};

#[derive(Clone, PartialEq, Message, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct <Service> {
    #[prost(int64,  tag = "1")] pub id: i64,
    #[prost(string, tag = "2")] pub name: String,
    #[prost(string, optional, tag = "3")] #[serde(skip_serializing_if = "Option::is_none")] pub description: Option<String>,
    #[prost(string, tag = "4")] pub created_at: String,
    #[prost(string, optional, tag = "5")] #[serde(skip_serializing_if = "Option::is_none")] pub updated_at: Option<String>,
}

#[derive(Clone, PartialEq, Message, Serialize, Deserialize, Default)]
#[serde(rename_all = "camelCase")]
pub struct List<Service>sRequest {
    #[prost(int32, optional, tag = "1")] #[serde(skip_serializing_if = "Option::is_none")] pub offset: Option<i32>,
    #[prost(int32, optional, tag = "2")] #[serde(skip_serializing_if = "Option::is_none")] pub limit:  Option<i32>,
}

#[derive(Clone, PartialEq, Message, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct List<Service>sResponse {
    #[prost(message, repeated, tag = "1")] pub items: Vec<<Service>>,
    #[prost(int64,   tag = "2")] pub total: i64,
    #[prost(int32,   tag = "3")] pub limit: i32,
    #[prost(int32,   tag = "4")] pub offset: i32,
}

#[derive(Clone, PartialEq, Message, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Create<Service>Request {
    #[prost(string, tag = "1")] pub name: String,
    #[prost(string, optional, tag = "2")] #[serde(skip_serializing_if = "Option::is_none")] pub description: Option<String>,
}

// Get/Update/Delete requests analogous.
```

### 4.2 api-client — `clients/core/crates/api-client/src/modules/<service>.rs`

```rust
use crate::ApiClient;
use crate::connect_call::connect_call;
use crate::error::ApiError;
use agentsmesh_types::*;

impl ApiClient {
    pub async fn list_<service>s(&self, req: &List<Service>sRequest) -> Result<List<Service>sResponse, ApiError> {
        connect_call(self, "/proto.<service>.v1.<Service>Service/List<Service>s", req).await
    }
    pub async fn create_<service>(&self, req: &Create<Service>Request) -> Result<<Service>, ApiError> {
        connect_call(self, "/proto.<service>.v1.<Service>Service/Create<Service>", req).await
    }
    // get/update/delete similar
}
```

`connect_call` helper (lives in `clients/core/crates/api-client/src/connect_call.rs`, added by infra or first-migration PR):

```rust
pub async fn connect_call<Req: Serialize, Res: DeserializeOwned>(
    client: &ApiClient, procedure: &str, body: &Req,
) -> Result<Res, ApiError> {
    let url = format!("{}{}", client.base_url(), procedure);
    let mut b = client.http.post(&url).json(body)
        .header("Connect-Protocol-Version", "1")
        .header("Content-Type", "application/json");
    if let Some(tok) = client.token().await { b = b.header("Authorization", format!("Bearer {tok}")); }
    let resp = b.send().await.map_err(ApiError::Http)?;
    if !resp.status().is_success() { return Err(ApiError::from_connect_response(resp).await); }
    Ok(resp.json::<Res>().await.map_err(ApiError::Http)?)
}
```

**Delete** old `get_resource`/`post_resource` calls in this file — Connect has no wrapper key.

### 4.3 Verify

```bash
bazel test //clients/core/crates/types/... //clients/core/crates/api-client/...
```

---

## Step 5 — wasm bridge

`clients/core/crates/wasm/src/service_<service>.rs`:

```rust
use agentsmesh_api_client::ApiClient;
use agentsmesh_types::*;
use std::sync::Arc;
use wasm_bindgen::prelude::*;

#[wasm_bindgen]
pub struct <Service>Service { client: Arc<ApiClient> }

#[wasm_bindgen]
impl <Service>Service {
    #[wasm_bindgen(constructor)]
    pub fn new(client: &WasmApiClient) -> Self { Self { client: client.inner() } }

    #[wasm_bindgen(js_name = list<Service>s)]
    pub async fn list_<service>s(&self, offset: Option<i32>, limit: Option<i32>) -> Result<JsValue, JsValue> {
        let req = List<Service>sRequest { offset, limit };
        let resp = self.client.list_<service>s(&req).await.map_err(api_err_to_js)?;
        serde_wasm_bindgen::to_value(&resp).map_err(|e| JsValue::from_str(&e.to_string()))
    }

    #[wasm_bindgen(js_name = create<Service>)]
    pub async fn create_<service>(&self, name: String, description: Option<String>) -> Result<JsValue, JsValue> {
        let req = Create<Service>Request { name, description };
        let entity = self.client.create_<service>(&req).await.map_err(api_err_to_js)?;
        serde_wasm_bindgen::to_value(&entity).map_err(|e| JsValue::from_str(&e.to_string()))
    }
}
```

**Keep method names stable.** TS callers depend on `list<Service>s`, `create<Service>` etc.

### Verify

```bash
bazel build //clients/core/crates/wasm:wasm_pkg
bazel test  //clients/core/crates/wasm:wasm_lib_test
```

---

## Step 6 — Web TS adapter

### 6.1 Find call sites

```bash
grep -rn "<entity>Service\|get<Service>Service\|use<Service>" clients/web/src/ | grep -v ".test."
```

For each:

- **List shape**: `resp.<plural>` → `resp.items`. Add destructuring for `total/limit/offset` if used.
- **Create/update**: `resp.<entity>` → `resp` (entity is the response directly).

### 6.2 Delete old TS types

```bash
rm clients/web/src/lib/api/<service>Types.ts                        # if exists
grep -rn "from ['\"]@/lib/api/<service>Types" clients/web/src/      # find imports
```

Replace each import with wasm-generated types (from `agentsmesh-wasm` `.d.ts`) or a thin local alias.

### 6.3 Update vitest mocks

```bash
grep -rn "<entity>\|<plural>" clients/web/src/test/setup.ts
```

Existing mocks were written against the drifted shape (watch list §6 / PR #368). Update to match the new `.proto` shape.

### 6.4 Verify

```bash
bazel test  //clients/web:unit //clients/web:lint
bazel build //clients/web:src //clients/web:next
```

---

## Step 7 — Tests (MANDATORY)

### 7.1 Rust round-trip — at the bottom of `clients/core/crates/types/src/<service>.rs`

```rust
#[cfg(test)]
mod tests {
    use super::*;

    const LIST_PAYLOAD: &str = r#"{
        "items":[{"id":1,"name":"sample","createdAt":"2026-05-12T13:16:10Z"}],
        "total":1,"limit":20,"offset":0
    }"#;

    #[test]
    fn list_response_round_trip() {
        let r: List<Service>sResponse = serde_json::from_str(LIST_PAYLOAD).unwrap();
        assert_eq!(r.items.len(), 1);
        assert_eq!(r.total, 1);
        let v: serde_json::Value = serde_json::from_str(&serde_json::to_string(&r).unwrap()).unwrap();
        assert!(v.get("items").is_some());
        assert!(v.get("<service>s").is_none(), "must not emit drifted wrapper key");
    }

    #[test]
    fn entity_camelcase_on_wire() {
        let e = <Service>{ id: 1, name: "x".into(), description: None,
                           created_at: "2026-05-12T00:00:00Z".into(), updated_at: None };
        let s = serde_json::to_string(&e).unwrap();
        assert!(s.contains("\"createdAt\""));
        assert!(!s.contains("\"created_at\""));
    }
    // Add tests per oneof variant if any.
    // Add explicit `offset: 0` vs absent test for any optional scalar.
}
```

### 7.2 Go handler unit test — `backend/internal/api/connect/<service>/<service>_test.go`

```go
package <service>connect

import (
    "context"
    "testing"

    "connectrpc.com/connect"
    <service>v1 "github.com/anthropics/agentsmesh/proto/gen/go/<service>/v1"
    "github.com/stretchr/testify/require"
)

func TestList_DefaultLimit(t *testing.T) {
    s := New(/* stub svc */)
    resp, err := s.List<Service>s(authCtxFor(123), connect.NewRequest(&<service>v1.List<Service>sRequest{}))
    require.NoError(t, err)
    require.Equal(t, int32(20), resp.Msg.Limit) // server default
}

func TestCreate_AlreadyExists(t *testing.T) {
    _, err := /* call with stub returning ErrAlreadyExists */
    require.Equal(t, connect.CodeAlreadyExists, connect.CodeOf(err))
}

func TestUnauthenticated_NoBearer(t *testing.T) {
    _, err := /* call without auth ctx */
    require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}
```

### 7.3 Drift-auditor

`clients/web/scripts/check-no-wrapper-key.sh <service>` — fails if `resp.<service>_plural` access pattern remains in TS.

### 7.4 Full Bazel test set

```bash
bazel test //proto/... \
           //backend/internal/api/connect/<service>:<service>_test \
           //clients/core/crates/types/... \
           //clients/core/crates/api-client/... \
           //clients/core/crates/wasm:wasm_lib_test \
           //clients/web:unit //clients/web:lint
bazel build //backend/cmd/server:server //clients/web:src //clients/web:next
```

All green → ready for PR.

---

## Step 8 — PR

### 8.1 Branch + commit

```bash
git checkout -b proto-migration/service-<service> origin/feat/proto-migration

git add proto/<service>/ backend/internal/api/connect/<service>/ backend/cmd/server/connect_init.go \
        clients/core/crates/types/src/<service>.rs \
        clients/core/crates/api-client/src/modules/<service>.rs \
        clients/core/crates/wasm/src/service_<service>.rs \
        clients/web/src/
git rm clients/web/src/lib/api/<service>Types.ts 2>/dev/null || true

git commit -m "$(cat <<'EOF'
feat(proto-migration): migrate <service> data plane to Connect-RPC

- proto/<service>/v1/<service>.proto with uniform list envelope + entity-direct
- backend/internal/api/connect/<service>/<service>.go behind auth interceptor
- clients/core/crates/types/src/<service>.rs rewritten as prost+serde
- clients/core/crates/api-client/src/modules/<service>.rs through connect_call
- clients/core/crates/wasm/src/service_<service>.rs signatures preserved
- clients/web/src/ call sites updated for {items,total,limit,offset}
- clients/web/src/lib/api/<service>Types.ts removed
- Existing REST handlers kept mounted (dual-track)

Round-trip tests pin camelCase wire + no drifted wrapper keys.
Part of feat/proto-migration. DO NOT merge to main.
EOF
)"
```

### 8.2 Push + PR

```bash
git push -u origin proto-migration/service-<service>
gh pr create --base feat/proto-migration \
  --title "feat(proto-migration): migrate <service> data plane to Connect-RPC" \
  --body "$(cat <<'EOF'
## Summary
Migrates the <service> data plane from REST + hand-written DTOs to Connect-RPC + .proto.
Follows conventions locked in .claude/plans/proto-naming-conventions.md.

## Drift reconciled (from runbook §1.5)
[Field-set diff Go vs Rust vs TS. Any drift fixed inline.]

## Field map
| .proto | Wire (camelCase) | Rust | Old REST gin.H key |
|---|---|---|---|
| id | id | i64 | id |
| created_at | createdAt | String | created_at |
| ... | ... | ... | ... |

## Test plan
- [x] bazel build //proto/<service>/v1:<service>_proto //proto/<service>/v1:<service>_go_proto
- [x] bazel test //backend/internal/api/connect/<service>:<service>_test
- [x] bazel build //backend/cmd/server:server
- [x] bazel test //clients/core/crates/types/... //clients/core/crates/api-client/...
- [x] bazel test //clients/core/crates/wasm:wasm_lib_test
- [x] bazel test //clients/web:unit //clients/web:lint
- [x] bazel build //clients/web:src //clients/web:next
- [ ] CI green on this PR
- [ ] Manual: load page that consumes <service>, no `undefined` renders

## Related drift PRs
<historical_drift_prs>

## Reviewer checklist
- [ ] Package = proto.<service>.v1
- [ ] No camelCase in .proto, no json_name annotations
- [ ] List response = {items, total, limit, offset}
- [ ] Create/update returns entity directly (no {entity:...} wrapper)
- [ ] No google.protobuf.Timestamp (string ISO-8601 only)
- [ ] Auth interceptor mounted on all RPCs
- [ ] Rust response messages have #[serde(rename_all = "camelCase")]
- [ ] Round-trip test asserts camelCase + no drift keys
- [ ] TS call sites read .items (not .<plural>)
EOF
)"

sleep 30
gh pr checks <pr-number> --watch --interval 20 --fail-fast
```

### 8.3 Do NOT merge

PR target is `feat/proto-migration`. **Do not merge** — orchestrator/team-lead decides merge order across the 26 parallel PRs.

---

## Specialist Agent Prompt Template

Copy + fill the variables below to spawn a service-migration agent.

```
# Migrate `{service_name}` service to Connect-RPC + protobuf

## Inputs
- service_name: {service_name}
- rest_handler_paths: {rest_handler_paths}
- rust_dto_path: {rust_dto_path}
- rust_api_client_path: {rust_api_client_path}
- rust_wasm_service_path: {rust_wasm_service_path}
- ts_types_path: {ts_types_path}
- historical_drift_prs: {historical_drift_prs}

## Must read first (in order)
1. .claude/plans/proto-migration-adr.md
2. .claude/plans/proto-naming-conventions.md  — every SHALL rule
3. .claude/plans/proto-watch-list.md          — 7 known hazards (especially #1 camelCase, #5 auth, #6 field accumulation)
4. .claude/plans/proto-migration-runbook.md   — THIS document

## Working environment
You are spawned into an isolated worktree, base reset to origin/feat/proto-migration latest.

git fetch origin feat/proto-migration
git checkout -b proto-migration/service-{service_name} origin/feat/proto-migration

## Steps
Execute runbook §1 through §8 verbatim. Each step's Bazel verification must pass before proceeding.
DO NOT skip §7.1 (round-trip test) — that is the entire reason for the migration.

## Hard constraints
- PR target = feat/proto-migration (NEVER main).
- Do NOT merge the PR. Push, wait CI green, return to caller.
- Do NOT delete the existing Gin REST handler. Dual-track.
- Do NOT bump connectrpc.com/connect (locked at v1.19.1).
- Do NOT introduce google.protobuf.Timestamp — use string ISO-8601.
- Do NOT use json_name annotations in .proto.
- Do NOT skip #[serde(rename_all = "camelCase")] on Rust response messages.

## Done means
- All Bazel targets green: //proto/{service_name}/..., //backend/internal/api/connect/{service_name}/...,
  //backend/cmd/server:server, //clients/core/..., //clients/web:unit, //clients/web:lint,
  //clients/web:src, //clients/web:next.
- PR open against feat/proto-migration with the body template from runbook §8.2 filled in.
- CI green on the PR.
- Round-trip test exists for every response message.
- Three-way drift diff (runbook §1.5) is in the PR description.

## Return when complete
- PR URL
- CI status
- Any deviations from conventions.md you had to make (if none, say "none")
- Any drift you discovered and fixed inline
```

---

## Service-specific deviations (handle case-by-case)

| Quirk | How to handle |
|---|---|
| Admin-only RPCs | Split into `<Service>Service` + `Admin<Service>Service`. Admin service has a separate interceptor checking `is_system_admin`. |
| Org-scoped vs not | Org slug in request body as `string organization_slug = 1;`, validated by tenant-isolation interceptor. NOT in URL — Connect URLs have no path params. |
| File upload (multipart) | Stays REST. Connect doesn't handle `multipart/form-data` cleanly. Document the exception in the PR. |
| Streaming responses (SSE) | Use `connect.NewServerStreamHandler`. wasm side needs an async-iterator wrapper — **defer to a follow-up PR**, v1 ships unary only. |
| WebSocket-backed (channels, terminal) | Stays on Relay. Connect replaces unary REST only; data-plane streaming uses the Relay. |

These deviations MUST be flagged in the PR description ("DEVIATION: file upload remains REST...").
