// Auto-generated from agentsmesh_wasm.d.ts
// Service interfaces for cross-platform abstraction (Web WASM / Tauri Native)

export interface IAcpSessionManager {
  add_content_chunk(pod_key: string, text: string, role: string): void;
  add_log(pod_key: string, level: string, message: string): void;
  add_permission_request(pod_key: string, request_json: string): void;
  add_thinking(pod_key: string, text: string): void;
  clear_session(pod_key: string): void;
  get_session_json(pod_key: string): any;
  mark_last_message_complete(pod_key: string): void;
  remove_permission_request(pod_key: string, request_id: string): void;
  set_tool_call_result(pod_key: string, tool_call_id: string, success: boolean, result_text?: string | null, error_message?: string | null): void;
  update_plan(pod_key: string, steps_json: string): void;
  update_session_state(pod_key: string, state_str: string): void;
  update_tool_call(pod_key: string, tool_call_json: string): void;
}

export interface IAgentService {
  create_provider(json: string): Promise<string>;
  delete_provider(id: bigint): Promise<void>;
  delete_user_config(agent_slug: string): Promise<void>;
  get_agentpod_settings(): Promise<string>;
  get_config_schema(agent_slug: string): Promise<string>;
  get_user_config(agent_slug: string): Promise<string>;
  list_agents(): Promise<string>;
  list_providers(): Promise<string>;
  list_user_configs(): Promise<string>;
  set_default_provider(id: bigint): Promise<void>;
  set_user_config(agent_slug: string, json: string): Promise<string>;
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
  create_message_service(): IMessageService;
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
  apply_session?(session_json: string): Promise<void> | void;
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
  set_current_org?(org_json: string): Promise<void> | void;
  set_organizations?(orgs_json: string): Promise<void> | void;
  switch_org(slug: string): void;
  readonly base_url: string;
}

export interface IAutopilotService {
  add_controller(json: string): void;
  add_iteration(key: string, json: string): void;
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
  remove_controller(key: string): void;
  resume_controller(key: string): Promise<void>;
  set_controllers(json: string): void;
  set_current_controller(json: string): void;
  set_iterations(key: string, json: string): void;
  stop_controller(key: string): Promise<void>;
  takeover_controller(key: string): Promise<void>;
  update_controller(key: string, json: string): void;
  update_thinking(key: string, json: string): void;
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
  accept_binding(json: string): Promise<string>;
  approve_scopes(binding_id: bigint, json: string): Promise<string>;
  check_binding(target_pod: string): Promise<string>;
  get_bound_pods(): Promise<string>;
  get_pending_bindings(): Promise<string>;
  list_bindings(status?: string | null): Promise<string>;
  reject_binding(json: string): Promise<void>;
  request_binding(json: string, pod_key?: string | null): Promise<string>;
  request_scopes(binding_id: bigint, json: string): Promise<string>;
  unbind(json: string): Promise<void>;
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
  update_channel_local(id: bigint, json: string): void;
  update_message_local(channel_id: bigint, json: string): void;
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
  install_skill_from_upload(repo_id: bigint, file_data: Uint8Array, file_name: string, scope?: string | null): Promise<string>;
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
  presign_upload(json: string): Promise<string>;
  upload_file(file_data: Uint8Array, filename: string, content_type: string): Promise<string>;
}

export interface IGitProviderState {
  add_provider(json: string): void;
  available_projects_json(): string;
  current_provider_json(): any;
  providers_json(): string;
  remove_provider(id: string): void;
  set_available_projects(json: string): void;
  set_current_provider(json: string): void;
  set_providers(json: string): void;
  update_provider(id: string, json: string): void;
}

export interface IInvitationService {
  accept(token: string): Promise<void>;
  create(json: string): Promise<string>;
  get_by_token(token: string): Promise<string>;
  list(): Promise<string>;
  list_pending(): Promise<string>;
  resend(id: bigint): Promise<void>;
  revoke(id: bigint): Promise<void>;
}

