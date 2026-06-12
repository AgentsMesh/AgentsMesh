// Auto-generated from agentsmesh_wasm.d.ts
// Service interfaces for cross-platform abstraction (Web WASM / Tauri Native)

export interface IAcpSessionManager {
  add_content_chunk(pod_key: string, text: string, role: string): void;
  add_log(pod_key: string, level: string, message: string): void;
  add_thinking(pod_key: string, text: string): void;
  clear_session(pod_key: string): void;
  get_session_json(pod_key: string): any;
  mark_last_message_complete(pod_key: string): void;
  remove_permission_request(pod_key: string, request_id: string): void;
  set_tool_call_result(pod_key: string, tool_call_id: string, success: boolean, result_text?: string | null, error_message?: string | null): void;
  update_session_state(pod_key: string, state_str: string): void;
  // Proto-bytes mutators (mirror state_acp.rs).
  update_tool_call(req_bytes: Uint8Array): void;
  update_plan(req_bytes: Uint8Array): void;
  add_permission_request(req_bytes: Uint8Array): void;
  update_configuration(req_bytes: Uint8Array): void;
}

export interface IAgentService {
  create_provider(json: string): Promise<string>;
  delete_provider(id: bigint): Promise<void>;
  get_agentpod_settings(): Promise<string>;
  list_agents(): Promise<string>;
  list_providers(): Promise<string>;
  set_default_provider(id: bigint): Promise<void>;
  update_agentpod_settings(json: string): Promise<string>;
  update_provider(id: bigint, json: string): Promise<string>;
}

export interface IApiClient {
  create_agent_service(): IAgentService;
  create_apikey_service(): IApiKeyService;
  create_autopilot_service(): IAutopilotService;
  create_billing_service(): IBillingService;
  create_binding_service(): IBindingService;
  create_channel_service(): IChannelService;
  create_extension_service(): IExtensionService;
  create_file_service(): IFileService;
  create_invitation_service(): IInvitationService;
  create_loop_service(): ILoopService;
  create_mesh_service(): IMeshService;
  create_notification_service(): INotificationService;
  create_org_api_service(): IOrgApiService;
  create_pod_service(): IPodService;
  create_promocode_service(): IPromoCodeService;
  create_repository_service(): IRepositoryService;
  create_runner_service(): IRunnerService;
  create_sso_service(): ISSOService;
  create_support_ticket_service(): ISupportTicketService;
  create_ticket_relations_service(): ITicketRelationsService;
  create_ticket_service(): ITicketService;
  create_token_usage_service(): ITokenUsageService;
  create_user_api_service(): IUserApiService;
  create_user_credential_service(): IUserCredentialService;
  delete(endpoint: string): Promise<string>;
  get(endpoint: string): Promise<string>;
  org_path(path: string): string;
  patch(endpoint: string, body: string): Promise<string>;
  post(endpoint: string, body: string): Promise<string>;
  public_get(endpoint: string): Promise<string>;
  public_post(endpoint: string, body: string): Promise<string>;
  put(endpoint: string, body: string): Promise<string>;
  readonly base_url: string;
}

export interface IApiKeyService {
  create(json: string): Promise<string>;
  delete(id: bigint): Promise<void>;
  get(id: bigint): Promise<string>;
  list(): Promise<string>;
  revoke(id: bigint): Promise<void>;
  update(id: bigint, json: string): Promise<string>;
}

export interface IAuthConnectService {
  loginConnect(request: Uint8Array): Promise<Uint8Array>;
  registerConnect(request: Uint8Array): Promise<Uint8Array>;
  refreshTokenConnect(request: Uint8Array): Promise<Uint8Array>;
  verifyEmailConnect(request: Uint8Array): Promise<Uint8Array>;
  resendVerificationConnect(request: Uint8Array): Promise<Uint8Array>;
  forgotPasswordConnect(request: Uint8Array): Promise<Uint8Array>;
  resetPasswordConnect(request: Uint8Array): Promise<Uint8Array>;
  oauthRedirectConnect(request: Uint8Array): Promise<Uint8Array>;
  oauthCallbackConnect(request: Uint8Array): Promise<Uint8Array>;
  logoutConnect(request: Uint8Array): Promise<Uint8Array>;
}

