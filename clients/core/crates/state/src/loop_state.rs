use std::sync::Arc;

use agentsmesh_persistence::{LoopRepo, LoopRow, LoopRunRow, StorageBackend};
use serde::{Deserialize, Serialize};

/// Client-side aggregated view of a scheduled Loop. proto.loop.v1.Loop drops
/// the legacy split between `schedule` / `is_enabled` and the unified
/// `cron_expression` / `status` fields; this view type preserves both shapes
/// so renderer state slots can project either presentation. Persisted to the
/// blockstore via serde derives.
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct LoopData {
    #[serde(default)]
    pub id: i64,
    pub slug: String,
    pub name: String,
    #[serde(default)]
    pub description: Option<String>,
    #[serde(default)]
    pub schedule: Option<String>,
    #[serde(default)]
    pub is_enabled: bool,
    #[serde(default)]
    pub status: Option<String>,
    #[serde(default)]
    pub agent_slug: Option<String>,
    #[serde(default)]
    pub permission_mode: Option<String>,
    #[serde(default)]
    pub prompt_template: Option<String>,
    #[serde(default)]
    pub config_overrides: Option<serde_json::Value>,
    #[serde(default)]
    pub prompt_variables: Option<serde_json::Value>,
    #[serde(default)]
    pub execution_mode: Option<String>,
    #[serde(default)]
    pub autopilot_config: Option<serde_json::Value>,
    #[serde(default)]
    pub sandbox_strategy: Option<String>,
    #[serde(default)]
    pub session_persistence: Option<bool>,
    #[serde(default)]
    pub concurrency_policy: Option<String>,
    #[serde(default)]
    pub max_concurrent_runs: Option<i32>,
    #[serde(default)]
    pub max_retained_runs: Option<i32>,
    #[serde(default)]
    pub timeout_minutes: Option<i32>,
    #[serde(default)]
    pub idle_timeout_sec: Option<i32>,
    #[serde(default)]
    pub total_runs: Option<i64>,
    #[serde(default)]
    pub successful_runs: Option<i64>,
    #[serde(default)]
    pub failed_runs: Option<i64>,
    #[serde(default)]
    pub active_run_count: Option<i64>,
    #[serde(default)]
    pub last_run_at: Option<String>,
    #[serde(default)]
    pub created_at: Option<String>,
    #[serde(default)]
    pub updated_at: Option<String>,
    // proto.loop.v1.Loop fields the loopToCache projection reads + the detail
    // ConfigPanel renders (cron/webhook/repo/runner/ticket/branch/credential/
    // avg-duration). Without these slots the fetch→state round-trip silently
    // dropped them — cron loops showed "On Demand", webhook URL never rendered.
    #[serde(default)]
    pub cron_expression: Option<String>,
    #[serde(default)]
    pub callback_url: Option<String>,
    #[serde(default)]
    pub repository_id: Option<i64>,
    #[serde(default)]
    pub runner_id: Option<i64>,
    #[serde(default)]
    pub branch_name: Option<String>,
    #[serde(default)]
    pub ticket_id: Option<i64>,
    #[serde(default)]
    pub credential_profile_id: Option<i64>,
    #[serde(default)]
    pub avg_duration_sec: Option<f64>,
    /// Ordered list of EnvBundle names attached to this Loop. Mirrors the
    /// `proto.loop.v1.Loop.used_env_bundles` field; preserved across the
    /// serde round-trip so the edit dialog can reconcile saved bundles back
    /// into the credential select + runtime checkbox split.
    #[serde(default)]
    pub used_env_bundles: Vec<String>,
}

/// Client-side aggregated view of a Loop run. proto.loop.v1.LoopRun only
/// carries `loop_id`; this view type additionally tracks `loop_slug` (looked
/// up at write-time from the parent loop) so callers can filter by slug
/// without a back-join against LoopState.loops.
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct LoopRunData {
    pub id: i64,
    #[serde(default)]
    pub loop_slug: String,
    #[serde(default)]
    pub run_number: Option<i64>,
    #[serde(default)]
    pub status: String,
    #[serde(default)]
    pub pod_key: Option<String>,
    #[serde(default)]
    pub started_at: Option<String>,
    #[serde(default)]
    pub completed_at: Option<String>,
    #[serde(default)]
    pub error_message: Option<String>,
    #[serde(default)]
    pub created_at: Option<String>,
}

