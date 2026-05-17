//! AgentsMesh cross-platform logging.
//!
//! Hosts call [`init`] once at bootstrap with a [`LogConfig`]; thereafter all
//! `tracing::warn!`/`info!`/`error!` calls across the workspace land in the
//! configured sinks. Native targets get a rolling file appender plus stderr;
//! wasm32 gets the browser console via tracing-wasm. Hosts may also push
//! pre-formatted events from non-Rust callers via [`log_event`].
//!
//! Sinks are picked at compile time via `cfg(target_arch = "wasm32")` — the
//! BUILD.bazel deps gate must mirror that, since tracing-appender doesn't
//! link on wasm (no filesystem) and tracing-wasm doesn't link on native.

mod config;
mod host_bridge;
mod init;
mod panic;
mod sinks;

pub use config::{FileSink, LogConfig};
pub use host_bridge::log_event;
pub use init::{init, LogError};
pub use panic::install_panic_hook;