export interface IAuthManager {
  // apply_session / clear_session / set_organizations may return Promise on
  // adapters that need to fan out to a remote SSOT (Electron IPC → Rust main).
  // Wasm AuthManager is sync (in-process); callers MUST `await` so both
  // shapes work — see clients/web/src/stores/auth.ts setAuth.
  apply_session?(req_bytes: Uint8Array): Promise<void> | void;
  bootstrap(): Promise<string>;
  clear_session?(): Promise<void> | void;
  fetch_organizations(): Promise<string>;
  get_current_org_json(): any;
  get_current_user_json(): any;
  get_organizations_json(): string;
  get_refresh_token(): string | undefined;
  get_token(): string | undefined;
  is_authenticated(): boolean;
  login(email: string, password: string): Promise<string>;
  logout(): Promise<void>;
  refresh_token(): Promise<string>;
  set_current_org?(req_bytes: Uint8Array): Promise<void> | void;
  set_organizations?(req_bytes: Uint8Array): Promise<void> | void;
  switch_org(slug: string): void;
  readonly base_url: string;
}

export interface IAutopilotService {
  approve_controller(key: string, request_json: string): Promise<void>;
  controllers_json(): string;
  create_controller(request_json: string): Promise<string>;
  current_controller_json(): any;
  fetch_controller(key: string): Promise<string>;
  fetch_controllers(): Promise<string>;
  fetch_iterations(key: string): Promise<string>;
  get_controller_by_pod_key_json(pod_key: string): any;
  get_iterations_json(key: string): any;
  get_thinking_history_json(key: string): any;
  get_thinking_json(key: string): any;
  handback_controller(key: string): Promise<void>;
  pause_controller(key: string): Promise<void>;
  resume_controller(key: string): Promise<void>;
  stop_controller(key: string): Promise<void>;
  takeover_controller(key: string): Promise<void>;
  // Proto-bytes mutators (mirror state_autopilot.rs).
  replace_cached_controllers(req_bytes: Uint8Array): void;
  set_current_controller_proto(req_bytes: Uint8Array): void;
  insert_controller(req_bytes: Uint8Array): void;
  patch_controller(req_bytes: Uint8Array): void;
  remove_controller_proto(req_bytes: Uint8Array): void;
  replace_cached_iterations(req_bytes: Uint8Array): void;
  append_iteration(req_bytes: Uint8Array): void;
  update_thinking_proto(req_bytes: Uint8Array): void;
}

export interface IAutopilotState {
  add_controller(json: string): void;
  add_iteration(key: string, json: string): void;
  controllers_json(): string;
  current_controller_json(): any;
  get_controller_by_pod_key_json(pod_key: string): any;
  get_iterations_json(key: string): any;
  get_thinking_history_json(key: string): any;
  get_thinking_json(key: string): any;
  remove_controller(key: string): void;
  set_controllers(json: string): void;
  set_current_controller(json: string): void;
  set_iterations(key: string, json: string): void;
  update_controller(key: string, json: string): void;
  update_thinking(key: string, json: string): void;
}

export interface IBillingService {
  cancel_subscription(): Promise<string>;
  change_cycle(json: string): Promise<string>;
  check_quota(resource: string, amount?: number | null): Promise<string>;
  create_checkout(json: string): Promise<string>;
  create_subscription(json: string): Promise<string>;
  get_checkout_status(order_no: string): Promise<string>;
  get_customer_portal(json: string): Promise<string>;
  get_deployment_info(): Promise<string>;
  get_overview(): Promise<string>;
  get_public_deployment_info(): Promise<string>;
  get_public_pricing(): Promise<string>;
  get_seat_usage(): Promise<string>;
  get_subscription(): Promise<string>;
  get_usage(usage_type?: string | null): Promise<string>;
  list_invoices(limit?: number | null, offset?: number | null): Promise<string>;
  list_plans(): Promise<string>;
  purchase_seats(json: string): Promise<string>;
  reactivate(): Promise<string>;
  request_cancel(json: string): Promise<string>;
  update_auto_renew(json: string): Promise<string>;
  update_subscription(json: string): Promise<string>;
  upgrade(json: string): Promise<string>;
}

