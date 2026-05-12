# Proto Naming & Wire Conventions (LOCKED)

Authoritative naming + shape rules for the proto-migration (see ADR). Every per-service PR MUST conform. **Convention violations are blocking review comments**, not nits.

Each section: rule → **bad example** → **good example** → **how CI / lint catches it**.

---

## 1. Proto package — `proto.<domain>.v1`

**Rule**: Every `.proto` declares `package proto.<domain>.v1;` and `option go_package = "...v1;<domain>v1";`.

- `<domain>` is **lowercase, single word**, matching the existing Rust crate / handler subpackage name (e.g., `extension`, `pod`, `billing`).
- `runner.v1` is **already in use** by the control plane (`proto/runner/v1/runner.proto`) — do not reuse, do not collide.
- The router uses the `/proto.` prefix as the dispatch key (see `backend/cmd/server/connect_init.go`). Skipping the prefix breaks routing.

**Bad**:
```protobuf
syntax = "proto3";
package extension.v1;            // missing /proto. prefix → won't dispatch
option go_package = "github.com/anthropics/agentsmesh/proto/extension/v1";
```

**Good**:
```protobuf
syntax = "proto3";
package proto.extension.v1;
option go_package = "github.com/anthropics/agentsmesh/proto/gen/go/extension/v1;extensionv1";
```

**CI gate**: `tools/proto_lint` (to be added) `grep -L '^package proto\.' proto/**/*.proto` returns non-empty → fail. Service URL test in handler test ensures `/proto.<domain>.v1.<Service>/Method` round-trips.

---

## 2. Service & method naming

**Rule**: `<Domain>Service.<Method>`. PascalCase for both. Methods are verbs (`List`, `Create`, `Update`, `Delete`, `Get`, `Toggle`, `Sync`).

- One `Service` per domain. Multiple services in one `.proto` is permitted only when a strict admin/non-admin split exists (e.g., `SkillRegistryService` + `AdminSkillRegistryService`).
- **No `Pb` / `Proto` / `_v1` suffixes** in service name — the package already encodes version.

**Bad**:
```protobuf
service extensionService {                  // lowercase
  rpc list_skill_registries(...) returns (...);  // snake_case method
}
```

**Good**:
```protobuf
service SkillRegistryService {
  rpc ListSkillRegistries(ListSkillRegistriesRequest) returns (ListSkillRegistriesResponse);
  rpc CreateSkillRegistry(CreateSkillRegistryRequest) returns (SkillRegistry);
  rpc ToggleSkillRegistry(ToggleSkillRegistryRequest) returns (SkillRegistry);
}
```

**CI gate**: `tools/proto_lint` runs `buf lint` with rule `SERVICE_SUFFIX = "Service"` and `METHOD_REQUEST_RESPONSE_UNIQUE` to forbid sharing request/response types across methods (each method must have its own request type to keep them evolvable).

---

## 2.5 Codec — Wire = protobuf binary (client zero JSON path)

### Client wire format (mandatory)

| Component | Out/In wire |
|---|---|
| Rust wasm crate | `prost::Message::encode_to_vec()` / `prost::Message::decode(bytes)` |
| wasm-bindgen bridge | `Result<Vec<u8>, JsValue>` (NOT `Result<String, JsValue>`) |
| TS web / desktop | `@bufbuild/protobuf` generated message class — `toBinary()` / `fromBinary()` |
| HTTP request | `Content-Type: application/proto` |

**Forbidden**: any client-side code (Rust / wasm / TS) using `application/json`. There is no JSON fallback. There is no `serde_json` on the request/response path. The wasm bridge surface is binary in, binary out.

### Server-side content-type negotiation (kept)

Connect-Go handlers accept both `application/proto` and `application/json` by default — this is the framework's content negotiator, costs us nothing, and gives a free curl-debug surface.

| Path | Content-Type |
|---|---|
| Production (wasm / TS → server) | `application/proto` — always binary |
| Debug (curl, admin scripts → server) | `application/json` — permitted, framework decodes |

Handler code (service implementation) is identical in both cases — Connect-Go materializes the request proto struct before calling the handler. Business code never sees the wire.

### Why no JSON on the client

