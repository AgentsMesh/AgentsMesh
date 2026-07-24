use std::time::Duration;

use agentsmesh_protocol::MsgType;

use super::*;

#[tokio::test]
async fn empty_pty_baseline_emits_output_barrier_before_subscribe_ready() {
    let mock = start_mock_relay().await;
    let (pool, _rx) = RelayConnectionPool::new();
    let (callback, output) = make_output_cb();
    let ready_pool = pool.clone();
    let ready_url = mock.url.clone();
    let ready = tokio::spawn(async move {
        ready_pool
            .subscribe_ready("pod-1", "empty-pty", &ready_url, "tok", callback)
            .await
    });
    assert!(wait_transport(&mock).await, "never connected");

    let snapshot = serde_json::json!({
        "reset_all": false,
        "cols": 80,
        "rows": 24,
    });
    mock.push(MsgType::Snapshot, snapshot.to_string().as_bytes());

    let handle = tokio::time::timeout(Duration::from_secs(3), ready)
        .await
        .expect("empty PTY baseline did not resolve readiness")
        .expect("subscribe task panicked")
        .expect("subscriber closed before its empty PTY baseline");
    assert_eq!(handle.subscription_id, "empty-pty");
    assert_eq!(
        *output.lock().unwrap(),
        vec![Vec::<u8>::new()],
        "the empty output barrier must be observed before readiness resolves",
    );
}

#[tokio::test]
async fn fresh_subscriber_gets_private_baseline_before_incremental_output() {
    let mock = start_mock_relay().await;
    let (pool, _rx) = RelayConnectionPool::new();
    let (cb1, buf1) = make_output_cb();
    pool.subscribe("pod-1", "sub-1", &mock.url, "tok", cb1)
        .await;
    assert!(wait_transport(&mock).await, "never connected");
    push_snapshot(&mock, "FIRST-BASELINE");
    assert!(wait_ready(&pool, "pod-1").await, "never ready");
    assert!(
        wait_until(
            || buf_contains(&buf1, b"FIRST-BASELINE"),
            Duration::from_secs(3),
        )
        .await,
        "first subscriber did not receive its baseline",
    );
    buf1.lock().unwrap().clear();

    let (cb2, buf2) = make_output_cb();
    pool.subscribe("pod-1", "sub-2", &mock.url, "tok", cb2)
        .await;
    let resync = mock
        .recv(Duration::from_secs(3))
        .await
        .expect("fresh subscriber did not request a baseline");
    assert_eq!(resync[0], MsgType::Resync as u8);
    mock.push(MsgType::Output, b"BEFORE-SECOND-BASELINE");
    assert!(
        wait_until(
            || buf_has(&buf1, b"BEFORE-SECOND-BASELINE"),
            Duration::from_secs(3),
        )
        .await,
        "ready subscriber did not receive incremental output",
    );
    assert!(!buf_has(&buf2, b"BEFORE-SECOND-BASELINE"));

    push_snapshot(&mock, "FRESH-SNAPSHOT");
    assert!(
        wait_until(
            || buf_contains(&buf2, b"FRESH-SNAPSHOT"),
            Duration::from_secs(3),
        )
        .await,
        "pending subscriber did not receive its baseline",
    );
    assert!(!buf_contains(&buf1, b"FRESH-SNAPSHOT"));
    mock.push(MsgType::Output, b"AFTER-SECOND-BASELINE");
    assert!(
        wait_until(
            || {
                buf_has(&buf1, b"AFTER-SECOND-BASELINE") && buf_has(&buf2, b"AFTER-SECOND-BASELINE")
            },
            Duration::from_secs(3),
        )
        .await,
        "incremental output did not reach both ready subscribers",
    );
}

#[tokio::test]
async fn recovery_snapshot_resets_ready_and_pending_subscribers_once() {
    let mock = start_mock_relay().await;
    let (pool, _rx) = RelayConnectionPool::new();
    let (cb1, buf1) = make_output_cb();
    pool.subscribe("pod-1", "sub-1", &mock.url, "tok", cb1)
        .await;
    assert!(wait_transport(&mock).await, "never connected");
    push_snapshot(&mock, "INITIAL");
    assert!(wait_ready(&pool, "pod-1").await, "never ready");
    assert!(
        wait_until(|| buf_contains(&buf1, b"INITIAL"), Duration::from_secs(3)).await,
        "first subscriber did not receive its baseline",
    );
    buf1.lock().unwrap().clear();

    let (cb2, buf2) = make_output_cb();
    pool.subscribe("pod-1", "sub-2", &mock.url, "tok", cb2)
        .await;
    let resync = mock
        .recv(Duration::from_secs(3))
        .await
        .expect("fresh subscriber did not request a baseline");
    assert_eq!(resync[0], MsgType::Resync as u8);
    let recovery = serde_json::json!({
        "serialized_content": "RECOVERED",
        "reset_all": true,
        "cols": 80,
        "rows": 24,
    });
    mock.push(MsgType::Snapshot, recovery.to_string().as_bytes());
    assert!(
        wait_until(
            || buf_contains(&buf1, b"RECOVERED") && buf_contains(&buf2, b"RECOVERED"),
            Duration::from_secs(3),
        )
        .await,
        "recovery snapshot did not reach both subscribers",
    );
    assert_eq!(buf_count_contains(&buf1, b"RECOVERED"), 1);
    assert_eq!(buf_count_contains(&buf2, b"RECOVERED"), 1);
}

