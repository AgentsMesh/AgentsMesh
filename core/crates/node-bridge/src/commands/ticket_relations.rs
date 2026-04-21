use napi_derive::napi;
use crate::{AppState, err};

#[napi]
impl AppState {
    #[napi]
    pub async fn ticket_relations_list_relations(&self, slug: String) -> napi::Result<String> {
        let svc = self.ticket_relations.lock().await;
            svc.list_relations(&slug).await.map_err(err)
    }

    #[napi]
    pub async fn ticket_relations_create_relation(&self, slug: String, json: String) -> napi::Result<String> {
        let svc = self.ticket_relations.lock().await;
            svc.create_relation(&slug, &json).await.map_err(err)
    }

    #[napi]
    pub async fn ticket_relations_delete_relation(&self, slug: String, relation_id: i64) -> napi::Result<()> {
        let svc = self.ticket_relations.lock().await;
            svc.delete_relation(&slug, relation_id).await.map_err(err)
    }

    #[napi]
    pub async fn ticket_relations_list_commits(&self, slug: String) -> napi::Result<String> {
        let svc = self.ticket_relations.lock().await;
            svc.list_commits(&slug).await.map_err(err)
    }

    #[napi]
    pub async fn ticket_relations_link_commit(&self, slug: String, json: String) -> napi::Result<String> {
        let svc = self.ticket_relations.lock().await;
            svc.link_commit(&slug, &json).await.map_err(err)
    }

    #[napi]
    pub async fn ticket_relations_unlink_commit(&self, slug: String, commit_id: i64) -> napi::Result<()> {
        let svc = self.ticket_relations.lock().await;
            svc.unlink_commit(&slug, commit_id).await.map_err(err)
    }

    #[napi]
    pub async fn ticket_relations_list_merge_requests(&self, slug: String) -> napi::Result<String> {
        let svc = self.ticket_relations.lock().await;
            svc.list_merge_requests(&slug).await.map_err(err)
    }

    #[napi]
    pub async fn ticket_relations_list_comments(&self, slug: String, limit: Option<u32>, offset: Option<u32>) -> napi::Result<String> {
        let svc = self.ticket_relations.lock().await;
            svc.list_comments(&slug, limit, offset).await.map_err(err)
    }

    #[napi]
    pub async fn ticket_relations_create_comment(&self, slug: String, json: String) -> napi::Result<String> {
        let svc = self.ticket_relations.lock().await;
            svc.create_comment(&slug, &json).await.map_err(err)
    }

    #[napi]
    pub async fn ticket_relations_update_comment(&self, slug: String, comment_id: i64, json: String) -> napi::Result<String> {
        let svc = self.ticket_relations.lock().await;
            svc.update_comment(&slug, comment_id, &json).await.map_err(err)
    }

    #[napi]
    pub async fn ticket_relations_delete_comment(&self, slug: String, comment_id: i64) -> napi::Result<()> {
        let svc = self.ticket_relations.lock().await;
            svc.delete_comment(&slug, comment_id).await.map_err(err)
    }

}
