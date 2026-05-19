use std::sync::Arc;

use agentsmesh_api_client::ApiClient;

pub struct UserCredentialService {
    client: Arc<ApiClient>,
}

impl UserCredentialService {
    pub fn new(client: Arc<ApiClient>) -> Self {
        Self { client }
    }

    /// Access the underlying ApiClient — used by sibling Connect-RPC impl
    /// blocks in user_credential_*_connect.rs.
    pub(crate) fn client(&self) -> &ApiClient {
        &self.client
    }
}
