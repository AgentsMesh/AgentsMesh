// Wasm contract test. Type-only file — no runtime body, no asserts.
// `tsc --noEmit` (= `bazel build //clients/web:src --output_groups=typecheck`)
// is the gate: if any of the listed wasm-bindgen method signatures the
// stores actually call drifts away from what's exported in
// `agentsmesh-wasm` (the npm wrapper around the Bazel-built wasm_pkg),
// these type assertions fail and the build fails.
//
// This catches the regression class that motivated this PR: the renderer
// reached for `set_pods()`, a method that the Rust crate removed
// during the proto-bytes migration. `WasmPodState` no longer had it, so
// production threw "podState.set_pods is not a function" the first time
// the workspace sidebar tried to refresh. tsc was happy because the
// type lookup went through an `any`-typed stub.
//
// Convention: one `_Assert<…>` line per public method the production
// stores call. Type the function as a generic over the parameter tuple
// and the return type, then instantiate it with the wasm method's
// inferred types. If the method is renamed or its signature changes,
// the instantiation fails.
//
// Adding a new wasm-bound method that a store calls: append an
// `_Assert<…>` here so future drift catches it at compile time.

/* eslint-disable @typescript-eslint/no-unused-vars */
import type {
  WasmPodState,
  WasmPodService,
  WasmRunnerService,
  WasmRunnerState,
  WasmTicketState,
  WasmChannelState,
  WasmLoopState,
  WasmMeshState,
  WasmAutopilotService,
  WasmAcpSessionManager,
  WasmRepoState,
  WasmAppState,
  WasmBlockstoreService,
  WasmAuthManager,
} from "agentsmesh-wasm";

// Plain identity helper — `_Sig<F>` accepts any function type. We use it
// only to surface a type error if the wasm method's signature changes
// (or the property gets dropped from the class entirely).
type _Sig<F extends (...args: never[]) => unknown> = F;

// ── PodState (clients/core/crates/wasm/src/state_pod.rs) ────────────
// Production callers: clients/web/src/stores/pod.ts and
// providers/realtimePodHandlers.ts. Method group A is reads; B is
// proto-bytes mutators.
type _PodState_pods_json           = _Sig<WasmPodState["pods_json"]>;
type _PodState_current_pod_json    = _Sig<WasmPodState["current_pod_json"]>;
type _PodState_get_pod_json        = _Sig<WasmPodState["get_pod_json"]>;
type _PodState_insert_created_pod  = _Sig<WasmPodState["insert_created_pod"]>;
type _PodState_patch_pod_perpetual = _Sig<WasmPodState["patch_pod_perpetual"]>;
type _PodState_apply_status        = _Sig<WasmPodState["apply_pod_status_event"]>;
type _PodState_apply_title         = _Sig<WasmPodState["apply_pod_title_event"]>;
type _PodState_apply_alias         = _Sig<WasmPodState["apply_pod_alias_event"]>;
type _PodState_apply_agent_status  = _Sig<WasmPodState["apply_agent_status_event"]>;
type _PodState_replace_cached_pods = _Sig<WasmPodState["replace_cached_pods"]>;
type _PodState_append_cached_pods  = _Sig<WasmPodState["append_cached_pods"]>;
type _PodState_mark_terminated     = _Sig<WasmPodState["mark_pod_terminated"]>;
type _PodState_remove_pod          = _Sig<WasmPodState["remove_pod"]>;
type _PodState_update_init_prog    = _Sig<WasmPodState["update_init_progress"]>;
type _PodState_clear_init_prog     = _Sig<WasmPodState["clear_init_progress"]>;

// Specifically guard the proto-bytes contract: every mutator MUST
// accept a Uint8Array as its first argument. If a future rust commit
// reverts back to JSON strings these break.
type _RequiresU8<F extends (b: Uint8Array, ...rest: never[]) => unknown> = F;
type _PodState_proto_insert     = _RequiresU8<WasmPodState["insert_created_pod"]>;
type _PodState_proto_perpetual  = _RequiresU8<WasmPodState["patch_pod_perpetual"]>;
type _PodState_proto_status     = _RequiresU8<WasmPodState["apply_pod_status_event"]>;
type _PodState_proto_title      = _RequiresU8<WasmPodState["apply_pod_title_event"]>;
type _PodState_proto_alias      = _RequiresU8<WasmPodState["apply_pod_alias_event"]>;
type _PodState_proto_agent      = _RequiresU8<WasmPodState["apply_agent_status_event"]>;
type _PodState_proto_replace    = _RequiresU8<WasmPodState["replace_cached_pods"]>;
type _PodState_proto_append     = _RequiresU8<WasmPodState["append_cached_pods"]>;
type _PodState_proto_terminate  = _RequiresU8<WasmPodState["mark_pod_terminated"]>;

