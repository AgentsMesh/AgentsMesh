use std::sync::Arc;

use agentsmesh_events::subscription_manager::EventSubscriptionManager;
use agentsmesh_events::types::{EventDispatchHook, RealtimeEvent};
use agentsmesh_persistence::StorageBackend;
use parking_lot::RwLock;

use crate::acp_session::AcpSessionManager;
use crate::autopilot_state::AutopilotState;
use crate::channel_state::ChannelState;
use crate::event_dispatch;
use crate::loop_state::LoopState;
use crate::loopal_session::LoopalSessionManager;
use crate::mesh_state::MeshState;
use crate::pod_state::PodState;
use crate::repo_state::RepoState;
use crate::runner_state::RunnerState;
use crate::ticket_state::TicketState;

/// Specification of a transient toast notification that the Rust-SSOT
/// dispatch wants the platform layer to display. Rust never ships locale
/// data; `title_key` + `title_params` are passed through and translated
/// on the JS/Swift side via the platform's i18n facility.
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct ToastSpec {
    pub kind: String,
    pub title_key: String,
    #[serde(default)]
    pub title_params: serde_json::Value,
    #[serde(default)]
    pub description: String,
    #[serde(default)]
    pub duration_ms: u32,
}

/// Specification of a browser/OS-native notification. The platform layer
/// decides whether to show via `Notification` API (web), `UNNotification`
/// (iOS), or in-app banner (electron-no-permission case).
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct NotificationSpec {
    pub title: String,
    pub body: String,
    #[serde(default)]
    pub icon: Option<String>,
    #[serde(default)]
    pub link: Option<String>,
}

pub struct AppState {
    pub pods: PodState,
    pub channels: ChannelState,
    pub runners: RunnerState,
    pub tickets: TicketState,
    pub loops: LoopState,
    pub mesh: MeshState,
    pub autopilot: AutopilotState,
    pub acp: AcpSessionManager,
    pub loopal: LoopalSessionManager,
    pub repo: RepoState,

    /// Toast notifications queued by dispatch (loop_run:warning,
    /// system:maintenance, etc). Drained per-tick by platform consumers.
    pub pending_toasts: Vec<ToastSpec>,
    /// Browser/OS-level notifications queued by `notification` events.
    pub pending_browser_notifications: Vec<NotificationSpec>,
    /// Ticket slugs whose details must be refetched (MR/Pipeline events
    /// only carry slug+podId — the actual MR/pipeline data must be
    /// pulled via Connect-RPC). Platforms drain per-tick.
    pub pending_refetch_ticket_slugs: Vec<String>,
    /// Pod keys whose details must be refetched (same rationale as above
    /// for MR/Pipeline events that carry only podId).
    pub pending_refetch_pod_keys: Vec<String>,
    /// Set to true when the realtime connection transitions back to
    /// `Connected` after a disconnect; platforms drain per-tick and run
    /// a global refetch (pods + tickets + channels) to catch up on
    /// events missed during the gap.
    pub pending_post_reconnect_refetch: bool,
    /// Autopilot controllers list needs refetch (created event carries
    /// partial data; full list pull picks up missing fields).
    pub pending_refetch_autopilot: bool,
}

impl AppState {
    pub fn new() -> Self {
        Self {
            pods: PodState::new(),
            channels: ChannelState::new(),
            runners: RunnerState::new(),
            tickets: TicketState::new(),
            loops: LoopState::new(),
            mesh: MeshState::default(),
            autopilot: AutopilotState::default(),
            acp: AcpSessionManager::new(),
            loopal: LoopalSessionManager::new(),
            repo: RepoState::new(),
            pending_toasts: Vec::new(),
            pending_browser_notifications: Vec::new(),
            pending_refetch_ticket_slugs: Vec::new(),
            pending_refetch_pod_keys: Vec::new(),
            pending_post_reconnect_refetch: false,
            pending_refetch_autopilot: false,
        }
    }

    pub fn with_storage(backend: Arc<dyn StorageBackend>) -> Self {
        Self {
            pods: PodState::with_storage(backend.clone()),
            channels: ChannelState::with_storage(backend.clone()),
            runners: RunnerState::with_storage(backend.clone()),
            tickets: TicketState::with_storage(backend.clone()),
            loops: LoopState::with_storage(backend.clone()),
            mesh: MeshState::default(),
            autopilot: AutopilotState::default(),
            acp: AcpSessionManager::new(),
            loopal: LoopalSessionManager::new(),
            repo: RepoState::with_storage(backend),
            pending_toasts: Vec::new(),
            pending_browser_notifications: Vec::new(),
            pending_refetch_ticket_slugs: Vec::new(),
            pending_refetch_pod_keys: Vec::new(),
            pending_post_reconnect_refetch: false,
            pending_refetch_autopilot: false,
        }
    }