export interface IBindingService {
  // Connect-RPC: proto.binding.v1.BindingService. Binary wire (Uint8Array
  // in, Uint8Array out). Callers encode/decode via @bufbuild/protobuf
  // — see clients/web/src/lib/api/bindingConnect.ts for the adapter.
  acceptBindingConnect(request: Uint8Array): Promise<Uint8Array>;
  approveScopesConnect(request: Uint8Array): Promise<Uint8Array>;
  checkBindingConnect(request: Uint8Array): Promise<Uint8Array>;
  getBoundPodsConnect(request: Uint8Array): Promise<Uint8Array>;
  getPendingBindingsConnect(request: Uint8Array): Promise<Uint8Array>;
  listBindingsConnect(request: Uint8Array): Promise<Uint8Array>;
  rejectBindingConnect(request: Uint8Array): Promise<Uint8Array>;
  requestBindingConnect(request: Uint8Array): Promise<Uint8Array>;
  requestScopesConnect(request: Uint8Array): Promise<Uint8Array>;
  unbindConnect(request: Uint8Array): Promise<Uint8Array>;
}

export interface IChannelService {
  add_channel_local(json: string): void;
  add_message(channel_id: bigint, json: string): void;
  archive_channel(id: bigint): Promise<void>;
  channels_json(): string;
  clear_channel_mentions(channel_id: bigint): void;
  clear_channel_unread(channel_id: bigint): void;
  create_channel(request_json: string): Promise<string>;
  current_channel_json(): any;
  delete_message(channel_id: bigint, message_id: bigint): Promise<void>;
  edit_message(channel_id: bigint, message_id: bigint, content: string): Promise<string>;
  fetch_channel(id: bigint): Promise<string>;
  fetch_channels(include_archived?: boolean | null): Promise<string>;
  fetch_messages(channel_id: bigint, limit?: number | null, before_id?: bigint | null): Promise<string>;
  fetch_unread_counts(): Promise<string>;
  filter_channels_json(query: string, include_archived: boolean): string;
  get_channel_json(id: bigint): any;
  get_channel_pods(id: bigint): Promise<string>;
  get_last_message_json(channel_id: bigint): any;
  get_mention_count(channel_id: bigint): number;
  get_messages_json(channel_id: bigint): any;
  get_unread_count(channel_id: bigint): number;
  increment_mention(channel_id: bigint): void;
  increment_unread(channel_id: bigint): void;
  join_channel(channel_id: bigint, pod_key: string): Promise<string>;
  leave_channel(channel_id: bigint, pod_key: string): Promise<string>;
  mark_read(channel_id: bigint, message_id: bigint): Promise<void>;
  mention_counts_json(): string;
  mute_channel(channel_id: bigint, muted: boolean): Promise<void>;
  on_new_message(json: string): boolean;
  prepend_messages(channel_id: bigint, json: string, has_more: boolean): void;
  remove_channel_local(id: bigint): void;
  remove_message_local(channel_id: bigint, message_id: bigint): void;
  select_channel(id?: bigint | null): any;
  send_message(channel_id: bigint, request_json: string): Promise<string>;
  set_channels(json: string): void;
  set_current_channel(id?: bigint | null): void;
  set_current_user(user_json: string): void;
  set_current_user_id(user_id?: bigint | null): void;
  set_last_message(channel_id: bigint, json: string): void;
  set_mention_counts(json: string): void;
  set_messages(channel_id: bigint, json: string, has_more: boolean): void;
  set_unread_counts(json: string): void;
  sorted_channel_ids_json(mode: string, include_archived: boolean): string;
  total_mention_count(): number;
  total_unread_count(): number;
  unarchive_channel(id: bigint): Promise<void>;
  unread_counts_json(): string;
  update_channel(id: bigint, request_json: string): Promise<string>;
  update_message_local(channel_id: bigint, json: string): void;
  // Proto-bytes mutators — bytes encode @bufbuild/protobuf-generated messages
  // from proto/channel_state/v1/mutations.proto.
  replace_channel_pods(req_bytes: Uint8Array): Promise<void>;
  replace_channel_members(req_bytes: Uint8Array): Promise<void>;
  remove_channel_member(req_bytes: Uint8Array): Promise<void>;
}

