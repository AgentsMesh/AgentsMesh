// Host-side log entrypoint. Non-Rust callers (TS/Swift) format their own
// messages and hand them over here; we re-emit through `tracing` so the
// configured sinks (file, console, …) pick them up alongside native Rust
// log events.
//
// `target` is the caller's logical source (e.g. "renderer", "storeBurst");
// we route it as a structured field rather than as tracing's `target:` —
// tracing's target must be a `'static` string literal in the macro, and we
// want any user-supplied value to survive into the log record.
pub fn log_event(level: &str, target: &str, msg: &str) {
    match parse_level(level) {
        tracing::Level::ERROR => {
            tracing::error!(target: "host", source = %target, "{}", msg)
        }
        tracing::Level::WARN => {
            tracing::warn!(target: "host", source = %target, "{}", msg)
        }
        tracing::Level::INFO => {
            tracing::info!(target: "host", source = %target, "{}", msg)
        }
        tracing::Level::DEBUG => {
            tracing::debug!(target: "host", source = %target, "{}", msg)
        }
        tracing::Level::TRACE => {
            tracing::trace!(target: "host", source = %target, "{}", msg)
        }
    }
}

fn parse_level(s: &str) -> tracing::Level {
    match s.to_ascii_lowercase().as_str() {
        "error" => tracing::Level::ERROR,
        "warn" | "warning" => tracing::Level::WARN,
        "debug" => tracing::Level::DEBUG,
        "trace" => tracing::Level::TRACE,
        _ => tracing::Level::INFO,
    }
}
