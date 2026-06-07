//! Injectable event-stream source: the seam between `connection_loop`'s
//! state machine and the real network. Production wraps `ApiClient`'s
//! Connect server-stream; tests inject a scripted stream. This is what lets
//! the reconnect / data-ready / timeout behavior be black-box tested without
//! a live HTTP server — the loop's bug is state-machine logic, not wire
//! parsing, so a `futures::stream` mock is enough and stays wasm/native-uniform.

use std::future::Future;
use std::pin::Pin;
use std::sync::Arc;

use agentsmesh_api_client::{ApiClient, ApiError};
use agentsmesh_types::proto_events_v1::{Event as ProtoEvent, SubscribeRequest};
use futures::stream::{Stream, StreamExt};
use tracing::debug;

use crate::event_types::EventType;
use crate::types::{EventCategory, RealtimeEvent};

/// One item off the realtime stream: a business event to dispatch, or a liveness
/// sentinel (server keepalive, `Event.keepalive=true`) that only proves the link
/// is alive. Distinct at the type level so a sentinel can never be dispatched as
/// a business event.
pub enum StreamFrame {
    Event(RealtimeEvent),
    Keepalive,
}

#[cfg(not(target_arch = "wasm32"))]
pub type BoxEventStream = Pin<Box<dyn Stream<Item = Result<StreamFrame, ApiError>> + Send>>;
#[cfg(target_arch = "wasm32")]
pub type BoxEventStream = Pin<Box<dyn Stream<Item = Result<StreamFrame, ApiError>>>>;

#[cfg(not(target_arch = "wasm32"))]
pub(crate) type SubscribeFuture = Pin<Box<dyn Future<Output = Result<BoxEventStream, ApiError>> + Send>>;
#[cfg(target_arch = "wasm32")]
pub(crate) type SubscribeFuture = Pin<Box<dyn Future<Output = Result<BoxEventStream, ApiError>>>>;

/// Opens one realtime event stream. Each call performs a fresh subscribe;
/// the returned stream yields decoded `RealtimeEvent`s until the link drops
/// (clean close → `None`, error → `Err`). `Send + Sync` on native because
/// `connection_loop` is spawned onto a multi-thread runtime; relaxed on wasm.
#[cfg(not(target_arch = "wasm32"))]
pub trait EventStreamSource: Send + Sync + 'static {
    fn subscribe(&self) -> SubscribeFuture;
}
#[cfg(target_arch = "wasm32")]
pub trait EventStreamSource: 'static {
    fn subscribe(&self) -> SubscribeFuture;
}

/// Production source over the real Connect server-stream. Reads the current
/// org slug per subscribe so an org switch is picked up on the next reconnect.
pub struct ApiClientStreamSource {
    api_client: Arc<ApiClient>,
}

impl ApiClientStreamSource {
    pub fn new(api_client: Arc<ApiClient>) -> Self {
        Self { api_client }
    }

    fn request(&self) -> SubscribeRequest {
        SubscribeRequest {
            org_slug: self.api_client.current_org_slug(),
            event_types: Vec::new(),
        }
    }
}

#[cfg(not(target_arch = "wasm32"))]
impl EventStreamSource for ApiClientStreamSource {
    fn subscribe(&self) -> SubscribeFuture {
        let client = Arc::clone(&self.api_client);
        let req = self.request();
        Box::pin(async move {
            let raw = client.subscribe_events_connect_native(&req).await?;
            Ok(Box::pin(raw.filter_map(map_proto)) as BoxEventStream)
        })
    }
}

#[cfg(target_arch = "wasm32")]
impl EventStreamSource for ApiClientStreamSource {
    fn subscribe(&self) -> SubscribeFuture {
        let client = Arc::clone(&self.api_client);
        let req = self.request();
        Box::pin(async move {
            let (raw, abort) = client.subscribe_events_connect_wasm(&req).await?;
            // Fold the abort handle into the stream state so its Drop (which
            // cancels the in-flight fetch) fires only when the loop drops the
            // stream — not at the end of this subscribe() future.
            let raw = Box::pin(raw);
            let kept = futures::stream::unfold((raw, abort), |(mut s, abort)| async move {
                s.next().await.map(|item| (item, (s, abort)))
            });
            Ok(Box::pin(kept.filter_map(map_proto)) as BoxEventStream)
        })
    }
}

async fn map_proto(item: Result<ProtoEvent, ApiError>) -> Option<Result<StreamFrame, ApiError>> {
    match item {
        Ok(p) if p.keepalive => Some(Ok(StreamFrame::Keepalive)),
        Ok(p) => proto_to_realtime(p).map(|e| Ok(StreamFrame::Event(e))),
        Err(e) => Some(Err(e)),
    }
}

/// Wire proto → domain event. Unknown server-side types are dropped (the
/// client may be older than the backend) rather than erroring the stream.
fn proto_to_realtime(p: ProtoEvent) -> Option<RealtimeEvent> {
    let event_type = match serde_json::from_str::<EventType>(&format!("\"{}\"", p.r#type)) {
        Ok(t) => t,
        Err(_) => {
            debug!("events: unknown type from server: {}", p.r#type);
            return None;
        }
    };
    let category = match p.category.as_str() {
        "entity" => Some(EventCategory::Entity),
        "notification" => Some(EventCategory::Notification),
        "system" => Some(EventCategory::System),
        _ => None,
    };
    let data: serde_json::Value =
        serde_json::from_str(&p.data_json).unwrap_or(serde_json::Value::Null);
    Some(RealtimeEvent {
        event_type,
        category,
        organization_id: p.organization_id,
        target_user_id: p.target_user_id,
        target_user_ids: if p.target_user_ids.is_empty() {
            None
        } else {
            Some(p.target_user_ids)
        },
        entity_type: p.entity_type,
        entity_id: p.entity_id,
        data,
        timestamp: p.timestamp,
    })
}
