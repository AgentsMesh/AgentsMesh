use std::time::Duration;

use agentsmesh_protocol::{encode_json_message, encode_message, encode_resize, MsgType};
use agentsmesh_transport::runtime::Runtime;
use agentsmesh_transport::{WsMessage, WsReceiver, WsSender};
use futures::channel::mpsc;
use futures::{select_biased, FutureExt, StreamExt};
use web_time::Instant;

use super::{Driver, SessionEnd, IDLE_TICK_MS};
use crate::command::Command;
use crate::retry;

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
                        let requests_baseline = matches!(&cmd, Command::AddSubscriber { .. });
                        if let Some(end) = self.handle_command(sender, cmd) {
                            return end;
                        }
                        if requests_baseline {
                            last_resync = Instant::now();
                            resync_count = 0;
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
            if self.needs_baseline() && elapsed_ms(now, last_resync) >= retry::SNAPSHOT_TIMEOUT_MS {
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
            self.flush_pending_resize_if_ready(sender, now);
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
        if self.needs_baseline() {
            next = next.min(remaining(now, last_resync, retry::SNAPSHOT_TIMEOUT_MS));
        }
        if !self.needs_baseline() {
            if let Some((_, _, at, force)) = self.pending_resize {
                if force {
                    return Duration::ZERO;
                }
                next = next.min(remaining(now, at, retry::RESIZE_DEBOUNCE_MS));
            }
        }
        if let Some(at) = self.grace_deadline {
            next = next.min(at.saturating_duration_since(now));
        }
        next
    }

    /// Returns `Some(end)` when the command terminates the driver.
    fn handle_command(&mut self, sender: &WsSender, cmd: Command) -> Option<SessionEnd> {
        match cmd {
            Command::AddSubscriber { sub_id, cb, ready } => {
                self.insert_pending_subscriber(sub_id, cb, ready);
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
                if force && !self.needs_baseline() {
                    let _ = sender.send_binary(encode_resize(cols, rows));
                    self.pending_resize = None;
                } else {
                    self.queue_resize(cols, rows, force);
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

// LCOV_EXCL_START: test-only code
#[cfg(all(test, not(target_arch = "wasm32")))]
#[path = "session_tests.rs"]
mod session_tests;
// LCOV_EXCL_STOP