export interface IChannelState {
  add_channel(json: string): void;
  add_message(channel_id: bigint, message_json: string): void;
  channels_json(): string;
  clear_channel_mentions(channel_id: bigint): void;
  clear_channel_unread(channel_id: bigint): void;
  current_channel_json(): any;
  filter_channels_json(query: string, include_archived: boolean): string;
  get_channel_json(id: bigint): any;
  get_last_message_json(channel_id: bigint): any;
  get_mention_count(channel_id: bigint): number;
  get_messages_json(channel_id: bigint): any;
  get_unread_count(channel_id: bigint): number;
  increment_mention(channel_id: bigint): void;
  increment_unread(channel_id: bigint): void;
  mention_counts_json(): string;
  on_new_message(message_json: string): boolean;
  prepend_messages(channel_id: bigint, messages_json: string, has_more: boolean): void;
  remove_channel(id: bigint): void;
  remove_message(channel_id: bigint, message_id: bigint): void;
  select_channel(id?: bigint | null): any;
  set_channels(json: string): void;
  set_current_channel(id?: bigint | null): void;
  set_current_user(user_json: string): void;
  set_current_user_id(user_id?: bigint | null): void;
  set_last_message(channel_id: bigint, preview_json: string): void;
  set_mention_counts(json: string): void;
  set_messages(channel_id: bigint, messages_json: string, has_more: boolean): void;
  set_unread_counts(json: string): void;
  sorted_channel_ids_json(mode: string, include_archived: boolean): string;
  total_mention_count(): number;
  total_unread_count(): number;
  unread_counts_json(): string;
  update_channel(id: bigint, json: string): void;
  update_message(channel_id: bigint, message_json: string): void;
}

export interface IEventsManager {
  connect(): Promise<void>;
  disconnect(): Promise<void>;
  get_connection_state(): Promise<string>;
  on_connection_state_change(callback: Function): Promise<number>;
  subscribe(event_type: string, callback: Function): Promise<number>;
  subscribe_all(callback: Function): Promise<number>;
  unsubscribe(id: number): Promise<void>;
}

export interface IExtensionService {
  create_skill_registry(json: string): Promise<string>;
  delete_skill_registry(id: bigint): Promise<void>;
  install_custom_mcp_server(repo_id: bigint, json: string): Promise<string>;
  install_mcp_from_market(repo_id: bigint, json: string): Promise<string>;
  install_skill_from_github(repo_id: bigint, json: string): Promise<string>;
  install_skill_from_market(repo_id: bigint, json: string): Promise<string>;
  presignSkillUploadConnect(request: Uint8Array): Promise<Uint8Array>;
  installSkillFromUploadedFileConnect(request: Uint8Array): Promise<Uint8Array>;
  list_market_mcp_servers(query?: string | null, limit?: number | null, offset?: number | null): Promise<string>;
  list_market_skills(query?: string | null, category?: string | null): Promise<string>;
  list_repo_mcp_servers(repo_id: bigint, scope?: string | null): Promise<string>;
  list_repo_skills(repo_id: bigint, scope?: string | null): Promise<string>;
  list_skill_registries(): Promise<string>;
  list_skill_registry_overrides(): Promise<string>;
  sync_skill_registry(id: bigint): Promise<void>;
  toggle_skill_registry(id: bigint, json: string): Promise<string>;
  uninstall_mcp_server(repo_id: bigint, install_id: bigint): Promise<void>;
  uninstall_skill(repo_id: bigint, install_id: bigint): Promise<void>;
  update_mcp_server(repo_id: bigint, install_id: bigint, json: string): Promise<string>;
  update_skill(repo_id: bigint, install_id: bigint, json: string): Promise<string>;
}

export interface IFileService {
  upload_file(file_data: Uint8Array, filename: string, content_type: string): Promise<string>;
}

export interface IInvitationService {
  listInvitationsConnect(request: Uint8Array): Promise<Uint8Array>;
  createInvitationConnect(request: Uint8Array): Promise<Uint8Array>;
  revokeInvitationConnect(request: Uint8Array): Promise<Uint8Array>;
  resendInvitationConnect(request: Uint8Array): Promise<Uint8Array>;
  acceptInvitationConnect(request: Uint8Array): Promise<Uint8Array>;
  listPendingInvitationsConnect(request: Uint8Array): Promise<Uint8Array>;
  getInvitationByTokenConnect(request: Uint8Array): Promise<Uint8Array>;
}

// Alias for the same surface — desktop adapter file uses this name to mirror
// the auth_connect.ts / *_connect.ts naming convention.
export type IInvitationConnectService = IInvitationService;

export type LocalRunnerStatus = "running" | "stopped" | "unknown" | "not_installed" | "stale";

