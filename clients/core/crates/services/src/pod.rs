use std::sync::Arc;
use std::sync::RwLock;

use agentsmesh_api_client::ApiClient;
use agentsmesh_state::pod_state::PodState;
use agentsmesh_types::proto_pod_v1 as pod_proto;
use agentsmesh_types::{Pod, PodStatus, CreatePodRequest};
use prost::Message;

use crate::parse_status;

fn sidebar_status_param(filter: &str) -> Option<&'static str> {
    match filter {
        "mine" => Some("running,initializing"),
        "org" => Some("running,initializing"),
        "completed" => Some("terminated,failed,paused,completed,error"),
        _ => None,
    }
}

const SIDEBAR_PAGE_SIZE: i64 = 20;

pub struct PodService {
    client: Arc<ApiClient>,
    state: RwLock<PodState>,
}

impl PodService {
    pub fn new(client: Arc<ApiClient>, state: PodState) -> Self {
        Self { client, state: RwLock::new(state) }
    }

    pub fn pods_json(&self) -> String {
        serde_json::to_string(self.state.read().unwrap().pods()).unwrap_or_default()
    }

    pub fn current_pod_json(&self) -> Option<String> {
        self.state.read().unwrap().current_pod()
            .map(|pod| serde_json::to_string(pod).unwrap_or_default())
    }

    pub fn get_pod_json(&self, pod_key: &str) -> Option<String> {
        self.state.read().unwrap().get_pod(pod_key)
            .map(|pod| serde_json::to_string(pod).unwrap_or_default())
    }

    pub fn upsert_pod(&self, pod_json: &str, timestamp: Option<i64>) {
        if let Ok(pod) = serde_json::from_str::<Pod>(pod_json) {
            self.state.write().unwrap().upsert_pod(pod, timestamp);
        }
    }

    pub fn set_pods(&self, pods_json: &str) {
        if let Ok(pods) = serde_json::from_str::<Vec<Pod>>(pods_json) {
            self.state.write().unwrap().set_pods(pods);
        }
    }

    pub fn set_current_pod(&self, pod_json: &str) {
        let pod = if pod_json.is_empty() {
            None
        } else {
            serde_json::from_str::<Pod>(pod_json).ok()
        };
        self.state.write().unwrap().set_current_pod(pod);
    }

    pub fn update_pod_status(
        &self, pod_key: &str, status: &str,
        agent_status: Option<String>, error_code: Option<String>,
        error_message: Option<String>, timestamp: Option<i64>,
    ) {
        let parsed = parse_status::<PodStatus>(status);
        self.state.write().unwrap().update_pod_status(
            pod_key, parsed, agent_status.as_deref(),
            error_code.as_deref(), error_message.as_deref(), timestamp,
        );
    }

    pub fn update_pod_title(&self, pod_key: &str, title: &str, timestamp: Option<i64>) {
        self.state.write().unwrap().update_pod_title(pod_key, title, timestamp);
    }

    pub fn update_pod_alias(&self, pod_key: &str, alias: &str) {
        self.state.write().unwrap().update_pod_alias(pod_key, alias);
    }

    pub fn update_agent_status(&self, pod_key: &str, agent_status: &str) {
        self.state.write().unwrap().update_agent_status(pod_key, agent_status);
    }

    pub fn remove_pod(&self, pod_key: &str) {
        self.state.write().unwrap().remove_pod(pod_key);
    }

    pub async fn fetch_pods(
        &self, status: Option<String>, runner_id: Option<i64>,
        created_by_id: Option<i64>, limit: Option<i64>, offset: Option<i64>,
    ) -> Result<String, String> {
        // runner_id filter is REST-only — proto.pod.v1.ListPodsRequest drops
        // it because the new surface routes per-runner lookup through
        // RunnerService.QuerySandboxes / ListRunnerPods. The few legacy
        // callers passing runner_id (none in active web code post-PR #340)
        // would have to switch to the runner-scoped endpoint.
        let _ = runner_id;
        let req = pod_proto::ListPodsRequest {
            org_slug: self.client.current_org_slug(),
            status,
            created_by_id,
            limit: limit.map(|v| v as i32),
            offset: offset.map(|v| v as i32),
        };
        let resp = self.client.list_pods_connect(&req).await.map_err(crate::wire)?;
        let total = resp.total;
        let resp_limit = resp.limit;
        let resp_offset = resp.offset;
        let pods: Vec<Pod> = resp.items.into_iter().map(crate::proto_convert::pod::from_proto).collect();
        self.state.write().unwrap().set_pods(pods.clone());
        let envelope = serde_json::json!({
            "pods": pods,
            "total": total,
            "limit": resp_limit,
            "offset": resp_offset,
        });
        serde_json::to_string(&envelope).map_err(crate::wire)
    }

