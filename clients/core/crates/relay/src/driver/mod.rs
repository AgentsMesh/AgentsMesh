use std::collections::HashMap;
use std::sync::Arc;

use agentsmesh_protocol::encode_resize;
use agentsmesh_transport::runtime::Runtime;
use agentsmesh_transport::WsSender;
use futures::channel::mpsc;
use futures::{select, FutureExt, StreamExt};
use parking_lot::RwLock;
use web_time::Instant;

use crate::command::Command;
use crate::connection;
use crate::pool::PoolRouter;
use crate::retry;
use crate::types::{
    OutputCallback, RelayStatus, RelayStatusInfo, StatusCallback, StatusSnapshot,
};

mod session;

/// Idle ceiling for the session select when no timer (snapshot retry / resize
/// debounce / disconnect grace) is pending — keeps commands responsive without
/// busy-waiting. 2b will reuse this tick for data-silence detection.
const IDLE_TICK_MS: u64 = 30_000;

pub(super) enum SessionEnd {
    /// Transport dropped (close/error) or data-dead — reconnect with backoff.
    Closed,
    /// Pod is done (explicit disconnect, or last subscriber gone past grace).
    Shutdown,
}

enum Flow {
    Reconnect,
    Stop,
}

/// Single-owner actor for one pod's relay link. Owns all connection state; the
/// pool reaches it only via the `Command` channel (writes) and the shared
/// `StatusSnapshot` (reads) — no `Arc<RwLock>` over connection internals, so the
/// state machine runs lock-free in one task, and the task owning every timer
/// means exit cancels them all at once.
pub(crate) struct Driver<R: Runtime> {
    runtime: R,
    router: Arc<RwLock<PoolRouter>>,
    pod_key: String,
    relay_url: String,
    relay_token: String,
    snapshot: Arc<RwLock<StatusSnapshot>>,

    status: RelayStatus,
    snapshot_received: bool,
    reconnect_attempts: u32,
    runner_disconnected: bool,
    pod_size: Option<(u16, u16)>,
    subscribers: HashMap<String, OutputCallback>,
    last_input: Option<(String, Instant)>,
    pending_resize: Option<(u16, u16, Instant)>,
    grace_deadline: Option<Instant>,
}

impl<R: Runtime> Driver<R> {
    pub(crate) fn spawn(
        runtime: R,
        router: Arc<RwLock<PoolRouter>>,
        pod_key: String,
        relay_url: String,
        relay_token: String,
        snapshot: Arc<RwLock<StatusSnapshot>>,
        cmd_rx: mpsc::UnboundedReceiver<Command>,
        first_sub: (String, OutputCallback),
    ) {
        let mut subscribers = HashMap::new();
        subscribers.insert(first_sub.0, first_sub.1);
        let driver = Self {
            runtime: runtime.clone(),
            router,
            pod_key,
            relay_url,
            relay_token,
            snapshot,
            status: RelayStatus::Connecting,
            snapshot_received: false,
            reconnect_attempts: 0,
            runner_disconnected: false,
            pod_size: None,
            subscribers,
            last_input: None,
            pending_resize: None,
            grace_deadline: None,
        };
        runtime.spawn(Box::pin(driver.run(cmd_rx)));
    }

    async fn run(mut self, mut cmd_rx: mpsc::UnboundedReceiver<Command>) {
        loop {
            self.set_status(RelayStatus::Connecting);
            let connect = agentsmesh_transport::timeout(
                &self.runtime,
                std::time::Duration::from_millis(retry::CONNECT_TIMEOUT_MS),
                connection::connect(&self.relay_url, &self.relay_token),
            )
            .await;
            let stop = match connect {
                Ok(Ok((sender, mut receiver))) => {
                    tracing::info!(target: "relay", pod_key = %self.pod_key, "ws connected");
                    self.snapshot_received = false;
                    // Fresh connection: assume the runner is up until a
                    // RunnerDisconnected frame says otherwise, and don't let input
                    // dedup carry a stale baseline across the reconnect.
                    self.runner_disconnected = false;
                    self.last_input = None;
                    self.write_snapshot();
                    self.flush_pending_resize(&sender);
                    match self.run_session(&sender, &mut receiver, &mut cmd_rx).await {
                        SessionEnd::Shutdown => true,
                        SessionEnd::Closed => {
                            matches!(self.backoff(&mut cmd_rx).await, Flow::Stop)
                        }
                    }
                }
                failed => {
                    match failed {
                        Ok(Err(e)) => {
                            tracing::warn!("relay connect failed for {}: {e}", self.pod_key)
                        }
                        _ => tracing::warn!("relay connect timed out for {}", self.pod_key),
                    }
                    self.set_status(RelayStatus::Error);
                    matches!(self.backoff(&mut cmd_rx).await, Flow::Stop)
                }
            };
            // Finalize a stop decision atomically — a subscribe that raced in
            // since the decision is caught here and revives the driver instead of
            // orphaning onto a dying one.
            if stop && self.try_finalize(&mut cmd_rx) {
                return;
            }
        }
    }

