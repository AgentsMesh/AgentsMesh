//! End-to-end integration tests: drive the real `RelayConnectionPool` against a
//! mock relay WebSocket server (tokio-tungstenite) through the native transport.
//! Unlike the unit tests (which poke `ConnectionState` directly), these exercise
//! connect → codec → dispatch → callbacks → reconnect over a real socket.

use std::sync::atomic::{AtomicUsize, Ordering::SeqCst};
use std::sync::{Arc, Mutex};
use std::time::Duration;

use agentsmesh_protocol::{encode_message, MsgType};
use futures_util::stream::SplitSink;
use futures_util::{SinkExt, StreamExt};
use tokio::net::TcpStream;
use tokio::sync::mpsc::{unbounded_channel, UnboundedReceiver, UnboundedSender};
use tokio::sync::Mutex as TokioMutex;
use tokio_tungstenite::tungstenite::Message;
use tokio_tungstenite::WebSocketStream;

use crate::pool::RelayConnectionPool;
use crate::types::{OutputCallback, RelayStatus};

type ServerSink = SplitSink<WebSocketStream<TcpStream>, Message>;

/// Scriptable mock relay. Accepts (re-)connections; `to_client` frames are sent
/// to the current connection, inbound frames land in `from_client`, `drop` closes
/// the active socket (to exercise reconnect). `conn_count` counts accepted sockets.
struct MockRelay {
    url: String,
    to_client: UnboundedSender<Vec<u8>>,
    from_client: TokioMutex<UnboundedReceiver<Vec<u8>>>,
    drop_signal: UnboundedSender<()>,
    conn_count: Arc<AtomicUsize>,
}

async fn start_mock_relay() -> MockRelay {
    let listener = tokio::net::TcpListener::bind("127.0.0.1:0").await.unwrap();
    let url = format!("ws://{}", listener.local_addr().unwrap());
    let (to_client, mut to_client_rx) = unbounded_channel::<Vec<u8>>();
    let (from_client_tx, from_client_rx) = unbounded_channel::<Vec<u8>>();
    let (drop_signal, mut drop_rx) = unbounded_channel::<()>();
    let conn_count = Arc::new(AtomicUsize::new(0));

    let active: Arc<TokioMutex<Option<ServerSink>>> = Arc::new(TokioMutex::new(None));

    // Drain test → current socket.
    {
        let active = active.clone();
        tokio::spawn(async move {
            while let Some(frame) = to_client_rx.recv().await {
                if let Some(sink) = active.lock().await.as_mut() {
                    let _ = sink.send(Message::Binary(frame.into())).await;
                }
            }
        });
    }
    // Drop signal → close current socket.
    {
        let active = active.clone();
        tokio::spawn(async move {
            while drop_rx.recv().await.is_some() {
                if let Some(sink) = active.lock().await.as_mut() {
                    let _ = sink.send(Message::Close(None)).await;
                }
            }
        });
    }
    // Accept loop: swap active sink BEFORE bumping conn_count so a >=N observation
    // guarantees the sink is ready (avoids a send-before-swap race in reconnect).
    {
        let active = active.clone();
        let cc = conn_count.clone();
        tokio::spawn(async move {
            loop {
                let Ok((stream, _)) = listener.accept().await else { break };
                let ws = match tokio_tungstenite::accept_async(stream).await {
                    Ok(w) => w,
                    Err(_) => continue,
                };
                let (write, mut read) = ws.split();
                *active.lock().await = Some(write);
                cc.fetch_add(1, SeqCst);
                let ftx = from_client_tx.clone();
                tokio::spawn(async move {
                    while let Some(Ok(msg)) = read.next().await {
                        if let Message::Binary(b) = msg {
                            let _ = ftx.send(b.to_vec());
                        }
                    }
                });
            }
        });
    }

    MockRelay {
        url,
        to_client,
        from_client: TokioMutex::new(from_client_rx),
        drop_signal,
        conn_count,
    }
}