POC report §3.4 ranked "JSON camelCase vs Rust naming mismatch" as the **#1 migration hazard** — and that is precisely the failure mode (PR #329–#368) the migration set out to eliminate. Keeping a client JSON path preserves the drift surface:

- JSON wire identifies fields by **string name**. Names in `.proto` are `snake_case`, names in JSON are `camelCase`, names in Rust structs are `snake_case` — three labels for the same field, three places a typo or `rename_all` slip can drift.
- Binary wire identifies fields by **tag number** (`prost(tag = N)` ↔ `.proto` field number). Field names are local identifiers; they never travel on the wire. A `Foo` in Rust, a `foo` in Go's generated type, a `foo` in TS — all decode the same `tag = 1` bytes. **Drift physically cannot happen.**

The drift bug class is closed by codec choice, not by lint discipline.

### What this replaces from the ADR

ADR §"Locked decisions" lists `Codec default = Connect+JSON`. That row reflected the POC-stage default. **For the 26-service rollout this convention overrides it**: client = binary, server = negotiating. The ADR decision (migrate to Connect-RPC + .proto SSOT) is unchanged.

### CI gate

1. `grep -rE "application/json" clients/core/crates/api-client/src/ clients/core/crates/wasm/src/ clients/web/src/lib/api/` returns non-empty → fail. Client paths must not mention JSON content-type.
2. `grep -r 'serde_wasm_bindgen\|serde_json' clients/core/crates/wasm/src/service_*.rs` — wasm service bridges return `Vec<u8>`, not `JsValue` serialized JSON. Existing uses (terminal events etc.) outside `service_*.rs` are fine.
3. wasm-bindgen `Promise<string>` returns in `service_*.rs` are a smell — review must catch them.

---

## 3. Field naming — `snake_case` everywhere, no `json_name`

**Rule**: Field names in `.proto` are `snake_case`. Rust prost structs are `snake_case`. TS proto-es generates camelCase getters from the same snake_case source. **Never** use `json_name = "..."` annotations.

`.proto` is still snake_case for two reasons even though client wire is binary:

1. **buf lint** rule `FIELD_LOWER_SNAKE_CASE` enforces it across the proto SSOT — keeps the source uniform.
2. **Server-side JSON negotiation** (curl / admin) still uses protojson rules, which map snake_case ↔ camelCase. A `.proto` with camelCase fields would break that path.

What changed from the legacy design: Rust DTOs **no longer carry `#[serde(...)]` derives**. The wire on the client side is binary, the type identity is the prost tag number, serde plays no role. See §4 (DTO template).

**Bad**:
```protobuf
message SkillRegistry {
  string repositoryUrl = 1;                          // camelCase in .proto
  string is_active = 2 [json_name = "isActive"];     // json_name annotation
}
```

**Good**:
```protobuf
message SkillRegistry {
  string repository_url = 1;       // snake_case in .proto
  bool is_active = 2;
}
```

```rust
// Rust DTO: prost::Message only, NO serde derives.
#[derive(Clone, PartialEq, prost::Message)]
pub struct SkillRegistry {
    #[prost(string, tag = "1")] pub repository_url: String,
    #[prost(bool,   tag = "2")] pub is_active: bool,
}
```

**CI gate**:
1. `buf lint` rule `FIELD_LOWER_SNAKE_CASE` rejects camelCase in `.proto`.
2. `grep -r 'json_name' proto/` returns non-empty → fail.
3. `grep -rE '#\[derive\(.*Serialize|Deserialize' clients/core/crates/types/src/` on migrated services → fail (no serde on migrated DTOs).
4. Per-service round-trip test (runbook §7): encode a request via prost, decode via prost, assert field-by-field equality. Tag-number drift between `.proto` and the Rust `prost(tag = N)` annotation surfaces here.

---

## 3.5 Tenant scope field — org-scoped RPC carries `org_slug` at tag 1

### Rule

Every **org-scoped** RPC request message has, as its **first field**:

```protobuf
message FooRequest {
  string org_slug = 1;     // ALWAYS field 1, ALWAYS named org_slug
  // business fields start from tag 2
}
```

### Why

