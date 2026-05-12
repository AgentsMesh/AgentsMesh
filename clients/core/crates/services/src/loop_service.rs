use std::sync::Arc;
use std::sync::RwLock;

use agentsmesh_api_client::ApiClient;
use agentsmesh_state::loop_state::LoopState;
use agentsmesh_types::{
    LoopData, LoopRunData, LoopRunStatus,
    CreateLoopRequest, UpdateLoopRequest,
};

use crate::parse_status;

pub struct LoopService {
    client: Arc<ApiClient>,
    state: RwLock<LoopState>,
}

impl LoopService {
    pub fn new(client: Arc<ApiClient>, state: LoopState) -> Self {
        Self { client, state: RwLock::new(state) }
    }

    pub fn loops_json(&self) -> String {
        serde_json::to_string(self.state.read().unwrap().get_loops()).unwrap_or_default()
    }

    pub fn current_loop_json(&self) -> Option<String> {
        self.state.read().unwrap().get_current_loop()
            .map(|l| serde_json::to_string(l).unwrap_or_default())
    }

    pub fn runs_json(&self) -> String {
        serde_json::to_string(self.state.read().unwrap().get_runs()).unwrap_or_default()
    }

    pub fn get_loop_by_slug_json(&self, slug: &str) -> Option<String> {
        self.state.read().unwrap().get_loop_by_slug(slug)
            .map(|l| serde_json::to_string(l).unwrap_or_default())
    }

    pub fn set_loops(&self, json: &str) {
        if let Ok(v) = serde_json::from_str::<Vec<LoopData>>(json) {
            self.state.write().unwrap().set_loops(v);
        }
    }

    pub fn set_current_loop(&self, json: &str) {
        let l = if json.is_empty() { None } else { serde_json::from_str::<LoopData>(json).ok() };
        self.state.write().unwrap().set_current_loop(l);
    }

    pub fn update_loop_local(&self, slug: &str, json: &str) {
        if let Ok(l) = serde_json::from_str::<LoopData>(json) {
            self.state.write().unwrap().update_loop(slug, l);
        }
    }

    pub fn add_run(&self, json: &str) {
        if let Ok(r) = serde_json::from_str::<LoopRunData>(json) {
            self.state.write().unwrap().add_run(r);
        }
    }

    pub fn set_runs(&self, json: &str) {
        if let Ok(v) = serde_json::from_str::<Vec<LoopRunData>>(json) {
            self.state.write().unwrap().set_runs(v);
        }
    }

    pub fn append_runs(&self, json: &str) {
        if let Ok(v) = serde_json::from_str::<Vec<LoopRunData>>(json) {
            self.state.write().unwrap().append_runs(v);
        }
    }

    pub fn update_run_status(&self, run_id: i64, status: &str) {
        let parsed = parse_status::<LoopRunStatus>(status);
        self.state.write().unwrap().update_run_status(run_id, parsed);
    }

    pub fn clear_runs(&self) {
        self.state.write().unwrap().clear_runs();
    }

    pub async fn fetch_loops(
        &self, status: Option<String>, limit: Option<u32>, offset: Option<u32>,
    ) -> Result<String, String> {
        let resp = self.client
            .list_loops(status.as_deref(), limit, offset)
            .await.map_err(crate::wire)?;
        self.state.write().unwrap().set_loops(resp.loops.clone());
        serde_json::to_string(&resp).map_err(crate::wire)
    }

    pub async fn fetch_loop(&self, slug: &str) -> Result<String, String> {
        let data: LoopData = self.client
            .get_loop(slug)
            .await.map_err(crate::wire)?;
        self.state.write().unwrap().set_current_loop(Some(data.clone()));
        serde_json::to_string(&data).map_err(crate::wire)
    }

    pub async fn create_loop(&self, request_json: &str) -> Result<String, String> {
        let req: CreateLoopRequest = serde_json::from_str(request_json)
            .map_err(crate::wire)?;
        let data: LoopData = self.client
            .create_loop(&req)
            .await.map_err(crate::wire)?;
        serde_json::to_string(&data).map_err(crate::wire)
    }

    pub async fn update_loop(&self, slug: &str, request_json: &str) -> Result<String, String> {
        let req: UpdateLoopRequest = serde_json::from_str(request_json)
            .map_err(crate::wire)?;
        let data: LoopData = self.client
            .update_loop(slug, &req)
            .await.map_err(crate::wire)?;
        self.state.write().unwrap().set_current_loop(Some(data.clone()));
        serde_json::to_string(&data).map_err(crate::wire)
    }

    pub async fn delete_loop(&self, slug: &str) -> Result<(), String> {
        self.client.delete_loop(slug).await.map_err(crate::wire)?;
        self.state.write().unwrap().set_current_loop(None);
        Ok(())
    }

