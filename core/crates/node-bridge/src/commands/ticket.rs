use napi_derive::napi;
use crate::AppState;

#[napi]
impl AppState {
    #[napi]
    pub async fn ticket_tickets_json(&self) -> napi::Result<String> {
        let svc = self.ticket.lock().await;
            Ok(svc.tickets_json())
    }

    #[napi]
    pub async fn ticket_get_ticket_by_slug_json(&self, slug: String) -> napi::Result<String> {
        let svc = self.ticket.lock().await;
            Ok(svc.get_ticket_by_slug_json(&slug).unwrap_or_default())
    }

    #[napi]
    pub async fn ticket_current_ticket_json(&self) -> napi::Result<String> {
        let svc = self.ticket.lock().await;
            Ok(svc.current_ticket_json().unwrap_or_default())
    }

    #[napi]
    pub async fn ticket_board_columns_json(&self) -> napi::Result<String> {
        let svc = self.ticket.lock().await;
            Ok(svc.board_columns_json())
    }

    #[napi]
    pub async fn ticket_labels_json(&self) -> napi::Result<String> {
        let svc = self.ticket.lock().await;
            Ok(svc.labels_json())
    }

    #[napi]
    pub async fn ticket_filter_tickets_json(&self, search: String, statuses_json: String, priorities_json: String, repository_ids_json: String) -> napi::Result<String> {
        let svc = self.ticket.lock().await;
            Ok(svc.filter_tickets_json(&search, &statuses_json, &priorities_json, &repository_ids_json))
    }

    #[napi]
    pub async fn ticket_set_tickets(&self, json: String) -> napi::Result<()> {
        let svc = self.ticket.lock().await;
            svc.set_tickets(&json);
            Ok(())
    }

    #[napi]
    pub async fn ticket_add_ticket(&self, json: String) -> napi::Result<()> {
        let svc = self.ticket.lock().await;
            svc.add_ticket(&json);
            Ok(())
    }

    #[napi]
    pub async fn ticket_update_ticket_local(&self, slug: String, json: String) -> napi::Result<()> {
        let svc = self.ticket.lock().await;
            svc.update_ticket_local(&slug, &json);
            Ok(())
    }

    #[napi]
    pub async fn ticket_update_ticket_status_local(&self, slug: String, status: String) -> napi::Result<()> {
        let svc = self.ticket.lock().await;
            svc.update_ticket_status_local(&slug, &status);
            Ok(())
    }

    #[napi]
    pub async fn ticket_remove_ticket(&self, slug: String) -> napi::Result<()> {
        let svc = self.ticket.lock().await;
            svc.remove_ticket(&slug);
            Ok(())
    }

    #[napi]
    pub async fn ticket_set_current_ticket(&self, json: String) -> napi::Result<()> {
        let svc = self.ticket.lock().await;
            svc.set_current_ticket(&json);
            Ok(())
    }

    #[napi]
    pub async fn ticket_set_board_columns(&self, json: String) -> napi::Result<()> {
        let svc = self.ticket.lock().await;
            svc.set_board_columns(&json);
            Ok(())
    }

    #[napi]
    pub async fn ticket_append_column_tickets(&self, status: String, json: String) -> napi::Result<()> {
        let svc = self.ticket.lock().await;
            svc.append_column_tickets(&status, &json);
            Ok(())
    }

    #[napi]
    pub async fn ticket_set_labels(&self, json: String) -> napi::Result<()> {
        let svc = self.ticket.lock().await;
            svc.set_labels(&json);
            Ok(())
    }

    #[napi]
    pub async fn ticket_add_label(&self, json: String) -> napi::Result<()> {
        let svc = self.ticket.lock().await;
            svc.add_label(&json);
            Ok(())
    }

    #[napi]
    pub async fn ticket_remove_label(&self, id: f64) -> napi::Result<()> {
        let svc = self.ticket.lock().await;
            svc.remove_label(id);
            Ok(())
    }

}
