use std::sync::{Arc, Mutex};
use std::time::Duration;

use agentsmesh_transport::runtime::{PlatformRuntime, Runtime};
use tokio::sync::mpsc;

use crate::pool::RelayConnectionPool;
use crate::types::{
    ConnectionState, OutputCallback, RelayStatus, RelayStatusInfo, StatusCallback,
};

fn make_output_cb() -> (OutputCallback, Arc<Mutex<Vec<Vec<u8>>>>) {
    let buf = Arc::new(Mutex::new(Vec::new()));
    let r = buf.clone();
    let cb: OutputCallback = Arc::new(move |data| r.lock().unwrap().push(data));
    (cb, buf)
}

fn rt() -> PlatformRuntime {
    PlatformRuntime
}

// ── ConnectionState::cancel_timers ──

#[tokio::test]
async fn cancel_timers_aborts_all_handles() {
    let r = rt();
    let h1 = r.spawn(Box::pin(async { tokio::time::sleep(Duration::from_secs(3600)).await }));
    let h2 = r.spawn(Box::pin(async { tokio::time::sleep(Duration::from_secs(3600)).await }));
    let h3 = r.spawn(Box::pin(async { tokio::time::sleep(Duration::from_secs(3600)).await }));

    let mut conn = ConnectionState::<PlatformRuntime>::new("ws://relay".into(), "tok".into());
    conn.reconnect_handle = Some(h1);
    conn.disconnect_handle = Some(h2);
    conn.snapshot_handle = Some(h3);

    conn.cancel_timers();

    assert!(conn.reconnect_handle.is_none());
    assert!(conn.disconnect_handle.is_none());
    assert!(conn.snapshot_handle.is_none());
}

#[tokio::test]
async fn cancel_timers_noop_when_no_handles() {
    let mut conn = ConnectionState::<PlatformRuntime>::new("ws://relay".into(), "tok".into());
    conn.cancel_timers();
    assert!(conn.reconnect_handle.is_none());
}

// ── handle_disconnect sets status and clears ws_write_tx ──

#[tokio::test]
async fn handle_disconnect_sets_status_and_clears_ws() {
    let (pool, _rx) = RelayConnectionPool::new();
    let (cb, _) = make_output_cb();

    let (ws_tx, _ws_rx) = mpsc::unbounded_channel();
    {
        let mut inner = pool.inner.write().await;
        let mut conn = ConnectionState::new("ws://relay".into(), "tok".into());
        conn.status = RelayStatus::Connected;
        conn.ws_write_tx = Some(ws_tx);
        conn.subscribers.insert("s1".to_string(), cb);
        inner.connections.insert("pod1".to_string(), conn);
    }

    pool.disconnect("pod1").await;
    assert_eq!(pool.get_status("pod1").await, RelayStatus::Disconnected);
}

#[tokio::test]
async fn handle_disconnect_nonexistent_pod_is_noop() {
    let (pool, _rx) = RelayConnectionPool::new();
    pool.disconnect("no-pod").await;
    assert_eq!(pool.get_status("no-pod").await, RelayStatus::Disconnected);
}

// ── notify_status reads runner_disconnected ──

#[tokio::test]
async fn notify_status_includes_runner_disconnected() {
    let (pool, _rx) = RelayConnectionPool::new();
    let (cb, _) = make_output_cb();

    {
        let mut inner = pool.inner.write().await;
        let mut conn = ConnectionState::new("ws://relay".into(), "tok".into());
        conn.runner_disconnected = true;
        conn.subscribers.insert("s1".to_string(), cb);
        inner.connections.insert("pod1".to_string(), conn);
    }

    let received = Arc::new(Mutex::new(Vec::<RelayStatusInfo>::new()));
    let r = received.clone();
    let listener: StatusCallback = Arc::new(move |info| r.lock().unwrap().push(info));
    pool.on_status_change("pod1", listener).await;

    pool.notify_status("pod1", RelayStatus::Connected).await;

    let infos = received.lock().unwrap();
    assert_eq!(infos.len(), 2);
    assert!(infos[1].runner_disconnected);
    assert_eq!(infos[1].status, RelayStatus::Connected);
}

