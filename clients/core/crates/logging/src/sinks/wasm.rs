#![cfg(target_arch = "wasm32")]

use tracing_subscriber::{layer::SubscriberExt, util::SubscriberInitExt, EnvFilter, Registry};
use tracing_wasm::{WASMLayer, WASMLayerConfigBuilder};

use crate::init::LogError;

pub fn install(filter: EnvFilter) -> Result<(), LogError> {
    let cfg = WASMLayerConfigBuilder::new()
        .set_max_level(tracing::Level::TRACE)
        .build();
    let layer = WASMLayer::new(cfg);
    let _ = Registry::default().with(filter).with(layer).try_init();
    Ok(())
}
