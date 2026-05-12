# Proto Migration Watch List — 26-Service Hazard Map

Eight known migration hazards, ordered by **likelihood of biting**. Each entry: **symptom → detection → mitigation**.

This document is companion to:
- ADR (`proto-migration-adr.md`) — the why
- Conventions (`proto-naming-conventions.md`) — the SHALL rules
- Runbook (`proto-migration-runbook.md`) — the how-to

POC §3.4 originally ranked five raw hazards. Hazard #1 (camelCase drift) was eliminated by the team-lead's binary-only-on-client pivot (conventions §2.5) — it now stands as a closed entry for historical traceability. Hazard #8 (tag-number drift in hand-maintained prost structs) is the new equivalent risk and is the single most important reason `tools/validate_prost_tags` lands with the first migration PR.

---

## 1. ~~camelCase / snake_case inconsistency~~ — **physically eliminated by binary wire**

### Status

**Closed by codec choice, not by lint discipline.** The client wire is `application/proto` binary only (conventions.md §2.5). Fields on the wire are identified by `prost(tag = N)` / `.proto` field number, not by string name. The whole class of "wasm sends `is_active`, backend reads `isActive`, field arrives as `undefined`" cannot happen — there are no field names on the binary wire.

The historical bug arc that owned this slot:

- **PR #329** — `RepositoryProvider.is_active/has_*` → would now be a tag-number lookup, can't miss.
- **PR #334** — `PricingConfig.currency` vs `currencies` (singular/plural drift) → still possible (it's a different field), but caught at compile time because the Rust prost struct mirrors the `.proto` field name 1:1, not a casing transform.
- **PR #341 / #349 / #368** — `SkillRegistry` list wrapper key → solved by §8 uniform `{items, total, limit, offset}` envelope, orthogonal to wire codec.
- **commit 986a38ca6** — sweep of seven wrapper drifts → solved by §8 + §9 (entity-direct create response), orthogonal to wire codec.

### What remains on this slot

The replacement risk — **tag number drift in hand-maintained prost structs** — is hazard #8 below. Same severity, different mechanism: instead of catching string-name typos, CI now catches numeric tag mismatches between `.proto` and Rust.

### Why the rule moved

POC §3.4 listed JSON casing as the #1 hazard *given* the POC's JSON wire. The team-lead's binary-only pivot turns that hazard into a non-event: no JSON wire, no string field names on the wire, no casing drift surface. The lint discipline (`#[serde(rename_all = "camelCase")]`, `buf lint FIELD_LOWER_SNAKE_CASE` on the **rust side**) that this section used to enforce is **deleted** in conventions.md §3.

### Residual buf lint

`FIELD_LOWER_SNAKE_CASE` on `.proto` still runs — keeps the source files uniform and keeps the server-side curl-debug JSON path consistent. But this is hygiene, not the bug-prevention mechanism it used to be.

---

## 2. `oneof` encoding — **mostly closed by binary wire; one residual surface**

### Status

Binary wire encodes `oneof` as the variant's nested message at its tag number. prost's stock `#[derive(prost::Oneof)]` enum is the canonical Rust shape. **No custom serde, no JSON encoding negotiation, no cross-codec mismatch.**

The original symptom — "prost's untagged JSON conflicts with protojson's tagged shape" — was a JSON-wire problem. On binary wire there is no JSON to disagree about.

### Residual surface

Server-side curl debugging hits `application/json` and uses protojson, which encodes `oneof` as `{<oneof_field>: {<variantName>: <payload>}}`. Variant field names go through snake_case → camelCase via protojson rules. This affects **only admin scripts that hand-author JSON payloads** — wasm/TS clients never see it.

### Detection

1. **Per-variant round-trip test** — `prost::Message::encode_to_vec` each variant, decode back, assert PartialEq. Tag-number drift between `.proto` and `tags = "..."` in the Rust enum surfaces here.
2. **buf lint** rule `ONEOF_LOWER_SNAKE_CASE` — variant field names must be `snake_case` so the server-side protojson curl path stays consistent (rare path, but kept for hygiene).

### Mitigation

