use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct LoopalSession {
    pub bg_tasks: Vec<BgTask>,
    pub crons: Vec<CronJob>,
    pub tasks: Vec<TaskItem>,
    pub topology: Vec<AgentNode>,
    pub mcp: Vec<McpServer>,
    pub thread_goal: Option<GoalInfo>,
    /// `"act"` / `"plan"`. None until the first ModeChanged event arrives.
    pub mode: Option<String>,
    /// Raw serialized ThinkingConfig JSON; the GUI normalizes it to a label.
    pub thinking: Option<String>,
    pub model: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BgTask {
    pub id: String,
    pub description: String,
    /// Loopal `BgTaskStatus` wire value (PascalCase: Running/Completed/Failed/Killed).
    pub status: String,
    pub exit_code: Option<i32>,
    pub output: String,
    pub created_at_unix_ms: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CronJob {
    pub id: String,
    #[serde(default)]
    pub cron_expr: String,
    #[serde(default)]
    pub prompt: String,
    #[serde(default)]
    pub recurring: bool,
    #[serde(default)]
    pub next_fire_unix_ms: Option<i64>,
    #[serde(default)]
    pub durable: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TaskItem {
    pub id: String,
    #[serde(default)]
    pub subject: String,
    #[serde(default)]
    pub status: String,
    #[serde(default)]
    pub blocked_by: Vec<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AgentNode {
    pub name: String,
    #[serde(default)]
    pub agent_id: String,
    #[serde(default)]
    pub parent: Option<String>,
    #[serde(default)]
    pub model: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct McpServer {
    pub name: String,
    #[serde(default)]
    pub status: String,
    #[serde(default)]
    pub tool_count: usize,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GoalInfo {
    #[serde(default)]
    pub goal_id: String,
    #[serde(default)]
    pub objective: String,
    #[serde(default)]
    pub status: String,
}

/// Capacity bounds mirroring acp_types MAX_*; guard a long-running pod against
/// unbounded growth of background-task output / spawned-agent count.
pub const MAX_BG_OUTPUT: usize = 64 * 1024;
pub const MAX_BG_TASKS: usize = 200;
pub const MAX_TOPOLOGY: usize = 200;
/// Defensive cap for full-replace lists (crons/tasks/mcp). These are normally
/// bounded by what Loopal sends, but a runaway producer must not be unbounded.
pub const MAX_LIST: usize = 500;
