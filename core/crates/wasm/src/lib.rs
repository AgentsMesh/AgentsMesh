use std::sync::Arc;
use wasm_bindgen::prelude::*;

mod api;
mod auth;
mod events_manager;
mod js_bridge;
mod protocol;
mod relay_manager;
mod service_agent;
mod service_apikey;
mod service_auth_api;
mod service_autopilot;
mod service_billing;
mod service_binding;
mod service_blockstore;
mod service_channel;
mod service_extension;
mod service_file;
mod service_grant;
mod service_invitation;
mod service_loop;
mod service_mesh;
mod service_message;
mod service_notification;
mod service_org;
mod service_pod;
mod service_promocode;
mod service_repository;
mod service_runner;
mod service_sso;
mod service_support_ticket;
mod service_ticket;
mod service_ticket_relations;
mod service_token_usage;
mod service_user;
mod service_user_credential;
mod ws_transport;
mod state_acp;
mod state_app;
mod state_autopilot;
mod state_channel;
mod state_git;
mod state_loop;
mod state_mesh;
mod state_org;
mod state_pod;
mod state_repo;
mod state_runner;
mod state_ticket;
mod state_user;

pub use api::*;
pub use auth::*;
pub use events_manager::*;
pub use protocol::*;
pub use relay_manager::*;
pub use service_agent::*;
pub use service_apikey::*;
pub use service_auth_api::*;
pub use service_autopilot::*;
pub use service_billing::*;
pub use service_binding::*;
pub use service_blockstore::*;
pub use service_channel::*;
pub use service_extension::*;
pub use service_file::*;
pub use service_grant::*;
pub use service_invitation::*;
pub use service_loop::*;
pub use service_mesh::*;
pub use service_message::*;
pub use service_notification::*;
pub use service_org::*;
pub use service_pod::*;
pub use service_promocode::*;
pub use service_repository::*;
pub use service_runner::*;
pub use service_sso::*;
pub use service_support_ticket::*;
pub use service_ticket::*;
pub use service_ticket_relations::*;
pub use service_token_usage::*;
pub use service_user::*;
pub use service_user_credential::*;
pub use ws_transport::*;
pub use state_acp::*;
pub use state_app::*;
pub use state_autopilot::*;
pub use state_channel::*;
pub use state_git::*;
pub use state_loop::*;
pub use state_mesh::*;
pub use state_org::*;
pub use state_pod::*;
pub use state_repo::*;
pub use state_runner::*;
pub use state_ticket::*;

pub(crate) fn parse_status<T: serde::de::DeserializeOwned + Default>(s: &str) -> T {
    serde_json::from_value(serde_json::Value::String(s.to_string())).unwrap_or_default()
}

pub(crate) fn new_memory_backend() -> Arc<dyn agentsmesh_persistence::StorageBackend> {
    use agentsmesh_persistence::StorageBackend;
    let b = Arc::new(agentsmesh_persistence::InMemoryBackend::new());
    let _ = b.migrate();
    b
}

#[wasm_bindgen(start)]
pub fn init_panic_hook() {
    console_error_panic_hook::set_once();
}

#[wasm_bindgen]
pub fn version() -> String {
    env!("CARGO_PKG_VERSION").to_string()
}