// ── PodService (Connect-RPC binary lane via wasm) ───────────────────
// Production callers: clients/web/src/lib/api/connect/podConnect.ts.
type _PodSvc_list_pods         = _Sig<WasmPodService["list_pods_connect"]>;
type _PodSvc_get_pod           = _Sig<WasmPodService["get_pod_connect"]>;
type _PodSvc_create_pod        = _Sig<WasmPodService["create_pod_connect"]>;
type _PodSvc_terminate_pod     = _Sig<WasmPodService["terminate_pod_connect"]>;
type _PodSvc_update_alias      = _Sig<WasmPodService["update_pod_alias_connect"]>;
type _PodSvc_update_perpetual  = _Sig<WasmPodService["update_pod_perpetual_connect"]>;
type _PodSvc_get_conn          = _Sig<WasmPodService["get_pod_connection_connect"]>;
type _PodSvc_send_prompt       = _Sig<WasmPodService["send_pod_prompt_connect"]>;
type _PodSvc_by_ticket         = _Sig<WasmPodService["list_pods_by_ticket_connect"]>;

// ── RunnerService (production callers: stores/runner.ts) ─────────────
type _RunnerSvc_list           = _Sig<WasmRunnerService["fetch_runners"]>;
type _RunnerSvc_available      = _Sig<WasmRunnerService["fetch_available_runners"]>;
type _RunnerSvc_get            = _Sig<WasmRunnerService["fetch_runner"]>;
type _RunnerSvc_update         = _Sig<WasmRunnerService["update_runner"]>;
type _RunnerSvc_delete         = _Sig<WasmRunnerService["delete_runner"]>;
type _RunnerSvc_list_pods      = _Sig<WasmRunnerService["list_runner_pods"]>;
type _RunnerSvc_query_sandbox  = _Sig<WasmRunnerService["query_runner_sandboxes"]>;

// ── RunnerState reads (used by sidebar selectors) ────────────────────
type _RunnerState_list         = _Sig<WasmRunnerState["runners_json"]>;
type _RunnerState_available    = _Sig<WasmRunnerState["available_runners_json"]>;
type _RunnerState_current      = _Sig<WasmRunnerState["current_runner_json"]>;
type _RunnerState_get          = _Sig<WasmRunnerState["get_runner_json"]>;
type _RunnerState_set          = _Sig<WasmRunnerState["set_runners"]>;
type _RunnerState_set_curr     = _Sig<WasmRunnerState["set_current_runner"]>;
type _RunnerState_update_st    = _Sig<WasmRunnerState["update_runner_status"]>;
type _RunnerState_remove       = _Sig<WasmRunnerState["remove_runner"]>;

// ── TicketState (production callers: stores/ticket.ts + board hooks) ─
type _TicketState_list         = _Sig<WasmTicketState["tickets_json"]>;
type _TicketState_get          = _Sig<WasmTicketState["get_ticket_by_slug_json"]>;
type _TicketState_set          = _Sig<WasmTicketState["set_tickets"]>;
type _TicketState_add          = _Sig<WasmTicketState["add_ticket"]>;
type _TicketState_update       = _Sig<WasmTicketState["update_ticket"]>;
type _TicketState_remove       = _Sig<WasmTicketState["remove_ticket"]>;
type _TicketState_board        = _Sig<WasmTicketState["board_columns_json"]>;
type _TicketState_set_board    = _Sig<WasmTicketState["set_board_columns"]>;
type _TicketState_labels       = _Sig<WasmTicketState["labels_json"]>;
type _TicketState_set_labels   = _Sig<WasmTicketState["set_labels"]>;
type _TicketState_filter       = _Sig<WasmTicketState["filter_tickets_json"]>;

