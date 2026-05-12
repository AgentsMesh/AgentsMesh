use crate::ApiClient;
use crate::connect_call::connect_call;
use crate::error::ApiError;
use agentsmesh_types::*;
use agentsmesh_types::proto_apikey_v1 as apikey_proto;

// =============================================================================
// Connect-RPC (binary wire). See proto-naming-conventions.md §2.5.
// =============================================================================
//
// These methods call the Connect handlers in
// backend/internal/api/connect/apikey/. Procedure paths derive from
// `proto.apikey.v1.ApiKeyService.<Method>` (conventions §12).
//
// PR #345 lineage: `create_api_key_connect` returns
// `CreateApiKeyResponse {api_key, raw_key}` — the multi-field exception
// per conventions §9. The raw key is now structurally pinned to tag 2;
// the wrapper-stripping bug class cannot recur.

impl ApiClient {
    pub async fn list_api_keys_connect(
        &self,
        req: &apikey_proto::ListApiKeysRequest,
    ) -> Result<apikey_proto::ListApiKeysResponse, ApiError> {
        connect_call(
            self,
            "/proto.apikey.v1.ApiKeyService/ListApiKeys",
            req,
        )
        .await
    }

    pub async fn get_api_key_connect(
        &self,
        req: &apikey_proto::GetApiKeyRequest,
    ) -> Result<apikey_proto::ApiKey, ApiError> {
        connect_call(
            self,
            "/proto.apikey.v1.ApiKeyService/GetApiKey",
            req,
        )
        .await
    }

    pub async fn create_api_key_connect(
        &self,
        req: &apikey_proto::CreateApiKeyRequest,
    ) -> Result<apikey_proto::CreateApiKeyResponse, ApiError> {
        connect_call(
            self,
            "/proto.apikey.v1.ApiKeyService/CreateApiKey",
            req,
        )
        .await
    }

    pub async fn update_api_key_connect(
        &self,
        req: &apikey_proto::UpdateApiKeyRequest,
    ) -> Result<apikey_proto::ApiKey, ApiError> {
        connect_call(
            self,
            "/proto.apikey.v1.ApiKeyService/UpdateApiKey",
            req,
        )
        .await
    }

    pub async fn revoke_api_key_connect(
        &self,
        req: &apikey_proto::RevokeApiKeyRequest,
    ) -> Result<apikey_proto::RevokeApiKeyResponse, ApiError> {
        connect_call(
            self,
            "/proto.apikey.v1.ApiKeyService/RevokeApiKey",
            req,
        )
        .await
    }

    pub async fn delete_api_key_connect(
        &self,
        req: &apikey_proto::DeleteApiKeyRequest,
    ) -> Result<apikey_proto::DeleteApiKeyResponse, ApiError> {
        connect_call(
            self,
            "/proto.apikey.v1.ApiKeyService/DeleteApiKey",
            req,
        )
        .await
    }
}

// =============================================================================
// Legacy REST methods — preserved for dual-track migration.
// =============================================================================

impl ApiClient {
    pub async fn list_api_keys(&self) -> Result<ApiKeyListResponse, ApiError> {
        self.get(&self.org_path("/api-keys")).await
    }

    pub async fn get_api_key(&self, id: i64) -> Result<ApiKey, ApiError> {
        self.get_resource(&self.org_path(&format!("/api-keys/{id}")), "api_key").await
    }

    pub async fn create_api_key(
        &self,
        data: &CreateApiKeyRequest,
    ) -> Result<CreateApiKeyResponse, ApiError> {
        self.post(&self.org_path("/api-keys"), data).await
    }

    pub async fn update_api_key(
        &self,
        id: i64,
        data: &UpdateApiKeyRequest,
    ) -> Result<ApiKey, ApiError> {
        self.put_resource(&self.org_path(&format!("/api-keys/{id}")), data, "api_key").await
    }

    pub async fn delete_api_key(&self, id: i64) -> Result<EmptyResponse, ApiError> {
        self.delete(&self.org_path(&format!("/api-keys/{id}")))
            .await
    }

    pub async fn revoke_api_key(&self, id: i64) -> Result<EmptyResponse, ApiError> {
        self.post(
            &self.org_path(&format!("/api-keys/{id}/revoke")),
            &serde_json::json!({}),
        )
        .await
    }
}