- Use prost's stock `#[derive(prost::Oneof)]`. **Do not** hand-write `Serialize` / `Deserialize` for oneof enums — conventions.md §7 explicitly forbids it.
- The "avoid `oneof` in v1" advice from the legacy version of this section is **dropped**. Binary wire makes `oneof` boring; specialist agents may use it freely where the domain calls for it.
- Specialist agents must still flag `oneof` in PR description so reviewers eyeball the tag-number annotations.

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
- **Binary wire handles this automatically** — `Option<T>` in the prost struct maps to "field present at tag N" vs "field absent". No `serde(skip_serializing_if)` needed; that was a JSON-wire workaround. The Rust struct's `Option<i32>` plus `#[prost(int32, optional, tag = "N")]` is the whole story.

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
- **`prost::Message` is mandatory** on every migrated DTO — binary wire requires it (conventions.md §2.5). The legacy "fall back to serde-only" escape valve is **gone**; there is no JSON lane to fall back to. If a service's bundle delta is unacceptable, the lever is message-tree shape (split into sub-messages, share with `proto.common.v1`), not codec choice.
- **No tokio/rustls/mio in wasm** (PR #349) still holds — `prost` is pure-Rust, no IO crates pulled in. Binary wire keeps this property.

---

## 8. Tag number stability (replaces the JSON casing slot)

### Background

Binary wire identifies each field by its **tag number** — the integer in `prost(tag = N)` and the integer after `=` in `.proto`. Field names are local labels with no wire presence.

This is the source of the binary lane's drift immunity. It is also a **single new failure mode** for hand-maintained prost structs (the path the POC chose, see ADR §"Codegen"): if a service migration types `tag = "7"` in Rust but the `.proto` says `tag = 8`, the wire decodes the wrong field into the wrong slot **and the type system doesn't catch it** (both might be `String`).

### Mitigation surface

`@bufbuild/protoc-gen-es` (TS proto codegen) and `protoc-gen-go` (Go proto codegen) **automatically** generate tag-correct code from `.proto`. The only hand-maintained surface is the Rust prost struct in `clients/core/crates/types/src/<service>.rs`. **This is single-point risk.**

### Detection

1. **`tools/validate_prost_tags` Bazel rule** (NEW, must land with the first service migration PR — `skill_registry` reference impl).

   Implementation sketch:
   - Parse `.proto` field declarations → `{message: {field_name: tag_number}}`.
   - Parse Rust source via `syn` → `{struct: {field_name: tag_number}}` (extracted from `prost(tag = "N")` attribute).
   - Assert one-to-one mapping. Fail at build time on mismatch.
   
   Reference targets: every `proto_library` gets a paired `validate_prost_tags(name="<svc>_validate", proto=":<svc>_proto", rust=":<svc>_rust")`.

2. **Per-message round-trip test in Rust** (already mandated in runbook §7.1, repurposed):
   ```rust
   let original = FooResponse { /* every field set to a distinguishing value */ };
   let bytes = original.encode_to_vec();
   let decoded = FooResponse::decode(&*bytes).unwrap();
   assert_eq!(original, decoded);
   ```
   A swapped tag pair (e.g., tag 2 and tag 3 transposed) shows up as field-value swap in the assertion.

3. **buf rules** on the `.proto` side:
   - `RESERVED_TAG` — reserved tag numbers can't be reused.
   - `PACKAGE_NO_IMPORT_CYCLE` — keeps tag-numbering schemes isolated per package.
   - `FIELD_LOWER_SNAKE_CASE` — orthogonal to tags but keeps `.proto` source consistent.

### What this does NOT catch

A field whose **type** matches but **semantics** are wrong is unrecoverable from tag-level checks alone. E.g., if `.proto` says `int32 user_id = 5;` and the Rust struct has `int32 group_id = 5;`, both encode/decode fine, the names just diverge. The `validate_prost_tags` rule MUST also compare **field names** between the two sources, treating snake_case ↔ snake_case identity as the contract (no case translation).

### Scope

The risk is bounded to the **Rust hand-written DTO** surface. Once the codegen tool (`tools/gen_rust_proto` per ADR §5) lands and the 26 services migrate to generated `.rs`, this hazard goes from "single-point" to "compiler-enforced" — the codegen reads the `.proto` as ground truth, can't transcribe wrong. Until then, this rule is the safety net.

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

### Tenant scope field placement

Specialist agents must place `string org_slug = 1;` as the first field of every org-scoped request message (conventions §3.5). The auth-interceptor agent surfaced this: Connect URLs have no path params, so the existing REST `:slug` middleware-injection pattern does not work — payload-carried `org_slug` is the only path.

**Detection**:
- Custom linter rule on `tools/proto_lint`: every RPC request message has `string org_slug = 1;` OR matches the user-scoped / admin whitelist.
- Per-handler test: send a request with empty `org_slug` → assert `connect.CodeInvalidArgument`.

**Mitigation**:
- Use the generic `authinterceptor.ResolveOrgScope[T]` helper (introduced by the first migrating service, locked in conventions §3.5).
- Wasm side reads org slug from the `AuthManager`-held current org (`client.org_slug()`), not from TS arguments — keeps TS call sites unchanged.

If an agent invents `organization_slug` or nests `org_slug` inside a sub-message, this is a **drift surface even without bugs shipping** — the helper signature breaks generic reuse, every service writes bespoke org-resolution boilerplate.

### Two services sharing a Rust type

If `proto.pod.v1.Pod` and `proto.autopilot.v1.Pod` both define a `Pod` message, they generate two different Rust types in two different modules — they will **not** interop. Specialist agents must:
- Either share a `proto.shared.v1.Pod` and import it from both services.
- Or accept that the types are distinct and convert at the service boundary.

For v1, prefer **shared messages in `proto.<domain>.v1`** only if the domain is genuinely shared (e.g., `proto.common.v1.Pagination`). Cross-service entity reuse stays per-service to avoid coupling migrations.

---

## Summary table

| # | Hazard | Severity | Detection mechanism | Mitigation |
|---|---|---|---|---|
| 1 | ~~camelCase drift~~ | **CLOSED** by binary wire | (codec choice eliminates surface) | conventions §2.5 — no client JSON path |
| 2 | `oneof` encoding | LOW (was MEDIUM) | per-variant round-trip | prost-stock `#[derive(prost::Oneof)]`, no custom serde |
| 3 | Optional vs default scalar | MEDIUM | per-RPC `offset=0` test | `optional` keyword |
| 4 | `Timestamp` introduction | LOW (gated) | grep CI gate | use `string` ISO-8601 |
| 5 | Auth interceptor gap | **HIGH** | "401 without bearer" test | block on interceptor PR |
| 6 | Backend field accumulation | **HIGH** (re-opens issues) | three-way diff before `.proto` | runbook §1 mandates the diff |
| 7 | wasm bundle creep | LOW | per-PR delta + cumulative budget | per-service ceiling; no codec fallback (binary mandatory) |
| 8 | Tag number drift in Rust prost structs | **HIGH** (replaces #1's slot) | `tools/validate_prost_tags` Bazel rule + round-trip test | codegen tool emits Rust from `.proto`; hand-written DTOs are interim only |
