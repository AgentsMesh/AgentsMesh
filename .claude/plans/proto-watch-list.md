# Proto Migration Watch List — 26-Service Hazard Map

Seven known migration hazards, ordered by **likelihood of biting**. Each entry: **symptom → detection → mitigation**.

This document is companion to:
- ADR (`proto-migration-adr.md`) — the why
- Conventions (`proto-naming-conventions.md`) — the SHALL rules
- Runbook (`proto-migration-runbook.md`) — the how-to

POC §3.4 covers items 1–5 in raw form; this expands them to actionable.

---

## 1. camelCase / snake_case inconsistency — **the #1 historical bug class**

### Symptom

Field arrives at TS as `undefined`. UI renders `${undefined}${undefined}` (literal "undefinedundefined"), badges stay at 0, paginators frozen at page 1, secrets render empty, list responses look empty even though DB rows exist.

This single root cause owns the entire drift bug arc:

- **PR #329** — `RepositoryProvider.is_active/has_*` dropped → every provider "disabled".
- **PR #334** — `PricingConfig.currency` (singular) vs Rust `currencies` (vector) → pricing card `${undefined}${undefined}`.
- **PR #341** + **#349** + **#368** — `SkillRegistry` list wrapper key drifted through wasm relay → UI list empty, re-adding same repo triggered unique-key violation (user reopened issue twice).
- **commit 986a38ca6** — sweep of seven simultaneous drifts: `{pod, warning}`, `{tickets, total, limit, offset}`, `{has_more}`, `{unread_count}`, `{raw_key}`, `{replayed_message}`.
- **PR #340** — resume pod missing `agent_slug`.

### Detection

1. **buf lint** rule `FIELD_LOWER_SNAKE_CASE` — fails on camelCase in `.proto`.
2. **Round-trip test per response message** — record a real backend JSON payload (or hand-author one matching the handler), deserialize it into the Rust struct, re-serialize, assert byte-equality with a normalized form. Catches any missing `#[serde(rename_all = "camelCase")]`, any field-name typo.
3. **Wasm relay test** — same round-trip but going through `wasm_pkg`: TS calls the service, Rust deserializes-then-reserializes, TS sees same shape it would see calling backend directly.
4. **CI grep gate**: `! grep -r 'json_name' proto/` — disallows the third-name escape hatch.

### Mitigation

- **Codegen template forces `#[serde(rename_all = "camelCase")]` on every response message.** Never hand-write a Rust DTO; always generate.
- **No `json_name` annotations in `.proto`.** Conventions §3.
- **Round-trip test is non-optional.** Specialist agents do not merge a service PR without the test (runbook §7).

---

## 2. `oneof` encoding disagreement

### Symptom

`oneof` fields are silently lost or misinterpreted between Rust and Go.

- **prost** generates a Rust enum: `Event::PodCreated(PodCreatedData)` with no inherent JSON shape.
- **protojson** (Go side) encodes `oneof` as a tagged shape: `{"event": {"podCreated": {...}}}`.
- A naive `serde(untagged)` enum on the Rust side encodes as `{"podCreated": {...}}` — outer `event` field missing, backend can't decode.
- Conversely, a `serde(tag = "type")` discriminator (e.g., `{"type": "POD_CREATED", "podId": "..."}`) is what some hand-written DTOs use today; protojson does **not** speak this dialect.

### Detection

1. **Round-trip test per `oneof` variant**: encode each variant from Rust → assert exact JSON shape → decode back → assert PartialEq with the original.
2. **Cross-language wire test**: stand up the Connect handler, send a hand-authored `application/json` payload matching protojson's documented form via `curl`, assert handler decodes correctly. Reverse: encode in Rust, send to handler, assert handler reaches expected branch.
3. **buf lint** rule `ONEOF_LOWER_SNAKE_CASE` — variant field names must be `snake_case` so camelCase conversion is unambiguous.

### Mitigation

- **Adopt the tagged shape uniformly**: `{<oneof_field>: {<variantName>: <payload>}}`. Implement custom `Serialize` + `Deserialize` for each Rust `oneof` enum (conventions §7 has the template).
- **No `oneof` in v1 of any service if avoidable.** Several existing REST endpoints would use a `oneof` naturally (e.g., the `PodEvent` stream). For the v1 migration, prefer flat messages with discriminator strings if the field set is small (<5 variants). Add `oneof` later as a non-breaking widening if needed.
- **Specialist agents must flag `oneof` usage in PR description** and include the variant round-trip test in the test plan.

