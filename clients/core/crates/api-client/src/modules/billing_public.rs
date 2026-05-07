use crate::ApiClient;
use crate::error::ApiError;
use agentsmesh_types::*;

impl ApiClient {
    pub async fn get_public_pricing(&self) -> Result<PublicPricingResponse, ApiError> {
        self.public_get("/api/v1/config/pricing").await
    }

    pub async fn get_public_deployment_info(&self) -> Result<DeploymentInfo, ApiError> {
        self.public_get("/api/v1/config/deployment").await
    }
}