Connect RPC URLs are `/proto.<domain>.v1.<Service>/<Method>` — **no path parameters**. The existing REST pattern `/api/v1/orgs/:slug/...` + middleware-injected tenant context **does not work** on Connect because there is no `:slug` to bind. Each request must carry its own org scope explicitly in the payload.

Without locking the field name + tag, 26 specialist agents will each invent their own (`org_slug` vs `organization_slug` vs `org_id` vs nested `Context.org_slug`) — that's a new drift surface, and worse, it blocks a generic `ResolveOrgScope[T]` helper from being written once.

### Exceptions

Only **two** classes of RPC are exempt from `org_slug`:

1. **User-scoped** (no org dependency): `Login`, `Register`, `ListMyOrganizations`, `GetMyProfile`, `RefreshToken`, etc.
2. **Platform-admin scoped**: every `Admin<Service>Service` RPC — tenant is implied by the admin interceptor / system-wide context.

Exempt request messages start their tag 1 with the **business field directly** (do not leave tag 1 reserved — the empty slot has no semantic).

### Handler usage

Every org-scoped Connect handler resolves the org scope first, using a generic helper introduced by the first migrating service (suggested: `skill_registry` as reference impl):

```go
// backend/internal/api/connect/interceptor/org_scope.go
package authinterceptor

type OrgScopedRequest interface { GetOrgSlug() string }

func ResolveOrgScope[T OrgScopedRequest](
    ctx context.Context, req *connect.Request[T], orgSvc organization.Service,
) (context.Context, *organization.Organization, error) {
    slug := req.Msg.GetOrgSlug()
    if slug == "" {
        return ctx, nil, connect.NewError(connect.CodeInvalidArgument, errors.New("missing org_slug"))
    }
    userID, err := UserID(ctx)
    if err != nil { return ctx, nil, connect.NewError(connect.CodeUnauthenticated, err) }
    org, err := orgSvc.GetBySlug(ctx, slug)
    if err != nil { return ctx, nil, connect.NewError(connect.CodeNotFound, err) }
    if !orgSvc.IsMember(ctx, org.ID, userID) {
        return ctx, nil, connect.NewError(connect.CodePermissionDenied, errors.New("not a member"))
    }
    return contextWithTenant(ctx, org), org, nil
}
```

Per-handler usage:

```go
func (s *Server) ListSkillRegistries(
    ctx context.Context, req *connect.Request[v1.ListSkillRegistriesRequest],
) (*connect.Response[v1.ListSkillRegistriesResponse], error) {
    ctx, org, err := authinterceptor.ResolveOrgScope(ctx, req, s.orgSvc)
    if err != nil { return nil, err }
    // ... use org.ID, ctx
}
```

### Bad

```protobuf
// org field in the middle
message ListSkillRegistriesRequest {
  int32 limit = 1;
  string org_slug = 2;          // wrong tag
}

// renamed field
message ListSkillRegistriesRequest {
  string organization_slug = 1; // must be `org_slug`
}

// nested in a sub-message
message ListSkillRegistriesRequest {
  RequestContext ctx = 1;       // do NOT nest tenant in a sub-message
  int32 limit = 2;
}
```

### Good

```protobuf
message ListSkillRegistriesRequest {
  string org_slug = 1;
  optional int32 offset = 2;
  optional int32 limit = 3;
}

message CreateSkillRegistryRequest {
  string org_slug = 1;
  string repository_url = 2;
  optional string branch = 3;
}
```

### CI gate

Custom linter rule (`tools/proto_lint`, Day-3 task): every RPC request message must satisfy one of:

1. The request message has field `string org_slug = 1;` as its first field, OR
2. The service name matches the user-scoped whitelist (`Auth*`, `User*` for self-profile RPCs), OR
3. The service name matches the admin pattern (`Admin*Service`).

Violations fail CI. The whitelist is checked into `tools/proto_lint/user_scoped_services.txt`.

---

## 4. `prost(tag = N)` stability — tags are forever, names are not

**Rule**: Every Rust field has `#[prost(<type>, tag = "N")]` where N matches the `.proto` field number exactly.

