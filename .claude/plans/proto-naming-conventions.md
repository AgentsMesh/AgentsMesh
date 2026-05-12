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

## 3. Field naming on the wire — `snake_case` in `.proto`, `camelCase` on the wire, no `json_name`

**Rule**: Field names in `.proto` are `snake_case`. Connect's `protojson` automatically maps `snake_case` ↔ `camelCase`. Rust DTOs declare `#[serde(rename_all = "camelCase")]`. **Never** use `json_name = "..."` annotations — they introduce a third name and split the brain.

This is the **single most important rule**. Every drift bug in PR #329–#368 was a casing mismatch.

**Bad**:
```protobuf
message SkillRegistry {
  string repositoryUrl = 1;                          // camelCase in .proto
  string is_active = 2 [json_name = "isActive"];     // json_name annotation
}
```

```rust
#[derive(Serialize, Deserialize)]
// MISSING: #[serde(rename_all = "camelCase")]
pub struct SkillRegistry {
    pub repository_url: String,    // serde keeps snake_case → wire mismatch
}
```

**Good**:
```protobuf
message SkillRegistry {
  string repository_url = 1;       // snake_case in .proto
  bool is_active = 2;              // protojson emits "isActive" automatically
}
```

```rust
#[derive(Clone, PartialEq, Message, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct SkillRegistry {
    #[prost(string, tag = "1")] pub repository_url: String,
    #[prost(bool, tag = "2")]   pub is_active: bool,
}
```

**CI gate**:
1. `buf lint` rule `FIELD_LOWER_SNAKE_CASE` rejects camelCase in `.proto`.
2. `grep -r 'json_name' proto/` returns non-empty → fail.
3. Per-service round-trip test (see runbook §7) decodes a recorded backend JSON payload, re-serializes through serde, asserts byte-equal — catches a missing `rename_all`.

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
#[derive(Message, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ListSkillRegistriesRequest {
    #[prost(int32, optional, tag = "1")]
    #[serde(skip_serializing_if = "Option::is_none")]
    pub offset: Option<i32>,
    #[prost(int32, optional, tag = "2")]
    #[serde(skip_serializing_if = "Option::is_none")]
    pub limit: Option<i32>,
}
```

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
#[derive(Message, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct SkillRegistry {
    #[prost(string, tag = "1")]
    pub last_synced_at: String,
}
```

**Use `Option<String>` for nullable timestamps** (most "last seen at" / "ended at" fields).

**CI gate**: `grep -r 'google.protobuf.Timestamp' proto/` returns non-empty → fail (until we explicitly lift this restriction).

---

## 7. `oneof` — tagged shape `{kind: {variantName: ...}}`, never untagged

**Rule**: `oneof` fields encode on the JSON wire as `{<oneof_field_name>: {<variantName>: <value>}}`. Variant names are camelCase (matching the field name in protojson rules). The Rust side uses an untagged enum with a custom deserialize matcher.

This disagreement between prost's default Rust codegen and protojson is the **#2 likeliest migration hazard** (POC §3.4).

**Bad** — wasm sends one shape, backend expects another:
```json
{
  "type": "POD_CREATED",        // discriminator-as-string
  "pod_id": "p-123"
}
```

**Good** — Connect protojson canonical:
```json
{
  "event": {
    "podCreated": {              // variantName matches oneof variant field
      "podId": "p-123"
    }
  }
}
```

```protobuf
message PodEvent {
  oneof event {
    PodCreatedData pod_created = 1;
    PodTerminatedData pod_terminated = 2;
  }
}
```

```rust
#[derive(Clone, PartialEq, Message)]
pub struct PodEvent {
    #[prost(oneof = "pod_event::Event", tags = "1, 2")]
    pub event: Option<pod_event::Event>,
}

pub mod pod_event {
    use serde::{Deserialize, Serialize};

    #[derive(Clone, PartialEq, prost::Oneof)]
    pub enum Event {
        #[prost(message, tag = "1")]
        PodCreated(super::PodCreatedData),
        #[prost(message, tag = "2")]
        PodTerminated(super::PodTerminatedData),
    }

    // Manual serde to emit/parse {"podCreated": {...}} tagged shape.
    impl Serialize for Event {
        fn serialize<S: serde::Serializer>(&self, s: S) -> Result<S::Ok, S::Error> {
            use serde::ser::SerializeMap;
            let mut m = s.serialize_map(Some(1))?;
            match self {
                Event::PodCreated(v)    => m.serialize_entry("podCreated", v)?,
                Event::PodTerminated(v) => m.serialize_entry("podTerminated", v)?,
            }
            m.end()
        }
    }
    // Deserialize: match the single key, dispatch by name. (See runbook for full template.)
}
```

**CI gate**: any service introducing a `oneof` adds a round-trip test that encodes each variant, asserts the JSON has exactly the tagged shape, and decodes it back to the same enum value. Without that test, do not merge.

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
| `Content-Type` | `application/json` (default) or `application/proto` | wasm helper picks per-service flag | `connect-go` content negotiator |
| `Authorization` | `Bearer <jwt>` | wasm helper, from `AuthManager` token store | auth interceptor |

`Connect-Protocol-Version: 1` is **not** an API version. It refers to the Connect protocol spec version. The wasm-side wrapper must inject it on every call. Forgetting it makes `connect-go` reject the request with `unsupported_protocol`.

**Bad**:
```rust
let res = reqwest::Client::new()
    .post(&url)
    .json(&req)
    .send().await?;     // missing Connect-Protocol-Version + Authorization
```

**Good** (wasm helper, factored once and reused):
```rust
pub async fn connect_call<Req: Serialize, Res: DeserializeOwned>(
    client: &ApiClient,
    procedure: &str,
    body: &Req,
) -> Result<Res, ApiError> {
    let url = format!("{}{}", client.base_url(), procedure);
    let mut req = client.http.post(&url).json(body);
    req = req.header("Connect-Protocol-Version", "1");
    if let Some(tok) = client.token().await {
        req = req.header("Authorization", format!("Bearer {tok}"));
    }
    let resp = req.send().await?;
    if !resp.status().is_success() {
        return Err(ApiError::from_connect_error(resp).await);
    }
    Ok(resp.json::<Res>().await?)
}
```

**CI gate**: round-trip test from wasm path against test backend — backend rejects requests without `Connect-Protocol-Version: 1`, so the test fails fast if the wasm helper is bypassed.

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
package      : proto.<domain>.v1
service      : <Domain>Service
method       : VerbNoun (List, Create, Update, Delete, Get, Toggle, Sync)
field name   : snake_case (.proto) ↔ camelCase (wire) ↔ #[serde(rename_all = "camelCase")] (Rust)
prost tag    : matches .proto field number, never reused, gaps go to `reserved`
optional     : proto3 `optional` keyword wherever zero is semantically meaningful
timestamps   : `string` ISO-8601, NOT google.protobuf.Timestamp
oneof JSON   : {<field>: {<variantName>: ...}} tagged
list resp    : {items, total, limit, offset} — UNIFORM across 26 services
create resp  : the entity itself (no {entity: ...} wrapper) — exception: multi-field returns
error        : connect.NewError(connect.CodeXxx, err) — never {error: "..."}
headers      : Connect-Protocol-Version: 1 (mandatory), Authorization: Bearer <jwt>
URL          : /<package>.<Service>/<Method>, prefix /proto. routed by connect_init.go
```