impl MockRelay {
    fn push(&self, msg_type: MsgType, payload: &[u8]) {
        self.to_client.send(encode_message(msg_type, payload)).unwrap();
    }
    async fn recv(&self, timeout: Duration) -> Option<Vec<u8>> {
        let mut rx = self.from_client.lock().await;
        tokio::time::timeout(timeout, rx.recv()).await.ok().flatten()
    }
}

fn make_output_cb() -> (OutputCallback, Arc<Mutex<Vec<Vec<u8>>>>) {
    let buf = Arc::new(Mutex::new(Vec::new()));
    let r = buf.clone();
    let cb: OutputCallback = Arc::new(move |data| r.lock().unwrap().push(data));
    (cb, buf)
}

async fn wait_until<F: Fn() -> bool>(f: F, timeout: Duration) -> bool {
    let start = std::time::Instant::now();
    while !f() {
        if start.elapsed() > timeout {
            return false;
        }
        tokio::time::sleep(Duration::from_millis(10)).await;
    }
    true
}

// Transport-up = the mock accepted a socket. Distinct from "Connected", which
// post-2b means data-ready (snapshot received), so tests that never push a
// snapshot wait on this instead of on the green light.
async fn wait_transport(mock: &MockRelay) -> bool {
    wait_until(|| mock.conn_count.load(SeqCst) >= 1, Duration::from_secs(3)).await
}

// Wait for the data-ready Connected state (snapshot received) — required before
// ACP commands (#4) and asserted by the Connected-semantics tests.
async fn wait_ready(pool: &RelayConnectionPool, pod: &str) -> bool {
    let start = std::time::Instant::now();
    while pool.get_status(pod).await != RelayStatus::Connected {
        if start.elapsed() > Duration::from_secs(3) {
            return false;
        }
        tokio::time::sleep(Duration::from_millis(10)).await;
    }
    true
}

fn buf_has(buf: &Arc<Mutex<Vec<Vec<u8>>>>, needle: &[u8]) -> bool {
    buf.lock().unwrap().iter().any(|f| f == needle)
}

#[tokio::test]
async fn output_frame_reaches_subscriber() {
    let mock = start_mock_relay().await;
    let (pool, _rx) = RelayConnectionPool::new();
    let (cb, buf) = make_output_cb();
    pool.subscribe("pod-1", "sub-1", &mock.url, "tok", cb).await;
    assert!(wait_transport(&mock).await, "never connected");

    mock.push(MsgType::Output, b"hello-terminal");
    assert!(
        wait_until(|| buf_has(&buf, b"hello-terminal"), Duration::from_secs(3)).await,
        "output frame did not reach subscriber callback",
    );
}

#[tokio::test]
async fn input_send_reaches_server() {
    let mock = start_mock_relay().await;
    let (pool, _rx) = RelayConnectionPool::new();
    let (cb, _buf) = make_output_cb();
    pool.subscribe("pod-1", "sub-1", &mock.url, "tok", cb).await;
    assert!(wait_transport(&mock).await, "never connected");

    pool.send("pod-1", "echo-me").await;
    let frame = mock.recv(Duration::from_secs(3)).await.expect("no input frame");
    assert_eq!(frame[0], MsgType::Input as u8, "wrong frame type");
    assert_eq!(&frame[1..], b"echo-me");
}

#[tokio::test]
async fn resize_debounced_reaches_server() {
    let mock = start_mock_relay().await;
    let (pool, _rx) = RelayConnectionPool::new();
    let (cb, _buf) = make_output_cb();
    pool.subscribe("pod-1", "sub-1", &mock.url, "tok", cb).await;
    assert!(wait_transport(&mock).await, "never connected");

    pool.send_resize("pod-1", 120, 40).await; // debounced ~150ms
    let frame = mock.recv(Duration::from_secs(3)).await.expect("no resize frame");
    assert_eq!(frame[0], MsgType::Resize as u8, "wrong frame type");
    // 4-byte big-endian cols,rows payload
    assert_eq!(&frame[1..], &[0, 120, 0, 40]);
}

