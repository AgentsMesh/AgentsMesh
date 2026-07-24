use std::collections::HashMap;
use std::sync::{Arc, Mutex};

use agentsmesh_transport::runtime::PlatformRuntime;
use futures::channel::{mpsc, oneshot};
use parking_lot::RwLock;

use super::*;
use crate::pool::PoolRouter;
use crate::types::{OutputCallback, StatusSnapshot};

fn callback() -> OutputCallback {
    Arc::new(|_| {})
}

fn router() -> Arc<RwLock<PoolRouter>> {
    Arc::new(RwLock::new(PoolRouter {
        pods: HashMap::new(),
        next_generation: 0,
        status_listeners: HashMap::new(),
        acp_listeners: HashMap::new(),
        on_pod_disconnected: None,
        on_pod_generation_disconnected: None,
    }))
}

fn driver(
    router: Arc<RwLock<PoolRouter>>,
    generation: u32,
    with_subscriber: bool,
) -> Driver<PlatformRuntime> {
    let mut subscribers = HashMap::new();
    if with_subscriber {
        subscribers.insert("sub-1".to_string(), Subscriber::pending(callback(), None));
    }
    Driver {
        runtime: PlatformRuntime,
        router,
        pod_key: "pod-1".to_string(),
        generation,
        relay_url: "ws://127.0.0.1:1".to_string(),
        relay_token: "token".to_string(),
        snapshot: Arc::new(RwLock::new(StatusSnapshot::default())),
        status: RelayStatus::Connecting,
        snapshot_received: false,
        reconnect_attempts: 3,
        runner_disconnected: false,
        pod_size: None,
        subscribers,
        last_input: None,
        pending_resize: None,
        grace_deadline: None,
    }
}

#[tokio::test]
async fn backoff_stops_without_subscribers_or_when_command_channel_closes() {
    let mut no_subscribers = driver(router(), 1, false);
    let (_tx, mut rx) = mpsc::unbounded();
    assert!(matches!(no_subscribers.backoff(&mut rx).await, Flow::Stop));

    let mut closed_channel = driver(router(), 1, true);
    let (tx, mut rx) = mpsc::unbounded();
    drop(tx);
    assert!(matches!(closed_channel.backoff(&mut rx).await, Flow::Stop));
}

#[tokio::test]
async fn backoff_processes_add_remove_resize_and_disconnect_commands() {
    let mut lifecycle = driver(router(), 1, true);
    let (tx, mut rx) = mpsc::unbounded();
    let (ready_tx, ready_rx) = oneshot::channel();
    tx.unbounded_send(Command::AddSubscriber {
        sub_id: "sub-2".to_string(),
        cb: callback(),
        ready: Some(ready_tx),
    })
    .unwrap();
    tx.unbounded_send(Command::RemoveSubscriber {
        sub_id: "sub-1".to_string(),
    })
    .unwrap();
    tx.unbounded_send(Command::Resize {
        cols: 120,
        rows: 40,
        force: true,
    })
    .unwrap();
    tx.unbounded_send(Command::Send {
        data: "dropped-offline".to_string(),
    })
    .unwrap();
    tx.unbounded_send(Command::RemoveSubscriber {
        sub_id: "sub-2".to_string(),
    })
    .unwrap();

    assert!(matches!(lifecycle.backoff(&mut rx).await, Flow::Stop));
    assert_eq!(
        lifecycle.pending_resize.map(|v| (v.0, v.1, v.3)),
        Some((120, 40, true))
    );
    assert!(
        ready_rx.await.is_err(),
        "removed subscriber must cancel readiness"
    );

    let mut disconnected = driver(router(), 1, true);
    let (tx, mut rx) = mpsc::unbounded();
    tx.unbounded_send(Command::Resize {
        cols: 0,
        rows: 0,
        force: false,
    })
    .unwrap();
    tx.unbounded_send(Command::Disconnect).unwrap();
    assert!(matches!(disconnected.backoff(&mut rx).await, Flow::Stop));
    assert!(disconnected.subscribers.is_empty());
    assert!(disconnected.pending_resize.is_none());
}

#[test]
fn finalize_revives_for_queued_subscriber_and_drains_other_commands() {
    let mut lifecycle = driver(router(), 1, false);
    let (tx, mut rx) = mpsc::unbounded();
    tx.unbounded_send(Command::Resize {
        cols: 90,
        rows: 30,
        force: false,
    })
    .unwrap();
    tx.unbounded_send(Command::AddSubscriber {
        sub_id: "late".to_string(),
        cb: callback(),
        ready: None,
    })
    .unwrap();

    assert!(!lifecycle.try_finalize(&mut rx));
    assert!(lifecycle.subscribers.contains_key("late"));
    assert_eq!(lifecycle.reconnect_attempts, 0);
}

#[test]
fn finalize_disconnect_wins_and_owned_generation_fires_both_callbacks() {
    let router = router();
    let _pod_rx = router
        .write()
        .insert_test_pod("pod-1", 7, RelayStatus::Connected);
    let plain = Arc::new(Mutex::new(Vec::new()));
    let generated = Arc::new(Mutex::new(Vec::new()));
    {
        let mut state = router.write();
        let plain_events = Arc::clone(&plain);
        state.on_pod_disconnected = Some(Arc::new(move |pod| {
            plain_events.lock().unwrap().push(pod);
        }));
        let generation_events = Arc::clone(&generated);
        state.on_pod_generation_disconnected = Some(Arc::new(move |pod, generation| {
            generation_events.lock().unwrap().push((pod, generation));
        }));
    }
    let mut lifecycle = driver(Arc::clone(&router), 7, true);
    let (tx, mut rx) = mpsc::unbounded();
    tx.unbounded_send(Command::RemoveSubscriber {
        sub_id: "missing".to_string(),
    })
    .unwrap();
    tx.unbounded_send(Command::Disconnect).unwrap();

    assert!(lifecycle.try_finalize(&mut rx));
    assert!(!router.read().pods.contains_key("pod-1"));
    assert_eq!(*plain.lock().unwrap(), vec!["pod-1".to_string()]);
    assert_eq!(*generated.lock().unwrap(), vec![("pod-1".to_string(), 7)]);
}

#[test]
fn stale_generation_cannot_remove_replacement_driver() {
    let router = router();
    let _replacement_rx = router
        .write()
        .insert_test_pod("pod-1", 9, RelayStatus::Connecting);
    let mut stale = driver(Arc::clone(&router), 8, false);
    let (_tx, mut rx) = mpsc::unbounded();

    assert!(stale.try_finalize(&mut rx));
    assert_eq!(router.read().pods.get("pod-1").unwrap().generation, 9);
}