    pub async fn fetch_sidebar_pods(
        &self, filter: &str, user_id: Option<i64>,
    ) -> Result<String, String> {
        let status_param = sidebar_status_param(filter).map(String::from);
        let created_by_id = if filter == "mine" { user_id } else { None };
        let req = pod_proto::ListPodsRequest {
            org_slug: self.client.current_org_slug(),
            status: status_param,
            created_by_id,
            limit: Some(SIDEBAR_PAGE_SIZE as i32),
            offset: Some(0),
        };
        let resp = self.client.list_pods_connect(&req).await.map_err(crate::wire)?;
        let total = resp.total;
        let pods: Vec<Pod> = resp.items.into_iter().map(crate::proto_convert::pod::from_proto).collect();
        let has_more = (pods.len() as i64) < total;
        self.state.write().unwrap().set_pods(pods.clone());
        let result = serde_json::json!({
            "pods": pods,
            "total": total,
            "hasMore": has_more,
        });
        serde_json::to_string(&result).map_err(crate::wire)
    }

    pub async fn load_more_pods(
        &self, filter: &str, user_id: Option<i64>, offset: i64,
    ) -> Result<String, String> {
        let status_param = sidebar_status_param(filter).map(String::from);
        let created_by_id = if filter == "mine" { user_id } else { None };
        let req = pod_proto::ListPodsRequest {
            org_slug: self.client.current_org_slug(),
            status: status_param,
            created_by_id,
            limit: Some(SIDEBAR_PAGE_SIZE as i32),
            offset: Some(offset as i32),
        };
        let resp = self.client.list_pods_connect(&req).await.map_err(crate::wire)?;
        let total = resp.total;
        let new_pods: Vec<Pod> = resp.items.into_iter().map(crate::proto_convert::pod::from_proto).collect();
        {
            let mut state = self.state.write().unwrap();
            for pod in &new_pods {
                state.upsert_pod(pod.clone(), None);
            }
        }
        let all_count = self.state.read().unwrap().pods().len() as i64;
        let has_more = all_count < total;
        let result = serde_json::json!({
            "newPods": new_pods,
            "total": total,
            "hasMore": has_more,
            "allCount": all_count,
        });
        serde_json::to_string(&result).map_err(crate::wire)
    }

    pub async fn fetch_pod(&self, pod_key: &str) -> Result<String, String> {
        let req = pod_proto::GetPodRequest {
            org_slug: self.client.current_org_slug(),
            pod_key: pod_key.to_string(),
        };
        let resp = self.client.get_pod_connect(&req).await.map_err(crate::wire)?;
        let pod = crate::proto_convert::pod::from_proto(resp);
        self.state.write().unwrap().upsert_pod(pod.clone(), None);
        serde_json::to_string(&pod).map_err(crate::wire)
    }

    pub async fn create_pod(&self, request_json: &str) -> Result<String, String> {
        let req_legacy: CreatePodRequest = serde_json::from_str(request_json)
            .map_err(crate::wire)?;
        let req = pod_proto::CreatePodRequest {
            org_slug: self.client.current_org_slug(),
            agent_slug: req_legacy.agent_slug,
            runner_id: req_legacy.runner_id,
            ticket_slug: req_legacy.ticket_slug,
            alias: req_legacy.alias,
            agentfile_layer: req_legacy.agentfile_layer,
            repository_id: None,
            credential_profile_id: None,
            cols: req_legacy.cols.map(|v| v as i32).unwrap_or(0),
            rows: req_legacy.rows.map(|v| v as i32).unwrap_or(0),
            source_pod_key: req_legacy.source_pod_key,
            resume_agent_session: req_legacy.resume_agent_session,
            perpetual: req_legacy.perpetual,
        };
        let resp = self.client.create_pod_connect(&req).await.map_err(crate::wire)?;
        let pod_proto_msg = resp.pod.ok_or_else(|| "create_pod: empty pod in response".to_string())?;
        let pod = crate::proto_convert::pod::from_proto(pod_proto_msg);
        self.state.write().unwrap().upsert_pod(pod.clone(), None);
        let envelope = serde_json::json!({
            "pod": pod,
            "warning": resp.warning,
        });
        serde_json::to_string(&envelope).map_err(crate::wire)
    }