export interface ILocalRunnerService {
  binary_path(): Promise<string>;
  host_target(): Promise<string | null>;
  fallback_version(): Promise<string>;
  is_installed(): Promise<boolean>;
  installed_version(): Promise<string | null>;
  install_binary(release_url: string, expected_sha256?: string | null): Promise<void>;
  is_registered(): Promise<boolean>;
  local_node_id(): Promise<string | null>;
  register(token: string): Promise<void>;
  service_install(): Promise<void>;
  service_uninstall(): Promise<void>;
  service_start(): Promise<void>;
  service_stop(): Promise<void>;
  service_status(): Promise<LocalRunnerStatus>;
}

export interface ILoopService {
  current_loop_json(): any;
  get_loop_by_slug_json(slug: string): any;
  loops_json(): string;
  runs_json(): string;
  // Proto-bytes mutators (mirror WasmLoopService).
  replace_cached_loops(req_bytes: Uint8Array): void;
  set_current_loop(req_bytes: Uint8Array): void;
  clear_current_loop(req_bytes: Uint8Array): void;
  patch_loop_from_action(req_bytes: Uint8Array): void;
  insert_loop_run(req_bytes: Uint8Array): void;
  replace_cached_runs(req_bytes: Uint8Array): void;
  append_cached_runs(req_bytes: Uint8Array): void;
  patch_loop_run_status(req_bytes: Uint8Array): void;
  clear_loop_runs(req_bytes: Uint8Array): void;
}

export interface ILoopState {
  add_run(run_json: string): void;
  append_runs(json: string): void;
  clear_runs(): void;
  current_loop_json(): any;
  get_loop_by_slug_json(slug: string): any;
  loops_json(): string;
  runs_json(): string;
  set_current_loop(json: string): void;
  set_loops(json: string): void;
  set_runs(json: string): void;
  update_loop(slug: string, json: string): void;
  update_run_status(run_id: bigint, status: string): void;
}

export interface IMeshService {
  clear_topology(): void;
  fetch_topology(): Promise<Uint8Array>;
  get_active_nodes_json(): string;
  get_channels_for_node_json(pod_key: string): string;
  get_edges_for_node_json(pod_key: string): string;
  get_node_json(pod_key: string): any;
  get_nodes_by_runner_json(runner_id: bigint): string;
  get_runner_info_json(runner_id: bigint): any;
  select_node(pod_key?: string | null): void;
  selected_node(): any;
  topology_json(): any;
  // Proto-bytes mutator (mirror state_mesh.rs). Carries an opaque
  // serialised proto.mesh.v1.MeshTopology JSON blob — UI owns the
  // schema, Rust state stores it verbatim.
  replace_topology(req_bytes: Uint8Array): void;
  // Connect-RPC: proto.mesh.v1.MeshService. Binary wire (Uint8Array in,
  // Uint8Array out). Callers encode/decode via @bufbuild/protobuf — see
  // clients/web/src/lib/api/meshConnect.ts for the adapter.
  batchGetTicketPodsConnect(request: Uint8Array): Promise<Uint8Array>;
  createPodForTicketConnect(request: Uint8Array): Promise<Uint8Array>;
  getMeshTopologyConnect(request: Uint8Array): Promise<Uint8Array>;
  getTicketPodsConnect(request: Uint8Array): Promise<Uint8Array>;
}

export interface IMeshState {
  clear_topology(): void;
  get_active_nodes_json(): string;
  get_channels_for_node_json(pod_key: string): string;
  get_edges_for_node_json(pod_key: string): string;
  get_node_json(pod_key: string): any;
  get_nodes_by_runner_json(runner_id: bigint): string;
  get_runner_info_json(runner_id: bigint): any;
  select_node(pod_key?: string | null): void;
  selected_node(): any;
  // Proto-bytes mutator (mirror state_mesh.rs).
  replace_topology(req_bytes: Uint8Array): void;
  topology_json(): any;
}

export interface INotificationService {
  get_preferences(): Promise<string>;
  set_preference(json: string): Promise<string>;
}

export interface IOrgApiService {
  create(json: string): Promise<string>;
  create_personal(): Promise<string>;
  delete(slug: string): Promise<void>;
  get(slug: string): Promise<string>;
  invite_member(slug: string, json: string): Promise<string>;
  list(): Promise<string>;
  list_members(slug: string): Promise<string>;
  remove_member(slug: string, user_id: bigint): Promise<void>;
  update(slug: string, json: string): Promise<string>;
  update_member_role(slug: string, user_id: bigint, json: string): Promise<string>;
}

