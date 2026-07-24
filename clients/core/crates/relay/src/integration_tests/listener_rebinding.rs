use super::*;

#[tokio::test]
async fn desktop_listener_lease_is_idempotent_within_one_driver_generation() {
    let mock = start_mock_relay().await;
    let (pool, _rx) = RelayConnectionPool::new();
    let first_status = Arc::new(AtomicUsize::new(0));
    let first_acp = Arc::new(AtomicUsize::new(0));

    let subscribe_with_listeners =
        |subscription_id: &'static str,
         status_count: Arc<AtomicUsize>,
         acp_count: Arc<AtomicUsize>| {
            let pool = pool.clone();
            let url = mock.url.clone();
            tokio::spawn(async move {
                let (output, _) = make_output_cb();
                let status: crate::types::GenerationStatusCallback = Arc::new(move |_, _| {
                    status_count.fetch_add(1, SeqCst);
                });
                let acp: crate::types::GenerationAcpCallback = Arc::new(move |_, _, _| {
                    acp_count.fetch_add(1, SeqCst);
                });
                pool.subscribe_ready_with_listeners(
                    "pod-1",
                    subscription_id,
                    &url,
                    "tok",
                    output,
                    "desktop-shared-listeners",
                    status,
                    acp,
                    Arc::new(|_| {}),
                )
                .await
            })
        };

    let first =
        subscribe_with_listeners("sub-1", Arc::clone(&first_status), Arc::clone(&first_acp));
    assert!(wait_transport(&mock).await, "driver never connected");
    push_snapshot(&mock, "FIRST");
    first
        .await
        .expect("first subscribe task panicked")
        .expect("first subscriber never became ready");

    let replacement_status = Arc::new(AtomicUsize::new(0));
    let replacement_acp = Arc::new(AtomicUsize::new(0));
    let second = subscribe_with_listeners(
        "sub-2",
        Arc::clone(&replacement_status),
        Arc::clone(&replacement_acp),
    );
    let resync = mock
        .recv(Duration::from_secs(3))
        .await
        .expect("second subscriber did not request a baseline");
    assert_eq!(resync[0], MsgType::Resync as u8);
    push_snapshot(&mock, "SECOND");
    second
        .await
        .expect("second subscribe task panicked")
        .expect("second subscriber never became ready");

    let status_before = first_status.load(SeqCst);
    mock.push(MsgType::RunnerDisconnected, &[]);
    mock.push(
        MsgType::AcpEvent,
        serde_json::json!({"event":"still-first-listener"})
            .to_string()
            .as_bytes(),
    );
    assert!(
        wait_until(
            || first_status.load(SeqCst) > status_before && first_acp.load(SeqCst) == 1,
            Duration::from_secs(3),
        )
        .await,
        "original listener pair did not remain active",
    );
    assert_eq!(
        replacement_status.load(SeqCst),
        0,
        "same-lease subscribe replaced the status listener",
    );
    assert_eq!(
        replacement_acp.load(SeqCst),
        0,
        "same-lease subscribe replaced the ACP listener",
    );
}

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
        pool.on_status_change(
            "pod-1",
            Arc::new(move |_info| {
                c.fetch_add(1, SeqCst);
            }),
        )
        .await;
    }
    let count_b = Arc::new(AtomicUsize::new(0));
    {
        let c = count_b.clone();
        pool.on_status_change(
            "pod-1",
            Arc::new(move |_info| {
                c.fetch_add(1, SeqCst);
            }),
        )
        .await;
    }
    let a_after_replace = count_a.load(SeqCst);

    let snap = serde_json::json!({
        "serialized_content":"X",
        "reset_all":false,
        "cols":80,
        "rows":24
    });
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