- **Never reuse a tag.** A removed field's tag goes to the `reserved` list in `.proto` (mirror `proto/runner/v1/runner.proto` line 56: `reserved 27, 28, 29, 30;`).
- **Never reorder tags** to match struct field order — they are independent.
- Field renames are safe (tag is the wire identity). Tag changes are wire-breaking.

**Bad**:
```protobuf
message Plan {
  string name = 1;
  // (removed) int32 deprecated_count = 2;
  int32 price_monthly = 2;            // tag 2 REUSED → old clients decode wrong type
}
```

**Good**:
```protobuf
message Plan {
  string name = 1;
  reserved 2;
  reserved "deprecated_count";        // both number and name reserved
  int32 price_monthly = 3;            // new tag
}
```

**CI gate**: `buf breaking` runs against `feat/proto-migration` base on each PR. Tag reuse + tag-renumber are wire-breaking and fail the check.

---

## 5. Optional vs default scalars — when zero is meaningful, use `optional`

**Rule**: Any scalar field where **zero is semantically distinct from absent** uses the proto3 `optional` keyword. This applies to:

- Pagination: `offset`, `count`, `page`, `limit` — `0` is a valid request offset.
- Counters / quotas: `usage`, `seats_used` — `0` is meaningful.
- Toggles where tri-state matters: `is_enabled` is `bool`, but if "unset" must be distinct from `false`, use `optional bool`.

For required scalars where zero == empty (the default semantic), use plain proto3 (no `optional`).

**Bad**:
```protobuf
message ListSkillRegistriesRequest {
  int32 offset = 1;        // default 0 — indistinguishable from "no offset specified"
  int32 limit = 2;         // default 0 — could mean "no limit" or "broken request"
}
```

**Good**:
```protobuf
message ListSkillRegistriesRequest {
  optional int32 offset = 1;     // present-with-value-0 vs absent are different
  optional int32 limit = 2;      // server can apply default if absent
}
```

```rust
#[derive(Clone, PartialEq, prost::Message)]
pub struct ListSkillRegistriesRequest {
    #[prost(string, tag = "1")] pub org_slug: String,
    #[prost(int32, optional, tag = "2")] pub offset: Option<i32>,
    #[prost(int32, optional, tag = "3")] pub limit:  Option<i32>,
}
```