export interface IPodService {
  current_pod_json(): any;
  get_pod_json(pod_key: string): any;
  pods_json(): string;
  // Proto-bytes mutators (mirror WasmPodState).
  replace_cached_pods(req_bytes: Uint8Array): void;
  append_cached_pods(req_bytes: Uint8Array): void;
  insert_created_pod(req_bytes: Uint8Array): void;
  mark_pod_terminated(req_bytes: Uint8Array): void;
  patch_pod_perpetual(req_bytes: Uint8Array): void;
  apply_pod_status_event(req_bytes: Uint8Array): void;
  apply_pod_title_event(req_bytes: Uint8Array): void;
  apply_pod_alias_event(req_bytes: Uint8Array): void;
  apply_agent_status_event(req_bytes: Uint8Array): void;
}

export interface IPodState {
  current_pod_json(): any;
  get_pod_json(pod_key: string): any;
  pods_json(): string;
  remove_pod(pod_key: string): void;
  set_current_pod(pod_json: string): void;
  set_pods(pods_json: string): void;
  update_agent_status(pod_key: string, agent_status: string): void;
  update_pod_alias(pod_key: string, alias: string): void;
  update_pod_status(pod_key: string, status: string, agent_status?: string | null, error_code?: string | null, error_message?: string | null, timestamp?: bigint | null): void;
  update_pod_title(pod_key: string, title: string, timestamp?: bigint | null): void;
  upsert_pod(pod_json: string, timestamp?: bigint | null): void;
}

export interface IPromoCodeService {
  validatePromoCodeConnect(request: Uint8Array): Promise<Uint8Array>;
  redeemPromoCodeConnect(request: Uint8Array): Promise<Uint8Array>;
  getRedemptionHistoryConnect(request: Uint8Array): Promise<Uint8Array>;
}

export interface IRelayManager {
  disconnect(pod_key: string): Promise<void>;
  disconnect_all(): Promise<void>;
  force_resize(pod_key: string, cols: number, rows: number): Promise<void>;
  get_pod_size(pod_key: string): Promise<any>;
  get_status(pod_key: string): Promise<string>;
  is_runner_disconnected(pod_key: string): Promise<boolean>;
  on_acp_message(pod_key: string, callback: Function): Promise<void>;
  on_status_change(pod_key: string, callback: Function): Promise<void>;
  send(pod_key: string, data: string): Promise<void>;
  send_acp_command(pod_key: string, command: string): Promise<void>;
  send_resize(pod_key: string, cols: number, rows: number): Promise<void>;
  subscribe(pod_key: string, subscription_id: string, relay_url: string, token: string, callback: Function): Promise<void>;
  unsubscribe(pod_key: string, subscription_id: string): Promise<void>;
}

export interface IRepoState {
  branches_json(): string;
  current_repo_json(): any;
  remove_repository(id: string): void;
  repositories_json(): string;
  // Proto-bytes mutators (mirror state_repo.rs).
  replace_cached_repositories(req_bytes: Uint8Array): void;
  set_current_repo_proto(req_bytes: Uint8Array): void;
  replace_branches(req_bytes: Uint8Array): void;
  insert_repository(req_bytes: Uint8Array): void;
  patch_repository(req_bytes: Uint8Array): void;
}

export interface IRepositoryService {
  create(json: string): Promise<string>;
  delete(id: bigint): Promise<void>;
  delete_webhook(id: bigint): Promise<void>;
  get(id: bigint): Promise<string>;
  get_webhook_secret(id: bigint): Promise<string>;
  get_webhook_status(id: bigint): Promise<string>;
  list(): Promise<string>;
  list_branches(id: bigint): Promise<string>;
  list_merge_requests(id: bigint, branch?: string | null, state?: string | null): Promise<string>;
  mark_webhook_configured(id: bigint): Promise<void>;
  register_webhook(id: bigint): Promise<void>;
  sync_branches(id: bigint, json: string): Promise<string>;
  update(id: bigint, json: string): Promise<string>;
}

