use std::collections::HashMap;
use std::sync::Arc;

use agentsmesh_persistence::{PodRepo, StorageBackend};
use agentsmesh_types::proto_pod_v1::Pod;

// PodStatus values are wire-strings from proto (proto3 stores enum status
// as `string` field). Replace previous `agentsmesh_types::PodStatus` enum
// comparisons with string comparisons against these constants.
pub mod pod_status {
    pub const PENDING: &str = "pending";
    pub const CREATING: &str = "creating";
    pub const INITIALIZING: &str = "initializing";
    pub const RUNNING: &str = "running";
    pub const PAUSED: &str = "paused";
    pub const STOPPED: &str = "stopped";
    pub const FAILED: &str = "failed";
    pub const TERMINATED: &str = "terminated";
    pub const DISCONNECTED: &str = "disconnected";
    pub const ORPHANED: &str = "orphaned";
    pub const COMPLETED: &str = "completed";
    pub const ERROR: &str = "error";
    pub const UNKNOWN: &str = "unknown";
}

#[derive(Debug, Clone)]
pub struct PodInitProgress {
    pub phase: String,
    pub progress: f64,
    pub message: Option<String>,
}

pub struct PodState {
    pods: Vec<Pod>,
    current_pod: Option<Pod>,
    init_progress: HashMap<String, PodInitProgress>,
    pod_timestamps: HashMap<String, i64>,
    repo: Option<PodRepo>,
}

fn save_pod(repo: &PodRepo, pod: &Pod) {
    let _ = repo.save_pod(pod);
}

impl PodState {
    pub fn new() -> Self {
        Self { pods: Vec::new(), current_pod: None, init_progress: HashMap::new(), pod_timestamps: HashMap::new(), repo: None }
    }

    pub fn with_storage(backend: Arc<dyn StorageBackend>) -> Self {
        let repo = PodRepo::new(backend);
        let pods = repo.list_pods().unwrap_or_default();
        Self { pods, current_pod: None, init_progress: HashMap::new(), pod_timestamps: HashMap::new(), repo: Some(repo) }
    }

    pub fn pods(&self) -> &[Pod] { &self.pods }
    pub fn current_pod(&self) -> Option<&Pod> { self.current_pod.as_ref() }

    pub fn should_update(&self, pod_key: &str, timestamp: i64) -> bool {
        match self.pod_timestamps.get(pod_key) {
            Some(&existing) => timestamp > existing,
            None => true,
        }
    }

    pub fn upsert_pod(&mut self, pod: Pod, timestamp: Option<i64>) {
        tracing::info!(target: "pod", pod_key = %pod.pod_key, status = %pod.status, "upsert pod");
        if let Some(ts) = timestamp {
            if !self.should_update(&pod.pod_key, ts) { return; }
            self.pod_timestamps.insert(pod.pod_key.clone(), ts);
        }
        if let Some(pos) = self.pods.iter().position(|p| p.pod_key == pod.pod_key) {
            self.pods[pos] = pod.clone();
        } else {
            self.pods.push(pod.clone());
        }
        if let Some(ref cur) = self.current_pod {
            if cur.pod_key == pod.pod_key { self.current_pod = Some(pod.clone()); }
        }
        if let Some(repo) = &self.repo { save_pod(repo, &pod); }
    }

    pub fn update_pod_status(
        &mut self, pod_key: &str, status: &str, agent_status: Option<&str>,
        error_code: Option<&str>, error_message: Option<&str>, timestamp: Option<i64>,
    ) {
        tracing::info!(target: "pod", pod_key, status, "status changed");
        if let Some(ts) = timestamp {
            if !self.should_update(pod_key, ts) { return; }
            self.pod_timestamps.insert(pod_key.to_string(), ts);
        }
        let status_owned = status.to_string();
        let agent_status_owned = agent_status.map(String::from);
        let error_code_owned = error_code.map(String::from);
        let error_message_owned = error_message.map(String::from);
        let updater = |pod: &mut Pod| {
            pod.status = status_owned.clone();
            if let Some(ref as_) = agent_status_owned { pod.agent_status = as_.clone(); }
            pod.error_code = error_code_owned.clone();
            pod.error_message = error_message_owned.clone();
        };
        if let Some(p) = self.pods.iter_mut().find(|p| p.pod_key == pod_key) {
            updater(p);
            if let Some(ref mut cur) = self.current_pod {
                if cur.pod_key == pod_key { updater(cur); }
            }
            if let Some(repo) = &self.repo { save_pod(repo, p); }
        }
    }