    pub async fn terminate_pod(&self, pod_key: &str) -> Result<(), String> {
        let req = pod_proto::TerminatePodRequest {
            org_slug: self.client.current_org_slug(),
            pod_key: pod_key.to_string(),
        };
        self.client.terminate_pod_connect(&req).await.map_err(crate::wire)?;
        self.state.write().unwrap().update_pod_status(
            pod_key, PodStatus::Terminated, None, None, None, None,
        );
        Ok(())
    }

    pub async fn update_pod_alias_api(
        &self, pod_key: &str, alias: Option<String>,
    ) -> Result<(), String> {
        self.state.write().unwrap().update_pod_alias(pod_key, alias.as_deref().unwrap_or(""));
        let req = pod_proto::UpdatePodAliasRequest {
            org_slug: self.client.current_org_slug(),
            pod_key: pod_key.to_string(),
            alias: alias.clone(),
        };
        match self.client.update_pod_alias_connect(&req).await {
            Ok(_) => Ok(()),
            Err(e) => {
                // Roll back local optimistic state to server truth on error.
                let get_req = pod_proto::GetPodRequest {
                    org_slug: self.client.current_org_slug(),
                    pod_key: pod_key.to_string(),
                };
                if let Ok(pod_proto_msg) = self.client.get_pod_connect(&get_req).await {
                    let pod = crate::proto_convert::pod::from_proto(pod_proto_msg);
                    self.state.write().unwrap().upsert_pod(pod, None);
                }
                Err(e.to_string())
            }
        }
    }

    pub async fn get_pod_connection(&self, pod_key: &str) -> Result<String, String> {
        let req = pod_proto::GetPodConnectionRequest {
            org_slug: self.client.current_org_slug(),
            pod_key: pod_key.to_string(),
        };
        let info_proto = self.client.get_pod_connection_connect(&req).await.map_err(crate::wire)?;
        let info = crate::proto_convert::pod::connection_info(info_proto);
        serde_json::to_string(&info).map_err(crate::wire)
    }

    // -------- Connect-RPC (binary wire) --------
    //
    // Each `*_connect` method takes prost-encoded bytes and returns
    // prost-encoded bytes — matching the wasm bridge's `Result<Vec<u8>, String>`
    // surface (conventions §2.5). Caller (TS) encodes via
    // @bufbuild/protobuf .toBinary() and decodes via .fromBinary().

    pub async fn list_pods_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = pod_proto::ListPodsRequest::decode(request_bytes)
            .map_err(|e| format!("decode list_pods request: {e}"))?;
        let resp = self.client.list_pods_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn get_pod_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = pod_proto::GetPodRequest::decode(request_bytes)
            .map_err(|e| format!("decode get_pod request: {e}"))?;
        let resp = self.client.get_pod_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn create_pod_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = pod_proto::CreatePodRequest::decode(request_bytes)
            .map_err(|e| format!("decode create_pod request: {e}"))?;
        let resp = self.client.create_pod_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn terminate_pod_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = pod_proto::TerminatePodRequest::decode(request_bytes)
            .map_err(|e| format!("decode terminate_pod request: {e}"))?;
        let resp = self.client.terminate_pod_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn update_pod_alias_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = pod_proto::UpdatePodAliasRequest::decode(request_bytes)
            .map_err(|e| format!("decode update_pod_alias request: {e}"))?;
        let resp = self.client.update_pod_alias_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn update_pod_perpetual_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = pod_proto::UpdatePodPerpetualRequest::decode(request_bytes)
            .map_err(|e| format!("decode update_pod_perpetual request: {e}"))?;
        let resp = self.client.update_pod_perpetual_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn get_pod_connection_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = pod_proto::GetPodConnectionRequest::decode(request_bytes)
            .map_err(|e| format!("decode get_pod_connection request: {e}"))?;
        let resp = self.client.get_pod_connection_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn send_pod_prompt_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = pod_proto::SendPodPromptRequest::decode(request_bytes)
            .map_err(|e| format!("decode send_pod_prompt request: {e}"))?;
        let resp = self.client.send_pod_prompt_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn list_pods_by_ticket_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = pod_proto::ListPodsByTicketRequest::decode(request_bytes)
            .map_err(|e| format!("decode list_pods_by_ticket request: {e}"))?;
        let resp = self.client.list_pods_by_ticket_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }
}
