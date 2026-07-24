use std::collections::HashMap;
use std::sync::Arc;

use agentsmesh_transport::runtime::Runtime;
use futures::channel::{mpsc, oneshot};
use parking_lot::RwLock;
use web_time::Instant;

use crate::command::Command;
use crate::connection;
use crate::pool::PoolRouter;
use crate::retry;
use crate::types::{OutputCallback, RelayStatus, StatusSnapshot};

mod frame;
mod lifecycle;
mod session;
mod subscriber;

use subscriber::Subscriber;

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
    generation: u32,
    relay_url: String,
    relay_token: String,
    snapshot: Arc<RwLock<StatusSnapshot>>,

    status: RelayStatus,
    snapshot_received: bool,
    reconnect_attempts: u32,
    runner_disconnected: bool,
    pod_size: Option<(u16, u16)>,
    subscribers: HashMap<String, Subscriber>,
    last_input: Option<(String, Instant)>,
    pending_resize: Option<(u16, u16, Instant, bool)>,
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
        generation: u32,
        cmd_rx: mpsc::UnboundedReceiver<Command>,
        first_sub: (String, OutputCallback, Option<oneshot::Sender<()>>),
    ) {
        let mut subscribers = HashMap::new();
        subscribers.insert(first_sub.0, Subscriber::pending(first_sub.1, first_sub.2));
        let driver = Self {
            runtime: runtime.clone(),
            router,
            pod_key,
            generation,
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
                    self.mark_subscribers_pending();
                    // Fresh connection: assume the runner is up until a
                    // RunnerDisconnected frame says otherwise, and don't let input
                    // dedup carry a stale baseline across the reconnect.
                    self.runner_disconnected = false;
                    self.last_input = None;
                    self.write_snapshot();
                    match self.run_session(&sender, &mut receiver, &mut cmd_rx).await {
                        SessionEnd::Shutdown => true,
                        SessionEnd::Closed => {
                            // A closed transport invalidates this generation's
                            // baseline immediately. Publish that transition before
                            // entering backoff so renderers cannot keep treating a
                            // stale Connected status as data-ready.
                            self.set_status(RelayStatus::Connecting);
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
}
