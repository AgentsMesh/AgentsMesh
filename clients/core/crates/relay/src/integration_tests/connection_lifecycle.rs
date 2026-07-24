use super::*;

#[tokio::test]
async fn default_pool_ignores_commands_for_unknown_pods() {
    let pool = RelayConnectionPool::default();

    assert_eq!(
        pool.get_status("unknown-pod").await,
        RelayStatus::Disconnected
    );
    assert!(!pool.is_runner_disconnected("unknown-pod").await);
    assert_eq!(pool.get_pod_size("unknown-pod").await, None);

    pool.send("unknown-pod", "input").await;
    pool.send_resize("unknown-pod", 80, 24).await;
    pool.force_resize("unknown-pod", 120, 40).await;
    pool.disconnect("unknown-pod").await;
    pool.disconnect_all().await;

    assert_eq!(
        pool.get_status("unknown-pod").await,
        RelayStatus::Disconnected,
        "commands for an unknown pod must not create a driver entry"
    );
}

#[tokio::test]
async fn connection_failure_publishes_error_until_disconnected() {
    let (pool, _rx) = RelayConnectionPool::new();
    let (cb, _buf) = make_output_cb();
    pool.subscribe("pod-error", "sub-1", "not-a-relay-url", "tok", cb)
        .await;

    let started = std::time::Instant::now();
    while pool.get_status("pod-error").await != RelayStatus::Error {
        assert!(
            started.elapsed() < Duration::from_secs(2),
            "failed connection never published Error",
        );
        tokio::time::sleep(Duration::from_millis(10)).await;
    }
    pool.disconnect("pod-error").await;
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
        assert!(
            start.elapsed() < Duration::from_secs(3),
            "runner_disconnected never set"
        );
        tokio::time::sleep(Duration::from_millis(10)).await;
    }
    mock.push(MsgType::RunnerReconnected, &[]);
    let start = std::time::Instant::now();
    while pool.is_runner_disconnected("pod-1").await {
        assert!(
            start.elapsed() < Duration::from_secs(3),
            "runner_disconnected never cleared"
        );
        tokio::time::sleep(Duration::from_millis(10)).await;
    }
}

#[tokio::test]
async fn reconnects_after_server_drop() {
    let mock = start_mock_relay().await;
    let (pool, _rx) = RelayConnectionPool::new();
    let (cb, buf) = make_output_cb();
    pool.subscribe("pod-1", "sub-1", &mock.url, "tok", cb).await;
    assert!(
        wait_until(
            || mock.conn_count.load(SeqCst) >= 1,
            Duration::from_secs(10)
        )
        .await,
        "no first connect"
    );

    mock.drop_signal.send(()).unwrap();
    // schedule_reconnect waits ~BASE_RECONNECT_DELAY_MS (1s) before re-dialing.
    // Timeouts are generous: under the test binary's full parallelism (one
    // tokio runtime per #[tokio::test] thread) the reconnect's wall clock can
    // stretch well past the ~1s backoff. This asserts the behavior, not an SLA.
    assert!(
        wait_until(
            || mock.conn_count.load(SeqCst) >= 2,
            Duration::from_secs(20)
        )
        .await,
        "pool did not reconnect after server drop",
    );

    push_snapshot(&mock, "RECONNECTED-BASELINE");
    assert!(
        wait_ready(&pool, "pod-1").await,
        "reconnected link never became ready"
    );
    mock.push(MsgType::Output, b"after-reconnect");
    assert!(
        wait_until(
            || buf_has(&buf, b"after-reconnect"),
            Duration::from_secs(10)
        )
        .await,
        "output did not flow after reconnect",
    );
}

