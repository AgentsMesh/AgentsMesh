use std::collections::HashMap;
use std::sync::Arc;

use agentsmesh_transport::runtime::PlatformRuntime;
use agentsmesh_transport::WebSocketConnection;
use parking_lot::RwLock;

use super::*;
use crate::pool::PoolRouter;
use crate::types::{RelayStatus, StatusSnapshot};

fn driver() -> Driver<PlatformRuntime> {
    Driver {
        runtime: PlatformRuntime,
        router: Arc::new(RwLock::new(PoolRouter {
            pods: HashMap::new(),
            next_generation: 0,
            status_listeners: HashMap::new(),
            acp_listeners: HashMap::new(),
            on_pod_disconnected: None,
            on_pod_generation_disconnected: None,
        })),
        pod_key: "pod-1".to_string(),
        generation: 1,
        relay_url: "ws://127.0.0.1:1".to_string(),
        relay_token: "token".to_string(),
        snapshot: Arc::new(RwLock::new(StatusSnapshot::default())),
        status: RelayStatus::Connected,
        snapshot_received: true,
        reconnect_attempts: 0,
        runner_disconnected: false,
        pod_size: None,
        subscribers: HashMap::new(),
        last_input: None,
        pending_resize: None,
        grace_deadline: None,
    }
}

async fn connected_transport() -> (WsSender, WsReceiver, tokio::task::JoinHandle<()>) {
    let listener = tokio::net::TcpListener::bind("127.0.0.1:0")
        .await
        .expect("bind session test relay");
    let url = format!("ws://{}", listener.local_addr().unwrap());
    let server = tokio::spawn(async move {
        let (stream, _) = listener.accept().await.expect("accept session test client");
        let websocket = tokio_tungstenite::accept_async(stream)
            .await
            .expect("complete session test handshake");
        futures::future::pending::<()>().await;
        drop(websocket);
    });
    let connection = WebSocketConnection::connect(&url)
        .await
        .expect("connect session test transport");
    let (sender, receiver) = connection.into_split();
    (sender, receiver, server)
}

#[test]
fn force_resize_and_expired_grace_are_immediate_timers() {
    let mut state = driver();
    state.pending_resize = Some((100, 30, Instant::now(), true));
    assert_eq!(state.next_timer(Instant::now()), Duration::ZERO);

    state.pending_resize = None;
    state.grace_deadline = Some(Instant::now());
    assert_eq!(state.next_timer(Instant::now()), Duration::ZERO);
}

#[tokio::test]
async fn expired_empty_grace_shuts_down_the_session() {
    let mut state = driver();
    state.grace_deadline = Some(Instant::now());
    let (sender, mut receiver, server) = connected_transport().await;
    let (cmd_tx, mut cmd_rx) = mpsc::unbounded();

    let end = tokio::time::timeout(
        Duration::from_secs(1),
        state.run_session(&sender, &mut receiver, &mut cmd_rx),
    )
    .await
    .expect("expired grace must be handled without a wall-clock wait");

    assert!(matches!(end, SessionEnd::Shutdown));
    assert!(state.grace_deadline.is_none());
    drop(cmd_tx);
    server.abort();
}

#[tokio::test]
async fn closed_command_channel_shuts_down_the_session() {
    let mut state = driver();
    let (sender, mut receiver, server) = connected_transport().await;
    let (cmd_tx, mut cmd_rx) = mpsc::unbounded();
    drop(cmd_tx);

    let end = tokio::time::timeout(
        Duration::from_secs(1),
        state.run_session(&sender, &mut receiver, &mut cmd_rx),
    )
    .await
    .expect("closed command channel must stop the session immediately");

    assert!(matches!(end, SessionEnd::Shutdown));
    server.abort();
}
