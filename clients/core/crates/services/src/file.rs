use std::sync::Arc;

use agentsmesh_api_client::ApiClient;
use agentsmesh_types::proto_file_v1 as fp;
use prost::Message;

pub struct FileService {
    client: Arc<ApiClient>,
}

impl FileService {
    pub fn new(client: Arc<ApiClient>) -> Self {
        Self { client }
    }

    pub async fn upload_file(
        &self, file_data: Vec<u8>, filename: &str, content_type: &str,
    ) -> Result<String, String> {
        // Connect-RPC presign → S3 PUT → return public get_url. Web/Desktop
        // still hand multipart bytes to this entrypoint because the browser
        // upload path is two-leg (presign + raw PUT).
        let req = fp::PresignUploadRequest {
            org_slug: self.client.current_org_slug(),
            filename: filename.to_string(),
            content_type: content_type.to_string(),
            size: file_data.len() as i64,
        };
        tracing::info!(target: "file", org_slug = %req.org_slug, content_type = %req.content_type, size = req.size, "upload file");
        let resp = self.client.presign_upload_connect(&req).await.map_err(crate::wire)?;

        self.client.put_raw_bytes(&resp.put_url, content_type, file_data)
            .await.map_err(crate::wire)?;
        Ok(resp.get_url)
    }

    pub async fn presign_upload_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = fp::PresignUploadRequest::decode(request_bytes)
            .map_err(|e| format!("decode presign_upload request: {e}"))?;
        tracing::info!(target: "file", org_slug = %req.org_slug, content_type = %req.content_type, size = req.size, "presign upload");
        let resp = self.client.presign_upload_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }
}
