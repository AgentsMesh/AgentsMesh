use agentsmesh_transport::runtime::Runtime;
use futures::channel::oneshot;

use super::Driver;
use crate::types::OutputCallback;

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(super) enum BaselineState {
    Pending,
    Ready,
}

pub(super) struct Subscriber {
    pub callback: OutputCallback,
    pub baseline: BaselineState,
    ready: Option<oneshot::Sender<()>>,
}

impl Subscriber {
    pub(super) fn pending(callback: OutputCallback, ready: Option<oneshot::Sender<()>>) -> Self {
        Self {
            callback,
            baseline: BaselineState::Pending,
            ready,
        }
    }
}

pub(super) struct BaselineDelivery {
    pub callback: OutputCallback,
    ready: Option<oneshot::Sender<()>>,
}

impl BaselineDelivery {
    pub(super) fn complete(self, replay: &[u8]) {
        // Every baseline must cross the output callback before readiness is
        // published. An empty Vec is an intentional ordering marker for a
        // legitimate empty PTY snapshot (and, below, an ACP snapshot); the
        // Electron renderer consumes it as a barrier without forwarding it to
        // the terminal callback.
        (self.callback)(replay.to_vec());
        if let Some(ready) = self.ready {
            let _ = ready.send(());
        }
    }
}

impl<R: Runtime> Driver<R> {
    pub(super) fn insert_pending_subscriber(
        &mut self,
        sub_id: String,
        cb: OutputCallback,
        ready: Option<oneshot::Sender<()>>,
    ) {
        self.subscribers
            .insert(sub_id, Subscriber::pending(cb, ready));
    }

    pub(super) fn mark_subscribers_pending(&mut self) {
        for subscriber in self.subscribers.values_mut() {
            subscriber.baseline = BaselineState::Pending;
        }
    }

    pub(super) fn needs_baseline(&self) -> bool {
        !self.snapshot_received
            || self
                .subscribers
                .values()
                .any(|subscriber| subscriber.baseline == BaselineState::Pending)
    }

    pub(super) fn ready_callbacks(&self) -> Vec<&OutputCallback> {
        self.subscribers
            .values()
            .filter(|subscriber| subscriber.baseline == BaselineState::Ready)
            .map(|subscriber| &subscriber.callback)
            .collect()
    }

    pub(super) fn take_pending_deliveries(&mut self) -> Vec<BaselineDelivery> {
        self.subscribers
            .values_mut()
            .filter_map(|subscriber| {
                if subscriber.baseline == BaselineState::Ready {
                    return None;
                }
                subscriber.baseline = BaselineState::Ready;
                Some(BaselineDelivery {
                    callback: subscriber.callback.clone(),
                    ready: subscriber.ready.take(),
                })
            })
            .collect()
    }

    pub(super) fn take_all_deliveries_ready(&mut self) -> Vec<BaselineDelivery> {
        self.subscribers
            .values_mut()
            .map(|subscriber| {
                subscriber.baseline = BaselineState::Ready;
                BaselineDelivery {
                    callback: subscriber.callback.clone(),
                    ready: subscriber.ready.take(),
                }
            })
            .collect()
    }

    pub(super) fn mark_pending_ready(&mut self) {
        // ACP snapshots have no PTY bytes to replay, but desktop subscribers
        // still need an observable output event before their renderer-level
        // subscribe promise can become ready.
        for delivery in self.take_pending_deliveries() {
            delivery.complete(&[]);
        }
    }
}
