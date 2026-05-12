mod agent;
mod agentpod_settings;
mod agentpod_settings_proto;
mod apikey;
mod apikey_proto;
mod auth;
mod auth_proto;
mod autopilot;
mod billing;
mod billing_proto;
mod binding;
mod blockstore;
mod channel;
mod channel_proto;
mod common;
mod enums;
mod extension;
mod extension_proto;
mod file_upload;
mod grant;
mod invitation;
mod invitation_proto;
mod loop_requests;
mod loop_types;
mod mesh;
mod message;
mod notification;
mod organization;
mod org_proto;
mod pod;
mod pod_proto;
mod promocode;
mod repository;
mod repository_proto;
mod runner;
mod runner_proto;
mod service_error;
mod sso;
mod support_ticket;
mod ticket;
mod ticket_proto;
mod ticket_relations_proto;
mod ticket_requests;
mod token_usage;
mod user_credential;
mod user_credential_proto;

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

/// Connect-RPC binary-wire DTOs for `proto.channel.v1`. Re-exported as a
/// distinct module so the legacy serde `Channel` (REST path) and the prost
/// `Channel` (Connect path) coexist during the dual-track migration window
/// without name collisions.
pub mod proto_channel_v1 {
    pub use super::channel_proto::*;
}

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

/// Connect-RPC binary-wire DTOs for `proto.pod.v1`. Bundles both `PodService`
/// (org-scoped pod lifecycle) and `AgentPodSettingsService` (user-scoped
/// settings/providers) because they share the same proto package. Coexists
/// with the legacy serde `Pod` / `AgentPodSettings` for the dual-track window.
pub mod proto_pod_v1 {
    pub use super::agentpod_settings_proto::*;
    pub use super::pod_proto::*;
}

/// Connect-RPC binary-wire DTOs for `proto.repository.v1`. Re-exported as a
/// distinct module so the legacy serde `Repository` (REST path) and the
/// prost `Repository` (Connect path) coexist during the dual-track migration
/// window without name collisions. PR #329 / #342 / #343 reconciliation:
/// the proto SSOT carries all 19 backend fields the legacy DTO dropped.
pub mod proto_repository_v1 {
    pub use super::repository_proto::*;
}

/// Connect-RPC binary-wire DTOs for `proto.user_credential.v1`. Three
/// services share this module because they live in the same proto package
/// (GitCredential, AgentCredentialProfile, RepositoryProvider). All are
/// user-scoped — no org_slug, conventions §3.5 exception #1.
pub mod proto_user_credential_v1 {
    pub use super::user_credential_proto::*;
}

/// Connect-RPC binary-wire DTOs for `proto.ticket_relations.v1`. Bundles
/// relations / comments / commits / merge-requests under one module — they
/// share the `ticket_slug` lookup, so the proto file is one. Coexists with
/// the legacy serde `TicketRelation` / `TicketComment` / `TicketCommit` /
/// `MergeRequest` for the dual-track window. PR 986a38ca6 reconciliation:
/// the comment list envelope (`{items, total, limit, offset}`) survives
/// the wire — the adapter remaps to the legacy `{comments, ...}` shape.
pub mod proto_ticket_relations_v1 {
    pub use super::ticket_relations_proto::*;
}

/// Connect-RPC binary-wire DTOs for `proto.ticket.v1`. Re-exported as a
/// distinct module so the legacy serde `Ticket` / `Label` / `BoardColumn`
/// (REST path) and the prost mirrors (Connect path) coexist during the
/// dual-track migration window without name collisions. PR 986a38ca6
/// reconciliation: list envelope unified to `{items, total, limit,
/// offset}`; legacy `{tickets, ...}` shape lives only in the TS adapter.
pub mod proto_ticket_v1 {
    pub use super::ticket_proto::*;
}

/// Connect-RPC binary-wire DTOs for `proto.org.v1`. Re-exported as a distinct
/// module so the legacy serde `Organization` (REST path, in `auth.rs`) and the
/// prost `Organization` (Connect path) coexist during the dual-track migration
/// window without name collisions.
pub mod proto_org_v1 {
    pub use super::org_proto::*;
}

/// Connect-RPC binary-wire DTOs for `proto.invitation.v1`. Bundles three
/// services (org-scoped `InvitationService`, invitee-scoped
/// `UserInvitationService`, public `PublicInvitationService`) because they
/// address the same `invitations` table via orthogonal scopes. Coexists
/// with the legacy serde `Invitation` for the dual-track window.
pub mod proto_invitation_v1 {
    pub use super::invitation_proto::*;
}

/// Connect-RPC binary-wire DTOs for `proto.auth.v1`. AuthService is public
/// (login / register / refresh / oauth / verify-email / forgot- and
/// reset-password — no token required); AuthSessionService.Logout is
/// authenticated. Coexists with the legacy serde `AuthSession` / `User` /
/// `LoginRequest` / `RegisterRequest` in `auth.rs` for the dual-track window.
pub mod proto_auth_v1 {
    pub use super::auth_proto::*;
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