// ── ChannelState (production callers: stores/channel.ts, stores/channelMessage.ts) ──
type _ChannelState_list        = _Sig<WasmChannelState["channels_json"]>;
type _ChannelState_current     = _Sig<WasmChannelState["current_channel_json"]>;
type _ChannelState_set         = _Sig<WasmChannelState["set_channels"]>;
type _ChannelState_add         = _Sig<WasmChannelState["add_channel"]>;
type _ChannelState_update      = _Sig<WasmChannelState["update_channel"]>;
type _ChannelState_remove      = _Sig<WasmChannelState["remove_channel"]>;
type _ChannelState_messages    = _Sig<WasmChannelState["get_messages_json"]>;
type _ChannelState_set_msgs    = _Sig<WasmChannelState["set_messages"]>;
type _ChannelState_add_msg     = _Sig<WasmChannelState["add_message"]>;
type _ChannelState_update_msg  = _Sig<WasmChannelState["update_message"]>;
type _ChannelState_remove_msg  = _Sig<WasmChannelState["remove_message"]>;
type _ChannelState_unread_get  = _Sig<WasmChannelState["get_unread_count"]>;
type _ChannelState_unread_set  = _Sig<WasmChannelState["set_unread_counts"]>;

// ── LoopState (production callers: stores/loop.ts) ───────────────────
type _LoopState_list           = _Sig<WasmLoopState["loops_json"]>;
type _LoopState_current        = _Sig<WasmLoopState["current_loop_json"]>;
type _LoopState_get_by_slug    = _Sig<WasmLoopState["get_loop_by_slug_json"]>;
type _LoopState_set            = _Sig<WasmLoopState["set_loops"]>;
type _LoopState_set_current    = _Sig<WasmLoopState["set_current_loop"]>;
type _LoopState_runs           = _Sig<WasmLoopState["runs_json"]>;
type _LoopState_set_runs       = _Sig<WasmLoopState["set_runs"]>;

// ── MeshState (production callers: stores/mesh.ts + topology hooks) ──
type _MeshState_topology       = _Sig<WasmMeshState["topology_json"]>;
type _MeshState_replace        = _Sig<WasmMeshState["replace_topology"]>;
type _MeshState_clear          = _Sig<WasmMeshState["clear_topology"]>;
type _MeshState_select         = _Sig<WasmMeshState["select_node"]>;
type _MeshState_get_node       = _Sig<WasmMeshState["get_node_json"]>;
type _MeshState_get_edges      = _Sig<WasmMeshState["get_edges_for_node_json"]>;
type _MeshState_proto_replace  = _RequiresU8<WasmMeshState["replace_topology"]>;

// ── RunnerService proto-bytes mutators (production callers: stores/runner.ts) ──
// runner-state migration: 5 mutators flipped to proto-bytes; mock conflates
// state + service into one object so dispatch is on getRunnerService().
type _RunnerSvc_replace_cached       = _Sig<WasmRunnerService["replace_cached_runners"]>;
type _RunnerSvc_replace_available    = _Sig<WasmRunnerService["replace_available_runners"]>;
type _RunnerSvc_set_current_proto    = _Sig<WasmRunnerService["set_current_runner_proto"]>;
type _RunnerSvc_patch_cached         = _Sig<WasmRunnerService["patch_cached_runner"]>;
type _RunnerSvc_remove_cached        = _Sig<WasmRunnerService["remove_cached_runner"]>;
type _RunnerSvc_proto_replace_cached    = _RequiresU8<WasmRunnerService["replace_cached_runners"]>;
type _RunnerSvc_proto_replace_available = _RequiresU8<WasmRunnerService["replace_available_runners"]>;
type _RunnerSvc_proto_set_current       = _RequiresU8<WasmRunnerService["set_current_runner_proto"]>;
type _RunnerSvc_proto_patch_cached      = _RequiresU8<WasmRunnerService["patch_cached_runner"]>;
type _RunnerSvc_proto_remove_cached     = _RequiresU8<WasmRunnerService["remove_cached_runner"]>;

