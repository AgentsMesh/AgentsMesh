use crate::ApiClient;
use crate::error::ApiError;
use agentsmesh_types::{
    ApplyOpsRequest, ApplyOpsResult, Block, BlockOp, BlockRef, ChildrenResult,
    SearchHit, SemanticSearchRequest, Workspace,
};
use serde::Deserialize;

#[derive(Debug, Deserialize)]
struct WorkspacesWrapper {
    #[serde(default)]
    workspaces: Vec<Workspace>,
}

#[derive(Debug, Deserialize)]
struct OpsWrapper {
    #[serde(default)]
    ops: Vec<BlockOp>,
}

#[derive(Debug, Deserialize)]
struct BlocksWrapper {
    #[serde(default)]
    blocks: Vec<Block>,
}

#[derive(Debug, Deserialize)]
struct BacklinksWrapper {
    #[serde(default)]
    refs: Vec<BlockRef>,
}

#[derive(Debug, Deserialize)]
struct HitsWrapper {
    #[serde(default)]
    hits: Vec<SearchHit>,
}

impl ApiClient {
    pub async fn blocks_apply_ops(&self, req: &ApplyOpsRequest) -> Result<ApplyOpsResult, ApiError> {
        self.post(&self.org_path("/blocks/ops"), req).await
    }

    pub async fn blocks_list_workspaces(&self) -> Result<Vec<Workspace>, ApiError> {
        let w: WorkspacesWrapper = self.get(&self.org_path("/blocks/workspaces")).await?;
        Ok(w.workspaces)
    }

    pub async fn blocks_ensure_default_workspace(&self) -> Result<Workspace, ApiError> {
        self.post(&self.org_path("/blocks/workspaces/default"), &serde_json::json!({})).await
    }

    pub async fn blocks_get_block(&self, id: &str) -> Result<Block, ApiError> {
        let enc = urlencoding::encode(id);
        self.get(&self.org_path(&format!("/blocks/{enc}"))).await
    }

    pub async fn blocks_list_children(&self, id: &str, rel: &str) -> Result<ChildrenResult, ApiError> {
        let enc = urlencoding::encode(id);
        let relenc = urlencoding::encode(rel);
        self.get(&self.org_path(&format!("/blocks/{enc}/children?rel={relenc}"))).await
    }

    pub async fn blocks_list_backlinks(&self, id: &str) -> Result<Vec<BlockRef>, ApiError> {
        let enc = urlencoding::encode(id);
        let w: BacklinksWrapper = self.get(&self.org_path(&format!("/blocks/{enc}/backlinks"))).await?;
        Ok(w.refs)
    }

    pub async fn blocks_get_subtree(
        &self, workspace_id: &str, root_id: &str, max_depth: u32,
    ) -> Result<ChildrenResult, ApiError> {
        let ws = urlencoding::encode(workspace_id);
        let root = urlencoding::encode(root_id);
        self.get(&self.org_path(&format!(
            "/blocks/workspaces/{ws}/subtree?root={root}&max_depth={max_depth}",
        ))).await
    }

    pub async fn blocks_catchup_ops(
        &self, workspace_id: &str, after: i64, limit: u32,
    ) -> Result<Vec<BlockOp>, ApiError> {
        let ws = urlencoding::encode(workspace_id);
        let w: OpsWrapper = self.get(&self.org_path(&format!(
            "/blocks/workspaces/{ws}/ops?after={after}&limit={limit}",
        ))).await?;
        Ok(w.ops)
    }

    pub async fn blocks_list_type_defs(&self, workspace_id: &str) -> Result<Vec<Block>, ApiError> {
        let ws = urlencoding::encode(workspace_id);
        let w: BlocksWrapper = self.get(&self.org_path(&format!("/blocks/workspaces/{ws}/type-defs"))).await?;
        Ok(w.blocks)
    }

    pub async fn blocks_semantic_search(
        &self, workspace_id: &str, req: &SemanticSearchRequest,
    ) -> Result<Vec<SearchHit>, ApiError> {
        let ws = urlencoding::encode(workspace_id);
        let w: HitsWrapper = self.post(&self.org_path(&format!("/blocks/workspaces/{ws}/search")), req).await?;
        Ok(w.hits)
    }
}