#[tokio::test]
async fn reconnect_revokes_ready_and_defers_resize_until_new_baseline() {
    let mock = start_mock_relay().await;
    let (pool, _rx) = RelayConnectionPool::new();
    let (cb, _buf) = make_output_cb();
    pool.subscribe("pod-1", "sub-1", &mock.url, "tok", cb).await;
    assert!(wait_transport(&mock).await, "never connected");

    push_snapshot(&mock, "INITIAL-BASELINE");
    assert!(wait_ready(&pool, "pod-1").await, "never became ready");

    mock.drop_signal.send(()).unwrap();
    let started = std::time::Instant::now();
    while pool.get_status("pod-1").await != RelayStatus::Connecting {
        assert!(
            started.elapsed() < Duration::from_millis(750),
            "transport close must revoke Connected before reconnect backoff"
        );
        tokio::time::sleep(Duration::from_millis(10)).await;
    }
    assert_eq!(
        mock.conn_count.load(SeqCst),
        1,
        "status transition must be visible before the next transport connects"
    );

    // Simulate a renderer resize racing with delayed cross-thread status
    // delivery. The core driver remains authoritative and must not write this
    // into the next transport generation before its snapshot baseline.
    pool.force_resize("pod-1", 96, 31).await;
    assert!(
        wait_until(
            || mock.conn_count.load(SeqCst) >= 2,
            Duration::from_secs(10)
        )
        .await,
        "pool did not reconnect after server drop",
    );
    assert_no_client_frame_type(&mock, MsgType::Resize, Duration::from_millis(400)).await;

    push_snapshot(&mock, "RECONNECTED-BASELINE");
    assert!(
        wait_ready(&pool, "pod-1").await,
        "reconnect never became ready"
    );
    let frame = mock
        .recv(Duration::from_secs(3))
        .await
        .expect("queued resize did not flush after reconnect baseline");
    assert_eq!(frame[0], MsgType::Resize as u8);
    assert_eq!(&frame[1..], &[0, 96, 0, 31]);
}