export type LocalRunnerStatus = "running" | "stopped" | "unknown" | "not_installed";

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
  add_run(json: string): void;
  append_runs(json: string): void;
  cancel_run(slug: string, run_id: bigint): Promise<void>;
  clear_runs(): void;
  create_loop(request_json: string): Promise<string>;
  current_loop_json(): any;
  delete_loop(slug: string): Promise<void>;
  disable_loop(slug: string): Promise<string>;
  enable_loop(slug: string): Promise<string>;
  fetch_loop(slug: string): Promise<string>;
  fetch_loops(status?: string | null, limit?: number | null, offset?: number | null): Promise<string>;
  fetch_runs(slug: string, status?: string | null, limit?: number | null, offset?: number | null): Promise<string>;
  get_loop_by_slug_json(slug: string): any;
  loops_json(): string;
  runs_json(): string;
  set_current_loop(json: string): void;
  set_loops(json: string): void;
  set_runs(json: string): void;
  trigger_loop(slug: string): Promise<string>;
  update_loop(slug: string, request_json: string): Promise<string>;
  update_loop_local(slug: string, json: string): void;
  update_run_status(run_id: bigint, status: string): void;
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
  fetch_topology(): Promise<string>;
  get_active_nodes_json(): string;
  get_channels_for_node_json(pod_key: string): string;
  get_edges_for_node_json(pod_key: string): string;
  get_node_json(pod_key: string): any;
  get_nodes_by_runner_json(runner_id: bigint): string;
  get_runner_info_json(runner_id: bigint): any;
  select_node(pod_key?: string | null): void;
  selected_node(): any;
  set_topology(json: string): void;
  topology_json(): any;
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
  set_topology(json: string): void;
  topology_json(): any;
}

export interface IMessageService {
  get_conversation(correlation_id: string, limit?: number | null): Promise<string>;
  get_dead_letters(limit?: number | null, offset?: number | null): Promise<string>;
  get_message(id: bigint): Promise<string>;
  get_messages(unread_only?: boolean | null, limit?: number | null, offset?: number | null): Promise<string>;
  get_sent_messages(limit?: number | null, offset?: number | null): Promise<string>;
  get_unread_count(): Promise<string>;
  mark_all_read(): Promise<string>;
  mark_read(json: string): Promise<string>;
  replay_dead_letter(entry_id: bigint): Promise<string>;
  send_message(json: string, pod_key?: string | null): Promise<string>;
}

export interface INotificationService {
  get_preferences(): Promise<string>;
  set_preference(json: string): Promise<string>;
}

export interface IOrgApiService {
  create(json: string): Promise<string>;
  delete(slug: string): Promise<void>;
  get(slug: string): Promise<string>;
  invite_member(slug: string, json: string): Promise<string>;
  list(): Promise<string>;
  list_members(slug: string): Promise<string>;
  remove_member(slug: string, user_id: bigint): Promise<void>;
  update(slug: string, json: string): Promise<string>;
  update_member_role(slug: string, user_id: bigint, json: string): Promise<string>;
}

export interface IOrgState {
  add_member(json: string): void;
  add_organization(json: string): void;
  current_org_json(): any;
  members_json(): string;
  organizations_json(): string;
  remove_member(id: string): void;
  remove_organization(id: number): void;
  set_current_org(json: string): void;
  set_members(json: string): void;
  set_organizations(json: string): void;
  update_member(user_id: number, json: string): void;
  update_organization(id: number, json: string): void;
}

