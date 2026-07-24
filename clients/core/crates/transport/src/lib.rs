mod error;
mod message;
pub mod runtime;

#[cfg(any(target_arch = "wasm32", test))]
mod callback_owner;

#[cfg(not(target_arch = "wasm32"))]
mod native;
#[cfg(target_arch = "wasm32")]
mod wasm;

pub use error::TransportError;
pub use message::WsMessage;
pub use runtime::{timeout, BoxFuture, Elapsed, PlatformRuntime, Runtime, TaskHandle};

#[cfg(not(target_arch = "wasm32"))]
pub use native::{WebSocketConnection, WsReceiver, WsSender};
#[cfg(target_arch = "wasm32")]
pub use wasm::{WebSocketConnection, WsReceiver, WsSender};

#[cfg(test)]
mod error_tests;
#[cfg(test)]
mod message_tests;
#[cfg(test)]
mod native_tests;
