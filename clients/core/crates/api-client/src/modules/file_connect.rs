use crate::ApiClient;
use crate::connect_call::connect_call;
use crate::error::ApiError;
use agentsmesh_types::proto_file_v1 as fp;

// =============================================================================
// Connect-RPC (binary wire). See proto-naming-conventions.md §2.5.
// =============================================================================

impl ApiClient {
    pub async fn presign_upload_connect(
        &self,
        req: &fp::PresignUploadRequest,
    ) -> Result<fp::PresignUploadResponse, ApiError> {
        connect_call(
            self,
            "/proto.file.v1.FileService/PresignUpload",
            req,
        )
        .await
    }
}
