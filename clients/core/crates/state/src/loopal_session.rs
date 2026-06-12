use std::collections::HashMap;

use crate::loopal_types::{
    AgentNode, BgTask, CronJob, GoalInfo, LoopalSession, McpServer, TaskItem, MAX_BG_OUTPUT,
    MAX_BG_TASKS, MAX_LIST, MAX_TOPOLOGY,
};

fn cap_output(s: &mut String) {
    if s.len() <= MAX_BG_OUTPUT {
        return;
    }
    let mut cut = s.len() - MAX_BG_OUTPUT;
    while cut < s.len() && !s.is_char_boundary(cut) {
        cut += 1;
    }
    s.drain(..cut);
}

// Keep the newest `max` entries. FIFO drop-oldest is intentional: bg_tasks /
// topology rarely approach the bound, so a simple eviction beats tracking
// terminal-vs-running. This is the single capacity chokepoint for all writes
// (live mutators + snapshot rebuild), so core stays bounded even if the runner
// snapshot cache ever regresses its own cap.
fn cap_vec<T>(v: &mut Vec<T>, max: usize) {
    if v.len() > max {
        v.drain(..v.len() - max);
    }
}

// Find-or-insert a bg task row. Output/completed events can legitimately arrive
// before their spawn (relay reordering) or after the spawn was evicted by the
// MAX_BG_TASKS cap — without an upsert those events would be silently dropped,
// leaving a task stuck "Running" or never shown. The placeholder carries only
// the id; spawned fills identity later and never resets status/output.
fn bg_task_mut<'a>(session: &'a mut LoopalSession, id: &str) -> &'a mut BgTask {
    if let Some(idx) = session.bg_tasks.iter().position(|t| t.id == id) {
        return &mut session.bg_tasks[idx];
    }
    session.bg_tasks.push(BgTask {
        id: id.to_string(),
        description: String::new(),
        status: "Running".to_string(),
        exit_code: None,
        output: String::new(),
        created_at_unix_ms: 0,
    });
    cap_vec(&mut session.bg_tasks, MAX_BG_TASKS);
    session.bg_tasks.last_mut().expect("just pushed")
}

#[derive(Default)]
pub struct LoopalSessionManager {
    sessions: HashMap<String, LoopalSession>,
}

impl LoopalSessionManager {
    pub fn new() -> Self {
        Self::default()
    }

    pub fn get(&self, pod_key: &str) -> Option<&LoopalSession> {
        self.sessions.get(pod_key)
    }

    fn entry(&mut self, pod_key: &str) -> &mut LoopalSession {
        self.sessions.entry(pod_key.to_string()).or_default()
    }

    pub fn bg_task_spawned(
        &mut self,
        pod_key: &str,
        id: &str,
        description: &str,
        created_at_unix_ms: u64,
    ) {
        let task = bg_task_mut(self.entry(pod_key), id);
        if task.description.is_empty() {
            task.description = description.to_string();
        }
        if task.created_at_unix_ms == 0 {
            task.created_at_unix_ms = created_at_unix_ms;
        }
    }

    pub fn bg_task_output(&mut self, pod_key: &str, id: &str, delta: &str) {
        let task = bg_task_mut(self.entry(pod_key), id);
        task.output.push_str(delta);
        cap_output(&mut task.output);
    }

    pub fn bg_task_completed(
        &mut self,
        pod_key: &str,
        id: &str,
        status: &str,
        exit_code: Option<i32>,
        output: &str,
    ) {
        let task = bg_task_mut(self.entry(pod_key), id);
        if !status.is_empty() {
            task.status = status.to_string();
        }
        if exit_code.is_some() {
            task.exit_code = exit_code;
        }
        if !output.is_empty() {
            task.output = output.to_string();
            cap_output(&mut task.output);
        }
    }

    pub fn set_bg_tasks(&mut self, pod_key: &str, mut tasks: Vec<BgTask>) {
        for t in &mut tasks {
            cap_output(&mut t.output);
        }
        cap_vec(&mut tasks, MAX_BG_TASKS);
        self.entry(pod_key).bg_tasks = tasks;
    }

    pub fn set_crons(&mut self, pod_key: &str, mut crons: Vec<CronJob>) {
        cap_vec(&mut crons, MAX_LIST);
        self.entry(pod_key).crons = crons;
    }

    pub fn set_tasks(&mut self, pod_key: &str, mut tasks: Vec<TaskItem>) {
        cap_vec(&mut tasks, MAX_LIST);
        self.entry(pod_key).tasks = tasks;
    }

    pub fn set_mcp(&mut self, pod_key: &str, mut mcp: Vec<McpServer>) {
        cap_vec(&mut mcp, MAX_LIST);
        self.entry(pod_key).mcp = mcp;
    }

    pub fn set_topology(&mut self, pod_key: &str, mut topology: Vec<AgentNode>) {
        topology.retain(|n| !n.agent_id.is_empty());
        cap_vec(&mut topology, MAX_TOPOLOGY);
        self.entry(pod_key).topology = topology;
    }

    pub fn set_goal(&mut self, pod_key: &str, goal: Option<GoalInfo>) {
        self.entry(pod_key).thread_goal = goal;
    }

    pub fn set_mode(&mut self, pod_key: &str, mode: Option<String>) {
        self.entry(pod_key).mode = mode;
    }

    pub fn set_thinking(&mut self, pod_key: &str, thinking: Option<String>) {
        self.entry(pod_key).thinking = thinking;
    }

    pub fn set_model(&mut self, pod_key: &str, model: Option<String>) {
        self.entry(pod_key).model = model;
    }

    pub fn add_agent(&mut self, pod_key: &str, node: AgentNode) {
        // Drop empty-agent_id spawns (matches the Go accumulator): agent_id is the
        // dedup key and the React Flow node id, so a node without one is degenerate
        // and would collide with others on the empty id.
        if node.agent_id.is_empty() {
            return;
        }
        let session = self.entry(pod_key);
        if session.topology.iter().any(|n| n.agent_id == node.agent_id) {
            return;
        }
        session.topology.push(node);
        cap_vec(&mut session.topology, MAX_TOPOLOGY);
    }

    pub fn clear(&mut self, pod_key: &str) {
        self.sessions.remove(pod_key);
    }
}
