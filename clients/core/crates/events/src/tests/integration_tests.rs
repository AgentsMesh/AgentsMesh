//! Black-box reconnect / data-ready / timeout coverage. Drives the real
//! `connection_loop` (via `EventSubscriptionManager`) against a scripted
//! in-memory stream source. The loop's bug was state-machine logic, not wire
//! parsing — a futures-channel mock is enough and stays wasm/native-uniform.
//! These are the tests the old white-box `manager_tests` could never express
//! because they never ran the loop. Native-only (drive a tokio runtime).

use std::sync::atomic::{AtomicU32, Ordering::SeqCst};
use std::sync::Arc;
use std::time::{Duration, Instant};

use agentsmesh_api_client::ApiError;
use agentsmesh_transport::runtime::PlatformRuntime;
use futures::channel::mpsc::{self, UnboundedSender};
use parking_lot::Mutex;

use crate::stream_source::{BoxEventStream, EventStreamSource, StreamFrame, SubscribeFuture};
use crate::types::{ConnectionState, EventSubscriptionManagerOptions, RealtimeEvent};
use crate::{EventHandler, EventSubscriptionManager, EventType};

type EvResult = Result<StreamFrame, ApiError>;
type Mgr = EventSubscriptionManager<PlatformRuntime>;

/// Scriptable source. `subscribe()` hands back a fresh channel whose sender the
/// test keeps (push events / errors / clean-close at will). The first
/// `fail_until` subscribes return ConnectFailed; the first `hang_until` never
/// resolve (drives connect-timeout). All state lives behind `Arc` so the test
/// retains control while the manager owns an `Arc<dyn EventStreamSource>`.
struct Shared {
    subscribe_count: AtomicU32,
    fail_until: u32,
    hang_until: u32,
    senders: Mutex<Vec<UnboundedSender<EvResult>>>,
}

impl Shared {
    fn new(fail_until: u32, hang_until: u32) -> Arc<Self> {
        Arc::new(Self {
            subscribe_count: AtomicU32::new(0),
            fail_until,
            hang_until,
            senders: Mutex::new(Vec::new()),
        })
    }
    fn count(&self) -> u32 {
        self.subscribe_count.load(SeqCst)
    }
    fn push_event(&self) {
        if let Some(tx) = self.senders.lock().last() {
            let _ = tx.unbounded_send(Ok(StreamFrame::Event(make_event())));
        }
    }
    fn push_keepalive(&self) {
        if let Some(tx) = self.senders.lock().last() {
            let _ = tx.unbounded_send(Ok(StreamFrame::Keepalive));
        }
    }
    fn close_last(&self) {
        self.senders.lock().pop(); // drop sender → stream ends → ServerClosed
    }
}

struct MockSource {
    shared: Arc<Shared>,
}

impl EventStreamSource for MockSource {
    fn subscribe(&self) -> SubscribeFuture {
        let shared = Arc::clone(&self.shared);
        Box::pin(async move {
            let n = shared.subscribe_count.fetch_add(1, SeqCst) + 1;
            if n <= shared.hang_until {
                // Never resolves → exercises the connect-timeout path.
                return futures::future::pending::<Result<BoxEventStream, ApiError>>().await;
            }
            if n <= shared.fail_until {
                return Err(ApiError::Decode(format!("scripted connect fail #{n}")));
            }
            let (tx, rx) = mpsc::unbounded::<EvResult>();
            shared.senders.lock().push(tx);
            Ok(Box::pin(rx) as BoxEventStream)
        })
    }
}

fn make_event() -> RealtimeEvent {
    serde_json::from_value(serde_json::json!({
        "type": EventType::PodCreated.as_str(),
        "category": "entity",
        "organization_id": 1,
        "data": {},
        "timestamp": 1
    }))
    .unwrap()
}

fn opts(initial_ms: u64, idle_ms: u64, connect_ms: u64) -> EventSubscriptionManagerOptions {
    EventSubscriptionManagerOptions {
        initial_reconnect_delay_ms: initial_ms,
        max_reconnect_delay_ms: 20,
        idle_timeout_ms: idle_ms,
        connect_timeout_ms: connect_ms,
    }
}

fn manager(shared: &Arc<Shared>, options: EventSubscriptionManagerOptions) -> Mgr {
    EventSubscriptionManager::with_stream_source(
        PlatformRuntime,
        Arc::new(MockSource { shared: Arc::clone(shared) }),
        options,
    )
}