export interface IRunnerService {
  authorize_runner(request_bytes: Uint8Array): Promise<Uint8Array>;
  available_runners_json(): string;
  create_token(request_json: string): Promise<string>;
  current_runner_json(): any;
  delete_runner(id: bigint): Promise<void>;
  delete_token(id: bigint): Promise<void>;
  fetch_available_runners(): Promise<string>;
  fetch_runner(id: bigint): Promise<string>;
  fetch_runners(status?: string | null): Promise<string>;
  fetch_tokens(): Promise<string>;
  get_auth_status(request_bytes: Uint8Array): Promise<Uint8Array>;
  get_runner_json(id: bigint): any;
  list_runner_logs(id: bigint): Promise<string>;
  query_runner_sandboxes(id: bigint, request_json: string): Promise<string>;
  // Proto-bytes mutators (mirror state_runner.rs).
  replace_cached_runners(req_bytes: Uint8Array): void;
  replace_available_runners(req_bytes: Uint8Array): void;
  set_current_runner_proto(req_bytes: Uint8Array): void;
  patch_cached_runner(req_bytes: Uint8Array): void;
  remove_cached_runner(req_bytes: Uint8Array): void;
  request_log_upload(id: bigint): Promise<void>;
  runners_json(): string;
  update_runner_status(id: bigint, status: string): void;
  upgrade_runner(id: bigint, request_json: string): Promise<string>;
}

export interface IRunnerState {
  available_runners_json(): string;
  current_runner_json(): any;
  get_runner_json(id: bigint): any;
  remove_runner(id: bigint): void;
  runners_json(): string;
  set_available_runners(json: string): void;
  set_current_runner(json: string): void;
  set_runners(json: string): void;
  update_runner(id: number, json: string): void;
  update_runner_status(id: bigint, status: string): void;
}

export interface ISSOService {
  discoverConnect(request: Uint8Array): Promise<Uint8Array>;
  ldapAuthConnect(request: Uint8Array): Promise<Uint8Array>;
}

export interface ISupportTicketService {
  add_message(ticket_id: bigint, content: string, file_data: Uint8Array[], file_names: string[]): Promise<string>;
  create_ticket(title: string, category: string, content: string, priority: string | null | undefined, file_data: Uint8Array[], file_names: string[]): Promise<string>;
  listSupportTicketsConnect(request: Uint8Array): Promise<Uint8Array>;
  getSupportTicketConnect(request: Uint8Array): Promise<Uint8Array>;
  getAttachmentUrlConnect(request: Uint8Array): Promise<Uint8Array>;
}

export interface ITicketRelationsService {
  // Connect-RPC binary wire — each method takes prost-encoded request bytes
  // and returns prost-encoded response bytes. Encoders / decoders live in
  // clients/web/src/lib/api/ticketRelations.ts.
  list_relations_connect(request: Uint8Array): Promise<Uint8Array>;
  create_relation_connect(request: Uint8Array): Promise<Uint8Array>;
  delete_relation_connect(request: Uint8Array): Promise<Uint8Array>;
  list_commits_connect(request: Uint8Array): Promise<Uint8Array>;
  link_commit_connect(request: Uint8Array): Promise<Uint8Array>;
  unlink_commit_connect(request: Uint8Array): Promise<Uint8Array>;
  list_merge_requests_connect(request: Uint8Array): Promise<Uint8Array>;
  list_comments_connect(request: Uint8Array): Promise<Uint8Array>;
  create_comment_connect(request: Uint8Array): Promise<Uint8Array>;
  update_comment_connect(request: Uint8Array): Promise<Uint8Array>;
  delete_comment_connect(request: Uint8Array): Promise<Uint8Array>;
}

export interface ITicketService {
  // REST-only (proto.ticket.v1 doesn't own ticket→pod lookup — MeshService does).
  get_ticket_pods(slug: string, active_only?: boolean | null): Promise<string>;
  ticket_pods_json(slug: string): string;
  // Connect-RPC binary wire — each method takes prost-encoded request bytes
  // and returns prost-encoded response bytes. Encoders / decoders live in
  // clients/web/src/lib/api/ticketConnect.ts.
  list_tickets_connect(request: Uint8Array): Promise<Uint8Array>;
  get_ticket_connect(request: Uint8Array): Promise<Uint8Array>;
  create_ticket_connect(request: Uint8Array): Promise<Uint8Array>;
  update_ticket_connect(request: Uint8Array): Promise<Uint8Array>;
  delete_ticket_connect(request: Uint8Array): Promise<Uint8Array>;
  update_ticket_status_connect(request: Uint8Array): Promise<Uint8Array>;
  get_active_tickets_connect(request: Uint8Array): Promise<Uint8Array>;
  get_board_connect(request: Uint8Array): Promise<Uint8Array>;
  get_sub_tickets_connect(request: Uint8Array): Promise<Uint8Array>;
  add_assignee_connect(request: Uint8Array): Promise<Uint8Array>;
  remove_assignee_connect(request: Uint8Array): Promise<Uint8Array>;
  list_labels_connect(request: Uint8Array): Promise<Uint8Array>;
  create_label_connect(request: Uint8Array): Promise<Uint8Array>;
  update_label_connect(request: Uint8Array): Promise<Uint8Array>;
  delete_label_connect(request: Uint8Array): Promise<Uint8Array>;
  add_label_connect(request: Uint8Array): Promise<Uint8Array>;
  remove_label_connect(request: Uint8Array): Promise<Uint8Array>;
}

