#[cfg(not(target_arch = "wasm32"))]
pub mod file;

#[cfg(target_arch = "wasm32")]
pub mod wasm;
