# ADR: Migrate Data Plane to Protobuf + Connect-RPC

- **Status**: Accepted (2026-05-12)
- **Deciders**: backend / web / core team
- **Branch**: `feat/proto-migration` (do **not** merge to `main` until 26 services complete)
- **POC report**: `/Users/stone/Works/AIO/AgentsMesh/.claude/worktrees/agent-a3d7d9878ce9caca4/poc-report.md`

## Context

### Current state — four-layer hand-written DTO chain

The data plane (Browser/Desktop ↔ Backend REST) currently maintains the **same shape four times**:

| Layer | File pattern | Count today |
|---|---|---|
| Backend Go DTO + `gin.H` wrapper | `backend/internal/api/rest/v1/*.go` | 26 service surfaces |
| Rust DTO (`agentsmesh-types`) | `clients/core/crates/types/src/*.rs` | 30 files, ~3.6k LoC |
| Rust api-client method | `clients/core/crates/api-client/src/modules/*.rs` | 32 files |
| Rust wasm service bridge | `clients/core/crates/wasm/src/service_*.rs` | 28 files |
| Web TS DTO (some still duplicated) | `clients/web/src/lib/api/*Types.ts` | 24 files |

Every field rename / addition has to land in all five layers consistently or **silent drift bugs ship to production**. There is no compile-time linkage between layers — the wire is `serde_json::Value`, the contract is convention.

### Bug history (last 3 months) — drift is the dominant bug class

The 10+ bugs traced to this drift, in chronological order:

| PR / Commit | Drift surface | Root cause |
|---|---|---|
| #200 (2026-03-27) | AgentFile DTO | Initial schema, multiple back-and-forth fixes |
| #305 (2026-04-21) | desktop SSOT closure | TS-side store out of sync with Rust |
| #329 (2026-05-06) | `RepositoryProvider.is_active/has_*` | 5 fields silently dropped — every provider rendered "disabled" |
| #334 (2026-05-07) | `PricingConfig.currency` (s) + `Plan.price_*` | Landing page rendered `undefinedundefined` |
| **986a38ca6** (2026-05-09) | **7 wrapper envelopes** in one sweep | `post_resource`/`get_resource` dropped sibling fields — quota toasts, infinite scroll, unread badges, paginators all broken |
| #341 / #349 / #368 | `SkillRegistry` list wrapper key | First attempt fixed item-level fields, second attempt fixed list wrapper key — **issue user re-opened twice** |
| #340 (2026-05-07) | resume pod missing `agent_slug` | Field omission in request body |
| #345 / #342 / #343 (in 986a38ca6) | ApiKey raw_key, ProviderRepository | More wrapper envelope drift |

**Pattern**: each fix is local, the next drift bug ships from a different DTO three weeks later. The audit in 986a38ca6 covered "all 26 wasm-bridged services" but only fixed the seven then-broken ones — the contract has no enforcement, only inspection.

### Why now

The just-landed **Phase 0 scaffolding** (`9e86141e3`) made the next-step economics trivial:

- `wrapWithConnect()` multiplexer in `backend/cmd/server/connect_init.go` routes `/proto.*` paths to a Connect mux, falls through to Gin for everything else. **Zero impact on existing REST.**
- `connectrpc.com/connect v1.19.1` in `go.mod` + `MODULE.bazel`.
- `prost 0.13` in the wasm `crate.spec` block (wasm32 builds — POC verified, +114 KB cost is bounded).
- `backend/internal/api/connect/{BUILD.bazel,doc.go}` package scaffold.
- POC validated **both** Connect+JSON and Connect+proto-binary end-to-end (Rust prost ↔ Go connect-go, JSON via `protojson`, binary via prost codec, wasm32 round-trip from Node).

Adding a service from here is **~80 LoC of new code** (proto + handler + Rust DTO) and zero infrastructure work.

### ROI calculation

| Cost | Estimate |
|---|---|
| 26 services × ~6 hours migration each (spec, code, test, PR review) | ~150 person-hours, parallelizable to **5-7 calendar days** if 5 agents run in parallel |
| One-shot codegen tool (`tools/gen_rust_proto`) | ~4 hours |
| Per-service training cost for team to author `.proto` | ~30 min/dev, one-time |

| Saving | Estimate |
|---|---|
| ~10k LoC of hand-written DTO + api-client + service-bridge plumbing **deleted** | quantified below |
| Future DTO drift bugs **prevented** | average 1.3 bugs/week × 4 weeks debugging time saved on Q3 backlog |
| Schema becomes copy-pasteable into iOS / Swift / Kotlin without re-typing | unlocks faster iOS feature parity |