    pub fn dispatch(&mut self, event: &RealtimeEvent) {
        event_dispatch::dispatch(self, event);
    }

    /// Atomic take-and-clear for pending toasts. Platform consumer drains
    /// this and emits via sonner / UNNotificationCenter etc. Items are
    /// not re-enqueued on consumer failure — log and drop.
    pub fn take_pending_toasts(&mut self) -> Vec<ToastSpec> {
        std::mem::take(&mut self.pending_toasts)
    }

    pub fn take_pending_browser_notifications(&mut self) -> Vec<NotificationSpec> {
        std::mem::take(&mut self.pending_browser_notifications)
    }

    pub fn take_pending_refetch_ticket_slugs(&mut self) -> Vec<String> {
        std::mem::take(&mut self.pending_refetch_ticket_slugs)
    }

    pub fn take_pending_refetch_pod_keys(&mut self) -> Vec<String> {
        std::mem::take(&mut self.pending_refetch_pod_keys)
    }

    pub fn take_pending_post_reconnect_refetch(&mut self) -> bool {
        std::mem::replace(&mut self.pending_post_reconnect_refetch, false)
    }

    pub fn take_pending_refetch_autopilot(&mut self) -> bool {
        std::mem::replace(&mut self.pending_refetch_autopilot, false)
    }

    /// Clear all org-scoped state on org switch. Keeps user-scoped
    /// settings (acp sessions, repo cache that's per-user) intact.
    /// Preferable to rebuilding the whole AppState because it preserves
    /// the live EventSubscriptionManager connection and its callbacks.
    pub fn reset_for_org_switch(&mut self) {
        self.pods = PodState::new();
        self.channels = ChannelState::new();
        self.runners = RunnerState::new();
        self.tickets = TicketState::new();
        self.loops = LoopState::new();
        self.mesh = MeshState::default();
        self.autopilot = AutopilotState::default();
        self.pending_toasts.clear();
        self.pending_browser_notifications.clear();
        self.pending_refetch_ticket_slugs.clear();
        self.pending_refetch_pod_keys.clear();
        self.pending_post_reconnect_refetch = false;
        self.pending_refetch_autopilot = false;
    }
}

impl Default for AppState {
    fn default() -> Self {
        Self::new()
    }
}

/// Lock-wrapped AppState + dispatch hook adapter. Owns the only mutable
/// reference path into the in-memory state tree; binding facades
/// (wasm/napi/ffi) share `Arc<AppRuntime>` and pass it into services.
pub struct AppRuntime {
    pub state: Arc<RwLock<AppState>>,
    pub events: Arc<EventSubscriptionManager>,
}

impl AppRuntime {
    /// Construct the runtime + wire the dispatch hook into the manager.
    /// The hook holds an `Arc<RwLock<AppState>>` weak-free; failure mode
    /// is "hook is dropped when AppRuntime is dropped".
    pub fn new(events: Arc<EventSubscriptionManager>) -> Arc<Self> {
        Self::with_state(events, AppState::new())
    }

    pub fn with_state(events: Arc<EventSubscriptionManager>, state: AppState) -> Arc<Self> {
        let state = Arc::new(RwLock::new(state));
        let hook: Arc<dyn EventDispatchHook> = Arc::new(AppStateDispatchHook::new(Arc::clone(&state)));
        events.set_dispatch_hook(hook);
        Arc::new(Self { state, events })
    }

    /// Snapshot of the events-side tick counter.
    pub fn tick(&self) -> u64 {
        self.events.tick()
    }
}

/// Adapter that lets `EventSubscriptionManager` call `AppState.dispatch`
/// without inverting the crate dep direction (state already depends on
/// events; events cannot depend on state).
///
/// Public so tests + alternate binding facades can construct one
/// without going through `AppRuntime::new`.
pub struct AppStateDispatchHook {
    state: Arc<RwLock<AppState>>,
}

impl AppStateDispatchHook {
    pub fn new(state: Arc<RwLock<AppState>>) -> Self {
        Self { state }
    }
}

impl EventDispatchHook for AppStateDispatchHook {
    fn dispatch(&self, event: &RealtimeEvent) {
        self.state.write().dispatch(event);
    }
}
