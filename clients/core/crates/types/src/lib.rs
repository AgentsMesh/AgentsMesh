mod agent;
mod agentpod_settings;
mod apikey;
mod apikey_proto;
mod auth;
mod autopilot;
mod billing;
mod billing_proto;
mod binding;
mod blockstore;
mod channel;
mod common;
mod enums;
mod extension;
mod extension_proto;
mod file_upload;
mod grant;
mod invitation;
mod loop_requests;
mod loop_types;
mod mesh;
mod message;
mod notification;
mod organization;
mod pod;
mod promocode;
mod repository;
mod runner;
mod runner_proto;
mod service_error;
mod sso;
mod support_ticket;
mod ticket;
mod ticket_requests;
mod token_usage;
mod user_credential;

pub use agent::*;
pub use agentpod_settings::*;
pub use apikey::*;
pub use auth::*;
pub use autopilot::*;
pub use billing::*;

/// Connect-RPC binary-wire DTOs for `proto.billing.v1`. Re-exported as a
/// distinct module so the legacy serde `Subscription` (REST path) and the
/// prost `Subscription` (Connect path) coexist during the dual-track
/// migration window without name collisions.
pub mod proto_billing_v1 {
    pub use super::billing_proto::*;
}
pub use binding::*;
pub use blockstore::*;
pub use channel::*;
pub use common::*;
pub use enums::*;
pub use extension::*;

/// Connect-RPC binary-wire DTOs for `proto.extension.v1`. Re-exported as a
/// distinct module so the legacy serde `SkillRegistry` (REST path) and the
/// prost `SkillRegistry` (Connect path) coexist during the dual-track
/// migration window without name collisions.
pub mod proto_extension_v1 {
    pub use super::extension_proto::*;
}

/// Connect-RPC binary-wire DTOs for `proto.apikey.v1`. Re-exported as a
/// distinct module so the legacy serde `ApiKey` (REST path) and the
/// prost `ApiKey` (Connect path) coexist during dual-track migration.
pub mod proto_apikey_v1 {
    pub use super::apikey_proto::*;
}

/// Connect-RPC binary-wire DTOs for `proto.runner_api.v1`. Re-exported as
/// `proto_runner_api_v1` so the legacy serde `Runner` (REST path) and the
/// prost `Runner` (Connect path) coexist during the dual-track migration
/// window without name collisions.
pub mod proto_runner_api_v1 {
    pub use super::runner_proto::*;
}
pub use file_upload::*;
pub use grant::*;
pub use invitation::*;
pub use loop_requests::*;
pub use loop_types::*;
pub use mesh::*;
pub use message::*;
pub use notification::*;
pub use organization::*;
pub use pod::*;
pub use promocode::*;
pub use repository::*;
pub use runner::*;
pub use service_error::*;
pub use sso::*;
pub use support_ticket::*;
pub use ticket::*;
pub use ticket_requests::*;
pub use token_usage::*;
pub use user_credential::*;