---

## 3. Optional vs default scalar — **breaks pagination silently**

### Symptom

Pagination `offset = 0` is indistinguishable from "no offset specified". On proto3, scalar defaults are absent from the wire; on the Rust prost side, `count: i32 = 0` looks the same whether the field was sent or not.

- **Today**: REST handlers use `c.DefaultQuery("offset", "0")` — query strings preserve presence.
- **After migration**: protobuf body has `offset = 0` → wire byte sequence elides the field → backend can't tell "first page" from "default page size 20".

This bit the existing `ListComments` endpoint (`backend/internal/api/rest/v1/ticket_comments.go:50`) before the wrapper-envelope fix: an `offset=0` request from infinite-scroll's first call returned the same page as `offset` missing, leading to a stuck paginator. After 986a38ca6, the fix was wrapper-side but the underlying optional-scalar trap is still there for any service that migrates.

### Detection

1. **Per list-RPC test**: assert that `req with offset=0 explicit` and `req with offset absent` lead to **distinguishable handler behavior**.
2. **buf lint rule** (custom): every `int32`/`int64` field whose name is in `{offset, limit, page, count, page_size}` MUST be declared `optional`.
3. **Code review checklist**: counters and quotas (`usage`, `seats_used`, `seats_remaining`) also need `optional` if zero is meaningful.

### Mitigation

- **Use proto3 `optional` keyword.** Conventions §5 has the full pattern.
- **Server-side default convention**: if `offset` is absent, default to 0 anyway — semantics match REST behavior. The point of `optional` is **client intent visibility**, not server semantics.
- **Pair with `#[serde(skip_serializing_if = "Option::is_none")]`** on the Rust side so omitted Options don't serialize as `null`.

---

## 4. `google.protobuf.Timestamp` introduction cost

### Symptom

A specialist agent reaches for `google.protobuf.Timestamp` for "last_synced_at" or "created_at", encounters:

1. `prost` doesn't include WKT decoders by default — need `prost-types` crate.
2. Adding `prost-types` to `MODULE.bazel` is a 4-line edit but pulls another crate, +bundle size, +another piece of the codegen tool's translation table.
3. The Rust ⇄ JSON encoding for `Timestamp` matches protojson's `2026-05-12T13:16:10Z` exactly — but only via `prost-types`'s `pbjson_types::Timestamp` re-export, not the plain `prost-types::Timestamp`. A subtle import mistake silently emits Unix epoch seconds or breaks at runtime.

### Detection

1. **Hard CI gate**: `! grep -r 'google.protobuf.Timestamp' proto/` returns non-empty → fail. Until the team explicitly decides to lift this, no service introduces it.
2. **Linter check on Rust side**: `! grep -r 'prost_types::Timestamp\|prost-types' clients/core/crates/wasm/` — same restriction at the import level.

### Mitigation

- **Use `string` ISO-8601 for v1.** Conventions §6. Backend already emits this from `time.Time` fields via `gin.H` JSON marshal.
- **Use `Option<String>` for nullable timestamps** (e.g., `last_synced_at` is `null` before first sync).
- **If a future service genuinely needs typed `Timestamp`** (e.g., arithmetic on backend without re-parsing ISO-8601), open a separate ADR — do not slip it into a migration PR.

---

## 5. Auth interceptor gap

### Symptom

A migrated service ships **unauthenticated**. The Gin handlers had `middleware.AuthMiddleware(jwtSecret)` on the router group; the Connect handler is mounted on a bare `http.ServeMux` with no interceptor.

POC §3.4 calls this out explicitly: *"This POC has no auth on the Connect handler. The production version will need `connect.WithInterceptors(authInterceptor)`."*

### Detection

1. **Curl test in CI**: every migrated service has a "401 without bearer" test — call the Connect endpoint with **no `Authorization` header**, assert `connect.CodeUnauthenticated` returned.
2. **Tenant isolation test**: call with a bearer for org A but `:slug` param of org B → assert `connect.CodePermissionDenied`.

### Mitigation

