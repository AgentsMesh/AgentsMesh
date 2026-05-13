use std::sync::Arc;

use agentsmesh_api_client::ApiClient;
use agentsmesh_types::proto_user_v1 as user_proto;
use prost::Message;

pub struct UserApiService {
    client: Arc<ApiClient>,
}

impl UserApiService {
    pub fn new(client: Arc<ApiClient>) -> Self {
        Self { client }
    }

    // -------- Connect-RPC (binary wire) --------

    pub async fn get_me_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = user_proto::GetMeRequest::decode(request_bytes)
            .map_err(|e| format!("decode get_me request: {e}"))?;
        let resp = self.client.get_me_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn update_me_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = user_proto::UpdateMeRequest::decode(request_bytes)
            .map_err(|e| format!("decode update_me request: {e}"))?;
        let resp = self.client.update_me_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn change_password_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = user_proto::ChangePasswordRequest::decode(request_bytes)
            .map_err(|e| format!("decode change_password request: {e}"))?;
        let resp = self.client.change_password_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn list_identities_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = user_proto::ListIdentitiesRequest::decode(request_bytes)
            .map_err(|e| format!("decode list_identities request: {e}"))?;
        let resp = self.client.list_identities_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn delete_identity_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = user_proto::DeleteIdentityRequest::decode(request_bytes)
            .map_err(|e| format!("decode delete_identity request: {e}"))?;
        let resp = self.client.delete_identity_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn search_users_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = user_proto::SearchUsersRequest::decode(request_bytes)
            .map_err(|e| format!("decode search_users request: {e}"))?;
        let resp = self.client.search_users_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }
}
