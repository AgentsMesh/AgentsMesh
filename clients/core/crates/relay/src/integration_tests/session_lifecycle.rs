use super::*;

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