async fn wait_count(shared: &Shared, n: u32, timeout: Duration) -> bool {
    let start = Instant::now();
    while shared.count() < n {
        if start.elapsed() > timeout {
            return false;
        }
        tokio::time::sleep(Duration::from_millis(5)).await;
    }
    true
}

async fn wait_state(mgr: &Mgr, want: ConnectionState, timeout: Duration) -> bool {
    let start = Instant::now();
    while mgr.get_connection_state().await != want {
        if start.elapsed() > timeout {
            return false;
        }
        tokio::time::sleep(Duration::from_millis(5)).await;
    }
    true
}

#[tokio::test]
async fn recovers_after_would_be_exhaustion() {
    // 15 connect failures — past the old max_attempts=10 that broke the loop
    // into the permanent Disconnected state. The fix must keep retrying.
    let shared = Shared::new(15, 0);
    let mgr = manager(&shared, opts(5, 2000, 200));
    let got = Arc::new(AtomicU32::new(0));
    let g = Arc::clone(&got);
    let h: EventHandler = Arc::new(move |_| {
        g.fetch_add(1, SeqCst);
    });
    mgr.subscribe_all(h).await;
    mgr.connect().await;

    assert!(
        wait_count(&shared, 16, Duration::from_secs(5)).await,
        "loop gave up before the 16th attempt — still has a give-up cap",
    );
    shared.push_event();
    assert!(
        wait_state(&mgr, ConnectionState::Connected, Duration::from_secs(2)).await,
        "did not reach Connected after recovery",
    );
    let start = Instant::now();
    while got.load(SeqCst) == 0 {
        assert!(start.elapsed() < Duration::from_secs(2), "event not delivered post-recovery");
        tokio::time::sleep(Duration::from_millis(5)).await;
    }
    mgr.disconnect().await;
}

#[tokio::test]
async fn connected_only_after_first_event() {
    let shared = Shared::new(0, 0);
    let mgr = manager(&shared, opts(5, 2000, 200));
    mgr.connect().await;

    assert!(wait_count(&shared, 1, Duration::from_secs(2)).await, "never subscribed");
    // Transport up but no event yet: must stay Connecting, not green.
    tokio::time::sleep(Duration::from_millis(40)).await;
    assert_eq!(
        mgr.get_connection_state().await,
        ConnectionState::Connecting,
        "handshake-but-silent stream must not report Connected",
    );
    shared.push_event();
    assert!(
        wait_state(&mgr, ConnectionState::Connected, Duration::from_secs(2)).await,
        "first event must flip the link to Connected",
    );
    mgr.disconnect().await;
}

#[tokio::test]
async fn idle_timeout_triggers_reconnect() {
    // Connect succeeds but no events ever arrive → idle timeout must rebuild.
    let shared = Shared::new(0, 0);
    let mgr = manager(&shared, opts(5, 80, 500));
    mgr.connect().await;

    assert!(wait_count(&shared, 1, Duration::from_secs(2)).await, "never subscribed");
    assert!(
        wait_count(&shared, 2, Duration::from_secs(2)).await,
        "idle stream was not rebuilt",
    );
    mgr.disconnect().await;
}

#[tokio::test]
async fn connect_timeout_then_retry() {
    // First two subscribes hang forever → each must time out and retry; the
    // third resolves and goes Connected.
    let shared = Shared::new(0, 2);
    let mgr = manager(&shared, opts(5, 2000, 60));
    mgr.connect().await;

    assert!(
        wait_count(&shared, 3, Duration::from_secs(3)).await,
        "hung connect was not timed out and retried",
    );
    shared.push_event();
    assert!(
        wait_state(&mgr, ConnectionState::Connected, Duration::from_secs(2)).await,
        "did not connect after the hung attempts timed out",
    );
    mgr.disconnect().await;
}

