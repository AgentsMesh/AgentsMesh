use std::sync::Arc;

use agentsmesh_api_client::ApiClient;
use agentsmesh_types::*;

pub struct GrantService {
    client: Arc<ApiClient>,
}

impl GrantService {
    pub fn new(client: Arc<ApiClient>) -> Self {
        Self { client }
    }

    pub async fn list(&self, resource_type: &str, resource_id: &str) -> Result<String, String> {
        let resp = self
            .client
            .list_resource_grants(resource_type, resource_id)
            .await
            .map_err(|e| e.to_string())?;
        serde_json::to_string(&resp).map_err(|e| e.to_string())
    }

    pub async fn grant(
        &self,
        resource_type: &str,
        resource_id: &str,
        user_id: i64,
    ) -> Result<String, String> {
        let req = CreateResourceGrantRequest { user_id };
        let resp = self
            .client
            .grant_resource_access(resource_type, resource_id, &req)
            .await
            .map_err(|e| e.to_string())?;
        serde_json::to_string(&resp).map_err(|e| e.to_string())
    }

    pub async fn revoke(
        &self,
        resource_type: &str,
        resource_id: &str,
        grant_id: i64,
    ) -> Result<(), String> {
        self.client
            .revoke_resource_grant(resource_type, resource_id, grant_id)
            .await
            .map_err(|e| e.to_string())?;
        Ok(())
    }
}