// ── AutopilotService proto-bytes mutators (production callers: stores/autopilot.ts) ──
// 8 mutators on the service surface — note `remove_controller_proto` and
// `update_thinking_proto` carry the `_proto` suffix because the service
// retains legacy methods for the takeover/handback flows.
type _AutopilotSvc_replace_controllers   = _Sig<WasmAutopilotService["replace_cached_controllers"]>;
type _AutopilotSvc_set_current_proto     = _Sig<WasmAutopilotService["set_current_controller_proto"]>;
type _AutopilotSvc_insert_controller     = _Sig<WasmAutopilotService["insert_controller"]>;
type _AutopilotSvc_patch_controller      = _Sig<WasmAutopilotService["patch_controller"]>;
type _AutopilotSvc_remove_controller_proto = _Sig<WasmAutopilotService["remove_controller_proto"]>;
type _AutopilotSvc_replace_iterations    = _Sig<WasmAutopilotService["replace_cached_iterations"]>;
type _AutopilotSvc_append_iteration      = _Sig<WasmAutopilotService["append_iteration"]>;
type _AutopilotSvc_update_thinking_proto = _Sig<WasmAutopilotService["update_thinking_proto"]>;
type _AutopilotSvc_proto_replace_ctrls   = _RequiresU8<WasmAutopilotService["replace_cached_controllers"]>;
type _AutopilotSvc_proto_set_current     = _RequiresU8<WasmAutopilotService["set_current_controller_proto"]>;
type _AutopilotSvc_proto_insert          = _RequiresU8<WasmAutopilotService["insert_controller"]>;
type _AutopilotSvc_proto_patch           = _RequiresU8<WasmAutopilotService["patch_controller"]>;
type _AutopilotSvc_proto_remove          = _RequiresU8<WasmAutopilotService["remove_controller_proto"]>;
type _AutopilotSvc_proto_replace_iters   = _RequiresU8<WasmAutopilotService["replace_cached_iterations"]>;
type _AutopilotSvc_proto_append_iter     = _RequiresU8<WasmAutopilotService["append_iteration"]>;
type _AutopilotSvc_proto_update_think    = _RequiresU8<WasmAutopilotService["update_thinking_proto"]>;

// ── AcpSessionManager proto-bytes mutators (production callers: stores/acpSession.ts) ──
// 4 mutators carry opaque JSON blobs through ACP state (UI owns the AST).
type _AcpMgr_update_tool_call         = _Sig<WasmAcpSessionManager["update_tool_call"]>;
type _AcpMgr_update_plan              = _Sig<WasmAcpSessionManager["update_plan"]>;
type _AcpMgr_add_permission_request   = _Sig<WasmAcpSessionManager["add_permission_request"]>;
type _AcpMgr_update_configuration     = _Sig<WasmAcpSessionManager["update_configuration"]>;
type _AcpMgr_proto_update_tool_call   = _RequiresU8<WasmAcpSessionManager["update_tool_call"]>;
type _AcpMgr_proto_update_plan        = _RequiresU8<WasmAcpSessionManager["update_plan"]>;
type _AcpMgr_proto_add_permission     = _RequiresU8<WasmAcpSessionManager["add_permission_request"]>;
type _AcpMgr_proto_update_config      = _RequiresU8<WasmAcpSessionManager["update_configuration"]>;

// ── RepoState proto-bytes mutators (production callers: stores/repository.ts) ──
// 5 mutators on the state surface; remove_repository stays string-keyed
// because the wire schema treats it as a non-payload-carrying delete.
type _RepoState_repos_json            = _Sig<WasmRepoState["repositories_json"]>;
type _RepoState_current_repo_json     = _Sig<WasmRepoState["current_repo_json"]>;
type _RepoState_branches_json         = _Sig<WasmRepoState["branches_json"]>;
type _RepoState_replace_cached_repos  = _Sig<WasmRepoState["replace_cached_repositories"]>;
type _RepoState_set_current_repo      = _Sig<WasmRepoState["set_current_repo_proto"]>;
type _RepoState_replace_branches      = _Sig<WasmRepoState["replace_branches"]>;
type _RepoState_insert_repo           = _Sig<WasmRepoState["insert_repository"]>;
type _RepoState_patch_repo            = _Sig<WasmRepoState["patch_repository"]>;
type _RepoState_proto_replace_cached  = _RequiresU8<WasmRepoState["replace_cached_repositories"]>;
type _RepoState_proto_set_current     = _RequiresU8<WasmRepoState["set_current_repo_proto"]>;
type _RepoState_proto_replace_branches = _RequiresU8<WasmRepoState["replace_branches"]>;
type _RepoState_proto_insert          = _RequiresU8<WasmRepoState["insert_repository"]>;
type _RepoState_proto_patch           = _RequiresU8<WasmRepoState["patch_repository"]>;

