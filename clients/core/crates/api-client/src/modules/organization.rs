use crate::ApiClient;
use crate::error::ApiError;
use agentsmesh_types::*;

impl ApiClient {
    pub async fn list_organizations(&self) -> Result<OrganizationListResponse, ApiError> {
        self.get("/api/v1/orgs").await
    }

    pub async fn get_organization(&self, slug: &str) -> Result<Organization, ApiError> {
        self.get_resource(&format!("/api/v1/orgs/{slug}"), "organization").await
    }

    pub async fn create_organization(
        &self,
        data: &CreateOrganizationRequest,
    ) -> Result<Organization, ApiError> {
        self.post_resource("/api/v1/orgs", data, "organization").await
    }

    // Personal workspace: server derives slug from users.username via
    // slugkit.Sanitize; callers MUST NOT construct the slug client-side.
    // Use this for onboarding "Quick Start" instead of create_organization.
    pub async fn create_personal_organization(&self) -> Result<Organization, ApiError> {
        self.post_resource("/api/v1/orgs/personal", &serde_json::json!({}), "organization").await
    }

    pub async fn update_organization(
        &self,
        slug: &str,
        data: &UpdateOrganizationRequest,
    ) -> Result<Organization, ApiError> {
        self.put_resource(&format!("/api/v1/orgs/{slug}"), data, "organization").await
    }

    pub async fn delete_organization(&self, slug: &str) -> Result<EmptyResponse, ApiError> {
        self.delete(&format!("/api/v1/orgs/{slug}")).await
    }

    pub async fn list_org_members(&self, slug: &str) -> Result<MemberListResponse, ApiError> {
        self.get(&format!("/api/v1/orgs/{slug}/members")).await
    }

    pub async fn invite_org_member(
        &self,
        slug: &str,
        data: &InviteMemberRequest,
    ) -> Result<OrgMember, ApiError> {
        self.post(&format!("/api/v1/orgs/{slug}/members"), data)
            .await
    }

    pub async fn remove_org_member(
        &self,
        slug: &str,
        user_id: i64,
    ) -> Result<EmptyResponse, ApiError> {
        self.delete(&format!("/api/v1/orgs/{slug}/members/{user_id}"))
            .await
    }

    pub async fn update_org_member_role(
        &self,
        slug: &str,
        user_id: i64,
        data: &UpdateMemberRoleRequest,
    ) -> Result<OrgMember, ApiError> {
        self.put(
            &format!("/api/v1/orgs/{slug}/members/{user_id}"),
            data,
        )
        .await
    }
}
