use napi_derive::napi;
use crate::{AppState, err};

#[napi]
impl AppState {
    #[napi]
    pub async fn ticket_fetch_tickets(&self, status: Option<String>, limit: Option<u32>, offset: Option<u32>) -> napi::Result<String> {
        let svc = self.ticket.lock().await;
            svc.fetch_tickets(status, limit, offset).await.map_err(err)
    }

    #[napi]
    pub async fn ticket_fetch_board(&self, repository_id: Option<i64>) -> napi::Result<String> {
        let svc = self.ticket.lock().await;
            svc.fetch_board(repository_id).await.map_err(err)
    }

    #[napi]
    pub async fn ticket_load_more_column(&self, status: String, offset: u32, limit: u32) -> napi::Result<String> {
        let svc = self.ticket.lock().await;
            svc.load_more_column(&status, offset, limit).await.map_err(err)
    }

    #[napi]
    pub async fn ticket_fetch_ticket(&self, slug: String) -> napi::Result<String> {
        let svc = self.ticket.lock().await;
            svc.fetch_ticket(&slug).await.map_err(err)
    }

    #[napi]
    pub async fn ticket_create_ticket(&self, request_json: String) -> napi::Result<String> {
        let svc = self.ticket.lock().await;
            svc.create_ticket(&request_json).await.map_err(err)
    }

    #[napi]
    pub async fn ticket_update_ticket(&self, slug: String, request_json: String) -> napi::Result<String> {
        let svc = self.ticket.lock().await;
            svc.update_ticket(&slug, &request_json).await.map_err(err)
    }

    #[napi]
    pub async fn ticket_delete_ticket(&self, slug: String) -> napi::Result<()> {
        let svc = self.ticket.lock().await;
            svc.delete_ticket(&slug).await.map_err(err)
    }

    #[napi]
    pub async fn ticket_update_ticket_status(&self, slug: String, status: String) -> napi::Result<String> {
        let svc = self.ticket.lock().await;
            svc.update_ticket_status(&slug, &status).await.map_err(err)
    }

    #[napi]
    pub async fn ticket_fetch_labels(&self, repository_id: Option<i64>) -> napi::Result<String> {
        let svc = self.ticket.lock().await;
            svc.fetch_labels(repository_id).await.map_err(err)
    }

    #[napi]
    pub async fn ticket_create_label(&self, name: String, color: String, repository_id: Option<i64>) -> napi::Result<String> {
        let svc = self.ticket.lock().await;
            svc.create_label(&name, &color, repository_id).await.map_err(err)
    }

    #[napi]
    pub async fn ticket_delete_label(&self, id: f64) -> napi::Result<()> {
        let svc = self.ticket.lock().await;
            svc.delete_label(id).await.map_err(err)
    }

    #[napi]
    pub async fn ticket_get_ticket_pods(&self, slug: String, active_only: Option<bool>) -> napi::Result<String> {
        let svc = self.ticket.lock().await;
            svc.get_ticket_pods(&slug, active_only).await.map_err(err)
    }

    #[napi]
    pub async fn ticket_ticket_pods_json(&self, slug: String) -> napi::Result<String> {
        let svc = self.ticket.lock().await;
        Ok(svc.ticket_pods_json(&slug))
    }

    #[napi]
    pub async fn ticket_get_sub_tickets(&self, slug: String) -> napi::Result<String> {
        let svc = self.ticket.lock().await;
            svc.get_sub_tickets(&slug).await.map_err(err)
    }

}
