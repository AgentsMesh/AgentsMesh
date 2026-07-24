use agentsmesh_protocol::{decode_message, MsgType};
use agentsmesh_transport::runtime::Runtime;

use super::Driver;
use crate::dispatch::{dispatch_message, DispatchAction};
use crate::types::RelayStatus;

impl<R: Runtime> Driver<R> {
    pub(super) fn handle_frame(&mut self, data: &[u8]) {
        let Ok((msg_type, payload)) = decode_message(data) else {
            return;
        };
        let action = dispatch_message(msg_type, payload, &self.ready_callbacks());
        match action {
            DispatchAction::Snapshot(snap) => {
                let targets = if snap.reset_all {
                    self.take_all_deliveries_ready()
                } else {
                    self.take_pending_deliveries()
                };
                for target in targets {
                    target.complete(&snap.replay);
                }
                self.snapshot_received = true;
                self.reconnect_attempts = 0;
                if snap.cols > 0 && snap.rows > 0 {
                    self.pod_size = Some((snap.cols, snap.rows));
                    self.write_snapshot();
                }
                self.set_status(RelayStatus::Connected);
                tracing::info!(target: "relay", pod_key = %self.pod_key, "data ready → connected");
            }
            DispatchAction::PodResized { cols, rows } => {
                self.pod_size = Some((cols, rows));
                self.write_snapshot();
            }
            DispatchAction::RunnerDisconnected => {
                self.runner_disconnected = true;
                tracing::warn!(target: "relay", pod_key = %self.pod_key, "runner disconnected");
                self.write_snapshot();
                self.notify_status();
            }
            DispatchAction::RunnerReconnected => {
                self.runner_disconnected = false;
                tracing::info!(target: "relay", pod_key = %self.pod_key, "runner reconnected");
                self.write_snapshot();
                self.notify_status();
            }
            DispatchAction::AcpMessage { msg_type, payload } => {
                // ACP pods use AcpSnapshot, rather than a PTY snapshot, as their
                // data-ready baseline.
                if msg_type == MsgType::AcpSnapshot {
                    self.mark_pending_ready();
                    self.snapshot_received = true;
                    self.reconnect_attempts = 0;
                    self.set_status(RelayStatus::Connected);
                }
                let listener = {
                    let router = self.router.read();
                    router.acp_listeners.get(&self.pod_key).cloned()
                };
                if let Some(listener) = listener {
                    let _ = std::panic::catch_unwind(std::panic::AssertUnwindSafe(|| {
                        listener.deliver(msg_type, payload)
                    }));
                }
            }
            DispatchAction::None => {}
        }
    }
}