    /// Wait out the (capped, escalating) reconnect delay while still draining
    /// commands — a subscriber arriving mid-backoff is kept; explicit disconnect
    /// or the last subscriber leaving stops the driver for good.
    async fn backoff(&mut self, cmd_rx: &mut mpsc::UnboundedReceiver<Command>) -> Flow {
        if self.subscribers.is_empty() {
            return Flow::Stop;
        }
        let delay =
            retry::compute_reconnect_delay(self.reconnect_attempts, retry::BASE_RECONNECT_DELAY_MS);
        self.reconnect_attempts += 1;
        let sleep = self.runtime.sleep(delay).fuse();
        futures::pin_mut!(sleep);
        loop {
            select! {
                _ = sleep => return Flow::Reconnect,
                cmd = cmd_rx.next().fuse() => match cmd {
                    None => return Flow::Stop,
                    Some(Command::Disconnect) => {
                        // Explicit disconnect during backoff also tears down.
                        self.subscribers.clear();
                        return Flow::Stop;
                    }
                    Some(Command::AddSubscriber { sub_id, cb }) => {
                        self.subscribers.insert(sub_id, cb);
                    }
                    Some(Command::RemoveSubscriber { sub_id }) => {
                        self.subscribers.remove(&sub_id);
                        if self.subscribers.is_empty() {
                            return Flow::Stop;
                        }
                    }
                    Some(Command::Resize { cols, rows, .. }) => {
                        // Keep the latest size so the reconnect flush re-applies it
                        // (force_resize is authoritative and must survive backoff).
                        if cols > 0 && rows > 0 {
                            self.pending_resize = Some((cols, rows, Instant::now()));
                        }
                    }
                    Some(_) => {} // Send/Acp dropped while offline
                }
            }
        }
    }

    fn set_status(&mut self, s: RelayStatus) {
        if self.status == s {
            return;
        }
        self.status = s;
        self.write_snapshot();
        self.notify_status();
    }

    fn write_snapshot(&self) {
        let mut g = self.snapshot.write();
        g.status = self.status;
        g.runner_disconnected = self.runner_disconnected;
        g.pod_size = self.pod_size;
    }

    fn notify_status(&self) {
        let info = self.status_info();
        let listeners: Vec<StatusCallback> = {
            let router = self.router.read();
            router
                .status_listeners
                .get(&self.pod_key)
                .cloned()
                .unwrap_or_default()
        };
        for l in &listeners {
            // An app-supplied listener panic must not kill the driver task (which
            // would skip teardown and wedge the pod). catch_unwind isolates it on
            // native; wasm is panic=abort so this is a no-op there.
            let info = info.clone();
            let _ = std::panic::catch_unwind(std::panic::AssertUnwindSafe(|| l(info)));
        }
    }

    fn status_info(&self) -> RelayStatusInfo {
        RelayStatusInfo {
            status: self.status,
            runner_disconnected: self.runner_disconnected,
        }
    }

    fn flush_pending_resize(&mut self, sender: &WsSender) {
        if let Some((c, r, _)) = self.pending_resize.take() {
            let _ = sender.send_binary(encode_resize(c, r));
        }
    }

    /// Finalize a stop decision atomically. Under the router write-lock — the
    /// same one `subscribe` takes to find/insert a PodHandle — drain commands
    /// queued since the stop was decided: a queued `AddSubscriber` revives the
    /// driver (return false → reconnect); otherwise remove the pod + its listeners
    /// and fire the pod-disconnected sink (return true → task exits). This closes
    /// the subscribe-vs-teardown race: a concurrent subscribe either lands its
    /// AddSubscriber here, or finds no pod and spawns a fresh driver — never
    /// orphans onto this dying one.
    fn try_finalize(&mut self, cmd_rx: &mut mpsc::UnboundedReceiver<Command>) -> bool {
        let cb = {
            let mut router = self.router.write();
            while let Ok(cmd) = cmd_rx.try_recv() {
                match cmd {
                    Command::AddSubscriber { sub_id, cb } => {
                        self.subscribers.insert(sub_id, cb);
                    }
                    Command::RemoveSubscriber { sub_id } => {
                        self.subscribers.remove(&sub_id);
                    }
                    // Disconnect overrides any queued subscriber — clear so we tear down.
                    Command::Disconnect => self.subscribers.clear(),
                    _ => {} // Send/Resize/Acp can't be served with no link
                }
            }
            if !self.subscribers.is_empty() {
                // A subscriber raced in after the stop decision — revive as a fresh
                // lifecycle: reset backoff so the new subscriber isn't penalized by
                // the old connection's accumulated reconnect_attempts.
                self.reconnect_attempts = 0;
                return false;
            }
            router.pods.remove(&self.pod_key);
            router.status_listeners.remove(&self.pod_key);
            router.acp_listeners.remove(&self.pod_key);
            router.on_pod_disconnected.clone()
        };
        if let Some(cb) = cb {
            cb(self.pod_key.clone());
        }
        true
    }
}
