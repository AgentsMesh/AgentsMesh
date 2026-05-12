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

// All org-scoped request messages have `string org_slug = 1;` as their first field.
// User-scoped / admin-only services drop this — see conventions.md §3.5.

message List<Service>sRequest {
  string org_slug = 1;
  optional int32 offset = 2;
  optional int32 limit = 3;
}

message List<Service>sResponse {
  repeated <Service> items = 1;
  int64 total = 2;
  int32 limit = 3;
  int32 offset = 4;
}

message Get<Service>Request    { string org_slug = 1; int64 id = 2; }
message Create<Service>Request { string org_slug = 1; string name = 2; optional string description = 3; }
message Update<Service>Request { string org_slug = 1; int64 id = 2; optional string name = 3; optional string description = 4; }
message Delete<Service>Request { string org_slug = 1; int64 id = 2; }
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

type Server struct {
    svc    *<service>svc.Service
    orgSvc organization.Service
}

func New(svc *<service>svc.Service, orgSvc organization.Service) *Server {
    return &Server{svc: svc, orgSvc: orgSvc}
}

// Org-scoped RPCs use authinterceptor.ResolveOrgScope (introduced by the first migrating
// service — pattern locked in conventions.md §3.5). User-scoped / admin RPCs skip it.

func (s *Server) List<Service>s(
    ctx context.Context, req *connect.Request[<service>v1.List<Service>sRequest],
) (*connect.Response[<service>v1.List<Service>sResponse], error) {
    ctx, org, err := authinterceptor.ResolveOrgScope(ctx, req, s.orgSvc)
    if err != nil { return nil, err }

    limit, offset := int(req.Msg.GetLimit()), int(req.Msg.GetOffset())
    if limit == 0 { limit = 20 }

    items, total, err := s.svc.List(ctx, org.ID, limit, offset)
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
    ctx, org, err := authinterceptor.ResolveOrgScope(ctx, req, s.orgSvc)
    if err != nil { return nil, err }
    entity, err := s.svc.Create(ctx, org.ID, &<service>svc.CreateRequest{
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

Binary wire only — `prost::Message` derive is the entire contract. **Do not add `Serialize` / `Deserialize` derives.** They were a JSON-wire concern that no longer applies (conventions §2.5, §3).

```rust
use prost::Message;

#[derive(Clone, PartialEq, prost::Message)]
pub struct <Service> {
    #[prost(int64,  tag = "1")] pub id: i64,
    #[prost(string, tag = "2")] pub name: String,
    #[prost(string, optional, tag = "3")] pub description: Option<String>,
    #[prost(string, tag = "4")] pub created_at: String,
    #[prost(string, optional, tag = "5")] pub updated_at: Option<String>,
}

#[derive(Clone, PartialEq, prost::Message, Default)]
pub struct List<Service>sRequest {
    #[prost(string, tag = "1")] pub org_slug: String,
    #[prost(int32, optional, tag = "2")] pub offset: Option<i32>,
    #[prost(int32, optional, tag = "3")] pub limit:  Option<i32>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct List<Service>sResponse {
    #[prost(message, repeated, tag = "1")] pub items: Vec<<Service>>,
    #[prost(int64,   tag = "2")] pub total: i64,
    #[prost(int32,   tag = "3")] pub limit: i32,
    #[prost(int32,   tag = "4")] pub offset: i32,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct Create<Service>Request {
    #[prost(string, tag = "1")] pub org_slug: String,
    #[prost(string, tag = "2")] pub name: String,
    #[prost(string, optional, tag = "3")] pub description: Option<String>,
}

// Get/Update/Delete requests analogous — each starts with `org_slug` at tag 1.
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

`connect_call` helper (lives in `clients/core/crates/api-client/src/connect_call.rs`, added by infra or first-migration PR) — **binary in, binary out, no JSON path**:

```rust
use prost::Message;

pub async fn connect_call<Req, Res>(
    client: &ApiClient, procedure: &str, body: &Req,
) -> Result<Res, ApiError>
where
    Req: Message,
    Res: Message + Default,
{
    let url = format!("{}{}", client.base_url(), procedure);
    let payload = body.encode_to_vec();
    let mut b = client.http.post(&url)
        .header("Connect-Protocol-Version", "1")
        .header("Content-Type", "application/proto")
        .body(payload);
    if let Some(tok) = client.token().await { b = b.header("Authorization", format!("Bearer {tok}")); }
    let resp = b.send().await.map_err(ApiError::Http)?;
    if !resp.status().is_success() { return Err(ApiError::from_connect_response(resp).await); }
    let resp_bytes = resp.bytes().await.map_err(ApiError::Http)?;
    Res::decode(resp_bytes).map_err(ApiError::Decode)
}
```

**Delete** old `get_resource`/`post_resource` calls in this file — Connect has no wrapper key. **Do not** add JSON-fallback branches "for debugging" — curl with `Content-Type: application/json` against the server still works on the server side; client code stays binary-only.

### 4.3 Verify

```bash
bazel test //clients/core/crates/types/... //clients/core/crates/api-client/...
```

---

## Step 5 — wasm bridge

The wasm bridge surface is **binary in, binary out** — `Vec<u8>` (= TS `Uint8Array`). TS decodes via `@bufbuild/protoc-gen-es`-generated `Message.fromBinary`. No `serde_wasm_bindgen`, no JSON intermediate.

`clients/core/crates/wasm/src/service_<service>.rs`:

```rust
use agentsmesh_api_client::ApiClient;
use agentsmesh_types::*;
use prost::Message;
use std::sync::Arc;
use wasm_bindgen::prelude::*;

#[wasm_bindgen]
pub struct <Service>Service { client: Arc<ApiClient> }

#[wasm_bindgen]
impl <Service>Service {
    #[wasm_bindgen(constructor)]
    pub fn new(client: &WasmApiClient) -> Self { Self { client: client.inner() } }

    // org_slug typically comes from the AuthManager-held current org (org_path()),
    // not from TS arguments — keeps existing TS call sites unchanged.

    #[wasm_bindgen(js_name = list<Service>s)]
    pub async fn list_<service>s(&self, offset: Option<i32>, limit: Option<i32>) -> Result<Vec<u8>, JsValue> {
        let req = List<Service>sRequest { org_slug: self.client.org_slug().await, offset, limit };
        let resp = self.client.list_<service>s(&req).await.map_err(api_err_to_js)?;
        Ok(resp.encode_to_vec())
    }

    #[wasm_bindgen(js_name = create<Service>)]
    pub async fn create_<service>(&self, name: String, description: Option<String>) -> Result<Vec<u8>, JsValue> {
        let req = Create<Service>Request { org_slug: self.client.org_slug().await, name, description };
        let entity = self.client.create_<service>(&req).await.map_err(api_err_to_js)?;
        Ok(entity.encode_to_vec())
    }
}
```

**Keep method names stable.** TS callers depend on `list<Service>s`, `create<Service>` etc.

**Return-type discipline**: `Result<Vec<u8>, JsValue>` only. **Do not** return `JsValue` from a `serde_wasm_bindgen::to_value(&resp)` call — that round-trips through JSON serialization and reintroduces the field-name drift surface. The wasm output is raw prost bytes; TS deserializes via the corresponding `@bufbuild/protobuf` message class (see Step 6).

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

- **Decode wasm output**: every call site now receives a `Uint8Array` from the wasm bridge. Decode via the `@bufbuild/protoc-gen-es`-generated message class:
  ```typescript
  import { ListFoosResponse, Foo } from "@/proto/gen/ts/foo/v1/foo_pb";

  // Before (JSON via serde_wasm_bindgen):
  // const resp = await wasmFooService.listFoos(offset, limit);
  // return resp;  // already a JS object

  // After (binary in, binary out):
  const bytes = await wasmFooService.listFoos(offset, limit);
  const resp = ListFoosResponse.fromBinary(new Uint8Array(bytes));
  return resp;  // typed message instance, .items / .total / .limit / .offset
  ```
- **List shape**: `resp.<plural>` → `resp.items`. Add destructuring for `total/limit/offset` if used.
- **Create/update**: `resp.<entity>` → `resp` (entity is the response directly). Decode via `<Entity>.fromBinary(new Uint8Array(bytes))`.

The `@bufbuild/protobuf` message class auto-generates **camelCase getters** from the snake_case `.proto`. Existing TS call sites reading `.createdAt` continue to work — same field-name surface, different wire path.

### 6.2 Delete old TS types

```bash
rm clients/web/src/lib/api/<service>Types.ts                        # if exists
grep -rn "from ['\"]@/lib/api/<service>Types" clients/web/src/      # find imports
```

Replace each import with the `@bufbuild/protoc-gen-es`-generated module at `@/proto/gen/ts/<service>/v1/<service>_pb` (output path defined by the `ts_proto_library` Bazel rule — see "TS proto codegen toolchain" section below).

### 6.3 Update vitest mocks

```bash
grep -rn "<entity>\|<plural>" clients/web/src/test/setup.ts
```

Existing mocks returned drifted JSON strings (watch list §6 / PR #368). Update each mock to return a **binary-encoded empty response** using the `@bufbuild/protobuf` class:

```typescript
import { ListFoosResponse, Foo } from "@/proto/gen/ts/foo/v1/foo_pb";

// Before:
// listFoos: vi.fn().mockResolvedValue('{"foos":[]}'),

// After (binary bytes, decoded by the call-site `fromBinary`):
listFoos: vi.fn().mockResolvedValue(new ListFoosResponse({}).toBinary()),

// Or with data:
listFoos: vi.fn().mockResolvedValue(
  new ListFoosResponse({
    items: [new Foo({ id: 1n, name: "sample" })],
    total: 1n,
    limit: 20,
    offset: 0,
  }).toBinary(),
),
```

The mock's return type is `Uint8Array`, matching the production wasm bridge surface. Call-site `Message.fromBinary` works against either.

### 6.4 Verify

```bash
bazel test  //clients/web:unit //clients/web:lint
bazel build //clients/web:src //clients/web:next
```

---

## Step 7 — Tests (MANDATORY)

### 7.1 Rust round-trip — at the bottom of `clients/core/crates/types/src/<service>.rs`

Binary wire round-trip — encode every distinguishing field, decode, assert. A swapped or transposed `prost(tag = N)` annotation manifests as field-value swap in the assertions (see watch list §8).

```rust
#[cfg(test)]
mod tests {
    use super::*;
    use prost::Message;

    #[test]
    fn list_response_round_trip() {
        let original = List<Service>sResponse {
            items: vec![<Service> {
                id: 1, name: "sample".into(), description: None,
                created_at: "2026-05-12T13:16:10Z".into(), updated_at: None,
            }],
            total: 1, limit: 20, offset: 0,
        };
        let bytes = original.encode_to_vec();
        let decoded = List<Service>sResponse::decode(&*bytes).unwrap();
        assert_eq!(original, decoded);
        // Every field at its expected position — drifted tag would swap values.
    }

    #[test]
    fn optional_offset_zero_vs_absent_distinguishable() {
        let with_zero = List<Service>sRequest { org_slug: "o".into(), offset: Some(0), limit: None };
        let absent    = List<Service>sRequest { org_slug: "o".into(), offset: None,    limit: None };
        // Binary wire: with_zero emits the tag with value 0; absent emits no tag.
        assert_ne!(with_zero.encode_to_vec(), absent.encode_to_vec());
        let r1 = List<Service>sRequest::decode(&*with_zero.encode_to_vec()).unwrap();
        let r2 = List<Service>sRequest::decode(&*absent.encode_to_vec()).unwrap();
        assert_eq!(r1.offset, Some(0));
        assert_eq!(r2.offset, None);
    }

    // Add tests per oneof variant if any — see conventions §7.
    // Add per-field tag-collision smoke test if the .proto has 10+ fields.
}
```

The "camelCase on wire" test from the legacy template is **removed** — there is no JSON wire on the client side, so casing has no production touchpoint. TS-side decoding via `@bufbuild/protoc-gen-es` generates camelCase getters from the snake_case `.proto` at the message-class level (see Step 6.1).

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
- [ ] Rust DTO is prost::Message only — NO `Serialize` / `Deserialize` derives
- [ ] Every Rust `prost(tag = N)` matches the .proto field number
- [ ] wasm bridge returns `Result<Vec<u8>, JsValue>` — NO `serde_wasm_bindgen::to_value`
- [ ] TS call sites decode via `<Message>.fromBinary(new Uint8Array(bytes))`
- [ ] vitest mocks return `.toBinary()` bytes, not JSON strings
- [ ] TS call sites read .items (not .<plural>)
- [ ] Round-trip test encodes → decodes via prost::Message, asserts PartialEq
- [ ] No `application/json` content-type in any client code (grep -rE "application/json" clients/core/crates/api-client clients/core/crates/wasm clients/web/src/lib/api)
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
2. .claude/plans/proto-naming-conventions.md  — every SHALL rule (especially §2.5 codec)
3. .claude/plans/proto-watch-list.md          — 8 known hazards (especially #5 auth, #6 field accumulation, #8 tag-number drift)
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
- Do NOT add `Serialize` / `Deserialize` derives on migrated Rust DTOs — binary wire only.
- Do NOT use `application/json` content-type in any client-side code (api-client crate, wasm crate, clients/web/src/lib/api). Client wire is binary, mandatory.
- Do NOT return `JsValue` from `serde_wasm_bindgen::to_value(&resp)` in wasm bridge — return `Result<Vec<u8>, JsValue>` carrying `prost::Message::encode_to_vec()` bytes.

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

## TS proto codegen toolchain (infrastructure prerequisite for binary-only wire)

The binary-only client wire (conventions §2.5) requires TS-side message classes that decode prost bytes. We use `@bufbuild/protoc-gen-es` v2 — the canonical Connect-ecosystem codegen for TS. The first service migration PR (the `skill_registry` reference) **must land this toolchain** because the remaining 25 services depend on it.

### Root `package.json` additions

```json
{
  "devDependencies": {
    "@bufbuild/buf": "^1.45.0",
    "@bufbuild/protoc-gen-es": "^2.2.0"
  },
  "dependencies": {
    "@bufbuild/protobuf": "^2.2.0"
  }
}
```

`@bufbuild/protobuf` is the runtime (`Message.toBinary` / `fromBinary` lives there); `@bufbuild/protoc-gen-es` is the codegen plugin invoked by `buf generate`; `@bufbuild/buf` provides the host `buf` CLI we drive from Bazel.

### Bazel rule — `build_defs/ts/ts_proto.bzl`

Macro wrapping `buf generate` with the hermetic toolchain pulled from pnpm. Output lands in `bazel-bin/proto/<domain>/v1/<name>_pb.ts` (and a `_pb.d.ts` for IDE consumption).

```python
load("@aspect_rules_js//js:defs.bzl", "js_run_binary")

def ts_proto_library(name, proto, visibility = None):
    """Generates a TS message-class module from a proto_library target.

    Wraps `buf generate` driven by the pnpm-resolved
    @bufbuild/buf binary, producing <name>_pb.ts + <name>_pb.d.ts.
    """
    js_run_binary(
        name = name,
        srcs = [proto],
        tool = "@npm//@bufbuild/buf/bin:buf",
        args = ["generate", "--template=$(BUF_TEMPLATE)", "--output=$(@D)"],
        # ... full implementation: locate proto descriptor set, invoke buf generate,
        # output one .ts file per .proto, with .d.ts beside it.
        visibility = visibility,
    )
```

Detailed implementation (descriptor-set wiring, buf.gen.yaml template, output-path normalization) is the first-service PR's deliverable. The macro signature above is the locked contract.

### Per-service `BUILD.bazel` — pair `proto_library` with `ts_proto_library`

Update `proto/<service>/v1/BUILD.bazel` (the Step 2 template) to add a third target:

```python
load("@rules_go//proto:def.bzl", "go_proto_library")
load("@rules_proto//proto:defs.bzl", "proto_library")
load("//build_defs/ts:ts_proto.bzl", "ts_proto_library")

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

ts_proto_library(
    name = "<service>_ts_proto",
    proto = ":<service>_proto",
    visibility = ["//visibility:public"],
)
```

TS consumers import from `@/proto/gen/ts/<service>/v1/<service>_pb` — the alias resolves to `bazel-bin/proto/<service>/v1/<service>_ts_proto/<service>_pb.ts` through the existing Bazel JS rule's path mapping.

### Verification (first-service PR adds these to CI)

```bash
bazel build //proto/<service>/v1:<service>_ts_proto                  # codegen runs
bazel test  //clients/web:unit                                       # vitest mocks compile against the new class
bazel build //clients/web:src                                        # tsc --noEmit catches import path mistakes
```

### Why not `protoc-gen-ts` (the legacy generator)?

`@bufbuild/protoc-gen-es` is the only TS codegen with first-class support for the `Message.toBinary` / `fromBinary` API that our binary wire requires. Alternatives (`protoc-gen-ts`, `ts-proto`) target JSON-first interop or generate non-buffered runtime code — neither matches the Connect-ecosystem contract we lock in conventions §2.5.

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