#[tokio::test]
async fn ready_subscriber_does_not_bypass_pending_subscriber_baseline() {
    let mock = start_mock_relay().await;
    let (pool, _rx) = RelayConnectionPool::new();
    let (first_cb, _first_buf) = make_output_cb();
    pool.subscribe("pod-1", "sub-ready", &mock.url, "tok", first_cb)
        .await;
    assert!(wait_transport(&mock).await, "never connected");
    push_snapshot(&mock, "INITIAL-BASELINE");
    assert!(wait_ready(&pool, "pod-1").await, "never became ready");

    let (pending_cb, _pending_buf) = make_output_cb();
    let subscribe = pool.subscribe_ready("pod-1", "sub-pending", &mock.url, "tok", pending_cb);
    tokio::pin!(subscribe);

    // Poll once so AddSubscriber reaches the actor, then issue a force resize
    // while the original subscriber remains ready. The pending candidate must
    // gate the entire pod until its own authoritative baseline is delivered.
    tokio::select! {
        _ = &mut subscribe => panic!("pending subscriber resolved without a baseline"),
        _ = tokio::time::sleep(Duration::from_millis(50)) => {}
    }
    pool.force_resize("pod-1", 101, 37).await;
    assert_no_client_frame_type(&mock, MsgType::Resize, Duration::from_millis(400)).await;

    push_snapshot(&mock, "CANDIDATE-BASELINE");
    let _candidate_handle = tokio::time::timeout(Duration::from_secs(3), &mut subscribe)
        .await
        .expect("pending subscriber did not become ready")
        .expect("pending subscriber closed before baseline");
    let frame = mock
        .recv(Duration::from_secs(3))
        .await
        .expect("queued resize did not flush after candidate baseline");
    assert_eq!(frame[0], MsgType::Resize as u8);
    assert_eq!(&frame[1..], &[0, 101, 0, 37]);
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
        serde_json::json!({"serialized_content":"x","reset_all":false,"cols":80,"rows":24})
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
async fn input_dedup_within_window() {
    let mock = start_mock_relay().await;
    let (pool, _rx) = RelayConnectionPool::new();
    let (cb, _buf) = make_output_cb();
    pool.subscribe("pod-1", "sub-1", &mock.url, "tok", cb).await;
    assert!(wait_transport(&mock).await, "never connected");

    pool.send("pod-1", "dupe").await;
    pool.send("pod-1", "dupe").await; // identical, within 50ms → deduped
    let f1 = mock
        .recv(Duration::from_secs(2))
        .await
        .expect("first input");
    assert_eq!(f1[0], MsgType::Input as u8);
    assert_eq!(&f1[1..], b"dupe");
    let f2 = mock.recv(Duration::from_millis(300)).await;
    assert!(
        f2.map_or(true, |f| f[0] != MsgType::Input as u8),
        "identical input within the dedup window must not reach the server twice",
    );
}

#[tokio::test]
async fn text_frames_and_zero_resize_do_not_poison_the_session() {
    let mock = start_mock_relay().await;
    let (pool, _rx) = RelayConnectionPool::new();
    let (cb, buf) = make_output_cb();
    pool.subscribe("pod-1", "sub-1", &mock.url, "tok", cb).await;
    assert!(wait_transport(&mock).await, "never connected");

    mock.push_text("relay metadata");
    pool.send_resize("pod-1", 0, 24).await;
    push_snapshot(&mock, "VALID-AFTER-NOOPS");
    assert!(
        wait_until(
            || buf_contains(&buf, b"VALID-AFTER-NOOPS"),
            Duration::from_secs(3),
        )
        .await,
        "ignored inputs terminated or poisoned the relay session",
    );
}

#[tokio::test]
async fn subscriber_rejoins_during_disconnect_grace_on_the_same_transport() {
    let mock = start_mock_relay().await;
    let (pool, _rx) = RelayConnectionPool::new();
    let (first, _first_buf) = make_output_cb();
    pool.subscribe("pod-1", "sub-1", &mock.url, "tok", first)
        .await;
    assert!(wait_transport(&mock).await, "never connected");
    push_snapshot(&mock, "FIRST-BASELINE");
    assert!(wait_ready(&pool, "pod-1").await, "never became ready");

    let connections = mock.conn_count.load(SeqCst);
    pool.unsubscribe("pod-1", "sub-1").await;
    let (second, second_buf) = make_output_cb();
    pool.subscribe("pod-1", "sub-2", &mock.url, "tok", second)
        .await;
    let resync = mock
        .recv(Duration::from_secs(3))
        .await
        .expect("replacement subscriber did not request a baseline");
    assert_eq!(resync[0], MsgType::Resync as u8);
    push_snapshot(&mock, "REJOINED-DURING-GRACE");

    assert!(
        wait_until(
            || buf_contains(&second_buf, b"REJOINED-DURING-GRACE"),
            Duration::from_secs(3),
        )
        .await,
        "replacement subscriber never received its baseline",
    );
    assert_eq!(
        mock.conn_count.load(SeqCst),
        connections,
        "grace-period rejoin should reuse the existing transport",
    );
}

#[tokio::test]
async fn force_resize_sends_immediately() {
    let mock = start_mock_relay().await;
    let (pool, _rx) = RelayConnectionPool::new();
    let (cb, _buf) = make_output_cb();
    pool.subscribe("pod-1", "sub-1", &mock.url, "tok", cb).await;
    assert!(wait_transport(&mock).await, "never connected");
    push_snapshot(&mock, "FORCE-RESIZE-BASELINE");
    assert!(wait_ready(&pool, "pod-1").await, "never became ready");

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
        serde_json::json!({"serialized_content":"a","reset_all":false,"cols":80,"rows":24})
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
        serde_json::json!({"serialized_content":"b","reset_all":false,"cols":120,"rows":40})
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

#[tokio::test]
async fn desktop_listener_lease_advances_across_driver_replacement() {
    let mock = start_mock_relay().await;
    let (pool, _rx) = RelayConnectionPool::new();
    let disconnected = Arc::new(Mutex::new(Vec::<u32>::new()));
    {
        let disconnected = Arc::clone(&disconnected);
        pool.set_on_pod_generation_disconnected(Arc::new(move |pod_key, generation| {
            assert_eq!(pod_key, "pod-1");
            disconnected.lock().unwrap().push(generation);
        }));
    }

    let bound = Arc::new(Mutex::new(Vec::<u32>::new()));
    let subscribe_generation = |subscription_id: &'static str| {
        let pool = pool.clone();
        let url = mock.url.clone();
        let bound = Arc::clone(&bound);
        tokio::spawn(async move {
            let (output, _) = make_output_cb();
            let status: crate::types::GenerationStatusCallback = Arc::new(|_, _| {});
            let acp: crate::types::GenerationAcpCallback = Arc::new(|_, _, _| {});
            pool.subscribe_ready_with_listeners(
                "pod-1",
                subscription_id,
                &url,
                "tok",
                output,
                "desktop-test-listeners",
                status,
                acp,
                Arc::new(move |generation| bound.lock().unwrap().push(generation)),
            )
            .await
        })
    };

    let first = subscribe_generation("sub-1");
    assert!(wait_transport(&mock).await, "first driver never connected");
    push_snapshot(&mock, "FIRST");
    first
        .await
        .expect("first subscribe task panicked")
        .expect("first subscriber never became ready");
    let first_generation = bound.lock().unwrap()[0];

    pool.disconnect("pod-1").await;
    assert!(
        wait_until(
            || disconnected.lock().unwrap().contains(&first_generation),
            Duration::from_secs(3),
        )
        .await,
        "retired driver generation was not reported",
    );

    let connections_before = mock.conn_count.load(SeqCst);
    let second = subscribe_generation("sub-2");
    assert!(
        wait_until(
            || mock.conn_count.load(SeqCst) > connections_before,
            Duration::from_secs(3),
        )
        .await,
        "replacement driver never connected",
    );
    push_snapshot(&mock, "SECOND");
    second
        .await
        .expect("second subscribe task panicked")
        .expect("second subscriber never became ready");
    let generations = bound.lock().unwrap().clone();
    assert_eq!(generations.len(), 2);
    assert!(
        generations[1] > generations[0],
        "replacement driver must publish a newer listener lease: {generations:?}",
    );
}

#[tokio::test]
async fn pod_resized_frame_updates_the_pool_snapshot() {
    let mock = start_mock_relay().await;
    let (pool, _rx) = RelayConnectionPool::new();
    let (cb, _buf) = make_output_cb();
    pool.subscribe("pod-1", "sub-1", &mock.url, "tok", cb).await;
    assert!(wait_transport(&mock).await, "never connected");

    mock.push(
        MsgType::Control,
        serde_json::json!({"type": "pod_resized", "cols": 132, "rows": 44})
            .to_string()
            .as_bytes(),
    );
    assert!(
        wait_until_pod_size(&pool, "pod-1", (132, 44)).await,
        "PodResized did not update the pool-readable size",
    );
}

#[tokio::test]
async fn malformed_frame_is_ignored_without_poisoning_the_session() {
    let mock = start_mock_relay().await;
    let (pool, _rx) = RelayConnectionPool::new();
    let (cb, buf) = make_output_cb();
    pool.subscribe("pod-1", "sub-1", &mock.url, "tok", cb).await;
    assert!(wait_transport(&mock).await, "never connected");

    mock.to_client.send(Vec::new()).unwrap();
    push_snapshot(&mock, "VALID-AFTER-MALFORMED");
    assert!(
        wait_until(
            || buf_contains(&buf, b"VALID-AFTER-MALFORMED"),
            Duration::from_secs(3),
        )
        .await,
        "malformed frame terminated or poisoned the relay session",
    );
}
