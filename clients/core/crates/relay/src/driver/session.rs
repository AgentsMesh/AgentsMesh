use std::time::Duration;

use agentsmesh_protocol::{
    decode_message, encode_json_message, encode_message, encode_resize, MsgType,
};
use agentsmesh_transport::runtime::Runtime;
use agentsmesh_transport::{WsMessage, WsReceiver, WsSender};
use futures::channel::mpsc;
use futures::{select_biased, FutureExt, StreamExt};
use web_time::Instant;

use super::{Driver, SessionEnd, IDLE_TICK_MS};
use crate::command::Command;
use crate::dispatch::{dispatch_message, DispatchAction};
use crate::retry;
use crate::types::{AcpCallback, OutputCallback, RelayStatus};

impl<R: Runtime> Driver<R> {
    /// One connected session: pump inbound frames, fire periodic timers
    /// (snapshot keepalive, resize debounce, disconnect grace), and apply
    /// commands — until the link drops or the pod is done.
    pub(super) async fn run_session(
        &mut self,
        sender: &WsSender,
        receiver: &mut WsReceiver,
        cmd_rx: &mut mpsc::UnboundedReceiver<Command>,
    ) -> SessionEnd {
        let mut last_resync = Instant::now();
        let mut resync_count: u32 = 0;

        loop {
            let sleep = self.runtime.sleep(self.next_timer(last_resync)).fuse();
            let recv = receiver.recv().fuse();
            let cmd = cmd_rx.next().fuse();
            futures::pin_mut!(sleep, recv, cmd);

            // biased: control commands (Disconnect/Resize/…) take priority over a
            // flood of inbound data frames, so shutdown/input can't be starved.
            select_biased! {
                c = cmd => match c {
                    None => return SessionEnd::Shutdown,
                    Some(cmd) => {
                        if let Some(end) = self.handle_command(sender, cmd) {
                            return end;
                        }
                    }
                },
                msg = recv => match msg {
                    Ok(WsMessage::Binary(data)) => self.handle_frame(&data),
                    Ok(WsMessage::Close(_)) | Err(_) => return SessionEnd::Closed,
                    Ok(WsMessage::Text(_)) => {}
                },
                _ = sleep => {}
            }

            let now = Instant::now();
            if !self.snapshot_received && elapsed_ms(now, last_resync) >= retry::SNAPSHOT_TIMEOUT_MS {
                resync_count += 1;
                if resync_count > retry::SNAPSHOT_GIVEUP_ATTEMPTS {
                    // Connected but data never arrived: rebuild the link so it
                    // self-heals once relay/runner recovers, not blank-forever.
                    tracing::warn!(target: "relay", pod_key = %self.pod_key, attempts = resync_count, "snapshot never arrived — rebuilding link");
                    return SessionEnd::Closed;
                }
                let _ = sender.send_binary(encode_message(MsgType::Resync, &[]));
                last_resync = now;
            }
            if let Some((c, r, at)) = self.pending_resize {
                if elapsed_ms(now, at) >= retry::RESIZE_DEBOUNCE_MS {
                    let _ = sender.send_binary(encode_resize(c, r));
                    self.pending_resize = None;
                }
            }
            if let Some(at) = self.grace_deadline {
                if now >= at {
                    self.grace_deadline = None;
                    if self.subscribers.is_empty() {
                        return SessionEnd::Shutdown;
                    }
                }
            }
        }
    }

    /// Soonest pending deadline (snapshot retry / resize debounce / grace),
    /// capped at the idle tick so the select stays command-responsive.
    fn next_timer(&self, last_resync: Instant) -> Duration {
        let now = Instant::now();
        let mut next = Duration::from_millis(IDLE_TICK_MS);
        if !self.snapshot_received {
            next = next.min(remaining(now, last_resync, retry::SNAPSHOT_TIMEOUT_MS));
        }
        if let Some((_, _, at)) = self.pending_resize {
            next = next.min(remaining(now, at, retry::RESIZE_DEBOUNCE_MS));
        }
        if let Some(at) = self.grace_deadline {
            next = next.min(at.saturating_duration_since(now));
        }
        next
    }

