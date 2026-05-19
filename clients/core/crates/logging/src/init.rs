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

fn parse_filter(config: &LogConfig) -> Result<EnvFilter, LogError> {
    EnvFilter::try_new(&config.level).map_err(|e| LogError::InvalidLevel(e.to_string()))
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
}