export interface IPodService {
  create_pod(request_json: string): Promise<string>;
  current_pod_json(): any;
  fetch_pod(pod_key: string): Promise<string>;
  fetch_pods(status?: string | null, runner_id?: bigint | null, created_by_id?: bigint | null, limit?: bigint | null, offset?: bigint | null): Promise<string>;
  fetch_sidebar_pods(filter: string, user_id?: bigint | null): Promise<string>;
  get_pod_connection(pod_key: string): Promise<string>;
  get_pod_json(pod_key: string): any;
  load_more_pods(filter: string, user_id: bigint | null | undefined, offset: bigint): Promise<string>;
  pods_json(): string;
  remove_pod(pod_key: string): void;
  set_current_pod(pod_json: string): void;
  set_pods(pods_json: string): void;
  terminate_pod(pod_key: string): Promise<void>;
  update_agent_status(pod_key: string, agent_status: string): void;
  update_pod_alias(pod_key: string, alias: string): void;
  update_pod_alias_api(pod_key: string, alias?: string | null): Promise<void>;
  update_pod_status(pod_key: string, status: string, agent_status?: string | null, error_code?: string | null, error_message?: string | null, timestamp?: bigint | null): void;
  update_pod_title(pod_key: string, title: string, timestamp?: bigint | null): void;
  upsert_pod(pod_json: string, timestamp?: bigint | null): void;
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
  get_history(): Promise<string>;
  redeem(json: string): Promise<void>;
  validate(json: string): Promise<string>;
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
  add_repository(json: string): void;
  branches_json(): string;
  current_repo_json(): any;
  remove_repository(id: string): void;
  repositories_json(): string;
  set_branches(json: string): void;
  set_current_repo(json: string): void;
  set_repositories(json: string): void;
  update_repository(id: string, json: string): void;
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
  authorize_runner(request_json: string): Promise<string>;
  available_runners_json(): string;
  create_token(request_json: string): Promise<string>;
  current_runner_json(): any;
  delete_runner(id: bigint): Promise<void>;
  delete_token(id: bigint): Promise<void>;
  fetch_available_runners(): Promise<string>;
  fetch_runner(id: bigint): Promise<string>;
  fetch_runners(status?: string | null): Promise<string>;
  fetch_tokens(): Promise<string>;
  get_auth_status(auth_key: string): Promise<string>;
  get_runner_json(id: bigint): any;
  list_runner_logs(id: bigint): Promise<string>;
  list_runner_pods(id: bigint, status?: string | null, limit?: number | null, offset?: number | null): Promise<string>;
  query_runner_sandboxes(id: bigint, request_json: string): Promise<string>;
  remove_runner_local(id: bigint): void;
  request_log_upload(id: bigint): Promise<void>;
  runners_json(): string;
  set_available_runners(json: string): void;
  set_current_runner(json: string): void;
  set_runners(json: string): void;
  update_runner(id: bigint, request_json: string): Promise<string>;
  update_runner_local(id: number, json: string): void;
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
  create_comment(slug: string, json: string): Promise<string>;
  create_relation(slug: string, json: string): Promise<string>;
  delete_comment(slug: string, comment_id: bigint): Promise<void>;
  delete_relation(slug: string, relation_id: bigint): Promise<void>;
  link_commit(slug: string, json: string): Promise<string>;
  list_comments(slug: string, limit?: number | null, offset?: number | null): Promise<string>;
  list_commits(slug: string): Promise<string>;
  list_merge_requests(slug: string): Promise<string>;
  list_relations(slug: string): Promise<string>;
  unlink_commit(slug: string, commit_id: bigint): Promise<void>;
  update_comment(slug: string, comment_id: bigint, json: string): Promise<string>;
}

export interface ITicketService {
  add_label(json: string): void;
  add_ticket(json: string): void;
  append_column_tickets(status: string, json: string): void;
  board_columns_json(): string;
  create_label(name: string, color: string, repository_id?: bigint | null): Promise<string>;
  create_ticket(request_json: string): Promise<string>;
  current_ticket_json(): any;
  delete_label(id: number): Promise<void>;
  delete_ticket(slug: string): Promise<void>;
  fetch_board(repository_id?: bigint | null): Promise<string>;
  fetch_labels(repository_id?: bigint | null): Promise<string>;
  fetch_ticket(slug: string): Promise<string>;
  fetch_tickets(status?: string | null, limit?: number | null, offset?: number | null): Promise<string>;
  filter_tickets_json(search: string, statuses_json: string, priorities_json: string, repository_ids_json: string): string;
  get_sub_tickets(slug: string): Promise<string>;
  get_ticket_by_slug_json(slug: string): any;
  get_ticket_pods(slug: string, active_only?: boolean | null): Promise<string>;
  labels_json(): string;
  load_more_column(status: string, offset: number, limit: number): Promise<string>;
  remove_label(id: number): void;
  remove_ticket(slug: string): void;
  set_board_columns(json: string): void;
  set_current_ticket(json: string): void;
  set_labels(json: string): void;
  set_tickets(json: string): void;
  tickets_json(): string;
  update_ticket(slug: string, request_json: string): Promise<string>;
  update_ticket_local(slug: string, json: string): void;
  update_ticket_status(slug: string, status: string): Promise<string>;
  update_ticket_status_local(slug: string, status: string): void;
}

export interface ITicketState {
  add_label(label_json: string): void;
  add_ticket(ticket_json: string): void;
  append_column_tickets(status: string, tickets_json: string): void;
  board_columns_json(): string;
  current_ticket_json(): any;
  filter_tickets_json(search: string, statuses_json: string, priorities_json: string, repository_ids_json: string): string;
  get_ticket_by_slug_json(slug: string): any;
  labels_json(): string;
  remove_label(id: number): void;
  remove_ticket(slug: string): void;
  set_board_columns(columns_json: string): void;
  set_current_ticket(ticket_json: string): void;
  set_labels(labels_json: string): void;
  set_tickets(tickets_json: string): void;
  tickets_json(): string;
  update_ticket(slug: string, ticket_json: string): void;
  update_ticket_status(slug: string, status: string): void;
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

export interface IUserState {
  add_identity(json: string): void;
  identities_json(): string;
  profile_json(): any;
  remove_identity(id: string): void;
  set_profile(json: string): void;
}
