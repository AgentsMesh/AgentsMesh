use agentsmesh_protocol::encode_resize;
use agentsmesh_transport::runtime::Runtime;
use agentsmesh_transport::WsSender;
use futures::channel::mpsc;
use futures::{select, FutureExt, StreamExt};
use web_time::Instant;

use super::subscriber::Subscriber;
use super::{Driver, Flow};
use crate::command::Command;
use crate::retry;
use crate::types::{RelayStatus, RelayStatusInfo};

impl<R: Runtime> Driver<R> {
    /// Wait out reconnect backoff while still accepting lifecycle commands.
    pub(super) async fn backoff(&mut self, cmd_rx: &mut mpsc::UnboundedReceiver<Command>) -> Flow {
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
                        self.subscribers.clear();
                        return Flow::Stop;
                    }
                    Some(Command::AddSubscriber { sub_id, cb, ready }) => {
                        self.insert_pending_subscriber(sub_id, cb, ready);
                    }
                    Some(Command::RemoveSubscriber { sub_id }) => {
                        self.subscribers.remove(&sub_id);
                        if self.subscribers.is_empty() {
                            return Flow::Stop;
                        }
                    }
                    Some(Command::Resize { cols, rows, force }) => {
                        if cols > 0 && rows > 0 {
                            self.queue_resize(cols, rows, force);
                        }
                    }
                    Some(_) => {}
                }
            }
        }
    }

    pub(super) fn set_status(&mut self, status: RelayStatus) {
        if self.status == status {
            return;
        }
        self.status = status;
        self.write_snapshot();
        self.notify_status();
    }

    pub(super) fn write_snapshot(&self) {
        let mut snapshot = self.snapshot.write();
        snapshot.status = self.status;
        snapshot.runner_disconnected = self.runner_disconnected;
        snapshot.pod_size = self.pod_size;
        snapshot.revision = snapshot.revision.wrapping_add(1);
    }

    pub(super) fn notify_status(&self) {
        let info = {
            let snapshot = self.snapshot.read();
            RelayStatusInfo {
                status: snapshot.status,
                runner_disconnected: snapshot.runner_disconnected,
                revision: snapshot.revision,
            }
        };
        let listener = {
            let router = self.router.read();
            router.status_listeners.get(&self.pod_key).cloned()
        };
        if let Some(listener) = listener {
            let _ =
                std::panic::catch_unwind(std::panic::AssertUnwindSafe(|| listener.deliver(info)));
        }
    }

    pub(super) fn queue_resize(&mut self, cols: u16, rows: u16, force: bool) {
        let force = force
            || self
                .pending_resize
                .is_some_and(|(_, _, _, pending_force)| pending_force);
        self.pending_resize = Some((cols, rows, Instant::now(), force));
    }

    /// Send a queued resize only after this transport generation has delivered
    /// its authoritative baseline. This invariant lives in the Rust driver so
    /// delayed renderer/main-process status notifications cannot reorder resize
    /// ahead of snapshot replay.
    pub(super) fn flush_pending_resize_if_ready(&mut self, sender: &WsSender, now: Instant) {
        let Some((cols, rows, queued_at, force)) = self.pending_resize else {
            return;
        };
        if self.needs_baseline()
            || (!force
                && now.saturating_duration_since(queued_at).as_millis()
                    < retry::RESIZE_DEBOUNCE_MS as u128)
        {
            return;
        }
        let _ = sender.send_binary(encode_resize(cols, rows));
        self.pending_resize = None;
    }

    /// Finalize under the same router lock used by subscribe, closing the race
    /// between a last-subscriber teardown and a newly queued subscriber.
    pub(super) fn try_finalize(&mut self, cmd_rx: &mut mpsc::UnboundedReceiver<Command>) -> bool {
        let callbacks = {
            let mut router = self.router.write();
            while let Ok(cmd) = cmd_rx.try_recv() {
                match cmd {
                    Command::AddSubscriber { sub_id, cb, ready } => {
                        self.subscribers
                            .insert(sub_id, Subscriber::pending(cb, ready));
                    }
                    Command::RemoveSubscriber { sub_id } => {
                        self.subscribers.remove(&sub_id);
                    }
                    Command::Disconnect => self.subscribers.clear(),
                    _ => {}
                }
            }
            if !self.subscribers.is_empty() {
                self.reconnect_attempts = 0;
                return false;
            }
            let owns_generation = router
                .pods
                .get(&self.pod_key)
                .map(|handle| handle.generation == self.generation)
                .unwrap_or(false);
            if !owns_generation {
                return true;
            }
            router.pods.remove(&self.pod_key);
            router.status_listeners.remove(&self.pod_key);
            router.acp_listeners.remove(&self.pod_key);
            (
                router.on_pod_disconnected.clone(),
                router.on_pod_generation_disconnected.clone(),
            )
        };
        if let Some(callback) = callbacks.0 {
            callback(self.pod_key.clone());
        }
        if let Some(callback) = callbacks.1 {
            callback(self.pod_key.clone(), self.generation);
        }
        true
    }
}

// LCOV_EXCL_START: test-only code
#[cfg(all(test, not(target_arch = "wasm32")))]
#[path = "lifecycle_tests.rs"]
mod lifecycle_tests;
// LCOV_EXCL_STOP
