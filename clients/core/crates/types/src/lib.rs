mod agent;
mod agent_proto;
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
mod blockstore_proto;
mod channel;
mod channel_proto;
mod common;
mod enums;
mod extension;
mod extension_market_proto;
mod extension_proto;
mod extension_repo_mcp_proto;
mod extension_repo_skill_proto;
mod file_upload;
mod grant;
mod grant_proto;
mod invitation;
mod invitation_proto;
mod loop_requests;
mod loop_types;
mod mesh;
mod message;
mod notification;
mod notification_proto;
mod organization;
mod org_proto;
mod pod;
mod pod_proto;
mod promocode;
mod promocode_proto;
mod repository;
mod repository_proto;
mod runner;
mod runner_proto;
mod service_error;
mod sso;
mod sso_proto;
mod support_ticket;
mod support_ticket_proto;
mod ticket;
mod ticket_proto;
mod ticket_relations_proto;
mod ticket_requests;
mod token_usage;
mod user_credential;
mod user_credential_proto;
mod user_proto;

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

/// Connect-RPC binary-wire DTOs for `proto.blockstore.v1`. Re-exported as a
/// distinct module so the legacy serde `Block` / `BlockRef` / `BlockOp`
/// (REST path) and the prost mirrors (Connect path) coexist during the
/// dual-track migration window without name collisions.
pub mod proto_blockstore_v1 {
    pub use super::blockstore_proto::*;
}
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
///
/// All sub-services of the extension domain (skill_registry / market /
/// repo_skill / repo_mcp) share the same proto package, so their prost
/// mirrors are unified under this single module — `use proto_extension_v1::*`
/// pulls in every prost type for the domain without per-sub-service imports.
pub mod proto_extension_v1 {
    pub use super::extension_market_proto::*;
    pub use super::extension_proto::*;
    pub use super::extension_repo_mcp_proto::*;
    pub use super::extension_repo_skill_proto::*;
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

/// Connect-RPC binary-wire DTOs for `proto.user.v1`. UserService is the
/// caller's profile / identity / search surface (REST `/api/v1/users/me*`
/// + `/search`). All RPCs are user-scoped — no org_slug, conventions
/// §3.5 exception #1. Coexists with the legacy serde `User` (REST path,
/// in `auth.rs`) for the dual-track migration window.
pub mod proto_user_v1 {
    pub use super::user_proto::*;
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

/// Connect-RPC binary-wire DTOs for `proto.agent.v1`. Bundles `AgentService`
/// (org-scoped agent catalog + custom CRUD) and `UserAgentConfigService`
/// (user-scoped personal config) because they share the same proto package.
/// Coexists with the legacy serde `Agent` / `UserAgentConfig` for the
/// dual-track window. AgentListResponse is the §9 multi-field exception
/// (builtin_agents + custom_agents kept separate per existing REST shape);
/// UserAgentConfigListResponse keeps the `configs` field (§9 exception #2,
/// sub-resource envelope).
pub mod proto_agent_v1 {
    pub use super::agent_proto::*;
}

/// Connect-RPC binary-wire DTOs for `proto.invitation.v1`. Bundles three
/// services (org-scoped `InvitationService`, invitee-scoped
/// `UserInvitationService`, public `PublicInvitationService`) because they
/// address the same `invitations` table via orthogonal scopes. Coexists
/// with the legacy serde `Invitation` for the dual-track window.
pub mod proto_invitation_v1 {
    pub use super::invitation_proto::*;
}

/// Connect-RPC binary-wire DTOs for `proto.support_ticket.v1`. User-scoped
/// service (no org_slug, conventions §3.5 exception #1). Coexists with the
/// legacy serde `SupportTicket` / `SupportTicketMessage` /
/// `SupportTicketListResponse` for the dual-track window. The list
/// envelope unified to `{items, total, limit, offset}` — the TS adapter
/// remaps to the legacy `{data, total, page, page_size, total_pages}`
/// surface so renderer-side call sites can flip one at a time.
pub mod proto_support_ticket_v1 {
    pub use super::support_ticket_proto::*;
}

/// Connect-RPC binary-wire DTOs for `proto.promocode.v1`. Re-exported as a
/// distinct module so the legacy serde `ValidatePromoRequest` etc. (REST
/// path) and the prost mirrors (Connect path) coexist during the dual-track
/// migration window without name collisions. Org-scoped surface only —
/// admin CRUD over promo codes stays on REST during this migration.
pub mod proto_promocode_v1 {
    pub use super::promocode_proto::*;
}

/// Connect-RPC binary-wire DTOs for `proto.auth.v1`. AuthService is public
/// (login / register / refresh / oauth / verify-email / forgot- and
/// reset-password — no token required); AuthSessionService.Logout is
/// authenticated. Coexists with the legacy serde `AuthSession` / `User` /
/// `LoginRequest` / `RegisterRequest` in `auth.rs` for the dual-track window.
pub mod proto_auth_v1 {
    pub use super::auth_proto::*;
}

/// Connect-RPC binary-wire DTOs for `proto.sso.v1`. PUBLIC service (no
/// auth, conventions §3.5 exception #1) — the login page hits Discover
/// before the user has a bearer token, and LdapAuth issues that token.
/// Coexists with the legacy serde `SSOConfig` / `LdapAuthRequest` /
/// `SSODiscoverResponse` in `auth.rs` + `sso.rs` for the dual-track window.
pub mod proto_sso_v1 {
    pub use super::sso_proto::*;
}

/// Connect-RPC binary-wire DTOs for `proto.notification.v1`. Org-scoped
/// service — every request carries `org_slug = 1`. Coexists with the
/// legacy serde `NotificationPreference` / `SetNotificationPreferenceRequest`
/// / `NotificationPreferenceListResponse` in `notification.rs` for the
/// dual-track migration window. The legacy REST envelope
/// `{preferences: [...]}` lives only in the TS adapter; the proto wire is
/// the §8 uniform `{items, total, limit, offset}`. SetPreference returns
/// the entity directly (§9) where REST returns `{status:"ok"}` — drift
/// reconciled inline (the TS adapter discards the response, so this is
/// backwards compatible at the call-site level).
pub mod proto_notification_v1 {
    pub use super::notification_proto::*;
}

/// Connect-RPC binary-wire DTOs for `proto.grant.v1`. Single
/// `GrantService` covers all three resource types (pod / runner /
/// repository) — the REST layer split was policy-only, the wire was
/// already unified. Coexists with the legacy serde `ResourceGrant` for
/// the dual-track window.
pub mod proto_grant_v1 {
    pub use super::grant_proto::*;
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