#[tokio::test]
async fn snapshot_replays_to_subscriber() {
    let mock = start_mock_relay().await;
    let (pool, _rx) = RelayConnectionPool::new();
    let (cb, buf) = make_output_cb();
    pool.subscribe("pod-1", "sub-1", &mock.url, "tok", cb).await;
    assert!(wait_transport(&mock).await, "never connected");

    let snap = serde_json::json!({"serialized_content":"RESTORED-STATE","cols":80,"rows":24});
    mock.push(MsgType::Snapshot, snap.to_string().as_bytes());
    assert!(
        wait_until(|| buf_has(&buf, b"RESTORED-STATE"), Duration::from_secs(3)).await,
        "snapshot content not replayed to subscriber",
    );
    assert!(buf_has(&buf, crate::dispatch::ANSI_CLEAR), "snapshot did not clear screen first");
}

#[tokio::test]
async fn runner_disconnect_then_reconnect_toggles_flag() {
    let mock = start_mock_relay().await;
    let (pool, _rx) = RelayConnectionPool::new();
    let (cb, _buf) = make_output_cb();
    pool.subscribe("pod-1", "sub-1", &mock.url, "tok", cb).await;
    assert!(wait_transport(&mock).await, "never connected");

    mock.push(MsgType::RunnerDisconnected, &[]);
    let start = std::time::Instant::now();
    while !pool.is_runner_disconnected("pod-1").await {
        assert!(start.elapsed() < Duration::from_secs(3), "runner_disconnected never set");
        tokio::time::sleep(Duration::from_millis(10)).await;
    }
    mock.push(MsgType::RunnerReconnected, &[]);
    let start = std::time::Instant::now();
    while pool.is_runner_disconnected("pod-1").await {
        assert!(start.elapsed() < Duration::from_secs(3), "runner_disconnected never cleared");
        tokio::time::sleep(Duration::from_millis(10)).await;
    }
}

#[tokio::test]
async fn acp_command_out_and_event_in() {
    let mock = start_mock_relay().await;
    let (pool, _rx) = RelayConnectionPool::new();
    let (cb, _buf) = make_output_cb();
    pool.subscribe("pod-1", "sub-1", &mock.url, "tok", cb).await;
    assert!(wait_transport(&mock).await, "never connected");
    // ACP requires a data-ready link (#4). An ACP pod's data-ready signal is
    // AcpSnapshot — it never sends the PTY MsgType::Snapshot — so drive Connected
    // through the realistic frame type (pushing PTY Snapshot here would mask the
    // ACP-stays-Connecting regression).
    mock.push(MsgType::AcpSnapshot, serde_json::json!({"session":{}}).to_string().as_bytes());
    assert!(wait_ready(&pool, "pod-1").await, "never ready");

    let acp_buf = Arc::new(Mutex::new(Vec::<serde_json::Value>::new()));
    {
        let b = acp_buf.clone();
        pool.on_acp_message(
            "pod-1",
            Arc::new(move |_mt, val| b.lock().unwrap().push(val)),
        )
        .await;
    }

    pool.send_acp_command("pod-1", &serde_json::json!({"cmd":"go"}))
        .await
        .unwrap();
    let frame = mock.recv(Duration::from_secs(3)).await.expect("no acp command frame");
    assert_eq!(frame[0], MsgType::AcpCommand as u8, "wrong frame type");

    mock.push(MsgType::AcpEvent, serde_json::json!({"event":"started"}).to_string().as_bytes());
    assert!(
        wait_until(|| !acp_buf.lock().unwrap().is_empty(), Duration::from_secs(3)).await,
        "acp event did not reach on_acp_message callback",
    );
}

#[tokio::test]
async fn reconnects_after_server_drop() {
    let mock = start_mock_relay().await;
    let (pool, _rx) = RelayConnectionPool::new();
    let (cb, buf) = make_output_cb();
    pool.subscribe("pod-1", "sub-1", &mock.url, "tok", cb).await;
    assert!(wait_until(|| mock.conn_count.load(SeqCst) >= 1, Duration::from_secs(10)).await, "no first connect");

    mock.drop_signal.send(()).unwrap();
    // schedule_reconnect waits ~BASE_RECONNECT_DELAY_MS (1s) before re-dialing.
    // Timeouts are generous: under the test binary's full parallelism (one
    // tokio runtime per #[tokio::test] thread) the reconnect's wall clock can
    // stretch well past the ~1s backoff. This asserts the behavior, not an SLA.
    assert!(
        wait_until(|| mock.conn_count.load(SeqCst) >= 2, Duration::from_secs(20)).await,
        "pool did not reconnect after server drop",
    );

    mock.push(MsgType::Output, b"after-reconnect");
    assert!(
        wait_until(|| buf_has(&buf, b"after-reconnect"), Duration::from_secs(10)).await,
        "output did not flow after reconnect",
    );
}

