use napi_derive::napi;
use crate::{AppState, err};

#[napi]
impl AppState {
    #[napi]
    pub async fn blockstore_apply_ops(&self, req_json: String) -> napi::Result<String> {
        let svc = self.blockstore.lock().await;
        svc.apply_ops(&req_json).await.map_err(err)
    }

    #[napi]
    pub async fn blockstore_list_workspaces(&self) -> napi::Result<String> {
        let svc = self.blockstore.lock().await;
        svc.list_workspaces().await.map_err(err)
    }

    #[napi]
    pub async fn blockstore_ensure_default_workspace(&self) -> napi::Result<String> {
        let svc = self.blockstore.lock().await;
        svc.ensure_default_workspace().await.map_err(err)
    }

    #[napi]
    pub async fn blockstore_load_subtree(&self, workspace_id: String, root_id: String) -> napi::Result<()> {
        let svc = self.blockstore.lock().await;
        svc.load_subtree(&workspace_id, &root_id).await.map_err(err)
    }

    #[napi]
    pub async fn blockstore_load_type_defs(&self, workspace_id: String) -> napi::Result<()> {
        let svc = self.blockstore.lock().await;
        svc.load_type_defs(&workspace_id).await.map_err(err)
    }

    #[napi]
    pub async fn blockstore_catchup(&self, workspace_id: String) -> napi::Result<()> {
        let svc = self.blockstore.lock().await;
        svc.catchup(&workspace_id).await.map_err(err)
    }

    #[napi]
    pub async fn blockstore_semantic_search(&self, workspace_id: String, req_json: String) -> napi::Result<String> {
        let svc = self.blockstore.lock().await;
        svc.semantic_search(&workspace_id, &req_json).await.map_err(err)
    }

    #[napi]
    pub async fn blockstore_apply_remote_op(&self, op_json: String) -> napi::Result<()> {
        let svc = self.blockstore.lock().await;
        svc.apply_remote_op(&op_json).map_err(err)
    }

    #[napi]
    pub async fn blockstore_workspaces_json(&self) -> napi::Result<String> {
        let svc = self.blockstore.lock().await;
        Ok(svc.workspaces_json())
    }

    #[napi]
    pub async fn blockstore_get_block_json(&self, id: String) -> napi::Result<Option<String>> {
        let svc = self.blockstore.lock().await;
        Ok(svc.get_block_json(&id))
    }

    #[napi]
    pub async fn blockstore_list_children_json(&self, parent_id: String) -> napi::Result<String> {
        let svc = self.blockstore.lock().await;
        Ok(svc.list_children_json(&parent_id))
    }

    #[napi]
    pub async fn blockstore_list_backlinks_json(&self, target_id: String) -> napi::Result<String> {
        let svc = self.blockstore.lock().await;
        Ok(svc.list_backlinks_json(&target_id))
    }

    #[napi]
    pub async fn blockstore_type_defs_json(&self, workspace_id: String) -> napi::Result<String> {
        let svc = self.blockstore.lock().await;
        Ok(svc.type_defs_json(&workspace_id))
    }

    #[napi]
    pub async fn blockstore_last_op_id(&self, workspace_id: String) -> napi::Result<i64> {
        let svc = self.blockstore.lock().await;
        Ok(svc.last_op_id(&workspace_id))
    }

    #[napi]
    pub async fn blockstore_set_last_op_id(&self, workspace_id: String, id: i64) -> napi::Result<()> {
        let svc = self.blockstore.lock().await;
        svc.set_last_op_id(&workspace_id, id);
        Ok(())
    }

    #[napi]
    pub async fn blockstore_blocks_json(&self) -> napi::Result<String> {
        let svc = self.blockstore.lock().await;
        Ok(svc.blocks_json())
    }

    #[napi]
    pub async fn blockstore_refs_json(&self) -> napi::Result<String> {
        let svc = self.blockstore.lock().await;
        Ok(svc.refs_json())
    }

    #[napi]
    pub async fn blockstore_nest_children_json(&self) -> napi::Result<String> {
        let svc = self.blockstore.lock().await;
        Ok(svc.nest_children_json())
    }

    #[napi]
    pub async fn blockstore_backlinks_json(&self) -> napi::Result<String> {
        let svc = self.blockstore.lock().await;
        Ok(svc.backlinks_json())
    }

    #[napi]
    pub async fn blockstore_last_op_ids_json(&self) -> napi::Result<String> {
        let svc = self.blockstore.lock().await;
        Ok(svc.last_op_ids_json())
    }

}
