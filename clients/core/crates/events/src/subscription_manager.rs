use std::collections::HashMap;
use std::sync::atomic::{AtomicU64, Ordering};
use std::sync::Arc;

use agentsmesh_api_client::ApiClient;
use agentsmesh_transport::runtime::{PlatformRuntime, Runtime};
use futures::channel::mpsc;
use parking_lot::RwLock;

use crate::event_types::EventType;
use crate::stream_source::{ApiClientStreamSource, EventStreamSource};
use crate::types::{
    ConnectionState, EventDispatchHook, EventHandler, EventSubscriptionManagerOptions,
    RealtimeEvent, StateListener, SubscriptionId, TickListener,
};

static NEXT_SUB_ID: AtomicU64 = AtomicU64::new(1);

fn next_subscription_id() -> SubscriptionId {
    SubscriptionId(NEXT_SUB_ID.fetch_add(1, Ordering::Relaxed))
}

pub(crate) type HandlerMap = HashMap<EventType, HashMap<SubscriptionId, EventHandler>>;

pub(crate) struct Inner {
    pub handlers: HandlerMap,
    pub global_handlers: HashMap<SubscriptionId, EventHandler>,
    pub state_listeners: HashMap<SubscriptionId, StateListener>,
    pub connection_state: ConnectionState,
    /// Optional Rust-SSOT dispatch hook. When set, every event is applied
    /// to the registered hook (typically `AppState.dispatch`) BEFORE the
    /// tick bump and BEFORE external handlers. See `EventDispatchHook`.
    pub dispatch_hook: Option<Arc<dyn EventDispatchHook>>,
    /// Monotonic counter incremented after every dispatched event. Read
    /// by platform selectors via `useCoreTick()` / `getTick()` /
    /// `TickCallback.on_tick` to invalidate cached views.
    pub tick: AtomicU64,
    /// Optional tick push-listener — fired after every event dispatched
    /// to AppState + tick bump. Used by iOS (UniFFI) to kick SwiftUI's
    /// reactive store; other platforms poll `tick()`.
    pub tick_listener: Option<Arc<dyn TickListener>>,
    /// Shutdown signal for the active connection_loop task. Moved into
    /// `Inner` so `connect()` / `disconnect()` can take `&self` — this
    /// lets binding facades hold `Arc<EventSubscriptionManager>` and
    /// pass it to `AppRuntime` without `Rc<RefCell<...>>` or `&mut`
    /// borrow contortions.
    pub shutdown_tx: Option<mpsc::UnboundedSender<()>>,
    /// Interrupts the current backoff sleep so a reconnect happens now
    /// (network regained / app foregrounded). None when no loop is running.
    pub nudge_tx: Option<mpsc::UnboundedSender<()>>,
}

pub struct EventSubscriptionManager<R: Runtime = PlatformRuntime> {
    pub(crate) inner: Arc<RwLock<Inner>>,
    pub(crate) options: EventSubscriptionManagerOptions,
    source: Arc<dyn EventStreamSource>,
    runtime: R,
}

impl EventSubscriptionManager<PlatformRuntime> {
    pub fn new(api_client: Arc<ApiClient>, options: EventSubscriptionManagerOptions) -> Self {
        Self::with_runtime(PlatformRuntime, api_client, options)
    }

    pub fn with_default_options(api_client: Arc<ApiClient>) -> Self {
        Self::new(api_client, EventSubscriptionManagerOptions::default())
    }
}

impl<R: Runtime> EventSubscriptionManager<R> {
    pub fn with_runtime(
        runtime: R,
        api_client: Arc<ApiClient>,
        options: EventSubscriptionManagerOptions,
    ) -> Self {
        Self::with_stream_source(
            runtime,
            Arc::new(ApiClientStreamSource::new(api_client)),
            options,
        )
    }

    /// Construct over an arbitrary stream source. Production goes through
    /// `with_runtime` (real ApiClient); tests inject a scripted source to
    /// exercise reconnect / data-ready / timeout without a live server.
    pub(crate) fn with_stream_source(
        runtime: R,
        source: Arc<dyn EventStreamSource>,
        options: EventSubscriptionManagerOptions,
    ) -> Self {
        Self {
            inner: Arc::new(RwLock::new(Inner {
                handlers: HashMap::new(),
                global_handlers: HashMap::new(),
                state_listeners: HashMap::new(),
                connection_state: ConnectionState::Disconnected,
                dispatch_hook: None,
                tick: AtomicU64::new(0),
                tick_listener: None,
                shutdown_tx: None,
                nudge_tx: None,
            })),
            options,
            source,
            runtime,
        }
    }

    /// Install a Rust-SSOT dispatch hook. Called by binding facades
    /// (wasm/napi/ffi) at construction time with an `AppStateDispatchHook`.
    /// Subsequent calls replace any prior hook.
    pub fn set_dispatch_hook(&self, hook: Arc<dyn EventDispatchHook>) {
        self.inner.write().dispatch_hook = Some(hook);
    }

    /// Install a tick push-listener. iOS uses this to invalidate SwiftUI
    /// state on each event. Other platforms can poll `tick()` instead.
    /// Replaces any prior listener — single-slot, not a Vec.
    pub fn set_tick_listener(&self, listener: Arc<dyn TickListener>) {
        self.inner.write().tick_listener = Some(listener);
    }

    /// Clear the tick listener (e.g. on iOS app teardown).
    pub fn clear_tick_listener(&self) {
        self.inner.write().tick_listener = None;
    }