#[tokio::test]
async fn notify_status_nonexistent_pod_uses_defaults() {
    let (pool, _rx) = RelayConnectionPool::new();
    let received = Arc::new(Mutex::new(Vec::<RelayStatusInfo>::new()));
    let r = received.clone();
    let listener: StatusCallback = Arc::new(move |info| r.lock().unwrap().push(info));
    pool.on_status_change("no-pod", listener).await;

    pool.notify_status("no-pod", RelayStatus::Disconnected).await;

    let infos = received.lock().unwrap();
    assert_eq!(infos.len(), 2);
    assert_eq!(infos[0].status, RelayStatus::Disconnected);
    assert!(!infos[1].runner_disconnected);
}

// ── notify_status_info static method ──

#[test]
fn notify_status_info_with_listeners() {
    let received = Arc::new(Mutex::new(Vec::<RelayStatusInfo>::new()));
    let r = received.clone();
    let cb: StatusCallback = Arc::new(move |info| r.lock().unwrap().push(info));

    let mut listeners = std::collections::HashMap::new();
    listeners.insert("pod1".to_string(), vec![cb]);

    let info = RelayStatusInfo {
        status: RelayStatus::Connected,
        runner_disconnected: false,
    };
    RelayConnectionPool::<PlatformRuntime>::notify_status_info(&listeners, "pod1", &info);

    let infos = received.lock().unwrap();
    assert_eq!(infos.len(), 1);
    assert_eq!(infos[0].status, RelayStatus::Connected);
}

#[test]
fn notify_status_info_no_listeners_is_noop() {
    let listeners = std::collections::HashMap::new();
    let info = RelayStatusInfo {
        status: RelayStatus::Connected,
        runner_disconnected: false,
    };
    RelayConnectionPool::<PlatformRuntime>::notify_status_info(&listeners, "pod1", &info);
}

// ── disconnect_inner cleans up completely ──

#[tokio::test]
async fn disconnect_inner_removes_all_state() {
    let (pool, _rx) = RelayConnectionPool::new();
    let (cb, _) = make_output_cb();

    let (ws_tx, _ws_rx) = mpsc::unbounded_channel();
    let resize_handle =
        rt().spawn(Box::pin(async { tokio::time::sleep(Duration::from_secs(3600)).await }));
    {
        let mut inner = pool.inner.write().await;
        let mut conn = ConnectionState::new("ws://relay".into(), "tok".into());
        conn.ws_write_tx = Some(ws_tx);
        conn.subscribers.insert("s1".to_string(), cb);
        inner.connections.insert("pod1".to_string(), conn);
        inner
            .last_inputs
            .insert("pod1".to_string(), ("x".to_string(), web_time::Instant::now()));
        inner.resize_debounce.insert("pod1".to_string(), resize_handle);
    }

    {
        let mut inner = pool.inner.write().await;
        RelayConnectionPool::disconnect_inner(&mut inner, "pod1");
        assert!(!inner.connections.contains_key("pod1"));
        assert!(!inner.last_inputs.contains_key("pod1"));
        assert!(!inner.resize_debounce.contains_key("pod1"));
    }
}

// ── do_send_resize ──

#[tokio::test]
async fn do_send_resize_sends_to_ws() {
    let (ws_tx, mut ws_rx) = mpsc::unbounded_channel();
    let mut conn = ConnectionState::<PlatformRuntime>::new("ws://relay".into(), "tok".into());
    conn.ws_write_tx = Some(ws_tx);

    RelayConnectionPool::do_send_resize(&conn, 120, 40);

    let msg = ws_rx.recv().await.unwrap();
    assert!(!msg.is_empty());
}

#[test]
fn do_send_resize_no_ws_tx_is_noop() {
    let conn = ConnectionState::<PlatformRuntime>::new("ws://relay".into(), "tok".into());
    RelayConnectionPool::do_send_resize(&conn, 120, 40);
}

// ── connection::connect URL construction (verified through error) ──

#[tokio::test]
async fn connect_to_invalid_url_returns_error() {
    let (msg_tx, _) = mpsc::unbounded_channel();
    let (close_tx, _) = mpsc::unbounded_channel();
    let (error_tx, _) = mpsc::unbounded_channel();

    let result = crate::connection::connect(
        &rt(),
        "ws://invalid-host-that-does-not-exist:0",
        "test-token",
        msg_tx,
        "pod1".to_string(),
        close_tx,
        error_tx,
    )
    .await;

    assert!(result.is_err());
    let err = result.err().unwrap();
    match err {
        crate::error::RelayError::Connection(msg) => {
            assert!(!msg.is_empty());
        }
        other => panic!("expected Connection error, got {other:?}"),
    }
}
