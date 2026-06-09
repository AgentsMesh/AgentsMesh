use std::sync::Arc;
use std::time::Duration;

use agentsmesh_transport::runtime::{timeout, Runtime};
use futures::channel::mpsc;
use futures::stream::StreamExt;
use futures::FutureExt;
use parking_lot::RwLock;
use tracing::{debug, error, info, warn};

use crate::reconnect::ReconnectPolicy;
use crate::stream_source::{EventStreamSource, StreamFrame};
use crate::subscription_manager::{dispatch_event, set_state, Inner};
use crate::types::{ConnectionState, EventSubscriptionManagerOptions};

/// No events for this long → the stream is stalled (proxy buffered the
/// response, or the server went silent). Rebuild the link. HTTP/2 PING keeps
/// the transport alive; this is the application-level "still receiving?" check.
const DEFAULT_IDLE_MS: u64 = 60_000;

/// Default cap for a single connect attempt when options leave it 0. A hung
/// TCP/TLS handshake has no OS-level timeout here, so without this the loop
/// could block in `subscribe().await` forever and never see a shutdown.
const DEFAULT_CONNECT_MS: u64 = 15_000;

pub(crate) struct ManagerOpts {
    pub initial_reconnect_delay_ms: u64,
    pub max_reconnect_delay_ms: u64,
    pub idle_timeout_ms: u64,
    pub connect_timeout_ms: u64,
}

impl ManagerOpts {
    pub fn from_options(o: &EventSubscriptionManagerOptions) -> Self {
        Self {
            initial_reconnect_delay_ms: o.initial_reconnect_delay_ms,
            max_reconnect_delay_ms: o.max_reconnect_delay_ms,
            idle_timeout_ms: if o.idle_timeout_ms == 0 {
                DEFAULT_IDLE_MS
            } else {
                o.idle_timeout_ms
            },
            connect_timeout_ms: if o.connect_timeout_ms == 0 {
                DEFAULT_CONNECT_MS
            } else {
                o.connect_timeout_ms
            },
        }
    }
}

pub(crate) async fn connection_loop<R: Runtime>(
    runtime: R,
    inner: Arc<RwLock<Inner>>,
    source: Arc<dyn EventStreamSource>,
    opts: ManagerOpts,
    mut shutdown_rx: mpsc::UnboundedReceiver<()>,
    mut nudge_rx: mpsc::UnboundedReceiver<()>,
) {
    let mut policy = ReconnectPolicy::new(opts.initial_reconnect_delay_ms, opts.max_reconnect_delay_ms);

    loop {
        set_state(&inner, ConnectionState::Connecting);

        let outcome = run_session(&runtime, &inner, &source, &opts, &mut policy, &mut shutdown_rx).await;

        match outcome {
            SessionClose::Shutdown => {
                set_state(&inner, ConnectionState::Disconnected);
                break;
            }
            // Clean server close → backoff already at baseline (reset on the
            // last data frame); reconnect promptly.
            SessionClose::ServerClosed => {
                policy.reset();
                if backoff_interrupted(&runtime, &inner, &mut policy, &mut shutdown_rx, &mut nudge_rx).await {
                    break;
                }
            }
            // No total cap: a realtime stream must self-heal indefinitely. The
            // backoff escalates (capped at max_reconnect_delay_ms) but never
            // gives up — the only exit is an explicit shutdown.
            SessionClose::ConnectFailed | SessionClose::StreamError | SessionClose::IdleTimeout => {
                if backoff_interrupted(&runtime, &inner, &mut policy, &mut shutdown_rx, &mut nudge_rx).await {
                    break;
                }
            }
        }
    }
}

enum SessionClose {
    /// `disconnect()` was called — exit cleanly, no reconnect.
    Shutdown,
    /// subscribe() errored or timed out before the stream was usable.
    ConnectFailed,
    /// Stream yielded an error mid-flight (network reset, 5xx).
    StreamError,
    /// No events for `idle_timeout_ms` — server silent or proxy buffered.
    IdleTimeout,
    /// Stream ended cleanly (server closed).
    ServerClosed,
}