Binary wire elides absent fields by tag-number absence (prost's `Option<T>` maps to "field not present"). No `serde(skip_serializing_if)` needed — that was a JSON-wire concern.

**CI gate**: round-trip test for each list-RPC with `offset: 0` explicit vs absent → assert the handler distinguishes them (e.g., absent uses server default page size, explicit `0` returns first page).

---

## 6. Timestamps — `string` ISO-8601, **no** `google.protobuf.Timestamp`

**Rule**: All time fields are `string`, formatted as RFC 3339 / ISO-8601 (`2026-05-12T13:16:10Z`).

Rationale (POC §3.4):
- `prost-types` is a separate crate. Adding it = one more `crate.spec` + wasm bundle cost.
- Backend already emits RFC 3339 (`time.RFC3339`) from `time.Time` fields via `gin.H` JSON marshal.
- Migrating to `google.protobuf.Timestamp` later is **non-breaking** on the JSON wire (`protojson` encodes `Timestamp` as the same ISO-8601 string).

**Bad**:
```protobuf
import "google/protobuf/timestamp.proto";

message SkillRegistry {
  google.protobuf.Timestamp last_synced_at = 1;   // pulls prost-types into wasm
}
```

**Good**:
```protobuf
message SkillRegistry {
  string last_synced_at = 1;       // RFC 3339 "2026-05-12T13:16:10Z"
}
```

```rust
#[derive(Clone, PartialEq, prost::Message)]
pub struct SkillRegistry {
    #[prost(string, tag = "1")]
    pub last_synced_at: String,
}
```

**Use `Option<String>` for nullable timestamps** (most "last seen at" / "ended at" fields) — `#[prost(string, optional, tag = "N")] pub field: Option<String>`.

**CI gate**: `grep -r 'google.protobuf.Timestamp' proto/` returns non-empty → fail (until we explicitly lift this restriction).

---

## 7. `oneof` — binary wire encodes the variant tag; Rust enum is the canonical shape

**Rule**: `oneof` fields encode on the binary wire as the variant's nested message at its tag number. Rust uses prost's standard `#[derive(prost::Oneof)]` enum. No custom serde, no JSON-side discriminator alignment — the binary lane handles this automatically.

When server-side admin/curl debugging hits a oneof endpoint with `application/json`, Connect-Go's `protojson` encodes/decodes the tagged shape `{<oneof_field>: {<variantName>: <payload>}}`. Variant field names are `snake_case` in `.proto`; protojson maps to camelCase. Client code never sees this shape.

**Bad** — splitting the wire form across two encodings:
```rust
// Don't hand-write serde derives for oneof — binary wire is sufficient.
impl Serialize for Event { /* ... custom JSON tagged shape ... */ }
impl Deserialize for Event { /* ... */ }
```

**Good** — prost-stock enum, no serde:
```protobuf
message PodEvent {
  oneof event {
    PodCreatedData pod_created = 1;
    PodTerminatedData pod_terminated = 2;
  }
}
```

```rust
#[derive(Clone, PartialEq, prost::Message)]
pub struct PodEvent {
    #[prost(oneof = "pod_event::Event", tags = "1, 2")]
    pub event: Option<pod_event::Event>,
}

pub mod pod_event {
    #[derive(Clone, PartialEq, prost::Oneof)]
    pub enum Event {
        #[prost(message, tag = "1")] PodCreated(super::PodCreatedData),
        #[prost(message, tag = "2")] PodTerminated(super::PodTerminatedData),
    }
}
```

Constraint that remains: `oneof` variant field names MUST be `snake_case` (buf lint `ONEOF_LOWER_SNAKE_CASE`) so the curl-debug JSON path stays consistent.

**CI gate**: every `oneof` adds a Rust round-trip test — `encode_to_vec` each variant, `decode` back, assert `PartialEq`. Tag-number drift between `.proto` and the `tags = "..."` annotation surfaces here. Cross-language wire test (curl with `application/proto` body) is optional — the binary codec is symmetric by construction.

---

## 8. List response envelope — `{items, total, limit, offset}` (UNIFORM)

**Rule**: Every list RPC's response message has the **same four fields**:

```protobuf
message ListXxxResponse {
  repeated Xxx items = 1;
  int64 total = 2;
  int32 limit = 3;
  int32 offset = 4;
}
```

This is the single biggest deviation from current REST handlers. Today the codebase has:
- `gin.H{"tickets": ..., "total": ..., "limit": ..., "offset": ...}`
- `gin.H{"items": ..., "total": ...}`  (admin/skill-registries — closest to target)
- `gin.H{"skill_registries": [...]}`   (no pagination, no envelope)
- `gin.H{"comments": ..., "total": ..., "limit": ..., "offset": ...}`

**Locked**: every list RPC migrates to `{items, total, limit, offset}`. No `{<domain>_plural: [...]}` keys, no service-specific envelopes. The Connect handler builds this envelope from the existing service-layer return values.

Rationale:
- Eliminates the per-service "what is the wrapper key" lookup that broke #341 / #368 / 986a38ca6.
- TS consumers can write one generic `extractList<T>(resp): {items: T[], total: number}` helper used across all 26 services.
- Frees the per-service Rust DTO from having to mirror each backend's plural-key spelling.

**Bad**:
```protobuf
message ListSkillRegistriesResponse {
  repeated SkillRegistry registries = 1;          // domain-specific plural
  int64 total_count = 2;                          // off-template name
}
```

**Good**:
```protobuf
message ListSkillRegistriesResponse {
  repeated SkillRegistry items = 1;
  int64 total = 2;
  int32 limit = 3;
  int32 offset = 4;
}
```

**Migration note**: existing TS code reading `resp.skill_registries` / `resp.tickets` / etc. has to be updated to read `resp.items`. Runbook §6 lists this as a required step per service.

**CI gate**: `tools/proto_lint` parses every `ListXxxResponse` message and asserts the field set is exactly `{items, total, limit, offset}` with the locked field numbers (1-4). Deviation fails.

---

## 9. Single-entity create/update response — message **is** the entity

**Rule**: For `CreateXxx` / `UpdateXxx` / `GetXxx` / `ToggleXxx`, the response message **is** the entity itself, with no `{entity: ...}` wrapper.

**Bad** (today's REST style):
```go
c.JSON(http.StatusCreated, gin.H{"promo_code": promoCode})
```

**Good** (Connect handler):
```go
func (s *Server) CreatePromoCode(
    ctx context.Context, req *connect.Request[promov1.CreatePromoCodeRequest],
) (*connect.Response[promov1.PromoCode], error) {
    pc, err := s.svc.Create(ctx, ...)
    if err != nil { return nil, connect.NewError(connect.CodeInternal, err) }
    return connect.NewResponse(toProto(pc)), nil
}
```

```protobuf
service PromoCodeService {
  rpc CreatePromoCode(CreatePromoCodeRequest) returns (PromoCode);   // returns the entity
}
```

**Rationale**: PR 986a38ca6 was a 7-service sweep specifically because `post_resource(endpoint, body, "promo_code")` unwrapping was dropping sibling fields. Removing the wrapper removes the entire failure mode.

**Exception** — `CreateXxxResponse` is permitted only if the operation returns **multiple** things alongside the entity (e.g., `{pod: Pod, warning: string}` from 986a38ca6's pod-quota flow). In that case the response message is named `CreateXxxResponse` and explicitly lists each field. The wrapper is intentional, not implicit.

**CI gate**: round-trip test for each create/update — assert the deserialized struct fields match the entity 1:1, no `entity` nesting on the wire.

---

## 10. Error model — Connect standard, not `{error: "..."}`

**Rule**: Handlers return errors via `connect.NewError(connect.CodeXxx, err)`. Connect serializes to the standard error envelope:

```json
{
  "code": "not_found",
  "message": "skill registry 42 not found"
}
```

Code mapping mirrors gRPC:

| Existing Gin handler | Connect code |
|---|---|
| `apierr.AbortUnauthorized` | `connect.CodeUnauthenticated` |
| `apierr.NotFound` | `connect.CodeNotFound` |
| `apierr.Conflict` | `connect.CodeAlreadyExists` |
| `apierr.ValidationError` | `connect.CodeInvalidArgument` |
| `apierr.Forbidden` | `connect.CodePermissionDenied` |
| `apierr.InternalError` | `connect.CodeInternal` |

**Bad**:
```go
return nil, errors.New("registry not found")                                    // raw error → 500
return connect.NewResponse(&pb.SkillRegistry{}), nil                            // empty struct on error
return nil, connect.NewError(connect.CodeUnknown, fmt.Errorf("not found"))       // wrong code
```

**Good**:
```go
if errors.Is(err, extension.ErrNotFound) {
    return nil, connect.NewError(connect.CodeNotFound, err)
}
if errors.Is(err, extension.ErrAlreadyExists) {
    return nil, connect.NewError(connect.CodeAlreadyExists, err)
}
return nil, connect.NewError(connect.CodeInternal, err)
```

**Forbidden**: returning a 200 response with `{error: "..."}` in the body. Existing REST handlers do this in a few places (`apierr.RespondWithExtra`) — the Connect equivalent is always a typed error code.

**CI gate**: handler unit test for each error path — assert `connect.CodeOf(err)` matches expected. `grep -r '"error":' proto/` returns non-empty (someone added an `error` field to a response message) → human review required.

---

## 11. Required headers

**Rule**: Every request includes:

| Header | Value | Set by | Validated by |
|---|---|---|---|
| `Connect-Protocol-Version` | `1` | wasm helper auto-injects | `connect-go` library |
| `Content-Type` | `application/proto` (MANDATORY for client) | wasm helper hard-codes | `connect-go` content negotiator |
| `Authorization` | `Bearer <jwt>` | wasm helper, from `AuthManager` token store | auth interceptor |

`Connect-Protocol-Version: 1` is **not** an API version. It refers to the Connect protocol spec version. The wasm-side wrapper must inject it on every call. Forgetting it makes `connect-go` reject the request with `unsupported_protocol`.

`Content-Type: application/proto` is **hard-coded** in the wasm + TS request helpers (see §2.5). The body is a prost-encoded `Vec<u8>` / `Uint8Array`. There is no per-service flag.

**Bad**:
```rust
let res = reqwest::Client::new()
    .post(&url)
    .json(&req)                                  // JSON path forbidden on client
    .send().await?;                              // missing headers
```

**Good** (wasm helper, factored once and reused — binary in, binary out):
```rust
pub async fn connect_call<Req, Res>(
    client: &ApiClient,
    procedure: &str,
    body: &Req,
) -> Result<Res, ApiError>
where
    Req: prost::Message,
    Res: prost::Message + Default,
{
    let url = format!("{}{}", client.base_url(), procedure);
    let bytes = body.encode_to_vec();
    let mut req = client.http.post(&url)
        .header("Connect-Protocol-Version", "1")
        .header("Content-Type", "application/proto")
        .body(bytes);
    if let Some(tok) = client.token().await {
        req = req.header("Authorization", format!("Bearer {tok}"));
    }
    let resp = req.send().await.map_err(ApiError::Http)?;
    if !resp.status().is_success() {
        return Err(ApiError::from_connect_response(resp).await);
    }
    let resp_bytes = resp.bytes().await.map_err(ApiError::Http)?;
    Res::decode(resp_bytes).map_err(ApiError::Decode)
}
```

**CI gate**:
1. Round-trip test from wasm path against test backend — backend rejects requests without `Connect-Protocol-Version: 1`, so the test fails fast if the wasm helper is bypassed.
2. `grep -r '"application/json"' clients/core/crates/api-client/src/ clients/core/crates/wasm/src/ clients/web/src/lib/api/` returns non-empty → fail (see §2.5).

---

## 12. Service URL — `/<package>.<Service>/<Method>`

**Rule**: The canonical URL is the Connect canonical form. The router (`backend/cmd/server/connect_init.go` `connectPathPrefix = "/proto."`) dispatches everything under `/proto.` to the Connect mux.

| Pattern | Example |
|---|---|
| `/<package>.<Service>/<Method>` | `/proto.extension.v1.SkillRegistryService/ListSkillRegistries` |

**Bad** — manual prefix or trailing slash:
```
/api/v1/proto/extension/skill_registries        # REST-style path
/proto.extension.v1.SkillRegistryService.ListSkillRegistries  # dots instead of /Method slash
```

**Good**:
```
/proto.extension.v1.SkillRegistryService/ListSkillRegistries
/proto.pod.v1.PodService/CreatePod
```

`connect-go`'s constants `<Service>Name + "/" + <Method>Procedure` generate these — derive from constants, never hand-string-format.

**CI gate**: each Connect handler's `Mount(mux *http.ServeMux, s *Server)` uses the procedure constant from the service file. The procedure constant must equal `"/" + ServiceName + "/" + MethodName` — test asserts this string identity per service.

---

## Quick reference card

```
codec        : Client = protobuf binary ONLY (application/proto). Server negotiates (admin curl can JSON).
package      : proto.<domain>.v1
service      : <Domain>Service
method       : VerbNoun (List, Create, Update, Delete, Get, Toggle, Sync)
field name   : snake_case in .proto. Rust prost struct = snake_case. NO serde derives on migrated DTOs.
org_slug     : org-scoped RPCs MUST have `string org_slug = 1;` as field 1 (User/Admin RPCs exempt)
prost tag    : matches .proto field number, never reused, gaps go to `reserved`
optional     : proto3 `optional` keyword wherever zero is semantically meaningful
timestamps   : `string` ISO-8601, NOT google.protobuf.Timestamp
oneof wire   : binary tag-number. Rust = #[derive(prost::Oneof)]. Server JSON shape (curl only) = {<field>: {<variantName>: ...}} tagged.
list resp    : {items, total, limit, offset} — UNIFORM across 26 services
create resp  : the entity itself (no {entity: ...} wrapper) — exception: multi-field returns
error        : connect.NewError(connect.CodeXxx, err) — never {error: "..."}
headers      : Connect-Protocol-Version: 1, Content-Type: application/proto, Authorization: Bearer <jwt>
URL          : /<package>.<Service>/<Method>, prefix /proto. routed by connect_init.go
```