    fn handle_frame(&mut self, data: &[u8]) {
        let Ok((msg_type, payload)) = decode_message(data) else {
            return;
        };
        let subs: Vec<&OutputCallback> = self.subscribers.values().collect();
        match dispatch_message(msg_type, payload, &subs) {
            DispatchAction::Snapshot(snap) => {
                self.snapshot_received = true;
                // Data-ready (snapshot in hand), not the bare handshake, resets backoff.
                self.reconnect_attempts = 0;
                if snap.cols > 0 && snap.rows > 0 {
                    self.pod_size = Some((snap.cols, snap.rows));
                    // Explicit: set_status(Connected) below short-circuits when
                    // already Connected, so flush the new size to the mirror here.
                    self.write_snapshot();
                }
                // Only now is the link truly Connected — the connection light goes
                // green on real data-ready, not the bare WS handshake (kills the
                // "green but blank" gap).
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
                // AcpSnapshot is the ACP pod's data-ready signal: ACP pods send
                // MsgType::AcpSnapshot on subscribe (pod_relay_acp.go), never the
                // PTY MsgType::Snapshot. Treat it like the Snapshot arm so an ACP
                // link reaches Connected — without this it sits Connecting forever
                // and send_acp_command (gated on Connected) always rejects.
                if msg_type == MsgType::AcpSnapshot {
                    self.snapshot_received = true;
                    self.reconnect_attempts = 0;
                    self.set_status(RelayStatus::Connected);
                }
                let listeners: Vec<AcpCallback> = {
                    let router = self.router.read();
                    router
                        .acp_listeners
                        .get(&self.pod_key)
                        .cloned()
                        .unwrap_or_default()
                };
                for l in &listeners {
                    // Isolate a panicking ACP listener (same rationale as
                    // notify_status); native only, wasm is panic=abort.
                    let payload = payload.clone();
                    let _ = std::panic::catch_unwind(std::panic::AssertUnwindSafe(|| {
                        l(msg_type, payload)
                    }));
                }
            }
            DispatchAction::None => {}
        }
    }

    /// Returns `Some(end)` when the command terminates the driver.
    fn handle_command(&mut self, sender: &WsSender, cmd: Command) -> Option<SessionEnd> {
        match cmd {
            Command::AddSubscriber { sub_id, cb } => {
                self.subscribers.insert(sub_id, cb);
                self.grace_deadline = None;
                // Replay the current screen to the freshly-joined subscriber.
                let _ = sender.send_binary(encode_message(MsgType::Resync, &[]));
            }
            Command::RemoveSubscriber { sub_id } => {
                self.subscribers.remove(&sub_id);
                if self.subscribers.is_empty() {
                    // Grace: a tab re-open within the window reuses this link.
                    self.grace_deadline =
                        Some(Instant::now() + Duration::from_millis(retry::DISCONNECT_DELAY_MS));
                }
            }
            Command::Send { data } => self.send_input(sender, &data),
            Command::Resize { cols, rows, force } => {
                if cols == 0 || rows == 0 {
                    return None;
                }
                if force {
                    let _ = sender.send_binary(encode_resize(cols, rows));
                    self.pending_resize = None;
                } else {
                    self.pending_resize = Some((cols, rows, Instant::now()));
                }
            }
            Command::SendAcp { command } => {
                if let Ok(msg) = encode_json_message(MsgType::AcpCommand, &command) {
                    let _ = sender.send_binary(msg);
                }
            }
            Command::Disconnect => {
                // Explicit disconnect tears down regardless of subscribers; clear
                // them so try_finalize sees empty and won't revive the link.
                self.subscribers.clear();
                return Some(SessionEnd::Shutdown);
            }
        }
        None
    }

    fn send_input(&mut self, sender: &WsSender, data: &str) {
        if data.len() > 1 {
            let now = Instant::now();
            if let Some((last, at)) = &self.last_input {
                if last == data
                    && now.saturating_duration_since(*at).as_millis()
                        < retry::INPUT_DEDUP_WINDOW_MS as u128
                {
                    return;
                }
            }
            self.last_input = Some((data.to_string(), now));
        }
        let _ = sender.send_binary(encode_message(MsgType::Input, data.as_bytes()));
    }
}

fn elapsed_ms(now: Instant, since: Instant) -> u64 {
    // saturating: web_time::Instant on wasm can momentarily go backwards
    // (performance.now() throttling), and a bare duration_since would panic and
    // kill the driver task (skipping teardown → wedged pod).
    now.saturating_duration_since(since).as_millis() as u64
}

fn remaining(now: Instant, since: Instant, window_ms: u64) -> Duration {
    Duration::from_millis(window_ms).saturating_sub(now.saturating_duration_since(since))
}
