//! Sinks pick at compile time via `cfg(target_arch = "wasm32")`. Keep
//! BUILD.bazel deps mirroring that — tracing-appender doesn't link on
//! wasm (no filesystem) and tracing-wasm doesn't link on native.

mod config;
mod host_bridge;
mod init;
mod panic;
mod sinks;

pub use config::{FileSink, LogConfig};
pub use host_bridge::log_event;
pub use init::{init, LogError};
pub use panic::install_panic_hook;
