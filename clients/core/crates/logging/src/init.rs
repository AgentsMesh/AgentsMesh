use std::sync::OnceLock;
use thiserror::Error;
use tracing_subscriber::EnvFilter;

use crate::config::LogConfig;

#[derive(Debug, Error)]
pub enum LogError {
    #[error("invalid log level: {0}")]
    InvalidLevel(String),
    #[error("file sink io error: {0}")]
    Io(#[from] std::io::Error),
}

// Guards the global subscriber install — tracing's global subscriber slot
// accepts at most one installer. Subsequent `init` calls become no-ops so
// hosts can call us from multiple bootstrap paths (StrictMode double-render,
// reconnect retry, etc.) without panicking.
static INSTALLED: OnceLock<()> = OnceLock::new();

#[cfg(not(target_arch = "wasm32"))]
static FILE_GUARD: OnceLock<tracing_appender::non_blocking::WorkerGuard> = OnceLock::new();

#[cfg(not(target_arch = "wasm32"))]
pub fn init(config: LogConfig) -> Result<(), LogError> {
    if INSTALLED.get().is_some() {
        return Ok(());
    }
    let filter = parse_filter(&config)?;
    if let Some(guard) = crate::sinks::file::install(&config, filter)? {
        let _ = FILE_GUARD.set(guard);
    }
    let _ = INSTALLED.set(());
    Ok(())
}

#[cfg(target_arch = "wasm32")]
pub fn init(config: LogConfig) -> Result<(), LogError> {
    if INSTALLED.get().is_some() {
        return Ok(());
    }
    let filter = parse_filter(&config)?;
    crate::sinks::wasm::install(filter)?;
    let _ = INSTALLED.set(());
    Ok(())
}

// Third-party crates flood debug/trace (hyper wire dumps, rustls handshake,
// tokio scheduler) and bury our own logs. A bare debug/trace level is scoped to
// our crates and clamps these to info. A directive already targeting crates
// (`=` present) or listing several (`,` present) is honored verbatim — RUST_LOG
// / power-user overrides keep full control.
const NOISY_CRATES: &[&str] = &[
    "hyper", "h2", "rustls", "tokio", "tungstenite", "tokio_tungstenite", "reqwest", "tower", "mio",
    "want", "tonic",
];

fn scoped_spec(level: &str) -> String {
    let level = level.trim();
    if !matches!(level, "debug" | "trace") {
        return level.to_string();
    }
    let mut s = String::with_capacity(level.len() + NOISY_CRATES.len() * 12);
    s.push_str(level);
    for c in NOISY_CRATES {
        s.push(',');
        s.push_str(c);
        s.push_str("=info");
    }
    s
}

fn parse_filter(config: &LogConfig) -> Result<EnvFilter, LogError> {
    EnvFilter::try_new(scoped_spec(&config.level)).map_err(|e| LogError::InvalidLevel(e.to_string()))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn init_is_idempotent() {
        // Either both succeed or the first one wins and the second short-circuits.
        let first = init(LogConfig::console("info"));
        let second = init(LogConfig::console("debug"));
        assert!(first.is_ok());
        assert!(second.is_ok());
    }

    #[test]
    fn invalid_filter_rejected() {
        // EnvFilter is permissive — bare unknown words are accepted as
        // target names with the default level. Use unambiguously-malformed
        // syntax instead so the rejection path actually fires.
        let cfg = LogConfig::console("=invalid=garbage=");
        let err = parse_filter(&cfg).err();
        assert!(matches!(err, Some(LogError::InvalidLevel(_))));
    }

    #[test]
    fn bare_debug_scopes_noisy_crates() {
        let spec = scoped_spec("debug");
        assert!(spec.starts_with("debug,"), "spec: {spec}");
        assert!(spec.contains("hyper=info"), "spec: {spec}");
        assert!(spec.contains("rustls=info"), "spec: {spec}");
        // our own crates keep the bare debug level — no clamp injected for them
        assert!(!spec.contains("agentsmesh"), "spec: {spec}");
    }

    #[test]
    fn bare_info_left_verbatim() {
        assert_eq!(scoped_spec("info"), "info");
    }

    #[test]
    fn explicit_directive_left_verbatim() {
        // RUST_LOG / power-user style: not a bare level, so pass through untouched
        assert_eq!(scoped_spec("agentsmesh_relay=debug"), "agentsmesh_relay=debug");
        assert_eq!(scoped_spec("debug,hyper=trace"), "debug,hyper=trace");
    }
}
