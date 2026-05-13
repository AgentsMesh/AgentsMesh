// proto.blockstore.v1.BlockstoreService Connect-RPC client bindings.
// Procedure paths derive from `proto.blockstore.v1.BlockstoreService.<Method>`
// (conventions §12, §2.5). The legacy REST surface was retired; Connect
// handlers in backend/internal/api/connect/blockstore/ now own the data plane.

use crate::ApiClient;
use crate::connect_call::connect_call;
use crate::error::ApiError;
use agentsmesh_types::proto_blockstore_v1 as blockstore_proto;

impl ApiClient {
    pub async fn blockstore_apply_ops_connect(
        &self,
        req: &blockstore_proto::ApplyOpsRequest,
    ) -> Result<blockstore_proto::ApplyOpsResponse, ApiError> {
        connect_call(self, "/proto.blockstore.v1.BlockstoreService/ApplyOps", req).await
    }

    pub async fn blockstore_list_workspaces_connect(
        &self,
        req: &blockstore_proto::ListWorkspacesRequest,
    ) -> Result<blockstore_proto::ListWorkspacesResponse, ApiError> {
        connect_call(self, "/proto.blockstore.v1.BlockstoreService/ListWorkspaces", req).await
    }

    pub async fn blockstore_ensure_default_workspace_connect(
        &self,
        req: &blockstore_proto::EnsureDefaultWorkspaceRequest,
    ) -> Result<blockstore_proto::Workspace, ApiError> {
        connect_call(self, "/proto.blockstore.v1.BlockstoreService/EnsureDefaultWorkspace", req).await
    }

    pub async fn blockstore_create_workspace_connect(
        &self,
        req: &blockstore_proto::CreateWorkspaceRequest,
    ) -> Result<blockstore_proto::Workspace, ApiError> {
        connect_call(self, "/proto.blockstore.v1.BlockstoreService/CreateWorkspace", req).await
    }

    pub async fn blockstore_delete_workspace_connect(
        &self,
        req: &blockstore_proto::DeleteWorkspaceRequest,
    ) -> Result<blockstore_proto::DeleteWorkspaceResponse, ApiError> {
        connect_call(self, "/proto.blockstore.v1.BlockstoreService/DeleteWorkspace", req).await
    }

    pub async fn blockstore_get_block_connect(
        &self,
        req: &blockstore_proto::GetBlockRequest,
    ) -> Result<blockstore_proto::Block, ApiError> {
        connect_call(self, "/proto.blockstore.v1.BlockstoreService/GetBlock", req).await
    }

    pub async fn blockstore_list_children_connect(
        &self,
        req: &blockstore_proto::ListChildrenRequest,
    ) -> Result<blockstore_proto::ChildrenResult, ApiError> {
        connect_call(self, "/proto.blockstore.v1.BlockstoreService/ListChildren", req).await
    }

    pub async fn blockstore_list_backlinks_connect(
        &self,
        req: &blockstore_proto::ListBacklinksRequest,
    ) -> Result<blockstore_proto::ListBacklinksResponse, ApiError> {
        connect_call(self, "/proto.blockstore.v1.BlockstoreService/ListBacklinks", req).await
    }

    pub async fn blockstore_get_subtree_connect(
        &self,
        req: &blockstore_proto::GetSubtreeRequest,
    ) -> Result<blockstore_proto::ChildrenResult, ApiError> {
        connect_call(self, "/proto.blockstore.v1.BlockstoreService/GetSubtree", req).await
    }

    pub async fn blockstore_stream_ops_connect(
        &self,
        req: &blockstore_proto::StreamOpsRequest,
    ) -> Result<blockstore_proto::StreamOpsResponse, ApiError> {
        connect_call(self, "/proto.blockstore.v1.BlockstoreService/StreamOps", req).await
    }

    pub async fn blockstore_export_workspace_connect(
        &self,
        req: &blockstore_proto::ExportWorkspaceRequest,
    ) -> Result<blockstore_proto::ExportWorkspaceResponse, ApiError> {
        connect_call(self, "/proto.blockstore.v1.BlockstoreService/ExportWorkspace", req).await
    }

    pub async fn blockstore_list_type_defs_connect(
        &self,
        req: &blockstore_proto::ListTypeDefsRequest,
    ) -> Result<blockstore_proto::ListTypeDefsResponse, ApiError> {
        connect_call(self, "/proto.blockstore.v1.BlockstoreService/ListTypeDefs", req).await
    }

    pub async fn blockstore_get_block_at_connect(
        &self,
        req: &blockstore_proto::GetBlockAtRequest,
    ) -> Result<blockstore_proto::Block, ApiError> {
        connect_call(self, "/proto.blockstore.v1.BlockstoreService/GetBlockAt", req).await
    }

    pub async fn blockstore_semantic_search_connect(
        &self,
        req: &blockstore_proto::SemanticSearchRequest,
    ) -> Result<blockstore_proto::SemanticSearchResponse, ApiError> {
        connect_call(self, "/proto.blockstore.v1.BlockstoreService/SemanticSearch", req).await
    }

    pub async fn blockstore_memory_retrieve_connect(
        &self,
        req: &blockstore_proto::MemoryRetrieveRequest,
    ) -> Result<blockstore_proto::MemoryRetrieveResponse, ApiError> {
        connect_call(self, "/proto.blockstore.v1.BlockstoreService/MemoryRetrieve", req).await
    }
}