pub mod loop_run_status {
    pub const PENDING: &str = "pending";
    pub const RUNNING: &str = "running";
    pub const COMPLETED: &str = "completed";
    pub const FAILED: &str = "failed";
    pub const CANCELLED: &str = "cancelled";
}

impl LoopRow for LoopData {
    fn slug(&self) -> &str { &self.slug }
}

impl LoopRunRow for LoopRunData {
    fn id(&self) -> i64 { self.id }
    fn loop_slug(&self) -> &str { &self.loop_slug }
}

pub struct LoopState {
    loops: Vec<LoopData>,
    current_loop: Option<LoopData>,
    runs: Vec<LoopRunData>,
    repo: Option<LoopRepo<LoopData, LoopRunData>>,
}

impl LoopState {
    pub fn new() -> Self {
        Self { loops: Vec::new(), current_loop: None, runs: Vec::new(), repo: None }
    }

    pub fn with_storage(backend: Arc<dyn StorageBackend>) -> Self {
        let repo = LoopRepo::new(backend);
        let loops = repo.list_loops().unwrap_or_default();
        Self { loops, current_loop: None, runs: Vec::new(), repo: Some(repo) }
    }

    pub fn get_loops(&self) -> &[LoopData] { &self.loops }
    pub fn get_current_loop(&self) -> Option<&LoopData> { self.current_loop.as_ref() }
    pub fn get_runs(&self) -> &[LoopRunData] { &self.runs }

    pub fn get_loop_by_slug(&self, slug: &str) -> Option<&LoopData> {
        self.loops.iter().find(|l| l.slug == slug)
    }

    pub fn set_loops(&mut self, loops: Vec<LoopData>) {
        tracing::debug!(target: "loop", count = loops.len(), "set loops (baseline)");
        self.loops = loops;
        if let Some(repo) = &self.repo {
            for l in &self.loops { let _ = repo.save_loop(l); }
        }
    }

    pub fn set_current_loop(&mut self, loop_data: Option<LoopData>) {
        self.current_loop = loop_data;
    }

    pub fn update_loop(&mut self, slug: &str, loop_data: LoopData) {
        tracing::info!(target: "loop", slug, status = ?loop_data.status, "update loop");
        if let Some(l) = self.loops.iter_mut().find(|l| l.slug == slug) {
            *l = loop_data.clone();
            if let Some(repo) = &self.repo { let _ = repo.save_loop(l); }
        }
        if self.current_loop.as_ref().is_some_and(|l| l.slug == slug) {
            self.current_loop = Some(loop_data);
        }
    }

    pub fn add_run(&mut self, run: LoopRunData) {
        tracing::info!(target: "loop", run_id = run.id, loop_slug = %run.loop_slug, status = %run.status, "add run");
        if let Some(repo) = &self.repo { let _ = repo.save_run(&run); }
        self.runs.push(run);
    }

    pub fn set_runs(&mut self, runs: Vec<LoopRunData>) {
        tracing::debug!(target: "loop", count = runs.len(), "set runs (baseline)");
        self.runs = runs;
    }

    pub fn append_runs(&mut self, runs: Vec<LoopRunData>) {
        tracing::debug!(target: "loop", count = runs.len(), "append runs");
        self.runs.extend(runs);
    }

    pub fn update_run_status(&mut self, run_id: i64, status: &str) {
        tracing::info!(target: "loop", run_id, status, "run status changed");
        if let Some(run) = self.runs.iter_mut().find(|r| r.id == run_id) {
            run.status = status.to_string();
            if let Some(repo) = &self.repo { let _ = repo.save_run(run); }
        }
    }

    pub fn clear_runs(&mut self) {
        tracing::debug!(target: "loop", "clear runs");
        self.runs.clear();
    }
}

impl Default for LoopState {
    fn default() -> Self { Self::new() }
}
