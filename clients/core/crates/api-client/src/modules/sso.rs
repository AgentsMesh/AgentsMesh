use crate::ApiClient;
use crate::connect_call::connect_call;
use crate::error::ApiError;
use agentsmesh_types::proto_sso_v1 as sso_proto;

// =============================================================================
// Connect-RPC (binary wire). See proto-naming-conventions.md §2.5.
// =============================================================================
//
// These methods call the Connect handlers in
// backend/internal/api/connect/sso/. Procedure paths derive from
// `proto.sso.v1.SSOService.<Method>` (conventions §12).
//
// USER-SCOPED + PUBLIC: no org_slug, no bearer token. `connect_call`
// silently omits the Authorization header when the auth store is
// empty, which is the expected pre-login state for these RPCs.

impl ApiClient {
    pub async fn sso_discover_connect(
        &self,
        req: &sso_proto::DiscoverRequest,
    ) -> Result<sso_proto::DiscoverResponse, ApiError> {
        connect_call(
            self,
            "/proto.sso.v1.SSOService/Discover",
            req,
        )
        .await
    }

    pub async fn sso_ldap_auth_connect(
        &self,
        req: &sso_proto::LdapAuthRequest,
    ) -> Result<sso_proto::LdapAuthResponse, ApiError> {
        connect_call(
            self,
            "/proto.sso.v1.SSOService/LdapAuth",
            req,
        )
        .await
    }
}
