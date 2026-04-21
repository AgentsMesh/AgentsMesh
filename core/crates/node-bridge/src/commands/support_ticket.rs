use napi_derive::napi;
use crate::{AppState, err};

#[napi]
impl AppState {
    #[napi]
    pub async fn support_ticket_list(&self, status: Option<String>, page: Option<u32>, page_size: Option<u32>) -> napi::Result<String> {
        let svc = self.support_ticket.lock().await;
            svc.list(status, page, page_size).await.map_err(err)
    }

    #[napi]
    pub async fn support_ticket_get_detail(&self, id: i64) -> napi::Result<String> {
        let svc = self.support_ticket.lock().await;
            svc.get_detail(id).await.map_err(err)
    }

    #[napi]
    pub async fn support_ticket_get_attachment_url(&self, id: i64) -> napi::Result<String> {
        let svc = self.support_ticket.lock().await;
            svc.get_attachment_url(id).await.map_err(err)
    }

    #[napi]
    pub async fn support_ticket_create_ticket(&self, title: String, category: String, content: String, priority: Option<String>, file_data: Vec<Vec<u8>>, file_names: Vec<String>) -> napi::Result<String> {
        let svc = self.support_ticket.lock().await;
            svc.create_ticket(&title, &category, &content, priority, file_data, file_names).await.map_err(err)
    }

    #[napi]
    pub async fn support_ticket_add_message(&self, ticket_id: i64, content: String, file_data: Vec<Vec<u8>>, file_names: Vec<String>) -> napi::Result<String> {
        let svc = self.support_ticket.lock().await;
            svc.add_message(ticket_id, &content, file_data, file_names).await.map_err(err)
    }

}