#[tokio::test]
async fn snapshot_resync_keeps_retrying_past_old_cap() {
    let mock = start_mock_relay().await;
    let (pool, _rx) = RelayConnectionPool::new();
    let (cb, _buf) = make_output_cb();
    pool.subscribe("pod-1", "sub-1", &mock.url, "tok", cb).await;
    assert!(wait_transport(&mock).await, "never connected");

    // Never deliver a Snapshot. The client must keep re-requesting (Resync) on
    // the SNAPSHOT_TIMEOUT_MS cadence — well past the old 3-attempt cap — rather
    // than give up and sit Connected-but-blank. Collecting >=4 proves keepalive.
    let mut resync_count = 0;
    let deadline = std::time::Instant::now() + Duration::from_secs(15);
    while resync_count < 4 && std::time::Instant::now() < deadline {
        if let Some(frame) = mock.recv(Duration::from_secs(4)).await {
            if frame[0] == MsgType::Resync as u8 {
                resync_count += 1;
            }
        }
    }
    assert!(
        resync_count >= 4,
        "expected sustained Resync keepalive past the old 3-cap, got {resync_count}",
    );
}

#[tokio::test]
async fn connected_reported_only_after_snapshot() {
    let mock = start_mock_relay().await;
    let (pool, _rx) = RelayConnectionPool::new();
    let (cb, _buf) = make_output_cb();
    pool.subscribe("pod-1", "sub-1", &mock.url, "tok", cb).await;
    assert!(wait_transport(&mock).await, "transport never connected");

    // Transport up but no snapshot yet: status must NOT be Connected (green), so
    // the connection light can't show green-but-blank.
    tokio::time::sleep(Duration::from_millis(200)).await;
    assert_eq!(
        pool.get_status("pod-1").await,
        RelayStatus::Connecting,
        "must report Connecting (not stale Disconnected, not premature Connected) before snapshot",
    );

    // Snapshot arrives → data-ready → Connected (green).
    mock.push(
        MsgType::Snapshot,
        serde_json::json!({"serialized_content":"x","cols":80,"rows":24})
            .to_string()
            .as_bytes(),
    );
    let start = std::time::Instant::now();
    while pool.get_status("pod-1").await != RelayStatus::Connected {
        assert!(
            start.elapsed() < Duration::from_secs(3),
            "Connected never reported after snapshot",
        );
        tokio::time::sleep(Duration::from_millis(10)).await;
    }
}

async fn wait_until_pod_size(pool: &RelayConnectionPool, pod: &str, want: (u16, u16)) -> bool {
    let start = std::time::Instant::now();
    while pool.get_pod_size(pod).await != Some(want) {
        if start.elapsed() > Duration::from_secs(3) {
            return false;
        }
        tokio::time::sleep(Duration::from_millis(10)).await;
    }
    true
}

#[tokio::test]
async fn acp_snapshot_marks_link_ready() {
    let mock = start_mock_relay().await;
    let (pool, _rx) = RelayConnectionPool::new();
    let (cb, _buf) = make_output_cb();
    pool.subscribe("pod-1", "sub-1", &mock.url, "tok", cb).await;
    assert!(wait_transport(&mock).await, "never connected");

    // An ACP pod reaches data-ready via AcpSnapshot, not the PTY Snapshot frame.
    // Regression: when only the Snapshot arm flipped Connected, ACP pods sat
    // Connecting forever and every send_acp_command was rejected.
    tokio::time::sleep(Duration::from_millis(100)).await;
    assert_eq!(
        pool.get_status("pod-1").await,
        RelayStatus::Connecting,
        "ACP link must not be Connected before its snapshot",
    );
    mock.push(MsgType::AcpSnapshot, serde_json::json!({"session":{}}).to_string().as_bytes());
    assert!(
        wait_ready(&pool, "pod-1").await,
        "AcpSnapshot must drive the link to Connected (data-ready)",
    );
}

