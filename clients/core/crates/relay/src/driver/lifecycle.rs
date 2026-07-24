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
mod tests {
    use std::collections::HashMap;
    use std::sync::{Arc, Mutex};

    use agentsmesh_transport::runtime::PlatformRuntime;
    use futures::channel::{mpsc, oneshot};
    use parking_lot::RwLock;

    use super::*;
    use crate::pool::PoolRouter;
    use crate::types::{OutputCallback, StatusSnapshot};

    fn callback() -> OutputCallback {
        Arc::new(|_| {})
    }

    fn router() -> Arc<RwLock<PoolRouter>> {
        Arc::new(RwLock::new(PoolRouter {
            pods: HashMap::new(),
            next_generation: 0,
            status_listeners: HashMap::new(),
            acp_listeners: HashMap::new(),
            on_pod_disconnected: None,
            on_pod_generation_disconnected: None,
        }))
    }

    fn driver(
        router: Arc<RwLock<PoolRouter>>,
        generation: u32,
        with_subscriber: bool,
    ) -> Driver<PlatformRuntime> {
        let mut subscribers = HashMap::new();
        if with_subscriber {
            subscribers.insert("sub-1".to_string(), Subscriber::pending(callback(), None));
        }
        Driver {
            runtime: PlatformRuntime,
            router,
            pod_key: "pod-1".to_string(),
            generation,
            relay_url: "ws://127.0.0.1:1".to_string(),
            relay_token: "token".to_string(),
            snapshot: Arc::new(RwLock::new(StatusSnapshot::default())),
            status: RelayStatus::Connecting,
            snapshot_received: false,
            reconnect_attempts: 3,
            runner_disconnected: false,
            pod_size: None,
            subscribers,
            last_input: None,
            pending_resize: None,
            grace_deadline: None,
        }
    }

    #[tokio::test]
    async fn backoff_stops_without_subscribers_or_when_command_channel_closes() {
        let mut no_subscribers = driver(router(), 1, false);
        let (_tx, mut rx) = mpsc::unbounded();
        assert!(matches!(no_subscribers.backoff(&mut rx).await, Flow::Stop));

        let mut closed_channel = driver(router(), 1, true);
        let (tx, mut rx) = mpsc::unbounded();
        drop(tx);
        assert!(matches!(closed_channel.backoff(&mut rx).await, Flow::Stop));
    }

    #[tokio::test]
    async fn backoff_processes_add_remove_resize_and_disconnect_commands() {
        let mut lifecycle = driver(router(), 1, true);
        let (tx, mut rx) = mpsc::unbounded();
        let (ready_tx, ready_rx) = oneshot::channel();
        tx.unbounded_send(Command::AddSubscriber {
            sub_id: "sub-2".to_string(),
            cb: callback(),
            ready: Some(ready_tx),
        })
        .unwrap();
        tx.unbounded_send(Command::RemoveSubscriber {
            sub_id: "sub-1".to_string(),
        })
        .unwrap();
        tx.unbounded_send(Command::Resize {
            cols: 120,
            rows: 40,
            force: true,
        })
        .unwrap();
        tx.unbounded_send(Command::Send {
            data: "dropped-offline".to_string(),
        })
        .unwrap();
        tx.unbounded_send(Command::RemoveSubscriber {
            sub_id: "sub-2".to_string(),
        })
        .unwrap();

        assert!(matches!(lifecycle.backoff(&mut rx).await, Flow::Stop));
        assert_eq!(
            lifecycle.pending_resize.map(|v| (v.0, v.1, v.3)),
            Some((120, 40, true))
        );
        assert!(
            ready_rx.await.is_err(),
            "removed subscriber must cancel readiness"
        );

        let mut disconnected = driver(router(), 1, true);
        let (tx, mut rx) = mpsc::unbounded();
        tx.unbounded_send(Command::Resize {
            cols: 0,
            rows: 0,
            force: false,
        })
        .unwrap();
        tx.unbounded_send(Command::Disconnect).unwrap();
        assert!(matches!(disconnected.backoff(&mut rx).await, Flow::Stop));
        assert!(disconnected.subscribers.is_empty());
        assert!(disconnected.pending_resize.is_none());
    }

    #[test]
    fn finalize_revives_for_queued_subscriber_and_drains_other_commands() {
        let mut lifecycle = driver(router(), 1, false);
        let (tx, mut rx) = mpsc::unbounded();
        tx.unbounded_send(Command::Resize {
            cols: 90,
            rows: 30,
            force: false,
        })
        .unwrap();
        tx.unbounded_send(Command::AddSubscriber {
            sub_id: "late".to_string(),
            cb: callback(),
            ready: None,
        })
        .unwrap();

        assert!(!lifecycle.try_finalize(&mut rx));
        assert!(lifecycle.subscribers.contains_key("late"));
        assert_eq!(lifecycle.reconnect_attempts, 0);
    }

    #[test]
    fn finalize_disconnect_wins_and_owned_generation_fires_both_callbacks() {
        let router = router();
        let _pod_rx = router
            .write()
            .insert_test_pod("pod-1", 7, RelayStatus::Connected);
        let plain = Arc::new(Mutex::new(Vec::new()));
        let generated = Arc::new(Mutex::new(Vec::new()));
        {
            let mut state = router.write();
            let plain_events = Arc::clone(&plain);
            state.on_pod_disconnected = Some(Arc::new(move |pod| {
                plain_events.lock().unwrap().push(pod);
            }));
            let generation_events = Arc::clone(&generated);
            state.on_pod_generation_disconnected = Some(Arc::new(move |pod, generation| {
                generation_events.lock().unwrap().push((pod, generation));
            }));
        }
        let mut lifecycle = driver(Arc::clone(&router), 7, true);
        let (tx, mut rx) = mpsc::unbounded();
        tx.unbounded_send(Command::RemoveSubscriber {
            sub_id: "missing".to_string(),
        })
        .unwrap();
        tx.unbounded_send(Command::Disconnect).unwrap();

        assert!(lifecycle.try_finalize(&mut rx));
        assert!(!router.read().pods.contains_key("pod-1"));
        assert_eq!(*plain.lock().unwrap(), vec!["pod-1".to_string()]);
        assert_eq!(*generated.lock().unwrap(), vec![("pod-1".to_string(), 7)]);
    }

    #[test]
    fn stale_generation_cannot_remove_replacement_driver() {
        let router = router();
        let _replacement_rx = router
            .write()
            .insert_test_pod("pod-1", 9, RelayStatus::Connecting);
        let mut stale = driver(Arc::clone(&router), 8, false);
        let (_tx, mut rx) = mpsc::unbounded();

        assert!(stale.try_finalize(&mut rx));
        assert_eq!(router.read().pods.get("pod-1").unwrap().generation, 9);
    }
}
// LCOV_EXCL_STOP