    pub fn update_pod_title(&mut self, pod_key: &str, title: &str, timestamp: Option<i64>) {
        tracing::debug!(target: "pod", pod_key, "title changed");
        if let Some(ts) = timestamp {
            if !self.should_update(pod_key, ts) { return; }
            self.pod_timestamps.insert(pod_key.to_string(), ts);
        }
        if let Some(p) = self.pods.iter_mut().find(|p| p.pod_key == pod_key) {
            p.title = Some(title.to_string());
            if let Some(ref mut cur) = self.current_pod {
                if cur.pod_key == pod_key { cur.title = Some(title.to_string()); }
            }
            if let Some(repo) = &self.repo { save_pod(repo, p); }
        }
    }

    pub fn update_agent_status(&mut self, pod_key: &str, agent_status: &str) {
        tracing::info!(target: "pod", pod_key, agent_status, "agent status changed");
        if let Some(p) = self.pods.iter_mut().find(|p| p.pod_key == pod_key) {
            p.agent_status = agent_status.to_string();
            if let Some(ref mut cur) = self.current_pod {
                if cur.pod_key == pod_key { cur.agent_status = agent_status.to_string(); }
            }
            if let Some(repo) = &self.repo { save_pod(repo, p); }
        }
    }

    pub fn update_pod_alias(&mut self, pod_key: &str, alias: &str) {
        tracing::info!(target: "pod", pod_key, "alias changed");
        if let Some(p) = self.pods.iter_mut().find(|p| p.pod_key == pod_key) {
            p.alias = Some(alias.to_string());
            if let Some(ref mut cur) = self.current_pod {
                if cur.pod_key == pod_key { cur.alias = Some(alias.to_string()); }
            }
            if let Some(repo) = &self.repo { save_pod(repo, p); }
        }
    }

    pub fn update_init_progress(&mut self, pod_key: &str, phase: &str, progress: f64, message: Option<&str>) {
        tracing::debug!(target: "pod", pod_key, phase, progress, "init progress");
        self.init_progress.insert(pod_key.to_string(), PodInitProgress {
            phase: phase.to_string(), progress, message: message.map(String::from),
        });
    }

    pub fn clear_init_progress(&mut self, pod_key: &str) { self.init_progress.remove(pod_key); }
    pub fn get_init_progress(&self, pod_key: &str) -> Option<&PodInitProgress> { self.init_progress.get(pod_key) }

    pub fn remove_pod(&mut self, pod_key: &str) {
        tracing::info!(target: "pod", pod_key, "remove pod");
        self.pods.retain(|p| p.pod_key != pod_key);
        self.pod_timestamps.remove(pod_key);
        self.init_progress.remove(pod_key);
        if let Some(ref cur) = self.current_pod {
            if cur.pod_key == pod_key { self.current_pod = None; }
        }
        if let Some(repo) = &self.repo { let _ = repo.delete_pod(pod_key); }
    }

    pub fn get_pod(&self, pod_key: &str) -> Option<&Pod> { self.pods.iter().find(|p| p.pod_key == pod_key) }
    pub fn set_current_pod(&mut self, pod: Option<Pod>) { self.current_pod = pod; }

    /// SRP patch — only the perpetual flag of an existing pod. No-op if
    /// pod not in cache (it'll arrive via a later upsert).
    pub fn patch_perpetual(&mut self, pod_key: &str, perpetual: bool) {
        tracing::info!(target: "pod", pod_key, perpetual, "patch perpetual");
        if let Some(pos) = self.pods.iter().position(|p| p.pod_key == pod_key) {
            self.pods[pos].perpetual = perpetual;
            if let Some(repo) = &self.repo {
                save_pod(repo, &self.pods[pos]);
            }
        }
    }

    pub fn set_pods(&mut self, pods: Vec<Pod>) {
        tracing::debug!(target: "pod", count = pods.len(), "set pods (baseline)");
        self.pods = pods;
        if let Some(repo) = &self.repo {
            let _ = repo.clear();
            for pod in &self.pods {
                save_pod(repo, pod);
            }
        }
    }
}

impl Default for PodState {
    fn default() -> Self { Self::new() }
}