#[tokio::test]
async fn send_acp_requires_ready() {
    let mock = start_mock_relay().await;
    let (pool, _rx) = RelayConnectionPool::new();
    let (cb, _buf) = make_output_cb();
    pool.subscribe("pod-1", "sub-1", &mock.url, "tok", cb).await;
    assert!(wait_transport(&mock).await, "never connected");

    // Transport up but not yet Ready (no snapshot): ACP must be rejected with an
    // error, not return Ok while the command is silently dropped (#4).
    let res = pool
        .send_acp_command("pod-1", &serde_json::json!({"cmd":"x"}))
        .await;
    assert!(res.is_err(), "ACP before Ready must return Err, got {res:?}");
}

#[tokio::test]
async fn input_dedup_within_window() {
    let mock = start_mock_relay().await;
    let (pool, _rx) = RelayConnectionPool::new();
    let (cb, _buf) = make_output_cb();
    pool.subscribe("pod-1", "sub-1", &mock.url, "tok", cb).await;
    assert!(wait_transport(&mock).await, "never connected");

    pool.send("pod-1", "dupe").await;
    pool.send("pod-1", "dupe").await; // identical, within 50ms → deduped
    let f1 = mock.recv(Duration::from_secs(2)).await.expect("first input");
    assert_eq!(f1[0], MsgType::Input as u8);
    assert_eq!(&f1[1..], b"dupe");
    let f2 = mock.recv(Duration::from_millis(300)).await;
    assert!(
        f2.map_or(true, |f| f[0] != MsgType::Input as u8),
        "identical input within the dedup window must not reach the server twice",
    );
}

#[tokio::test]
async fn force_resize_sends_immediately() {
    let mock = start_mock_relay().await;
    let (pool, _rx) = RelayConnectionPool::new();
    let (cb, _buf) = make_output_cb();
    pool.subscribe("pod-1", "sub-1", &mock.url, "tok", cb).await;
    assert!(wait_transport(&mock).await, "never connected");

    pool.force_resize("pod-1", 100, 30).await; // bypasses the 150ms debounce
    let frame = mock
        .recv(Duration::from_millis(500))
        .await
        .expect("no resize frame");
    assert_eq!(frame[0], MsgType::Resize as u8);
    assert_eq!(&frame[1..], &[0, 100, 0, 30]);
}

#[tokio::test]
async fn snapshot_updates_pod_size_when_already_connected() {
    let mock = start_mock_relay().await;
    let (pool, _rx) = RelayConnectionPool::new();
    let (cb, _buf) = make_output_cb();
    pool.subscribe("pod-1", "sub-1", &mock.url, "tok", cb).await;
    assert!(wait_transport(&mock).await, "never connected");

    mock.push(
        MsgType::Snapshot,
        serde_json::json!({"serialized_content":"a","cols":80,"rows":24})
            .to_string()
            .as_bytes(),
    );
    assert!(wait_ready(&pool, "pod-1").await, "never ready");
    assert_eq!(pool.get_pod_size("pod-1").await, Some((80, 24)));

    // A second snapshot while ALREADY Connected must still flush the new size to
    // the pool-readable mirror (#3 — set_status short-circuits, so write_snapshot
    // must run explicitly).
    mock.push(
        MsgType::Snapshot,
        serde_json::json!({"serialized_content":"b","cols":120,"rows":40})
            .to_string()
            .as_bytes(),
    );
    assert!(
        wait_until_pod_size(&pool, "pod-1", (120, 40)).await,
        "re-snapshot did not update pod_size mirror while Connected",
    );
}