#[tokio::test]
async fn reconnects_across_clean_close_cycles() {
    // Connect → event → server close, three times. Each clean close resets
    // backoff; the loop must re-establish and re-deliver every cycle.
    let shared = Shared::new(0, 0);
    let mgr = manager(&shared, opts(5, 2000, 200));
    mgr.connect().await;

    for cycle in 1..=3 {
        assert!(
            wait_count(&shared, cycle, Duration::from_secs(2)).await,
            "cycle {cycle}: never re-subscribed",
        );
        shared.push_event();
        assert!(
            wait_state(&mgr, ConnectionState::Connected, Duration::from_secs(2)).await,
            "cycle {cycle}: not Connected",
        );
        shared.close_last();
        assert!(
            wait_state(&mgr, ConnectionState::Reconnecting, Duration::from_secs(2)).await
                || mgr.get_connection_state().await == ConnectionState::Connecting,
            "cycle {cycle}: did not leave Connected after close",
        );
    }
    mgr.disconnect().await;
}

#[tokio::test]
async fn disconnect_is_terminal() {
    let shared = Shared::new(0, 0);
    let mgr = manager(&shared, opts(5, 2000, 200));
    mgr.connect().await;
    assert!(wait_count(&shared, 1, Duration::from_secs(2)).await, "never subscribed");
    shared.push_event();
    assert!(wait_state(&mgr, ConnectionState::Connected, Duration::from_secs(2)).await);

    mgr.disconnect().await;
    assert_eq!(mgr.get_connection_state().await, ConnectionState::Disconnected);

    let count_at_disconnect = shared.count();
    shared.close_last(); // a drop after disconnect must NOT revive the loop
    tokio::time::sleep(Duration::from_millis(120)).await;
    assert_eq!(
        shared.count(),
        count_at_disconnect,
        "disconnect must be terminal — no reconnect after shutdown",
    );
}

#[tokio::test]
async fn nudge_skips_backoff() {
    // 10s backoff: without nudge the next subscribe wouldn't arrive for seconds.
    // nudge() must interrupt the sleep and reconnect promptly (network-regained).
    let shared = Shared::new(0, 0);
    let long_backoff = EventSubscriptionManagerOptions {
        initial_reconnect_delay_ms: 10_000,
        max_reconnect_delay_ms: 30_000,
        idle_timeout_ms: 2000,
        connect_timeout_ms: 500,
    };
    let mgr = manager(&shared, long_backoff);
    mgr.connect().await;
    assert!(wait_count(&shared, 1, Duration::from_secs(2)).await, "never subscribed");

    shared.close_last(); // ServerClosed → reset + ~10s backoff
    tokio::time::sleep(Duration::from_millis(80)).await;
    let before = shared.count();
    mgr.nudge().await;
    assert!(
        wait_count(&shared, before + 1, Duration::from_secs(2)).await,
        "nudge did not interrupt the 10s backoff to reconnect",
    );
    mgr.disconnect().await;
}

#[tokio::test]
async fn keepalive_sentinels_flip_connected_without_dispatch() {
    // backend 订阅建立后发 liveness 哨兵帧(keepalive=true)+ 周期保活。它们让
    // 链路翻 Connected / 刷新 idle,但绝不能进业务 handler——否则每次心跳都
    // bump tick / 污染 AppState。connection_loop 按 StreamFrame::Keepalive 跳过。
    let shared = Shared::new(0, 0);
    let mgr = manager(&shared, opts(5, 2000, 200));
    let got = Arc::new(AtomicU32::new(0));
    let g = Arc::clone(&got);
    let h: EventHandler = Arc::new(move |_| {
        g.fetch_add(1, SeqCst);
    });
    mgr.subscribe_all(h).await;
    mgr.connect().await;
    assert!(wait_count(&shared, 1, Duration::from_secs(2)).await, "never subscribed");

    // keepalive 哨兵:翻 Connected(链路活着),但不是业务 event。
    shared.push_keepalive();
    assert!(
        wait_state(&mgr, ConnectionState::Connected, Duration::from_secs(2)).await,
        "keepalive sentinel must flip the link to Connected",
    );
    shared.push_keepalive();
    tokio::time::sleep(Duration::from_millis(40)).await;
    assert_eq!(got.load(SeqCst), 0, "sentinels must not reach business handlers");

    // 真实 event 才派发。
    shared.push_event();
    let start = Instant::now();
    while got.load(SeqCst) == 0 {
        assert!(start.elapsed() < Duration::from_secs(2), "real event not delivered");
        tokio::time::sleep(Duration::from_millis(5)).await;
    }
    assert_eq!(got.load(SeqCst), 1, "exactly one business event dispatched");
    mgr.disconnect().await;
}
