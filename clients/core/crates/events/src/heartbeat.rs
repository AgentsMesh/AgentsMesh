use std::time::Duration;

use agentsmesh_transport::runtime::{PlatformRuntime, Runtime, TaskHandle};
use futures::channel::mpsc;
use futures::stream::StreamExt;
use tracing::{debug, warn};

pub(crate) enum HeartbeatCommand {
    PongReceived,
    Stop,
}

pub(crate) enum HeartbeatEvent {
    SendPing,
    PongTimeout,
}

pub(crate) struct HeartbeatManager<R: Runtime = PlatformRuntime> {
    ping_interval: Duration,
    pong_timeout: Duration,
    cmd_tx: Option<mpsc::UnboundedSender<HeartbeatCommand>>,
    task: Option<R::TaskHandle>,
    runtime: R,
}

impl HeartbeatManager<PlatformRuntime> {
    #[allow(dead_code)]
    pub fn new(ping_interval_ms: u64, pong_timeout_ms: u64) -> Self {
        Self::with_runtime(PlatformRuntime, ping_interval_ms, pong_timeout_ms)
    }
}

impl<R: Runtime> HeartbeatManager<R> {
    pub fn with_runtime(runtime: R, ping_interval_ms: u64, pong_timeout_ms: u64) -> Self {
        Self {
            ping_interval: Duration::from_millis(ping_interval_ms),
            pong_timeout: Duration::from_millis(pong_timeout_ms),
            cmd_tx: None,
            task: None,
            runtime,
        }
    }

    pub fn start(&mut self, event_tx: mpsc::UnboundedSender<HeartbeatEvent>) {
        self.stop();
        let (cmd_tx, cmd_rx) = mpsc::unbounded();
        self.cmd_tx = Some(cmd_tx);

        let ping_interval = self.ping_interval;
        let pong_timeout = self.pong_timeout;
        let rt = self.runtime.clone();

        self.task = Some(self.runtime.spawn(Box::pin(
            heartbeat_loop(rt, ping_interval, pong_timeout, cmd_rx, event_tx),
        )));
    }

    pub fn stop(&mut self) {
        if let Some(tx) = self.cmd_tx.take() {
            let _ = tx.unbounded_send(HeartbeatCommand::Stop);
        }
        if let Some(task) = self.task.take() {
            task.abort();
        }
    }

    pub fn pong_received(&self) {
        if let Some(tx) = &self.cmd_tx {
            let _ = tx.unbounded_send(HeartbeatCommand::PongReceived);
        }
    }
}

impl<R: Runtime> Drop for HeartbeatManager<R> {
    fn drop(&mut self) {
        self.stop();
    }
}

async fn recv_with_timeout<R: Runtime>(
    runtime: &R,
    duration: Duration,
    cmd_rx: &mut mpsc::UnboundedReceiver<HeartbeatCommand>,
) -> Option<Option<HeartbeatCommand>> {
    use futures::future::{select, Either};
    let sleep = runtime.sleep(duration);
    let recv = cmd_rx.next();
    futures::pin_mut!(sleep);
    futures::pin_mut!(recv);
    match select(sleep, recv).await {
        Either::Left((_, _)) => None,
        Either::Right((cmd, _)) => Some(cmd),
    }
}

async fn heartbeat_loop<R: Runtime>(
    runtime: R,
    ping_interval: Duration,
    pong_timeout: Duration,
    mut cmd_rx: mpsc::UnboundedReceiver<HeartbeatCommand>,
    event_tx: mpsc::UnboundedSender<HeartbeatEvent>,
) {
    runtime.sleep(ping_interval).await;

    loop {
        if event_tx.unbounded_send(HeartbeatEvent::SendPing).is_err() {
            break;
        }
        debug!("heartbeat: sending ping");

        match recv_with_timeout(&runtime, pong_timeout, &mut cmd_rx).await {
            None => {
                warn!("heartbeat: pong timeout");
                let _ = event_tx.unbounded_send(HeartbeatEvent::PongTimeout);
                break;
            }
            Some(Some(HeartbeatCommand::PongReceived)) => {
                debug!("heartbeat: pong received");
            }
            Some(Some(HeartbeatCommand::Stop)) | Some(None) => break,
        }

        match recv_with_timeout(&runtime, ping_interval, &mut cmd_rx).await {
            None => {}
            Some(Some(HeartbeatCommand::Stop)) | Some(None) => break,
            Some(Some(HeartbeatCommand::PongReceived)) => {}
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_heartbeat_sends_ping() {
        let mut hb = HeartbeatManager::new(100, 5000);
        let (event_tx, mut event_rx) = mpsc::unbounded();
        hb.start(event_tx);

        match tokio::time::timeout(Duration::from_millis(300), event_rx.next()).await {
            Ok(Some(HeartbeatEvent::SendPing)) => {}
            other => panic!("expected SendPing, got {:?}", other.is_ok()),
        }
        hb.stop();
    }

    #[tokio::test]
    async fn test_heartbeat_pong_timeout() {
        let mut hb = HeartbeatManager::new(200, 50);
        let (event_tx, mut event_rx) = mpsc::unbounded();
        hb.start(event_tx);

        match tokio::time::timeout(Duration::from_millis(500), event_rx.next()).await {
            Ok(Some(HeartbeatEvent::SendPing)) => {}
            _ => panic!("expected SendPing first"),
        }
        match tokio::time::timeout(Duration::from_millis(300), event_rx.next()).await {
            Ok(Some(HeartbeatEvent::PongTimeout)) => {}
            other => panic!("expected PongTimeout, got {:?}", other.is_ok()),
        }
    }

    #[tokio::test]
    async fn test_heartbeat_pong_clears_timeout() {
        let mut hb = HeartbeatManager::new(50, 80);
        let (event_tx, mut event_rx) = mpsc::unbounded();
        hb.start(event_tx);

        let _ = event_rx.next().await; // SendPing
        hb.pong_received();

        match tokio::time::timeout(Duration::from_millis(200), event_rx.next()).await {
            Ok(Some(HeartbeatEvent::SendPing)) => {}
            other => panic!("expected second SendPing, got {:?}", other.is_ok()),
        }
        hb.stop();
    }

    #[tokio::test]
    async fn test_heartbeat_stop() {
        let mut hb = HeartbeatManager::new(50, 50);
        let (event_tx, mut event_rx) = mpsc::unbounded();
        hb.start(event_tx);
        hb.stop();

        tokio::time::sleep(Duration::from_millis(100)).await;
        assert!(event_rx.try_next().is_err() || event_rx.try_next().unwrap().is_none());
    }
}
