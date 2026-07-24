use super::*;

#[tokio::test]
async fn acp_snapshot_emits_output_barrier_before_subscribe_ready() {
    let mock = start_mock_relay().await;
    let (pool, _rx) = RelayConnectionPool::new();
    let (callback, output) = make_output_cb();
    let ready_pool = pool.clone();
    let ready_url = mock.url.clone();
    let ready = tokio::spawn(async move {
        ready_pool
            .subscribe_ready("pod-1", "acp-pane", &ready_url, "tok", callback)
            .await
    });
    assert!(wait_transport(&mock).await, "never connected");

    mock.push(
        MsgType::AcpSnapshot,
        serde_json::json!({"session":{}}).to_string().as_bytes(),
    );

    let handle = tokio::time::timeout(Duration::from_secs(3), ready)
        .await
        .expect("ACP baseline did not resolve readiness")
        .expect("subscribe task panicked")
        .expect("subscriber closed before its ACP baseline");
    assert_eq!(handle.subscription_id, "acp-pane");
    assert_eq!(
        *output.lock().unwrap(),
        vec![Vec::<u8>::new()],
        "AcpSnapshot must cross the output barrier before readiness resolves",
    );
}

#[tokio::test]
async fn acp_command_out_and_event_in() {
    let mock = start_mock_relay().await;
    let (pool, _rx) = RelayConnectionPool::new();
    let (cb, _buf) = make_output_cb();
    pool.subscribe("pod-1", "sub-1", &mock.url, "tok", cb).await;
    assert!(wait_transport(&mock).await, "never connected");
    mock.push(
        MsgType::AcpSnapshot,
        serde_json::json!({"session":{}}).to_string().as_bytes(),
    );
    assert!(wait_ready(&pool, "pod-1").await, "never ready");

    let acp_buf = Arc::new(Mutex::new(Vec::<serde_json::Value>::new()));
    let output = acp_buf.clone();
    pool.on_acp_message(
        "pod-1",
        Arc::new(move |_mt, value| output.lock().unwrap().push(value)),
    )
    .await;
    pool.send_acp_command("pod-1", &serde_json::json!({"cmd":"go"}))
        .await
        .unwrap();
    let frame = mock
        .recv(Duration::from_secs(3))
        .await
        .expect("no acp command frame");
    assert_eq!(frame[0], MsgType::AcpCommand as u8);

    mock.push(
        MsgType::AcpEvent,
        serde_json::json!({"event":"started"})
            .to_string()
            .as_bytes(),
    );
    assert!(
        wait_until(
            || !acp_buf.lock().unwrap().is_empty(),
            Duration::from_secs(3),
        )
        .await,
        "ACP event did not reach its listener",
    );
}

#[tokio::test]
async fn acp_snapshot_marks_link_ready() {
    let mock = start_mock_relay().await;
    let (pool, _rx) = RelayConnectionPool::new();
    let (cb, _buf) = make_output_cb();
    pool.subscribe("pod-1", "sub-1", &mock.url, "tok", cb).await;
    assert!(wait_transport(&mock).await, "never connected");
    tokio::time::sleep(Duration::from_millis(100)).await;
    assert_eq!(pool.get_status("pod-1").await, RelayStatus::Connecting);

    mock.push(
        MsgType::AcpSnapshot,
        serde_json::json!({"session":{}}).to_string().as_bytes(),
    );
    assert!(
        wait_ready(&pool, "pod-1").await,
        "AcpSnapshot did not mark the link ready",
    );
}

#[tokio::test]
async fn send_acp_requires_ready() {
    let mock = start_mock_relay().await;
    let (pool, _rx) = RelayConnectionPool::new();
    let (cb, _buf) = make_output_cb();
    pool.subscribe("pod-1", "sub-1", &mock.url, "tok", cb).await;
    assert!(wait_transport(&mock).await, "never connected");

    let result = pool
        .send_acp_command("pod-1", &serde_json::json!({"cmd":"x"}))
        .await;
    assert!(
        result.is_err(),
        "ACP before readiness unexpectedly succeeded"
    );
}

#[tokio::test]
async fn acp_listener_replaces_not_accumulates() {
    let mock = start_mock_relay().await;
    let (pool, _rx) = RelayConnectionPool::new();
    let (cb, _buf) = make_output_cb();
    pool.subscribe("pod-1", "sub-1", &mock.url, "tok", cb).await;
    assert!(wait_transport(&mock).await, "never connected");
    mock.push(
        MsgType::AcpSnapshot,
        serde_json::json!({"session":{}}).to_string().as_bytes(),
    );
    assert!(wait_ready(&pool, "pod-1").await, "never ready");

    let count_a = Arc::new(AtomicUsize::new(0));
    let first = count_a.clone();
    pool.on_acp_message(
        "pod-1",
        Arc::new(move |_mt, _value| {
            first.fetch_add(1, SeqCst);
        }),
    )
    .await;
    let count_b = Arc::new(AtomicUsize::new(0));
    let replacement = count_b.clone();
    pool.on_acp_message(
        "pod-1",
        Arc::new(move |_mt, _value| {
            replacement.fetch_add(1, SeqCst);
        }),
    )
    .await;

    mock.push(
        MsgType::AcpEvent,
        serde_json::json!({"event":"started"})
            .to_string()
            .as_bytes(),
    );
    assert!(
        wait_until(|| count_b.load(SeqCst) == 1, Duration::from_secs(3)).await,
        "replacement ACP listener never fired",
    );
    assert_eq!(count_a.load(SeqCst), 0);
}
