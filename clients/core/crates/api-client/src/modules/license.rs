// proto.license.v1.LicenseService + LicensePublicService Connect-RPC client
// bindings. Procedure paths derive from
// `proto.license.v1.<Service>.<Method>` (conventions §12). The REST surface
// is retired in this PR; Connect handlers in backend/internal/api/connect/
// license/ now own the data plane.
//
// Two services share this file because they share the same wire vocabulary
// (LicenseStatus + LicenseLimits). The auth distinction lives on the server
// — every call still goes through `connect_call`, which threads the bearer
// token when one exists. The public-service endpoints work with or without
// the token; the auth-required ones surface 401 → ApiError::AuthExpired.

use crate::ApiClient;
use crate::connect_call::connect_call;
use crate::error::ApiError;
use agentsmesh_types::proto_license_v1 as lp;

impl ApiClient {
    pub async fn activate_license_connect(
        &self,
        req: &lp::ActivateLicenseRequest,
    ) -> Result<lp::LicenseStatus, ApiError> {
        connect_call(
            self,
            "/proto.license.v1.LicenseService/ActivateLicense",
            req,
        )
        .await
    }

    pub async fn refresh_license_connect(
        &self,
        req: &lp::RefreshLicenseRequest,
    ) -> Result<lp::LicenseStatus, ApiError> {
        connect_call(self, "/proto.license.v1.LicenseService/RefreshLicense", req).await
    }

    pub async fn validate_license_connect(
        &self,
        req: &lp::ValidateLicenseRequest,
    ) -> Result<lp::ValidatedLicense, ApiError> {
        connect_call(self, "/proto.license.v1.LicenseService/ValidateLicense", req).await
    }

    pub async fn get_license_status_connect(
        &self,
        req: &lp::GetLicenseStatusRequest,
    ) -> Result<lp::LicenseStatus, ApiError> {
        connect_call(
            self,
            "/proto.license.v1.LicensePublicService/GetLicenseStatus",
            req,
        )
        .await
    }

    pub async fn get_license_limits_connect(
        &self,
        req: &lp::GetLicenseLimitsRequest,
    ) -> Result<lp::LicenseLimitsResponse, ApiError> {
        connect_call(
            self,
            "/proto.license.v1.LicensePublicService/GetLicenseLimits",
            req,
        )
        .await
    }

    pub async fn check_license_feature_connect(
        &self,
        req: &lp::CheckFeatureRequest,
    ) -> Result<lp::CheckFeatureResponse, ApiError> {
        connect_call(
            self,
            "/proto.license.v1.LicensePublicService/CheckFeature",
            req,
        )
        .await
    }
}