- **Block the migration on the auth interceptor PR.** This is in flight on a parallel agent. Until that lands on `feat/proto-migration`, no service migration starts.
- **Centralize the interceptor**: `backend/internal/api/connect/interceptor/auth.go` — mirrors `middleware.AuthMiddleware`, parses `Authorization: Bearer <jwt>`, populates a `context.Context` value `userIDKey`, `emailKey`, `usernameKey`. Per-handler boilerplate: `userID := connect_auth.UserID(ctx)`.
- **Tenant isolation interceptor** (org slug check): orthogonal to auth, mounted as a second interceptor. Mirrors the existing `middleware.TenantIsolation`.
- **Runbook §3 includes the interceptor usage template** — every handler reads `userID` from context, never from the request body.

---

## 6. Backend-side "additional field" trap — SkillRegistry case study

### Symptom

The current backend has accumulated **fields that were added late** to the existing REST DTO. Examples on `SkillRegistry`:

- `last_synced_at` — added when marketplace sync was wired up.
- `last_commit_sha` — added for cache invalidation.
- `sync_status`, `sync_error` — added for the sync UI.
- `skill_count` — added for the dashboard count badge.

Issue #341 stayed open across **three PRs** in part because each PR fixed *some* of these fields and shipped, then the user reopened with a different symptom from a different field.

When migrating a service to `.proto`, the agent reads the **current** REST handler's response shape and writes the `.proto` to match. **But the agent must also**:

1. Read the **Rust DTO** to spot fields the REST handler doesn't currently emit but the Rust layer expects (a hint a backend change was incomplete).
2. Read the **TS consumer code** to spot fields the UI reads but the Rust DTO doesn't have (another drift signal).
3. Read **the open issues** for the domain — anything still flagged as "field X doesn't work" is a field the `.proto` must include.

A `.proto` that **omits an existing field** is a wire regression — older TS clients reading that field break.

### Detection

1. **Three-way diff per service**: at the start of a migration, dump the field names from:
   - Backend handler (`grep -A 30 "gin.H{" backend/internal/api/rest/v1/<service>.go`)
   - Rust DTO (`grep "pub " clients/core/crates/types/src/<service>.rs`)
   - TS consumer (`grep -r "<dto>\." clients/web/src/`)