    /// Current tick value. Increments once per event dispatched, after
    /// the dispatch hook has applied the event to AppState. Platform
    /// selectors use this as the React `useSyncExternalStore` snapshot.
    pub fn tick(&self) -> u64 {
        self.inner.read().tick.load(Ordering::Acquire)
    }

    pub async fn connect(&self) {
        let state = self.inner.read().connection_state;
        if state == ConnectionState::Connected || state == ConnectionState::Connecting {
            return;
        }

        let (shutdown_tx, shutdown_rx) = mpsc::unbounded();
        let (nudge_tx, nudge_rx) = mpsc::unbounded();
        {
            let mut w = self.inner.write();
            w.shutdown_tx = Some(shutdown_tx);
            w.nudge_tx = Some(nudge_tx);
        }

        let inner = Arc::clone(&self.inner);
        let source = Arc::clone(&self.source);
        let opts = crate::connection_loop::ManagerOpts::from_options(&self.options);
        let rt = self.runtime.clone();

        self.runtime.spawn(Box::pin(
            crate::connection_loop::connection_loop(rt, inner, source, opts, shutdown_rx, nudge_rx),
        ));
    }

    pub async fn disconnect(&self) {
        let tx = {
            let mut w = self.inner.write();
            w.nudge_tx = None;
            w.shutdown_tx.take()
        };
        if let Some(tx) = tx {
            let _ = tx.unbounded_send(());
        }
        set_state(&self.inner, ConnectionState::Disconnected);
    }

    /// Interrupt the current reconnect backoff and retry immediately. Front-ends
    /// call this on network-regained / app-foreground so recovery doesn't wait
    /// out the (capped, up to 30s) backoff. No-op when connected or shut down.
    pub async fn nudge(&self) {
        let tx = self.inner.read().nudge_tx.clone();
        if let Some(tx) = tx {
            let _ = tx.unbounded_send(());
        }
    }

    pub async fn subscribe(
        &self,
        event_type: EventType,
        handler: EventHandler,
    ) -> SubscriptionId {
        let id = next_subscription_id();
        let mut inner = self.inner.write();
        inner.handlers.entry(event_type).or_default().insert(id, handler);
        id
    }

    pub async fn subscribe_all(&self, handler: EventHandler) -> SubscriptionId {
        let id = next_subscription_id();
        self.inner.write().global_handlers.insert(id, handler);
        id
    }

    pub async fn unsubscribe(&self, id: SubscriptionId) {
        let mut inner = self.inner.write();
        for handlers in inner.handlers.values_mut() {
            handlers.remove(&id);
        }
        inner.global_handlers.remove(&id);
        inner.state_listeners.remove(&id);
    }

    pub async fn on_connection_state_change(
        &self,
        listener: StateListener,
    ) -> SubscriptionId {
        let id = next_subscription_id();
        let mut inner = self.inner.write();
        let current = inner.connection_state;
        inner.state_listeners.insert(id, Arc::clone(&listener));
        listener(current);
        id
    }

    pub async fn get_connection_state(&self) -> ConnectionState {
        self.inner.read().connection_state
    }
}

pub(crate) fn set_state(inner: &Arc<RwLock<Inner>>, state: ConnectionState) {
    let listeners: Vec<StateListener> = {
        let mut guard = inner.write();
        if guard.connection_state == state {
            return;
        }
        guard.connection_state = state;
        guard.state_listeners.values().cloned().collect()
    };
    for listener in listeners {
        listener(state);
    }
}

pub(crate) fn dispatch_event(inner: &Arc<RwLock<Inner>>, event: &RealtimeEvent) {
    tracing::debug!(target: "events", event_type = ?event.event_type, "dispatch");
    // Step 1: Rust-SSOT dispatch hook applies event to AppState. We clone
    // the Arc out under read lock, then drop the events-side lock before
    // calling the hook — the hook will acquire its own AppState write
    // lock, and we must not hold the events lock during that operation
    // to avoid lock-order inversion with reader paths.
    let hook = inner.read().dispatch_hook.clone();
    if let Some(h) = hook {
        // Wrap in catch_unwind so a panic inside one event's apply logic
        // never poisons the dispatcher. parking_lot doesn't poison locks
        // but a panicking hook would still abort the connection loop.
        let result = std::panic::catch_unwind(std::panic::AssertUnwindSafe(|| {
            h.dispatch(event);
        }));
        if result.is_err() {
            tracing::error!(
                event_type = ?event.event_type,
                "events: dispatch hook panicked; continuing"
            );
        }
    }

    // Step 2: bump tick. Release ordering pairs with the Acquire load in
    // `tick()` so platform selectors observe the AppState mutations
    // above before reading the new tick value.
    let new_tick = inner.read().tick.fetch_add(1, Ordering::Release) + 1;

    // Step 2b: push tick to FFI listener (iOS). Wasm/napi poll instead.
    let tick_listener = inner.read().tick_listener.clone();
    if let Some(l) = tick_listener {
        let result = std::panic::catch_unwind(std::panic::AssertUnwindSafe(|| {
            l.on_tick(new_tick);
        }));
        if result.is_err() {
            tracing::error!("events: tick listener panicked; continuing");
        }
    }

    // Step 3: external handlers (legacy JS handler path; transition-only).
    // Cloned out under read lock then invoked without holding it.
    let (typed, global): (Vec<EventHandler>, Vec<EventHandler>) = {
        let guard = inner.read();
        let typed = guard
            .handlers
            .get(&event.event_type)
            .map(|m| m.values().cloned().collect())
            .unwrap_or_default();
        let global = guard.global_handlers.values().cloned().collect();
        (typed, global)
    };
    for handler in typed.iter().chain(global.iter()) {
        handler(event);
    }
}