export interface ITicketState {
  // Read accessors — JSON for ergonomic React consumers (parsed once on read).
  board_columns_json(): string;
  current_ticket_json(): any;
  labels_json(): string;
  tickets_json(): string;
  // Proto bytes mutators — each takes prost-encoded Uint8Array; the schema
  // lives in proto/ticket_state/v1/ticket_state.proto. Mirrors the pod_state
  // bridge in shape (apply_*_event for realtime, replace_cached_* for fetch
  // results, insert_created_* for fresh entities, patch_cached_* for local
  // mutation results, set_current_ticket / append_board_column_tickets /
  // remove_cached_label for the rest).
  apply_ticket_status_event(req: Uint8Array): void;
  apply_ticket_deleted_event(req: Uint8Array): void;
  replace_cached_tickets(req: Uint8Array): void;
  insert_created_ticket(req: Uint8Array): void;
  patch_cached_ticket(req: Uint8Array): void;
  replace_board_columns(req: Uint8Array): void;
  append_board_column_tickets(req: Uint8Array): void;
  set_current_ticket(req: Uint8Array): void;
  replace_cached_labels(req: Uint8Array): void;
  insert_created_label(req: Uint8Array): void;
  remove_cached_label(req: Uint8Array): void;
  filter_tickets(req: Uint8Array): Uint8Array;
}

export interface ITokenUsageService {
  get_dashboard(start_time?: string | null, end_time?: string | null, agent_slug?: string | null, user_id?: bigint | null, model?: string | null, granularity?: string | null): Promise<string>;
}

export interface IUserApiService {
  getMeConnect(request: Uint8Array): Promise<Uint8Array>;
  updateMeConnect(request: Uint8Array): Promise<Uint8Array>;
  changePasswordConnect(request: Uint8Array): Promise<Uint8Array>;
  listIdentitiesConnect(request: Uint8Array): Promise<Uint8Array>;
  deleteIdentityConnect(request: Uint8Array): Promise<Uint8Array>;
  searchUsersConnect(request: Uint8Array): Promise<Uint8Array>;
}

export interface IUserCredentialService {
  clear_default_git_credential(): Promise<void>;
  create_agent_credential(agent_slug: string, json: string): Promise<string>;
  create_git_credential(json: string): Promise<string>;
  create_repo_provider(json: string): Promise<string>;
  delete_agent_credential(id: bigint): Promise<void>;
  delete_git_credential(id: bigint): Promise<void>;
  delete_repo_provider(id: bigint): Promise<void>;
  get_agent_credential(id: bigint): Promise<string>;
  get_default_git_credential(): Promise<string>;
  get_git_credential(id: bigint): Promise<string>;
  get_repo_provider(id: bigint): Promise<string>;
  list_agent_credentials(): Promise<string>;
  list_agent_credentials_for_agent(agent_slug: string): Promise<string>;
  list_git_credentials(): Promise<string>;
  list_provider_repositories(id: bigint, page?: number | null, per_page?: number | null, search?: string | null): Promise<string>;
  list_repo_providers(): Promise<string>;
  set_default_agent_credential(id: bigint): Promise<void>;
  set_default_git_credential(json: string): Promise<void>;
  set_default_repo_provider(id: bigint): Promise<void>;
  test_repo_provider(id: bigint): Promise<void>;
  update_agent_credential(id: bigint, json: string): Promise<string>;
  update_git_credential(id: bigint, json: string): Promise<string>;
  update_repo_provider(id: bigint, json: string): Promise<string>;
}

export {
  SERVICE_ERROR_KINDS,
  SERVICE_ERROR_KIND_SET,
  type ServiceErrorKind,
} from "./service-error-kinds";
