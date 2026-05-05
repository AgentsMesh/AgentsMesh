use std::sync::Arc;

use agentsmesh_transport::runtime::Runtime;
use agentsmesh_transport::{WebSocketConnection, WsMessage, WsReceiver, WsSender};
use tokio::sync::{mpsc, RwLock};
use tracing::{debug, error, info, warn};

use crate::event_types::EventType;
use crate::heartbeat::{HeartbeatEvent, HeartbeatManager};
use crate::reconnect::{self, ReconnectPolicy};
use crate::subscription_manager::{dispatch_event, set_state, Inner};
use crate::types::{ConnectionState, EventSubscriptionManagerOptions, PingMessage};

pub(crate) struct ManagerOpts {
    pub max_reconnect_attempts: u32,
    pub initial_reconnect_delay_ms: u64,
    pub max_reconnect_delay_ms: u64,
    pub ping_interval_ms: u64,
    pub pong_timeout_ms: u64,
}

impl ManagerOpts {
    pub fn from_options(o: &EventSubscriptionManagerOptions) -> Self {
        Self {
            max_reconnect_attempts: o.max_reconnect_attempts,
            initial_reconnect_delay_ms: o.initial_reconnect_delay_ms,
            max_reconnect_delay_ms: o.max_reconnect_delay_ms,
            ping_interval_ms: o.ping_interval_ms,
            pong_timeout_ms: o.pong_timeout_ms,
        }
    }
}

pub(crate) async fn connection_loop<R: Runtime>(
    runtime: R,
    inner: Arc<RwLock<Inner>>,
    url: String,
    opts: ManagerOpts,
    mut shutdown_rx: mpsc::Receiver<()>,
) {
    let mut reconnect_policy = ReconnectPolicy::new(
        opts.initial_reconnect_delay_ms,
        opts.max_reconnect_delay_ms,
        opts.max_reconnect_attempts,
    );

    loop {
        set_state(&inner, ConnectionState::Connecting).await;
        info!("events: connecting to {}", url);

        let conn = match WebSocketConnection::connect(&url).await {
            Ok(c) => c,
            Err(e) => {
                error!("events: connection failed: {}", e);
                if !schedule_reconnect(&runtime, &inner, &mut reconnect_policy, &mut shutdown_rx)
                    .await
                {
                    break;
                }
                continue;
            }
        };

        set_state(&inner, ConnectionState::Connected).await;
        reconnect_policy.reset();
        info!("events: connected");

        let (sender, mut receiver) = conn.into_split();
        let close_code =
            run_session(&runtime, &inner, sender, &mut receiver, &opts, &mut shutdown_rx).await;

        if !reconnect::should_reconnect(close_code) {
            debug!("events: clean close (1000), not reconnecting");
            set_state(&inner, ConnectionState::Disconnected).await;
            break;
        }

        if !schedule_reconnect(&runtime, &inner, &mut reconnect_policy, &mut shutdown_rx).await {
            break;
        }
    }
}

async fn run_session<R: Runtime>(
    runtime: &R,
    inner: &Arc<RwLock<Inner>>,
    sender: WsSender,
    receiver: &mut WsReceiver,
    opts: &ManagerOpts,
    shutdown_rx: &mut mpsc::Receiver<()>,
) -> Option<u16> {
    let (hb_event_tx, mut hb_event_rx) = mpsc::channel(16);
    let mut heartbeat =
        HeartbeatManager::with_runtime(runtime.clone(), opts.ping_interval_ms, opts.pong_timeout_ms);
    heartbeat.start(hb_event_tx);

    let close_code: Option<u16>;

    loop {
        tokio::select! {
            result = receiver.recv() => {
                match result {
                    Ok(WsMessage::Text(text)) => {
                        handle_text_message(inner, &mut heartbeat, &text).await;
                    }
                    Ok(WsMessage::Close(code)) => {
                        close_code = code;
                        break;
                    }
                    Err(_) => { close_code = None; break; }
                    _ => {}
                }
            }
            hb_event = hb_event_rx.recv() => {
                match hb_event {
                    Some(HeartbeatEvent::SendPing) => {
                        if !send_ping(&sender) { close_code = None; break; }
                    }
                    Some(HeartbeatEvent::PongTimeout) => {
                        warn!("events: pong timeout, closing");
                        sender.close();
                        close_code = Some(4000);
                        break;
                    }
                    None => { close_code = None; break; }
                }
            }
            _ = shutdown_rx.recv() => {
                sender.close();
                close_code = Some(1000);
                break;
            }
        }
    }

    heartbeat.stop();
    close_code
}

async fn handle_text_message<R: Runtime>(
    inner: &Arc<RwLock<Inner>>,
    heartbeat: &mut HeartbeatManager<R>,
    text: &str,
) {
    use crate::types::RealtimeEvent;
    match serde_json::from_str::<RealtimeEvent>(text) {
        Ok(event) => {
            if event.event_type == EventType::Pong {
                heartbeat.pong_received();
            } else {
                dispatch_event(inner, &event).await;
            }
        }
        Err(e) => warn!("events: parse error: {}", e),
    }
}

fn send_ping(sender: &WsSender) -> bool {
    let ts = web_time::SystemTime::now()
        .duration_since(web_time::UNIX_EPOCH)
        .unwrap_or_default()
        .as_millis() as i64;
    let ping = serde_json::to_string(&PingMessage::new(ts)).unwrap_or_default();
    sender.send_text(ping).is_ok()
}

async fn schedule_reconnect<R: Runtime>(
    runtime: &R,
    inner: &Arc<RwLock<Inner>>,
    policy: &mut ReconnectPolicy,
    shutdown_rx: &mut mpsc::Receiver<()>,
) -> bool {
    match policy.next_delay() {
        Some(delay) => {
            set_state(inner, ConnectionState::Reconnecting).await;
            info!(
                "events: reconnecting in {:?} (attempt {}/{})",
                delay,
                policy.attempts(),
                policy.max_attempts()
            );
            tokio::select! {
                _ = runtime.sleep(delay) => true,
                _ = shutdown_rx.recv() => {
                    set_state(inner, ConnectionState::Disconnected).await;
                    false
                }
            }
        }
        None => {
            error!("events: max reconnect attempts reached");
            set_state(inner, ConnectionState::Disconnected).await;
            false
        }
    }
}