#[tokio::test]
async fn disconnect_with_subscriber_tears_down() {
    let mock = start_mock_relay().await;
    let (pool, _rx) = RelayConnectionPool::new();
    let (cb, _buf) = make_output_cb();
    pool.subscribe("pod-1", "sub-1", &mock.url, "tok", cb).await;
    assert!(wait_transport(&mock).await, "never connected");
    let conns_before = mock.conn_count.load(SeqCst);

    // Explicit disconnect must tear down even with a subscriber still registered —
    // it must NOT revive/reconnect (the try_finalize-vs-Disconnect bug).
    pool.disconnect("pod-1").await;
    let start = std::time::Instant::now();
    while pool.get_status("pod-1").await != RelayStatus::Disconnected {
        assert!(
            start.elapsed() < Duration::from_secs(3),
            "disconnect did not tear the pod down",
        );
        tokio::time::sleep(Duration::from_millis(10)).await;
    }
    tokio::time::sleep(Duration::from_millis(200)).await;
    assert_eq!(
        mock.conn_count.load(SeqCst),
        conns_before,
        "disconnect wrongly revived and reconnected the link",
    );
}

// Single-value listener map (Phase 3): a re-registration for the same pod REPLACES
// the previous listener rather than accumulating, so a status transition fires the
// pod's one listener exactly once — the desync a Vec-append could reintroduce.
#[tokio::test]
async fn status_listener_replaces_not_accumulates() {
    let mock = start_mock_relay().await;
    let (pool, _rx) = RelayConnectionPool::new();
    let (cb, _buf) = make_output_cb();
    pool.subscribe("pod-1", "sub-1", &mock.url, "tok", cb).await;
    assert!(wait_transport(&mock).await, "never connected");

    let count_a = Arc::new(AtomicUsize::new(0));
    {
        let c = count_a.clone();
        pool.on_status_change("pod-1", Arc::new(move |_info| {
            c.fetch_add(1, SeqCst);
        }))
        .await;
    }
    let count_b = Arc::new(AtomicUsize::new(0));
    {
        let c = count_b.clone();
        pool.on_status_change("pod-1", Arc::new(move |_info| {
            c.fetch_add(1, SeqCst);
        }))
        .await;
    }
    let a_after_replace = count_a.load(SeqCst);

    // Drive a transition (snapshot → Connected) so notify_status fires.
    let snap = serde_json::json!({"serialized_content":"X","cols":80,"rows":24});
    mock.push(MsgType::Snapshot, snap.to_string().as_bytes());
    assert!(wait_ready(&pool, "pod-1").await, "never reached Connected");
    assert!(
        wait_until(|| count_b.load(SeqCst) > 1, Duration::from_secs(3)).await,
        "replacement listener B never fired on the transition",
    );
    assert_eq!(
        count_a.load(SeqCst),
        a_after_replace,
        "replaced listener A still fired — Vec-accumulate regression",
    );
}

// Same single-value guarantee for ACP listeners.
#[tokio::test]
async fn acp_listener_replaces_not_accumulates() {
    let mock = start_mock_relay().await;
    let (pool, _rx) = RelayConnectionPool::new();
    let (cb, _buf) = make_output_cb();
    pool.subscribe("pod-1", "sub-1", &mock.url, "tok", cb).await;
    assert!(wait_transport(&mock).await, "never connected");
    mock.push(MsgType::AcpSnapshot, serde_json::json!({"session":{}}).to_string().as_bytes());
    assert!(wait_ready(&pool, "pod-1").await, "never ready");

    let count_a = Arc::new(AtomicUsize::new(0));
    {
        let c = count_a.clone();
        pool.on_acp_message("pod-1", Arc::new(move |_mt, _val| {
            c.fetch_add(1, SeqCst);
        }))
        .await;
    }
    let count_b = Arc::new(AtomicUsize::new(0));
    {
        let c = count_b.clone();
        pool.on_acp_message("pod-1", Arc::new(move |_mt, _val| {
            c.fetch_add(1, SeqCst);
        }))
        .await;
    }

    mock.push(MsgType::AcpEvent, serde_json::json!({"event":"started"}).to_string().as_bytes());
    assert!(
        wait_until(|| count_b.load(SeqCst) >= 1, Duration::from_secs(3)).await,
        "replacement ACP listener B never fired",
    );
    assert_eq!(count_a.load(SeqCst), 0, "replaced ACP listener A still fired");
}
