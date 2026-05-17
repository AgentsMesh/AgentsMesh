#![cfg(target_arch = "wasm32")]

use tracing_subscriber::{layer::SubscriberExt, util::SubscriberInitExt, EnvFilter, Registry};
use tracing_wasm::{WASMLayer, WASMLayerConfigBuilder};

use crate::init::LogError;

// Installs the wasm-bindgen console layer. There is no filesystem in the
// wasm sandbox; the only sink is the browser console. tracing-wasm formats
// each event with a `[level] target — message` prefix and routes by level
// to console.log/info/warn/error.
pub fn install(filter: EnvFilter) -> Result<(), LogError> {
    let cfg = WASMLayerConfigBuilder::new()
        .set_max_level(tracing::Level::TRACE)
        .build();
    let layer = WASMLayer::new(cfg);
    let _ = Registry::default().with(filter).with(layer).try_init();
    Ok(())
}
