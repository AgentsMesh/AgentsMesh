use std::sync::{Arc, Mutex};

use agentsmesh_protocol::MsgType;
use agentsmesh_transport::runtime::PlatformRuntime;

use super::*;

#[test]
fn bind_listeners_returns_zero_for_missing_pod() {
    let (pool, _rx) = RelayConnectionPool::with_runtime(PlatformRuntime);
    let generation = pool.bind_listeners_if_active(
        "missing",
        "lease",
        Arc::new(|_, _| panic!("missing pod status callback must not fire")),
        Arc::new(|_, _, _| panic!("missing pod ACP callback must not fire")),
    );
    assert_eq!(generation, 0);
}

#[test]
fn active_listener_binding_is_generation_scoped_idempotent_and_rebindable() {
    let (pool, _rx) = RelayConnectionPool::with_runtime(PlatformRuntime);
    let _cmd_rx = pool
        .inner
        .write()
        .insert_test_pod("pod-1", 42, RelayStatus::Connected);
    {
        let handle = pool.inner.read();
        let snapshot = handle.pods.get("pod-1").unwrap().snapshot.clone();
        let mut snapshot = snapshot.write();
        snapshot.runner_disconnected = true;
        snapshot.revision = 5;
    }

    let first_status = Arc::new(Mutex::new(Vec::new()));
    let first_acp = Arc::new(Mutex::new(Vec::new()));
    let status_events = Arc::clone(&first_status);
    let acp_events = Arc::clone(&first_acp);
    let generation = pool.bind_listeners_if_active(
        "pod-1",
        "lease-a",
        Arc::new(move |generation, info| {
            status_events.lock().unwrap().push((generation, info));
        }),
        Arc::new(move |generation, msg_type, value| {
            acp_events
                .lock()
                .unwrap()
                .push((generation, msg_type, value));
        }),
    );
    assert_eq!(generation, 42);
    let status = first_status.lock().unwrap();
    assert_eq!(status.len(), 1);
    assert_eq!(status[0].0, 42);
    assert_eq!(status[0].1.status, RelayStatus::Connected);
    assert!(status[0].1.runner_disconnected);
    assert_eq!(status[0].1.revision, 5);
    drop(status);

    let duplicate_calls = Arc::new(Mutex::new(0usize));
    let calls = Arc::clone(&duplicate_calls);
    assert_eq!(
        pool.bind_listeners_if_active(
            "pod-1",
            "lease-a",
            Arc::new(move |_, _| *calls.lock().unwrap() += 1),
            Arc::new(|_, _, _| {}),
        ),
        42
    );
    assert_eq!(*duplicate_calls.lock().unwrap(), 0);

    let rebound = Arc::new(Mutex::new(Vec::new()));
    let rebound_events = Arc::clone(&rebound);
    assert_eq!(
        pool.bind_listeners_if_active(
            "pod-1",
            "lease-b",
            Arc::new(move |generation, info| {
                rebound_events
                    .lock()
                    .unwrap()
                    .push((generation, info.revision));
            }),
            Arc::new(|_, _, _| {}),
        ),
        42
    );
    assert_eq!(*rebound.lock().unwrap(), vec![(42, 5)]);

    let listener = pool
        .inner
        .read()
        .acp_listeners
        .get("pod-1")
        .cloned()
        .unwrap();
    listener.deliver(MsgType::AcpEvent, serde_json::json!({"event": "ready"}));
    assert!(
        first_acp.lock().unwrap().is_empty(),
        "rebind must replace ACP listener"
    );
}

#[test]
fn disconnect_callbacks_are_registered_independently() {
    let (pool, _rx) = RelayConnectionPool::with_runtime(PlatformRuntime);
    pool.set_on_pod_disconnected(Arc::new(|_| {}));
    pool.set_on_pod_generation_disconnected(Arc::new(|_, _| {}));
    let router = pool.inner.read();
    assert!(router.on_pod_disconnected.is_some());
    assert!(router.on_pod_generation_disconnected.is_some());
}