    pub async fn enable_loop(&self, slug: &str) -> Result<String, String> {
        self.client.enable_loop(slug).await.map_err(crate::wire)?;
        let data = self.client.get_loop(slug).await.map_err(crate::wire)?;
        self.state.write().unwrap().update_loop(slug, data.clone());
        serde_json::to_string(&data).map_err(crate::wire)
    }

    pub async fn disable_loop(&self, slug: &str) -> Result<String, String> {
        self.client.disable_loop(slug).await.map_err(crate::wire)?;
        let data = self.client.get_loop(slug).await.map_err(crate::wire)?;
        self.state.write().unwrap().update_loop(slug, data.clone());
        serde_json::to_string(&data).map_err(crate::wire)
    }

    pub async fn trigger_loop(&self, slug: &str) -> Result<String, String> {
        let run: LoopRunData = self.client
            .trigger_loop(slug)
            .await.map_err(crate::wire)?;
        self.state.write().unwrap().add_run(run.clone());
        serde_json::to_string(&run).map_err(crate::wire)
    }

    pub async fn fetch_runs(
        &self, slug: &str, status: Option<String>,
        limit: Option<u32>, offset: Option<u32>,
    ) -> Result<String, String> {
        let resp = self.client
            .list_loop_runs(slug, status.as_deref(), limit, offset)
            .await.map_err(crate::wire)?;
        if offset.unwrap_or(0) > 0 {
            self.state.write().unwrap().append_runs(resp.runs.clone());
        } else {
            self.state.write().unwrap().set_runs(resp.runs.clone());
        }
        serde_json::to_string(&resp).map_err(crate::wire)
    }

    pub async fn cancel_run(&self, slug: &str, run_id: i64) -> Result<(), String> {
        self.client.cancel_loop_run(slug, run_id).await.map_err(crate::wire)?;
        self.state.write().unwrap().update_run_status(run_id, LoopRunStatus::Cancelled);
        Ok(())
    }

    // -------- Connect-RPC (binary wire) --------
    //
    // 10 Connect lanes — request bytes in, response bytes out. State is
    // bypassed (caller is the TS adapter); the existing REST methods
    // above keep updating the LoopState during the dual-track migration
    // so realtime event handlers stay correct.

    pub async fn list_loops_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        use agentsmesh_types::proto_loop_v1 as lp;
        use prost::Message;
        let req = lp::ListLoopsRequest::decode(request_bytes)
            .map_err(|e| format!("decode list_loops request: {e}"))?;
        let resp = self.client.list_loops_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn get_loop_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        use agentsmesh_types::proto_loop_v1 as lp;
        use prost::Message;
        let req = lp::GetLoopRequest::decode(request_bytes)
            .map_err(|e| format!("decode get_loop request: {e}"))?;
        let resp = self.client.get_loop_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn create_loop_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        use agentsmesh_types::proto_loop_v1 as lp;
        use prost::Message;
        let req = lp::CreateLoopRequest::decode(request_bytes)
            .map_err(|e| format!("decode create_loop request: {e}"))?;
        let resp = self.client.create_loop_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn update_loop_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        use agentsmesh_types::proto_loop_v1 as lp;
        use prost::Message;
        let req = lp::UpdateLoopRequest::decode(request_bytes)
            .map_err(|e| format!("decode update_loop request: {e}"))?;
        let resp = self.client.update_loop_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn delete_loop_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        use agentsmesh_types::proto_loop_v1 as lp;
        use prost::Message;
        let req = lp::DeleteLoopRequest::decode(request_bytes)
            .map_err(|e| format!("decode delete_loop request: {e}"))?;
        let resp = self.client.delete_loop_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn loop_action_connect(
        &self, action: &str, request_bytes: &[u8],
    ) -> Result<Vec<u8>, String> {
        use agentsmesh_types::proto_loop_v1 as lp;
        use prost::Message;
        let req = lp::LoopActionRequest::decode(request_bytes)
            .map_err(|e| format!("decode loop_action request: {e}"))?;
        let resp = match action {
            "enable" => self.client.enable_loop_connect(&req).await,
            "disable" => self.client.disable_loop_connect(&req).await,
            other => return Err(format!("unknown loop action: {other}")),
        }.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn trigger_loop_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        use agentsmesh_types::proto_loop_v1 as lp;
        use prost::Message;
        let req = lp::TriggerLoopRequest::decode(request_bytes)
            .map_err(|e| format!("decode trigger_loop request: {e}"))?;
        let resp = self.client.trigger_loop_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn list_runs_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        use agentsmesh_types::proto_loop_v1 as lp;
        use prost::Message;
        let req = lp::ListRunsRequest::decode(request_bytes)
            .map_err(|e| format!("decode list_runs request: {e}"))?;
        let resp = self.client.list_loop_runs_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn cancel_run_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        use agentsmesh_types::proto_loop_v1 as lp;
        use prost::Message;
        let req = lp::CancelRunRequest::decode(request_bytes)
            .map_err(|e| format!("decode cancel_run request: {e}"))?;
        let resp = self.client.cancel_loop_run_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }
}