### Lines deleted at full migration (estimate)

```
clients/core/crates/types/src/*.rs                   ~3.6k LoC
clients/core/crates/api-client/src/modules/*.rs      ~2.4k LoC
clients/core/crates/wasm/src/service_*.rs            ~3.0k LoC
clients/web/src/lib/api/*Types.ts                    ~1.2k LoC
                                                    ───────────
                                                     ~10.2k LoC
```

Replaced with generated Rust modules from `.proto` (committed but not hand-edited) + per-service Connect handler + per-service wasm export.

## Decision

**Migrate the data plane to protobuf schemas served over Connect-RPC, with the schema (`.proto`) as the single source of truth across Go / Rust / TS.**

### Specifics

1. **Codec**: Connect+JSON is the default wire format (negotiated via `Content-Type: application/json`). Connect+proto-binary is opt-in per service (set `Content-Type: application/proto`) — leveraging the same handler. JSON wins because:
   - It is curl-debuggable, matches existing dev habits.
   - It is already in the wasm dep graph (`serde_json`); no new wire-format runtime cost.
   - The drift cost we want to eliminate is **field naming/typing drift**, which is fully solved by a shared `.proto` regardless of codec.
   - 2-3× decode speedup of binary buys little when payloads are <10 KB.

2. **Migration model**: One-way migration. Each service migrates from REST + hand-written DTO → Connect-RPC + `.proto`. **No long-term dual-track.** Per-service old REST handlers can stay mounted in parallel during the migration window for safety, but are removed when the new Connect handler ships green CI.

3. **Time window**: 5-7 calendar days. Five agent teams running in parallel against the 26-service list, each PR target `feat/proto-migration`. Phase 0 scaffolding already merged.

4. **Branch model**:
   - **Long-lived branch**: `feat/proto-migration` (already pushed).
   - Per-service PRs target `feat/proto-migration`, **not** `main`.
   - When all 26 services done + CI green + e2e green + manual smoke, a single PR merges `feat/proto-migration → main`.
   - **Until** that final merge, `main` keeps shipping over REST. Production rolls forward via dev → staging → prod stage promotion of the merged commit.

5. **Codegen**: **No `rules_rust_prost`** (POC §3.3). Instead, write a host-side codegen tool that wraps `protoc + protoc-gen-prost`, run it once per service, **commit the `.rs`** output. Bazel just consumes the committed `.rs` files. Rationale:
   - `rules_rust_prost` requires `rules_rust 0.70+`. Repo is on 0.68.1.
   - Adding it means ~200-LoC infra PR + 4 new `crate.spec` entries + toolchain registration.
   - The committed `.rs` artifacts make migrations atomic: a service PR includes both `.proto` and the generated DTO, no toolchain coupling.

6. **Auth**: Connect auth interceptor mirrors the existing `middleware.AuthMiddleware` (parses `Authorization: Bearer <jwt>`, populates a request context). Lands in a separate PR **before** the first non-trivial service migration (already in flight on another agent).

7. **Naming**: see `proto-naming-conventions.md` — every convention is locked, codegen template enforces it, CI gates it.

## Consequences

### Positive

- **Schema-enforced consistency**. Adding / renaming a field is one edit in `.proto`; codegen propagates the rename through Rust + Go. The TS layer reads the same Connect-emitted JSON shape (camelCase from snake_case `.proto`) without a hand-written DTO.
- **~10k LoC deletion** of hand-written DTO + plumbing across the four-layer chain.
- **Drift bugs become compile errors**, not silent zero / undefined renders.
- **iOS / Kotlin parity unlocked** — same `.proto` generates Swift via SwiftProtobuf, Kotlin via protoc-gen-kotlin, without re-typing the contract.
- **OpenAPI auto-generatable** from `.proto` if we ever need a public REST gateway (Connect-go has `vanguard-go` for REST-from-proto).
- **No new runtime dependency** on the wire path: Connect speaks gRPC, gRPC-Web, **and** Connect on the same path via content negotiation — no Envoy, no gateway.
- **Forward-compat by design**: proto3 unknown-field tolerance means a new backend field is non-breaking for older clients.

### Negative

