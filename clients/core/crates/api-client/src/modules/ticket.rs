use crate::ApiClient;
use crate::connect_call::connect_call;
use crate::error::ApiError;
use agentsmesh_types::proto_ticket_v1 as ticket_proto;

// =============================================================================
// Connect-RPC (binary wire). See proto-naming-conventions.md §2.5.
// =============================================================================
//
// These methods call the Connect handlers in
// backend/internal/api/connect/ticket/. Procedure paths derive from
// `proto.ticket.v1.TicketService.<Method>` (conventions §12).

impl ApiClient {
    pub async fn list_tickets_connect(
        &self,
        req: &ticket_proto::ListTicketsRequest,
    ) -> Result<ticket_proto::ListTicketsResponse, ApiError> {
        connect_call(self, "/proto.ticket.v1.TicketService/ListTickets", req).await
    }

    pub async fn get_ticket_connect(
        &self,
        req: &ticket_proto::GetTicketRequest,
    ) -> Result<ticket_proto::Ticket, ApiError> {
        connect_call(self, "/proto.ticket.v1.TicketService/GetTicket", req).await
    }

    pub async fn create_ticket_connect(
        &self,
        req: &ticket_proto::CreateTicketRequest,
    ) -> Result<ticket_proto::Ticket, ApiError> {
        connect_call(self, "/proto.ticket.v1.TicketService/CreateTicket", req).await
    }

    pub async fn update_ticket_connect(
        &self,
        req: &ticket_proto::UpdateTicketRequest,
    ) -> Result<ticket_proto::Ticket, ApiError> {
        connect_call(self, "/proto.ticket.v1.TicketService/UpdateTicket", req).await
    }

    pub async fn delete_ticket_connect(
        &self,
        req: &ticket_proto::DeleteTicketRequest,
    ) -> Result<ticket_proto::DeleteTicketResponse, ApiError> {
        connect_call(self, "/proto.ticket.v1.TicketService/DeleteTicket", req).await
    }

    pub async fn update_ticket_status_connect(
        &self,
        req: &ticket_proto::UpdateTicketStatusRequest,
    ) -> Result<ticket_proto::UpdateTicketStatusResponse, ApiError> {
        connect_call(self, "/proto.ticket.v1.TicketService/UpdateTicketStatus", req).await
    }

    pub async fn get_active_tickets_connect(
        &self,
        req: &ticket_proto::GetActiveTicketsRequest,
    ) -> Result<ticket_proto::ListTicketsResponse, ApiError> {
        connect_call(self, "/proto.ticket.v1.TicketService/GetActiveTickets", req).await
    }

    pub async fn get_board_connect(
        &self,
        req: &ticket_proto::GetBoardRequest,
    ) -> Result<ticket_proto::Board, ApiError> {
        connect_call(self, "/proto.ticket.v1.TicketService/GetBoard", req).await
    }

    pub async fn get_sub_tickets_connect(
        &self,
        req: &ticket_proto::GetSubTicketsRequest,
    ) -> Result<ticket_proto::ListTicketsResponse, ApiError> {
        connect_call(self, "/proto.ticket.v1.TicketService/GetSubTickets", req).await
    }

    pub async fn add_assignee_connect(
        &self,
        req: &ticket_proto::AddAssigneeRequest,
    ) -> Result<ticket_proto::AddAssigneeResponse, ApiError> {
        connect_call(self, "/proto.ticket.v1.TicketService/AddAssignee", req).await
    }

    pub async fn remove_assignee_connect(
        &self,
        req: &ticket_proto::RemoveAssigneeRequest,
    ) -> Result<ticket_proto::RemoveAssigneeResponse, ApiError> {
        connect_call(self, "/proto.ticket.v1.TicketService/RemoveAssignee", req).await
    }

    pub async fn list_labels_connect(
        &self,
        req: &ticket_proto::ListLabelsRequest,
    ) -> Result<ticket_proto::ListLabelsResponse, ApiError> {
        connect_call(self, "/proto.ticket.v1.TicketService/ListLabels", req).await
    }

    pub async fn create_label_connect(
        &self,
        req: &ticket_proto::CreateLabelRequest,
    ) -> Result<ticket_proto::Label, ApiError> {
        connect_call(self, "/proto.ticket.v1.TicketService/CreateLabel", req).await
    }

    pub async fn update_label_connect(
        &self,
        req: &ticket_proto::UpdateLabelRequest,
    ) -> Result<ticket_proto::Label, ApiError> {
        connect_call(self, "/proto.ticket.v1.TicketService/UpdateLabel", req).await
    }

    pub async fn delete_label_connect(
        &self,
        req: &ticket_proto::DeleteLabelRequest,
    ) -> Result<ticket_proto::DeleteLabelResponse, ApiError> {
        connect_call(self, "/proto.ticket.v1.TicketService/DeleteLabel", req).await
    }

    pub async fn add_label_connect(
        &self,
        req: &ticket_proto::AddLabelRequest,
    ) -> Result<ticket_proto::AddLabelResponse, ApiError> {
        connect_call(self, "/proto.ticket.v1.TicketService/AddLabel", req).await
    }

    pub async fn remove_label_connect(
        &self,
        req: &ticket_proto::RemoveLabelRequest,
    ) -> Result<ticket_proto::RemoveLabelResponse, ApiError> {
        connect_call(self, "/proto.ticket.v1.TicketService/RemoveLabel", req).await
    }
}