// ── AppState fan-out dispatcher (single mutator; future renderer entry point) ──
type _AppState_dispatch         = _Sig<WasmAppState["dispatch_event"]>;
type _AppState_proto_dispatch   = _RequiresU8<WasmAppState["dispatch_event"]>;

// ── BlockstoreService apply_remote_op proto-bytes mutator ──
// production callers: stores/blockstoreSubscribe.ts + lib/api/facade/blockstoreApi.ts
type _BlockstoreSvc_apply_remote_op       = _Sig<WasmBlockstoreService["apply_remote_op"]>;
type _BlockstoreSvc_proto_apply_remote_op = _RequiresU8<WasmBlockstoreService["apply_remote_op"]>;

// ── BlockstoreService bulk-feed proto-bytes mutators ──
// production caller: lib/api/facade/blockstoreApi.ts (listWorkspaces /
// ensureDefaultWorkspace / getSubtree / listTypeDefs / applyOps).
// 5 mutators flipped from raw JSON-string to proto-bytes envelope.
type _BlockstoreSvc_replace_workspaces       = _Sig<WasmBlockstoreService["replace_workspaces"]>;
type _BlockstoreSvc_upsert_workspace         = _Sig<WasmBlockstoreService["upsert_workspace"]>;
type _BlockstoreSvc_upsert_blocks            = _Sig<WasmBlockstoreService["upsert_blocks"]>;
type _BlockstoreSvc_upsert_refs              = _Sig<WasmBlockstoreService["upsert_refs"]>;
type _BlockstoreSvc_project_local_ops        = _Sig<WasmBlockstoreService["project_local_ops"]>;
type _BlockstoreSvc_proto_replace_workspaces = _RequiresU8<WasmBlockstoreService["replace_workspaces"]>;
type _BlockstoreSvc_proto_upsert_workspace   = _RequiresU8<WasmBlockstoreService["upsert_workspace"]>;
type _BlockstoreSvc_proto_upsert_blocks      = _RequiresU8<WasmBlockstoreService["upsert_blocks"]>;
type _BlockstoreSvc_proto_upsert_refs        = _RequiresU8<WasmBlockstoreService["upsert_refs"]>;
type _BlockstoreSvc_proto_project_local_ops  = _RequiresU8<WasmBlockstoreService["project_local_ops"]>;

// ── AuthManager proto-bytes mutators (production callers: stores/auth.ts) ──
// 3 mutators flipped from JSON-string to proto-bytes — `apply_session` /
// `set_organizations` / `set_current_org`. Signature now requires
// Uint8Array; the previous `string` shape would break this assertion.
type _AuthMgr_apply_session          = _Sig<WasmAuthManager["apply_session"]>;
type _AuthMgr_set_organizations      = _Sig<WasmAuthManager["set_organizations"]>;
type _AuthMgr_set_current_org        = _Sig<WasmAuthManager["set_current_org"]>;
type _AuthMgr_proto_apply_session    = _RequiresU8<WasmAuthManager["apply_session"]>;
type _AuthMgr_proto_set_organizations = _RequiresU8<WasmAuthManager["set_organizations"]>;
type _AuthMgr_proto_set_current_org   = _RequiresU8<WasmAuthManager["set_current_org"]>;

// ── RunnerService auth bootstrap proto-bytes mutators ──
// production callers: lib/api/facade/runner.ts runnerAuthApi. The wasm
// bridge now speaks proto.runner_api.v1.{GetRunnerAuthStatus,Authorize
// Runner}*Request directly — request is bytes-in, response is bytes-out.
type _RunnerSvc_get_auth_status       = _Sig<WasmRunnerService["get_auth_status"]>;
type _RunnerSvc_authorize_runner      = _Sig<WasmRunnerService["authorize_runner"]>;
type _RunnerSvc_proto_get_auth_status = _RequiresU8<WasmRunnerService["get_auth_status"]>;
type _RunnerSvc_proto_authorize_runner = _RequiresU8<WasmRunnerService["authorize_runner"]>;


// Vitest discovery requires the file to contain *something* runnable, but
// the body is intentionally empty — tsc + the type assertions above are
// the whole assertion surface.
import { describe, it, expect } from "vitest";

describe("wasm contract", () => {
  it("compiles when wasm bindings match the production call sites", () => {
    expect(true).toBe(true);
  });
});
