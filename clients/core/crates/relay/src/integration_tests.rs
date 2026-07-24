//! End-to-end integration tests: drive the real `RelayConnectionPool` against a
//! mock relay WebSocket server (tokio-tungstenite) through the native transport.
//! Unlike the unit tests (which poke `ConnectionState` directly), these exercise
//! connect → codec → dispatch → callbacks → reconnect over a real socket.
mod acp_lifecycle;
mod connection_lifecycle;
mod listener_rebinding;
mod subscriber_readiness;

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
    text_to_client: UnboundedSender<String>,
    from_client: TokioMutex<UnboundedReceiver<Vec<u8>>>,
    drop_signal: UnboundedSender<()>,
    conn_count: Arc<AtomicUsize>,
}

async fn start_mock_relay() -> MockRelay {
    let listener = tokio::net::TcpListener::bind("127.0.0.1:0").await.unwrap();
    let url = format!("ws://{}", listener.local_addr().unwrap());
    let (to_client, mut to_client_rx) = unbounded_channel::<Vec<u8>>();
    let (text_to_client, mut text_to_client_rx) = unbounded_channel::<String>();
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
    // Deliver server text frames independently from the binary protocol stream.
    {
        let active = active.clone();
        tokio::spawn(async move {
            while let Some(text) = text_to_client_rx.recv().await {
                if let Some(sink) = active.lock().await.as_mut() {
                    let _ = sink.send(Message::Text(text.into())).await;
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
                let Ok((stream, _)) = listener.accept().await else {
                    break;
                };
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
        text_to_client,
        from_client: TokioMutex::new(from_client_rx),
        drop_signal,
        conn_count,
    }
}

impl MockRelay {
    fn push(&self, msg_type: MsgType, payload: &[u8]) {
        self.to_client
            .send(encode_message(msg_type, payload))
            .unwrap();
    }
    fn push_text(&self, text: &str) {
        self.text_to_client.send(text.to_string()).unwrap();
    }
    async fn recv(&self, timeout: Duration) -> Option<Vec<u8>> {
        let mut rx = self.from_client.lock().await;
        tokio::time::timeout(timeout, rx.recv())
            .await
            .ok()
            .flatten()
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

async fn assert_no_client_frame_type(mock: &MockRelay, msg_type: MsgType, window: Duration) {
    let deadline = std::time::Instant::now() + window;
    loop {
        let remaining = deadline.saturating_duration_since(std::time::Instant::now());
        if remaining.is_zero() {
            return;
        }
        let Some(frame) = mock.recv(remaining).await else {
            return;
        };
        assert_ne!(
            frame.first().copied(),
            Some(msg_type as u8),
            "unexpected {msg_type:?} frame before the test barrier"
        );
    }
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

fn buf_contains(buf: &Arc<Mutex<Vec<Vec<u8>>>>, needle: &[u8]) -> bool {
    buf.lock()
        .unwrap()
        .iter()
        .any(|frame| frame.windows(needle.len()).any(|window| window == needle))
}

fn buf_count_contains(buf: &Arc<Mutex<Vec<Vec<u8>>>>, needle: &[u8]) -> usize {
    buf.lock()
        .unwrap()
        .iter()
        .filter(|frame| frame.windows(needle.len()).any(|window| window == needle))
        .count()
}

fn push_snapshot(mock: &MockRelay, content: &str) {
    let snapshot = serde_json::json!({
        "serialized_content": content,
        "reset_all": false,
        "cols": 80,
        "rows": 24,
    });
    mock.push(MsgType::Snapshot, snapshot.to_string().as_bytes());
}

#[tokio::test]
async fn output_frame_reaches_subscriber() {
    let mock = start_mock_relay().await;
    let (pool, _rx) = RelayConnectionPool::new();
    let (cb, buf) = make_output_cb();
    pool.subscribe("pod-1", "sub-1", &mock.url, "tok", cb).await;
    assert!(wait_transport(&mock).await, "never connected");

    push_snapshot(&mock, "BASELINE");
    assert!(wait_ready(&pool, "pod-1").await, "never ready");
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
    let frame = mock
        .recv(Duration::from_secs(3))
        .await
        .expect("no input frame");
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
    push_snapshot(&mock, "RESIZE-BASELINE");
    assert!(wait_ready(&pool, "pod-1").await, "never became ready");

    pool.send_resize("pod-1", 120, 40).await; // debounced ~150ms
    let frame = mock
        .recv(Duration::from_secs(3))
        .await
        .expect("no resize frame");
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

    let snap = serde_json::json!({
        "serialized_content":"RESTORED-STATE",
        "reset_all": false,
        "cols":80,
        "rows":24
    });
    mock.push(MsgType::Snapshot, snap.to_string().as_bytes());
    assert!(
        wait_until(
            || buf_contains(&buf, b"RESTORED-STATE"),
            Duration::from_secs(3)
        )
        .await,
        "snapshot content not replayed to subscriber",
    );
    assert!(
        buf_contains(&buf, crate::dispatch_snapshot::ANSI_CLEAR),
        "snapshot did not clear screen first"
    );
}
