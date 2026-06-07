use std::sync::Arc;

use agentsmesh_api_client::ApiClient;
use agentsmesh_types::proto_notification_v1 as np;
use prost::Message;

pub struct NotificationService {
    client: Arc<ApiClient>,
}

impl NotificationService {
    pub fn new(client: Arc<ApiClient>) -> Self {
        Self { client }
    }

    // -------- Connect-RPC (binary wire) --------
    //
    // Two lanes — request bytes in, response bytes out. The TS adapter
    // (`notificationConnect.ts`) is the only caller.

    pub async fn list_preferences_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = np::ListPreferencesRequest::decode(request_bytes)
            .map_err(|e| format!("decode list_preferences request: {e}"))?;
        tracing::debug!(target: "notification", org_slug = %req.org_slug, "list preferences");
        let resp = self.client.list_notification_preferences_connect(&req)
            .await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn set_preference_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = np::SetPreferenceRequest::decode(request_bytes)
            .map_err(|e| format!("decode set_preference request: {e}"))?;
        tracing::info!(target: "notification", org_slug = %req.org_slug, source = %req.source, is_muted = req.is_muted, "set preference");
        let resp = self.client.set_notification_preference_connect(&req)
            .await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }
}