#[tokio::test]
async fn ready_subscriber_ignores_normal_snapshot_but_accepts_reset_all() {
    let mock = start_mock_relay().await;
    let (pool, _rx) = RelayConnectionPool::new();
    let (cb, buf) = make_output_cb();
    pool.subscribe("pod-1", "sub-1", &mock.url, "tok", cb).await;
    assert!(wait_transport(&mock).await, "never connected");
    push_snapshot(&mock, "INITIAL");
    assert!(wait_ready(&pool, "pod-1").await, "never ready");
    assert!(
        wait_until(|| buf_contains(&buf, b"INITIAL"), Duration::from_secs(3)).await,
        "subscriber did not receive its initial baseline",
    );
    buf.lock().unwrap().clear();

    push_snapshot(&mock, "IGNORED-SCREEN");
    mock.push(MsgType::Output, b"ORDER-MARKER");
    assert!(
        wait_until(|| buf_has(&buf, b"ORDER-MARKER"), Duration::from_secs(3)).await,
        "ordered output marker did not arrive",
    );
    assert!(!buf_contains(&buf, b"IGNORED-SCREEN"));
    let reset = serde_json::json!({
        "serialized_content": "AUTHORITATIVE-RESET",
        "reset_all": true,
        "cols": 80,
        "rows": 24,
    });
    mock.push(MsgType::Snapshot, reset.to_string().as_bytes());
    assert!(
        wait_until(
            || buf_contains(&buf, b"AUTHORITATIVE-RESET"),
            Duration::from_secs(3),
        )
        .await,
        "reset_all snapshot did not reset the ready subscriber",
    );
    assert_eq!(buf_count_contains(&buf, b"AUTHORITATIVE-RESET"), 1);
}

#[tokio::test]
async fn subscribe_ready_waits_for_candidate_baseline_before_handoff() {
    let mock = start_mock_relay().await;
    let (pool, _rx) = RelayConnectionPool::new();
    let (active_cb, active_buf) = make_output_cb();
    pool.subscribe("pod-1", "active", &mock.url, "tok", active_cb)
        .await;
    assert!(wait_transport(&mock).await, "never connected");
    push_snapshot(&mock, "ACTIVE-BASELINE");
    assert!(wait_ready(&pool, "pod-1").await, "never ready");
    active_buf.lock().unwrap().clear();

    let (candidate_cb, candidate_buf) = make_output_cb();
    let ready_pool = pool.clone();
    let ready_url = mock.url.clone();
    let ready = tokio::spawn(async move {
        ready_pool
            .subscribe_ready("pod-1", "candidate", &ready_url, "tok", candidate_cb)
            .await
    });
    let resync = mock
        .recv(Duration::from_secs(3))
        .await
        .expect("candidate did not request a baseline");
    assert_eq!(resync[0], MsgType::Resync as u8);

    mock.push(MsgType::Output, b"WHILE-CANDIDATE-PENDING");
    assert!(
        wait_until(
            || buf_has(&active_buf, b"WHILE-CANDIDATE-PENDING"),
            Duration::from_secs(3),
        )
        .await,
        "active subscriber stopped before candidate was ready",
    );
    assert!(!buf_has(&candidate_buf, b"WHILE-CANDIDATE-PENDING"));
    assert!(
        !ready.is_finished(),
        "candidate reported ready without a baseline"
    );

    push_snapshot(&mock, "CANDIDATE-BASELINE");
    let handle = tokio::time::timeout(Duration::from_secs(3), ready)
        .await
        .expect("candidate readiness timed out")
        .expect("candidate task panicked")
        .expect("candidate closed before baseline");
    assert_eq!(handle.subscription_id, "candidate");
    assert!(buf_contains(&candidate_buf, b"CANDIDATE-BASELINE"));

    pool.unsubscribe("pod-1", "active").await;
    active_buf.lock().unwrap().clear();
    candidate_buf.lock().unwrap().clear();
    mock.push(MsgType::Output, b"AFTER-HANDOFF");
    assert!(
        wait_until(
            || buf_has(&candidate_buf, b"AFTER-HANDOFF"),
            Duration::from_secs(3),
        )
        .await,
        "ready candidate did not receive post-handoff output",
    );
    assert!(!buf_has(&active_buf, b"AFTER-HANDOFF"));
}

#[tokio::test]
async fn subscribe_ready_fails_when_candidate_is_removed() {
    let mock = start_mock_relay().await;
    let (pool, _rx) = RelayConnectionPool::new();
    let (active_cb, _active_buf) = make_output_cb();
    pool.subscribe("pod-1", "active", &mock.url, "tok", active_cb)
        .await;
    assert!(wait_transport(&mock).await, "never connected");
    push_snapshot(&mock, "ACTIVE-BASELINE");
    assert!(wait_ready(&pool, "pod-1").await, "never ready");

    let (candidate_cb, _) = make_output_cb();
    let ready_pool = pool.clone();
    let ready_url = mock.url.clone();
    let ready = tokio::spawn(async move {
        ready_pool
            .subscribe_ready("pod-1", "candidate", &ready_url, "tok", candidate_cb)
            .await
    });
    let resync = mock
        .recv(Duration::from_secs(3))
        .await
        .expect("candidate did not request a baseline");
    assert_eq!(resync[0], MsgType::Resync as u8);

    pool.unsubscribe("pod-1", "candidate").await;
    let result = tokio::time::timeout(Duration::from_secs(3), ready)
        .await
        .expect("candidate removal did not resolve readiness")
        .expect("candidate task panicked");
    assert!(result.is_err(), "removed candidate reported ready");
}
