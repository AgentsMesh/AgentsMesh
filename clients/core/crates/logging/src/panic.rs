use std::sync::Once;

static HOOK: Once = Once::new();

// Routes Rust panics through `tracing::error!` so they land in the same
// sinks as regular logs. The previous hook (default panic handler) is still
// invoked after we log — keeping abort/stderr behaviour intact for native
// crashes while ensuring the panic message reaches the persisted log file.
pub fn install_panic_hook() {
    HOOK.call_once(|| {
        let prev = std::panic::take_hook();
        std::panic::set_hook(Box::new(move |info| {
            let location = info
                .location()
                .map(|l| format!("{}:{}:{}", l.file(), l.line(), l.column()))
                .unwrap_or_else(|| "<unknown>".into());
            let payload = info
                .payload()
                .downcast_ref::<&str>()
                .copied()
                .or_else(|| info.payload().downcast_ref::<String>().map(|s| s.as_str()))
                .unwrap_or("<non-string panic payload>");
            tracing::error!(target: "panic", location = %location, "panic: {}", payload);
            prev(info);
        }));
    });
}
