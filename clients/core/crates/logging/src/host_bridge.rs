// `target` is routed as a structured field rather than tracing's `target:`
// because tracing's target must be a `'static` string literal in the macro;
// user-supplied values would not survive otherwise.
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
