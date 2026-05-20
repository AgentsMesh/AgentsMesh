use std::sync::Arc;

use agentsmesh_api_client::ApiClient;
use agentsmesh_types::*;

pub struct OrgApiService {
    client: Arc<ApiClient>,
}

impl OrgApiService {
    pub fn new(client: Arc<ApiClient>) -> Self {
        Self { client }
    }

    pub async fn list(&self) -> Result<String, String> {
        let resp = self.client.list_organizations().await.map_err(crate::wire)?;
        serde_json::to_string(&resp).map_err(crate::wire)
    }

    pub async fn get(&self, slug: &str) -> Result<String, String> {
        let resp = self.client.get_organization(slug).await.map_err(crate::wire)?;
        serde_json::to_string(&resp).map_err(crate::wire)
    }

    pub async fn create(&self, json: &str) -> Result<String, String> {
        let req: CreateOrganizationRequest = serde_json::from_str(json).map_err(crate::wire)?;
        let resp = self.client.create_organization(&req).await.map_err(crate::wire)?;
        serde_json::to_string(&resp).map_err(crate::wire)
    }

    pub async fn create_personal(&self) -> Result<String, String> {
        let resp = self.client.create_personal_organization().await.map_err(crate::wire)?;
        serde_json::to_string(&resp).map_err(crate::wire)
    }

    pub async fn update(&self, slug: &str, json: &str) -> Result<String, String> {
        let req: UpdateOrganizationRequest = serde_json::from_str(json).map_err(crate::wire)?;
        let resp = self.client
            .update_organization(slug, &req)
            .await.map_err(crate::wire)?;
        serde_json::to_string(&resp).map_err(crate::wire)
    }

    pub async fn delete(&self, slug: &str) -> Result<(), String> {
        self.client.delete_organization(slug).await.map_err(crate::wire)?;
        Ok(())
    }

    pub async fn list_members(&self, slug: &str) -> Result<String, String> {
        let resp = self.client.list_org_members(slug).await.map_err(crate::wire)?;
        serde_json::to_string(&resp).map_err(crate::wire)
    }

    pub async fn invite_member(&self, slug: &str, json: &str) -> Result<String, String> {
        let req: InviteMemberRequest = serde_json::from_str(json).map_err(crate::wire)?;
        let resp = self.client.invite_org_member(slug, &req).await.map_err(crate::wire)?;
        serde_json::to_string(&resp).map_err(crate::wire)
    }

    pub async fn remove_member(&self, slug: &str, user_id: i64) -> Result<(), String> {
        self.client
            .remove_org_member(slug, user_id)
            .await.map_err(crate::wire)?;
        Ok(())
    }

    pub async fn update_member_role(
        &self, slug: &str, user_id: i64, json: &str,
    ) -> Result<String, String> {
        let req: UpdateMemberRoleRequest = serde_json::from_str(json).map_err(crate::wire)?;
        let resp = self.client
            .update_org_member_role(slug, user_id, &req)
            .await.map_err(crate::wire)?;
        serde_json::to_string(&resp).map_err(crate::wire)
    }
}
