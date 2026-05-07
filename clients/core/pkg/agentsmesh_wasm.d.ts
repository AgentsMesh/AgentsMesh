/* tslint:disable */
/* eslint-disable */

export class WasmAcpSessionManager {
    free(): void;
    [Symbol.dispose](): void;
    add_content_chunk(pod_key: string, text: string, role: string): void;
    add_log(pod_key: string, level: string, message: string): void;
    add_permission_request(pod_key: string, request_json: string): void;
    add_thinking(pod_key: string, text: string): void;
    clear_session(pod_key: string): void;
    get_session_json(pod_key: string): any;
    mark_last_message_complete(pod_key: string): void;
    constructor();
    remove_permission_request(pod_key: string, request_id: string): void;
    set_tool_call_result(pod_key: string, tool_call_id: string, success: boolean, result_text?: string | null, error_message?: string | null): void;
    update_plan(pod_key: string, steps_json: string): void;
    update_session_state(pod_key: string, state_str: string): void;
    update_tool_call(pod_key: string, tool_call_json: string): void;
}

export class WasmAgentService {
    private constructor();
    free(): void;
    [Symbol.dispose](): void;
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

export class WasmApiClient {
    free(): void;
    [Symbol.dispose](): void;
    clear_auth(): void;
    create_agent_service(): WasmAgentService;
    create_apikey_service(): WasmApiKeyService;
    create_auth_api_service(): WasmAuthApiService;
    create_autopilot_service(): WasmAutopilotService;
    create_billing_service(): WasmBillingService;
    create_binding_service(): WasmBindingService;
    create_blockstore_service(): WasmBlockstoreService;
    create_channel_service(): WasmChannelService;
    create_extension_service(): WasmExtensionService;
    create_file_service(): WasmFileService;
    create_grant_service(): WasmGrantService;
    create_invitation_service(): WasmInvitationService;
    create_loop_service(): WasmLoopService;
    create_mesh_service(): WasmMeshService;
    create_message_service(): WasmMessageService;
    create_notification_service(): WasmNotificationService;
    create_org_api_service(): WasmOrgApiService;
    /**
     * Create a WasmPodService that shares this client's ApiClient and auth.
     */
    create_pod_service(): WasmPodService;
    create_promocode_service(): WasmPromoCodeService;
    create_repository_service(): WasmRepositoryService;
    create_runner_service(): WasmRunnerService;
    create_sso_service(): WasmSSOService;
    create_support_ticket_service(): WasmSupportTicketService;
    create_ticket_relations_service(): WasmTicketRelationsService;
    create_ticket_service(): WasmTicketService;
    create_token_usage_service(): WasmTokenUsageService;
    create_user_api_service(): WasmUserApiService;
    create_user_credential_service(): WasmUserCredentialService;
    delete(endpoint: string): Promise<string>;
    get(endpoint: string): Promise<string>;
    get_org_slug(): string | undefined;
    get_token(): string | undefined;
    constructor(base_url: string);
    org_path(path: string): string;
    patch(endpoint: string, body: string): Promise<string>;
    post(endpoint: string, body: string): Promise<string>;
    public_get(endpoint: string): Promise<string>;
    public_post(endpoint: string, body: string): Promise<string>;
    put(endpoint: string, body: string): Promise<string>;
    set_org_slug(slug: string): void;
    set_token(token: string, refresh_token: string): void;
    readonly base_url: string;
}

export class WasmApiKeyService {
    private constructor();
    free(): void;
    [Symbol.dispose](): void;
    create(json: string): Promise<string>;
    delete(id: bigint): Promise<void>;
    get(id: bigint): Promise<string>;
    list(): Promise<string>;
    revoke(id: bigint): Promise<void>;
    update(id: bigint, json: string): Promise<string>;
}

export class WasmAppState {
    free(): void;
    [Symbol.dispose](): void;
    channels_json(): string;
    dispatch_event(event_json: string): void;
    loops_json(): string;
    mesh_json(): string;
    constructor();
    pods_json(): string;
    runners_json(): string;
    tickets_json(): string;
}

export class WasmAuthApiService {
    private constructor();
    free(): void;
    [Symbol.dispose](): void;
    forgot_password(email: string): Promise<string>;
    register(json: string): Promise<string>;
    resend_verification(email: string): Promise<string>;
    reset_password(json: string): Promise<string>;
    verify_email(token: string): Promise<string>;
}

export class WasmAuthManager {
    free(): void;
    [Symbol.dispose](): void;
    /**
     * Apply an already-obtained AuthSession (SSO / register callback path).
     * Writes token + refresh_token + user into Rust AuthState and persists.
     */
    apply_session(session_json: string): void;
    /**
     * Clear all session data (logout without API call). Useful for test reset.
     */
    clear_session(): void;
    fetch_organizations(): Promise<string>;
    get_current_org_json(): any;
    get_current_user_json(): any;
    get_organizations_json(): string;
    get_refresh_token(): string | undefined;
    get_token(): string | undefined;
    is_authenticated(): boolean;
    login(email: string, password: string): Promise<string>;
    logout(): Promise<void>;
    constructor(base_url: string);
    static new_with_storage(base_url: string, storage: any): WasmAuthManager;
    refresh_token(): Promise<string>;
    restore_session(): boolean;
    /**
     * Set or clear current organization. Empty json string clears it.
     */
    set_current_org(org_json: string): void;
    /**
     * Replace the organizations list (e.g. after a refetch outside fetch_organizations).
     * Also promotes the first org to current_org if none is set.
     */
    set_organizations(orgs_json: string): void;
    switch_org(slug: string): void;
    readonly base_url: string;
}

export class WasmAutopilotService {
    private constructor();
    free(): void;
    [Symbol.dispose](): void;
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

export class WasmAutopilotState {
    free(): void;
    [Symbol.dispose](): void;
    add_controller(json: string): void;
    add_iteration(key: string, json: string): void;
    controllers_json(): string;
    current_controller_json(): any;
    get_controller_by_pod_key_json(pod_key: string): any;
    get_iterations_json(key: string): any;
    get_thinking_history_json(key: string): any;
    get_thinking_json(key: string): any;
    constructor();
    remove_controller(key: string): void;
    set_controllers(json: string): void;
    set_current_controller(json: string): void;
    set_iterations(key: string, json: string): void;
    update_controller(key: string, json: string): void;
    update_thinking(key: string, json: string): void;
}

export class WasmBillingService {
    private constructor();
    free(): void;
    [Symbol.dispose](): void;
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

export class WasmBindingService {
    private constructor();
    free(): void;
    [Symbol.dispose](): void;
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

export class WasmBlockstoreService {
    private constructor();
    free(): void;
    [Symbol.dispose](): void;
    apply_ops(req_json: string): Promise<string>;
    apply_remote_op(op_json: string): void;
    backlinks_json(): string;
    blocks_json(): string;
    catchup(workspace_id: string): Promise<void>;
    ensure_default_workspace(): Promise<string>;
    get_block_json(id: string): any;
    last_op_id(workspace_id: string): bigint;
    last_op_ids_json(): string;
    list_backlinks_json(target_id: string): string;
    list_children_json(parent_id: string): string;
    list_workspaces(): Promise<string>;
    load_subtree(workspace_id: string, root_id: string): Promise<void>;
    load_type_defs(workspace_id: string): Promise<void>;
    nest_children_json(): string;
    refs_json(): string;
    semantic_search(workspace_id: string, req_json: string): Promise<string>;
    set_last_op_id(workspace_id: string, id: bigint): void;
    type_defs_json(workspace_id: string): string;
    workspaces_json(): string;
}

export class WasmChannelService {
    private constructor();
    free(): void;
    [Symbol.dispose](): void;
    add_channel_local(json: string): void;
    add_message(channel_id: bigint, json: string): void;
    archive_channel(id: bigint): Promise<void>;
    channel_members_json(id: bigint): string;
    channel_pods_json(id: bigint): string;
    channels_json(): string;
    clear_channel_mentions(channel_id: bigint): void;
    clear_channel_unread(channel_id: bigint): void;
    create_channel(request_json: string): Promise<string>;
    current_channel_json(): any;
    delete_message(channel_id: bigint, message_id: bigint): Promise<void>;
    edit_message(channel_id: bigint, message_id: bigint, content: string): Promise<string>;
    fetch_channel(id: bigint): Promise<string>;
    fetch_channel_members(id: bigint): Promise<string>;
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
    invite_channel_members(id: bigint, user_ids_json: string): Promise<void>;
    join_channel(channel_id: bigint, pod_key: string): Promise<string>;
    leave_channel(channel_id: bigint, pod_key: string): Promise<string>;
    mark_read(channel_id: bigint, message_id: bigint): Promise<void>;
    mention_counts_json(): string;
    mute_channel(channel_id: bigint, muted: boolean): Promise<void>;
    on_new_message(json: string): boolean;
    prepend_messages(channel_id: bigint, json: string, has_more: boolean): void;
    remove_channel_local(id: bigint): void;
    remove_channel_member(id: bigint, user_id: bigint): Promise<void>;
    remove_message_local(channel_id: bigint, message_id: bigint): void;
    search_channel_messages(id: bigint, q: string, limit?: number | null): Promise<string>;
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

export class WasmChannelState {
    free(): void;
    [Symbol.dispose](): void;
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
    /**
     * Return all mention counts as JSON.
     */
    mention_counts_json(): string;
    constructor();
    /**
     * Handle a new incoming message (from realtime event).
     * Enriches sender, updates preview, increments unread if appropriate.
     * Returns true if the message was new (not a duplicate).
     */
    on_new_message(message_json: string): boolean;
    prepend_messages(channel_id: bigint, messages_json: string, has_more: boolean): void;
    remove_channel(id: bigint): void;
    remove_message(channel_id: bigint, message_id: bigint): void;
    /**
     * Atomically: set current channel + clear unread + clear mentions.
     */
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
    /**
     * Return all unread counts as JSON: `{"1": 3, "2": 5}`.
     */
    unread_counts_json(): string;
    update_channel(id: bigint, json: string): void;
    update_message(channel_id: bigint, message_json: string): void;
}

export class WasmEventsManager {
    free(): void;
    [Symbol.dispose](): void;
    connect(): Promise<void>;
    disconnect(): Promise<void>;
    get_connection_state(): Promise<string>;
    constructor(ws_url: string);
    static new_with_options(ws_url: string, max_reconnect_attempts: number, initial_reconnect_delay_ms: number, max_reconnect_delay_ms: number, ping_interval_ms: number, pong_timeout_ms: number): WasmEventsManager;
    on_connection_state_change(callback: Function): Promise<number>;
    subscribe(event_type: string, callback: Function): Promise<number>;
    subscribe_all(callback: Function): Promise<number>;
    unsubscribe(id: number): Promise<void>;
}

export class WasmExtensionService {
    private constructor();
    free(): void;
    [Symbol.dispose](): void;
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

export class WasmFileService {
    private constructor();
    free(): void;
    [Symbol.dispose](): void;
    presign_upload(json: string): Promise<string>;
    upload_file(file_data: Uint8Array, filename: string, content_type: string): Promise<string>;
}

export class WasmGitProviderState {
    free(): void;
    [Symbol.dispose](): void;
    add_provider(json: string): void;
    available_projects_json(): string;
    current_provider_json(): any;
    constructor();
    providers_json(): string;
    remove_provider(id: string): void;
    set_available_projects(json: string): void;
    set_current_provider(json: string): void;
    set_providers(json: string): void;
    update_provider(id: string, json: string): void;
}

export class WasmGrantService {
    private constructor();
    free(): void;
    [Symbol.dispose](): void;
    grant(resource_type: string, resource_id: string, user_id: bigint): Promise<string>;
    list(resource_type: string, resource_id: string): Promise<string>;
    revoke(resource_type: string, resource_id: string, grant_id: bigint): Promise<void>;
}

export class WasmInvitationService {
    private constructor();
    free(): void;
    [Symbol.dispose](): void;
    accept(token: string): Promise<void>;
    create(json: string): Promise<string>;
    get_by_token(token: string): Promise<string>;
    list(): Promise<string>;
    list_pending(): Promise<string>;
    resend(id: bigint): Promise<void>;
    revoke(id: bigint): Promise<void>;
}

export class WasmLoopService {
    private constructor();
    free(): void;
    [Symbol.dispose](): void;
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

export class WasmLoopState {
    free(): void;
    [Symbol.dispose](): void;
    add_run(run_json: string): void;
    append_runs(json: string): void;
    clear_runs(): void;
    current_loop_json(): any;
    get_loop_by_slug_json(slug: string): any;
    loops_json(): string;
    constructor();
    runs_json(): string;
    set_current_loop(json: string): void;
    set_loops(json: string): void;
    set_runs(json: string): void;
    update_loop(slug: string, json: string): void;
    update_run_status(run_id: bigint, status: string): void;
}

export class WasmMeshService {
    private constructor();
    free(): void;
    [Symbol.dispose](): void;
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

export class WasmMeshState {
    free(): void;
    [Symbol.dispose](): void;
    clear_topology(): void;
    get_active_nodes_json(): string;
    get_channels_for_node_json(pod_key: string): string;
    get_edges_for_node_json(pod_key: string): string;
    get_node_json(pod_key: string): any;
    get_nodes_by_runner_json(runner_id: bigint): string;
    get_runner_info_json(runner_id: bigint): any;
    constructor();
    select_node(pod_key?: string | null): void;
    selected_node(): any;
    set_topology(json: string): void;
    topology_json(): any;
}

export class WasmMessageService {
    private constructor();
    free(): void;
    [Symbol.dispose](): void;
    get_conversation(correlation_id: string, limit?: number | null): Promise<string>;
    get_dead_letters(limit?: number | null, offset?: number | null): Promise<string>;
    get_message(id: bigint): Promise<string>;
    get_messages(unread_only?: boolean | null, limit?: number | null, offset?: number | null): Promise<string>;
    get_sent_messages(limit?: number | null, offset?: number | null): Promise<string>;
    get_unread_count(): Promise<string>;
    mark_all_read(): Promise<void>;
    mark_read(json: string): Promise<void>;
    replay_dead_letter(entry_id: bigint): Promise<void>;
    send_message(json: string, pod_key?: string | null): Promise<string>;
}

export class WasmNotificationService {
    private constructor();
    free(): void;
    [Symbol.dispose](): void;
    get_preferences(): Promise<string>;
    set_preference(json: string): Promise<string>;
}

export class WasmOrgApiService {
    private constructor();
    free(): void;
    [Symbol.dispose](): void;
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

export class WasmOrgState {
    free(): void;
    [Symbol.dispose](): void;
    add_member(json: string): void;
    add_organization(json: string): void;
    current_org_json(): any;
    members_json(): string;
    constructor();
    organizations_json(): string;
    remove_member(id: string): void;
    remove_organization(id: number): void;
    set_current_org(json: string): void;
    set_members(json: string): void;
    set_organizations(json: string): void;
    update_member(user_id: number, json: string): void;
    update_organization(id: number, json: string): void;
}

export class WasmPodService {
    private constructor();
    free(): void;
    [Symbol.dispose](): void;
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

export class WasmPodState {
    free(): void;
    [Symbol.dispose](): void;
    current_pod_json(): any;
    get_pod_json(pod_key: string): any;
    constructor();
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

export class WasmPromoCodeService {
    private constructor();
    free(): void;
    [Symbol.dispose](): void;
    get_history(): Promise<string>;
    redeem(json: string): Promise<void>;
    validate(json: string): Promise<string>;
}

export class WasmRelayManager {
    free(): void;
    [Symbol.dispose](): void;
    disconnect(pod_key: string): Promise<void>;
    disconnect_all(): Promise<void>;
    force_resize(pod_key: string, cols: number, rows: number): Promise<void>;
    get_pod_size(pod_key: string): Promise<any>;
    get_status(pod_key: string): Promise<string>;
    is_runner_disconnected(pod_key: string): Promise<boolean>;
    constructor();
    on_acp_message(pod_key: string, callback: Function): Promise<void>;
    on_status_change(pod_key: string, callback: Function): Promise<void>;
    send(pod_key: string, data: string): Promise<void>;
    send_acp_command(pod_key: string, command: string): Promise<void>;
    send_resize(pod_key: string, cols: number, rows: number): Promise<void>;
    subscribe(pod_key: string, subscription_id: string, relay_url: string, token: string, callback: Function): Promise<void>;
    unsubscribe(pod_key: string, subscription_id: string): Promise<void>;
}

export class WasmRepoState {
    free(): void;
    [Symbol.dispose](): void;
    add_repository(json: string): void;
    branches_json(): string;
    current_repo_json(): any;
    constructor();
    remove_repository(id: string): void;
    repositories_json(): string;
    set_branches(json: string): void;
    set_current_repo(json: string): void;
    set_repositories(json: string): void;
    update_repository(id: string, json: string): void;
}

export class WasmRepositoryService {
    private constructor();
    free(): void;
    [Symbol.dispose](): void;
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

export class WasmRunnerService {
    private constructor();
    free(): void;
    [Symbol.dispose](): void;
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

export class WasmRunnerState {
    free(): void;
    [Symbol.dispose](): void;
    available_runners_json(): string;
    static can_accept_pods(runner_json: string): boolean;
    current_runner_json(): any;
    get_runner_json(id: bigint): any;
    constructor();
    remove_runner(id: bigint): void;
    runners_json(): string;
    set_available_runners(json: string): void;
    set_current_runner(json: string): void;
    set_runners(json: string): void;
    update_runner(id: number, json: string): void;
    update_runner_status(id: bigint, status: string): void;
}

export class WasmSSOService {
    private constructor();
    free(): void;
    [Symbol.dispose](): void;
    discover(email: string): Promise<string>;
    ldap_auth(domain: string, json: string): Promise<string>;
}

export class WasmSupportTicketService {
    private constructor();
    free(): void;
    [Symbol.dispose](): void;
    add_message(ticket_id: bigint, content: string, file_data: Uint8Array[], file_names: string[]): Promise<string>;
    create_ticket(title: string, category: string, content: string, priority: string | null | undefined, file_data: Uint8Array[], file_names: string[]): Promise<string>;
    get_attachment_url(id: bigint): Promise<string>;
    get_detail(id: bigint): Promise<string>;
    list(status?: string | null, page?: number | null, page_size?: number | null): Promise<string>;
}

export class WasmTicketRelationsService {
    private constructor();
    free(): void;
    [Symbol.dispose](): void;
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

export class WasmTicketService {
    private constructor();
    free(): void;
    [Symbol.dispose](): void;
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
    ticket_pods_json(slug: string): string;
    tickets_json(): string;
    update_ticket(slug: string, request_json: string): Promise<string>;
    update_ticket_local(slug: string, json: string): void;
    update_ticket_status(slug: string, status: string): Promise<string>;
    update_ticket_status_local(slug: string, status: string): void;
}

export class WasmTicketState {
    free(): void;
    [Symbol.dispose](): void;
    add_label(label_json: string): void;
    add_ticket(ticket_json: string): void;
    append_column_tickets(status: string, tickets_json: string): void;
    board_columns_json(): string;
    current_ticket_json(): any;
    filter_tickets_json(search: string, statuses_json: string, priorities_json: string, repository_ids_json: string): string;
    get_ticket_by_slug_json(slug: string): any;
    labels_json(): string;
    constructor();
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

export class WasmTokenUsageService {
    private constructor();
    free(): void;
    [Symbol.dispose](): void;
    get_dashboard(start_time?: string | null, end_time?: string | null, agent_slug?: string | null, user_id?: bigint | null, model?: string | null, granularity?: string | null): Promise<string>;
}

export class WasmUserApiService {
    private constructor();
    free(): void;
    [Symbol.dispose](): void;
    get_me(): Promise<string>;
    get_organizations(): Promise<string>;
}

export class WasmUserCredentialService {
    private constructor();
    free(): void;
    [Symbol.dispose](): void;
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

export class WasmUserState {
    free(): void;
    [Symbol.dispose](): void;
    add_identity(json: string): void;
    identities_json(): string;
    constructor();
    profile_json(): any;
    remove_identity(id: string): void;
    set_profile(json: string): void;
}

export class WasmWebSocket {
    private constructor();
    free(): void;
    [Symbol.dispose](): void;
    close(): void;
    static connect(url: string, on_open: Function, on_message: Function, on_close: Function, on_error: Function): WasmWebSocket;
    is_closed(): boolean;
    is_open(): boolean;
    send_binary(data: Uint8Array): void;
    send_text(text: string): void;
}

export function init_panic_hook(): void;

export function relay_decode_message(data: Uint8Array): any;

export function relay_encode_acp_command(data: Uint8Array): Uint8Array;

export function relay_encode_control(data: Uint8Array): Uint8Array;

export function relay_encode_input(data: Uint8Array): Uint8Array;

export function relay_encode_ping(): Uint8Array;

export function relay_encode_resize(cols: number, rows: number): Uint8Array;

export function relay_encode_resync(): Uint8Array;

export function version(): string;

export type InitInput = RequestInfo | URL | Response | BufferSource | WebAssembly.Module;

export interface InitOutput {
    readonly memory: WebAssembly.Memory;
    readonly __wbg_wasmapiclient_free: (a: number, b: number) => void;
    readonly __wbg_wasmbindingservice_free: (a: number, b: number) => void;
    readonly __wbg_wasmgitproviderstate_free: (a: number, b: number) => void;
    readonly __wbg_wasmrunnerstate_free: (a: number, b: number) => void;
    readonly __wbg_wasmuserapiservice_free: (a: number, b: number) => void;
    readonly wasmapiclient_base_url: (a: number) => [number, number];
    readonly wasmapiclient_clear_auth: (a: number) => void;
    readonly wasmapiclient_create_agent_service: (a: number) => number;
    readonly wasmapiclient_create_autopilot_service: (a: number) => number;
    readonly wasmapiclient_create_blockstore_service: (a: number) => number;
    readonly wasmapiclient_create_channel_service: (a: number) => number;
    readonly wasmapiclient_create_loop_service: (a: number) => number;
    readonly wasmapiclient_create_mesh_service: (a: number) => number;
    readonly wasmapiclient_create_pod_service: (a: number) => number;
    readonly wasmapiclient_create_runner_service: (a: number) => number;
    readonly wasmapiclient_create_ticket_service: (a: number) => number;
    readonly wasmapiclient_delete: (a: number, b: number, c: number) => any;
    readonly wasmapiclient_get: (a: number, b: number, c: number) => any;
    readonly wasmapiclient_get_org_slug: (a: number) => [number, number];
    readonly wasmapiclient_get_token: (a: number) => [number, number];
    readonly wasmapiclient_new: (a: number, b: number) => number;
    readonly wasmapiclient_org_path: (a: number, b: number, c: number) => [number, number];
    readonly wasmapiclient_patch: (a: number, b: number, c: number, d: number, e: number) => any;
    readonly wasmapiclient_post: (a: number, b: number, c: number, d: number, e: number) => any;
    readonly wasmapiclient_public_get: (a: number, b: number, c: number) => any;
    readonly wasmapiclient_public_post: (a: number, b: number, c: number, d: number, e: number) => any;
    readonly wasmapiclient_put: (a: number, b: number, c: number, d: number, e: number) => any;
    readonly wasmapiclient_set_org_slug: (a: number, b: number, c: number) => void;
    readonly wasmapiclient_set_token: (a: number, b: number, c: number, d: number, e: number) => void;
    readonly wasmbindingservice_accept_binding: (a: number, b: number, c: number) => any;
    readonly wasmbindingservice_approve_scopes: (a: number, b: bigint, c: number, d: number) => any;
    readonly wasmbindingservice_check_binding: (a: number, b: number, c: number) => any;
    readonly wasmbindingservice_get_bound_pods: (a: number) => any;
    readonly wasmbindingservice_get_pending_bindings: (a: number) => any;
    readonly wasmbindingservice_list_bindings: (a: number, b: number, c: number) => any;
    readonly wasmbindingservice_reject_binding: (a: number, b: number, c: number) => any;
    readonly wasmbindingservice_request_binding: (a: number, b: number, c: number, d: number, e: number) => any;
    readonly wasmbindingservice_request_scopes: (a: number, b: bigint, c: number, d: number) => any;
    readonly wasmbindingservice_unbind: (a: number, b: number, c: number) => any;
    readonly wasmgitproviderstate_add_provider: (a: number, b: number, c: number) => void;
    readonly wasmgitproviderstate_available_projects_json: (a: number) => [number, number];
    readonly wasmgitproviderstate_current_provider_json: (a: number) => any;
    readonly wasmgitproviderstate_new: () => number;
    readonly wasmgitproviderstate_providers_json: (a: number) => [number, number];
    readonly wasmgitproviderstate_remove_provider: (a: number, b: number, c: number) => void;
    readonly wasmgitproviderstate_set_available_projects: (a: number, b: number, c: number) => void;
    readonly wasmgitproviderstate_set_current_provider: (a: number, b: number, c: number) => void;
    readonly wasmgitproviderstate_set_providers: (a: number, b: number, c: number) => void;
    readonly wasmgitproviderstate_update_provider: (a: number, b: number, c: number, d: number, e: number) => void;
    readonly wasmrunnerstate_available_runners_json: (a: number) => [number, number];
    readonly wasmrunnerstate_can_accept_pods: (a: number, b: number) => number;
    readonly wasmrunnerstate_current_runner_json: (a: number) => any;
    readonly wasmrunnerstate_get_runner_json: (a: number, b: bigint) => any;
    readonly wasmrunnerstate_new: () => number;
    readonly wasmrunnerstate_remove_runner: (a: number, b: bigint) => void;
    readonly wasmrunnerstate_runners_json: (a: number) => [number, number];
    readonly wasmrunnerstate_set_available_runners: (a: number, b: number, c: number) => void;
    readonly wasmrunnerstate_set_current_runner: (a: number, b: number, c: number) => void;
    readonly wasmrunnerstate_set_runners: (a: number, b: number, c: number) => void;
    readonly wasmrunnerstate_update_runner: (a: number, b: number, c: number, d: number) => void;
    readonly wasmrunnerstate_update_runner_status: (a: number, b: bigint, c: number, d: number) => void;
    readonly wasmuserapiservice_get_me: (a: number) => any;
    readonly wasmuserapiservice_get_organizations: (a: number) => any;
    readonly wasmapiclient_create_apikey_service: (a: number) => number;
    readonly wasmapiclient_create_auth_api_service: (a: number) => number;
    readonly wasmapiclient_create_billing_service: (a: number) => number;
    readonly wasmapiclient_create_binding_service: (a: number) => number;
    readonly wasmapiclient_create_extension_service: (a: number) => number;
    readonly wasmapiclient_create_file_service: (a: number) => number;
    readonly wasmapiclient_create_grant_service: (a: number) => number;
    readonly wasmapiclient_create_invitation_service: (a: number) => number;
    readonly wasmapiclient_create_message_service: (a: number) => number;
    readonly wasmapiclient_create_notification_service: (a: number) => number;
    readonly wasmapiclient_create_org_api_service: (a: number) => number;
    readonly wasmapiclient_create_promocode_service: (a: number) => number;
    readonly wasmapiclient_create_repository_service: (a: number) => number;
    readonly wasmapiclient_create_sso_service: (a: number) => number;
    readonly wasmapiclient_create_support_ticket_service: (a: number) => number;
    readonly wasmapiclient_create_ticket_relations_service: (a: number) => number;
    readonly wasmapiclient_create_token_usage_service: (a: number) => number;
    readonly wasmapiclient_create_user_api_service: (a: number) => number;
    readonly wasmapiclient_create_user_credential_service: (a: number) => number;
    readonly __wbg_wasmagentservice_free: (a: number, b: number) => void;
    readonly __wbg_wasmchannelservice_free: (a: number, b: number) => void;
    readonly __wbg_wasmextensionservice_free: (a: number, b: number) => void;
    readonly __wbg_wasmnotificationservice_free: (a: number, b: number) => void;
    readonly __wbg_wasmsupportticketservice_free: (a: number, b: number) => void;
    readonly wasmagentservice_create_provider: (a: number, b: number, c: number) => any;
    readonly wasmagentservice_delete_provider: (a: number, b: bigint) => any;
    readonly wasmagentservice_delete_user_config: (a: number, b: number, c: number) => any;
    readonly wasmagentservice_get_agentpod_settings: (a: number) => any;
    readonly wasmagentservice_get_config_schema: (a: number, b: number, c: number) => any;
    readonly wasmagentservice_get_user_config: (a: number, b: number, c: number) => any;
    readonly wasmagentservice_list_agents: (a: number) => any;
    readonly wasmagentservice_list_providers: (a: number) => any;
    readonly wasmagentservice_list_user_configs: (a: number) => any;
    readonly wasmagentservice_set_default_provider: (a: number, b: bigint) => any;
    readonly wasmagentservice_set_user_config: (a: number, b: number, c: number, d: number, e: number) => any;
    readonly wasmagentservice_update_agentpod_settings: (a: number, b: number, c: number) => any;
    readonly wasmagentservice_update_provider: (a: number, b: bigint, c: number, d: number) => any;
    readonly wasmchannelservice_add_channel_local: (a: number, b: number, c: number) => void;
    readonly wasmchannelservice_add_message: (a: number, b: bigint, c: number, d: number) => void;
    readonly wasmchannelservice_archive_channel: (a: number, b: bigint) => any;
    readonly wasmchannelservice_channel_members_json: (a: number, b: bigint) => [number, number];
    readonly wasmchannelservice_channel_pods_json: (a: number, b: bigint) => [number, number];
    readonly wasmchannelservice_channels_json: (a: number) => [number, number];
    readonly wasmchannelservice_clear_channel_mentions: (a: number, b: bigint) => void;
    readonly wasmchannelservice_clear_channel_unread: (a: number, b: bigint) => void;
    readonly wasmchannelservice_create_channel: (a: number, b: number, c: number) => any;
    readonly wasmchannelservice_current_channel_json: (a: number) => any;
    readonly wasmchannelservice_delete_message: (a: number, b: bigint, c: bigint) => any;
    readonly wasmchannelservice_edit_message: (a: number, b: bigint, c: bigint, d: number, e: number) => any;
    readonly wasmchannelservice_fetch_channel: (a: number, b: bigint) => any;
    readonly wasmchannelservice_fetch_channel_members: (a: number, b: bigint) => any;
    readonly wasmchannelservice_fetch_channels: (a: number, b: number) => any;
    readonly wasmchannelservice_fetch_messages: (a: number, b: bigint, c: number, d: number, e: bigint) => any;
    readonly wasmchannelservice_fetch_unread_counts: (a: number) => any;
    readonly wasmchannelservice_filter_channels_json: (a: number, b: number, c: number, d: number) => [number, number];
    readonly wasmchannelservice_get_channel_json: (a: number, b: bigint) => any;
    readonly wasmchannelservice_get_channel_pods: (a: number, b: bigint) => any;
    readonly wasmchannelservice_get_last_message_json: (a: number, b: bigint) => any;
    readonly wasmchannelservice_get_mention_count: (a: number, b: bigint) => number;
    readonly wasmchannelservice_get_messages_json: (a: number, b: bigint) => any;
    readonly wasmchannelservice_get_unread_count: (a: number, b: bigint) => number;
    readonly wasmchannelservice_increment_mention: (a: number, b: bigint) => void;
    readonly wasmchannelservice_increment_unread: (a: number, b: bigint) => void;
    readonly wasmchannelservice_invite_channel_members: (a: number, b: bigint, c: number, d: number) => any;
    readonly wasmchannelservice_join_channel: (a: number, b: bigint, c: number, d: number) => any;
    readonly wasmchannelservice_leave_channel: (a: number, b: bigint, c: number, d: number) => any;
    readonly wasmchannelservice_mark_read: (a: number, b: bigint, c: bigint) => any;
    readonly wasmchannelservice_mention_counts_json: (a: number) => [number, number];
    readonly wasmchannelservice_mute_channel: (a: number, b: bigint, c: number) => any;
    readonly wasmchannelservice_on_new_message: (a: number, b: number, c: number) => number;
    readonly wasmchannelservice_prepend_messages: (a: number, b: bigint, c: number, d: number, e: number) => void;
    readonly wasmchannelservice_remove_channel_local: (a: number, b: bigint) => void;
    readonly wasmchannelservice_remove_channel_member: (a: number, b: bigint, c: bigint) => any;
    readonly wasmchannelservice_remove_message_local: (a: number, b: bigint, c: bigint) => void;
    readonly wasmchannelservice_search_channel_messages: (a: number, b: bigint, c: number, d: number, e: number) => any;
    readonly wasmchannelservice_select_channel: (a: number, b: number, c: bigint) => any;
    readonly wasmchannelservice_send_message: (a: number, b: bigint, c: number, d: number) => any;
    readonly wasmchannelservice_set_channels: (a: number, b: number, c: number) => void;
    readonly wasmchannelservice_set_current_channel: (a: number, b: number, c: bigint) => void;
    readonly wasmchannelservice_set_current_user: (a: number, b: number, c: number) => void;
    readonly wasmchannelservice_set_current_user_id: (a: number, b: number, c: bigint) => void;
    readonly wasmchannelservice_set_last_message: (a: number, b: bigint, c: number, d: number) => void;
    readonly wasmchannelservice_set_mention_counts: (a: number, b: number, c: number) => void;
    readonly wasmchannelservice_set_messages: (a: number, b: bigint, c: number, d: number, e: number) => void;
    readonly wasmchannelservice_set_unread_counts: (a: number, b: number, c: number) => void;
    readonly wasmchannelservice_sorted_channel_ids_json: (a: number, b: number, c: number, d: number) => [number, number];
    readonly wasmchannelservice_total_mention_count: (a: number) => number;
    readonly wasmchannelservice_total_unread_count: (a: number) => number;
    readonly wasmchannelservice_unarchive_channel: (a: number, b: bigint) => any;
    readonly wasmchannelservice_unread_counts_json: (a: number) => [number, number];
    readonly wasmchannelservice_update_channel: (a: number, b: bigint, c: number, d: number) => any;
    readonly wasmchannelservice_update_channel_local: (a: number, b: bigint, c: number, d: number) => void;
    readonly wasmchannelservice_update_message_local: (a: number, b: bigint, c: number, d: number) => void;
    readonly wasmextensionservice_create_skill_registry: (a: number, b: number, c: number) => any;
    readonly wasmextensionservice_delete_skill_registry: (a: number, b: bigint) => any;
    readonly wasmextensionservice_install_custom_mcp_server: (a: number, b: bigint, c: number, d: number) => any;
    readonly wasmextensionservice_install_mcp_from_market: (a: number, b: bigint, c: number, d: number) => any;
    readonly wasmextensionservice_install_skill_from_github: (a: number, b: bigint, c: number, d: number) => any;
    readonly wasmextensionservice_install_skill_from_market: (a: number, b: bigint, c: number, d: number) => any;
    readonly wasmextensionservice_install_skill_from_upload: (a: number, b: bigint, c: any, d: number, e: number, f: number, g: number) => any;
    readonly wasmextensionservice_list_market_mcp_servers: (a: number, b: number, c: number, d: number, e: number) => any;
    readonly wasmextensionservice_list_market_skills: (a: number, b: number, c: number, d: number, e: number) => any;
    readonly wasmextensionservice_list_repo_mcp_servers: (a: number, b: bigint, c: number, d: number) => any;
    readonly wasmextensionservice_list_repo_skills: (a: number, b: bigint, c: number, d: number) => any;
    readonly wasmextensionservice_list_skill_registries: (a: number) => any;
    readonly wasmextensionservice_list_skill_registry_overrides: (a: number) => any;
    readonly wasmextensionservice_sync_skill_registry: (a: number, b: bigint) => any;
    readonly wasmextensionservice_toggle_skill_registry: (a: number, b: bigint, c: number, d: number) => any;
    readonly wasmextensionservice_uninstall_mcp_server: (a: number, b: bigint, c: bigint) => any;
    readonly wasmextensionservice_uninstall_skill: (a: number, b: bigint, c: bigint) => any;
    readonly wasmextensionservice_update_mcp_server: (a: number, b: bigint, c: bigint, d: number, e: number) => any;
    readonly wasmextensionservice_update_skill: (a: number, b: bigint, c: bigint, d: number, e: number) => any;
    readonly wasmnotificationservice_get_preferences: (a: number) => any;
    readonly wasmnotificationservice_set_preference: (a: number, b: number, c: number) => any;
    readonly wasmsupportticketservice_add_message: (a: number, b: bigint, c: number, d: number, e: number, f: number, g: number, h: number) => any;
    readonly wasmsupportticketservice_create_ticket: (a: number, b: number, c: number, d: number, e: number, f: number, g: number, h: number, i: number, j: number, k: number, l: number, m: number) => any;
    readonly wasmsupportticketservice_get_attachment_url: (a: number, b: bigint) => any;
    readonly wasmsupportticketservice_get_detail: (a: number, b: bigint) => any;
    readonly wasmsupportticketservice_list: (a: number, b: number, c: number, d: number, e: number) => any;
    readonly __wbg_wasmapikeyservice_free: (a: number, b: number) => void;
    readonly __wbg_wasmbillingservice_free: (a: number, b: number) => void;
    readonly __wbg_wasmloopservice_free: (a: number, b: number) => void;
    readonly __wbg_wasmticketrelationsservice_free: (a: number, b: number) => void;
    readonly __wbg_wasmticketservice_free: (a: number, b: number) => void;
    readonly wasmapikeyservice_create: (a: number, b: number, c: number) => any;
    readonly wasmapikeyservice_delete: (a: number, b: bigint) => any;
    readonly wasmapikeyservice_get: (a: number, b: bigint) => any;
    readonly wasmapikeyservice_list: (a: number) => any;
    readonly wasmapikeyservice_revoke: (a: number, b: bigint) => any;
    readonly wasmapikeyservice_update: (a: number, b: bigint, c: number, d: number) => any;
    readonly wasmbillingservice_cancel_subscription: (a: number) => any;
    readonly wasmbillingservice_change_cycle: (a: number, b: number, c: number) => any;
    readonly wasmbillingservice_check_quota: (a: number, b: number, c: number, d: number) => any;
    readonly wasmbillingservice_create_checkout: (a: number, b: number, c: number) => any;
    readonly wasmbillingservice_create_subscription: (a: number, b: number, c: number) => any;
    readonly wasmbillingservice_get_checkout_status: (a: number, b: number, c: number) => any;
    readonly wasmbillingservice_get_customer_portal: (a: number, b: number, c: number) => any;
    readonly wasmbillingservice_get_deployment_info: (a: number) => any;
    readonly wasmbillingservice_get_overview: (a: number) => any;
    readonly wasmbillingservice_get_public_deployment_info: (a: number) => any;
    readonly wasmbillingservice_get_public_pricing: (a: number) => any;
    readonly wasmbillingservice_get_seat_usage: (a: number) => any;
    readonly wasmbillingservice_get_subscription: (a: number) => any;
    readonly wasmbillingservice_get_usage: (a: number, b: number, c: number) => any;
    readonly wasmbillingservice_list_invoices: (a: number, b: number, c: number) => any;
    readonly wasmbillingservice_list_plans: (a: number) => any;
    readonly wasmbillingservice_purchase_seats: (a: number, b: number, c: number) => any;
    readonly wasmbillingservice_reactivate: (a: number) => any;
    readonly wasmbillingservice_request_cancel: (a: number, b: number, c: number) => any;
    readonly wasmbillingservice_update_auto_renew: (a: number, b: number, c: number) => any;
    readonly wasmbillingservice_update_subscription: (a: number, b: number, c: number) => any;
    readonly wasmbillingservice_upgrade: (a: number, b: number, c: number) => any;
    readonly wasmloopservice_add_run: (a: number, b: number, c: number) => void;
    readonly wasmloopservice_append_runs: (a: number, b: number, c: number) => void;
    readonly wasmloopservice_cancel_run: (a: number, b: number, c: number, d: bigint) => any;
    readonly wasmloopservice_clear_runs: (a: number) => void;
    readonly wasmloopservice_create_loop: (a: number, b: number, c: number) => any;
    readonly wasmloopservice_current_loop_json: (a: number) => any;
    readonly wasmloopservice_delete_loop: (a: number, b: number, c: number) => any;
    readonly wasmloopservice_disable_loop: (a: number, b: number, c: number) => any;
    readonly wasmloopservice_enable_loop: (a: number, b: number, c: number) => any;
    readonly wasmloopservice_fetch_loop: (a: number, b: number, c: number) => any;
    readonly wasmloopservice_fetch_loops: (a: number, b: number, c: number, d: number, e: number) => any;
    readonly wasmloopservice_fetch_runs: (a: number, b: number, c: number, d: number, e: number, f: number, g: number) => any;
    readonly wasmloopservice_get_loop_by_slug_json: (a: number, b: number, c: number) => any;
    readonly wasmloopservice_loops_json: (a: number) => [number, number];
    readonly wasmloopservice_runs_json: (a: number) => [number, number];
    readonly wasmloopservice_set_current_loop: (a: number, b: number, c: number) => void;
    readonly wasmloopservice_set_loops: (a: number, b: number, c: number) => void;
    readonly wasmloopservice_set_runs: (a: number, b: number, c: number) => void;
    readonly wasmloopservice_trigger_loop: (a: number, b: number, c: number) => any;
    readonly wasmloopservice_update_loop: (a: number, b: number, c: number, d: number, e: number) => any;
    readonly wasmloopservice_update_loop_local: (a: number, b: number, c: number, d: number, e: number) => void;
    readonly wasmloopservice_update_run_status: (a: number, b: bigint, c: number, d: number) => void;
    readonly wasmticketrelationsservice_create_comment: (a: number, b: number, c: number, d: number, e: number) => any;
    readonly wasmticketrelationsservice_create_relation: (a: number, b: number, c: number, d: number, e: number) => any;
    readonly wasmticketrelationsservice_delete_comment: (a: number, b: number, c: number, d: bigint) => any;
    readonly wasmticketrelationsservice_delete_relation: (a: number, b: number, c: number, d: bigint) => any;
    readonly wasmticketrelationsservice_link_commit: (a: number, b: number, c: number, d: number, e: number) => any;
    readonly wasmticketrelationsservice_list_comments: (a: number, b: number, c: number, d: number, e: number) => any;
    readonly wasmticketrelationsservice_list_commits: (a: number, b: number, c: number) => any;
    readonly wasmticketrelationsservice_list_merge_requests: (a: number, b: number, c: number) => any;
    readonly wasmticketrelationsservice_list_relations: (a: number, b: number, c: number) => any;
    readonly wasmticketrelationsservice_unlink_commit: (a: number, b: number, c: number, d: bigint) => any;
    readonly wasmticketrelationsservice_update_comment: (a: number, b: number, c: number, d: bigint, e: number, f: number) => any;
    readonly wasmticketservice_add_label: (a: number, b: number, c: number) => void;
    readonly wasmticketservice_add_ticket: (a: number, b: number, c: number) => void;
    readonly wasmticketservice_append_column_tickets: (a: number, b: number, c: number, d: number, e: number) => void;
    readonly wasmticketservice_board_columns_json: (a: number) => [number, number];
    readonly wasmticketservice_create_label: (a: number, b: number, c: number, d: number, e: number, f: number, g: bigint) => any;
    readonly wasmticketservice_create_ticket: (a: number, b: number, c: number) => any;
    readonly wasmticketservice_current_ticket_json: (a: number) => any;
    readonly wasmticketservice_delete_label: (a: number, b: number) => any;
    readonly wasmticketservice_delete_ticket: (a: number, b: number, c: number) => any;
    readonly wasmticketservice_fetch_board: (a: number, b: number, c: bigint) => any;
    readonly wasmticketservice_fetch_labels: (a: number, b: number, c: bigint) => any;
    readonly wasmticketservice_fetch_ticket: (a: number, b: number, c: number) => any;
    readonly wasmticketservice_fetch_tickets: (a: number, b: number, c: number, d: number, e: number) => any;
    readonly wasmticketservice_filter_tickets_json: (a: number, b: number, c: number, d: number, e: number, f: number, g: number, h: number, i: number) => [number, number];
    readonly wasmticketservice_get_sub_tickets: (a: number, b: number, c: number) => any;
    readonly wasmticketservice_get_ticket_by_slug_json: (a: number, b: number, c: number) => any;
    readonly wasmticketservice_get_ticket_pods: (a: number, b: number, c: number, d: number) => any;
    readonly wasmticketservice_labels_json: (a: number) => [number, number];
    readonly wasmticketservice_load_more_column: (a: number, b: number, c: number, d: number, e: number) => any;
    readonly wasmticketservice_remove_label: (a: number, b: number) => void;
    readonly wasmticketservice_remove_ticket: (a: number, b: number, c: number) => void;
    readonly wasmticketservice_set_board_columns: (a: number, b: number, c: number) => void;
    readonly wasmticketservice_set_current_ticket: (a: number, b: number, c: number) => void;
    readonly wasmticketservice_set_labels: (a: number, b: number, c: number) => void;
    readonly wasmticketservice_set_tickets: (a: number, b: number, c: number) => void;
    readonly wasmticketservice_ticket_pods_json: (a: number, b: number, c: number) => [number, number];
    readonly wasmticketservice_tickets_json: (a: number) => [number, number];
    readonly wasmticketservice_update_ticket: (a: number, b: number, c: number, d: number, e: number) => any;
    readonly wasmticketservice_update_ticket_local: (a: number, b: number, c: number, d: number, e: number) => void;
    readonly wasmticketservice_update_ticket_status: (a: number, b: number, c: number, d: number, e: number) => any;
    readonly wasmticketservice_update_ticket_status_local: (a: number, b: number, c: number, d: number, e: number) => void;
    readonly __wbg_wasmacpsessionmanager_free: (a: number, b: number) => void;
    readonly __wbg_wasmautopilotstate_free: (a: number, b: number) => void;
    readonly __wbg_wasmblockstoreservice_free: (a: number, b: number) => void;
    readonly __wbg_wasmmessageservice_free: (a: number, b: number) => void;
    readonly __wbg_wasmorgstate_free: (a: number, b: number) => void;
    readonly __wbg_wasmpromocodeservice_free: (a: number, b: number) => void;
    readonly __wbg_wasmticketstate_free: (a: number, b: number) => void;
    readonly wasmacpsessionmanager_add_content_chunk: (a: number, b: number, c: number, d: number, e: number, f: number, g: number) => void;
    readonly wasmacpsessionmanager_add_log: (a: number, b: number, c: number, d: number, e: number, f: number, g: number) => void;
    readonly wasmacpsessionmanager_add_permission_request: (a: number, b: number, c: number, d: number, e: number) => void;
    readonly wasmacpsessionmanager_add_thinking: (a: number, b: number, c: number, d: number, e: number) => void;
    readonly wasmacpsessionmanager_clear_session: (a: number, b: number, c: number) => void;
    readonly wasmacpsessionmanager_get_session_json: (a: number, b: number, c: number) => any;
    readonly wasmacpsessionmanager_mark_last_message_complete: (a: number, b: number, c: number) => void;
    readonly wasmacpsessionmanager_new: () => number;
    readonly wasmacpsessionmanager_remove_permission_request: (a: number, b: number, c: number, d: number, e: number) => void;
    readonly wasmacpsessionmanager_set_tool_call_result: (a: number, b: number, c: number, d: number, e: number, f: number, g: number, h: number, i: number, j: number) => void;
    readonly wasmacpsessionmanager_update_plan: (a: number, b: number, c: number, d: number, e: number) => void;
    readonly wasmacpsessionmanager_update_session_state: (a: number, b: number, c: number, d: number, e: number) => void;
    readonly wasmacpsessionmanager_update_tool_call: (a: number, b: number, c: number, d: number, e: number) => void;
    readonly wasmautopilotstate_add_controller: (a: number, b: number, c: number) => void;
    readonly wasmautopilotstate_add_iteration: (a: number, b: number, c: number, d: number, e: number) => void;
    readonly wasmautopilotstate_controllers_json: (a: number) => [number, number];
    readonly wasmautopilotstate_current_controller_json: (a: number) => any;
    readonly wasmautopilotstate_get_controller_by_pod_key_json: (a: number, b: number, c: number) => any;
    readonly wasmautopilotstate_get_iterations_json: (a: number, b: number, c: number) => any;
    readonly wasmautopilotstate_get_thinking_history_json: (a: number, b: number, c: number) => any;
    readonly wasmautopilotstate_get_thinking_json: (a: number, b: number, c: number) => any;
    readonly wasmautopilotstate_new: () => number;
    readonly wasmautopilotstate_remove_controller: (a: number, b: number, c: number) => void;
    readonly wasmautopilotstate_set_controllers: (a: number, b: number, c: number) => void;
    readonly wasmautopilotstate_set_current_controller: (a: number, b: number, c: number) => void;
    readonly wasmautopilotstate_set_iterations: (a: number, b: number, c: number, d: number, e: number) => void;
    readonly wasmautopilotstate_update_controller: (a: number, b: number, c: number, d: number, e: number) => void;
    readonly wasmautopilotstate_update_thinking: (a: number, b: number, c: number, d: number, e: number) => void;
    readonly wasmblockstoreservice_apply_ops: (a: number, b: number, c: number) => any;
    readonly wasmblockstoreservice_apply_remote_op: (a: number, b: number, c: number) => [number, number];
    readonly wasmblockstoreservice_backlinks_json: (a: number) => [number, number];
    readonly wasmblockstoreservice_blocks_json: (a: number) => [number, number];
    readonly wasmblockstoreservice_catchup: (a: number, b: number, c: number) => any;
    readonly wasmblockstoreservice_ensure_default_workspace: (a: number) => any;
    readonly wasmblockstoreservice_get_block_json: (a: number, b: number, c: number) => any;
    readonly wasmblockstoreservice_last_op_id: (a: number, b: number, c: number) => bigint;
    readonly wasmblockstoreservice_last_op_ids_json: (a: number) => [number, number];
    readonly wasmblockstoreservice_list_backlinks_json: (a: number, b: number, c: number) => [number, number];
    readonly wasmblockstoreservice_list_children_json: (a: number, b: number, c: number) => [number, number];
    readonly wasmblockstoreservice_list_workspaces: (a: number) => any;
    readonly wasmblockstoreservice_load_subtree: (a: number, b: number, c: number, d: number, e: number) => any;
    readonly wasmblockstoreservice_load_type_defs: (a: number, b: number, c: number) => any;
    readonly wasmblockstoreservice_nest_children_json: (a: number) => [number, number];
    readonly wasmblockstoreservice_refs_json: (a: number) => [number, number];
    readonly wasmblockstoreservice_semantic_search: (a: number, b: number, c: number, d: number, e: number) => any;
    readonly wasmblockstoreservice_set_last_op_id: (a: number, b: number, c: number, d: bigint) => void;
    readonly wasmblockstoreservice_type_defs_json: (a: number, b: number, c: number) => [number, number];
    readonly wasmblockstoreservice_workspaces_json: (a: number) => [number, number];
    readonly wasmmessageservice_get_conversation: (a: number, b: number, c: number, d: number) => any;
    readonly wasmmessageservice_get_dead_letters: (a: number, b: number, c: number) => any;
    readonly wasmmessageservice_get_message: (a: number, b: bigint) => any;
    readonly wasmmessageservice_get_messages: (a: number, b: number, c: number, d: number) => any;
    readonly wasmmessageservice_get_sent_messages: (a: number, b: number, c: number) => any;
    readonly wasmmessageservice_get_unread_count: (a: number) => any;
    readonly wasmmessageservice_mark_all_read: (a: number) => any;
    readonly wasmmessageservice_mark_read: (a: number, b: number, c: number) => any;
    readonly wasmmessageservice_replay_dead_letter: (a: number, b: bigint) => any;
    readonly wasmmessageservice_send_message: (a: number, b: number, c: number, d: number, e: number) => any;
    readonly wasmorgstate_add_member: (a: number, b: number, c: number) => void;
    readonly wasmorgstate_add_organization: (a: number, b: number, c: number) => void;
    readonly wasmorgstate_current_org_json: (a: number) => any;
    readonly wasmorgstate_members_json: (a: number) => [number, number];
    readonly wasmorgstate_new: () => number;
    readonly wasmorgstate_organizations_json: (a: number) => [number, number];
    readonly wasmorgstate_remove_member: (a: number, b: number, c: number) => void;
    readonly wasmorgstate_remove_organization: (a: number, b: number) => void;
    readonly wasmorgstate_set_current_org: (a: number, b: number, c: number) => void;
    readonly wasmorgstate_set_members: (a: number, b: number, c: number) => void;
    readonly wasmorgstate_set_organizations: (a: number, b: number, c: number) => void;
    readonly wasmorgstate_update_member: (a: number, b: number, c: number, d: number) => void;
    readonly wasmorgstate_update_organization: (a: number, b: number, c: number, d: number) => void;
    readonly wasmpromocodeservice_get_history: (a: number) => any;
    readonly wasmpromocodeservice_redeem: (a: number, b: number, c: number) => any;
    readonly wasmpromocodeservice_validate: (a: number, b: number, c: number) => any;
    readonly wasmticketstate_add_label: (a: number, b: number, c: number) => void;
    readonly wasmticketstate_add_ticket: (a: number, b: number, c: number) => void;
    readonly wasmticketstate_append_column_tickets: (a: number, b: number, c: number, d: number, e: number) => void;
    readonly wasmticketstate_board_columns_json: (a: number) => [number, number];
    readonly wasmticketstate_current_ticket_json: (a: number) => any;
    readonly wasmticketstate_filter_tickets_json: (a: number, b: number, c: number, d: number, e: number, f: number, g: number, h: number, i: number) => [number, number];
    readonly wasmticketstate_get_ticket_by_slug_json: (a: number, b: number, c: number) => any;
    readonly wasmticketstate_labels_json: (a: number) => [number, number];
    readonly wasmticketstate_new: () => number;
    readonly wasmticketstate_remove_label: (a: number, b: number) => void;
    readonly wasmticketstate_remove_ticket: (a: number, b: number, c: number) => void;
    readonly wasmticketstate_set_board_columns: (a: number, b: number, c: number) => void;
    readonly wasmticketstate_set_current_ticket: (a: number, b: number, c: number) => void;
    readonly wasmticketstate_set_labels: (a: number, b: number, c: number) => void;
    readonly wasmticketstate_set_tickets: (a: number, b: number, c: number) => void;
    readonly wasmticketstate_tickets_json: (a: number) => [number, number];
    readonly wasmticketstate_update_ticket: (a: number, b: number, c: number, d: number, e: number) => void;
    readonly wasmticketstate_update_ticket_status: (a: number, b: number, c: number, d: number, e: number) => void;
    readonly __wbg_wasmappstate_free: (a: number, b: number) => void;
    readonly __wbg_wasminvitationservice_free: (a: number, b: number) => void;
    readonly __wbg_wasmmeshservice_free: (a: number, b: number) => void;
    readonly __wbg_wasmmeshstate_free: (a: number, b: number) => void;
    readonly __wbg_wasmpodstate_free: (a: number, b: number) => void;
    readonly __wbg_wasmrepositoryservice_free: (a: number, b: number) => void;
    readonly __wbg_wasmrepostate_free: (a: number, b: number) => void;
    readonly __wbg_wasmssoservice_free: (a: number, b: number) => void;
    readonly __wbg_wasmusercredentialservice_free: (a: number, b: number) => void;
    readonly relay_decode_message: (a: number, b: number) => any;
    readonly relay_encode_acp_command: (a: number, b: number) => [number, number];
    readonly relay_encode_control: (a: number, b: number) => [number, number];
    readonly relay_encode_input: (a: number, b: number) => [number, number];
    readonly relay_encode_ping: () => [number, number];
    readonly relay_encode_resize: (a: number, b: number) => [number, number];
    readonly relay_encode_resync: () => [number, number];
    readonly wasmappstate_channels_json: (a: number) => [number, number];
    readonly wasmappstate_dispatch_event: (a: number, b: number, c: number) => void;
    readonly wasmappstate_loops_json: (a: number) => [number, number];
    readonly wasmappstate_mesh_json: (a: number) => [number, number];
    readonly wasmappstate_new: () => number;
    readonly wasmappstate_pods_json: (a: number) => [number, number];
    readonly wasmappstate_runners_json: (a: number) => [number, number];
    readonly wasmappstate_tickets_json: (a: number) => [number, number];
    readonly wasminvitationservice_accept: (a: number, b: number, c: number) => any;
    readonly wasminvitationservice_create: (a: number, b: number, c: number) => any;
    readonly wasminvitationservice_get_by_token: (a: number, b: number, c: number) => any;
    readonly wasminvitationservice_list: (a: number) => any;
    readonly wasminvitationservice_list_pending: (a: number) => any;
    readonly wasminvitationservice_resend: (a: number, b: bigint) => any;
    readonly wasminvitationservice_revoke: (a: number, b: bigint) => any;
    readonly wasmmeshservice_clear_topology: (a: number) => void;
    readonly wasmmeshservice_fetch_topology: (a: number) => any;
    readonly wasmmeshservice_get_active_nodes_json: (a: number) => [number, number];
    readonly wasmmeshservice_get_channels_for_node_json: (a: number, b: number, c: number) => [number, number];
    readonly wasmmeshservice_get_edges_for_node_json: (a: number, b: number, c: number) => [number, number];
    readonly wasmmeshservice_get_node_json: (a: number, b: number, c: number) => any;
    readonly wasmmeshservice_get_nodes_by_runner_json: (a: number, b: bigint) => [number, number];
    readonly wasmmeshservice_get_runner_info_json: (a: number, b: bigint) => any;
    readonly wasmmeshservice_select_node: (a: number, b: number, c: number) => void;
    readonly wasmmeshservice_selected_node: (a: number) => any;
    readonly wasmmeshservice_set_topology: (a: number, b: number, c: number) => void;
    readonly wasmmeshservice_topology_json: (a: number) => any;
    readonly wasmmeshstate_clear_topology: (a: number) => void;
    readonly wasmmeshstate_get_active_nodes_json: (a: number) => [number, number];
    readonly wasmmeshstate_get_channels_for_node_json: (a: number, b: number, c: number) => [number, number];
    readonly wasmmeshstate_get_edges_for_node_json: (a: number, b: number, c: number) => [number, number];
    readonly wasmmeshstate_get_node_json: (a: number, b: number, c: number) => any;
    readonly wasmmeshstate_get_nodes_by_runner_json: (a: number, b: bigint) => [number, number];
    readonly wasmmeshstate_get_runner_info_json: (a: number, b: bigint) => any;
    readonly wasmmeshstate_new: () => number;
    readonly wasmmeshstate_select_node: (a: number, b: number, c: number) => void;
    readonly wasmmeshstate_selected_node: (a: number) => any;
    readonly wasmmeshstate_set_topology: (a: number, b: number, c: number) => void;
    readonly wasmmeshstate_topology_json: (a: number) => any;
    readonly wasmpodstate_current_pod_json: (a: number) => any;
    readonly wasmpodstate_get_pod_json: (a: number, b: number, c: number) => any;
    readonly wasmpodstate_new: () => number;
    readonly wasmpodstate_pods_json: (a: number) => [number, number];
    readonly wasmpodstate_remove_pod: (a: number, b: number, c: number) => void;
    readonly wasmpodstate_set_current_pod: (a: number, b: number, c: number) => void;
    readonly wasmpodstate_set_pods: (a: number, b: number, c: number) => void;
    readonly wasmpodstate_update_agent_status: (a: number, b: number, c: number, d: number, e: number) => void;
    readonly wasmpodstate_update_pod_alias: (a: number, b: number, c: number, d: number, e: number) => void;
    readonly wasmpodstate_update_pod_status: (a: number, b: number, c: number, d: number, e: number, f: number, g: number, h: number, i: number, j: number, k: number, l: number, m: bigint) => void;
    readonly wasmpodstate_update_pod_title: (a: number, b: number, c: number, d: number, e: number, f: number, g: bigint) => void;
    readonly wasmpodstate_upsert_pod: (a: number, b: number, c: number, d: number, e: bigint) => void;
    readonly wasmrepositoryservice_create: (a: number, b: number, c: number) => any;
    readonly wasmrepositoryservice_delete: (a: number, b: bigint) => any;
    readonly wasmrepositoryservice_delete_webhook: (a: number, b: bigint) => any;
    readonly wasmrepositoryservice_get: (a: number, b: bigint) => any;
    readonly wasmrepositoryservice_get_webhook_secret: (a: number, b: bigint) => any;
    readonly wasmrepositoryservice_get_webhook_status: (a: number, b: bigint) => any;
    readonly wasmrepositoryservice_list: (a: number) => any;
    readonly wasmrepositoryservice_list_branches: (a: number, b: bigint) => any;
    readonly wasmrepositoryservice_list_merge_requests: (a: number, b: bigint, c: number, d: number, e: number, f: number) => any;
    readonly wasmrepositoryservice_mark_webhook_configured: (a: number, b: bigint) => any;
    readonly wasmrepositoryservice_register_webhook: (a: number, b: bigint) => any;
    readonly wasmrepositoryservice_sync_branches: (a: number, b: bigint, c: number, d: number) => any;
    readonly wasmrepositoryservice_update: (a: number, b: bigint, c: number, d: number) => any;
    readonly wasmrepostate_add_repository: (a: number, b: number, c: number) => void;
    readonly wasmrepostate_branches_json: (a: number) => [number, number];
    readonly wasmrepostate_current_repo_json: (a: number) => any;
    readonly wasmrepostate_new: () => number;
    readonly wasmrepostate_remove_repository: (a: number, b: number, c: number) => void;
    readonly wasmrepostate_repositories_json: (a: number) => [number, number];
    readonly wasmrepostate_set_branches: (a: number, b: number, c: number) => void;
    readonly wasmrepostate_set_current_repo: (a: number, b: number, c: number) => void;
    readonly wasmrepostate_set_repositories: (a: number, b: number, c: number) => void;
    readonly wasmrepostate_update_repository: (a: number, b: number, c: number, d: number, e: number) => void;
    readonly wasmssoservice_discover: (a: number, b: number, c: number) => any;
    readonly wasmssoservice_ldap_auth: (a: number, b: number, c: number, d: number, e: number) => any;
    readonly wasmusercredentialservice_clear_default_git_credential: (a: number) => any;
    readonly wasmusercredentialservice_create_agent_credential: (a: number, b: number, c: number, d: number, e: number) => any;
    readonly wasmusercredentialservice_create_git_credential: (a: number, b: number, c: number) => any;
    readonly wasmusercredentialservice_create_repo_provider: (a: number, b: number, c: number) => any;
    readonly wasmusercredentialservice_delete_agent_credential: (a: number, b: bigint) => any;
    readonly wasmusercredentialservice_delete_git_credential: (a: number, b: bigint) => any;
    readonly wasmusercredentialservice_delete_repo_provider: (a: number, b: bigint) => any;
    readonly wasmusercredentialservice_get_agent_credential: (a: number, b: bigint) => any;
    readonly wasmusercredentialservice_get_default_git_credential: (a: number) => any;
    readonly wasmusercredentialservice_get_git_credential: (a: number, b: bigint) => any;
    readonly wasmusercredentialservice_get_repo_provider: (a: number, b: bigint) => any;
    readonly wasmusercredentialservice_list_agent_credentials: (a: number) => any;
    readonly wasmusercredentialservice_list_agent_credentials_for_agent: (a: number, b: number, c: number) => any;
    readonly wasmusercredentialservice_list_git_credentials: (a: number) => any;
    readonly wasmusercredentialservice_list_provider_repositories: (a: number, b: bigint, c: number, d: number, e: number, f: number) => any;
    readonly wasmusercredentialservice_list_repo_providers: (a: number) => any;
    readonly wasmusercredentialservice_set_default_agent_credential: (a: number, b: bigint) => any;
    readonly wasmusercredentialservice_set_default_git_credential: (a: number, b: number, c: number) => any;
    readonly wasmusercredentialservice_set_default_repo_provider: (a: number, b: bigint) => any;
    readonly wasmusercredentialservice_test_repo_provider: (a: number, b: bigint) => any;
    readonly wasmusercredentialservice_update_agent_credential: (a: number, b: bigint, c: number, d: number) => any;
    readonly wasmusercredentialservice_update_git_credential: (a: number, b: bigint, c: number, d: number) => any;
    readonly wasmusercredentialservice_update_repo_provider: (a: number, b: bigint, c: number, d: number) => any;
    readonly __wbg_wasmauthapiservice_free: (a: number, b: number) => void;
    readonly __wbg_wasmauthmanager_free: (a: number, b: number) => void;
    readonly __wbg_wasmchannelstate_free: (a: number, b: number) => void;
    readonly __wbg_wasmeventsmanager_free: (a: number, b: number) => void;
    readonly __wbg_wasmloopstate_free: (a: number, b: number) => void;
    readonly __wbg_wasmrelaymanager_free: (a: number, b: number) => void;
    readonly __wbg_wasmuserstate_free: (a: number, b: number) => void;
    readonly wasmauthapiservice_forgot_password: (a: number, b: number, c: number) => any;
    readonly wasmauthapiservice_register: (a: number, b: number, c: number) => any;
    readonly wasmauthapiservice_resend_verification: (a: number, b: number, c: number) => any;
    readonly wasmauthapiservice_reset_password: (a: number, b: number, c: number) => any;
    readonly wasmauthapiservice_verify_email: (a: number, b: number, c: number) => any;
    readonly wasmauthmanager_apply_session: (a: number, b: number, c: number) => [number, number];
    readonly wasmauthmanager_base_url: (a: number) => [number, number];
    readonly wasmauthmanager_clear_session: (a: number) => void;
    readonly wasmauthmanager_fetch_organizations: (a: number) => any;
    readonly wasmauthmanager_get_current_org_json: (a: number) => any;
    readonly wasmauthmanager_get_current_user_json: (a: number) => any;
    readonly wasmauthmanager_get_organizations_json: (a: number) => [number, number];
    readonly wasmauthmanager_get_refresh_token: (a: number) => [number, number];
    readonly wasmauthmanager_get_token: (a: number) => [number, number];
    readonly wasmauthmanager_is_authenticated: (a: number) => number;
    readonly wasmauthmanager_login: (a: number, b: number, c: number, d: number, e: number) => any;
    readonly wasmauthmanager_logout: (a: number) => any;
    readonly wasmauthmanager_new: (a: number, b: number) => number;
    readonly wasmauthmanager_new_with_storage: (a: number, b: number, c: any) => number;
    readonly wasmauthmanager_refresh_token: (a: number) => any;
    readonly wasmauthmanager_restore_session: (a: number) => [number, number, number];
    readonly wasmauthmanager_set_current_org: (a: number, b: number, c: number) => [number, number];
    readonly wasmauthmanager_set_organizations: (a: number, b: number, c: number) => [number, number];
    readonly wasmauthmanager_switch_org: (a: number, b: number, c: number) => [number, number];
    readonly wasmchannelstate_add_channel: (a: number, b: number, c: number) => void;
    readonly wasmchannelstate_add_message: (a: number, b: bigint, c: number, d: number) => void;
    readonly wasmchannelstate_channels_json: (a: number) => [number, number];
    readonly wasmchannelstate_clear_channel_mentions: (a: number, b: bigint) => void;
    readonly wasmchannelstate_clear_channel_unread: (a: number, b: bigint) => void;
    readonly wasmchannelstate_current_channel_json: (a: number) => any;
    readonly wasmchannelstate_filter_channels_json: (a: number, b: number, c: number, d: number) => [number, number];
    readonly wasmchannelstate_get_channel_json: (a: number, b: bigint) => any;
    readonly wasmchannelstate_get_last_message_json: (a: number, b: bigint) => any;
    readonly wasmchannelstate_get_mention_count: (a: number, b: bigint) => number;
    readonly wasmchannelstate_get_messages_json: (a: number, b: bigint) => any;
    readonly wasmchannelstate_get_unread_count: (a: number, b: bigint) => number;
    readonly wasmchannelstate_increment_mention: (a: number, b: bigint) => void;
    readonly wasmchannelstate_increment_unread: (a: number, b: bigint) => void;
    readonly wasmchannelstate_mention_counts_json: (a: number) => [number, number];
    readonly wasmchannelstate_new: () => number;
    readonly wasmchannelstate_on_new_message: (a: number, b: number, c: number) => number;
    readonly wasmchannelstate_prepend_messages: (a: number, b: bigint, c: number, d: number, e: number) => void;
    readonly wasmchannelstate_remove_channel: (a: number, b: bigint) => void;
    readonly wasmchannelstate_remove_message: (a: number, b: bigint, c: bigint) => void;
    readonly wasmchannelstate_select_channel: (a: number, b: number, c: bigint) => any;
    readonly wasmchannelstate_set_channels: (a: number, b: number, c: number) => void;
    readonly wasmchannelstate_set_current_channel: (a: number, b: number, c: bigint) => void;
    readonly wasmchannelstate_set_current_user: (a: number, b: number, c: number) => void;
    readonly wasmchannelstate_set_current_user_id: (a: number, b: number, c: bigint) => void;
    readonly wasmchannelstate_set_last_message: (a: number, b: bigint, c: number, d: number) => void;
    readonly wasmchannelstate_set_mention_counts: (a: number, b: number, c: number) => void;
    readonly wasmchannelstate_set_messages: (a: number, b: bigint, c: number, d: number, e: number) => void;
    readonly wasmchannelstate_set_unread_counts: (a: number, b: number, c: number) => void;
    readonly wasmchannelstate_sorted_channel_ids_json: (a: number, b: number, c: number, d: number) => [number, number];
    readonly wasmchannelstate_total_mention_count: (a: number) => number;
    readonly wasmchannelstate_total_unread_count: (a: number) => number;
    readonly wasmchannelstate_unread_counts_json: (a: number) => [number, number];
    readonly wasmchannelstate_update_channel: (a: number, b: bigint, c: number, d: number) => void;
    readonly wasmchannelstate_update_message: (a: number, b: bigint, c: number, d: number) => void;
    readonly wasmeventsmanager_connect: (a: number) => any;
    readonly wasmeventsmanager_disconnect: (a: number) => any;
    readonly wasmeventsmanager_get_connection_state: (a: number) => any;
    readonly wasmeventsmanager_new: (a: number, b: number) => number;
    readonly wasmeventsmanager_new_with_options: (a: number, b: number, c: number, d: number, e: number, f: number, g: number) => number;
    readonly wasmeventsmanager_on_connection_state_change: (a: number, b: any) => any;
    readonly wasmeventsmanager_subscribe: (a: number, b: number, c: number, d: any) => any;
    readonly wasmeventsmanager_subscribe_all: (a: number, b: any) => any;
    readonly wasmeventsmanager_unsubscribe: (a: number, b: number) => any;
    readonly wasmloopstate_add_run: (a: number, b: number, c: number) => void;
    readonly wasmloopstate_append_runs: (a: number, b: number, c: number) => void;
    readonly wasmloopstate_clear_runs: (a: number) => void;
    readonly wasmloopstate_current_loop_json: (a: number) => any;
    readonly wasmloopstate_get_loop_by_slug_json: (a: number, b: number, c: number) => any;
    readonly wasmloopstate_loops_json: (a: number) => [number, number];
    readonly wasmloopstate_new: () => number;
    readonly wasmloopstate_runs_json: (a: number) => [number, number];
    readonly wasmloopstate_set_current_loop: (a: number, b: number, c: number) => void;
    readonly wasmloopstate_set_loops: (a: number, b: number, c: number) => void;
    readonly wasmloopstate_set_runs: (a: number, b: number, c: number) => void;
    readonly wasmloopstate_update_loop: (a: number, b: number, c: number, d: number, e: number) => void;
    readonly wasmloopstate_update_run_status: (a: number, b: bigint, c: number, d: number) => void;
    readonly wasmrelaymanager_disconnect: (a: number, b: number, c: number) => any;
    readonly wasmrelaymanager_disconnect_all: (a: number) => any;
    readonly wasmrelaymanager_force_resize: (a: number, b: number, c: number, d: number, e: number) => any;
    readonly wasmrelaymanager_get_pod_size: (a: number, b: number, c: number) => any;
    readonly wasmrelaymanager_get_status: (a: number, b: number, c: number) => any;
    readonly wasmrelaymanager_is_runner_disconnected: (a: number, b: number, c: number) => any;
    readonly wasmrelaymanager_new: () => number;
    readonly wasmrelaymanager_on_acp_message: (a: number, b: number, c: number, d: any) => any;
    readonly wasmrelaymanager_on_status_change: (a: number, b: number, c: number, d: any) => any;
    readonly wasmrelaymanager_send: (a: number, b: number, c: number, d: number, e: number) => any;
    readonly wasmrelaymanager_send_acp_command: (a: number, b: number, c: number, d: number, e: number) => any;
    readonly wasmrelaymanager_send_resize: (a: number, b: number, c: number, d: number, e: number) => any;
    readonly wasmrelaymanager_subscribe: (a: number, b: number, c: number, d: number, e: number, f: number, g: number, h: number, i: number, j: any) => any;
    readonly wasmrelaymanager_unsubscribe: (a: number, b: number, c: number, d: number, e: number) => any;
    readonly wasmuserstate_add_identity: (a: number, b: number, c: number) => void;
    readonly wasmuserstate_identities_json: (a: number) => [number, number];
    readonly wasmuserstate_new: () => number;
    readonly wasmuserstate_profile_json: (a: number) => any;
    readonly wasmuserstate_remove_identity: (a: number, b: number, c: number) => void;
    readonly wasmuserstate_set_profile: (a: number, b: number, c: number) => void;
    readonly __wbg_wasmautopilotservice_free: (a: number, b: number) => void;
    readonly __wbg_wasmfileservice_free: (a: number, b: number) => void;
    readonly __wbg_wasmorgapiservice_free: (a: number, b: number) => void;
    readonly __wbg_wasmpodservice_free: (a: number, b: number) => void;
    readonly __wbg_wasmrunnerservice_free: (a: number, b: number) => void;
    readonly wasmautopilotservice_add_controller: (a: number, b: number, c: number) => void;
    readonly wasmautopilotservice_add_iteration: (a: number, b: number, c: number, d: number, e: number) => void;
    readonly wasmautopilotservice_approve_controller: (a: number, b: number, c: number, d: number, e: number) => any;
    readonly wasmautopilotservice_controllers_json: (a: number) => [number, number];
    readonly wasmautopilotservice_create_controller: (a: number, b: number, c: number) => any;
    readonly wasmautopilotservice_current_controller_json: (a: number) => any;
    readonly wasmautopilotservice_fetch_controller: (a: number, b: number, c: number) => any;
    readonly wasmautopilotservice_fetch_controllers: (a: number) => any;
    readonly wasmautopilotservice_fetch_iterations: (a: number, b: number, c: number) => any;
    readonly wasmautopilotservice_get_controller_by_pod_key_json: (a: number, b: number, c: number) => any;
    readonly wasmautopilotservice_get_iterations_json: (a: number, b: number, c: number) => any;
    readonly wasmautopilotservice_get_thinking_history_json: (a: number, b: number, c: number) => any;
    readonly wasmautopilotservice_get_thinking_json: (a: number, b: number, c: number) => any;
    readonly wasmautopilotservice_handback_controller: (a: number, b: number, c: number) => any;
    readonly wasmautopilotservice_pause_controller: (a: number, b: number, c: number) => any;
    readonly wasmautopilotservice_remove_controller: (a: number, b: number, c: number) => void;
    readonly wasmautopilotservice_resume_controller: (a: number, b: number, c: number) => any;
    readonly wasmautopilotservice_set_controllers: (a: number, b: number, c: number) => void;
    readonly wasmautopilotservice_set_current_controller: (a: number, b: number, c: number) => void;
    readonly wasmautopilotservice_set_iterations: (a: number, b: number, c: number, d: number, e: number) => void;
    readonly wasmautopilotservice_stop_controller: (a: number, b: number, c: number) => any;
    readonly wasmautopilotservice_takeover_controller: (a: number, b: number, c: number) => any;
    readonly wasmautopilotservice_update_controller: (a: number, b: number, c: number, d: number, e: number) => void;
    readonly wasmautopilotservice_update_thinking: (a: number, b: number, c: number, d: number, e: number) => void;
    readonly wasmfileservice_presign_upload: (a: number, b: number, c: number) => any;
    readonly wasmfileservice_upload_file: (a: number, b: any, c: number, d: number, e: number, f: number) => any;
    readonly wasmgrantservice_grant: (a: number, b: number, c: number, d: number, e: number, f: bigint) => any;
    readonly wasmgrantservice_list: (a: number, b: number, c: number, d: number, e: number) => any;
    readonly wasmgrantservice_revoke: (a: number, b: number, c: number, d: number, e: number, f: bigint) => any;
    readonly wasmorgapiservice_create: (a: number, b: number, c: number) => any;
    readonly wasmorgapiservice_delete: (a: number, b: number, c: number) => any;
    readonly wasmorgapiservice_get: (a: number, b: number, c: number) => any;
    readonly wasmorgapiservice_invite_member: (a: number, b: number, c: number, d: number, e: number) => any;
    readonly wasmorgapiservice_list: (a: number) => any;
    readonly wasmorgapiservice_list_members: (a: number, b: number, c: number) => any;
    readonly wasmorgapiservice_remove_member: (a: number, b: number, c: number, d: bigint) => any;
    readonly wasmorgapiservice_update: (a: number, b: number, c: number, d: number, e: number) => any;
    readonly wasmorgapiservice_update_member_role: (a: number, b: number, c: number, d: bigint, e: number, f: number) => any;
    readonly wasmpodservice_create_pod: (a: number, b: number, c: number) => any;
    readonly wasmpodservice_current_pod_json: (a: number) => any;
    readonly wasmpodservice_fetch_pod: (a: number, b: number, c: number) => any;
    readonly wasmpodservice_fetch_pods: (a: number, b: number, c: number, d: number, e: bigint, f: number, g: bigint, h: number, i: bigint, j: number, k: bigint) => any;
    readonly wasmpodservice_fetch_sidebar_pods: (a: number, b: number, c: number, d: number, e: bigint) => any;
    readonly wasmpodservice_get_pod_connection: (a: number, b: number, c: number) => any;
    readonly wasmpodservice_get_pod_json: (a: number, b: number, c: number) => any;
    readonly wasmpodservice_load_more_pods: (a: number, b: number, c: number, d: number, e: bigint, f: bigint) => any;
    readonly wasmpodservice_pods_json: (a: number) => [number, number];
    readonly wasmpodservice_remove_pod: (a: number, b: number, c: number) => void;
    readonly wasmpodservice_set_current_pod: (a: number, b: number, c: number) => void;
    readonly wasmpodservice_set_pods: (a: number, b: number, c: number) => void;
    readonly wasmpodservice_terminate_pod: (a: number, b: number, c: number) => any;
    readonly wasmpodservice_update_agent_status: (a: number, b: number, c: number, d: number, e: number) => void;
    readonly wasmpodservice_update_pod_alias: (a: number, b: number, c: number, d: number, e: number) => void;
    readonly wasmpodservice_update_pod_alias_api: (a: number, b: number, c: number, d: number, e: number) => any;
    readonly wasmpodservice_update_pod_status: (a: number, b: number, c: number, d: number, e: number, f: number, g: number, h: number, i: number, j: number, k: number, l: number, m: bigint) => void;
    readonly wasmpodservice_update_pod_title: (a: number, b: number, c: number, d: number, e: number, f: number, g: bigint) => void;
    readonly wasmpodservice_upsert_pod: (a: number, b: number, c: number, d: number, e: bigint) => void;
    readonly wasmrunnerservice_authorize_runner: (a: number, b: number, c: number) => any;
    readonly wasmrunnerservice_available_runners_json: (a: number) => [number, number];
    readonly wasmrunnerservice_create_token: (a: number, b: number, c: number) => any;
    readonly wasmrunnerservice_current_runner_json: (a: number) => any;
    readonly wasmrunnerservice_delete_runner: (a: number, b: bigint) => any;
    readonly wasmrunnerservice_delete_token: (a: number, b: bigint) => any;
    readonly wasmrunnerservice_fetch_available_runners: (a: number) => any;
    readonly wasmrunnerservice_fetch_runner: (a: number, b: bigint) => any;
    readonly wasmrunnerservice_fetch_runners: (a: number, b: number, c: number) => any;
    readonly wasmrunnerservice_fetch_tokens: (a: number) => any;
    readonly wasmrunnerservice_get_auth_status: (a: number, b: number, c: number) => any;
    readonly wasmrunnerservice_get_runner_json: (a: number, b: bigint) => any;
    readonly wasmrunnerservice_list_runner_logs: (a: number, b: bigint) => any;
    readonly wasmrunnerservice_list_runner_pods: (a: number, b: bigint, c: number, d: number, e: number, f: number) => any;
    readonly wasmrunnerservice_query_runner_sandboxes: (a: number, b: bigint, c: number, d: number) => any;
    readonly wasmrunnerservice_remove_runner_local: (a: number, b: bigint) => void;
    readonly wasmrunnerservice_request_log_upload: (a: number, b: bigint) => any;
    readonly wasmrunnerservice_runners_json: (a: number) => [number, number];
    readonly wasmrunnerservice_set_available_runners: (a: number, b: number, c: number) => void;
    readonly wasmrunnerservice_set_current_runner: (a: number, b: number, c: number) => void;
    readonly wasmrunnerservice_set_runners: (a: number, b: number, c: number) => void;
    readonly wasmrunnerservice_update_runner: (a: number, b: bigint, c: number, d: number) => any;
    readonly wasmrunnerservice_update_runner_local: (a: number, b: number, c: number, d: number) => void;
    readonly wasmrunnerservice_update_runner_status: (a: number, b: bigint, c: number, d: number) => void;
    readonly wasmrunnerservice_upgrade_runner: (a: number, b: bigint, c: number, d: number) => any;
    readonly wasmtokenusageservice_get_dashboard: (a: number, b: number, c: number, d: number, e: number, f: number, g: number, h: number, i: bigint, j: number, k: number, l: number, m: number) => any;
    readonly __wbg_wasmgrantservice_free: (a: number, b: number) => void;
    readonly __wbg_wasmtokenusageservice_free: (a: number, b: number) => void;
    readonly __wbg_wasmwebsocket_free: (a: number, b: number) => void;
    readonly version: () => [number, number];
    readonly wasmwebsocket_close: (a: number) => void;
    readonly wasmwebsocket_connect: (a: number, b: number, c: any, d: any, e: any, f: any) => [number, number, number];
    readonly wasmwebsocket_is_closed: (a: number) => number;
    readonly wasmwebsocket_is_open: (a: number) => number;
    readonly wasmwebsocket_send_binary: (a: number, b: number, c: number) => [number, number];
    readonly wasmwebsocket_send_text: (a: number, b: number, c: number) => [number, number];
    readonly init_panic_hook: () => void;
    readonly wasm_bindgen__convert__closures_____invoke__hb7777b724403c6c8: (a: number, b: number, c: any) => [number, number];
    readonly wasm_bindgen__convert__closures_____invoke__h234923932d6562e7: (a: number, b: number, c: any, d: any) => void;
    readonly wasm_bindgen__convert__closures_____invoke__h821db2d1ab133cd5: (a: number, b: number, c: any) => void;
    readonly wasm_bindgen__convert__closures_____invoke__h821db2d1ab133cd5_2: (a: number, b: number, c: any) => void;
    readonly wasm_bindgen__convert__closures_____invoke__h821db2d1ab133cd5_3: (a: number, b: number, c: any) => void;
    readonly wasm_bindgen__convert__closures_____invoke__hb409e0dd58c7af9a: (a: number, b: number) => void;
    readonly __wbindgen_malloc: (a: number, b: number) => number;
    readonly __wbindgen_realloc: (a: number, b: number, c: number, d: number) => number;
    readonly __wbindgen_exn_store: (a: number) => void;
    readonly __externref_table_alloc: () => number;
    readonly __wbindgen_externrefs: WebAssembly.Table;
    readonly __wbindgen_free: (a: number, b: number, c: number) => void;
    readonly __wbindgen_destroy_closure: (a: number, b: number) => void;
    readonly __externref_table_dealloc: (a: number) => void;
    readonly __wbindgen_start: () => void;
}

export type SyncInitInput = BufferSource | WebAssembly.Module;

/**
 * Instantiates the given `module`, which can either be bytes or
 * a precompiled `WebAssembly.Module`.
 *
 * @param {{ module: SyncInitInput }} module - Passing `SyncInitInput` directly is deprecated.
 *
 * @returns {InitOutput}
 */
export function initSync(module: { module: SyncInitInput } | SyncInitInput): InitOutput;

/**
 * If `module_or_path` is {RequestInfo} or {URL}, makes a request and
 * for everything else, calls `WebAssembly.instantiate` directly.
 *
 * @param {{ module_or_path: InitInput | Promise<InitInput> }} module_or_path - Passing `InitInput` directly is deprecated.
 *
 * @returns {Promise<InitOutput>}
 */
export default function __wbg_init (module_or_path?: { module_or_path: InitInput | Promise<InitInput> } | InitInput | Promise<InitInput>): Promise<InitOutput>;
