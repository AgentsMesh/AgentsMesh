use serde::{Deserialize, Serialize};
use std::collections::HashMap;

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum AcpState {
    Idle,
    Processing,
    WaitingPermission,
}

impl Default for AcpState {
    fn default() -> Self {
        Self::Idle
    }
}

impl AcpState {
    pub fn as_str(&self) -> &'static str {
        match self {
            Self::Idle => "idle",
            Self::Processing => "processing",
            Self::WaitingPermission => "waiting_permission",
        }
    }

    pub fn from_str_lossy(s: &str) -> Self {
        match s {
            "processing" => Self::Processing,
            "waiting_permission" => Self::WaitingPermission,
            _ => Self::Idle,
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AcpContentChunk {
    pub text: String,
    pub role: String,
    pub timestamp: i64,
    pub complete: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AcpToolCall {
    pub id: String,
    pub name: String,
    pub status: String,
    pub args: Option<serde_json::Value>,
    pub result_text: Option<String>,
    pub error_message: Option<String>,
    pub success: Option<bool>,
    // 客户端事件分发器透传 wire payload — 这些 payload 没有 timestamp
    // 字段（ToolCallUpdate / ToolCallSnapshot Go 结构未带）。AcpSessionManager
    // 在 update_tool_call 里会用 now_millis() 覆盖，所以这里只需要默认值。
    #[serde(default)]
    pub timestamp: i64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AcpPlanStep {
    pub title: String,
    pub status: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AcpThinking {
    pub text: String,
    pub timestamp: i64,
    pub complete: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AcpLog {
    pub level: String,
    pub message: String,
    pub timestamp: i64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AcpPermissionRequest {
    pub id: String,
    pub tool_name: String,
    pub args: Option<serde_json::Value>,
    pub description: Option<String>,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct AcpConfiguration {
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub permission_mode: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub model: String,
    // Static agent capability (advertised at initialize), not live state. Flows
    // via snapshot only; the configChanged delta path leaves it empty so
    // update_configuration's merge guard preserves it.
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub supported_permission_modes: Vec<String>,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct AcpSession {
    pub messages: Vec<AcpContentChunk>,
    pub tool_calls: HashMap<String, AcpToolCall>,
    pub plan: Vec<AcpPlanStep>,
    pub thinkings: Vec<AcpThinking>,
    pub logs: Vec<AcpLog>,
    pub state: AcpState,
    pub pending_permissions: Vec<AcpPermissionRequest>,
    #[serde(default)]
    pub configuration: AcpConfiguration,
}

pub const MAX_MESSAGES: usize = 500;
pub const MAX_TOOL_CALLS: usize = 500;
pub const MAX_THINKINGS: usize = 100;
pub const MAX_LOGS: usize = 50;
