use crate::ApiClient;
use crate::error::ApiError;
use agentsmesh_types::*;

fn resource_plural(resource_type: &str) -> String {
    match resource_type {
        "pod" => "pods".into(),
        "runner" => "runners".into(),
        "repository" => "repositories".into(),
        other => format!("{other}s"),
    }
}

impl ApiClient {
    pub async fn list_resource_grants(
        &self,
        resource_type: &str,
        resource_id: &str,
    ) -> Result<ResourceGrantListResponse, ApiError> {
        let plural = resource_plural(resource_type);
        let path = self.org_path(&format!("/{plural}/{resource_id}/grants"));
        self.get(&path).await
    }

    pub async fn grant_resource_access(
        &self,
        resource_type: &str,
        resource_id: &str,
        req: &CreateResourceGrantRequest,
    ) -> Result<ResourceGrantResponse, ApiError> {
        let plural = resource_plural(resource_type);
        let path = self.org_path(&format!("/{plural}/{resource_id}/grants"));
        self.post(&path, req).await
    }

    pub async fn revoke_resource_grant(
        &self,
        resource_type: &str,
        resource_id: &str,
        grant_id: i64,
    ) -> Result<EmptyResponse, ApiError> {
        let plural = resource_plural(resource_type);
        let path = self.org_path(&format!("/{plural}/{resource_id}/grants/{grant_id}"));
        self.delete(&path).await
    }
}