async fn run_session<R: Runtime>(
    runtime: &R,
    inner: &Arc<RwLock<Inner>>,
    source: &Arc<dyn EventStreamSource>,
    opts: &ManagerOpts,
    policy: &mut ReconnectPolicy,
    shutdown_rx: &mut mpsc::UnboundedReceiver<()>,
) -> SessionClose {
    let connect = timeout(
        runtime,
        Duration::from_millis(opts.connect_timeout_ms),
        source.subscribe(),
    )
    .await;
    let stream = match connect {
        Ok(Ok(s)) => s,
        Ok(Err(e)) => {
            error!("events: subscribe failed: {:?}", e);
            return SessionClose::ConnectFailed;
        }
        Err(_) => {
            warn!("events: connect timed out after {}ms", opts.connect_timeout_ms);
            return SessionClose::ConnectFailed;
        }
    };
    futures::pin_mut!(stream);

    // NOT Connected yet. Only the first real frame (data-ready) flips the light
    // green, so a handshake-but-silent stream can't masquerade as healthy.
    let idle = Duration::from_millis(opts.idle_timeout_ms);
    let mut data_ready = false;

    loop {
        let next_fut = stream.next().fuse();
        let sleep_fut = runtime.sleep(idle).fuse();
        futures::pin_mut!(next_fut, sleep_fut);

        // Biased: a pending shutdown must win even when the stream closes in
        // the same poll. Plain select! picks a ready branch at random, so a
        // server close racing disconnect() could be read as ServerClosed and
        // reconnect after shutdown — breaking disconnect terminality.
        futures::select_biased! {
            _ = shutdown_rx.next() => return SessionClose::Shutdown,
            item = next_fut => match item {
                Some(Ok(frame)) => {
                    if !data_ready {
                        data_ready = true;
                        // First real frame proves the link is alive: reset
                        // backoff (not on the bare handshake) and report
                        // Connected, so a long-lived session that later drops
                        // doesn't carry a stale failure count into reconnect.
                        policy.reset();
                        set_state(inner, ConnectionState::Connected);
                    }
                    // Keepalive sentinels only prove liveness — Connected + idle
                    // reset are both handled above. Only business events dispatch.
                    if let StreamFrame::Event(evt) = frame {
                        dispatch_event(inner, &evt);
                    }
                }
                Some(Err(e)) => {
                    warn!("events: stream error: {:?}", e);
                    return SessionClose::StreamError;
                }
                None => {
                    debug!("events: server closed stream");
                    return SessionClose::ServerClosed;
                }
            },
            _ = sleep_fut => {
                warn!("events: idle timeout ({}ms) — reconnecting", opts.idle_timeout_ms);
                return SessionClose::IdleTimeout;
            }
        }
    }
}

/// Wait the escalating (capped, never-exhausting) reconnect delay while still
/// watching for shutdown. Returns true if shutdown was requested mid-wait.
async fn backoff_interrupted<R: Runtime>(
    runtime: &R,
    inner: &Arc<RwLock<Inner>>,
    policy: &mut ReconnectPolicy,
    shutdown_rx: &mut mpsc::UnboundedReceiver<()>,
    nudge_rx: &mut mpsc::UnboundedReceiver<()>,
) -> bool {
    let delay = policy.next_delay();
    set_state(inner, ConnectionState::Reconnecting);
    info!("events: reconnecting in {:?}", delay);
    let sleep_fut = runtime.sleep(delay).fuse();
    futures::pin_mut!(sleep_fut);
    // Biased: a shutdown during backoff must win over both the nudge and the
    // elapsing sleep so disconnect() terminates the loop deterministically.
    futures::select_biased! {
        _ = shutdown_rx.next() => {
            set_state(inner, ConnectionState::Disconnected);
            true
        }
        _ = nudge_rx.next() => {
            debug!("events: nudge — skipping backoff, reconnecting now");
            false
        }
        _ = sleep_fut => false,
    }
}
