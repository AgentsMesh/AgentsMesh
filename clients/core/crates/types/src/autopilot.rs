use serde::{Deserialize, Serialize};

use crate::AutopilotStatus;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AutopilotController {
    #[serde(alias = "key")]
    pub autopilot_controller_key: String,
    pub pod_key: String,
    #[serde(default)]
    pub status: Option<AutopilotStatus>,
    pub phase: Option<String>,
    pub prompt: Option<String>,
    pub max_iterations: Option<i64>,
    pub iteration_timeout_sec: Option<i64>,
    pub no_progress_threshold: Option<i64>,
    pub same_error_threshold: Option<i64>,
    pub approval_timeout_min: Option<i64>,
    pub current_iteration: Option<i64>,
    pub control_agent_slug: Option<String>,
    pub circuit_breaker_state: Option<String>,
    pub circuit_breaker_reason: Option<String>,
    pub created_at: Option<String>,
    pub updated_at: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateAutopilotRequest {
    pub pod_key: String,
    pub prompt: Option<String>,
    pub max_iterations: Option<i64>,
    pub iteration_timeout_sec: Option<i64>,
    pub no_progress_threshold: Option<i64>,
    pub same_error_threshold: Option<i64>,
    pub approval_timeout_min: Option<i64>,
    pub control_agent_slug: Option<String>,
    pub control_prompt_template: Option<String>,
    pub mcp_config_json: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ApproveAutopilotRequest {
    pub continue_execution: Option<bool>,
    pub additional_iterations: Option<i64>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AutopilotIteration {
    pub id: i64,
    pub controller_key: String,
    pub iteration_number: Option<i64>,
    pub status: Option<String>,
    pub result: Option<String>,
    pub started_at: Option<String>,
    pub completed_at: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AutopilotListResponse {
    pub controllers: Vec<AutopilotController>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AutopilotActionResponse {
    pub status: String,
    #[serde(default)]
    pub action: Option<String>,
}
