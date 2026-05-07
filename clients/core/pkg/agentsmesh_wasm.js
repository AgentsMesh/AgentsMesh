/* @ts-self-types="./agentsmesh_wasm.d.ts" */

export class WasmAcpSessionManager {
    __destroy_into_raw() {
        const ptr = this.__wbg_ptr;
        this.__wbg_ptr = 0;
        WasmAcpSessionManagerFinalization.unregister(this);
        return ptr;
    }
    free() {
        const ptr = this.__destroy_into_raw();
        wasm.__wbg_wasmacpsessionmanager_free(ptr, 0);
    }
    /**
     * @param {string} pod_key
     * @param {string} text
     * @param {string} role
     */
    add_content_chunk(pod_key, text, role) {
        const ptr0 = passStringToWasm0(pod_key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(text, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        const ptr2 = passStringToWasm0(role, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len2 = WASM_VECTOR_LEN;
        wasm.wasmacpsessionmanager_add_content_chunk(this.__wbg_ptr, ptr0, len0, ptr1, len1, ptr2, len2);
    }
    /**
     * @param {string} pod_key
     * @param {string} level
     * @param {string} message
     */
    add_log(pod_key, level, message) {
        const ptr0 = passStringToWasm0(pod_key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(level, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        const ptr2 = passStringToWasm0(message, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len2 = WASM_VECTOR_LEN;
        wasm.wasmacpsessionmanager_add_log(this.__wbg_ptr, ptr0, len0, ptr1, len1, ptr2, len2);
    }
    /**
     * @param {string} pod_key
     * @param {string} request_json
     */
    add_permission_request(pod_key, request_json) {
        const ptr0 = passStringToWasm0(pod_key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(request_json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        wasm.wasmacpsessionmanager_add_permission_request(this.__wbg_ptr, ptr0, len0, ptr1, len1);
    }
    /**
     * @param {string} pod_key
     * @param {string} text
     */
    add_thinking(pod_key, text) {
        const ptr0 = passStringToWasm0(pod_key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(text, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        wasm.wasmacpsessionmanager_add_thinking(this.__wbg_ptr, ptr0, len0, ptr1, len1);
    }
    /**
     * @param {string} pod_key
     */
    clear_session(pod_key) {
        const ptr0 = passStringToWasm0(pod_key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmacpsessionmanager_clear_session(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {string} pod_key
     * @returns {any}
     */
    get_session_json(pod_key) {
        const ptr0 = passStringToWasm0(pod_key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmacpsessionmanager_get_session_json(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {string} pod_key
     */
    mark_last_message_complete(pod_key) {
        const ptr0 = passStringToWasm0(pod_key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmacpsessionmanager_mark_last_message_complete(this.__wbg_ptr, ptr0, len0);
    }
    constructor() {
        const ret = wasm.wasmacpsessionmanager_new();
        this.__wbg_ptr = ret >>> 0;
        WasmAcpSessionManagerFinalization.register(this, this.__wbg_ptr, this);
        return this;
    }
    /**
     * @param {string} pod_key
     * @param {string} request_id
     */
    remove_permission_request(pod_key, request_id) {
        const ptr0 = passStringToWasm0(pod_key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(request_id, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        wasm.wasmacpsessionmanager_remove_permission_request(this.__wbg_ptr, ptr0, len0, ptr1, len1);
    }
    /**
     * @param {string} pod_key
     * @param {string} tool_call_id
     * @param {boolean} success
     * @param {string | null} [result_text]
     * @param {string | null} [error_message]
     */
    set_tool_call_result(pod_key, tool_call_id, success, result_text, error_message) {
        const ptr0 = passStringToWasm0(pod_key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(tool_call_id, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        var ptr2 = isLikeNone(result_text) ? 0 : passStringToWasm0(result_text, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        var len2 = WASM_VECTOR_LEN;
        var ptr3 = isLikeNone(error_message) ? 0 : passStringToWasm0(error_message, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        var len3 = WASM_VECTOR_LEN;
        wasm.wasmacpsessionmanager_set_tool_call_result(this.__wbg_ptr, ptr0, len0, ptr1, len1, success, ptr2, len2, ptr3, len3);
    }
    /**
     * @param {string} pod_key
     * @param {string} steps_json
     */
    update_plan(pod_key, steps_json) {
        const ptr0 = passStringToWasm0(pod_key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(steps_json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        wasm.wasmacpsessionmanager_update_plan(this.__wbg_ptr, ptr0, len0, ptr1, len1);
    }
    /**
     * @param {string} pod_key
     * @param {string} state_str
     */
    update_session_state(pod_key, state_str) {
        const ptr0 = passStringToWasm0(pod_key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(state_str, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        wasm.wasmacpsessionmanager_update_session_state(this.__wbg_ptr, ptr0, len0, ptr1, len1);
    }
    /**
     * @param {string} pod_key
     * @param {string} tool_call_json
     */
    update_tool_call(pod_key, tool_call_json) {
        const ptr0 = passStringToWasm0(pod_key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(tool_call_json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        wasm.wasmacpsessionmanager_update_tool_call(this.__wbg_ptr, ptr0, len0, ptr1, len1);
    }
}
if (Symbol.dispose) WasmAcpSessionManager.prototype[Symbol.dispose] = WasmAcpSessionManager.prototype.free;

export class WasmAgentService {
    static __wrap(ptr) {
        ptr = ptr >>> 0;
        const obj = Object.create(WasmAgentService.prototype);
        obj.__wbg_ptr = ptr;
        WasmAgentServiceFinalization.register(obj, obj.__wbg_ptr, obj);
        return obj;
    }
    __destroy_into_raw() {
        const ptr = this.__wbg_ptr;
        this.__wbg_ptr = 0;
        WasmAgentServiceFinalization.unregister(this);
        return ptr;
    }
    free() {
        const ptr = this.__destroy_into_raw();
        wasm.__wbg_wasmagentservice_free(ptr, 0);
    }
    /**
     * @param {string} json
     * @returns {Promise<string>}
     */
    create_provider(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmagentservice_create_provider(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {bigint} id
     * @returns {Promise<void>}
     */
    delete_provider(id) {
        const ret = wasm.wasmagentservice_delete_provider(this.__wbg_ptr, id);
        return ret;
    }
    /**
     * @param {string} agent_slug
     * @returns {Promise<void>}
     */
    delete_user_config(agent_slug) {
        const ptr0 = passStringToWasm0(agent_slug, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmagentservice_delete_user_config(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @returns {Promise<string>}
     */
    get_agentpod_settings() {
        const ret = wasm.wasmagentservice_get_agentpod_settings(this.__wbg_ptr);
        return ret;
    }
    /**
     * @param {string} agent_slug
     * @returns {Promise<string>}
     */
    get_config_schema(agent_slug) {
        const ptr0 = passStringToWasm0(agent_slug, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmagentservice_get_config_schema(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {string} agent_slug
     * @returns {Promise<string>}
     */
    get_user_config(agent_slug) {
        const ptr0 = passStringToWasm0(agent_slug, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmagentservice_get_user_config(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @returns {Promise<string>}
     */
    list_agents() {
        const ret = wasm.wasmagentservice_list_agents(this.__wbg_ptr);
        return ret;
    }
    /**
     * @returns {Promise<string>}
     */
    list_providers() {
        const ret = wasm.wasmagentservice_list_providers(this.__wbg_ptr);
        return ret;
    }
    /**
     * @returns {Promise<string>}
     */
    list_user_configs() {
        const ret = wasm.wasmagentservice_list_user_configs(this.__wbg_ptr);
        return ret;
    }
    /**
     * @param {bigint} id
     * @returns {Promise<void>}
     */
    set_default_provider(id) {
        const ret = wasm.wasmagentservice_set_default_provider(this.__wbg_ptr, id);
        return ret;
    }
    /**
     * @param {string} agent_slug
     * @param {string} json
     * @returns {Promise<string>}
     */
    set_user_config(agent_slug, json) {
        const ptr0 = passStringToWasm0(agent_slug, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        const ret = wasm.wasmagentservice_set_user_config(this.__wbg_ptr, ptr0, len0, ptr1, len1);
        return ret;
    }
    /**
     * @param {string} json
     * @returns {Promise<string>}
     */
    update_agentpod_settings(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmagentservice_update_agentpod_settings(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {bigint} id
     * @param {string} json
     * @returns {Promise<string>}
     */
    update_provider(id, json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmagentservice_update_provider(this.__wbg_ptr, id, ptr0, len0);
        return ret;
    }
}
if (Symbol.dispose) WasmAgentService.prototype[Symbol.dispose] = WasmAgentService.prototype.free;

export class WasmApiClient {
    __destroy_into_raw() {
        const ptr = this.__wbg_ptr;
        this.__wbg_ptr = 0;
        WasmApiClientFinalization.unregister(this);
        return ptr;
    }
    free() {
        const ptr = this.__destroy_into_raw();
        wasm.__wbg_wasmapiclient_free(ptr, 0);
    }
    /**
     * @returns {string}
     */
    get base_url() {
        let deferred1_0;
        let deferred1_1;
        try {
            const ret = wasm.wasmapiclient_base_url(this.__wbg_ptr);
            deferred1_0 = ret[0];
            deferred1_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred1_0, deferred1_1, 1);
        }
    }
    clear_auth() {
        wasm.wasmapiclient_clear_auth(this.__wbg_ptr);
    }
    /**
     * @returns {WasmAgentService}
     */
    create_agent_service() {
        const ret = wasm.wasmapiclient_create_agent_service(this.__wbg_ptr);
        return WasmAgentService.__wrap(ret);
    }
    /**
     * @returns {WasmApiKeyService}
     */
    create_apikey_service() {
        const ret = wasm.wasmapiclient_create_apikey_service(this.__wbg_ptr);
        return WasmApiKeyService.__wrap(ret);
    }
    /**
     * @returns {WasmAuthApiService}
     */
    create_auth_api_service() {
        const ret = wasm.wasmapiclient_create_auth_api_service(this.__wbg_ptr);
        return WasmAuthApiService.__wrap(ret);
    }
    /**
     * @returns {WasmAutopilotService}
     */
    create_autopilot_service() {
        const ret = wasm.wasmapiclient_create_autopilot_service(this.__wbg_ptr);
        return WasmAutopilotService.__wrap(ret);
    }
    /**
     * @returns {WasmBillingService}
     */
    create_billing_service() {
        const ret = wasm.wasmapiclient_create_billing_service(this.__wbg_ptr);
        return WasmBillingService.__wrap(ret);
    }
    /**
     * @returns {WasmBindingService}
     */
    create_binding_service() {
        const ret = wasm.wasmapiclient_create_binding_service(this.__wbg_ptr);
        return WasmBindingService.__wrap(ret);
    }
    /**
     * @returns {WasmBlockstoreService}
     */
    create_blockstore_service() {
        const ret = wasm.wasmapiclient_create_blockstore_service(this.__wbg_ptr);
        return WasmBlockstoreService.__wrap(ret);
    }
    /**
     * @returns {WasmChannelService}
     */
    create_channel_service() {
        const ret = wasm.wasmapiclient_create_channel_service(this.__wbg_ptr);
        return WasmChannelService.__wrap(ret);
    }
    /**
     * @returns {WasmExtensionService}
     */
    create_extension_service() {
        const ret = wasm.wasmapiclient_create_extension_service(this.__wbg_ptr);
        return WasmExtensionService.__wrap(ret);
    }
    /**
     * @returns {WasmFileService}
     */
    create_file_service() {
        const ret = wasm.wasmapiclient_create_file_service(this.__wbg_ptr);
        return WasmFileService.__wrap(ret);
    }
    /**
     * @returns {WasmGrantService}
     */
    create_grant_service() {
        const ret = wasm.wasmapiclient_create_grant_service(this.__wbg_ptr);
        return WasmGrantService.__wrap(ret);
    }
    /**
     * @returns {WasmInvitationService}
     */
    create_invitation_service() {
        const ret = wasm.wasmapiclient_create_invitation_service(this.__wbg_ptr);
        return WasmInvitationService.__wrap(ret);
    }
    /**
     * @returns {WasmLoopService}
     */
    create_loop_service() {
        const ret = wasm.wasmapiclient_create_loop_service(this.__wbg_ptr);
        return WasmLoopService.__wrap(ret);
    }
    /**
     * @returns {WasmMeshService}
     */
    create_mesh_service() {
        const ret = wasm.wasmapiclient_create_mesh_service(this.__wbg_ptr);
        return WasmMeshService.__wrap(ret);
    }
    /**
     * @returns {WasmMessageService}
     */
    create_message_service() {
        const ret = wasm.wasmapiclient_create_message_service(this.__wbg_ptr);
        return WasmMessageService.__wrap(ret);
    }
    /**
     * @returns {WasmNotificationService}
     */
    create_notification_service() {
        const ret = wasm.wasmapiclient_create_notification_service(this.__wbg_ptr);
        return WasmNotificationService.__wrap(ret);
    }
    /**
     * @returns {WasmOrgApiService}
     */
    create_org_api_service() {
        const ret = wasm.wasmapiclient_create_org_api_service(this.__wbg_ptr);
        return WasmOrgApiService.__wrap(ret);
    }
    /**
     * Create a WasmPodService that shares this client's ApiClient and auth.
     * @returns {WasmPodService}
     */
    create_pod_service() {
        const ret = wasm.wasmapiclient_create_pod_service(this.__wbg_ptr);
        return WasmPodService.__wrap(ret);
    }
    /**
     * @returns {WasmPromoCodeService}
     */
    create_promocode_service() {
        const ret = wasm.wasmapiclient_create_promocode_service(this.__wbg_ptr);
        return WasmPromoCodeService.__wrap(ret);
    }
    /**
     * @returns {WasmRepositoryService}
     */
    create_repository_service() {
        const ret = wasm.wasmapiclient_create_repository_service(this.__wbg_ptr);
        return WasmRepositoryService.__wrap(ret);
    }
    /**
     * @returns {WasmRunnerService}
     */
    create_runner_service() {
        const ret = wasm.wasmapiclient_create_runner_service(this.__wbg_ptr);
        return WasmRunnerService.__wrap(ret);
    }
    /**
     * @returns {WasmSSOService}
     */
    create_sso_service() {
        const ret = wasm.wasmapiclient_create_sso_service(this.__wbg_ptr);
        return WasmSSOService.__wrap(ret);
    }
    /**
     * @returns {WasmSupportTicketService}
     */
    create_support_ticket_service() {
        const ret = wasm.wasmapiclient_create_support_ticket_service(this.__wbg_ptr);
        return WasmSupportTicketService.__wrap(ret);
    }
    /**
     * @returns {WasmTicketRelationsService}
     */
    create_ticket_relations_service() {
        const ret = wasm.wasmapiclient_create_ticket_relations_service(this.__wbg_ptr);
        return WasmTicketRelationsService.__wrap(ret);
    }
    /**
     * @returns {WasmTicketService}
     */
    create_ticket_service() {
        const ret = wasm.wasmapiclient_create_ticket_service(this.__wbg_ptr);
        return WasmTicketService.__wrap(ret);
    }
    /**
     * @returns {WasmTokenUsageService}
     */
    create_token_usage_service() {
        const ret = wasm.wasmapiclient_create_token_usage_service(this.__wbg_ptr);
        return WasmTokenUsageService.__wrap(ret);
    }
    /**
     * @returns {WasmUserApiService}
     */
    create_user_api_service() {
        const ret = wasm.wasmapiclient_create_user_api_service(this.__wbg_ptr);
        return WasmUserApiService.__wrap(ret);
    }
    /**
     * @returns {WasmUserCredentialService}
     */
    create_user_credential_service() {
        const ret = wasm.wasmapiclient_create_user_credential_service(this.__wbg_ptr);
        return WasmUserCredentialService.__wrap(ret);
    }
    /**
     * @param {string} endpoint
     * @returns {Promise<string>}
     */
    delete(endpoint) {
        const ptr0 = passStringToWasm0(endpoint, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmapiclient_delete(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {string} endpoint
     * @returns {Promise<string>}
     */
    get(endpoint) {
        const ptr0 = passStringToWasm0(endpoint, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmapiclient_get(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @returns {string | undefined}
     */
    get_org_slug() {
        const ret = wasm.wasmapiclient_get_org_slug(this.__wbg_ptr);
        let v1;
        if (ret[0] !== 0) {
            v1 = getStringFromWasm0(ret[0], ret[1]).slice();
            wasm.__wbindgen_free(ret[0], ret[1] * 1, 1);
        }
        return v1;
    }
    /**
     * @returns {string | undefined}
     */
    get_token() {
        const ret = wasm.wasmapiclient_get_token(this.__wbg_ptr);
        let v1;
        if (ret[0] !== 0) {
            v1 = getStringFromWasm0(ret[0], ret[1]).slice();
            wasm.__wbindgen_free(ret[0], ret[1] * 1, 1);
        }
        return v1;
    }
    /**
     * @param {string} base_url
     */
    constructor(base_url) {
        const ptr0 = passStringToWasm0(base_url, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmapiclient_new(ptr0, len0);
        this.__wbg_ptr = ret >>> 0;
        WasmApiClientFinalization.register(this, this.__wbg_ptr, this);
        return this;
    }
    /**
     * @param {string} path
     * @returns {string}
     */
    org_path(path) {
        let deferred2_0;
        let deferred2_1;
        try {
            const ptr0 = passStringToWasm0(path, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
            const len0 = WASM_VECTOR_LEN;
            const ret = wasm.wasmapiclient_org_path(this.__wbg_ptr, ptr0, len0);
            deferred2_0 = ret[0];
            deferred2_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred2_0, deferred2_1, 1);
        }
    }
    /**
     * @param {string} endpoint
     * @param {string} body
     * @returns {Promise<string>}
     */
    patch(endpoint, body) {
        const ptr0 = passStringToWasm0(endpoint, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(body, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        const ret = wasm.wasmapiclient_patch(this.__wbg_ptr, ptr0, len0, ptr1, len1);
        return ret;
    }
    /**
     * @param {string} endpoint
     * @param {string} body
     * @returns {Promise<string>}
     */
    post(endpoint, body) {
        const ptr0 = passStringToWasm0(endpoint, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(body, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        const ret = wasm.wasmapiclient_post(this.__wbg_ptr, ptr0, len0, ptr1, len1);
        return ret;
    }
    /**
     * @param {string} endpoint
     * @returns {Promise<string>}
     */
    public_get(endpoint) {
        const ptr0 = passStringToWasm0(endpoint, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmapiclient_public_get(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {string} endpoint
     * @param {string} body
     * @returns {Promise<string>}
     */
    public_post(endpoint, body) {
        const ptr0 = passStringToWasm0(endpoint, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(body, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        const ret = wasm.wasmapiclient_public_post(this.__wbg_ptr, ptr0, len0, ptr1, len1);
        return ret;
    }
    /**
     * @param {string} endpoint
     * @param {string} body
     * @returns {Promise<string>}
     */
    put(endpoint, body) {
        const ptr0 = passStringToWasm0(endpoint, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(body, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        const ret = wasm.wasmapiclient_put(this.__wbg_ptr, ptr0, len0, ptr1, len1);
        return ret;
    }
    /**
     * @param {string} slug
     */
    set_org_slug(slug) {
        const ptr0 = passStringToWasm0(slug, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmapiclient_set_org_slug(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {string} token
     * @param {string} refresh_token
     */
    set_token(token, refresh_token) {
        const ptr0 = passStringToWasm0(token, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(refresh_token, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        wasm.wasmapiclient_set_token(this.__wbg_ptr, ptr0, len0, ptr1, len1);
    }
}
if (Symbol.dispose) WasmApiClient.prototype[Symbol.dispose] = WasmApiClient.prototype.free;

export class WasmApiKeyService {
    static __wrap(ptr) {
        ptr = ptr >>> 0;
        const obj = Object.create(WasmApiKeyService.prototype);
        obj.__wbg_ptr = ptr;
        WasmApiKeyServiceFinalization.register(obj, obj.__wbg_ptr, obj);
        return obj;
    }
    __destroy_into_raw() {
        const ptr = this.__wbg_ptr;
        this.__wbg_ptr = 0;
        WasmApiKeyServiceFinalization.unregister(this);
        return ptr;
    }
    free() {
        const ptr = this.__destroy_into_raw();
        wasm.__wbg_wasmapikeyservice_free(ptr, 0);
    }
    /**
     * @param {string} json
     * @returns {Promise<string>}
     */
    create(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmapikeyservice_create(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {bigint} id
     * @returns {Promise<void>}
     */
    delete(id) {
        const ret = wasm.wasmapikeyservice_delete(this.__wbg_ptr, id);
        return ret;
    }
    /**
     * @param {bigint} id
     * @returns {Promise<string>}
     */
    get(id) {
        const ret = wasm.wasmapikeyservice_get(this.__wbg_ptr, id);
        return ret;
    }
    /**
     * @returns {Promise<string>}
     */
    list() {
        const ret = wasm.wasmapikeyservice_list(this.__wbg_ptr);
        return ret;
    }
    /**
     * @param {bigint} id
     * @returns {Promise<void>}
     */
    revoke(id) {
        const ret = wasm.wasmapikeyservice_revoke(this.__wbg_ptr, id);
        return ret;
    }
    /**
     * @param {bigint} id
     * @param {string} json
     * @returns {Promise<string>}
     */
    update(id, json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmapikeyservice_update(this.__wbg_ptr, id, ptr0, len0);
        return ret;
    }
}
if (Symbol.dispose) WasmApiKeyService.prototype[Symbol.dispose] = WasmApiKeyService.prototype.free;

export class WasmAppState {
    __destroy_into_raw() {
        const ptr = this.__wbg_ptr;
        this.__wbg_ptr = 0;
        WasmAppStateFinalization.unregister(this);
        return ptr;
    }
    free() {
        const ptr = this.__destroy_into_raw();
        wasm.__wbg_wasmappstate_free(ptr, 0);
    }
    /**
     * @returns {string}
     */
    channels_json() {
        let deferred1_0;
        let deferred1_1;
        try {
            const ret = wasm.wasmappstate_channels_json(this.__wbg_ptr);
            deferred1_0 = ret[0];
            deferred1_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred1_0, deferred1_1, 1);
        }
    }
    /**
     * @param {string} event_json
     */
    dispatch_event(event_json) {
        const ptr0 = passStringToWasm0(event_json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmappstate_dispatch_event(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @returns {string}
     */
    loops_json() {
        let deferred1_0;
        let deferred1_1;
        try {
            const ret = wasm.wasmappstate_loops_json(this.__wbg_ptr);
            deferred1_0 = ret[0];
            deferred1_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred1_0, deferred1_1, 1);
        }
    }
    /**
     * @returns {string}
     */
    mesh_json() {
        let deferred1_0;
        let deferred1_1;
        try {
            const ret = wasm.wasmappstate_mesh_json(this.__wbg_ptr);
            deferred1_0 = ret[0];
            deferred1_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred1_0, deferred1_1, 1);
        }
    }
    constructor() {
        const ret = wasm.wasmappstate_new();
        this.__wbg_ptr = ret >>> 0;
        WasmAppStateFinalization.register(this, this.__wbg_ptr, this);
        return this;
    }
    /**
     * @returns {string}
     */
    pods_json() {
        let deferred1_0;
        let deferred1_1;
        try {
            const ret = wasm.wasmappstate_pods_json(this.__wbg_ptr);
            deferred1_0 = ret[0];
            deferred1_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred1_0, deferred1_1, 1);
        }
    }
    /**
     * @returns {string}
     */
    runners_json() {
        let deferred1_0;
        let deferred1_1;
        try {
            const ret = wasm.wasmappstate_runners_json(this.__wbg_ptr);
            deferred1_0 = ret[0];
            deferred1_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred1_0, deferred1_1, 1);
        }
    }
    /**
     * @returns {string}
     */
    tickets_json() {
        let deferred1_0;
        let deferred1_1;
        try {
            const ret = wasm.wasmappstate_tickets_json(this.__wbg_ptr);
            deferred1_0 = ret[0];
            deferred1_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred1_0, deferred1_1, 1);
        }
    }
}
if (Symbol.dispose) WasmAppState.prototype[Symbol.dispose] = WasmAppState.prototype.free;

export class WasmAuthApiService {
    static __wrap(ptr) {
        ptr = ptr >>> 0;
        const obj = Object.create(WasmAuthApiService.prototype);
        obj.__wbg_ptr = ptr;
        WasmAuthApiServiceFinalization.register(obj, obj.__wbg_ptr, obj);
        return obj;
    }
    __destroy_into_raw() {
        const ptr = this.__wbg_ptr;
        this.__wbg_ptr = 0;
        WasmAuthApiServiceFinalization.unregister(this);
        return ptr;
    }
    free() {
        const ptr = this.__destroy_into_raw();
        wasm.__wbg_wasmauthapiservice_free(ptr, 0);
    }
    /**
     * @param {string} email
     * @returns {Promise<string>}
     */
    forgot_password(email) {
        const ptr0 = passStringToWasm0(email, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmauthapiservice_forgot_password(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {string} json
     * @returns {Promise<string>}
     */
    register(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmauthapiservice_register(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {string} email
     * @returns {Promise<string>}
     */
    resend_verification(email) {
        const ptr0 = passStringToWasm0(email, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmauthapiservice_resend_verification(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {string} json
     * @returns {Promise<string>}
     */
    reset_password(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmauthapiservice_reset_password(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {string} token
     * @returns {Promise<string>}
     */
    verify_email(token) {
        const ptr0 = passStringToWasm0(token, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmauthapiservice_verify_email(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
}
if (Symbol.dispose) WasmAuthApiService.prototype[Symbol.dispose] = WasmAuthApiService.prototype.free;

export class WasmAuthManager {
    static __wrap(ptr) {
        ptr = ptr >>> 0;
        const obj = Object.create(WasmAuthManager.prototype);
        obj.__wbg_ptr = ptr;
        WasmAuthManagerFinalization.register(obj, obj.__wbg_ptr, obj);
        return obj;
    }
    __destroy_into_raw() {
        const ptr = this.__wbg_ptr;
        this.__wbg_ptr = 0;
        WasmAuthManagerFinalization.unregister(this);
        return ptr;
    }
    free() {
        const ptr = this.__destroy_into_raw();
        wasm.__wbg_wasmauthmanager_free(ptr, 0);
    }
    /**
     * Apply an already-obtained AuthSession (SSO / register callback path).
     * Writes token + refresh_token + user into Rust AuthState and persists.
     * @param {string} session_json
     */
    apply_session(session_json) {
        const ptr0 = passStringToWasm0(session_json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmauthmanager_apply_session(this.__wbg_ptr, ptr0, len0);
        if (ret[1]) {
            throw takeFromExternrefTable0(ret[0]);
        }
    }
    /**
     * @returns {string}
     */
    get base_url() {
        let deferred1_0;
        let deferred1_1;
        try {
            const ret = wasm.wasmauthmanager_base_url(this.__wbg_ptr);
            deferred1_0 = ret[0];
            deferred1_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred1_0, deferred1_1, 1);
        }
    }
    /**
     * Clear all session data (logout without API call). Useful for test reset.
     */
    clear_session() {
        wasm.wasmauthmanager_clear_session(this.__wbg_ptr);
    }
    /**
     * @returns {Promise<string>}
     */
    fetch_organizations() {
        const ret = wasm.wasmauthmanager_fetch_organizations(this.__wbg_ptr);
        return ret;
    }
    /**
     * @returns {any}
     */
    get_current_org_json() {
        const ret = wasm.wasmauthmanager_get_current_org_json(this.__wbg_ptr);
        return ret;
    }
    /**
     * @returns {any}
     */
    get_current_user_json() {
        const ret = wasm.wasmauthmanager_get_current_user_json(this.__wbg_ptr);
        return ret;
    }
    /**
     * @returns {string}
     */
    get_organizations_json() {
        let deferred1_0;
        let deferred1_1;
        try {
            const ret = wasm.wasmauthmanager_get_organizations_json(this.__wbg_ptr);
            deferred1_0 = ret[0];
            deferred1_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred1_0, deferred1_1, 1);
        }
    }
    /**
     * @returns {string | undefined}
     */
    get_refresh_token() {
        const ret = wasm.wasmauthmanager_get_refresh_token(this.__wbg_ptr);
        let v1;
        if (ret[0] !== 0) {
            v1 = getStringFromWasm0(ret[0], ret[1]).slice();
            wasm.__wbindgen_free(ret[0], ret[1] * 1, 1);
        }
        return v1;
    }
    /**
     * @returns {string | undefined}
     */
    get_token() {
        const ret = wasm.wasmauthmanager_get_token(this.__wbg_ptr);
        let v1;
        if (ret[0] !== 0) {
            v1 = getStringFromWasm0(ret[0], ret[1]).slice();
            wasm.__wbindgen_free(ret[0], ret[1] * 1, 1);
        }
        return v1;
    }
    /**
     * @returns {boolean}
     */
    is_authenticated() {
        const ret = wasm.wasmauthmanager_is_authenticated(this.__wbg_ptr);
        return ret !== 0;
    }
    /**
     * @param {string} email
     * @param {string} password
     * @returns {Promise<string>}
     */
    login(email, password) {
        const ptr0 = passStringToWasm0(email, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(password, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        const ret = wasm.wasmauthmanager_login(this.__wbg_ptr, ptr0, len0, ptr1, len1);
        return ret;
    }
    /**
     * @returns {Promise<void>}
     */
    logout() {
        const ret = wasm.wasmauthmanager_logout(this.__wbg_ptr);
        return ret;
    }
    /**
     * @param {string} base_url
     */
    constructor(base_url) {
        const ptr0 = passStringToWasm0(base_url, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmauthmanager_new(ptr0, len0);
        this.__wbg_ptr = ret >>> 0;
        WasmAuthManagerFinalization.register(this, this.__wbg_ptr, this);
        return this;
    }
    /**
     * @param {string} base_url
     * @param {any} storage
     * @returns {WasmAuthManager}
     */
    static new_with_storage(base_url, storage) {
        const ptr0 = passStringToWasm0(base_url, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmauthmanager_new_with_storage(ptr0, len0, storage);
        return WasmAuthManager.__wrap(ret);
    }
    /**
     * @returns {Promise<string>}
     */
    refresh_token() {
        const ret = wasm.wasmauthmanager_refresh_token(this.__wbg_ptr);
        return ret;
    }
    /**
     * @returns {boolean}
     */
    restore_session() {
        const ret = wasm.wasmauthmanager_restore_session(this.__wbg_ptr);
        if (ret[2]) {
            throw takeFromExternrefTable0(ret[1]);
        }
        return ret[0] !== 0;
    }
    /**
     * Set or clear current organization. Empty json string clears it.
     * @param {string} org_json
     */
    set_current_org(org_json) {
        const ptr0 = passStringToWasm0(org_json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmauthmanager_set_current_org(this.__wbg_ptr, ptr0, len0);
        if (ret[1]) {
            throw takeFromExternrefTable0(ret[0]);
        }
    }
    /**
     * Replace the organizations list (e.g. after a refetch outside fetch_organizations).
     * Also promotes the first org to current_org if none is set.
     * @param {string} orgs_json
     */
    set_organizations(orgs_json) {
        const ptr0 = passStringToWasm0(orgs_json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmauthmanager_set_organizations(this.__wbg_ptr, ptr0, len0);
        if (ret[1]) {
            throw takeFromExternrefTable0(ret[0]);
        }
    }
    /**
     * @param {string} slug
     */
    switch_org(slug) {
        const ptr0 = passStringToWasm0(slug, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmauthmanager_switch_org(this.__wbg_ptr, ptr0, len0);
        if (ret[1]) {
            throw takeFromExternrefTable0(ret[0]);
        }
    }
}
if (Symbol.dispose) WasmAuthManager.prototype[Symbol.dispose] = WasmAuthManager.prototype.free;

export class WasmAutopilotService {
    static __wrap(ptr) {
        ptr = ptr >>> 0;
        const obj = Object.create(WasmAutopilotService.prototype);
        obj.__wbg_ptr = ptr;
        WasmAutopilotServiceFinalization.register(obj, obj.__wbg_ptr, obj);
        return obj;
    }
    __destroy_into_raw() {
        const ptr = this.__wbg_ptr;
        this.__wbg_ptr = 0;
        WasmAutopilotServiceFinalization.unregister(this);
        return ptr;
    }
    free() {
        const ptr = this.__destroy_into_raw();
        wasm.__wbg_wasmautopilotservice_free(ptr, 0);
    }
    /**
     * @param {string} json
     */
    add_controller(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmautopilotservice_add_controller(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {string} key
     * @param {string} json
     */
    add_iteration(key, json) {
        const ptr0 = passStringToWasm0(key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        wasm.wasmautopilotservice_add_iteration(this.__wbg_ptr, ptr0, len0, ptr1, len1);
    }
    /**
     * @param {string} key
     * @param {string} request_json
     * @returns {Promise<void>}
     */
    approve_controller(key, request_json) {
        const ptr0 = passStringToWasm0(key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(request_json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        const ret = wasm.wasmautopilotservice_approve_controller(this.__wbg_ptr, ptr0, len0, ptr1, len1);
        return ret;
    }
    /**
     * @returns {string}
     */
    controllers_json() {
        let deferred1_0;
        let deferred1_1;
        try {
            const ret = wasm.wasmautopilotservice_controllers_json(this.__wbg_ptr);
            deferred1_0 = ret[0];
            deferred1_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred1_0, deferred1_1, 1);
        }
    }
    /**
     * @param {string} request_json
     * @returns {Promise<string>}
     */
    create_controller(request_json) {
        const ptr0 = passStringToWasm0(request_json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmautopilotservice_create_controller(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @returns {any}
     */
    current_controller_json() {
        const ret = wasm.wasmautopilotservice_current_controller_json(this.__wbg_ptr);
        return ret;
    }
    /**
     * @param {string} key
     * @returns {Promise<string>}
     */
    fetch_controller(key) {
        const ptr0 = passStringToWasm0(key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmautopilotservice_fetch_controller(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @returns {Promise<string>}
     */
    fetch_controllers() {
        const ret = wasm.wasmautopilotservice_fetch_controllers(this.__wbg_ptr);
        return ret;
    }
    /**
     * @param {string} key
     * @returns {Promise<string>}
     */
    fetch_iterations(key) {
        const ptr0 = passStringToWasm0(key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmautopilotservice_fetch_iterations(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {string} pod_key
     * @returns {any}
     */
    get_controller_by_pod_key_json(pod_key) {
        const ptr0 = passStringToWasm0(pod_key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmautopilotservice_get_controller_by_pod_key_json(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {string} key
     * @returns {any}
     */
    get_iterations_json(key) {
        const ptr0 = passStringToWasm0(key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmautopilotservice_get_iterations_json(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {string} key
     * @returns {any}
     */
    get_thinking_history_json(key) {
        const ptr0 = passStringToWasm0(key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmautopilotservice_get_thinking_history_json(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {string} key
     * @returns {any}
     */
    get_thinking_json(key) {
        const ptr0 = passStringToWasm0(key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmautopilotservice_get_thinking_json(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {string} key
     * @returns {Promise<void>}
     */
    handback_controller(key) {
        const ptr0 = passStringToWasm0(key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmautopilotservice_handback_controller(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {string} key
     * @returns {Promise<void>}
     */
    pause_controller(key) {
        const ptr0 = passStringToWasm0(key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmautopilotservice_pause_controller(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {string} key
     */
    remove_controller(key) {
        const ptr0 = passStringToWasm0(key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmautopilotservice_remove_controller(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {string} key
     * @returns {Promise<void>}
     */
    resume_controller(key) {
        const ptr0 = passStringToWasm0(key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmautopilotservice_resume_controller(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {string} json
     */
    set_controllers(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmautopilotservice_set_controllers(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {string} json
     */
    set_current_controller(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmautopilotservice_set_current_controller(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {string} key
     * @param {string} json
     */
    set_iterations(key, json) {
        const ptr0 = passStringToWasm0(key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        wasm.wasmautopilotservice_set_iterations(this.__wbg_ptr, ptr0, len0, ptr1, len1);
    }
    /**
     * @param {string} key
     * @returns {Promise<void>}
     */
    stop_controller(key) {
        const ptr0 = passStringToWasm0(key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmautopilotservice_stop_controller(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {string} key
     * @returns {Promise<void>}
     */
    takeover_controller(key) {
        const ptr0 = passStringToWasm0(key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmautopilotservice_takeover_controller(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {string} key
     * @param {string} json
     */
    update_controller(key, json) {
        const ptr0 = passStringToWasm0(key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        wasm.wasmautopilotservice_update_controller(this.__wbg_ptr, ptr0, len0, ptr1, len1);
    }
    /**
     * @param {string} key
     * @param {string} json
     */
    update_thinking(key, json) {
        const ptr0 = passStringToWasm0(key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        wasm.wasmautopilotservice_update_thinking(this.__wbg_ptr, ptr0, len0, ptr1, len1);
    }
}
if (Symbol.dispose) WasmAutopilotService.prototype[Symbol.dispose] = WasmAutopilotService.prototype.free;

export class WasmAutopilotState {
    __destroy_into_raw() {
        const ptr = this.__wbg_ptr;
        this.__wbg_ptr = 0;
        WasmAutopilotStateFinalization.unregister(this);
        return ptr;
    }
    free() {
        const ptr = this.__destroy_into_raw();
        wasm.__wbg_wasmautopilotstate_free(ptr, 0);
    }
    /**
     * @param {string} json
     */
    add_controller(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmautopilotstate_add_controller(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {string} key
     * @param {string} json
     */
    add_iteration(key, json) {
        const ptr0 = passStringToWasm0(key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        wasm.wasmautopilotstate_add_iteration(this.__wbg_ptr, ptr0, len0, ptr1, len1);
    }
    /**
     * @returns {string}
     */
    controllers_json() {
        let deferred1_0;
        let deferred1_1;
        try {
            const ret = wasm.wasmautopilotstate_controllers_json(this.__wbg_ptr);
            deferred1_0 = ret[0];
            deferred1_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred1_0, deferred1_1, 1);
        }
    }
    /**
     * @returns {any}
     */
    current_controller_json() {
        const ret = wasm.wasmautopilotstate_current_controller_json(this.__wbg_ptr);
        return ret;
    }
    /**
     * @param {string} pod_key
     * @returns {any}
     */
    get_controller_by_pod_key_json(pod_key) {
        const ptr0 = passStringToWasm0(pod_key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmautopilotstate_get_controller_by_pod_key_json(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {string} key
     * @returns {any}
     */
    get_iterations_json(key) {
        const ptr0 = passStringToWasm0(key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmautopilotstate_get_iterations_json(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {string} key
     * @returns {any}
     */
    get_thinking_history_json(key) {
        const ptr0 = passStringToWasm0(key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmautopilotstate_get_thinking_history_json(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {string} key
     * @returns {any}
     */
    get_thinking_json(key) {
        const ptr0 = passStringToWasm0(key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmautopilotstate_get_thinking_json(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    constructor() {
        const ret = wasm.wasmautopilotstate_new();
        this.__wbg_ptr = ret >>> 0;
        WasmAutopilotStateFinalization.register(this, this.__wbg_ptr, this);
        return this;
    }
    /**
     * @param {string} key
     */
    remove_controller(key) {
        const ptr0 = passStringToWasm0(key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmautopilotstate_remove_controller(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {string} json
     */
    set_controllers(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmautopilotstate_set_controllers(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {string} json
     */
    set_current_controller(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmautopilotstate_set_current_controller(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {string} key
     * @param {string} json
     */
    set_iterations(key, json) {
        const ptr0 = passStringToWasm0(key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        wasm.wasmautopilotstate_set_iterations(this.__wbg_ptr, ptr0, len0, ptr1, len1);
    }
    /**
     * @param {string} key
     * @param {string} json
     */
    update_controller(key, json) {
        const ptr0 = passStringToWasm0(key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        wasm.wasmautopilotstate_update_controller(this.__wbg_ptr, ptr0, len0, ptr1, len1);
    }
    /**
     * @param {string} key
     * @param {string} json
     */
    update_thinking(key, json) {
        const ptr0 = passStringToWasm0(key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        wasm.wasmautopilotstate_update_thinking(this.__wbg_ptr, ptr0, len0, ptr1, len1);
    }
}
if (Symbol.dispose) WasmAutopilotState.prototype[Symbol.dispose] = WasmAutopilotState.prototype.free;

export class WasmBillingService {
    static __wrap(ptr) {
        ptr = ptr >>> 0;
        const obj = Object.create(WasmBillingService.prototype);
        obj.__wbg_ptr = ptr;
        WasmBillingServiceFinalization.register(obj, obj.__wbg_ptr, obj);
        return obj;
    }
    __destroy_into_raw() {
        const ptr = this.__wbg_ptr;
        this.__wbg_ptr = 0;
        WasmBillingServiceFinalization.unregister(this);
        return ptr;
    }
    free() {
        const ptr = this.__destroy_into_raw();
        wasm.__wbg_wasmbillingservice_free(ptr, 0);
    }
    /**
     * @returns {Promise<string>}
     */
    cancel_subscription() {
        const ret = wasm.wasmbillingservice_cancel_subscription(this.__wbg_ptr);
        return ret;
    }
    /**
     * @param {string} json
     * @returns {Promise<string>}
     */
    change_cycle(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmbillingservice_change_cycle(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {string} resource
     * @param {number | null} [amount]
     * @returns {Promise<string>}
     */
    check_quota(resource, amount) {
        const ptr0 = passStringToWasm0(resource, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmbillingservice_check_quota(this.__wbg_ptr, ptr0, len0, isLikeNone(amount) ? 0x100000001 : (amount) >>> 0);
        return ret;
    }
    /**
     * @param {string} json
     * @returns {Promise<string>}
     */
    create_checkout(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmbillingservice_create_checkout(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {string} json
     * @returns {Promise<string>}
     */
    create_subscription(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmbillingservice_create_subscription(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {string} order_no
     * @returns {Promise<string>}
     */
    get_checkout_status(order_no) {
        const ptr0 = passStringToWasm0(order_no, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmbillingservice_get_checkout_status(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {string} json
     * @returns {Promise<string>}
     */
    get_customer_portal(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmbillingservice_get_customer_portal(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @returns {Promise<string>}
     */
    get_deployment_info() {
        const ret = wasm.wasmbillingservice_get_deployment_info(this.__wbg_ptr);
        return ret;
    }
    /**
     * @returns {Promise<string>}
     */
    get_overview() {
        const ret = wasm.wasmbillingservice_get_overview(this.__wbg_ptr);
        return ret;
    }
    /**
     * @returns {Promise<string>}
     */
    get_public_deployment_info() {
        const ret = wasm.wasmbillingservice_get_public_deployment_info(this.__wbg_ptr);
        return ret;
    }
    /**
     * @returns {Promise<string>}
     */
    get_public_pricing() {
        const ret = wasm.wasmbillingservice_get_public_pricing(this.__wbg_ptr);
        return ret;
    }
    /**
     * @returns {Promise<string>}
     */
    get_seat_usage() {
        const ret = wasm.wasmbillingservice_get_seat_usage(this.__wbg_ptr);
        return ret;
    }
    /**
     * @returns {Promise<string>}
     */
    get_subscription() {
        const ret = wasm.wasmbillingservice_get_subscription(this.__wbg_ptr);
        return ret;
    }
    /**
     * @param {string | null} [usage_type]
     * @returns {Promise<string>}
     */
    get_usage(usage_type) {
        var ptr0 = isLikeNone(usage_type) ? 0 : passStringToWasm0(usage_type, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        var len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmbillingservice_get_usage(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {number | null} [limit]
     * @param {number | null} [offset]
     * @returns {Promise<string>}
     */
    list_invoices(limit, offset) {
        const ret = wasm.wasmbillingservice_list_invoices(this.__wbg_ptr, isLikeNone(limit) ? 0x100000001 : (limit) >>> 0, isLikeNone(offset) ? 0x100000001 : (offset) >>> 0);
        return ret;
    }
    /**
     * @returns {Promise<string>}
     */
    list_plans() {
        const ret = wasm.wasmbillingservice_list_plans(this.__wbg_ptr);
        return ret;
    }
    /**
     * @param {string} json
     * @returns {Promise<string>}
     */
    purchase_seats(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmbillingservice_purchase_seats(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @returns {Promise<string>}
     */
    reactivate() {
        const ret = wasm.wasmbillingservice_reactivate(this.__wbg_ptr);
        return ret;
    }
    /**
     * @param {string} json
     * @returns {Promise<string>}
     */
    request_cancel(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmbillingservice_request_cancel(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {string} json
     * @returns {Promise<string>}
     */
    update_auto_renew(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmbillingservice_update_auto_renew(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {string} json
     * @returns {Promise<string>}
     */
    update_subscription(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmbillingservice_update_subscription(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {string} json
     * @returns {Promise<string>}
     */
    upgrade(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmbillingservice_upgrade(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
}
if (Symbol.dispose) WasmBillingService.prototype[Symbol.dispose] = WasmBillingService.prototype.free;

export class WasmBindingService {
    static __wrap(ptr) {
        ptr = ptr >>> 0;
        const obj = Object.create(WasmBindingService.prototype);
        obj.__wbg_ptr = ptr;
        WasmBindingServiceFinalization.register(obj, obj.__wbg_ptr, obj);
        return obj;
    }
    __destroy_into_raw() {
        const ptr = this.__wbg_ptr;
        this.__wbg_ptr = 0;
        WasmBindingServiceFinalization.unregister(this);
        return ptr;
    }
    free() {
        const ptr = this.__destroy_into_raw();
        wasm.__wbg_wasmbindingservice_free(ptr, 0);
    }
    /**
     * @param {string} json
     * @returns {Promise<string>}
     */
    accept_binding(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmbindingservice_accept_binding(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {bigint} binding_id
     * @param {string} json
     * @returns {Promise<string>}
     */
    approve_scopes(binding_id, json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmbindingservice_approve_scopes(this.__wbg_ptr, binding_id, ptr0, len0);
        return ret;
    }
    /**
     * @param {string} target_pod
     * @returns {Promise<string>}
     */
    check_binding(target_pod) {
        const ptr0 = passStringToWasm0(target_pod, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmbindingservice_check_binding(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @returns {Promise<string>}
     */
    get_bound_pods() {
        const ret = wasm.wasmbindingservice_get_bound_pods(this.__wbg_ptr);
        return ret;
    }
    /**
     * @returns {Promise<string>}
     */
    get_pending_bindings() {
        const ret = wasm.wasmbindingservice_get_pending_bindings(this.__wbg_ptr);
        return ret;
    }
    /**
     * @param {string | null} [status]
     * @returns {Promise<string>}
     */
    list_bindings(status) {
        var ptr0 = isLikeNone(status) ? 0 : passStringToWasm0(status, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        var len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmbindingservice_list_bindings(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {string} json
     * @returns {Promise<void>}
     */
    reject_binding(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmbindingservice_reject_binding(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {string} json
     * @param {string | null} [pod_key]
     * @returns {Promise<string>}
     */
    request_binding(json, pod_key) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        var ptr1 = isLikeNone(pod_key) ? 0 : passStringToWasm0(pod_key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        var len1 = WASM_VECTOR_LEN;
        const ret = wasm.wasmbindingservice_request_binding(this.__wbg_ptr, ptr0, len0, ptr1, len1);
        return ret;
    }
    /**
     * @param {bigint} binding_id
     * @param {string} json
     * @returns {Promise<string>}
     */
    request_scopes(binding_id, json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmbindingservice_request_scopes(this.__wbg_ptr, binding_id, ptr0, len0);
        return ret;
    }
    /**
     * @param {string} json
     * @returns {Promise<void>}
     */
    unbind(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmbindingservice_unbind(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
}
if (Symbol.dispose) WasmBindingService.prototype[Symbol.dispose] = WasmBindingService.prototype.free;

export class WasmBlockstoreService {
    static __wrap(ptr) {
        ptr = ptr >>> 0;
        const obj = Object.create(WasmBlockstoreService.prototype);
        obj.__wbg_ptr = ptr;
        WasmBlockstoreServiceFinalization.register(obj, obj.__wbg_ptr, obj);
        return obj;
    }
    __destroy_into_raw() {
        const ptr = this.__wbg_ptr;
        this.__wbg_ptr = 0;
        WasmBlockstoreServiceFinalization.unregister(this);
        return ptr;
    }
    free() {
        const ptr = this.__destroy_into_raw();
        wasm.__wbg_wasmblockstoreservice_free(ptr, 0);
    }
    /**
     * @param {string} req_json
     * @returns {Promise<string>}
     */
    apply_ops(req_json) {
        const ptr0 = passStringToWasm0(req_json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmblockstoreservice_apply_ops(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {string} op_json
     */
    apply_remote_op(op_json) {
        const ptr0 = passStringToWasm0(op_json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmblockstoreservice_apply_remote_op(this.__wbg_ptr, ptr0, len0);
        if (ret[1]) {
            throw takeFromExternrefTable0(ret[0]);
        }
    }
    /**
     * @returns {string}
     */
    backlinks_json() {
        let deferred1_0;
        let deferred1_1;
        try {
            const ret = wasm.wasmblockstoreservice_backlinks_json(this.__wbg_ptr);
            deferred1_0 = ret[0];
            deferred1_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred1_0, deferred1_1, 1);
        }
    }
    /**
     * @returns {string}
     */
    blocks_json() {
        let deferred1_0;
        let deferred1_1;
        try {
            const ret = wasm.wasmblockstoreservice_blocks_json(this.__wbg_ptr);
            deferred1_0 = ret[0];
            deferred1_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred1_0, deferred1_1, 1);
        }
    }
    /**
     * @param {string} workspace_id
     * @returns {Promise<void>}
     */
    catchup(workspace_id) {
        const ptr0 = passStringToWasm0(workspace_id, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmblockstoreservice_catchup(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @returns {Promise<string>}
     */
    ensure_default_workspace() {
        const ret = wasm.wasmblockstoreservice_ensure_default_workspace(this.__wbg_ptr);
        return ret;
    }
    /**
     * @param {string} id
     * @returns {any}
     */
    get_block_json(id) {
        const ptr0 = passStringToWasm0(id, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmblockstoreservice_get_block_json(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {string} workspace_id
     * @returns {bigint}
     */
    last_op_id(workspace_id) {
        const ptr0 = passStringToWasm0(workspace_id, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmblockstoreservice_last_op_id(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @returns {string}
     */
    last_op_ids_json() {
        let deferred1_0;
        let deferred1_1;
        try {
            const ret = wasm.wasmblockstoreservice_last_op_ids_json(this.__wbg_ptr);
            deferred1_0 = ret[0];
            deferred1_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred1_0, deferred1_1, 1);
        }
    }
    /**
     * @param {string} target_id
     * @returns {string}
     */
    list_backlinks_json(target_id) {
        let deferred2_0;
        let deferred2_1;
        try {
            const ptr0 = passStringToWasm0(target_id, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
            const len0 = WASM_VECTOR_LEN;
            const ret = wasm.wasmblockstoreservice_list_backlinks_json(this.__wbg_ptr, ptr0, len0);
            deferred2_0 = ret[0];
            deferred2_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred2_0, deferred2_1, 1);
        }
    }
    /**
     * @param {string} parent_id
     * @returns {string}
     */
    list_children_json(parent_id) {
        let deferred2_0;
        let deferred2_1;
        try {
            const ptr0 = passStringToWasm0(parent_id, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
            const len0 = WASM_VECTOR_LEN;
            const ret = wasm.wasmblockstoreservice_list_children_json(this.__wbg_ptr, ptr0, len0);
            deferred2_0 = ret[0];
            deferred2_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred2_0, deferred2_1, 1);
        }
    }
    /**
     * @returns {Promise<string>}
     */
    list_workspaces() {
        const ret = wasm.wasmblockstoreservice_list_workspaces(this.__wbg_ptr);
        return ret;
    }
    /**
     * @param {string} workspace_id
     * @param {string} root_id
     * @returns {Promise<void>}
     */
    load_subtree(workspace_id, root_id) {
        const ptr0 = passStringToWasm0(workspace_id, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(root_id, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        const ret = wasm.wasmblockstoreservice_load_subtree(this.__wbg_ptr, ptr0, len0, ptr1, len1);
        return ret;
    }
    /**
     * @param {string} workspace_id
     * @returns {Promise<void>}
     */
    load_type_defs(workspace_id) {
        const ptr0 = passStringToWasm0(workspace_id, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmblockstoreservice_load_type_defs(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @returns {string}
     */
    nest_children_json() {
        let deferred1_0;
        let deferred1_1;
        try {
            const ret = wasm.wasmblockstoreservice_nest_children_json(this.__wbg_ptr);
            deferred1_0 = ret[0];
            deferred1_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred1_0, deferred1_1, 1);
        }
    }
    /**
     * @returns {string}
     */
    refs_json() {
        let deferred1_0;
        let deferred1_1;
        try {
            const ret = wasm.wasmblockstoreservice_refs_json(this.__wbg_ptr);
            deferred1_0 = ret[0];
            deferred1_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred1_0, deferred1_1, 1);
        }
    }
    /**
     * @param {string} workspace_id
     * @param {string} req_json
     * @returns {Promise<string>}
     */
    semantic_search(workspace_id, req_json) {
        const ptr0 = passStringToWasm0(workspace_id, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(req_json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        const ret = wasm.wasmblockstoreservice_semantic_search(this.__wbg_ptr, ptr0, len0, ptr1, len1);
        return ret;
    }
    /**
     * @param {string} workspace_id
     * @param {bigint} id
     */
    set_last_op_id(workspace_id, id) {
        const ptr0 = passStringToWasm0(workspace_id, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmblockstoreservice_set_last_op_id(this.__wbg_ptr, ptr0, len0, id);
    }
    /**
     * @param {string} workspace_id
     * @returns {string}
     */
    type_defs_json(workspace_id) {
        let deferred2_0;
        let deferred2_1;
        try {
            const ptr0 = passStringToWasm0(workspace_id, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
            const len0 = WASM_VECTOR_LEN;
            const ret = wasm.wasmblockstoreservice_type_defs_json(this.__wbg_ptr, ptr0, len0);
            deferred2_0 = ret[0];
            deferred2_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred2_0, deferred2_1, 1);
        }
    }
    /**
     * @returns {string}
     */
    workspaces_json() {
        let deferred1_0;
        let deferred1_1;
        try {
            const ret = wasm.wasmblockstoreservice_workspaces_json(this.__wbg_ptr);
            deferred1_0 = ret[0];
            deferred1_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred1_0, deferred1_1, 1);
        }
    }
}
if (Symbol.dispose) WasmBlockstoreService.prototype[Symbol.dispose] = WasmBlockstoreService.prototype.free;

export class WasmChannelService {
    static __wrap(ptr) {
        ptr = ptr >>> 0;
        const obj = Object.create(WasmChannelService.prototype);
        obj.__wbg_ptr = ptr;
        WasmChannelServiceFinalization.register(obj, obj.__wbg_ptr, obj);
        return obj;
    }
    __destroy_into_raw() {
        const ptr = this.__wbg_ptr;
        this.__wbg_ptr = 0;
        WasmChannelServiceFinalization.unregister(this);
        return ptr;
    }
    free() {
        const ptr = this.__destroy_into_raw();
        wasm.__wbg_wasmchannelservice_free(ptr, 0);
    }
    /**
     * @param {string} json
     */
    add_channel_local(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmchannelservice_add_channel_local(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {bigint} channel_id
     * @param {string} json
     */
    add_message(channel_id, json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmchannelservice_add_message(this.__wbg_ptr, channel_id, ptr0, len0);
    }
    /**
     * @param {bigint} id
     * @returns {Promise<void>}
     */
    archive_channel(id) {
        const ret = wasm.wasmchannelservice_archive_channel(this.__wbg_ptr, id);
        return ret;
    }
    /**
     * @param {bigint} id
     * @returns {string}
     */
    channel_members_json(id) {
        let deferred1_0;
        let deferred1_1;
        try {
            const ret = wasm.wasmchannelservice_channel_members_json(this.__wbg_ptr, id);
            deferred1_0 = ret[0];
            deferred1_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred1_0, deferred1_1, 1);
        }
    }
    /**
     * @param {bigint} id
     * @returns {string}
     */
    channel_pods_json(id) {
        let deferred1_0;
        let deferred1_1;
        try {
            const ret = wasm.wasmchannelservice_channel_pods_json(this.__wbg_ptr, id);
            deferred1_0 = ret[0];
            deferred1_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred1_0, deferred1_1, 1);
        }
    }
    /**
     * @returns {string}
     */
    channels_json() {
        let deferred1_0;
        let deferred1_1;
        try {
            const ret = wasm.wasmchannelservice_channels_json(this.__wbg_ptr);
            deferred1_0 = ret[0];
            deferred1_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred1_0, deferred1_1, 1);
        }
    }
    /**
     * @param {bigint} channel_id
     */
    clear_channel_mentions(channel_id) {
        wasm.wasmchannelservice_clear_channel_mentions(this.__wbg_ptr, channel_id);
    }
    /**
     * @param {bigint} channel_id
     */
    clear_channel_unread(channel_id) {
        wasm.wasmchannelservice_clear_channel_unread(this.__wbg_ptr, channel_id);
    }
    /**
     * @param {string} request_json
     * @returns {Promise<string>}
     */
    create_channel(request_json) {
        const ptr0 = passStringToWasm0(request_json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmchannelservice_create_channel(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @returns {any}
     */
    current_channel_json() {
        const ret = wasm.wasmchannelservice_current_channel_json(this.__wbg_ptr);
        return ret;
    }
    /**
     * @param {bigint} channel_id
     * @param {bigint} message_id
     * @returns {Promise<void>}
     */
    delete_message(channel_id, message_id) {
        const ret = wasm.wasmchannelservice_delete_message(this.__wbg_ptr, channel_id, message_id);
        return ret;
    }
    /**
     * @param {bigint} channel_id
     * @param {bigint} message_id
     * @param {string} content
     * @returns {Promise<string>}
     */
    edit_message(channel_id, message_id, content) {
        const ptr0 = passStringToWasm0(content, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmchannelservice_edit_message(this.__wbg_ptr, channel_id, message_id, ptr0, len0);
        return ret;
    }
    /**
     * @param {bigint} id
     * @returns {Promise<string>}
     */
    fetch_channel(id) {
        const ret = wasm.wasmchannelservice_fetch_channel(this.__wbg_ptr, id);
        return ret;
    }
    /**
     * @param {bigint} id
     * @returns {Promise<string>}
     */
    fetch_channel_members(id) {
        const ret = wasm.wasmchannelservice_fetch_channel_members(this.__wbg_ptr, id);
        return ret;
    }
    /**
     * @param {boolean | null} [include_archived]
     * @returns {Promise<string>}
     */
    fetch_channels(include_archived) {
        const ret = wasm.wasmchannelservice_fetch_channels(this.__wbg_ptr, isLikeNone(include_archived) ? 0xFFFFFF : include_archived ? 1 : 0);
        return ret;
    }
    /**
     * @param {bigint} channel_id
     * @param {number | null} [limit]
     * @param {bigint | null} [before_id]
     * @returns {Promise<string>}
     */
    fetch_messages(channel_id, limit, before_id) {
        const ret = wasm.wasmchannelservice_fetch_messages(this.__wbg_ptr, channel_id, isLikeNone(limit) ? 0x100000001 : (limit) >>> 0, !isLikeNone(before_id), isLikeNone(before_id) ? BigInt(0) : before_id);
        return ret;
    }
    /**
     * @returns {Promise<string>}
     */
    fetch_unread_counts() {
        const ret = wasm.wasmchannelservice_fetch_unread_counts(this.__wbg_ptr);
        return ret;
    }
    /**
     * @param {string} query
     * @param {boolean} include_archived
     * @returns {string}
     */
    filter_channels_json(query, include_archived) {
        let deferred2_0;
        let deferred2_1;
        try {
            const ptr0 = passStringToWasm0(query, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
            const len0 = WASM_VECTOR_LEN;
            const ret = wasm.wasmchannelservice_filter_channels_json(this.__wbg_ptr, ptr0, len0, include_archived);
            deferred2_0 = ret[0];
            deferred2_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred2_0, deferred2_1, 1);
        }
    }
    /**
     * @param {bigint} id
     * @returns {any}
     */
    get_channel_json(id) {
        const ret = wasm.wasmchannelservice_get_channel_json(this.__wbg_ptr, id);
        return ret;
    }
    /**
     * @param {bigint} id
     * @returns {Promise<string>}
     */
    get_channel_pods(id) {
        const ret = wasm.wasmchannelservice_get_channel_pods(this.__wbg_ptr, id);
        return ret;
    }
    /**
     * @param {bigint} channel_id
     * @returns {any}
     */
    get_last_message_json(channel_id) {
        const ret = wasm.wasmchannelservice_get_last_message_json(this.__wbg_ptr, channel_id);
        return ret;
    }
    /**
     * @param {bigint} channel_id
     * @returns {number}
     */
    get_mention_count(channel_id) {
        const ret = wasm.wasmchannelservice_get_mention_count(this.__wbg_ptr, channel_id);
        return ret >>> 0;
    }
    /**
     * @param {bigint} channel_id
     * @returns {any}
     */
    get_messages_json(channel_id) {
        const ret = wasm.wasmchannelservice_get_messages_json(this.__wbg_ptr, channel_id);
        return ret;
    }
    /**
     * @param {bigint} channel_id
     * @returns {number}
     */
    get_unread_count(channel_id) {
        const ret = wasm.wasmchannelservice_get_unread_count(this.__wbg_ptr, channel_id);
        return ret >>> 0;
    }
    /**
     * @param {bigint} channel_id
     */
    increment_mention(channel_id) {
        wasm.wasmchannelservice_increment_mention(this.__wbg_ptr, channel_id);
    }
    /**
     * @param {bigint} channel_id
     */
    increment_unread(channel_id) {
        wasm.wasmchannelservice_increment_unread(this.__wbg_ptr, channel_id);
    }
    /**
     * @param {bigint} id
     * @param {string} user_ids_json
     * @returns {Promise<void>}
     */
    invite_channel_members(id, user_ids_json) {
        const ptr0 = passStringToWasm0(user_ids_json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmchannelservice_invite_channel_members(this.__wbg_ptr, id, ptr0, len0);
        return ret;
    }
    /**
     * @param {bigint} channel_id
     * @param {string} pod_key
     * @returns {Promise<string>}
     */
    join_channel(channel_id, pod_key) {
        const ptr0 = passStringToWasm0(pod_key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmchannelservice_join_channel(this.__wbg_ptr, channel_id, ptr0, len0);
        return ret;
    }
    /**
     * @param {bigint} channel_id
     * @param {string} pod_key
     * @returns {Promise<string>}
     */
    leave_channel(channel_id, pod_key) {
        const ptr0 = passStringToWasm0(pod_key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmchannelservice_leave_channel(this.__wbg_ptr, channel_id, ptr0, len0);
        return ret;
    }
    /**
     * @param {bigint} channel_id
     * @param {bigint} message_id
     * @returns {Promise<void>}
     */
    mark_read(channel_id, message_id) {
        const ret = wasm.wasmchannelservice_mark_read(this.__wbg_ptr, channel_id, message_id);
        return ret;
    }
    /**
     * @returns {string}
     */
    mention_counts_json() {
        let deferred1_0;
        let deferred1_1;
        try {
            const ret = wasm.wasmchannelservice_mention_counts_json(this.__wbg_ptr);
            deferred1_0 = ret[0];
            deferred1_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred1_0, deferred1_1, 1);
        }
    }
    /**
     * @param {bigint} channel_id
     * @param {boolean} muted
     * @returns {Promise<void>}
     */
    mute_channel(channel_id, muted) {
        const ret = wasm.wasmchannelservice_mute_channel(this.__wbg_ptr, channel_id, muted);
        return ret;
    }
    /**
     * @param {string} json
     * @returns {boolean}
     */
    on_new_message(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmchannelservice_on_new_message(this.__wbg_ptr, ptr0, len0);
        return ret !== 0;
    }
    /**
     * @param {bigint} channel_id
     * @param {string} json
     * @param {boolean} has_more
     */
    prepend_messages(channel_id, json, has_more) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmchannelservice_prepend_messages(this.__wbg_ptr, channel_id, ptr0, len0, has_more);
    }
    /**
     * @param {bigint} id
     */
    remove_channel_local(id) {
        wasm.wasmchannelservice_remove_channel_local(this.__wbg_ptr, id);
    }
    /**
     * @param {bigint} id
     * @param {bigint} user_id
     * @returns {Promise<void>}
     */
    remove_channel_member(id, user_id) {
        const ret = wasm.wasmchannelservice_remove_channel_member(this.__wbg_ptr, id, user_id);
        return ret;
    }
    /**
     * @param {bigint} channel_id
     * @param {bigint} message_id
     */
    remove_message_local(channel_id, message_id) {
        wasm.wasmchannelservice_remove_message_local(this.__wbg_ptr, channel_id, message_id);
    }
    /**
     * @param {bigint} id
     * @param {string} q
     * @param {number | null} [limit]
     * @returns {Promise<string>}
     */
    search_channel_messages(id, q, limit) {
        const ptr0 = passStringToWasm0(q, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmchannelservice_search_channel_messages(this.__wbg_ptr, id, ptr0, len0, isLikeNone(limit) ? 0x100000001 : (limit) >>> 0);
        return ret;
    }
    /**
     * @param {bigint | null} [id]
     * @returns {any}
     */
    select_channel(id) {
        const ret = wasm.wasmchannelservice_select_channel(this.__wbg_ptr, !isLikeNone(id), isLikeNone(id) ? BigInt(0) : id);
        return ret;
    }
    /**
     * @param {bigint} channel_id
     * @param {string} request_json
     * @returns {Promise<string>}
     */
    send_message(channel_id, request_json) {
        const ptr0 = passStringToWasm0(request_json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmchannelservice_send_message(this.__wbg_ptr, channel_id, ptr0, len0);
        return ret;
    }
    /**
     * @param {string} json
     */
    set_channels(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmchannelservice_set_channels(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {bigint | null} [id]
     */
    set_current_channel(id) {
        wasm.wasmchannelservice_set_current_channel(this.__wbg_ptr, !isLikeNone(id), isLikeNone(id) ? BigInt(0) : id);
    }
    /**
     * @param {string} user_json
     */
    set_current_user(user_json) {
        const ptr0 = passStringToWasm0(user_json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmchannelservice_set_current_user(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {bigint | null} [user_id]
     */
    set_current_user_id(user_id) {
        wasm.wasmchannelservice_set_current_user_id(this.__wbg_ptr, !isLikeNone(user_id), isLikeNone(user_id) ? BigInt(0) : user_id);
    }
    /**
     * @param {bigint} channel_id
     * @param {string} json
     */
    set_last_message(channel_id, json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmchannelservice_set_last_message(this.__wbg_ptr, channel_id, ptr0, len0);
    }
    /**
     * @param {string} json
     */
    set_mention_counts(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmchannelservice_set_mention_counts(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {bigint} channel_id
     * @param {string} json
     * @param {boolean} has_more
     */
    set_messages(channel_id, json, has_more) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmchannelservice_set_messages(this.__wbg_ptr, channel_id, ptr0, len0, has_more);
    }
    /**
     * @param {string} json
     */
    set_unread_counts(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmchannelservice_set_unread_counts(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {string} mode
     * @param {boolean} include_archived
     * @returns {string}
     */
    sorted_channel_ids_json(mode, include_archived) {
        let deferred2_0;
        let deferred2_1;
        try {
            const ptr0 = passStringToWasm0(mode, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
            const len0 = WASM_VECTOR_LEN;
            const ret = wasm.wasmchannelservice_sorted_channel_ids_json(this.__wbg_ptr, ptr0, len0, include_archived);
            deferred2_0 = ret[0];
            deferred2_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred2_0, deferred2_1, 1);
        }
    }
    /**
     * @returns {number}
     */
    total_mention_count() {
        const ret = wasm.wasmchannelservice_total_mention_count(this.__wbg_ptr);
        return ret >>> 0;
    }
    /**
     * @returns {number}
     */
    total_unread_count() {
        const ret = wasm.wasmchannelservice_total_unread_count(this.__wbg_ptr);
        return ret >>> 0;
    }
    /**
     * @param {bigint} id
     * @returns {Promise<void>}
     */
    unarchive_channel(id) {
        const ret = wasm.wasmchannelservice_unarchive_channel(this.__wbg_ptr, id);
        return ret;
    }
    /**
     * @returns {string}
     */
    unread_counts_json() {
        let deferred1_0;
        let deferred1_1;
        try {
            const ret = wasm.wasmchannelservice_unread_counts_json(this.__wbg_ptr);
            deferred1_0 = ret[0];
            deferred1_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred1_0, deferred1_1, 1);
        }
    }
    /**
     * @param {bigint} id
     * @param {string} request_json
     * @returns {Promise<string>}
     */
    update_channel(id, request_json) {
        const ptr0 = passStringToWasm0(request_json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmchannelservice_update_channel(this.__wbg_ptr, id, ptr0, len0);
        return ret;
    }
    /**
     * @param {bigint} id
     * @param {string} json
     */
    update_channel_local(id, json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmchannelservice_update_channel_local(this.__wbg_ptr, id, ptr0, len0);
    }
    /**
     * @param {bigint} channel_id
     * @param {string} json
     */
    update_message_local(channel_id, json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmchannelservice_update_message_local(this.__wbg_ptr, channel_id, ptr0, len0);
    }
}
if (Symbol.dispose) WasmChannelService.prototype[Symbol.dispose] = WasmChannelService.prototype.free;

export class WasmChannelState {
    __destroy_into_raw() {
        const ptr = this.__wbg_ptr;
        this.__wbg_ptr = 0;
        WasmChannelStateFinalization.unregister(this);
        return ptr;
    }
    free() {
        const ptr = this.__destroy_into_raw();
        wasm.__wbg_wasmchannelstate_free(ptr, 0);
    }
    /**
     * @param {string} json
     */
    add_channel(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmchannelstate_add_channel(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {bigint} channel_id
     * @param {string} message_json
     */
    add_message(channel_id, message_json) {
        const ptr0 = passStringToWasm0(message_json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmchannelstate_add_message(this.__wbg_ptr, channel_id, ptr0, len0);
    }
    /**
     * @returns {string}
     */
    channels_json() {
        let deferred1_0;
        let deferred1_1;
        try {
            const ret = wasm.wasmchannelstate_channels_json(this.__wbg_ptr);
            deferred1_0 = ret[0];
            deferred1_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred1_0, deferred1_1, 1);
        }
    }
    /**
     * @param {bigint} channel_id
     */
    clear_channel_mentions(channel_id) {
        wasm.wasmchannelstate_clear_channel_mentions(this.__wbg_ptr, channel_id);
    }
    /**
     * @param {bigint} channel_id
     */
    clear_channel_unread(channel_id) {
        wasm.wasmchannelstate_clear_channel_unread(this.__wbg_ptr, channel_id);
    }
    /**
     * @returns {any}
     */
    current_channel_json() {
        const ret = wasm.wasmchannelstate_current_channel_json(this.__wbg_ptr);
        return ret;
    }
    /**
     * @param {string} query
     * @param {boolean} include_archived
     * @returns {string}
     */
    filter_channels_json(query, include_archived) {
        let deferred2_0;
        let deferred2_1;
        try {
            const ptr0 = passStringToWasm0(query, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
            const len0 = WASM_VECTOR_LEN;
            const ret = wasm.wasmchannelstate_filter_channels_json(this.__wbg_ptr, ptr0, len0, include_archived);
            deferred2_0 = ret[0];
            deferred2_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred2_0, deferred2_1, 1);
        }
    }
    /**
     * @param {bigint} id
     * @returns {any}
     */
    get_channel_json(id) {
        const ret = wasm.wasmchannelstate_get_channel_json(this.__wbg_ptr, id);
        return ret;
    }
    /**
     * @param {bigint} channel_id
     * @returns {any}
     */
    get_last_message_json(channel_id) {
        const ret = wasm.wasmchannelstate_get_last_message_json(this.__wbg_ptr, channel_id);
        return ret;
    }
    /**
     * @param {bigint} channel_id
     * @returns {number}
     */
    get_mention_count(channel_id) {
        const ret = wasm.wasmchannelstate_get_mention_count(this.__wbg_ptr, channel_id);
        return ret >>> 0;
    }
    /**
     * @param {bigint} channel_id
     * @returns {any}
     */
    get_messages_json(channel_id) {
        const ret = wasm.wasmchannelstate_get_messages_json(this.__wbg_ptr, channel_id);
        return ret;
    }
    /**
     * @param {bigint} channel_id
     * @returns {number}
     */
    get_unread_count(channel_id) {
        const ret = wasm.wasmchannelstate_get_unread_count(this.__wbg_ptr, channel_id);
        return ret >>> 0;
    }
    /**
     * @param {bigint} channel_id
     */
    increment_mention(channel_id) {
        wasm.wasmchannelstate_increment_mention(this.__wbg_ptr, channel_id);
    }
    /**
     * @param {bigint} channel_id
     */
    increment_unread(channel_id) {
        wasm.wasmchannelstate_increment_unread(this.__wbg_ptr, channel_id);
    }
    /**
     * Return all mention counts as JSON.
     * @returns {string}
     */
    mention_counts_json() {
        let deferred1_0;
        let deferred1_1;
        try {
            const ret = wasm.wasmchannelstate_mention_counts_json(this.__wbg_ptr);
            deferred1_0 = ret[0];
            deferred1_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred1_0, deferred1_1, 1);
        }
    }
    constructor() {
        const ret = wasm.wasmchannelstate_new();
        this.__wbg_ptr = ret >>> 0;
        WasmChannelStateFinalization.register(this, this.__wbg_ptr, this);
        return this;
    }
    /**
     * Handle a new incoming message (from realtime event).
     * Enriches sender, updates preview, increments unread if appropriate.
     * Returns true if the message was new (not a duplicate).
     * @param {string} message_json
     * @returns {boolean}
     */
    on_new_message(message_json) {
        const ptr0 = passStringToWasm0(message_json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmchannelstate_on_new_message(this.__wbg_ptr, ptr0, len0);
        return ret !== 0;
    }
    /**
     * @param {bigint} channel_id
     * @param {string} messages_json
     * @param {boolean} has_more
     */
    prepend_messages(channel_id, messages_json, has_more) {
        const ptr0 = passStringToWasm0(messages_json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmchannelstate_prepend_messages(this.__wbg_ptr, channel_id, ptr0, len0, has_more);
    }
    /**
     * @param {bigint} id
     */
    remove_channel(id) {
        wasm.wasmchannelstate_remove_channel(this.__wbg_ptr, id);
    }
    /**
     * @param {bigint} channel_id
     * @param {bigint} message_id
     */
    remove_message(channel_id, message_id) {
        wasm.wasmchannelstate_remove_message(this.__wbg_ptr, channel_id, message_id);
    }
    /**
     * Atomically: set current channel + clear unread + clear mentions.
     * @param {bigint | null} [id]
     * @returns {any}
     */
    select_channel(id) {
        const ret = wasm.wasmchannelstate_select_channel(this.__wbg_ptr, !isLikeNone(id), isLikeNone(id) ? BigInt(0) : id);
        return ret;
    }
    /**
     * @param {string} json
     */
    set_channels(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmchannelstate_set_channels(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {bigint | null} [id]
     */
    set_current_channel(id) {
        wasm.wasmchannelstate_set_current_channel(this.__wbg_ptr, !isLikeNone(id), isLikeNone(id) ? BigInt(0) : id);
    }
    /**
     * @param {string} user_json
     */
    set_current_user(user_json) {
        const ptr0 = passStringToWasm0(user_json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmchannelstate_set_current_user(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {bigint | null} [user_id]
     */
    set_current_user_id(user_id) {
        wasm.wasmchannelstate_set_current_user_id(this.__wbg_ptr, !isLikeNone(user_id), isLikeNone(user_id) ? BigInt(0) : user_id);
    }
    /**
     * @param {bigint} channel_id
     * @param {string} preview_json
     */
    set_last_message(channel_id, preview_json) {
        const ptr0 = passStringToWasm0(preview_json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmchannelstate_set_last_message(this.__wbg_ptr, channel_id, ptr0, len0);
    }
    /**
     * @param {string} json
     */
    set_mention_counts(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmchannelstate_set_mention_counts(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {bigint} channel_id
     * @param {string} messages_json
     * @param {boolean} has_more
     */
    set_messages(channel_id, messages_json, has_more) {
        const ptr0 = passStringToWasm0(messages_json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmchannelstate_set_messages(this.__wbg_ptr, channel_id, ptr0, len0, has_more);
    }
    /**
     * @param {string} json
     */
    set_unread_counts(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmchannelstate_set_unread_counts(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {string} mode
     * @param {boolean} include_archived
     * @returns {string}
     */
    sorted_channel_ids_json(mode, include_archived) {
        let deferred2_0;
        let deferred2_1;
        try {
            const ptr0 = passStringToWasm0(mode, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
            const len0 = WASM_VECTOR_LEN;
            const ret = wasm.wasmchannelstate_sorted_channel_ids_json(this.__wbg_ptr, ptr0, len0, include_archived);
            deferred2_0 = ret[0];
            deferred2_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred2_0, deferred2_1, 1);
        }
    }
    /**
     * @returns {number}
     */
    total_mention_count() {
        const ret = wasm.wasmchannelstate_total_mention_count(this.__wbg_ptr);
        return ret >>> 0;
    }
    /**
     * @returns {number}
     */
    total_unread_count() {
        const ret = wasm.wasmchannelstate_total_unread_count(this.__wbg_ptr);
        return ret >>> 0;
    }
    /**
     * Return all unread counts as JSON: `{"1": 3, "2": 5}`.
     * @returns {string}
     */
    unread_counts_json() {
        let deferred1_0;
        let deferred1_1;
        try {
            const ret = wasm.wasmchannelstate_unread_counts_json(this.__wbg_ptr);
            deferred1_0 = ret[0];
            deferred1_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred1_0, deferred1_1, 1);
        }
    }
    /**
     * @param {bigint} id
     * @param {string} json
     */
    update_channel(id, json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmchannelstate_update_channel(this.__wbg_ptr, id, ptr0, len0);
    }
    /**
     * @param {bigint} channel_id
     * @param {string} message_json
     */
    update_message(channel_id, message_json) {
        const ptr0 = passStringToWasm0(message_json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmchannelstate_update_message(this.__wbg_ptr, channel_id, ptr0, len0);
    }
}
if (Symbol.dispose) WasmChannelState.prototype[Symbol.dispose] = WasmChannelState.prototype.free;

export class WasmEventsManager {
    static __wrap(ptr) {
        ptr = ptr >>> 0;
        const obj = Object.create(WasmEventsManager.prototype);
        obj.__wbg_ptr = ptr;
        WasmEventsManagerFinalization.register(obj, obj.__wbg_ptr, obj);
        return obj;
    }
    __destroy_into_raw() {
        const ptr = this.__wbg_ptr;
        this.__wbg_ptr = 0;
        WasmEventsManagerFinalization.unregister(this);
        return ptr;
    }
    free() {
        const ptr = this.__destroy_into_raw();
        wasm.__wbg_wasmeventsmanager_free(ptr, 0);
    }
    /**
     * @returns {Promise<void>}
     */
    connect() {
        const ret = wasm.wasmeventsmanager_connect(this.__wbg_ptr);
        return ret;
    }
    /**
     * @returns {Promise<void>}
     */
    disconnect() {
        const ret = wasm.wasmeventsmanager_disconnect(this.__wbg_ptr);
        return ret;
    }
    /**
     * @returns {Promise<string>}
     */
    get_connection_state() {
        const ret = wasm.wasmeventsmanager_get_connection_state(this.__wbg_ptr);
        return ret;
    }
    /**
     * @param {string} ws_url
     */
    constructor(ws_url) {
        const ptr0 = passStringToWasm0(ws_url, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmeventsmanager_new(ptr0, len0);
        this.__wbg_ptr = ret >>> 0;
        WasmEventsManagerFinalization.register(this, this.__wbg_ptr, this);
        return this;
    }
    /**
     * @param {string} ws_url
     * @param {number} max_reconnect_attempts
     * @param {number} initial_reconnect_delay_ms
     * @param {number} max_reconnect_delay_ms
     * @param {number} ping_interval_ms
     * @param {number} pong_timeout_ms
     * @returns {WasmEventsManager}
     */
    static new_with_options(ws_url, max_reconnect_attempts, initial_reconnect_delay_ms, max_reconnect_delay_ms, ping_interval_ms, pong_timeout_ms) {
        const ptr0 = passStringToWasm0(ws_url, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmeventsmanager_new_with_options(ptr0, len0, max_reconnect_attempts, initial_reconnect_delay_ms, max_reconnect_delay_ms, ping_interval_ms, pong_timeout_ms);
        return WasmEventsManager.__wrap(ret);
    }
    /**
     * @param {Function} callback
     * @returns {Promise<number>}
     */
    on_connection_state_change(callback) {
        const ret = wasm.wasmeventsmanager_on_connection_state_change(this.__wbg_ptr, callback);
        return ret;
    }
    /**
     * @param {string} event_type
     * @param {Function} callback
     * @returns {Promise<number>}
     */
    subscribe(event_type, callback) {
        const ptr0 = passStringToWasm0(event_type, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmeventsmanager_subscribe(this.__wbg_ptr, ptr0, len0, callback);
        return ret;
    }
    /**
     * @param {Function} callback
     * @returns {Promise<number>}
     */
    subscribe_all(callback) {
        const ret = wasm.wasmeventsmanager_subscribe_all(this.__wbg_ptr, callback);
        return ret;
    }
    /**
     * @param {number} id
     * @returns {Promise<void>}
     */
    unsubscribe(id) {
        const ret = wasm.wasmeventsmanager_unsubscribe(this.__wbg_ptr, id);
        return ret;
    }
}
if (Symbol.dispose) WasmEventsManager.prototype[Symbol.dispose] = WasmEventsManager.prototype.free;

export class WasmExtensionService {
    static __wrap(ptr) {
        ptr = ptr >>> 0;
        const obj = Object.create(WasmExtensionService.prototype);
        obj.__wbg_ptr = ptr;
        WasmExtensionServiceFinalization.register(obj, obj.__wbg_ptr, obj);
        return obj;
    }
    __destroy_into_raw() {
        const ptr = this.__wbg_ptr;
        this.__wbg_ptr = 0;
        WasmExtensionServiceFinalization.unregister(this);
        return ptr;
    }
    free() {
        const ptr = this.__destroy_into_raw();
        wasm.__wbg_wasmextensionservice_free(ptr, 0);
    }
    /**
     * @param {string} json
     * @returns {Promise<string>}
     */
    create_skill_registry(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmextensionservice_create_skill_registry(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {bigint} id
     * @returns {Promise<void>}
     */
    delete_skill_registry(id) {
        const ret = wasm.wasmextensionservice_delete_skill_registry(this.__wbg_ptr, id);
        return ret;
    }
    /**
     * @param {bigint} repo_id
     * @param {string} json
     * @returns {Promise<string>}
     */
    install_custom_mcp_server(repo_id, json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmextensionservice_install_custom_mcp_server(this.__wbg_ptr, repo_id, ptr0, len0);
        return ret;
    }
    /**
     * @param {bigint} repo_id
     * @param {string} json
     * @returns {Promise<string>}
     */
    install_mcp_from_market(repo_id, json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmextensionservice_install_mcp_from_market(this.__wbg_ptr, repo_id, ptr0, len0);
        return ret;
    }
    /**
     * @param {bigint} repo_id
     * @param {string} json
     * @returns {Promise<string>}
     */
    install_skill_from_github(repo_id, json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmextensionservice_install_skill_from_github(this.__wbg_ptr, repo_id, ptr0, len0);
        return ret;
    }
    /**
     * @param {bigint} repo_id
     * @param {string} json
     * @returns {Promise<string>}
     */
    install_skill_from_market(repo_id, json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmextensionservice_install_skill_from_market(this.__wbg_ptr, repo_id, ptr0, len0);
        return ret;
    }
    /**
     * @param {bigint} repo_id
     * @param {Uint8Array} file_data
     * @param {string} file_name
     * @param {string | null} [scope]
     * @returns {Promise<string>}
     */
    install_skill_from_upload(repo_id, file_data, file_name, scope) {
        const ptr0 = passStringToWasm0(file_name, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        var ptr1 = isLikeNone(scope) ? 0 : passStringToWasm0(scope, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        var len1 = WASM_VECTOR_LEN;
        const ret = wasm.wasmextensionservice_install_skill_from_upload(this.__wbg_ptr, repo_id, file_data, ptr0, len0, ptr1, len1);
        return ret;
    }
    /**
     * @param {string | null} [query]
     * @param {number | null} [limit]
     * @param {number | null} [offset]
     * @returns {Promise<string>}
     */
    list_market_mcp_servers(query, limit, offset) {
        var ptr0 = isLikeNone(query) ? 0 : passStringToWasm0(query, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        var len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmextensionservice_list_market_mcp_servers(this.__wbg_ptr, ptr0, len0, isLikeNone(limit) ? 0x100000001 : (limit) >>> 0, isLikeNone(offset) ? 0x100000001 : (offset) >>> 0);
        return ret;
    }
    /**
     * @param {string | null} [query]
     * @param {string | null} [category]
     * @returns {Promise<string>}
     */
    list_market_skills(query, category) {
        var ptr0 = isLikeNone(query) ? 0 : passStringToWasm0(query, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        var len0 = WASM_VECTOR_LEN;
        var ptr1 = isLikeNone(category) ? 0 : passStringToWasm0(category, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        var len1 = WASM_VECTOR_LEN;
        const ret = wasm.wasmextensionservice_list_market_skills(this.__wbg_ptr, ptr0, len0, ptr1, len1);
        return ret;
    }
    /**
     * @param {bigint} repo_id
     * @param {string | null} [scope]
     * @returns {Promise<string>}
     */
    list_repo_mcp_servers(repo_id, scope) {
        var ptr0 = isLikeNone(scope) ? 0 : passStringToWasm0(scope, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        var len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmextensionservice_list_repo_mcp_servers(this.__wbg_ptr, repo_id, ptr0, len0);
        return ret;
    }
    /**
     * @param {bigint} repo_id
     * @param {string | null} [scope]
     * @returns {Promise<string>}
     */
    list_repo_skills(repo_id, scope) {
        var ptr0 = isLikeNone(scope) ? 0 : passStringToWasm0(scope, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        var len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmextensionservice_list_repo_skills(this.__wbg_ptr, repo_id, ptr0, len0);
        return ret;
    }
    /**
     * @returns {Promise<string>}
     */
    list_skill_registries() {
        const ret = wasm.wasmextensionservice_list_skill_registries(this.__wbg_ptr);
        return ret;
    }
    /**
     * @returns {Promise<string>}
     */
    list_skill_registry_overrides() {
        const ret = wasm.wasmextensionservice_list_skill_registry_overrides(this.__wbg_ptr);
        return ret;
    }
    /**
     * @param {bigint} id
     * @returns {Promise<void>}
     */
    sync_skill_registry(id) {
        const ret = wasm.wasmextensionservice_sync_skill_registry(this.__wbg_ptr, id);
        return ret;
    }
    /**
     * @param {bigint} id
     * @param {string} json
     * @returns {Promise<string>}
     */
    toggle_skill_registry(id, json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmextensionservice_toggle_skill_registry(this.__wbg_ptr, id, ptr0, len0);
        return ret;
    }
    /**
     * @param {bigint} repo_id
     * @param {bigint} install_id
     * @returns {Promise<void>}
     */
    uninstall_mcp_server(repo_id, install_id) {
        const ret = wasm.wasmextensionservice_uninstall_mcp_server(this.__wbg_ptr, repo_id, install_id);
        return ret;
    }
    /**
     * @param {bigint} repo_id
     * @param {bigint} install_id
     * @returns {Promise<void>}
     */
    uninstall_skill(repo_id, install_id) {
        const ret = wasm.wasmextensionservice_uninstall_skill(this.__wbg_ptr, repo_id, install_id);
        return ret;
    }
    /**
     * @param {bigint} repo_id
     * @param {bigint} install_id
     * @param {string} json
     * @returns {Promise<string>}
     */
    update_mcp_server(repo_id, install_id, json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmextensionservice_update_mcp_server(this.__wbg_ptr, repo_id, install_id, ptr0, len0);
        return ret;
    }
    /**
     * @param {bigint} repo_id
     * @param {bigint} install_id
     * @param {string} json
     * @returns {Promise<string>}
     */
    update_skill(repo_id, install_id, json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmextensionservice_update_skill(this.__wbg_ptr, repo_id, install_id, ptr0, len0);
        return ret;
    }
}
if (Symbol.dispose) WasmExtensionService.prototype[Symbol.dispose] = WasmExtensionService.prototype.free;

export class WasmFileService {
    static __wrap(ptr) {
        ptr = ptr >>> 0;
        const obj = Object.create(WasmFileService.prototype);
        obj.__wbg_ptr = ptr;
        WasmFileServiceFinalization.register(obj, obj.__wbg_ptr, obj);
        return obj;
    }
    __destroy_into_raw() {
        const ptr = this.__wbg_ptr;
        this.__wbg_ptr = 0;
        WasmFileServiceFinalization.unregister(this);
        return ptr;
    }
    free() {
        const ptr = this.__destroy_into_raw();
        wasm.__wbg_wasmfileservice_free(ptr, 0);
    }
    /**
     * @param {string} json
     * @returns {Promise<string>}
     */
    presign_upload(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmfileservice_presign_upload(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {Uint8Array} file_data
     * @param {string} filename
     * @param {string} content_type
     * @returns {Promise<string>}
     */
    upload_file(file_data, filename, content_type) {
        const ptr0 = passStringToWasm0(filename, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(content_type, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        const ret = wasm.wasmfileservice_upload_file(this.__wbg_ptr, file_data, ptr0, len0, ptr1, len1);
        return ret;
    }
}
if (Symbol.dispose) WasmFileService.prototype[Symbol.dispose] = WasmFileService.prototype.free;

export class WasmGitProviderState {
    __destroy_into_raw() {
        const ptr = this.__wbg_ptr;
        this.__wbg_ptr = 0;
        WasmGitProviderStateFinalization.unregister(this);
        return ptr;
    }
    free() {
        const ptr = this.__destroy_into_raw();
        wasm.__wbg_wasmgitproviderstate_free(ptr, 0);
    }
    /**
     * @param {string} json
     */
    add_provider(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmgitproviderstate_add_provider(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @returns {string}
     */
    available_projects_json() {
        let deferred1_0;
        let deferred1_1;
        try {
            const ret = wasm.wasmgitproviderstate_available_projects_json(this.__wbg_ptr);
            deferred1_0 = ret[0];
            deferred1_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred1_0, deferred1_1, 1);
        }
    }
    /**
     * @returns {any}
     */
    current_provider_json() {
        const ret = wasm.wasmgitproviderstate_current_provider_json(this.__wbg_ptr);
        return ret;
    }
    constructor() {
        const ret = wasm.wasmgitproviderstate_new();
        this.__wbg_ptr = ret >>> 0;
        WasmGitProviderStateFinalization.register(this, this.__wbg_ptr, this);
        return this;
    }
    /**
     * @returns {string}
     */
    providers_json() {
        let deferred1_0;
        let deferred1_1;
        try {
            const ret = wasm.wasmgitproviderstate_providers_json(this.__wbg_ptr);
            deferred1_0 = ret[0];
            deferred1_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred1_0, deferred1_1, 1);
        }
    }
    /**
     * @param {string} id
     */
    remove_provider(id) {
        const ptr0 = passStringToWasm0(id, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmgitproviderstate_remove_provider(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {string} json
     */
    set_available_projects(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmgitproviderstate_set_available_projects(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {string} json
     */
    set_current_provider(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmgitproviderstate_set_current_provider(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {string} json
     */
    set_providers(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmgitproviderstate_set_providers(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {string} id
     * @param {string} json
     */
    update_provider(id, json) {
        const ptr0 = passStringToWasm0(id, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        wasm.wasmgitproviderstate_update_provider(this.__wbg_ptr, ptr0, len0, ptr1, len1);
    }
}
if (Symbol.dispose) WasmGitProviderState.prototype[Symbol.dispose] = WasmGitProviderState.prototype.free;

export class WasmGrantService {
    static __wrap(ptr) {
        ptr = ptr >>> 0;
        const obj = Object.create(WasmGrantService.prototype);
        obj.__wbg_ptr = ptr;
        WasmGrantServiceFinalization.register(obj, obj.__wbg_ptr, obj);
        return obj;
    }
    __destroy_into_raw() {
        const ptr = this.__wbg_ptr;
        this.__wbg_ptr = 0;
        WasmGrantServiceFinalization.unregister(this);
        return ptr;
    }
    free() {
        const ptr = this.__destroy_into_raw();
        wasm.__wbg_wasmgrantservice_free(ptr, 0);
    }
    /**
     * @param {string} resource_type
     * @param {string} resource_id
     * @param {bigint} user_id
     * @returns {Promise<string>}
     */
    grant(resource_type, resource_id, user_id) {
        const ptr0 = passStringToWasm0(resource_type, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(resource_id, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        const ret = wasm.wasmgrantservice_grant(this.__wbg_ptr, ptr0, len0, ptr1, len1, user_id);
        return ret;
    }
    /**
     * @param {string} resource_type
     * @param {string} resource_id
     * @returns {Promise<string>}
     */
    list(resource_type, resource_id) {
        const ptr0 = passStringToWasm0(resource_type, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(resource_id, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        const ret = wasm.wasmgrantservice_list(this.__wbg_ptr, ptr0, len0, ptr1, len1);
        return ret;
    }
    /**
     * @param {string} resource_type
     * @param {string} resource_id
     * @param {bigint} grant_id
     * @returns {Promise<void>}
     */
    revoke(resource_type, resource_id, grant_id) {
        const ptr0 = passStringToWasm0(resource_type, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(resource_id, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        const ret = wasm.wasmgrantservice_revoke(this.__wbg_ptr, ptr0, len0, ptr1, len1, grant_id);
        return ret;
    }
}
if (Symbol.dispose) WasmGrantService.prototype[Symbol.dispose] = WasmGrantService.prototype.free;

export class WasmInvitationService {
    static __wrap(ptr) {
        ptr = ptr >>> 0;
        const obj = Object.create(WasmInvitationService.prototype);
        obj.__wbg_ptr = ptr;
        WasmInvitationServiceFinalization.register(obj, obj.__wbg_ptr, obj);
        return obj;
    }
    __destroy_into_raw() {
        const ptr = this.__wbg_ptr;
        this.__wbg_ptr = 0;
        WasmInvitationServiceFinalization.unregister(this);
        return ptr;
    }
    free() {
        const ptr = this.__destroy_into_raw();
        wasm.__wbg_wasminvitationservice_free(ptr, 0);
    }
    /**
     * @param {string} token
     * @returns {Promise<void>}
     */
    accept(token) {
        const ptr0 = passStringToWasm0(token, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasminvitationservice_accept(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {string} json
     * @returns {Promise<string>}
     */
    create(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasminvitationservice_create(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {string} token
     * @returns {Promise<string>}
     */
    get_by_token(token) {
        const ptr0 = passStringToWasm0(token, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasminvitationservice_get_by_token(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @returns {Promise<string>}
     */
    list() {
        const ret = wasm.wasminvitationservice_list(this.__wbg_ptr);
        return ret;
    }
    /**
     * @returns {Promise<string>}
     */
    list_pending() {
        const ret = wasm.wasminvitationservice_list_pending(this.__wbg_ptr);
        return ret;
    }
    /**
     * @param {bigint} id
     * @returns {Promise<void>}
     */
    resend(id) {
        const ret = wasm.wasminvitationservice_resend(this.__wbg_ptr, id);
        return ret;
    }
    /**
     * @param {bigint} id
     * @returns {Promise<void>}
     */
    revoke(id) {
        const ret = wasm.wasminvitationservice_revoke(this.__wbg_ptr, id);
        return ret;
    }
}
if (Symbol.dispose) WasmInvitationService.prototype[Symbol.dispose] = WasmInvitationService.prototype.free;

export class WasmLoopService {
    static __wrap(ptr) {
        ptr = ptr >>> 0;
        const obj = Object.create(WasmLoopService.prototype);
        obj.__wbg_ptr = ptr;
        WasmLoopServiceFinalization.register(obj, obj.__wbg_ptr, obj);
        return obj;
    }
    __destroy_into_raw() {
        const ptr = this.__wbg_ptr;
        this.__wbg_ptr = 0;
        WasmLoopServiceFinalization.unregister(this);
        return ptr;
    }
    free() {
        const ptr = this.__destroy_into_raw();
        wasm.__wbg_wasmloopservice_free(ptr, 0);
    }
    /**
     * @param {string} json
     */
    add_run(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmloopservice_add_run(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {string} json
     */
    append_runs(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmloopservice_append_runs(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {string} slug
     * @param {bigint} run_id
     * @returns {Promise<void>}
     */
    cancel_run(slug, run_id) {
        const ptr0 = passStringToWasm0(slug, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmloopservice_cancel_run(this.__wbg_ptr, ptr0, len0, run_id);
        return ret;
    }
    clear_runs() {
        wasm.wasmloopservice_clear_runs(this.__wbg_ptr);
    }
    /**
     * @param {string} request_json
     * @returns {Promise<string>}
     */
    create_loop(request_json) {
        const ptr0 = passStringToWasm0(request_json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmloopservice_create_loop(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @returns {any}
     */
    current_loop_json() {
        const ret = wasm.wasmloopservice_current_loop_json(this.__wbg_ptr);
        return ret;
    }
    /**
     * @param {string} slug
     * @returns {Promise<void>}
     */
    delete_loop(slug) {
        const ptr0 = passStringToWasm0(slug, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmloopservice_delete_loop(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {string} slug
     * @returns {Promise<string>}
     */
    disable_loop(slug) {
        const ptr0 = passStringToWasm0(slug, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmloopservice_disable_loop(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {string} slug
     * @returns {Promise<string>}
     */
    enable_loop(slug) {
        const ptr0 = passStringToWasm0(slug, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmloopservice_enable_loop(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {string} slug
     * @returns {Promise<string>}
     */
    fetch_loop(slug) {
        const ptr0 = passStringToWasm0(slug, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmloopservice_fetch_loop(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {string | null} [status]
     * @param {number | null} [limit]
     * @param {number | null} [offset]
     * @returns {Promise<string>}
     */
    fetch_loops(status, limit, offset) {
        var ptr0 = isLikeNone(status) ? 0 : passStringToWasm0(status, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        var len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmloopservice_fetch_loops(this.__wbg_ptr, ptr0, len0, isLikeNone(limit) ? 0x100000001 : (limit) >>> 0, isLikeNone(offset) ? 0x100000001 : (offset) >>> 0);
        return ret;
    }
    /**
     * @param {string} slug
     * @param {string | null} [status]
     * @param {number | null} [limit]
     * @param {number | null} [offset]
     * @returns {Promise<string>}
     */
    fetch_runs(slug, status, limit, offset) {
        const ptr0 = passStringToWasm0(slug, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        var ptr1 = isLikeNone(status) ? 0 : passStringToWasm0(status, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        var len1 = WASM_VECTOR_LEN;
        const ret = wasm.wasmloopservice_fetch_runs(this.__wbg_ptr, ptr0, len0, ptr1, len1, isLikeNone(limit) ? 0x100000001 : (limit) >>> 0, isLikeNone(offset) ? 0x100000001 : (offset) >>> 0);
        return ret;
    }
    /**
     * @param {string} slug
     * @returns {any}
     */
    get_loop_by_slug_json(slug) {
        const ptr0 = passStringToWasm0(slug, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmloopservice_get_loop_by_slug_json(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @returns {string}
     */
    loops_json() {
        let deferred1_0;
        let deferred1_1;
        try {
            const ret = wasm.wasmloopservice_loops_json(this.__wbg_ptr);
            deferred1_0 = ret[0];
            deferred1_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred1_0, deferred1_1, 1);
        }
    }
    /**
     * @returns {string}
     */
    runs_json() {
        let deferred1_0;
        let deferred1_1;
        try {
            const ret = wasm.wasmloopservice_runs_json(this.__wbg_ptr);
            deferred1_0 = ret[0];
            deferred1_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred1_0, deferred1_1, 1);
        }
    }
    /**
     * @param {string} json
     */
    set_current_loop(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmloopservice_set_current_loop(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {string} json
     */
    set_loops(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmloopservice_set_loops(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {string} json
     */
    set_runs(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmloopservice_set_runs(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {string} slug
     * @returns {Promise<string>}
     */
    trigger_loop(slug) {
        const ptr0 = passStringToWasm0(slug, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmloopservice_trigger_loop(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {string} slug
     * @param {string} request_json
     * @returns {Promise<string>}
     */
    update_loop(slug, request_json) {
        const ptr0 = passStringToWasm0(slug, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(request_json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        const ret = wasm.wasmloopservice_update_loop(this.__wbg_ptr, ptr0, len0, ptr1, len1);
        return ret;
    }
    /**
     * @param {string} slug
     * @param {string} json
     */
    update_loop_local(slug, json) {
        const ptr0 = passStringToWasm0(slug, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        wasm.wasmloopservice_update_loop_local(this.__wbg_ptr, ptr0, len0, ptr1, len1);
    }
    /**
     * @param {bigint} run_id
     * @param {string} status
     */
    update_run_status(run_id, status) {
        const ptr0 = passStringToWasm0(status, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmloopservice_update_run_status(this.__wbg_ptr, run_id, ptr0, len0);
    }
}
if (Symbol.dispose) WasmLoopService.prototype[Symbol.dispose] = WasmLoopService.prototype.free;

export class WasmLoopState {
    __destroy_into_raw() {
        const ptr = this.__wbg_ptr;
        this.__wbg_ptr = 0;
        WasmLoopStateFinalization.unregister(this);
        return ptr;
    }
    free() {
        const ptr = this.__destroy_into_raw();
        wasm.__wbg_wasmloopstate_free(ptr, 0);
    }
    /**
     * @param {string} run_json
     */
    add_run(run_json) {
        const ptr0 = passStringToWasm0(run_json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmloopstate_add_run(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {string} json
     */
    append_runs(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmloopstate_append_runs(this.__wbg_ptr, ptr0, len0);
    }
    clear_runs() {
        wasm.wasmloopstate_clear_runs(this.__wbg_ptr);
    }
    /**
     * @returns {any}
     */
    current_loop_json() {
        const ret = wasm.wasmloopstate_current_loop_json(this.__wbg_ptr);
        return ret;
    }
    /**
     * @param {string} slug
     * @returns {any}
     */
    get_loop_by_slug_json(slug) {
        const ptr0 = passStringToWasm0(slug, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmloopstate_get_loop_by_slug_json(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @returns {string}
     */
    loops_json() {
        let deferred1_0;
        let deferred1_1;
        try {
            const ret = wasm.wasmloopstate_loops_json(this.__wbg_ptr);
            deferred1_0 = ret[0];
            deferred1_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred1_0, deferred1_1, 1);
        }
    }
    constructor() {
        const ret = wasm.wasmloopstate_new();
        this.__wbg_ptr = ret >>> 0;
        WasmLoopStateFinalization.register(this, this.__wbg_ptr, this);
        return this;
    }
    /**
     * @returns {string}
     */
    runs_json() {
        let deferred1_0;
        let deferred1_1;
        try {
            const ret = wasm.wasmloopstate_runs_json(this.__wbg_ptr);
            deferred1_0 = ret[0];
            deferred1_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred1_0, deferred1_1, 1);
        }
    }
    /**
     * @param {string} json
     */
    set_current_loop(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmloopstate_set_current_loop(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {string} json
     */
    set_loops(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmloopstate_set_loops(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {string} json
     */
    set_runs(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmloopstate_set_runs(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {string} slug
     * @param {string} json
     */
    update_loop(slug, json) {
        const ptr0 = passStringToWasm0(slug, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        wasm.wasmloopstate_update_loop(this.__wbg_ptr, ptr0, len0, ptr1, len1);
    }
    /**
     * @param {bigint} run_id
     * @param {string} status
     */
    update_run_status(run_id, status) {
        const ptr0 = passStringToWasm0(status, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmloopstate_update_run_status(this.__wbg_ptr, run_id, ptr0, len0);
    }
}
if (Symbol.dispose) WasmLoopState.prototype[Symbol.dispose] = WasmLoopState.prototype.free;

export class WasmMeshService {
    static __wrap(ptr) {
        ptr = ptr >>> 0;
        const obj = Object.create(WasmMeshService.prototype);
        obj.__wbg_ptr = ptr;
        WasmMeshServiceFinalization.register(obj, obj.__wbg_ptr, obj);
        return obj;
    }
    __destroy_into_raw() {
        const ptr = this.__wbg_ptr;
        this.__wbg_ptr = 0;
        WasmMeshServiceFinalization.unregister(this);
        return ptr;
    }
    free() {
        const ptr = this.__destroy_into_raw();
        wasm.__wbg_wasmmeshservice_free(ptr, 0);
    }
    clear_topology() {
        wasm.wasmmeshservice_clear_topology(this.__wbg_ptr);
    }
    /**
     * @returns {Promise<string>}
     */
    fetch_topology() {
        const ret = wasm.wasmmeshservice_fetch_topology(this.__wbg_ptr);
        return ret;
    }
    /**
     * @returns {string}
     */
    get_active_nodes_json() {
        let deferred1_0;
        let deferred1_1;
        try {
            const ret = wasm.wasmmeshservice_get_active_nodes_json(this.__wbg_ptr);
            deferred1_0 = ret[0];
            deferred1_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred1_0, deferred1_1, 1);
        }
    }
    /**
     * @param {string} pod_key
     * @returns {string}
     */
    get_channels_for_node_json(pod_key) {
        let deferred2_0;
        let deferred2_1;
        try {
            const ptr0 = passStringToWasm0(pod_key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
            const len0 = WASM_VECTOR_LEN;
            const ret = wasm.wasmmeshservice_get_channels_for_node_json(this.__wbg_ptr, ptr0, len0);
            deferred2_0 = ret[0];
            deferred2_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred2_0, deferred2_1, 1);
        }
    }
    /**
     * @param {string} pod_key
     * @returns {string}
     */
    get_edges_for_node_json(pod_key) {
        let deferred2_0;
        let deferred2_1;
        try {
            const ptr0 = passStringToWasm0(pod_key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
            const len0 = WASM_VECTOR_LEN;
            const ret = wasm.wasmmeshservice_get_edges_for_node_json(this.__wbg_ptr, ptr0, len0);
            deferred2_0 = ret[0];
            deferred2_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred2_0, deferred2_1, 1);
        }
    }
    /**
     * @param {string} pod_key
     * @returns {any}
     */
    get_node_json(pod_key) {
        const ptr0 = passStringToWasm0(pod_key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmmeshservice_get_node_json(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {bigint} runner_id
     * @returns {string}
     */
    get_nodes_by_runner_json(runner_id) {
        let deferred1_0;
        let deferred1_1;
        try {
            const ret = wasm.wasmmeshservice_get_nodes_by_runner_json(this.__wbg_ptr, runner_id);
            deferred1_0 = ret[0];
            deferred1_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred1_0, deferred1_1, 1);
        }
    }
    /**
     * @param {bigint} runner_id
     * @returns {any}
     */
    get_runner_info_json(runner_id) {
        const ret = wasm.wasmmeshservice_get_runner_info_json(this.__wbg_ptr, runner_id);
        return ret;
    }
    /**
     * @param {string | null} [pod_key]
     */
    select_node(pod_key) {
        var ptr0 = isLikeNone(pod_key) ? 0 : passStringToWasm0(pod_key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        var len0 = WASM_VECTOR_LEN;
        wasm.wasmmeshservice_select_node(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @returns {any}
     */
    selected_node() {
        const ret = wasm.wasmmeshservice_selected_node(this.__wbg_ptr);
        return ret;
    }
    /**
     * @param {string} json
     */
    set_topology(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmmeshservice_set_topology(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @returns {any}
     */
    topology_json() {
        const ret = wasm.wasmmeshservice_topology_json(this.__wbg_ptr);
        return ret;
    }
}
if (Symbol.dispose) WasmMeshService.prototype[Symbol.dispose] = WasmMeshService.prototype.free;

export class WasmMeshState {
    __destroy_into_raw() {
        const ptr = this.__wbg_ptr;
        this.__wbg_ptr = 0;
        WasmMeshStateFinalization.unregister(this);
        return ptr;
    }
    free() {
        const ptr = this.__destroy_into_raw();
        wasm.__wbg_wasmmeshstate_free(ptr, 0);
    }
    clear_topology() {
        wasm.wasmmeshstate_clear_topology(this.__wbg_ptr);
    }
    /**
     * @returns {string}
     */
    get_active_nodes_json() {
        let deferred1_0;
        let deferred1_1;
        try {
            const ret = wasm.wasmmeshstate_get_active_nodes_json(this.__wbg_ptr);
            deferred1_0 = ret[0];
            deferred1_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred1_0, deferred1_1, 1);
        }
    }
    /**
     * @param {string} pod_key
     * @returns {string}
     */
    get_channels_for_node_json(pod_key) {
        let deferred2_0;
        let deferred2_1;
        try {
            const ptr0 = passStringToWasm0(pod_key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
            const len0 = WASM_VECTOR_LEN;
            const ret = wasm.wasmmeshstate_get_channels_for_node_json(this.__wbg_ptr, ptr0, len0);
            deferred2_0 = ret[0];
            deferred2_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred2_0, deferred2_1, 1);
        }
    }
    /**
     * @param {string} pod_key
     * @returns {string}
     */
    get_edges_for_node_json(pod_key) {
        let deferred2_0;
        let deferred2_1;
        try {
            const ptr0 = passStringToWasm0(pod_key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
            const len0 = WASM_VECTOR_LEN;
            const ret = wasm.wasmmeshstate_get_edges_for_node_json(this.__wbg_ptr, ptr0, len0);
            deferred2_0 = ret[0];
            deferred2_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred2_0, deferred2_1, 1);
        }
    }
    /**
     * @param {string} pod_key
     * @returns {any}
     */
    get_node_json(pod_key) {
        const ptr0 = passStringToWasm0(pod_key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmmeshstate_get_node_json(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {bigint} runner_id
     * @returns {string}
     */
    get_nodes_by_runner_json(runner_id) {
        let deferred1_0;
        let deferred1_1;
        try {
            const ret = wasm.wasmmeshstate_get_nodes_by_runner_json(this.__wbg_ptr, runner_id);
            deferred1_0 = ret[0];
            deferred1_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred1_0, deferred1_1, 1);
        }
    }
    /**
     * @param {bigint} runner_id
     * @returns {any}
     */
    get_runner_info_json(runner_id) {
        const ret = wasm.wasmmeshstate_get_runner_info_json(this.__wbg_ptr, runner_id);
        return ret;
    }
    constructor() {
        const ret = wasm.wasmmeshstate_new();
        this.__wbg_ptr = ret >>> 0;
        WasmMeshStateFinalization.register(this, this.__wbg_ptr, this);
        return this;
    }
    /**
     * @param {string | null} [pod_key]
     */
    select_node(pod_key) {
        var ptr0 = isLikeNone(pod_key) ? 0 : passStringToWasm0(pod_key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        var len0 = WASM_VECTOR_LEN;
        wasm.wasmmeshstate_select_node(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @returns {any}
     */
    selected_node() {
        const ret = wasm.wasmmeshstate_selected_node(this.__wbg_ptr);
        return ret;
    }
    /**
     * @param {string} json
     */
    set_topology(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmmeshstate_set_topology(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @returns {any}
     */
    topology_json() {
        const ret = wasm.wasmmeshstate_topology_json(this.__wbg_ptr);
        return ret;
    }
}
if (Symbol.dispose) WasmMeshState.prototype[Symbol.dispose] = WasmMeshState.prototype.free;

export class WasmMessageService {
    static __wrap(ptr) {
        ptr = ptr >>> 0;
        const obj = Object.create(WasmMessageService.prototype);
        obj.__wbg_ptr = ptr;
        WasmMessageServiceFinalization.register(obj, obj.__wbg_ptr, obj);
        return obj;
    }
    __destroy_into_raw() {
        const ptr = this.__wbg_ptr;
        this.__wbg_ptr = 0;
        WasmMessageServiceFinalization.unregister(this);
        return ptr;
    }
    free() {
        const ptr = this.__destroy_into_raw();
        wasm.__wbg_wasmmessageservice_free(ptr, 0);
    }
    /**
     * @param {string} correlation_id
     * @param {number | null} [limit]
     * @returns {Promise<string>}
     */
    get_conversation(correlation_id, limit) {
        const ptr0 = passStringToWasm0(correlation_id, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmmessageservice_get_conversation(this.__wbg_ptr, ptr0, len0, isLikeNone(limit) ? 0x100000001 : (limit) >>> 0);
        return ret;
    }
    /**
     * @param {number | null} [limit]
     * @param {number | null} [offset]
     * @returns {Promise<string>}
     */
    get_dead_letters(limit, offset) {
        const ret = wasm.wasmmessageservice_get_dead_letters(this.__wbg_ptr, isLikeNone(limit) ? 0x100000001 : (limit) >>> 0, isLikeNone(offset) ? 0x100000001 : (offset) >>> 0);
        return ret;
    }
    /**
     * @param {bigint} id
     * @returns {Promise<string>}
     */
    get_message(id) {
        const ret = wasm.wasmmessageservice_get_message(this.__wbg_ptr, id);
        return ret;
    }
    /**
     * @param {boolean | null} [unread_only]
     * @param {number | null} [limit]
     * @param {number | null} [offset]
     * @returns {Promise<string>}
     */
    get_messages(unread_only, limit, offset) {
        const ret = wasm.wasmmessageservice_get_messages(this.__wbg_ptr, isLikeNone(unread_only) ? 0xFFFFFF : unread_only ? 1 : 0, isLikeNone(limit) ? 0x100000001 : (limit) >>> 0, isLikeNone(offset) ? 0x100000001 : (offset) >>> 0);
        return ret;
    }
    /**
     * @param {number | null} [limit]
     * @param {number | null} [offset]
     * @returns {Promise<string>}
     */
    get_sent_messages(limit, offset) {
        const ret = wasm.wasmmessageservice_get_sent_messages(this.__wbg_ptr, isLikeNone(limit) ? 0x100000001 : (limit) >>> 0, isLikeNone(offset) ? 0x100000001 : (offset) >>> 0);
        return ret;
    }
    /**
     * @returns {Promise<string>}
     */
    get_unread_count() {
        const ret = wasm.wasmmessageservice_get_unread_count(this.__wbg_ptr);
        return ret;
    }
    /**
     * @returns {Promise<void>}
     */
    mark_all_read() {
        const ret = wasm.wasmmessageservice_mark_all_read(this.__wbg_ptr);
        return ret;
    }
    /**
     * @param {string} json
     * @returns {Promise<void>}
     */
    mark_read(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmmessageservice_mark_read(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {bigint} entry_id
     * @returns {Promise<void>}
     */
    replay_dead_letter(entry_id) {
        const ret = wasm.wasmmessageservice_replay_dead_letter(this.__wbg_ptr, entry_id);
        return ret;
    }
    /**
     * @param {string} json
     * @param {string | null} [pod_key]
     * @returns {Promise<string>}
     */
    send_message(json, pod_key) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        var ptr1 = isLikeNone(pod_key) ? 0 : passStringToWasm0(pod_key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        var len1 = WASM_VECTOR_LEN;
        const ret = wasm.wasmmessageservice_send_message(this.__wbg_ptr, ptr0, len0, ptr1, len1);
        return ret;
    }
}
if (Symbol.dispose) WasmMessageService.prototype[Symbol.dispose] = WasmMessageService.prototype.free;

export class WasmNotificationService {
    static __wrap(ptr) {
        ptr = ptr >>> 0;
        const obj = Object.create(WasmNotificationService.prototype);
        obj.__wbg_ptr = ptr;
        WasmNotificationServiceFinalization.register(obj, obj.__wbg_ptr, obj);
        return obj;
    }
    __destroy_into_raw() {
        const ptr = this.__wbg_ptr;
        this.__wbg_ptr = 0;
        WasmNotificationServiceFinalization.unregister(this);
        return ptr;
    }
    free() {
        const ptr = this.__destroy_into_raw();
        wasm.__wbg_wasmnotificationservice_free(ptr, 0);
    }
    /**
     * @returns {Promise<string>}
     */
    get_preferences() {
        const ret = wasm.wasmnotificationservice_get_preferences(this.__wbg_ptr);
        return ret;
    }
    /**
     * @param {string} json
     * @returns {Promise<string>}
     */
    set_preference(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmnotificationservice_set_preference(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
}
if (Symbol.dispose) WasmNotificationService.prototype[Symbol.dispose] = WasmNotificationService.prototype.free;

export class WasmOrgApiService {
    static __wrap(ptr) {
        ptr = ptr >>> 0;
        const obj = Object.create(WasmOrgApiService.prototype);
        obj.__wbg_ptr = ptr;
        WasmOrgApiServiceFinalization.register(obj, obj.__wbg_ptr, obj);
        return obj;
    }
    __destroy_into_raw() {
        const ptr = this.__wbg_ptr;
        this.__wbg_ptr = 0;
        WasmOrgApiServiceFinalization.unregister(this);
        return ptr;
    }
    free() {
        const ptr = this.__destroy_into_raw();
        wasm.__wbg_wasmorgapiservice_free(ptr, 0);
    }
    /**
     * @param {string} json
     * @returns {Promise<string>}
     */
    create(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmorgapiservice_create(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {string} slug
     * @returns {Promise<void>}
     */
    delete(slug) {
        const ptr0 = passStringToWasm0(slug, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmorgapiservice_delete(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {string} slug
     * @returns {Promise<string>}
     */
    get(slug) {
        const ptr0 = passStringToWasm0(slug, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmorgapiservice_get(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {string} slug
     * @param {string} json
     * @returns {Promise<string>}
     */
    invite_member(slug, json) {
        const ptr0 = passStringToWasm0(slug, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        const ret = wasm.wasmorgapiservice_invite_member(this.__wbg_ptr, ptr0, len0, ptr1, len1);
        return ret;
    }
    /**
     * @returns {Promise<string>}
     */
    list() {
        const ret = wasm.wasmorgapiservice_list(this.__wbg_ptr);
        return ret;
    }
    /**
     * @param {string} slug
     * @returns {Promise<string>}
     */
    list_members(slug) {
        const ptr0 = passStringToWasm0(slug, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmorgapiservice_list_members(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {string} slug
     * @param {bigint} user_id
     * @returns {Promise<void>}
     */
    remove_member(slug, user_id) {
        const ptr0 = passStringToWasm0(slug, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmorgapiservice_remove_member(this.__wbg_ptr, ptr0, len0, user_id);
        return ret;
    }
    /**
     * @param {string} slug
     * @param {string} json
     * @returns {Promise<string>}
     */
    update(slug, json) {
        const ptr0 = passStringToWasm0(slug, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        const ret = wasm.wasmorgapiservice_update(this.__wbg_ptr, ptr0, len0, ptr1, len1);
        return ret;
    }
    /**
     * @param {string} slug
     * @param {bigint} user_id
     * @param {string} json
     * @returns {Promise<string>}
     */
    update_member_role(slug, user_id, json) {
        const ptr0 = passStringToWasm0(slug, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        const ret = wasm.wasmorgapiservice_update_member_role(this.__wbg_ptr, ptr0, len0, user_id, ptr1, len1);
        return ret;
    }
}
if (Symbol.dispose) WasmOrgApiService.prototype[Symbol.dispose] = WasmOrgApiService.prototype.free;

export class WasmOrgState {
    __destroy_into_raw() {
        const ptr = this.__wbg_ptr;
        this.__wbg_ptr = 0;
        WasmOrgStateFinalization.unregister(this);
        return ptr;
    }
    free() {
        const ptr = this.__destroy_into_raw();
        wasm.__wbg_wasmorgstate_free(ptr, 0);
    }
    /**
     * @param {string} json
     */
    add_member(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmorgstate_add_member(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {string} json
     */
    add_organization(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmorgstate_add_organization(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @returns {any}
     */
    current_org_json() {
        const ret = wasm.wasmorgstate_current_org_json(this.__wbg_ptr);
        return ret;
    }
    /**
     * @returns {string}
     */
    members_json() {
        let deferred1_0;
        let deferred1_1;
        try {
            const ret = wasm.wasmorgstate_members_json(this.__wbg_ptr);
            deferred1_0 = ret[0];
            deferred1_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred1_0, deferred1_1, 1);
        }
    }
    constructor() {
        const ret = wasm.wasmorgstate_new();
        this.__wbg_ptr = ret >>> 0;
        WasmOrgStateFinalization.register(this, this.__wbg_ptr, this);
        return this;
    }
    /**
     * @returns {string}
     */
    organizations_json() {
        let deferred1_0;
        let deferred1_1;
        try {
            const ret = wasm.wasmorgstate_organizations_json(this.__wbg_ptr);
            deferred1_0 = ret[0];
            deferred1_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred1_0, deferred1_1, 1);
        }
    }
    /**
     * @param {string} id
     */
    remove_member(id) {
        const ptr0 = passStringToWasm0(id, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmorgstate_remove_member(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {number} id
     */
    remove_organization(id) {
        wasm.wasmorgstate_remove_organization(this.__wbg_ptr, id);
    }
    /**
     * @param {string} json
     */
    set_current_org(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmorgstate_set_current_org(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {string} json
     */
    set_members(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmorgstate_set_members(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {string} json
     */
    set_organizations(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmorgstate_set_organizations(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {number} user_id
     * @param {string} json
     */
    update_member(user_id, json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmorgstate_update_member(this.__wbg_ptr, user_id, ptr0, len0);
    }
    /**
     * @param {number} id
     * @param {string} json
     */
    update_organization(id, json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmorgstate_update_organization(this.__wbg_ptr, id, ptr0, len0);
    }
}
if (Symbol.dispose) WasmOrgState.prototype[Symbol.dispose] = WasmOrgState.prototype.free;

export class WasmPodService {
    static __wrap(ptr) {
        ptr = ptr >>> 0;
        const obj = Object.create(WasmPodService.prototype);
        obj.__wbg_ptr = ptr;
        WasmPodServiceFinalization.register(obj, obj.__wbg_ptr, obj);
        return obj;
    }
    __destroy_into_raw() {
        const ptr = this.__wbg_ptr;
        this.__wbg_ptr = 0;
        WasmPodServiceFinalization.unregister(this);
        return ptr;
    }
    free() {
        const ptr = this.__destroy_into_raw();
        wasm.__wbg_wasmpodservice_free(ptr, 0);
    }
    /**
     * @param {string} request_json
     * @returns {Promise<string>}
     */
    create_pod(request_json) {
        const ptr0 = passStringToWasm0(request_json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmpodservice_create_pod(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @returns {any}
     */
    current_pod_json() {
        const ret = wasm.wasmpodservice_current_pod_json(this.__wbg_ptr);
        return ret;
    }
    /**
     * @param {string} pod_key
     * @returns {Promise<string>}
     */
    fetch_pod(pod_key) {
        const ptr0 = passStringToWasm0(pod_key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmpodservice_fetch_pod(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {string | null} [status]
     * @param {bigint | null} [runner_id]
     * @param {bigint | null} [created_by_id]
     * @param {bigint | null} [limit]
     * @param {bigint | null} [offset]
     * @returns {Promise<string>}
     */
    fetch_pods(status, runner_id, created_by_id, limit, offset) {
        var ptr0 = isLikeNone(status) ? 0 : passStringToWasm0(status, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        var len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmpodservice_fetch_pods(this.__wbg_ptr, ptr0, len0, !isLikeNone(runner_id), isLikeNone(runner_id) ? BigInt(0) : runner_id, !isLikeNone(created_by_id), isLikeNone(created_by_id) ? BigInt(0) : created_by_id, !isLikeNone(limit), isLikeNone(limit) ? BigInt(0) : limit, !isLikeNone(offset), isLikeNone(offset) ? BigInt(0) : offset);
        return ret;
    }
    /**
     * @param {string} filter
     * @param {bigint | null} [user_id]
     * @returns {Promise<string>}
     */
    fetch_sidebar_pods(filter, user_id) {
        const ptr0 = passStringToWasm0(filter, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmpodservice_fetch_sidebar_pods(this.__wbg_ptr, ptr0, len0, !isLikeNone(user_id), isLikeNone(user_id) ? BigInt(0) : user_id);
        return ret;
    }
    /**
     * @param {string} pod_key
     * @returns {Promise<string>}
     */
    get_pod_connection(pod_key) {
        const ptr0 = passStringToWasm0(pod_key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmpodservice_get_pod_connection(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {string} pod_key
     * @returns {any}
     */
    get_pod_json(pod_key) {
        const ptr0 = passStringToWasm0(pod_key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmpodservice_get_pod_json(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {string} filter
     * @param {bigint | null | undefined} user_id
     * @param {bigint} offset
     * @returns {Promise<string>}
     */
    load_more_pods(filter, user_id, offset) {
        const ptr0 = passStringToWasm0(filter, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmpodservice_load_more_pods(this.__wbg_ptr, ptr0, len0, !isLikeNone(user_id), isLikeNone(user_id) ? BigInt(0) : user_id, offset);
        return ret;
    }
    /**
     * @returns {string}
     */
    pods_json() {
        let deferred1_0;
        let deferred1_1;
        try {
            const ret = wasm.wasmpodservice_pods_json(this.__wbg_ptr);
            deferred1_0 = ret[0];
            deferred1_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred1_0, deferred1_1, 1);
        }
    }
    /**
     * @param {string} pod_key
     */
    remove_pod(pod_key) {
        const ptr0 = passStringToWasm0(pod_key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmpodservice_remove_pod(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {string} pod_json
     */
    set_current_pod(pod_json) {
        const ptr0 = passStringToWasm0(pod_json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmpodservice_set_current_pod(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {string} pods_json
     */
    set_pods(pods_json) {
        const ptr0 = passStringToWasm0(pods_json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmpodservice_set_pods(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {string} pod_key
     * @returns {Promise<void>}
     */
    terminate_pod(pod_key) {
        const ptr0 = passStringToWasm0(pod_key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmpodservice_terminate_pod(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {string} pod_key
     * @param {string} agent_status
     */
    update_agent_status(pod_key, agent_status) {
        const ptr0 = passStringToWasm0(pod_key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(agent_status, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        wasm.wasmpodservice_update_agent_status(this.__wbg_ptr, ptr0, len0, ptr1, len1);
    }
    /**
     * @param {string} pod_key
     * @param {string} alias
     */
    update_pod_alias(pod_key, alias) {
        const ptr0 = passStringToWasm0(pod_key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(alias, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        wasm.wasmpodservice_update_pod_alias(this.__wbg_ptr, ptr0, len0, ptr1, len1);
    }
    /**
     * @param {string} pod_key
     * @param {string | null} [alias]
     * @returns {Promise<void>}
     */
    update_pod_alias_api(pod_key, alias) {
        const ptr0 = passStringToWasm0(pod_key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        var ptr1 = isLikeNone(alias) ? 0 : passStringToWasm0(alias, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        var len1 = WASM_VECTOR_LEN;
        const ret = wasm.wasmpodservice_update_pod_alias_api(this.__wbg_ptr, ptr0, len0, ptr1, len1);
        return ret;
    }
    /**
     * @param {string} pod_key
     * @param {string} status
     * @param {string | null} [agent_status]
     * @param {string | null} [error_code]
     * @param {string | null} [error_message]
     * @param {bigint | null} [timestamp]
     */
    update_pod_status(pod_key, status, agent_status, error_code, error_message, timestamp) {
        const ptr0 = passStringToWasm0(pod_key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(status, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        var ptr2 = isLikeNone(agent_status) ? 0 : passStringToWasm0(agent_status, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        var len2 = WASM_VECTOR_LEN;
        var ptr3 = isLikeNone(error_code) ? 0 : passStringToWasm0(error_code, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        var len3 = WASM_VECTOR_LEN;
        var ptr4 = isLikeNone(error_message) ? 0 : passStringToWasm0(error_message, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        var len4 = WASM_VECTOR_LEN;
        wasm.wasmpodservice_update_pod_status(this.__wbg_ptr, ptr0, len0, ptr1, len1, ptr2, len2, ptr3, len3, ptr4, len4, !isLikeNone(timestamp), isLikeNone(timestamp) ? BigInt(0) : timestamp);
    }
    /**
     * @param {string} pod_key
     * @param {string} title
     * @param {bigint | null} [timestamp]
     */
    update_pod_title(pod_key, title, timestamp) {
        const ptr0 = passStringToWasm0(pod_key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(title, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        wasm.wasmpodservice_update_pod_title(this.__wbg_ptr, ptr0, len0, ptr1, len1, !isLikeNone(timestamp), isLikeNone(timestamp) ? BigInt(0) : timestamp);
    }
    /**
     * @param {string} pod_json
     * @param {bigint | null} [timestamp]
     */
    upsert_pod(pod_json, timestamp) {
        const ptr0 = passStringToWasm0(pod_json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmpodservice_upsert_pod(this.__wbg_ptr, ptr0, len0, !isLikeNone(timestamp), isLikeNone(timestamp) ? BigInt(0) : timestamp);
    }
}
if (Symbol.dispose) WasmPodService.prototype[Symbol.dispose] = WasmPodService.prototype.free;

export class WasmPodState {
    __destroy_into_raw() {
        const ptr = this.__wbg_ptr;
        this.__wbg_ptr = 0;
        WasmPodStateFinalization.unregister(this);
        return ptr;
    }
    free() {
        const ptr = this.__destroy_into_raw();
        wasm.__wbg_wasmpodstate_free(ptr, 0);
    }
    /**
     * @returns {any}
     */
    current_pod_json() {
        const ret = wasm.wasmpodstate_current_pod_json(this.__wbg_ptr);
        return ret;
    }
    /**
     * @param {string} pod_key
     * @returns {any}
     */
    get_pod_json(pod_key) {
        const ptr0 = passStringToWasm0(pod_key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmpodstate_get_pod_json(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    constructor() {
        const ret = wasm.wasmpodstate_new();
        this.__wbg_ptr = ret >>> 0;
        WasmPodStateFinalization.register(this, this.__wbg_ptr, this);
        return this;
    }
    /**
     * @returns {string}
     */
    pods_json() {
        let deferred1_0;
        let deferred1_1;
        try {
            const ret = wasm.wasmpodstate_pods_json(this.__wbg_ptr);
            deferred1_0 = ret[0];
            deferred1_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred1_0, deferred1_1, 1);
        }
    }
    /**
     * @param {string} pod_key
     */
    remove_pod(pod_key) {
        const ptr0 = passStringToWasm0(pod_key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmpodstate_remove_pod(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {string} pod_json
     */
    set_current_pod(pod_json) {
        const ptr0 = passStringToWasm0(pod_json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmpodstate_set_current_pod(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {string} pods_json
     */
    set_pods(pods_json) {
        const ptr0 = passStringToWasm0(pods_json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmpodstate_set_pods(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {string} pod_key
     * @param {string} agent_status
     */
    update_agent_status(pod_key, agent_status) {
        const ptr0 = passStringToWasm0(pod_key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(agent_status, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        wasm.wasmpodstate_update_agent_status(this.__wbg_ptr, ptr0, len0, ptr1, len1);
    }
    /**
     * @param {string} pod_key
     * @param {string} alias
     */
    update_pod_alias(pod_key, alias) {
        const ptr0 = passStringToWasm0(pod_key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(alias, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        wasm.wasmpodstate_update_pod_alias(this.__wbg_ptr, ptr0, len0, ptr1, len1);
    }
    /**
     * @param {string} pod_key
     * @param {string} status
     * @param {string | null} [agent_status]
     * @param {string | null} [error_code]
     * @param {string | null} [error_message]
     * @param {bigint | null} [timestamp]
     */
    update_pod_status(pod_key, status, agent_status, error_code, error_message, timestamp) {
        const ptr0 = passStringToWasm0(pod_key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(status, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        var ptr2 = isLikeNone(agent_status) ? 0 : passStringToWasm0(agent_status, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        var len2 = WASM_VECTOR_LEN;
        var ptr3 = isLikeNone(error_code) ? 0 : passStringToWasm0(error_code, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        var len3 = WASM_VECTOR_LEN;
        var ptr4 = isLikeNone(error_message) ? 0 : passStringToWasm0(error_message, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        var len4 = WASM_VECTOR_LEN;
        wasm.wasmpodstate_update_pod_status(this.__wbg_ptr, ptr0, len0, ptr1, len1, ptr2, len2, ptr3, len3, ptr4, len4, !isLikeNone(timestamp), isLikeNone(timestamp) ? BigInt(0) : timestamp);
    }
    /**
     * @param {string} pod_key
     * @param {string} title
     * @param {bigint | null} [timestamp]
     */
    update_pod_title(pod_key, title, timestamp) {
        const ptr0 = passStringToWasm0(pod_key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(title, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        wasm.wasmpodstate_update_pod_title(this.__wbg_ptr, ptr0, len0, ptr1, len1, !isLikeNone(timestamp), isLikeNone(timestamp) ? BigInt(0) : timestamp);
    }
    /**
     * @param {string} pod_json
     * @param {bigint | null} [timestamp]
     */
    upsert_pod(pod_json, timestamp) {
        const ptr0 = passStringToWasm0(pod_json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmpodstate_upsert_pod(this.__wbg_ptr, ptr0, len0, !isLikeNone(timestamp), isLikeNone(timestamp) ? BigInt(0) : timestamp);
    }
}
if (Symbol.dispose) WasmPodState.prototype[Symbol.dispose] = WasmPodState.prototype.free;

export class WasmPromoCodeService {
    static __wrap(ptr) {
        ptr = ptr >>> 0;
        const obj = Object.create(WasmPromoCodeService.prototype);
        obj.__wbg_ptr = ptr;
        WasmPromoCodeServiceFinalization.register(obj, obj.__wbg_ptr, obj);
        return obj;
    }
    __destroy_into_raw() {
        const ptr = this.__wbg_ptr;
        this.__wbg_ptr = 0;
        WasmPromoCodeServiceFinalization.unregister(this);
        return ptr;
    }
    free() {
        const ptr = this.__destroy_into_raw();
        wasm.__wbg_wasmpromocodeservice_free(ptr, 0);
    }
    /**
     * @returns {Promise<string>}
     */
    get_history() {
        const ret = wasm.wasmpromocodeservice_get_history(this.__wbg_ptr);
        return ret;
    }
    /**
     * @param {string} json
     * @returns {Promise<void>}
     */
    redeem(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmpromocodeservice_redeem(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {string} json
     * @returns {Promise<string>}
     */
    validate(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmpromocodeservice_validate(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
}
if (Symbol.dispose) WasmPromoCodeService.prototype[Symbol.dispose] = WasmPromoCodeService.prototype.free;

export class WasmRelayManager {
    __destroy_into_raw() {
        const ptr = this.__wbg_ptr;
        this.__wbg_ptr = 0;
        WasmRelayManagerFinalization.unregister(this);
        return ptr;
    }
    free() {
        const ptr = this.__destroy_into_raw();
        wasm.__wbg_wasmrelaymanager_free(ptr, 0);
    }
    /**
     * @param {string} pod_key
     * @returns {Promise<void>}
     */
    disconnect(pod_key) {
        const ptr0 = passStringToWasm0(pod_key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmrelaymanager_disconnect(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @returns {Promise<void>}
     */
    disconnect_all() {
        const ret = wasm.wasmrelaymanager_disconnect_all(this.__wbg_ptr);
        return ret;
    }
    /**
     * @param {string} pod_key
     * @param {number} cols
     * @param {number} rows
     * @returns {Promise<void>}
     */
    force_resize(pod_key, cols, rows) {
        const ptr0 = passStringToWasm0(pod_key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmrelaymanager_force_resize(this.__wbg_ptr, ptr0, len0, cols, rows);
        return ret;
    }
    /**
     * @param {string} pod_key
     * @returns {Promise<any>}
     */
    get_pod_size(pod_key) {
        const ptr0 = passStringToWasm0(pod_key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmrelaymanager_get_pod_size(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {string} pod_key
     * @returns {Promise<string>}
     */
    get_status(pod_key) {
        const ptr0 = passStringToWasm0(pod_key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmrelaymanager_get_status(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {string} pod_key
     * @returns {Promise<boolean>}
     */
    is_runner_disconnected(pod_key) {
        const ptr0 = passStringToWasm0(pod_key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmrelaymanager_is_runner_disconnected(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    constructor() {
        const ret = wasm.wasmrelaymanager_new();
        this.__wbg_ptr = ret >>> 0;
        WasmRelayManagerFinalization.register(this, this.__wbg_ptr, this);
        return this;
    }
    /**
     * @param {string} pod_key
     * @param {Function} callback
     * @returns {Promise<void>}
     */
    on_acp_message(pod_key, callback) {
        const ptr0 = passStringToWasm0(pod_key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmrelaymanager_on_acp_message(this.__wbg_ptr, ptr0, len0, callback);
        return ret;
    }
    /**
     * @param {string} pod_key
     * @param {Function} callback
     * @returns {Promise<void>}
     */
    on_status_change(pod_key, callback) {
        const ptr0 = passStringToWasm0(pod_key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmrelaymanager_on_status_change(this.__wbg_ptr, ptr0, len0, callback);
        return ret;
    }
    /**
     * @param {string} pod_key
     * @param {string} data
     * @returns {Promise<void>}
     */
    send(pod_key, data) {
        const ptr0 = passStringToWasm0(pod_key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(data, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        const ret = wasm.wasmrelaymanager_send(this.__wbg_ptr, ptr0, len0, ptr1, len1);
        return ret;
    }
    /**
     * @param {string} pod_key
     * @param {string} command
     * @returns {Promise<void>}
     */
    send_acp_command(pod_key, command) {
        const ptr0 = passStringToWasm0(pod_key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(command, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        const ret = wasm.wasmrelaymanager_send_acp_command(this.__wbg_ptr, ptr0, len0, ptr1, len1);
        return ret;
    }
    /**
     * @param {string} pod_key
     * @param {number} cols
     * @param {number} rows
     * @returns {Promise<void>}
     */
    send_resize(pod_key, cols, rows) {
        const ptr0 = passStringToWasm0(pod_key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmrelaymanager_send_resize(this.__wbg_ptr, ptr0, len0, cols, rows);
        return ret;
    }
    /**
     * @param {string} pod_key
     * @param {string} subscription_id
     * @param {string} relay_url
     * @param {string} token
     * @param {Function} callback
     * @returns {Promise<void>}
     */
    subscribe(pod_key, subscription_id, relay_url, token, callback) {
        const ptr0 = passStringToWasm0(pod_key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(subscription_id, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        const ptr2 = passStringToWasm0(relay_url, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len2 = WASM_VECTOR_LEN;
        const ptr3 = passStringToWasm0(token, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len3 = WASM_VECTOR_LEN;
        const ret = wasm.wasmrelaymanager_subscribe(this.__wbg_ptr, ptr0, len0, ptr1, len1, ptr2, len2, ptr3, len3, callback);
        return ret;
    }
    /**
     * @param {string} pod_key
     * @param {string} subscription_id
     * @returns {Promise<void>}
     */
    unsubscribe(pod_key, subscription_id) {
        const ptr0 = passStringToWasm0(pod_key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(subscription_id, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        const ret = wasm.wasmrelaymanager_unsubscribe(this.__wbg_ptr, ptr0, len0, ptr1, len1);
        return ret;
    }
}
if (Symbol.dispose) WasmRelayManager.prototype[Symbol.dispose] = WasmRelayManager.prototype.free;

export class WasmRepoState {
    __destroy_into_raw() {
        const ptr = this.__wbg_ptr;
        this.__wbg_ptr = 0;
        WasmRepoStateFinalization.unregister(this);
        return ptr;
    }
    free() {
        const ptr = this.__destroy_into_raw();
        wasm.__wbg_wasmrepostate_free(ptr, 0);
    }
    /**
     * @param {string} json
     */
    add_repository(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmrepostate_add_repository(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @returns {string}
     */
    branches_json() {
        let deferred1_0;
        let deferred1_1;
        try {
            const ret = wasm.wasmrepostate_branches_json(this.__wbg_ptr);
            deferred1_0 = ret[0];
            deferred1_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred1_0, deferred1_1, 1);
        }
    }
    /**
     * @returns {any}
     */
    current_repo_json() {
        const ret = wasm.wasmrepostate_current_repo_json(this.__wbg_ptr);
        return ret;
    }
    constructor() {
        const ret = wasm.wasmrepostate_new();
        this.__wbg_ptr = ret >>> 0;
        WasmRepoStateFinalization.register(this, this.__wbg_ptr, this);
        return this;
    }
    /**
     * @param {string} id
     */
    remove_repository(id) {
        const ptr0 = passStringToWasm0(id, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmrepostate_remove_repository(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @returns {string}
     */
    repositories_json() {
        let deferred1_0;
        let deferred1_1;
        try {
            const ret = wasm.wasmrepostate_repositories_json(this.__wbg_ptr);
            deferred1_0 = ret[0];
            deferred1_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred1_0, deferred1_1, 1);
        }
    }
    /**
     * @param {string} json
     */
    set_branches(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmrepostate_set_branches(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {string} json
     */
    set_current_repo(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmrepostate_set_current_repo(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {string} json
     */
    set_repositories(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmrepostate_set_repositories(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {string} id
     * @param {string} json
     */
    update_repository(id, json) {
        const ptr0 = passStringToWasm0(id, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        wasm.wasmrepostate_update_repository(this.__wbg_ptr, ptr0, len0, ptr1, len1);
    }
}
if (Symbol.dispose) WasmRepoState.prototype[Symbol.dispose] = WasmRepoState.prototype.free;

export class WasmRepositoryService {
    static __wrap(ptr) {
        ptr = ptr >>> 0;
        const obj = Object.create(WasmRepositoryService.prototype);
        obj.__wbg_ptr = ptr;
        WasmRepositoryServiceFinalization.register(obj, obj.__wbg_ptr, obj);
        return obj;
    }
    __destroy_into_raw() {
        const ptr = this.__wbg_ptr;
        this.__wbg_ptr = 0;
        WasmRepositoryServiceFinalization.unregister(this);
        return ptr;
    }
    free() {
        const ptr = this.__destroy_into_raw();
        wasm.__wbg_wasmrepositoryservice_free(ptr, 0);
    }
    /**
     * @param {string} json
     * @returns {Promise<string>}
     */
    create(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmrepositoryservice_create(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {bigint} id
     * @returns {Promise<void>}
     */
    delete(id) {
        const ret = wasm.wasmrepositoryservice_delete(this.__wbg_ptr, id);
        return ret;
    }
    /**
     * @param {bigint} id
     * @returns {Promise<void>}
     */
    delete_webhook(id) {
        const ret = wasm.wasmrepositoryservice_delete_webhook(this.__wbg_ptr, id);
        return ret;
    }
    /**
     * @param {bigint} id
     * @returns {Promise<string>}
     */
    get(id) {
        const ret = wasm.wasmrepositoryservice_get(this.__wbg_ptr, id);
        return ret;
    }
    /**
     * @param {bigint} id
     * @returns {Promise<string>}
     */
    get_webhook_secret(id) {
        const ret = wasm.wasmrepositoryservice_get_webhook_secret(this.__wbg_ptr, id);
        return ret;
    }
    /**
     * @param {bigint} id
     * @returns {Promise<string>}
     */
    get_webhook_status(id) {
        const ret = wasm.wasmrepositoryservice_get_webhook_status(this.__wbg_ptr, id);
        return ret;
    }
    /**
     * @returns {Promise<string>}
     */
    list() {
        const ret = wasm.wasmrepositoryservice_list(this.__wbg_ptr);
        return ret;
    }
    /**
     * @param {bigint} id
     * @returns {Promise<string>}
     */
    list_branches(id) {
        const ret = wasm.wasmrepositoryservice_list_branches(this.__wbg_ptr, id);
        return ret;
    }
    /**
     * @param {bigint} id
     * @param {string | null} [branch]
     * @param {string | null} [state]
     * @returns {Promise<string>}
     */
    list_merge_requests(id, branch, state) {
        var ptr0 = isLikeNone(branch) ? 0 : passStringToWasm0(branch, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        var len0 = WASM_VECTOR_LEN;
        var ptr1 = isLikeNone(state) ? 0 : passStringToWasm0(state, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        var len1 = WASM_VECTOR_LEN;
        const ret = wasm.wasmrepositoryservice_list_merge_requests(this.__wbg_ptr, id, ptr0, len0, ptr1, len1);
        return ret;
    }
    /**
     * @param {bigint} id
     * @returns {Promise<void>}
     */
    mark_webhook_configured(id) {
        const ret = wasm.wasmrepositoryservice_mark_webhook_configured(this.__wbg_ptr, id);
        return ret;
    }
    /**
     * @param {bigint} id
     * @returns {Promise<void>}
     */
    register_webhook(id) {
        const ret = wasm.wasmrepositoryservice_register_webhook(this.__wbg_ptr, id);
        return ret;
    }
    /**
     * @param {bigint} id
     * @param {string} json
     * @returns {Promise<string>}
     */
    sync_branches(id, json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmrepositoryservice_sync_branches(this.__wbg_ptr, id, ptr0, len0);
        return ret;
    }
    /**
     * @param {bigint} id
     * @param {string} json
     * @returns {Promise<string>}
     */
    update(id, json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmrepositoryservice_update(this.__wbg_ptr, id, ptr0, len0);
        return ret;
    }
}
if (Symbol.dispose) WasmRepositoryService.prototype[Symbol.dispose] = WasmRepositoryService.prototype.free;

export class WasmRunnerService {
    static __wrap(ptr) {
        ptr = ptr >>> 0;
        const obj = Object.create(WasmRunnerService.prototype);
        obj.__wbg_ptr = ptr;
        WasmRunnerServiceFinalization.register(obj, obj.__wbg_ptr, obj);
        return obj;
    }
    __destroy_into_raw() {
        const ptr = this.__wbg_ptr;
        this.__wbg_ptr = 0;
        WasmRunnerServiceFinalization.unregister(this);
        return ptr;
    }
    free() {
        const ptr = this.__destroy_into_raw();
        wasm.__wbg_wasmrunnerservice_free(ptr, 0);
    }
    /**
     * @param {string} request_json
     * @returns {Promise<string>}
     */
    authorize_runner(request_json) {
        const ptr0 = passStringToWasm0(request_json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmrunnerservice_authorize_runner(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @returns {string}
     */
    available_runners_json() {
        let deferred1_0;
        let deferred1_1;
        try {
            const ret = wasm.wasmrunnerservice_available_runners_json(this.__wbg_ptr);
            deferred1_0 = ret[0];
            deferred1_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred1_0, deferred1_1, 1);
        }
    }
    /**
     * @param {string} request_json
     * @returns {Promise<string>}
     */
    create_token(request_json) {
        const ptr0 = passStringToWasm0(request_json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmrunnerservice_create_token(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @returns {any}
     */
    current_runner_json() {
        const ret = wasm.wasmrunnerservice_current_runner_json(this.__wbg_ptr);
        return ret;
    }
    /**
     * @param {bigint} id
     * @returns {Promise<void>}
     */
    delete_runner(id) {
        const ret = wasm.wasmrunnerservice_delete_runner(this.__wbg_ptr, id);
        return ret;
    }
    /**
     * @param {bigint} id
     * @returns {Promise<void>}
     */
    delete_token(id) {
        const ret = wasm.wasmrunnerservice_delete_token(this.__wbg_ptr, id);
        return ret;
    }
    /**
     * @returns {Promise<string>}
     */
    fetch_available_runners() {
        const ret = wasm.wasmrunnerservice_fetch_available_runners(this.__wbg_ptr);
        return ret;
    }
    /**
     * @param {bigint} id
     * @returns {Promise<string>}
     */
    fetch_runner(id) {
        const ret = wasm.wasmrunnerservice_fetch_runner(this.__wbg_ptr, id);
        return ret;
    }
    /**
     * @param {string | null} [status]
     * @returns {Promise<string>}
     */
    fetch_runners(status) {
        var ptr0 = isLikeNone(status) ? 0 : passStringToWasm0(status, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        var len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmrunnerservice_fetch_runners(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @returns {Promise<string>}
     */
    fetch_tokens() {
        const ret = wasm.wasmrunnerservice_fetch_tokens(this.__wbg_ptr);
        return ret;
    }
    /**
     * @param {string} auth_key
     * @returns {Promise<string>}
     */
    get_auth_status(auth_key) {
        const ptr0 = passStringToWasm0(auth_key, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmrunnerservice_get_auth_status(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {bigint} id
     * @returns {any}
     */
    get_runner_json(id) {
        const ret = wasm.wasmrunnerservice_get_runner_json(this.__wbg_ptr, id);
        return ret;
    }
    /**
     * @param {bigint} id
     * @returns {Promise<string>}
     */
    list_runner_logs(id) {
        const ret = wasm.wasmrunnerservice_list_runner_logs(this.__wbg_ptr, id);
        return ret;
    }
    /**
     * @param {bigint} id
     * @param {string | null} [status]
     * @param {number | null} [limit]
     * @param {number | null} [offset]
     * @returns {Promise<string>}
     */
    list_runner_pods(id, status, limit, offset) {
        var ptr0 = isLikeNone(status) ? 0 : passStringToWasm0(status, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        var len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmrunnerservice_list_runner_pods(this.__wbg_ptr, id, ptr0, len0, isLikeNone(limit) ? 0x100000001 : (limit) >>> 0, isLikeNone(offset) ? 0x100000001 : (offset) >>> 0);
        return ret;
    }
    /**
     * @param {bigint} id
     * @param {string} request_json
     * @returns {Promise<string>}
     */
    query_runner_sandboxes(id, request_json) {
        const ptr0 = passStringToWasm0(request_json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmrunnerservice_query_runner_sandboxes(this.__wbg_ptr, id, ptr0, len0);
        return ret;
    }
    /**
     * @param {bigint} id
     */
    remove_runner_local(id) {
        wasm.wasmrunnerservice_remove_runner_local(this.__wbg_ptr, id);
    }
    /**
     * @param {bigint} id
     * @returns {Promise<void>}
     */
    request_log_upload(id) {
        const ret = wasm.wasmrunnerservice_request_log_upload(this.__wbg_ptr, id);
        return ret;
    }
    /**
     * @returns {string}
     */
    runners_json() {
        let deferred1_0;
        let deferred1_1;
        try {
            const ret = wasm.wasmrunnerservice_runners_json(this.__wbg_ptr);
            deferred1_0 = ret[0];
            deferred1_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred1_0, deferred1_1, 1);
        }
    }
    /**
     * @param {string} json
     */
    set_available_runners(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmrunnerservice_set_available_runners(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {string} json
     */
    set_current_runner(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmrunnerservice_set_current_runner(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {string} json
     */
    set_runners(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmrunnerservice_set_runners(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {bigint} id
     * @param {string} request_json
     * @returns {Promise<string>}
     */
    update_runner(id, request_json) {
        const ptr0 = passStringToWasm0(request_json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmrunnerservice_update_runner(this.__wbg_ptr, id, ptr0, len0);
        return ret;
    }
    /**
     * @param {number} id
     * @param {string} json
     */
    update_runner_local(id, json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmrunnerservice_update_runner_local(this.__wbg_ptr, id, ptr0, len0);
    }
    /**
     * @param {bigint} id
     * @param {string} status
     */
    update_runner_status(id, status) {
        const ptr0 = passStringToWasm0(status, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmrunnerservice_update_runner_status(this.__wbg_ptr, id, ptr0, len0);
    }
    /**
     * @param {bigint} id
     * @param {string} request_json
     * @returns {Promise<string>}
     */
    upgrade_runner(id, request_json) {
        const ptr0 = passStringToWasm0(request_json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmrunnerservice_upgrade_runner(this.__wbg_ptr, id, ptr0, len0);
        return ret;
    }
}
if (Symbol.dispose) WasmRunnerService.prototype[Symbol.dispose] = WasmRunnerService.prototype.free;

export class WasmRunnerState {
    __destroy_into_raw() {
        const ptr = this.__wbg_ptr;
        this.__wbg_ptr = 0;
        WasmRunnerStateFinalization.unregister(this);
        return ptr;
    }
    free() {
        const ptr = this.__destroy_into_raw();
        wasm.__wbg_wasmrunnerstate_free(ptr, 0);
    }
    /**
     * @returns {string}
     */
    available_runners_json() {
        let deferred1_0;
        let deferred1_1;
        try {
            const ret = wasm.wasmrunnerstate_available_runners_json(this.__wbg_ptr);
            deferred1_0 = ret[0];
            deferred1_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred1_0, deferred1_1, 1);
        }
    }
    /**
     * @param {string} runner_json
     * @returns {boolean}
     */
    static can_accept_pods(runner_json) {
        const ptr0 = passStringToWasm0(runner_json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmrunnerstate_can_accept_pods(ptr0, len0);
        return ret !== 0;
    }
    /**
     * @returns {any}
     */
    current_runner_json() {
        const ret = wasm.wasmrunnerstate_current_runner_json(this.__wbg_ptr);
        return ret;
    }
    /**
     * @param {bigint} id
     * @returns {any}
     */
    get_runner_json(id) {
        const ret = wasm.wasmrunnerstate_get_runner_json(this.__wbg_ptr, id);
        return ret;
    }
    constructor() {
        const ret = wasm.wasmrunnerstate_new();
        this.__wbg_ptr = ret >>> 0;
        WasmRunnerStateFinalization.register(this, this.__wbg_ptr, this);
        return this;
    }
    /**
     * @param {bigint} id
     */
    remove_runner(id) {
        wasm.wasmrunnerstate_remove_runner(this.__wbg_ptr, id);
    }
    /**
     * @returns {string}
     */
    runners_json() {
        let deferred1_0;
        let deferred1_1;
        try {
            const ret = wasm.wasmrunnerstate_runners_json(this.__wbg_ptr);
            deferred1_0 = ret[0];
            deferred1_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred1_0, deferred1_1, 1);
        }
    }
    /**
     * @param {string} json
     */
    set_available_runners(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmrunnerstate_set_available_runners(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {string} json
     */
    set_current_runner(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmrunnerstate_set_current_runner(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {string} json
     */
    set_runners(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmrunnerstate_set_runners(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {number} id
     * @param {string} json
     */
    update_runner(id, json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmrunnerstate_update_runner(this.__wbg_ptr, id, ptr0, len0);
    }
    /**
     * @param {bigint} id
     * @param {string} status
     */
    update_runner_status(id, status) {
        const ptr0 = passStringToWasm0(status, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmrunnerstate_update_runner_status(this.__wbg_ptr, id, ptr0, len0);
    }
}
if (Symbol.dispose) WasmRunnerState.prototype[Symbol.dispose] = WasmRunnerState.prototype.free;

export class WasmSSOService {
    static __wrap(ptr) {
        ptr = ptr >>> 0;
        const obj = Object.create(WasmSSOService.prototype);
        obj.__wbg_ptr = ptr;
        WasmSSOServiceFinalization.register(obj, obj.__wbg_ptr, obj);
        return obj;
    }
    __destroy_into_raw() {
        const ptr = this.__wbg_ptr;
        this.__wbg_ptr = 0;
        WasmSSOServiceFinalization.unregister(this);
        return ptr;
    }
    free() {
        const ptr = this.__destroy_into_raw();
        wasm.__wbg_wasmssoservice_free(ptr, 0);
    }
    /**
     * @param {string} email
     * @returns {Promise<string>}
     */
    discover(email) {
        const ptr0 = passStringToWasm0(email, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmssoservice_discover(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {string} domain
     * @param {string} json
     * @returns {Promise<string>}
     */
    ldap_auth(domain, json) {
        const ptr0 = passStringToWasm0(domain, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        const ret = wasm.wasmssoservice_ldap_auth(this.__wbg_ptr, ptr0, len0, ptr1, len1);
        return ret;
    }
}
if (Symbol.dispose) WasmSSOService.prototype[Symbol.dispose] = WasmSSOService.prototype.free;

export class WasmSupportTicketService {
    static __wrap(ptr) {
        ptr = ptr >>> 0;
        const obj = Object.create(WasmSupportTicketService.prototype);
        obj.__wbg_ptr = ptr;
        WasmSupportTicketServiceFinalization.register(obj, obj.__wbg_ptr, obj);
        return obj;
    }
    __destroy_into_raw() {
        const ptr = this.__wbg_ptr;
        this.__wbg_ptr = 0;
        WasmSupportTicketServiceFinalization.unregister(this);
        return ptr;
    }
    free() {
        const ptr = this.__destroy_into_raw();
        wasm.__wbg_wasmsupportticketservice_free(ptr, 0);
    }
    /**
     * @param {bigint} ticket_id
     * @param {string} content
     * @param {Uint8Array[]} file_data
     * @param {string[]} file_names
     * @returns {Promise<string>}
     */
    add_message(ticket_id, content, file_data, file_names) {
        const ptr0 = passStringToWasm0(content, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passArrayJsValueToWasm0(file_data, wasm.__wbindgen_malloc);
        const len1 = WASM_VECTOR_LEN;
        const ptr2 = passArrayJsValueToWasm0(file_names, wasm.__wbindgen_malloc);
        const len2 = WASM_VECTOR_LEN;
        const ret = wasm.wasmsupportticketservice_add_message(this.__wbg_ptr, ticket_id, ptr0, len0, ptr1, len1, ptr2, len2);
        return ret;
    }
    /**
     * @param {string} title
     * @param {string} category
     * @param {string} content
     * @param {string | null | undefined} priority
     * @param {Uint8Array[]} file_data
     * @param {string[]} file_names
     * @returns {Promise<string>}
     */
    create_ticket(title, category, content, priority, file_data, file_names) {
        const ptr0 = passStringToWasm0(title, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(category, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        const ptr2 = passStringToWasm0(content, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len2 = WASM_VECTOR_LEN;
        var ptr3 = isLikeNone(priority) ? 0 : passStringToWasm0(priority, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        var len3 = WASM_VECTOR_LEN;
        const ptr4 = passArrayJsValueToWasm0(file_data, wasm.__wbindgen_malloc);
        const len4 = WASM_VECTOR_LEN;
        const ptr5 = passArrayJsValueToWasm0(file_names, wasm.__wbindgen_malloc);
        const len5 = WASM_VECTOR_LEN;
        const ret = wasm.wasmsupportticketservice_create_ticket(this.__wbg_ptr, ptr0, len0, ptr1, len1, ptr2, len2, ptr3, len3, ptr4, len4, ptr5, len5);
        return ret;
    }
    /**
     * @param {bigint} id
     * @returns {Promise<string>}
     */
    get_attachment_url(id) {
        const ret = wasm.wasmsupportticketservice_get_attachment_url(this.__wbg_ptr, id);
        return ret;
    }
    /**
     * @param {bigint} id
     * @returns {Promise<string>}
     */
    get_detail(id) {
        const ret = wasm.wasmsupportticketservice_get_detail(this.__wbg_ptr, id);
        return ret;
    }
    /**
     * @param {string | null} [status]
     * @param {number | null} [page]
     * @param {number | null} [page_size]
     * @returns {Promise<string>}
     */
    list(status, page, page_size) {
        var ptr0 = isLikeNone(status) ? 0 : passStringToWasm0(status, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        var len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmsupportticketservice_list(this.__wbg_ptr, ptr0, len0, isLikeNone(page) ? 0x100000001 : (page) >>> 0, isLikeNone(page_size) ? 0x100000001 : (page_size) >>> 0);
        return ret;
    }
}
if (Symbol.dispose) WasmSupportTicketService.prototype[Symbol.dispose] = WasmSupportTicketService.prototype.free;

export class WasmTicketRelationsService {
    static __wrap(ptr) {
        ptr = ptr >>> 0;
        const obj = Object.create(WasmTicketRelationsService.prototype);
        obj.__wbg_ptr = ptr;
        WasmTicketRelationsServiceFinalization.register(obj, obj.__wbg_ptr, obj);
        return obj;
    }
    __destroy_into_raw() {
        const ptr = this.__wbg_ptr;
        this.__wbg_ptr = 0;
        WasmTicketRelationsServiceFinalization.unregister(this);
        return ptr;
    }
    free() {
        const ptr = this.__destroy_into_raw();
        wasm.__wbg_wasmticketrelationsservice_free(ptr, 0);
    }
    /**
     * @param {string} slug
     * @param {string} json
     * @returns {Promise<string>}
     */
    create_comment(slug, json) {
        const ptr0 = passStringToWasm0(slug, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        const ret = wasm.wasmticketrelationsservice_create_comment(this.__wbg_ptr, ptr0, len0, ptr1, len1);
        return ret;
    }
    /**
     * @param {string} slug
     * @param {string} json
     * @returns {Promise<string>}
     */
    create_relation(slug, json) {
        const ptr0 = passStringToWasm0(slug, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        const ret = wasm.wasmticketrelationsservice_create_relation(this.__wbg_ptr, ptr0, len0, ptr1, len1);
        return ret;
    }
    /**
     * @param {string} slug
     * @param {bigint} comment_id
     * @returns {Promise<void>}
     */
    delete_comment(slug, comment_id) {
        const ptr0 = passStringToWasm0(slug, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmticketrelationsservice_delete_comment(this.__wbg_ptr, ptr0, len0, comment_id);
        return ret;
    }
    /**
     * @param {string} slug
     * @param {bigint} relation_id
     * @returns {Promise<void>}
     */
    delete_relation(slug, relation_id) {
        const ptr0 = passStringToWasm0(slug, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmticketrelationsservice_delete_relation(this.__wbg_ptr, ptr0, len0, relation_id);
        return ret;
    }
    /**
     * @param {string} slug
     * @param {string} json
     * @returns {Promise<string>}
     */
    link_commit(slug, json) {
        const ptr0 = passStringToWasm0(slug, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        const ret = wasm.wasmticketrelationsservice_link_commit(this.__wbg_ptr, ptr0, len0, ptr1, len1);
        return ret;
    }
    /**
     * @param {string} slug
     * @param {number | null} [limit]
     * @param {number | null} [offset]
     * @returns {Promise<string>}
     */
    list_comments(slug, limit, offset) {
        const ptr0 = passStringToWasm0(slug, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmticketrelationsservice_list_comments(this.__wbg_ptr, ptr0, len0, isLikeNone(limit) ? 0x100000001 : (limit) >>> 0, isLikeNone(offset) ? 0x100000001 : (offset) >>> 0);
        return ret;
    }
    /**
     * @param {string} slug
     * @returns {Promise<string>}
     */
    list_commits(slug) {
        const ptr0 = passStringToWasm0(slug, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmticketrelationsservice_list_commits(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {string} slug
     * @returns {Promise<string>}
     */
    list_merge_requests(slug) {
        const ptr0 = passStringToWasm0(slug, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmticketrelationsservice_list_merge_requests(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {string} slug
     * @returns {Promise<string>}
     */
    list_relations(slug) {
        const ptr0 = passStringToWasm0(slug, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmticketrelationsservice_list_relations(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {string} slug
     * @param {bigint} commit_id
     * @returns {Promise<void>}
     */
    unlink_commit(slug, commit_id) {
        const ptr0 = passStringToWasm0(slug, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmticketrelationsservice_unlink_commit(this.__wbg_ptr, ptr0, len0, commit_id);
        return ret;
    }
    /**
     * @param {string} slug
     * @param {bigint} comment_id
     * @param {string} json
     * @returns {Promise<string>}
     */
    update_comment(slug, comment_id, json) {
        const ptr0 = passStringToWasm0(slug, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        const ret = wasm.wasmticketrelationsservice_update_comment(this.__wbg_ptr, ptr0, len0, comment_id, ptr1, len1);
        return ret;
    }
}
if (Symbol.dispose) WasmTicketRelationsService.prototype[Symbol.dispose] = WasmTicketRelationsService.prototype.free;

export class WasmTicketService {
    static __wrap(ptr) {
        ptr = ptr >>> 0;
        const obj = Object.create(WasmTicketService.prototype);
        obj.__wbg_ptr = ptr;
        WasmTicketServiceFinalization.register(obj, obj.__wbg_ptr, obj);
        return obj;
    }
    __destroy_into_raw() {
        const ptr = this.__wbg_ptr;
        this.__wbg_ptr = 0;
        WasmTicketServiceFinalization.unregister(this);
        return ptr;
    }
    free() {
        const ptr = this.__destroy_into_raw();
        wasm.__wbg_wasmticketservice_free(ptr, 0);
    }
    /**
     * @param {string} json
     */
    add_label(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmticketservice_add_label(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {string} json
     */
    add_ticket(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmticketservice_add_ticket(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {string} status
     * @param {string} json
     */
    append_column_tickets(status, json) {
        const ptr0 = passStringToWasm0(status, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        wasm.wasmticketservice_append_column_tickets(this.__wbg_ptr, ptr0, len0, ptr1, len1);
    }
    /**
     * @returns {string}
     */
    board_columns_json() {
        let deferred1_0;
        let deferred1_1;
        try {
            const ret = wasm.wasmticketservice_board_columns_json(this.__wbg_ptr);
            deferred1_0 = ret[0];
            deferred1_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred1_0, deferred1_1, 1);
        }
    }
    /**
     * @param {string} name
     * @param {string} color
     * @param {bigint | null} [repository_id]
     * @returns {Promise<string>}
     */
    create_label(name, color, repository_id) {
        const ptr0 = passStringToWasm0(name, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(color, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        const ret = wasm.wasmticketservice_create_label(this.__wbg_ptr, ptr0, len0, ptr1, len1, !isLikeNone(repository_id), isLikeNone(repository_id) ? BigInt(0) : repository_id);
        return ret;
    }
    /**
     * @param {string} request_json
     * @returns {Promise<string>}
     */
    create_ticket(request_json) {
        const ptr0 = passStringToWasm0(request_json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmticketservice_create_ticket(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @returns {any}
     */
    current_ticket_json() {
        const ret = wasm.wasmticketservice_current_ticket_json(this.__wbg_ptr);
        return ret;
    }
    /**
     * @param {number} id
     * @returns {Promise<void>}
     */
    delete_label(id) {
        const ret = wasm.wasmticketservice_delete_label(this.__wbg_ptr, id);
        return ret;
    }
    /**
     * @param {string} slug
     * @returns {Promise<void>}
     */
    delete_ticket(slug) {
        const ptr0 = passStringToWasm0(slug, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmticketservice_delete_ticket(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {bigint | null} [repository_id]
     * @returns {Promise<string>}
     */
    fetch_board(repository_id) {
        const ret = wasm.wasmticketservice_fetch_board(this.__wbg_ptr, !isLikeNone(repository_id), isLikeNone(repository_id) ? BigInt(0) : repository_id);
        return ret;
    }
    /**
     * @param {bigint | null} [repository_id]
     * @returns {Promise<string>}
     */
    fetch_labels(repository_id) {
        const ret = wasm.wasmticketservice_fetch_labels(this.__wbg_ptr, !isLikeNone(repository_id), isLikeNone(repository_id) ? BigInt(0) : repository_id);
        return ret;
    }
    /**
     * @param {string} slug
     * @returns {Promise<string>}
     */
    fetch_ticket(slug) {
        const ptr0 = passStringToWasm0(slug, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmticketservice_fetch_ticket(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {string | null} [status]
     * @param {number | null} [limit]
     * @param {number | null} [offset]
     * @returns {Promise<string>}
     */
    fetch_tickets(status, limit, offset) {
        var ptr0 = isLikeNone(status) ? 0 : passStringToWasm0(status, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        var len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmticketservice_fetch_tickets(this.__wbg_ptr, ptr0, len0, isLikeNone(limit) ? 0x100000001 : (limit) >>> 0, isLikeNone(offset) ? 0x100000001 : (offset) >>> 0);
        return ret;
    }
    /**
     * @param {string} search
     * @param {string} statuses_json
     * @param {string} priorities_json
     * @param {string} repository_ids_json
     * @returns {string}
     */
    filter_tickets_json(search, statuses_json, priorities_json, repository_ids_json) {
        let deferred5_0;
        let deferred5_1;
        try {
            const ptr0 = passStringToWasm0(search, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
            const len0 = WASM_VECTOR_LEN;
            const ptr1 = passStringToWasm0(statuses_json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
            const len1 = WASM_VECTOR_LEN;
            const ptr2 = passStringToWasm0(priorities_json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
            const len2 = WASM_VECTOR_LEN;
            const ptr3 = passStringToWasm0(repository_ids_json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
            const len3 = WASM_VECTOR_LEN;
            const ret = wasm.wasmticketservice_filter_tickets_json(this.__wbg_ptr, ptr0, len0, ptr1, len1, ptr2, len2, ptr3, len3);
            deferred5_0 = ret[0];
            deferred5_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred5_0, deferred5_1, 1);
        }
    }
    /**
     * @param {string} slug
     * @returns {Promise<string>}
     */
    get_sub_tickets(slug) {
        const ptr0 = passStringToWasm0(slug, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmticketservice_get_sub_tickets(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {string} slug
     * @returns {any}
     */
    get_ticket_by_slug_json(slug) {
        const ptr0 = passStringToWasm0(slug, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmticketservice_get_ticket_by_slug_json(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {string} slug
     * @param {boolean | null} [active_only]
     * @returns {Promise<string>}
     */
    get_ticket_pods(slug, active_only) {
        const ptr0 = passStringToWasm0(slug, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmticketservice_get_ticket_pods(this.__wbg_ptr, ptr0, len0, isLikeNone(active_only) ? 0xFFFFFF : active_only ? 1 : 0);
        return ret;
    }
    /**
     * @returns {string}
     */
    labels_json() {
        let deferred1_0;
        let deferred1_1;
        try {
            const ret = wasm.wasmticketservice_labels_json(this.__wbg_ptr);
            deferred1_0 = ret[0];
            deferred1_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred1_0, deferred1_1, 1);
        }
    }
    /**
     * @param {string} status
     * @param {number} offset
     * @param {number} limit
     * @returns {Promise<string>}
     */
    load_more_column(status, offset, limit) {
        const ptr0 = passStringToWasm0(status, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmticketservice_load_more_column(this.__wbg_ptr, ptr0, len0, offset, limit);
        return ret;
    }
    /**
     * @param {number} id
     */
    remove_label(id) {
        wasm.wasmticketservice_remove_label(this.__wbg_ptr, id);
    }
    /**
     * @param {string} slug
     */
    remove_ticket(slug) {
        const ptr0 = passStringToWasm0(slug, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmticketservice_remove_ticket(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {string} json
     */
    set_board_columns(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmticketservice_set_board_columns(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {string} json
     */
    set_current_ticket(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmticketservice_set_current_ticket(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {string} json
     */
    set_labels(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmticketservice_set_labels(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {string} json
     */
    set_tickets(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmticketservice_set_tickets(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {string} slug
     * @returns {string}
     */
    ticket_pods_json(slug) {
        let deferred2_0;
        let deferred2_1;
        try {
            const ptr0 = passStringToWasm0(slug, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
            const len0 = WASM_VECTOR_LEN;
            const ret = wasm.wasmticketservice_ticket_pods_json(this.__wbg_ptr, ptr0, len0);
            deferred2_0 = ret[0];
            deferred2_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred2_0, deferred2_1, 1);
        }
    }
    /**
     * @returns {string}
     */
    tickets_json() {
        let deferred1_0;
        let deferred1_1;
        try {
            const ret = wasm.wasmticketservice_tickets_json(this.__wbg_ptr);
            deferred1_0 = ret[0];
            deferred1_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred1_0, deferred1_1, 1);
        }
    }
    /**
     * @param {string} slug
     * @param {string} request_json
     * @returns {Promise<string>}
     */
    update_ticket(slug, request_json) {
        const ptr0 = passStringToWasm0(slug, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(request_json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        const ret = wasm.wasmticketservice_update_ticket(this.__wbg_ptr, ptr0, len0, ptr1, len1);
        return ret;
    }
    /**
     * @param {string} slug
     * @param {string} json
     */
    update_ticket_local(slug, json) {
        const ptr0 = passStringToWasm0(slug, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        wasm.wasmticketservice_update_ticket_local(this.__wbg_ptr, ptr0, len0, ptr1, len1);
    }
    /**
     * @param {string} slug
     * @param {string} status
     * @returns {Promise<string>}
     */
    update_ticket_status(slug, status) {
        const ptr0 = passStringToWasm0(slug, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(status, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        const ret = wasm.wasmticketservice_update_ticket_status(this.__wbg_ptr, ptr0, len0, ptr1, len1);
        return ret;
    }
    /**
     * @param {string} slug
     * @param {string} status
     */
    update_ticket_status_local(slug, status) {
        const ptr0 = passStringToWasm0(slug, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(status, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        wasm.wasmticketservice_update_ticket_status_local(this.__wbg_ptr, ptr0, len0, ptr1, len1);
    }
}
if (Symbol.dispose) WasmTicketService.prototype[Symbol.dispose] = WasmTicketService.prototype.free;

export class WasmTicketState {
    __destroy_into_raw() {
        const ptr = this.__wbg_ptr;
        this.__wbg_ptr = 0;
        WasmTicketStateFinalization.unregister(this);
        return ptr;
    }
    free() {
        const ptr = this.__destroy_into_raw();
        wasm.__wbg_wasmticketstate_free(ptr, 0);
    }
    /**
     * @param {string} label_json
     */
    add_label(label_json) {
        const ptr0 = passStringToWasm0(label_json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmticketstate_add_label(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {string} ticket_json
     */
    add_ticket(ticket_json) {
        const ptr0 = passStringToWasm0(ticket_json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmticketstate_add_ticket(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {string} status
     * @param {string} tickets_json
     */
    append_column_tickets(status, tickets_json) {
        const ptr0 = passStringToWasm0(status, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(tickets_json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        wasm.wasmticketstate_append_column_tickets(this.__wbg_ptr, ptr0, len0, ptr1, len1);
    }
    /**
     * @returns {string}
     */
    board_columns_json() {
        let deferred1_0;
        let deferred1_1;
        try {
            const ret = wasm.wasmticketstate_board_columns_json(this.__wbg_ptr);
            deferred1_0 = ret[0];
            deferred1_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred1_0, deferred1_1, 1);
        }
    }
    /**
     * @returns {any}
     */
    current_ticket_json() {
        const ret = wasm.wasmticketstate_current_ticket_json(this.__wbg_ptr);
        return ret;
    }
    /**
     * @param {string} search
     * @param {string} statuses_json
     * @param {string} priorities_json
     * @param {string} repository_ids_json
     * @returns {string}
     */
    filter_tickets_json(search, statuses_json, priorities_json, repository_ids_json) {
        let deferred5_0;
        let deferred5_1;
        try {
            const ptr0 = passStringToWasm0(search, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
            const len0 = WASM_VECTOR_LEN;
            const ptr1 = passStringToWasm0(statuses_json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
            const len1 = WASM_VECTOR_LEN;
            const ptr2 = passStringToWasm0(priorities_json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
            const len2 = WASM_VECTOR_LEN;
            const ptr3 = passStringToWasm0(repository_ids_json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
            const len3 = WASM_VECTOR_LEN;
            const ret = wasm.wasmticketstate_filter_tickets_json(this.__wbg_ptr, ptr0, len0, ptr1, len1, ptr2, len2, ptr3, len3);
            deferred5_0 = ret[0];
            deferred5_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred5_0, deferred5_1, 1);
        }
    }
    /**
     * @param {string} slug
     * @returns {any}
     */
    get_ticket_by_slug_json(slug) {
        const ptr0 = passStringToWasm0(slug, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmticketstate_get_ticket_by_slug_json(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @returns {string}
     */
    labels_json() {
        let deferred1_0;
        let deferred1_1;
        try {
            const ret = wasm.wasmticketstate_labels_json(this.__wbg_ptr);
            deferred1_0 = ret[0];
            deferred1_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred1_0, deferred1_1, 1);
        }
    }
    constructor() {
        const ret = wasm.wasmticketstate_new();
        this.__wbg_ptr = ret >>> 0;
        WasmTicketStateFinalization.register(this, this.__wbg_ptr, this);
        return this;
    }
    /**
     * @param {number} id
     */
    remove_label(id) {
        wasm.wasmticketstate_remove_label(this.__wbg_ptr, id);
    }
    /**
     * @param {string} slug
     */
    remove_ticket(slug) {
        const ptr0 = passStringToWasm0(slug, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmticketstate_remove_ticket(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {string} columns_json
     */
    set_board_columns(columns_json) {
        const ptr0 = passStringToWasm0(columns_json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmticketstate_set_board_columns(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {string} ticket_json
     */
    set_current_ticket(ticket_json) {
        const ptr0 = passStringToWasm0(ticket_json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmticketstate_set_current_ticket(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {string} labels_json
     */
    set_labels(labels_json) {
        const ptr0 = passStringToWasm0(labels_json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmticketstate_set_labels(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {string} tickets_json
     */
    set_tickets(tickets_json) {
        const ptr0 = passStringToWasm0(tickets_json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmticketstate_set_tickets(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @returns {string}
     */
    tickets_json() {
        let deferred1_0;
        let deferred1_1;
        try {
            const ret = wasm.wasmticketstate_tickets_json(this.__wbg_ptr);
            deferred1_0 = ret[0];
            deferred1_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred1_0, deferred1_1, 1);
        }
    }
    /**
     * @param {string} slug
     * @param {string} ticket_json
     */
    update_ticket(slug, ticket_json) {
        const ptr0 = passStringToWasm0(slug, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(ticket_json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        wasm.wasmticketstate_update_ticket(this.__wbg_ptr, ptr0, len0, ptr1, len1);
    }
    /**
     * @param {string} slug
     * @param {string} status
     */
    update_ticket_status(slug, status) {
        const ptr0 = passStringToWasm0(slug, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(status, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        wasm.wasmticketstate_update_ticket_status(this.__wbg_ptr, ptr0, len0, ptr1, len1);
    }
}
if (Symbol.dispose) WasmTicketState.prototype[Symbol.dispose] = WasmTicketState.prototype.free;

export class WasmTokenUsageService {
    static __wrap(ptr) {
        ptr = ptr >>> 0;
        const obj = Object.create(WasmTokenUsageService.prototype);
        obj.__wbg_ptr = ptr;
        WasmTokenUsageServiceFinalization.register(obj, obj.__wbg_ptr, obj);
        return obj;
    }
    __destroy_into_raw() {
        const ptr = this.__wbg_ptr;
        this.__wbg_ptr = 0;
        WasmTokenUsageServiceFinalization.unregister(this);
        return ptr;
    }
    free() {
        const ptr = this.__destroy_into_raw();
        wasm.__wbg_wasmtokenusageservice_free(ptr, 0);
    }
    /**
     * @param {string | null} [start_time]
     * @param {string | null} [end_time]
     * @param {string | null} [agent_slug]
     * @param {bigint | null} [user_id]
     * @param {string | null} [model]
     * @param {string | null} [granularity]
     * @returns {Promise<string>}
     */
    get_dashboard(start_time, end_time, agent_slug, user_id, model, granularity) {
        var ptr0 = isLikeNone(start_time) ? 0 : passStringToWasm0(start_time, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        var len0 = WASM_VECTOR_LEN;
        var ptr1 = isLikeNone(end_time) ? 0 : passStringToWasm0(end_time, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        var len1 = WASM_VECTOR_LEN;
        var ptr2 = isLikeNone(agent_slug) ? 0 : passStringToWasm0(agent_slug, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        var len2 = WASM_VECTOR_LEN;
        var ptr3 = isLikeNone(model) ? 0 : passStringToWasm0(model, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        var len3 = WASM_VECTOR_LEN;
        var ptr4 = isLikeNone(granularity) ? 0 : passStringToWasm0(granularity, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        var len4 = WASM_VECTOR_LEN;
        const ret = wasm.wasmtokenusageservice_get_dashboard(this.__wbg_ptr, ptr0, len0, ptr1, len1, ptr2, len2, !isLikeNone(user_id), isLikeNone(user_id) ? BigInt(0) : user_id, ptr3, len3, ptr4, len4);
        return ret;
    }
}
if (Symbol.dispose) WasmTokenUsageService.prototype[Symbol.dispose] = WasmTokenUsageService.prototype.free;

export class WasmUserApiService {
    static __wrap(ptr) {
        ptr = ptr >>> 0;
        const obj = Object.create(WasmUserApiService.prototype);
        obj.__wbg_ptr = ptr;
        WasmUserApiServiceFinalization.register(obj, obj.__wbg_ptr, obj);
        return obj;
    }
    __destroy_into_raw() {
        const ptr = this.__wbg_ptr;
        this.__wbg_ptr = 0;
        WasmUserApiServiceFinalization.unregister(this);
        return ptr;
    }
    free() {
        const ptr = this.__destroy_into_raw();
        wasm.__wbg_wasmuserapiservice_free(ptr, 0);
    }
    /**
     * @returns {Promise<string>}
     */
    get_me() {
        const ret = wasm.wasmuserapiservice_get_me(this.__wbg_ptr);
        return ret;
    }
    /**
     * @returns {Promise<string>}
     */
    get_organizations() {
        const ret = wasm.wasmuserapiservice_get_organizations(this.__wbg_ptr);
        return ret;
    }
}
if (Symbol.dispose) WasmUserApiService.prototype[Symbol.dispose] = WasmUserApiService.prototype.free;

export class WasmUserCredentialService {
    static __wrap(ptr) {
        ptr = ptr >>> 0;
        const obj = Object.create(WasmUserCredentialService.prototype);
        obj.__wbg_ptr = ptr;
        WasmUserCredentialServiceFinalization.register(obj, obj.__wbg_ptr, obj);
        return obj;
    }
    __destroy_into_raw() {
        const ptr = this.__wbg_ptr;
        this.__wbg_ptr = 0;
        WasmUserCredentialServiceFinalization.unregister(this);
        return ptr;
    }
    free() {
        const ptr = this.__destroy_into_raw();
        wasm.__wbg_wasmusercredentialservice_free(ptr, 0);
    }
    /**
     * @returns {Promise<void>}
     */
    clear_default_git_credential() {
        const ret = wasm.wasmusercredentialservice_clear_default_git_credential(this.__wbg_ptr);
        return ret;
    }
    /**
     * @param {string} agent_slug
     * @param {string} json
     * @returns {Promise<string>}
     */
    create_agent_credential(agent_slug, json) {
        const ptr0 = passStringToWasm0(agent_slug, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ptr1 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len1 = WASM_VECTOR_LEN;
        const ret = wasm.wasmusercredentialservice_create_agent_credential(this.__wbg_ptr, ptr0, len0, ptr1, len1);
        return ret;
    }
    /**
     * @param {string} json
     * @returns {Promise<string>}
     */
    create_git_credential(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmusercredentialservice_create_git_credential(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {string} json
     * @returns {Promise<string>}
     */
    create_repo_provider(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmusercredentialservice_create_repo_provider(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {bigint} id
     * @returns {Promise<void>}
     */
    delete_agent_credential(id) {
        const ret = wasm.wasmusercredentialservice_delete_agent_credential(this.__wbg_ptr, id);
        return ret;
    }
    /**
     * @param {bigint} id
     * @returns {Promise<void>}
     */
    delete_git_credential(id) {
        const ret = wasm.wasmusercredentialservice_delete_git_credential(this.__wbg_ptr, id);
        return ret;
    }
    /**
     * @param {bigint} id
     * @returns {Promise<void>}
     */
    delete_repo_provider(id) {
        const ret = wasm.wasmusercredentialservice_delete_repo_provider(this.__wbg_ptr, id);
        return ret;
    }
    /**
     * @param {bigint} id
     * @returns {Promise<string>}
     */
    get_agent_credential(id) {
        const ret = wasm.wasmusercredentialservice_get_agent_credential(this.__wbg_ptr, id);
        return ret;
    }
    /**
     * @returns {Promise<string>}
     */
    get_default_git_credential() {
        const ret = wasm.wasmusercredentialservice_get_default_git_credential(this.__wbg_ptr);
        return ret;
    }
    /**
     * @param {bigint} id
     * @returns {Promise<string>}
     */
    get_git_credential(id) {
        const ret = wasm.wasmusercredentialservice_get_git_credential(this.__wbg_ptr, id);
        return ret;
    }
    /**
     * @param {bigint} id
     * @returns {Promise<string>}
     */
    get_repo_provider(id) {
        const ret = wasm.wasmusercredentialservice_get_repo_provider(this.__wbg_ptr, id);
        return ret;
    }
    /**
     * @returns {Promise<string>}
     */
    list_agent_credentials() {
        const ret = wasm.wasmusercredentialservice_list_agent_credentials(this.__wbg_ptr);
        return ret;
    }
    /**
     * @param {string} agent_slug
     * @returns {Promise<string>}
     */
    list_agent_credentials_for_agent(agent_slug) {
        const ptr0 = passStringToWasm0(agent_slug, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmusercredentialservice_list_agent_credentials_for_agent(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @returns {Promise<string>}
     */
    list_git_credentials() {
        const ret = wasm.wasmusercredentialservice_list_git_credentials(this.__wbg_ptr);
        return ret;
    }
    /**
     * @param {bigint} id
     * @param {number | null} [page]
     * @param {number | null} [per_page]
     * @param {string | null} [search]
     * @returns {Promise<string>}
     */
    list_provider_repositories(id, page, per_page, search) {
        var ptr0 = isLikeNone(search) ? 0 : passStringToWasm0(search, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        var len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmusercredentialservice_list_provider_repositories(this.__wbg_ptr, id, isLikeNone(page) ? 0x100000001 : (page) >>> 0, isLikeNone(per_page) ? 0x100000001 : (per_page) >>> 0, ptr0, len0);
        return ret;
    }
    /**
     * @returns {Promise<string>}
     */
    list_repo_providers() {
        const ret = wasm.wasmusercredentialservice_list_repo_providers(this.__wbg_ptr);
        return ret;
    }
    /**
     * @param {bigint} id
     * @returns {Promise<void>}
     */
    set_default_agent_credential(id) {
        const ret = wasm.wasmusercredentialservice_set_default_agent_credential(this.__wbg_ptr, id);
        return ret;
    }
    /**
     * @param {string} json
     * @returns {Promise<void>}
     */
    set_default_git_credential(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmusercredentialservice_set_default_git_credential(this.__wbg_ptr, ptr0, len0);
        return ret;
    }
    /**
     * @param {bigint} id
     * @returns {Promise<void>}
     */
    set_default_repo_provider(id) {
        const ret = wasm.wasmusercredentialservice_set_default_repo_provider(this.__wbg_ptr, id);
        return ret;
    }
    /**
     * @param {bigint} id
     * @returns {Promise<void>}
     */
    test_repo_provider(id) {
        const ret = wasm.wasmusercredentialservice_test_repo_provider(this.__wbg_ptr, id);
        return ret;
    }
    /**
     * @param {bigint} id
     * @param {string} json
     * @returns {Promise<string>}
     */
    update_agent_credential(id, json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmusercredentialservice_update_agent_credential(this.__wbg_ptr, id, ptr0, len0);
        return ret;
    }
    /**
     * @param {bigint} id
     * @param {string} json
     * @returns {Promise<string>}
     */
    update_git_credential(id, json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmusercredentialservice_update_git_credential(this.__wbg_ptr, id, ptr0, len0);
        return ret;
    }
    /**
     * @param {bigint} id
     * @param {string} json
     * @returns {Promise<string>}
     */
    update_repo_provider(id, json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmusercredentialservice_update_repo_provider(this.__wbg_ptr, id, ptr0, len0);
        return ret;
    }
}
if (Symbol.dispose) WasmUserCredentialService.prototype[Symbol.dispose] = WasmUserCredentialService.prototype.free;

export class WasmUserState {
    __destroy_into_raw() {
        const ptr = this.__wbg_ptr;
        this.__wbg_ptr = 0;
        WasmUserStateFinalization.unregister(this);
        return ptr;
    }
    free() {
        const ptr = this.__destroy_into_raw();
        wasm.__wbg_wasmuserstate_free(ptr, 0);
    }
    /**
     * @param {string} json
     */
    add_identity(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmuserstate_add_identity(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @returns {string}
     */
    identities_json() {
        let deferred1_0;
        let deferred1_1;
        try {
            const ret = wasm.wasmuserstate_identities_json(this.__wbg_ptr);
            deferred1_0 = ret[0];
            deferred1_1 = ret[1];
            return getStringFromWasm0(ret[0], ret[1]);
        } finally {
            wasm.__wbindgen_free(deferred1_0, deferred1_1, 1);
        }
    }
    constructor() {
        const ret = wasm.wasmuserstate_new();
        this.__wbg_ptr = ret >>> 0;
        WasmUserStateFinalization.register(this, this.__wbg_ptr, this);
        return this;
    }
    /**
     * @returns {any}
     */
    profile_json() {
        const ret = wasm.wasmuserstate_profile_json(this.__wbg_ptr);
        return ret;
    }
    /**
     * @param {string} id
     */
    remove_identity(id) {
        const ptr0 = passStringToWasm0(id, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmuserstate_remove_identity(this.__wbg_ptr, ptr0, len0);
    }
    /**
     * @param {string} json
     */
    set_profile(json) {
        const ptr0 = passStringToWasm0(json, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        wasm.wasmuserstate_set_profile(this.__wbg_ptr, ptr0, len0);
    }
}
if (Symbol.dispose) WasmUserState.prototype[Symbol.dispose] = WasmUserState.prototype.free;

export class WasmWebSocket {
    static __wrap(ptr) {
        ptr = ptr >>> 0;
        const obj = Object.create(WasmWebSocket.prototype);
        obj.__wbg_ptr = ptr;
        WasmWebSocketFinalization.register(obj, obj.__wbg_ptr, obj);
        return obj;
    }
    __destroy_into_raw() {
        const ptr = this.__wbg_ptr;
        this.__wbg_ptr = 0;
        WasmWebSocketFinalization.unregister(this);
        return ptr;
    }
    free() {
        const ptr = this.__destroy_into_raw();
        wasm.__wbg_wasmwebsocket_free(ptr, 0);
    }
    close() {
        wasm.wasmwebsocket_close(this.__wbg_ptr);
    }
    /**
     * @param {string} url
     * @param {Function} on_open
     * @param {Function} on_message
     * @param {Function} on_close
     * @param {Function} on_error
     * @returns {WasmWebSocket}
     */
    static connect(url, on_open, on_message, on_close, on_error) {
        const ptr0 = passStringToWasm0(url, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmwebsocket_connect(ptr0, len0, on_open, on_message, on_close, on_error);
        if (ret[2]) {
            throw takeFromExternrefTable0(ret[1]);
        }
        return WasmWebSocket.__wrap(ret[0]);
    }
    /**
     * @returns {boolean}
     */
    is_closed() {
        const ret = wasm.wasmwebsocket_is_closed(this.__wbg_ptr);
        return ret !== 0;
    }
    /**
     * @returns {boolean}
     */
    is_open() {
        const ret = wasm.wasmwebsocket_is_open(this.__wbg_ptr);
        return ret !== 0;
    }
    /**
     * @param {Uint8Array} data
     */
    send_binary(data) {
        const ptr0 = passArray8ToWasm0(data, wasm.__wbindgen_malloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmwebsocket_send_binary(this.__wbg_ptr, ptr0, len0);
        if (ret[1]) {
            throw takeFromExternrefTable0(ret[0]);
        }
    }
    /**
     * @param {string} text
     */
    send_text(text) {
        const ptr0 = passStringToWasm0(text, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
        const len0 = WASM_VECTOR_LEN;
        const ret = wasm.wasmwebsocket_send_text(this.__wbg_ptr, ptr0, len0);
        if (ret[1]) {
            throw takeFromExternrefTable0(ret[0]);
        }
    }
}
if (Symbol.dispose) WasmWebSocket.prototype[Symbol.dispose] = WasmWebSocket.prototype.free;

export function init_panic_hook() {
    wasm.init_panic_hook();
}

/**
 * @param {Uint8Array} data
 * @returns {any}
 */
export function relay_decode_message(data) {
    const ptr0 = passArray8ToWasm0(data, wasm.__wbindgen_malloc);
    const len0 = WASM_VECTOR_LEN;
    const ret = wasm.relay_decode_message(ptr0, len0);
    return ret;
}

/**
 * @param {Uint8Array} data
 * @returns {Uint8Array}
 */
export function relay_encode_acp_command(data) {
    const ptr0 = passArray8ToWasm0(data, wasm.__wbindgen_malloc);
    const len0 = WASM_VECTOR_LEN;
    const ret = wasm.relay_encode_acp_command(ptr0, len0);
    var v2 = getArrayU8FromWasm0(ret[0], ret[1]).slice();
    wasm.__wbindgen_free(ret[0], ret[1] * 1, 1);
    return v2;
}

/**
 * @param {Uint8Array} data
 * @returns {Uint8Array}
 */
export function relay_encode_control(data) {
    const ptr0 = passArray8ToWasm0(data, wasm.__wbindgen_malloc);
    const len0 = WASM_VECTOR_LEN;
    const ret = wasm.relay_encode_control(ptr0, len0);
    var v2 = getArrayU8FromWasm0(ret[0], ret[1]).slice();
    wasm.__wbindgen_free(ret[0], ret[1] * 1, 1);
    return v2;
}

/**
 * @param {Uint8Array} data
 * @returns {Uint8Array}
 */
export function relay_encode_input(data) {
    const ptr0 = passArray8ToWasm0(data, wasm.__wbindgen_malloc);
    const len0 = WASM_VECTOR_LEN;
    const ret = wasm.relay_encode_input(ptr0, len0);
    var v2 = getArrayU8FromWasm0(ret[0], ret[1]).slice();
    wasm.__wbindgen_free(ret[0], ret[1] * 1, 1);
    return v2;
}

/**
 * @returns {Uint8Array}
 */
export function relay_encode_ping() {
    const ret = wasm.relay_encode_ping();
    var v1 = getArrayU8FromWasm0(ret[0], ret[1]).slice();
    wasm.__wbindgen_free(ret[0], ret[1] * 1, 1);
    return v1;
}

/**
 * @param {number} cols
 * @param {number} rows
 * @returns {Uint8Array}
 */
export function relay_encode_resize(cols, rows) {
    const ret = wasm.relay_encode_resize(cols, rows);
    var v1 = getArrayU8FromWasm0(ret[0], ret[1]).slice();
    wasm.__wbindgen_free(ret[0], ret[1] * 1, 1);
    return v1;
}

/**
 * @returns {Uint8Array}
 */
export function relay_encode_resync() {
    const ret = wasm.relay_encode_resync();
    var v1 = getArrayU8FromWasm0(ret[0], ret[1]).slice();
    wasm.__wbindgen_free(ret[0], ret[1] * 1, 1);
    return v1;
}

/**
 * @returns {string}
 */
export function version() {
    let deferred1_0;
    let deferred1_1;
    try {
        const ret = wasm.version();
        deferred1_0 = ret[0];
        deferred1_1 = ret[1];
        return getStringFromWasm0(ret[0], ret[1]);
    } finally {
        wasm.__wbindgen_free(deferred1_0, deferred1_1, 1);
    }
}
function __wbg_get_imports() {
    const import0 = {
        __proto__: null,
        __wbg___wbindgen_debug_string_ab4b34d23d6778bd: function(arg0, arg1) {
            const ret = debugString(arg1);
            const ptr1 = passStringToWasm0(ret, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
            const len1 = WASM_VECTOR_LEN;
            getDataViewMemory0().setInt32(arg0 + 4 * 1, len1, true);
            getDataViewMemory0().setInt32(arg0 + 4 * 0, ptr1, true);
        },
        __wbg___wbindgen_is_function_3baa9db1a987f47d: function(arg0) {
            const ret = typeof(arg0) === 'function';
            return ret;
        },
        __wbg___wbindgen_is_object_63322ec0cd6ea4ef: function(arg0) {
            const val = arg0;
            const ret = typeof(val) === 'object' && val !== null;
            return ret;
        },
        __wbg___wbindgen_is_undefined_29a43b4d42920abd: function(arg0) {
            const ret = arg0 === undefined;
            return ret;
        },
        __wbg___wbindgen_string_get_7ed5322991caaec5: function(arg0, arg1) {
            const obj = arg1;
            const ret = typeof(obj) === 'string' ? obj : undefined;
            var ptr1 = isLikeNone(ret) ? 0 : passStringToWasm0(ret, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
            var len1 = WASM_VECTOR_LEN;
            getDataViewMemory0().setInt32(arg0 + 4 * 1, len1, true);
            getDataViewMemory0().setInt32(arg0 + 4 * 0, ptr1, true);
        },
        __wbg___wbindgen_throw_6b64449b9b9ed33c: function(arg0, arg1) {
            throw new Error(getStringFromWasm0(arg0, arg1));
        },
        __wbg__wbg_cb_unref_b46c9b5a9f08ec37: function(arg0) {
            arg0._wbg_cb_unref();
        },
        __wbg_abort_4ce5b484434ef6fd: function(arg0) {
            arg0.abort();
        },
        __wbg_abort_d53712380a54cc81: function(arg0, arg1) {
            arg0.abort(arg1);
        },
        __wbg_append_5c18df3fff2aba00: function() { return handleError(function (arg0, arg1, arg2, arg3) {
            arg0.append(getStringFromWasm0(arg1, arg2), arg3);
        }, arguments); },
        __wbg_append_5eab0932fa0874d3: function() { return handleError(function (arg0, arg1, arg2, arg3, arg4) {
            arg0.append(getStringFromWasm0(arg1, arg2), getStringFromWasm0(arg3, arg4));
        }, arguments); },
        __wbg_append_8310360c57a185e4: function() { return handleError(function (arg0, arg1, arg2, arg3, arg4, arg5) {
            arg0.append(getStringFromWasm0(arg1, arg2), arg3, getStringFromWasm0(arg4, arg5));
        }, arguments); },
        __wbg_append_e8fc56ce7c00e874: function() { return handleError(function (arg0, arg1, arg2, arg3, arg4) {
            arg0.append(getStringFromWasm0(arg1, arg2), getStringFromWasm0(arg3, arg4));
        }, arguments); },
        __wbg_arrayBuffer_848c392b70c67d3d: function() { return handleError(function (arg0) {
            const ret = arg0.arrayBuffer();
            return ret;
        }, arguments); },
        __wbg_call_14b169f759b26747: function() { return handleError(function (arg0, arg1) {
            const ret = arg0.call(arg1);
            return ret;
        }, arguments); },
        __wbg_call_a24592a6f349a97e: function() { return handleError(function (arg0, arg1, arg2) {
            const ret = arg0.call(arg1, arg2);
            return ret;
        }, arguments); },
        __wbg_call_bb28efe6b2f55b86: function() { return handleError(function (arg0, arg1, arg2, arg3) {
            const ret = arg0.call(arg1, arg2, arg3);
            return ret;
        }, arguments); },
        __wbg_clearTimeout_6b8d9a38b9263d65: function(arg0) {
            const ret = clearTimeout(arg0);
            return ret;
        },
        __wbg_close_88106990eea7f544: function() { return handleError(function (arg0) {
            arg0.close();
        }, arguments); },
        __wbg_code_c4f315d8dc91de14: function(arg0) {
            const ret = arg0.code;
            return ret;
        },
        __wbg_data_bb9dffdd1e99cf2d: function(arg0) {
            const ret = arg0.data;
            return ret;
        },
        __wbg_done_9158f7cc8751ba32: function(arg0) {
            const ret = arg0.done;
            return ret;
        },
        __wbg_error_a6fa202b58aa1cd3: function(arg0, arg1) {
            let deferred0_0;
            let deferred0_1;
            try {
                deferred0_0 = arg0;
                deferred0_1 = arg1;
                console.error(getStringFromWasm0(arg0, arg1));
            } finally {
                wasm.__wbindgen_free(deferred0_0, deferred0_1, 1);
            }
        },
        __wbg_fetch_0d322c0aed196b8b: function(arg0, arg1) {
            const ret = arg0.fetch(arg1);
            return ret;
        },
        __wbg_fetch_9dad4fe911207b37: function(arg0) {
            const ret = fetch(arg0);
            return ret;
        },
        __wbg_getItem_7fe1351b9ea3b2f3: function() { return handleError(function (arg0, arg1, arg2, arg3) {
            const ret = arg1.getItem(getStringFromWasm0(arg2, arg3));
            var ptr1 = isLikeNone(ret) ? 0 : passStringToWasm0(ret, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
            var len1 = WASM_VECTOR_LEN;
            getDataViewMemory0().setInt32(arg0 + 4 * 1, len1, true);
            getDataViewMemory0().setInt32(arg0 + 4 * 0, ptr1, true);
        }, arguments); },
        __wbg_getRandomValues_3f44b700395062e5: function() { return handleError(function (arg0, arg1) {
            globalThis.crypto.getRandomValues(getArrayU8FromWasm0(arg0, arg1));
        }, arguments); },
        __wbg_get_1affdbdd5573b16a: function() { return handleError(function (arg0, arg1) {
            const ret = Reflect.get(arg0, arg1);
            return ret;
        }, arguments); },
        __wbg_get_6011fa3a58f61074: function() { return handleError(function (arg0, arg1) {
            const ret = Reflect.get(arg0, arg1);
            return ret;
        }, arguments); },
        __wbg_get_77069f42a845013b: function(arg0, arg1, arg2, arg3) {
            const ret = arg1.get(getStringFromWasm0(arg2, arg3));
            var ptr1 = isLikeNone(ret) ? 0 : passStringToWasm0(ret, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
            var len1 = WASM_VECTOR_LEN;
            getDataViewMemory0().setInt32(arg0 + 4 * 1, len1, true);
            getDataViewMemory0().setInt32(arg0 + 4 * 0, ptr1, true);
        },
        __wbg_has_880f1d472f7cecba: function() { return handleError(function (arg0, arg1) {
            const ret = Reflect.has(arg0, arg1);
            return ret;
        }, arguments); },
        __wbg_headers_6022deb4e576fb8e: function(arg0) {
            const ret = arg0.headers;
            return ret;
        },
        __wbg_instanceof_ArrayBuffer_7c8433c6ed14ffe3: function(arg0) {
            let result;
            try {
                result = arg0 instanceof ArrayBuffer;
            } catch (_) {
                result = false;
            }
            const ret = result;
            return ret;
        },
        __wbg_instanceof_Response_9b2d111407865ff2: function(arg0) {
            let result;
            try {
                result = arg0 instanceof Response;
            } catch (_) {
                result = false;
            }
            const ret = result;
            return ret;
        },
        __wbg_instanceof_Window_cc64c86c8ef9e02b: function(arg0) {
            let result;
            try {
                result = arg0 instanceof Window;
            } catch (_) {
                result = false;
            }
            const ret = result;
            return ret;
        },
        __wbg_iterator_013bc09ec998c2a7: function() {
            const ret = Symbol.iterator;
            return ret;
        },
        __wbg_length_9f1775224cf1d815: function(arg0) {
            const ret = arg0.length;
            return ret;
        },
        __wbg_localStorage_f5f66b1ffd2486bc: function() { return handleError(function (arg0) {
            const ret = arg0.localStorage;
            return isLikeNone(ret) ? 0 : addToExternrefTable0(ret);
        }, arguments); },
        __wbg_message_aa7e2704b8b86e2a: function(arg0, arg1) {
            const ret = arg1.message;
            const ptr1 = passStringToWasm0(ret, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
            const len1 = WASM_VECTOR_LEN;
            getDataViewMemory0().setInt32(arg0 + 4 * 1, len1, true);
            getDataViewMemory0().setInt32(arg0 + 4 * 0, ptr1, true);
        },
        __wbg_new_036bd6cd9cea9e73: function(arg0, arg1) {
            try {
                var state0 = {a: arg0, b: arg1};
                var cb0 = (arg0, arg1) => {
                    const a = state0.a;
                    state0.a = 0;
                    try {
                        return wasm_bindgen__convert__closures_____invoke__h234923932d6562e7(a, state0.b, arg0, arg1);
                    } finally {
                        state0.a = a;
                    }
                };
                const ret = new Promise(cb0);
                return ret;
            } finally {
                state0.a = 0;
            }
        },
        __wbg_new_0c7403db6e782f19: function(arg0) {
            const ret = new Uint8Array(arg0);
            return ret;
        },
        __wbg_new_15a4889b4b90734d: function() { return handleError(function () {
            const ret = new Headers();
            return ret;
        }, arguments); },
        __wbg_new_227d7c05414eb861: function() {
            const ret = new Error();
            return ret;
        },
        __wbg_new_2a6e9133304ae2bf: function() { return handleError(function (arg0, arg1) {
            const ret = new WebSocket(getStringFromWasm0(arg0, arg1));
            return ret;
        }, arguments); },
        __wbg_new_5e360d2ff7b9e1c3: function(arg0, arg1) {
            const ret = new Error(getStringFromWasm0(arg0, arg1));
            return ret;
        },
        __wbg_new_682678e2f47e32bc: function() {
            const ret = new Array();
            return ret;
        },
        __wbg_new_8c9321bc03b315e9: function() { return handleError(function () {
            const ret = new FormData();
            return ret;
        }, arguments); },
        __wbg_new_98c22165a42231aa: function() { return handleError(function () {
            const ret = new AbortController();
            return ret;
        }, arguments); },
        __wbg_new_aa8d0fa9762c29bd: function() {
            const ret = new Object();
            return ret;
        },
        __wbg_new_from_slice_b5ea43e23f6008c0: function(arg0, arg1) {
            const ret = new Uint8Array(getArrayU8FromWasm0(arg0, arg1));
            return ret;
        },
        __wbg_new_typed_323f37fd55ab048d: function(arg0, arg1) {
            try {
                var state0 = {a: arg0, b: arg1};
                var cb0 = (arg0, arg1) => {
                    const a = state0.a;
                    state0.a = 0;
                    try {
                        return wasm_bindgen__convert__closures_____invoke__h234923932d6562e7(a, state0.b, arg0, arg1);
                    } finally {
                        state0.a = a;
                    }
                };
                const ret = new Promise(cb0);
                return ret;
            } finally {
                state0.a = 0;
            }
        },
        __wbg_new_with_str_and_init_897be1708e42f39d: function() { return handleError(function (arg0, arg1, arg2) {
            const ret = new Request(getStringFromWasm0(arg0, arg1), arg2);
            return ret;
        }, arguments); },
        __wbg_new_with_u8_array_sequence_and_options_afc143a3fe3b3456: function() { return handleError(function (arg0, arg1) {
            const ret = new Blob(arg0, arg1);
            return ret;
        }, arguments); },
        __wbg_next_0340c4ae324393c3: function() { return handleError(function (arg0) {
            const ret = arg0.next();
            return ret;
        }, arguments); },
        __wbg_next_7646edaa39458ef7: function(arg0) {
            const ret = arg0.next;
            return ret;
        },
        __wbg_now_a9b7df1cbee90986: function() {
            const ret = Date.now();
            return ret;
        },
        __wbg_now_e7c6795a7f81e10f: function(arg0) {
            const ret = arg0.now();
            return ret;
        },
        __wbg_onerror_7e73045e83499332: function(arg0) {
            const ret = arg0.onerror;
            return isLikeNone(ret) ? 0 : addToExternrefTable0(ret);
        },
        __wbg_parse_1bbc9c053611d0a7: function() { return handleError(function (arg0, arg1) {
            const ret = JSON.parse(getStringFromWasm0(arg0, arg1));
            return ret;
        }, arguments); },
        __wbg_performance_3fcf6e32a7e1ed0a: function(arg0) {
            const ret = arg0.performance;
            return ret;
        },
        __wbg_prototypesetcall_a6b02eb00b0f4ce2: function(arg0, arg1, arg2) {
            Uint8Array.prototype.set.call(getArrayU8FromWasm0(arg0, arg1), arg2);
        },
        __wbg_push_471a5b068a5295f6: function(arg0, arg1) {
            const ret = arg0.push(arg1);
            return ret;
        },
        __wbg_queueMicrotask_5d15a957e6aa920e: function(arg0) {
            queueMicrotask(arg0);
        },
        __wbg_queueMicrotask_f8819e5ffc402f36: function(arg0) {
            const ret = arg0.queueMicrotask;
            return ret;
        },
        __wbg_readyState_c78e609c7de3b381: function(arg0) {
            const ret = arg0.readyState;
            return ret;
        },
        __wbg_reason_e943590a4ef0d587: function(arg0, arg1) {
            const ret = arg1.reason;
            const ptr1 = passStringToWasm0(ret, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
            const len1 = WASM_VECTOR_LEN;
            getDataViewMemory0().setInt32(arg0 + 4 * 1, len1, true);
            getDataViewMemory0().setInt32(arg0 + 4 * 0, ptr1, true);
        },
        __wbg_removeItem_487c385a3066a8ed: function() { return handleError(function (arg0, arg1, arg2) {
            arg0.removeItem(getStringFromWasm0(arg1, arg2));
        }, arguments); },
        __wbg_remove_78d9bb173c1541b7: function(arg0, arg1, arg2) {
            arg0.remove(getStringFromWasm0(arg1, arg2));
        },
        __wbg_resolve_e6c466bc1052f16c: function(arg0) {
            const ret = Promise.resolve(arg0);
            return ret;
        },
        __wbg_send_15358dbe221c6258: function() { return handleError(function (arg0, arg1, arg2) {
            arg0.send(getStringFromWasm0(arg1, arg2));
        }, arguments); },
        __wbg_send_186c85704c7f2d00: function() { return handleError(function (arg0, arg1, arg2) {
            arg0.send(getArrayU8FromWasm0(arg1, arg2));
        }, arguments); },
        __wbg_setItem_e6399d3faae141dc: function() { return handleError(function (arg0, arg1, arg2, arg3, arg4) {
            arg0.setItem(getStringFromWasm0(arg1, arg2), getStringFromWasm0(arg3, arg4));
        }, arguments); },
        __wbg_setTimeout_f757f00851f76c42: function(arg0, arg1) {
            const ret = setTimeout(arg0, arg1);
            return ret;
        },
        __wbg_set_022bee52d0b05b19: function() { return handleError(function (arg0, arg1, arg2) {
            const ret = Reflect.set(arg0, arg1, arg2);
            return ret;
        }, arguments); },
        __wbg_set_100121392a24b659: function(arg0, arg1, arg2, arg3, arg4) {
            arg0.set(getStringFromWasm0(arg1, arg2), getStringFromWasm0(arg3, arg4));
        },
        __wbg_set_binaryType_770e68648ca5e83d: function(arg0, arg1) {
            arg0.binaryType = __wbindgen_enum_BinaryType[arg1];
        },
        __wbg_set_body_be11680f34217f75: function(arg0, arg1) {
            arg0.body = arg1;
        },
        __wbg_set_cache_968edea422613d1b: function(arg0, arg1) {
            arg0.cache = __wbindgen_enum_RequestCache[arg1];
        },
        __wbg_set_credentials_6577be90e0e85eb6: function(arg0, arg1) {
            arg0.credentials = __wbindgen_enum_RequestCredentials[arg1];
        },
        __wbg_set_headers_50fc01786240a440: function(arg0, arg1) {
            arg0.headers = arg1;
        },
        __wbg_set_method_c9f1f985f6b6c427: function(arg0, arg1, arg2) {
            arg0.method = getStringFromWasm0(arg1, arg2);
        },
        __wbg_set_mode_5e08d503428c06b9: function(arg0, arg1) {
            arg0.mode = __wbindgen_enum_RequestMode[arg1];
        },
        __wbg_set_onclose_17fa3bbcc4ba3541: function(arg0, arg1) {
            arg0.onclose = arg1;
        },
        __wbg_set_onerror_da99c4232662a084: function(arg0, arg1) {
            arg0.onerror = arg1;
        },
        __wbg_set_onmessage_c1db358b9c38e3f1: function(arg0, arg1) {
            arg0.onmessage = arg1;
        },
        __wbg_set_onopen_cd47b8fb1d92dee9: function(arg0, arg1) {
            arg0.onopen = arg1;
        },
        __wbg_set_signal_1d4e73c2305a0e7c: function(arg0, arg1) {
            arg0.signal = arg1;
        },
        __wbg_set_type_8b2743f6b4de4035: function(arg0, arg1, arg2) {
            arg0.type = getStringFromWasm0(arg1, arg2);
        },
        __wbg_signal_fdc54643b47bf85b: function(arg0) {
            const ret = arg0.signal;
            return ret;
        },
        __wbg_stack_3b0d974bbf31e44f: function(arg0, arg1) {
            const ret = arg1.stack;
            const ptr1 = passStringToWasm0(ret, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
            const len1 = WASM_VECTOR_LEN;
            getDataViewMemory0().setInt32(arg0 + 4 * 1, len1, true);
            getDataViewMemory0().setInt32(arg0 + 4 * 0, ptr1, true);
        },
        __wbg_static_accessor_GLOBAL_8cfadc87a297ca02: function() {
            const ret = typeof global === 'undefined' ? null : global;
            return isLikeNone(ret) ? 0 : addToExternrefTable0(ret);
        },
        __wbg_static_accessor_GLOBAL_THIS_602256ae5c8f42cf: function() {
            const ret = typeof globalThis === 'undefined' ? null : globalThis;
            return isLikeNone(ret) ? 0 : addToExternrefTable0(ret);
        },
        __wbg_static_accessor_SELF_e445c1c7484aecc3: function() {
            const ret = typeof self === 'undefined' ? null : self;
            return isLikeNone(ret) ? 0 : addToExternrefTable0(ret);
        },
        __wbg_static_accessor_WINDOW_f20e8576ef1e0f17: function() {
            const ret = typeof window === 'undefined' ? null : window;
            return isLikeNone(ret) ? 0 : addToExternrefTable0(ret);
        },
        __wbg_status_43e0d2f15b22d69f: function(arg0) {
            const ret = arg0.status;
            return ret;
        },
        __wbg_stringify_91082ed7a5a5769e: function() { return handleError(function (arg0) {
            const ret = JSON.stringify(arg0);
            return ret;
        }, arguments); },
        __wbg_then_792e0c862b060889: function(arg0, arg1, arg2) {
            const ret = arg0.then(arg1, arg2);
            return ret;
        },
        __wbg_then_8e16ee11f05e4827: function(arg0, arg1) {
            const ret = arg0.then(arg1);
            return ret;
        },
        __wbg_url_2bf741820e6563a0: function(arg0, arg1) {
            const ret = arg1.url;
            const ptr1 = passStringToWasm0(ret, wasm.__wbindgen_malloc, wasm.__wbindgen_realloc);
            const len1 = WASM_VECTOR_LEN;
            getDataViewMemory0().setInt32(arg0 + 4 * 1, len1, true);
            getDataViewMemory0().setInt32(arg0 + 4 * 0, ptr1, true);
        },
        __wbg_value_ee3a06f4579184fa: function(arg0) {
            const ret = arg0.value;
            return ret;
        },
        __wbindgen_cast_0000000000000001: function(arg0, arg1) {
            // Cast intrinsic for `Closure(Closure { owned: true, function: Function { arguments: [Externref], shim_idx: 2067, ret: Result(Unit), inner_ret: Some(Result(Unit)) }, mutable: true }) -> Externref`.
            const ret = makeMutClosure(arg0, arg1, wasm_bindgen__convert__closures_____invoke__hb7777b724403c6c8);
            return ret;
        },
        __wbindgen_cast_0000000000000002: function(arg0, arg1) {
            // Cast intrinsic for `Closure(Closure { owned: true, function: Function { arguments: [NamedExternref("CloseEvent")], shim_idx: 1941, ret: Unit, inner_ret: Some(Unit) }, mutable: true }) -> Externref`.
            const ret = makeMutClosure(arg0, arg1, wasm_bindgen__convert__closures_____invoke__h821db2d1ab133cd5);
            return ret;
        },
        __wbindgen_cast_0000000000000003: function(arg0, arg1) {
            // Cast intrinsic for `Closure(Closure { owned: true, function: Function { arguments: [NamedExternref("ErrorEvent")], shim_idx: 1941, ret: Unit, inner_ret: Some(Unit) }, mutable: true }) -> Externref`.
            const ret = makeMutClosure(arg0, arg1, wasm_bindgen__convert__closures_____invoke__h821db2d1ab133cd5_2);
            return ret;
        },
        __wbindgen_cast_0000000000000004: function(arg0, arg1) {
            // Cast intrinsic for `Closure(Closure { owned: true, function: Function { arguments: [NamedExternref("MessageEvent")], shim_idx: 1941, ret: Unit, inner_ret: Some(Unit) }, mutable: true }) -> Externref`.
            const ret = makeMutClosure(arg0, arg1, wasm_bindgen__convert__closures_____invoke__h821db2d1ab133cd5_3);
            return ret;
        },
        __wbindgen_cast_0000000000000005: function(arg0, arg1) {
            // Cast intrinsic for `Closure(Closure { owned: true, function: Function { arguments: [], shim_idx: 2029, ret: Unit, inner_ret: Some(Unit) }, mutable: true }) -> Externref`.
            const ret = makeMutClosure(arg0, arg1, wasm_bindgen__convert__closures_____invoke__hb409e0dd58c7af9a);
            return ret;
        },
        __wbindgen_cast_0000000000000006: function(arg0) {
            // Cast intrinsic for `F64 -> Externref`.
            const ret = arg0;
            return ret;
        },
        __wbindgen_cast_0000000000000007: function(arg0, arg1) {
            // Cast intrinsic for `Ref(String) -> Externref`.
            const ret = getStringFromWasm0(arg0, arg1);
            return ret;
        },
        __wbindgen_init_externref_table: function() {
            const table = wasm.__wbindgen_externrefs;
            const offset = table.grow(4);
            table.set(0, undefined);
            table.set(offset + 0, undefined);
            table.set(offset + 1, null);
            table.set(offset + 2, true);
            table.set(offset + 3, false);
        },
    };
    return {
        __proto__: null,
        "./agentsmesh_wasm_bg.js": import0,
    };
}

function wasm_bindgen__convert__closures_____invoke__hb409e0dd58c7af9a(arg0, arg1) {
    wasm.wasm_bindgen__convert__closures_____invoke__hb409e0dd58c7af9a(arg0, arg1);
}

function wasm_bindgen__convert__closures_____invoke__h821db2d1ab133cd5(arg0, arg1, arg2) {
    wasm.wasm_bindgen__convert__closures_____invoke__h821db2d1ab133cd5(arg0, arg1, arg2);
}

function wasm_bindgen__convert__closures_____invoke__h821db2d1ab133cd5_2(arg0, arg1, arg2) {
    wasm.wasm_bindgen__convert__closures_____invoke__h821db2d1ab133cd5_2(arg0, arg1, arg2);
}

function wasm_bindgen__convert__closures_____invoke__h821db2d1ab133cd5_3(arg0, arg1, arg2) {
    wasm.wasm_bindgen__convert__closures_____invoke__h821db2d1ab133cd5_3(arg0, arg1, arg2);
}

function wasm_bindgen__convert__closures_____invoke__hb7777b724403c6c8(arg0, arg1, arg2) {
    const ret = wasm.wasm_bindgen__convert__closures_____invoke__hb7777b724403c6c8(arg0, arg1, arg2);
    if (ret[1]) {
        throw takeFromExternrefTable0(ret[0]);
    }
}

function wasm_bindgen__convert__closures_____invoke__h234923932d6562e7(arg0, arg1, arg2, arg3) {
    wasm.wasm_bindgen__convert__closures_____invoke__h234923932d6562e7(arg0, arg1, arg2, arg3);
}


const __wbindgen_enum_BinaryType = ["blob", "arraybuffer"];


const __wbindgen_enum_RequestCache = ["default", "no-store", "reload", "no-cache", "force-cache", "only-if-cached"];


const __wbindgen_enum_RequestCredentials = ["omit", "same-origin", "include"];


const __wbindgen_enum_RequestMode = ["same-origin", "no-cors", "cors", "navigate"];
const WasmAcpSessionManagerFinalization = (typeof FinalizationRegistry === 'undefined')
    ? { register: () => {}, unregister: () => {} }
    : new FinalizationRegistry(ptr => wasm.__wbg_wasmacpsessionmanager_free(ptr >>> 0, 1));
const WasmAgentServiceFinalization = (typeof FinalizationRegistry === 'undefined')
    ? { register: () => {}, unregister: () => {} }
    : new FinalizationRegistry(ptr => wasm.__wbg_wasmagentservice_free(ptr >>> 0, 1));
const WasmApiClientFinalization = (typeof FinalizationRegistry === 'undefined')
    ? { register: () => {}, unregister: () => {} }
    : new FinalizationRegistry(ptr => wasm.__wbg_wasmapiclient_free(ptr >>> 0, 1));
const WasmApiKeyServiceFinalization = (typeof FinalizationRegistry === 'undefined')
    ? { register: () => {}, unregister: () => {} }
    : new FinalizationRegistry(ptr => wasm.__wbg_wasmapikeyservice_free(ptr >>> 0, 1));
const WasmAppStateFinalization = (typeof FinalizationRegistry === 'undefined')
    ? { register: () => {}, unregister: () => {} }
    : new FinalizationRegistry(ptr => wasm.__wbg_wasmappstate_free(ptr >>> 0, 1));
const WasmAuthApiServiceFinalization = (typeof FinalizationRegistry === 'undefined')
    ? { register: () => {}, unregister: () => {} }
    : new FinalizationRegistry(ptr => wasm.__wbg_wasmauthapiservice_free(ptr >>> 0, 1));
const WasmAuthManagerFinalization = (typeof FinalizationRegistry === 'undefined')
    ? { register: () => {}, unregister: () => {} }
    : new FinalizationRegistry(ptr => wasm.__wbg_wasmauthmanager_free(ptr >>> 0, 1));
const WasmAutopilotServiceFinalization = (typeof FinalizationRegistry === 'undefined')
    ? { register: () => {}, unregister: () => {} }
    : new FinalizationRegistry(ptr => wasm.__wbg_wasmautopilotservice_free(ptr >>> 0, 1));
const WasmAutopilotStateFinalization = (typeof FinalizationRegistry === 'undefined')
    ? { register: () => {}, unregister: () => {} }
    : new FinalizationRegistry(ptr => wasm.__wbg_wasmautopilotstate_free(ptr >>> 0, 1));
const WasmBillingServiceFinalization = (typeof FinalizationRegistry === 'undefined')
    ? { register: () => {}, unregister: () => {} }
    : new FinalizationRegistry(ptr => wasm.__wbg_wasmbillingservice_free(ptr >>> 0, 1));
const WasmBindingServiceFinalization = (typeof FinalizationRegistry === 'undefined')
    ? { register: () => {}, unregister: () => {} }
    : new FinalizationRegistry(ptr => wasm.__wbg_wasmbindingservice_free(ptr >>> 0, 1));
const WasmBlockstoreServiceFinalization = (typeof FinalizationRegistry === 'undefined')
    ? { register: () => {}, unregister: () => {} }
    : new FinalizationRegistry(ptr => wasm.__wbg_wasmblockstoreservice_free(ptr >>> 0, 1));
const WasmChannelServiceFinalization = (typeof FinalizationRegistry === 'undefined')
    ? { register: () => {}, unregister: () => {} }
    : new FinalizationRegistry(ptr => wasm.__wbg_wasmchannelservice_free(ptr >>> 0, 1));
const WasmChannelStateFinalization = (typeof FinalizationRegistry === 'undefined')
    ? { register: () => {}, unregister: () => {} }
    : new FinalizationRegistry(ptr => wasm.__wbg_wasmchannelstate_free(ptr >>> 0, 1));
const WasmEventsManagerFinalization = (typeof FinalizationRegistry === 'undefined')
    ? { register: () => {}, unregister: () => {} }
    : new FinalizationRegistry(ptr => wasm.__wbg_wasmeventsmanager_free(ptr >>> 0, 1));
const WasmExtensionServiceFinalization = (typeof FinalizationRegistry === 'undefined')
    ? { register: () => {}, unregister: () => {} }
    : new FinalizationRegistry(ptr => wasm.__wbg_wasmextensionservice_free(ptr >>> 0, 1));
const WasmFileServiceFinalization = (typeof FinalizationRegistry === 'undefined')
    ? { register: () => {}, unregister: () => {} }
    : new FinalizationRegistry(ptr => wasm.__wbg_wasmfileservice_free(ptr >>> 0, 1));
const WasmGitProviderStateFinalization = (typeof FinalizationRegistry === 'undefined')
    ? { register: () => {}, unregister: () => {} }
    : new FinalizationRegistry(ptr => wasm.__wbg_wasmgitproviderstate_free(ptr >>> 0, 1));
const WasmGrantServiceFinalization = (typeof FinalizationRegistry === 'undefined')
    ? { register: () => {}, unregister: () => {} }
    : new FinalizationRegistry(ptr => wasm.__wbg_wasmgrantservice_free(ptr >>> 0, 1));
const WasmInvitationServiceFinalization = (typeof FinalizationRegistry === 'undefined')
    ? { register: () => {}, unregister: () => {} }
    : new FinalizationRegistry(ptr => wasm.__wbg_wasminvitationservice_free(ptr >>> 0, 1));
const WasmLoopServiceFinalization = (typeof FinalizationRegistry === 'undefined')
    ? { register: () => {}, unregister: () => {} }
    : new FinalizationRegistry(ptr => wasm.__wbg_wasmloopservice_free(ptr >>> 0, 1));
const WasmLoopStateFinalization = (typeof FinalizationRegistry === 'undefined')
    ? { register: () => {}, unregister: () => {} }
    : new FinalizationRegistry(ptr => wasm.__wbg_wasmloopstate_free(ptr >>> 0, 1));
const WasmMeshServiceFinalization = (typeof FinalizationRegistry === 'undefined')
    ? { register: () => {}, unregister: () => {} }
    : new FinalizationRegistry(ptr => wasm.__wbg_wasmmeshservice_free(ptr >>> 0, 1));
const WasmMeshStateFinalization = (typeof FinalizationRegistry === 'undefined')
    ? { register: () => {}, unregister: () => {} }
    : new FinalizationRegistry(ptr => wasm.__wbg_wasmmeshstate_free(ptr >>> 0, 1));
const WasmMessageServiceFinalization = (typeof FinalizationRegistry === 'undefined')
    ? { register: () => {}, unregister: () => {} }
    : new FinalizationRegistry(ptr => wasm.__wbg_wasmmessageservice_free(ptr >>> 0, 1));
const WasmNotificationServiceFinalization = (typeof FinalizationRegistry === 'undefined')
    ? { register: () => {}, unregister: () => {} }
    : new FinalizationRegistry(ptr => wasm.__wbg_wasmnotificationservice_free(ptr >>> 0, 1));
const WasmOrgApiServiceFinalization = (typeof FinalizationRegistry === 'undefined')
    ? { register: () => {}, unregister: () => {} }
    : new FinalizationRegistry(ptr => wasm.__wbg_wasmorgapiservice_free(ptr >>> 0, 1));
const WasmOrgStateFinalization = (typeof FinalizationRegistry === 'undefined')
    ? { register: () => {}, unregister: () => {} }
    : new FinalizationRegistry(ptr => wasm.__wbg_wasmorgstate_free(ptr >>> 0, 1));
const WasmPodServiceFinalization = (typeof FinalizationRegistry === 'undefined')
    ? { register: () => {}, unregister: () => {} }
    : new FinalizationRegistry(ptr => wasm.__wbg_wasmpodservice_free(ptr >>> 0, 1));
const WasmPodStateFinalization = (typeof FinalizationRegistry === 'undefined')
    ? { register: () => {}, unregister: () => {} }
    : new FinalizationRegistry(ptr => wasm.__wbg_wasmpodstate_free(ptr >>> 0, 1));
const WasmPromoCodeServiceFinalization = (typeof FinalizationRegistry === 'undefined')
    ? { register: () => {}, unregister: () => {} }
    : new FinalizationRegistry(ptr => wasm.__wbg_wasmpromocodeservice_free(ptr >>> 0, 1));
const WasmRelayManagerFinalization = (typeof FinalizationRegistry === 'undefined')
    ? { register: () => {}, unregister: () => {} }
    : new FinalizationRegistry(ptr => wasm.__wbg_wasmrelaymanager_free(ptr >>> 0, 1));
const WasmRepoStateFinalization = (typeof FinalizationRegistry === 'undefined')
    ? { register: () => {}, unregister: () => {} }
    : new FinalizationRegistry(ptr => wasm.__wbg_wasmrepostate_free(ptr >>> 0, 1));
const WasmRepositoryServiceFinalization = (typeof FinalizationRegistry === 'undefined')
    ? { register: () => {}, unregister: () => {} }
    : new FinalizationRegistry(ptr => wasm.__wbg_wasmrepositoryservice_free(ptr >>> 0, 1));
const WasmRunnerServiceFinalization = (typeof FinalizationRegistry === 'undefined')
    ? { register: () => {}, unregister: () => {} }
    : new FinalizationRegistry(ptr => wasm.__wbg_wasmrunnerservice_free(ptr >>> 0, 1));
const WasmRunnerStateFinalization = (typeof FinalizationRegistry === 'undefined')
    ? { register: () => {}, unregister: () => {} }
    : new FinalizationRegistry(ptr => wasm.__wbg_wasmrunnerstate_free(ptr >>> 0, 1));
const WasmSSOServiceFinalization = (typeof FinalizationRegistry === 'undefined')
    ? { register: () => {}, unregister: () => {} }
    : new FinalizationRegistry(ptr => wasm.__wbg_wasmssoservice_free(ptr >>> 0, 1));
const WasmSupportTicketServiceFinalization = (typeof FinalizationRegistry === 'undefined')
    ? { register: () => {}, unregister: () => {} }
    : new FinalizationRegistry(ptr => wasm.__wbg_wasmsupportticketservice_free(ptr >>> 0, 1));
const WasmTicketRelationsServiceFinalization = (typeof FinalizationRegistry === 'undefined')
    ? { register: () => {}, unregister: () => {} }
    : new FinalizationRegistry(ptr => wasm.__wbg_wasmticketrelationsservice_free(ptr >>> 0, 1));
const WasmTicketServiceFinalization = (typeof FinalizationRegistry === 'undefined')
    ? { register: () => {}, unregister: () => {} }
    : new FinalizationRegistry(ptr => wasm.__wbg_wasmticketservice_free(ptr >>> 0, 1));
const WasmTicketStateFinalization = (typeof FinalizationRegistry === 'undefined')
    ? { register: () => {}, unregister: () => {} }
    : new FinalizationRegistry(ptr => wasm.__wbg_wasmticketstate_free(ptr >>> 0, 1));
const WasmTokenUsageServiceFinalization = (typeof FinalizationRegistry === 'undefined')
    ? { register: () => {}, unregister: () => {} }
    : new FinalizationRegistry(ptr => wasm.__wbg_wasmtokenusageservice_free(ptr >>> 0, 1));
const WasmUserApiServiceFinalization = (typeof FinalizationRegistry === 'undefined')
    ? { register: () => {}, unregister: () => {} }
    : new FinalizationRegistry(ptr => wasm.__wbg_wasmuserapiservice_free(ptr >>> 0, 1));
const WasmUserCredentialServiceFinalization = (typeof FinalizationRegistry === 'undefined')
    ? { register: () => {}, unregister: () => {} }
    : new FinalizationRegistry(ptr => wasm.__wbg_wasmusercredentialservice_free(ptr >>> 0, 1));
const WasmUserStateFinalization = (typeof FinalizationRegistry === 'undefined')
    ? { register: () => {}, unregister: () => {} }
    : new FinalizationRegistry(ptr => wasm.__wbg_wasmuserstate_free(ptr >>> 0, 1));
const WasmWebSocketFinalization = (typeof FinalizationRegistry === 'undefined')
    ? { register: () => {}, unregister: () => {} }
    : new FinalizationRegistry(ptr => wasm.__wbg_wasmwebsocket_free(ptr >>> 0, 1));

function addToExternrefTable0(obj) {
    const idx = wasm.__externref_table_alloc();
    wasm.__wbindgen_externrefs.set(idx, obj);
    return idx;
}

const CLOSURE_DTORS = (typeof FinalizationRegistry === 'undefined')
    ? { register: () => {}, unregister: () => {} }
    : new FinalizationRegistry(state => wasm.__wbindgen_destroy_closure(state.a, state.b));

function debugString(val) {
    // primitive types
    const type = typeof val;
    if (type == 'number' || type == 'boolean' || val == null) {
        return  `${val}`;
    }
    if (type == 'string') {
        return `"${val}"`;
    }
    if (type == 'symbol') {
        const description = val.description;
        if (description == null) {
            return 'Symbol';
        } else {
            return `Symbol(${description})`;
        }
    }
    if (type == 'function') {
        const name = val.name;
        if (typeof name == 'string' && name.length > 0) {
            return `Function(${name})`;
        } else {
            return 'Function';
        }
    }
    // objects
    if (Array.isArray(val)) {
        const length = val.length;
        let debug = '[';
        if (length > 0) {
            debug += debugString(val[0]);
        }
        for(let i = 1; i < length; i++) {
            debug += ', ' + debugString(val[i]);
        }
        debug += ']';
        return debug;
    }
    // Test for built-in
    const builtInMatches = /\[object ([^\]]+)\]/.exec(toString.call(val));
    let className;
    if (builtInMatches && builtInMatches.length > 1) {
        className = builtInMatches[1];
    } else {
        // Failed to match the standard '[object ClassName]'
        return toString.call(val);
    }
    if (className == 'Object') {
        // we're a user defined class or Object
        // JSON.stringify avoids problems with cycles, and is generally much
        // easier than looping through ownProperties of `val`.
        try {
            return 'Object(' + JSON.stringify(val) + ')';
        } catch (_) {
            return 'Object';
        }
    }
    // errors
    if (val instanceof Error) {
        return `${val.name}: ${val.message}\n${val.stack}`;
    }
    // TODO we could test for more things here, like `Set`s and `Map`s.
    return className;
}

function getArrayU8FromWasm0(ptr, len) {
    ptr = ptr >>> 0;
    return getUint8ArrayMemory0().subarray(ptr / 1, ptr / 1 + len);
}

let cachedDataViewMemory0 = null;
function getDataViewMemory0() {
    if (cachedDataViewMemory0 === null || cachedDataViewMemory0.buffer.detached === true || (cachedDataViewMemory0.buffer.detached === undefined && cachedDataViewMemory0.buffer !== wasm.memory.buffer)) {
        cachedDataViewMemory0 = new DataView(wasm.memory.buffer);
    }
    return cachedDataViewMemory0;
}

function getStringFromWasm0(ptr, len) {
    ptr = ptr >>> 0;
    return decodeText(ptr, len);
}

let cachedUint8ArrayMemory0 = null;
function getUint8ArrayMemory0() {
    if (cachedUint8ArrayMemory0 === null || cachedUint8ArrayMemory0.byteLength === 0) {
        cachedUint8ArrayMemory0 = new Uint8Array(wasm.memory.buffer);
    }
    return cachedUint8ArrayMemory0;
}

function handleError(f, args) {
    try {
        return f.apply(this, args);
    } catch (e) {
        const idx = addToExternrefTable0(e);
        wasm.__wbindgen_exn_store(idx);
    }
}

function isLikeNone(x) {
    return x === undefined || x === null;
}

function makeMutClosure(arg0, arg1, f) {
    const state = { a: arg0, b: arg1, cnt: 1 };
    const real = (...args) => {

        // First up with a closure we increment the internal reference
        // count. This ensures that the Rust closure environment won't
        // be deallocated while we're invoking it.
        state.cnt++;
        const a = state.a;
        state.a = 0;
        try {
            return f(a, state.b, ...args);
        } finally {
            state.a = a;
            real._wbg_cb_unref();
        }
    };
    real._wbg_cb_unref = () => {
        if (--state.cnt === 0) {
            wasm.__wbindgen_destroy_closure(state.a, state.b);
            state.a = 0;
            CLOSURE_DTORS.unregister(state);
        }
    };
    CLOSURE_DTORS.register(real, state, state);
    return real;
}

function passArray8ToWasm0(arg, malloc) {
    const ptr = malloc(arg.length * 1, 1) >>> 0;
    getUint8ArrayMemory0().set(arg, ptr / 1);
    WASM_VECTOR_LEN = arg.length;
    return ptr;
}

function passArrayJsValueToWasm0(array, malloc) {
    const ptr = malloc(array.length * 4, 4) >>> 0;
    for (let i = 0; i < array.length; i++) {
        const add = addToExternrefTable0(array[i]);
        getDataViewMemory0().setUint32(ptr + 4 * i, add, true);
    }
    WASM_VECTOR_LEN = array.length;
    return ptr;
}

function passStringToWasm0(arg, malloc, realloc) {
    if (realloc === undefined) {
        const buf = cachedTextEncoder.encode(arg);
        const ptr = malloc(buf.length, 1) >>> 0;
        getUint8ArrayMemory0().subarray(ptr, ptr + buf.length).set(buf);
        WASM_VECTOR_LEN = buf.length;
        return ptr;
    }

    let len = arg.length;
    let ptr = malloc(len, 1) >>> 0;

    const mem = getUint8ArrayMemory0();

    let offset = 0;

    for (; offset < len; offset++) {
        const code = arg.charCodeAt(offset);
        if (code > 0x7F) break;
        mem[ptr + offset] = code;
    }
    if (offset !== len) {
        if (offset !== 0) {
            arg = arg.slice(offset);
        }
        ptr = realloc(ptr, len, len = offset + arg.length * 3, 1) >>> 0;
        const view = getUint8ArrayMemory0().subarray(ptr + offset, ptr + len);
        const ret = cachedTextEncoder.encodeInto(arg, view);

        offset += ret.written;
        ptr = realloc(ptr, len, offset, 1) >>> 0;
    }

    WASM_VECTOR_LEN = offset;
    return ptr;
}

function takeFromExternrefTable0(idx) {
    const value = wasm.__wbindgen_externrefs.get(idx);
    wasm.__externref_table_dealloc(idx);
    return value;
}

let cachedTextDecoder = new TextDecoder('utf-8', { ignoreBOM: true, fatal: true });
cachedTextDecoder.decode();
const MAX_SAFARI_DECODE_BYTES = 2146435072;
let numBytesDecoded = 0;
function decodeText(ptr, len) {
    numBytesDecoded += len;
    if (numBytesDecoded >= MAX_SAFARI_DECODE_BYTES) {
        cachedTextDecoder = new TextDecoder('utf-8', { ignoreBOM: true, fatal: true });
        cachedTextDecoder.decode();
        numBytesDecoded = len;
    }
    return cachedTextDecoder.decode(getUint8ArrayMemory0().subarray(ptr, ptr + len));
}

const cachedTextEncoder = new TextEncoder();

if (!('encodeInto' in cachedTextEncoder)) {
    cachedTextEncoder.encodeInto = function (arg, view) {
        const buf = cachedTextEncoder.encode(arg);
        view.set(buf);
        return {
            read: arg.length,
            written: buf.length
        };
    };
}

let WASM_VECTOR_LEN = 0;

let wasmModule, wasm;
function __wbg_finalize_init(instance, module) {
    wasm = instance.exports;
    wasmModule = module;
    cachedDataViewMemory0 = null;
    cachedUint8ArrayMemory0 = null;
    wasm.__wbindgen_start();
    return wasm;
}

async function __wbg_load(module, imports) {
    if (typeof Response === 'function' && module instanceof Response) {
        if (typeof WebAssembly.instantiateStreaming === 'function') {
            try {
                return await WebAssembly.instantiateStreaming(module, imports);
            } catch (e) {
                const validResponse = module.ok && expectedResponseType(module.type);

                if (validResponse && module.headers.get('Content-Type') !== 'application/wasm') {
                    console.warn("`WebAssembly.instantiateStreaming` failed because your server does not serve Wasm with `application/wasm` MIME type. Falling back to `WebAssembly.instantiate` which is slower. Original error:\n", e);

                } else { throw e; }
            }
        }

        const bytes = await module.arrayBuffer();
        return await WebAssembly.instantiate(bytes, imports);
    } else {
        const instance = await WebAssembly.instantiate(module, imports);

        if (instance instanceof WebAssembly.Instance) {
            return { instance, module };
        } else {
            return instance;
        }
    }

    function expectedResponseType(type) {
        switch (type) {
            case 'basic': case 'cors': case 'default': return true;
        }
        return false;
    }
}

function initSync(module) {
    if (wasm !== undefined) return wasm;


    if (module !== undefined) {
        if (Object.getPrototypeOf(module) === Object.prototype) {
            ({module} = module)
        } else {
            console.warn('using deprecated parameters for `initSync()`; pass a single object instead')
        }
    }

    const imports = __wbg_get_imports();
    if (!(module instanceof WebAssembly.Module)) {
        module = new WebAssembly.Module(module);
    }
    const instance = new WebAssembly.Instance(module, imports);
    return __wbg_finalize_init(instance, module);
}

async function __wbg_init(module_or_path) {
    if (wasm !== undefined) return wasm;


    if (module_or_path !== undefined) {
        if (Object.getPrototypeOf(module_or_path) === Object.prototype) {
            ({module_or_path} = module_or_path)
        } else {
            console.warn('using deprecated parameters for the initialization function; pass a single object instead')
        }
    }

    if (module_or_path === undefined) {
        module_or_path = new URL('agentsmesh_wasm_bg.wasm', import.meta.url);
    }
    const imports = __wbg_get_imports();

    if (typeof module_or_path === 'string' || (typeof Request === 'function' && module_or_path instanceof Request) || (typeof URL === 'function' && module_or_path instanceof URL)) {
        module_or_path = fetch(module_or_path);
    }

    const { instance, module } = await __wbg_load(await module_or_path, imports);

    return __wbg_finalize_init(instance, module);
}

export { initSync, __wbg_init as default };
