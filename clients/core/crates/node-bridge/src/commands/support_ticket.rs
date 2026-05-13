use napi_derive::napi;
use crate::{AppState, err};

#[napi]
impl AppState {
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

    #[napi]
    pub async fn support_ticket_list_support_tickets_connect(&self, request: Vec<u8>) -> napi::Result<Vec<u8>> {
        let svc = self.support_ticket.lock().await;
        svc.list_support_tickets_connect(&request).await.map_err(err)
    }

    #[napi]
    pub async fn support_ticket_get_support_ticket_connect(&self, request: Vec<u8>) -> napi::Result<Vec<u8>> {
        let svc = self.support_ticket.lock().await;
        svc.get_support_ticket_connect(&request).await.map_err(err)
    }

    #[napi]
    pub async fn support_ticket_get_attachment_url_connect(&self, request: Vec<u8>) -> napi::Result<Vec<u8>> {
        let svc = self.support_ticket.lock().await;
        svc.get_attachment_url_connect(&request).await.map_err(err)
    }
}
