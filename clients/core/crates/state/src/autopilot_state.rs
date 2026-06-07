use std::collections::HashMap;

use serde::{Deserialize, Serialize};
use serde_json::Value;

const MAX_ITERATIONS: usize = 200;
const MAX_THINKING_HISTORY: usize = 100;

const ACTIVE_PHASES: &[&str] = &["initializing", "running", "paused", "user_takeover", "waiting_approval"];

/// Client-side aggregated view of an Autopilot controller. Lives in the state
/// crate because the cache shape carries fields the proto wire never had
/// (status, iteration_timeout_sec, no_progress_threshold, same_error_threshold,
/// approval_timeout_min, control_agent_slug, circuit_breaker_state/reason)
/// alongside the proto-mapped core. Renderer state slots project to/from JSON
/// through this struct's serde derives.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AutopilotController {
    #[serde(alias = "key")]
    pub autopilot_controller_key: String,
    pub pod_key: String,
    #[serde(default)]
    pub status: Option<String>,
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
pub struct AutopilotIteration {
    pub id: i64,
    pub controller_key: String,
    pub iteration_number: Option<i64>,
    pub status: Option<String>,
    pub result: Option<String>,
    pub started_at: Option<String>,
    pub completed_at: Option<String>,
}

#[derive(Debug, Default)]
pub struct AutopilotState {
    controllers: Vec<AutopilotController>,
    current_controller: Option<AutopilotController>,
    iterations: HashMap<String, Vec<AutopilotIteration>>,
    thinking: HashMap<String, Option<Value>>,
    thinking_history: HashMap<String, Vec<Value>>,
}

impl AutopilotState {
    pub fn new() -> Self {
        Self::default()
    }

    pub fn controllers(&self) -> &[AutopilotController] {
        &self.controllers
    }

    pub fn current_controller(&self) -> Option<&AutopilotController> {
        self.current_controller.as_ref()
    }

    pub fn set_controllers(&mut self, controllers: Vec<AutopilotController>) {
        tracing::debug!(target: "autopilot", count = controllers.len(), "set controllers (baseline)");
        self.controllers = controllers;
    }

    pub fn set_current_controller(&mut self, controller: Option<AutopilotController>) {
        self.current_controller = controller;
    }

    pub fn add_controller(&mut self, controller: AutopilotController) {
        tracing::info!(
            target: "autopilot",
            controller_key = %controller.autopilot_controller_key,
            pod_key = %controller.pod_key,
            status = ?controller.status,
            "add controller"
        );
        self.controllers.push(controller);
    }

    pub fn update_controller(&mut self, key: &str, controller: AutopilotController) {
        tracing::info!(
            target: "autopilot",
            controller_key = key,
            status = ?controller.status,
            circuit_breaker_state = ?controller.circuit_breaker_state,
            "update controller"
        );
        for c in &mut self.controllers {
            if c.autopilot_controller_key == key {
                *c = controller.clone();
            }
        }
        if let Some(ref mut cur) = self.current_controller {
            if cur.autopilot_controller_key == key {
                *cur = controller;
            }
        }
    }

    pub fn remove_controller(&mut self, key: &str) {
        tracing::info!(target: "autopilot", controller_key = key, "remove controller");
        self.controllers.retain(|c| c.autopilot_controller_key != key);
        if self.current_controller
            .as_ref()
            .is_some_and(|c| c.autopilot_controller_key == key)
        {
            self.current_controller = None;
        }
    }

    pub fn get_iterations(&self, key: &str) -> Option<&[AutopilotIteration]> {
        self.iterations.get(key).map(|v| v.as_slice())
    }

    pub fn set_iterations(&mut self, key: String, iters: Vec<AutopilotIteration>) {
        tracing::debug!(target: "autopilot", controller_key = %key, count = iters.len(), "set iterations (baseline)");
        self.iterations.insert(key, iters);
    }

    pub fn add_iteration(&mut self, key: String, iteration: AutopilotIteration) {
        tracing::info!(
            target: "autopilot",
            controller_key = %key,
            iteration_id = iteration.id,
            status = ?iteration.status,
            "add iteration"
        );
        let iters = self.iterations.entry(key).or_default();
        iters.push(iteration);
        if iters.len() > MAX_ITERATIONS {
            let drain = iters.len() - MAX_ITERATIONS;
            iters.drain(..drain);
        }
    }

    pub fn get_thinking(&self, key: &str) -> Option<&Value> {
        self.thinking.get(key).and_then(|v| v.as_ref())
    }

    pub fn get_thinking_history(&self, key: &str) -> Option<&[Value]> {
        self.thinking_history.get(key).map(|v| v.as_slice())
    }

    pub fn update_thinking(&mut self, key: String, thinking: Value) {
        tracing::debug!(target: "autopilot", controller_key = %key, "update thinking");
        self.thinking.insert(key.clone(), Some(thinking.clone()));
        let history = self.thinking_history.entry(key).or_default();
        history.push(thinking);
        if history.len() > MAX_THINKING_HISTORY {
            let drain = history.len() - MAX_THINKING_HISTORY;
            history.drain(..drain);
        }
    }

    pub fn get_controller_by_pod_key(&self, pod_key: &str) -> Option<&AutopilotController> {
        self.controllers.iter().find(|c| {
            c.pod_key == pod_key
                && c.phase.as_deref().is_some_and(|p| ACTIVE_PHASES.contains(&p))
        })
    }
}