2. Reconcile the three lists. **All fields union into the `.proto`.** Any difference is a drift signal — investigate before writing `.proto`.
3. **Vitest mock alignment**: existing mocks in `clients/web/src/test/setup.ts` and Rust `api_core_tests.rs` are sometimes **written against the drifted shape** (PR #368 root cause). Update both to match the new `.proto`-defined shape.

### Mitigation

- **Runbook §1 (predicate phase)** mandates the three-way diff before any `.proto` is written.
- **CI gate**: round-trip test must include **every field** from the three sources, not just the ones the handler emits today. Adding a field later is harmless (tag-stable); omitting one is a regression.
- **Code-review checklist**: a reviewer's first action is to compare `.proto` field list to the existing Rust DTO field list. Missing fields = blocking comment.

---

## 7. wasm bundle size accumulation

### Symptom

Each migrated service adds ~5 KB of Rust DTO + prost decoder to the wasm bundle. 26 services × ~5 KB = ~+130 KB. Plus the +114 KB baseline cost prost contributed in Phase 0 (POC §5). Total: ~22.5 MB after migration vs ~21.3 MB before.

Two concerns:

1. **Marketing pages must stay 0-wasm**. Already enforced by `clients/web/scripts/check-no-wasm-in-marketing.sh` (introduced in PR #349). Migration is invisible to this check **because** marketing pages don't import service modules. Still — a migration PR that accidentally imports `agentsmesh-wasm` from a marketing component would slip through if the check is bypassed.
2. **30% bundle-growth threshold**: ad-hoc rule. 22.5 MB / 21.3 MB = +5.6%. Well under 30%. But each new service is **monotonically additive** — no service will *reduce* bundle size — so the trajectory matters. If we add 100 more services post-migration, we hit the limit.

### Detection

1. **Per-PR bundle-size check**: CI emits `wasm_pkg_bg.wasm` size before and after the PR diff, posts the delta as a PR comment. >2% jump from a single service PR is suspicious (the per-service ceiling is ~10 KB ≈ +0.05%).
2. **Per-migration absolute check**: ` clients/web/scripts/check-no-wasm-in-marketing.sh` runs unchanged.
3. **Cumulative budget**: add `clients/web/scripts/check-wasm-budget.sh` — fails if `wasm_pkg_bg.wasm` > 25 MB. 12% of headroom from today, ample for the migration.

### Mitigation

- **Per-service ceiling**: a single DTO file generating >10 KB suggests the message tree is excessive. Specialist agent splits it into sub-messages (which then deduplicate via prost's message-name interning).
- **Skip the prost binary-codec lane** for services that don't need it. The +50 KB prost runtime is already paid; per-service additional cost for binary support is negligible, so this is rarely a knob. But: a service that returns `string` everywhere and never needs proto-binary can derive only `Serialize/Deserialize` and skip `prost::Message` — saves a few KB per message. Most services don't; this is a fallback if budget pressure spikes.
- **Plan B (cross-platform crates tokio-free, from PR #349)** is already paying off here. The wasm bundle does not pull mio/rustls/tokio's IO stack; migrating to Connect+JSON keeps this property.

---

## Cross-cutting watch points

### Connect-go version drift

`connectrpc.com/connect v1.19.1` is pinned in `go.mod`. Any service PR that bumps it (via `go mod tidy` rounding) breaks the lockstep with the wasm-side helper, which is locked to the v1 protocol. **Lock**: do not bump `connectrpc.com/connect` in a service migration PR. Bumps land in a dedicated infra PR.

### Wasm rebuild on service PR

Every service PR changes wasm crate sources, which forces a full wasm rebuild (~3 min on CI). Acceptable cost. The alternative — splitting wasm into per-service cdylibs — was rejected in PR #349 (wasm-bindgen cannot pass Rust types across cdylib boundaries; total bundle would grow).

### Test mock alignment

`clients/web/src/test/setup.ts` and Rust `api_core_tests.rs` mocks are written **by hand against the drifted shape**. PR #368 exposed this: vitest passed with the drifted shape, only the e2e against real backend caught it. Migration PRs MUST update both mock sources alongside the `.proto`. Runbook §6 lists this as a required step.

### Existing tests that hard-code the wrapper key

```
grep -r '\.skill_registries\|\.tickets\|\.comments\|\.runners' clients/web/src/
```

Each match is a place where TS code reads a per-domain wrapper key. After migration to the uniform `{items, total, ...}` envelope, all these read `.items` instead. **Find them at migration start, fix in the same PR.**

### Type-only TypeScript imports

After deleting `clients/web/src/lib/api/<service>Types.ts`, any TS file that imported types from there breaks. Search:

```
grep -r "from '@/lib/api/<service>Types'" clients/web/src/
```

Replace with imports from the wasm-generated `.d.ts` (or hand-author a thin type alias if the wasm-generated type is awkward).

### Service is unauthenticated by mistake

The auth-interceptor-gap (item 5) bites if a specialist agent uses the `http.ServeMux` registration without going through the wrapper that applies interceptors. **Runbook §3 mandates** the helper `MountWithInterceptors(mux, server, deps)` pattern, never raw `mux.Handle()`.

### Two services sharing a Rust type

If `proto.pod.v1.Pod` and `proto.autopilot.v1.Pod` both define a `Pod` message, they generate two different Rust types in two different modules — they will **not** interop. Specialist agents must:
- Either share a `proto.shared.v1.Pod` and import it from both services.
- Or accept that the types are distinct and convert at the service boundary.

For v1, prefer **shared messages in `proto.<domain>.v1`** only if the domain is genuinely shared (e.g., `proto.common.v1.Pagination`). Cross-service entity reuse stays per-service to avoid coupling migrations.

---

## Summary table

| # | Hazard | Severity | Detection mechanism | Mitigation |
|---|---|---|---|---|
| 1 | camelCase drift | **HIGH** (entire bug arc) | buf lint + round-trip test | codegen template forces `rename_all` |
| 2 | `oneof` encoding | MEDIUM | per-variant round-trip + curl | tagged shape, custom serde |
| 3 | Optional vs default scalar | MEDIUM | per-RPC `offset=0` test | `optional` keyword |
| 4 | `Timestamp` introduction | LOW (gated) | grep CI gate | use `string` ISO-8601 |
| 5 | Auth interceptor gap | **HIGH** | "401 without bearer" test | block on interceptor PR |
| 6 | Backend field accumulation | **HIGH** (re-opens issues) | three-way diff before `.proto` | runbook §1 mandates the diff |
| 7 | wasm bundle creep | LOW | per-PR delta + cumulative budget | per-service ceiling, fallback to derive-only |