- **Short-term engineering pressure**: 5-7 calendar days of focused migration work, plus per-developer learning curve to author `.proto` (most have seen it, few have written it).
- **`.proto` SSOT discipline**: any out-of-band hand-edit to generated `.rs` will silently break the round-trip. CI must `bazel run //tools/gen_rust_proto:check` against committed output.
- **Wrapper-envelope discipline**: list responses MUST follow `{items, total, limit, offset}` (locked in conventions.md). Anyone deviating reintroduces 986a38ca6.
- **wasm bundle growth**: prost adds ~114 KB baseline (one-time, already paid in Phase 0); each service adds ~5 KB DTO. Full 26-service total: ~+130 KB ≈ 22.5 MB wasm. Well under the 30% threshold (POC §5) but **must be monitored** — `clients/web/scripts/check-no-wasm-in-marketing.sh` already gates marketing-page leak; we add a wasm-size budget check.
- **`google.protobuf.Timestamp` is deferred** — every service uses `string` ISO-8601 for timestamps. Migrating to `Timestamp` later requires another sweep but is non-breaking on JSON wire (`protojson` already emits ISO-8601 for `Timestamp` fields).
- **Auth interceptor must ship before service migrations land** — without it the first migrated service is unauthenticated. Hard sequencing constraint.

## Alternatives considered

### OpenAPI / Swagger — REJECTED

- **Why considered**: industry default, decent tooling.
- **Why rejected**: still requires hand-written Rust + TS bindings (openapi-generator output is verbose and serde-incompatible without forks); does not solve the codec drift; does not provide the gRPC / iOS / Kotlin codegen we'll want later; does not have Connect-go's interceptor composition story.

### Continue hand-written DTOs + tooling discipline — REJECTED

- **Why considered**: zero migration cost.
- **Why rejected**: failed empirically. 986a38ca6 was a 26-service audit and it shipped **incomplete** (issue #341 had to reopen). Convention without enforcement is the status quo and the status quo bleeds bugs every other week.

### `rules_rust_prost` for in-Bazel Rust proto codegen — REJECTED FOR NOW

- **Why considered**: pure Bazel, no `protoc` shell-out.
- **Why rejected**: BCR ships `rules_rust_prost 0.70+`, repo is on `rules_rust 0.68.1`. Bumping is ~200-LoC infra PR; the out-of-band codegen + commit approach has zero coupling to the rules_rust version and is equivalent in correctness. Revisit when `rules_rust` next bumps.

### Connect for control plane (Runner ↔ Backend) too — DEFERRED

- **Why considered**: unify all the data flows on Connect.
- **Why rejected for now**: Runner control plane already uses gRPC + mTLS (`proto/runner/v1/runner.proto`), works fine, has bidirectional streaming requirements that Connect-unary cannot serve. Out of scope for this migration.

## References

### PRs / commits

- Phase 0 scaffolding: `9e86141e3` (2026-05-12)
- DTO drift bug PRs: #329 (2026-05-06), #334 (2026-05-07), #340 (2026-05-07), #341, #342, #343, #345, #349, #368 (2026-05-12)
- Sweep audit: `986a38ca6` (2026-05-09, "preserve wrapper envelope fields across wasm relay")
- Issue: #341 (user re-opened after first fix)

### Sibling documents

- `proto-naming-conventions.md` — locked conventions, CI gates
- `proto-watch-list.md` — 7 known migration hazards
- `proto-migration-runbook.md` — step-by-step for specialist agents

### POC validation

- Worktree: `.claude/worktrees/agent-a3d7d9878ce9caca4/` (not pushed)
- Report: `poc-report.md`
- Verified: both JSON and proto-binary lanes, Bazel build, wasm32 round-trip, +114 KB bundle delta

## Locked decisions (quick reference for migration PRs)

| Item | Locked value |
|---|---|
| Codec default | Connect+JSON |
| Proto package | `proto.<domain>.v1` (e.g., `proto.extension.v1`) |
| Service URL | `/<package>.<Service>/<Method>` |
| Field naming on wire | camelCase (Connect protojson auto from snake_case `.proto`) |
| Rust struct rename | `#[serde(rename_all = "camelCase")]` on every response message |
| List response shape | `{items: [...], total: int64, limit: int32, offset: int32}` — uniform across 26 services |
| Single-entity create/update response | message **is** the entity (no `{entity: ...}` wrapper) |
| Timestamp type | `string` ISO-8601 (no `google.protobuf.Timestamp`) |
| Optional scalars | proto3 `optional` keyword (not default scalars) where zero is meaningful |
| `oneof` JSON shape | `{kind: {variantName: ...}}` tagged shape, untagged Rust enum + custom deserialize |
| Error model | Connect standard (`connect.NewError(connect.CodeXxx, err)`) — **forbidden**: `{error: "..."}` |
| Required header | `Connect-Protocol-Version: 1` (wasm helper injects) |
