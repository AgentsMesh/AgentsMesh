use agentsmesh_types::proto_ticket_v1 as ticket_proto;

use crate::core::AgentsMeshCore;
use crate::dto::{
    BoardResponseDto, CreateLabelRequestDto, CreateTicketRequestDto, LabelDto,
    LabelListResponseDto, PodListResponseDto, TicketDto, TicketListResponseDto, TicketStatusDto,
    UpdateLabelRequestDto, UpdateTicketRequestDto,
};
use crate::error::CoreError;

fn ticket_status_to_proto(s: TicketStatusDto) -> String {
    match s {
        TicketStatusDto::Backlog => "backlog".into(),
        TicketStatusDto::Todo => "todo".into(),
        TicketStatusDto::InProgress => "in_progress".into(),
        TicketStatusDto::InReview => "in_review".into(),
        TicketStatusDto::Done => "done".into(),
        TicketStatusDto::Unknown => "unknown".into(),
    }
}

fn ticket_from_proto(t: ticket_proto::Ticket) -> TicketDto {
    use agentsmesh_types::Ticket;
    Ticket {
        slug: t.slug,
        title: t.title,
        content: t.content,
        status: serde_json::from_value(serde_json::Value::String(t.status)).unwrap_or_default(),
        priority: serde_json::from_value(serde_json::Value::String(t.priority)).unwrap_or_default(),
        repository_id: t.repository_id,
        parent_slug: t.parent_ticket_slug,
        created_at: if t.created_at.is_empty() { None } else { Some(t.created_at) },
        updated_at: if t.updated_at.is_empty() { None } else { Some(t.updated_at) },
    }
    .into()
}

fn label_from_proto(l: ticket_proto::Label) -> LabelDto {
    agentsmesh_types::Label { id: l.id, name: l.name, color: l.color }.into()
}

#[uniffi::export(async_runtime = "tokio")]
impl AgentsMeshCore {
    pub async fn list_tickets(
        &self,
        status: Option<String>,
        limit: Option<u32>,
        offset: Option<u32>,
    ) -> Result<TicketListResponseDto, CoreError> {
        let req = ticket_proto::ListTicketsRequest {
            org_slug: self.org_slug()?,
            repository_id: None,
            status,
            priority: None,
            assignee_id: None,
            labels: vec![],
            query: None,
            offset: offset.map(|v| v as i32),
            limit: limit.map(|v| v as i32),
        };
        let resp = self.api.list_tickets_connect(&req).await?;
        Ok(TicketListResponseDto {
            tickets: resp.items.into_iter().map(ticket_from_proto).collect(),
            total: Some(resp.total),
            limit: Some(resp.limit as i64),
            offset: Some(resp.offset as i64),
        })
    }

    pub async fn get_ticket(&self, slug: String) -> Result<TicketDto, CoreError> {
        let req = ticket_proto::GetTicketRequest {
            org_slug: self.org_slug()?,
            ticket_slug: slug,
        };
        let resp = self.api.get_ticket_connect(&req).await?;
        Ok(ticket_from_proto(resp))
    }

    pub async fn create_ticket(
        &self,
        req: CreateTicketRequestDto,
    ) -> Result<TicketDto, CoreError> {
        let proto_req = ticket_proto::CreateTicketRequest {
            org_slug: self.org_slug()?,
            title: req.title,
            content: req.content,
            status: None,
            priority: req.priority.map(|p| {
                serde_json::to_value(agentsmesh_types::TicketPriority::from(p))
                    .ok()
                    .and_then(|v| v.as_str().map(String::from))
                    .unwrap_or_default()
            }),
            repository_id: req.repository_id,
            assignee_ids: req.assignee_ids.unwrap_or_default(),
            labels: req.labels.unwrap_or_default().into_iter().map(|id| id.to_string()).collect(),
            parent_ticket_slug: req.parent_slug,
            due_date: None,
        };
        let resp = self.api.create_ticket_connect(&proto_req).await?;
        Ok(ticket_from_proto(resp))
    }

    pub async fn update_ticket(
        &self,
        slug: String,
        req: UpdateTicketRequestDto,
    ) -> Result<TicketDto, CoreError> {
        let proto_req = ticket_proto::UpdateTicketRequest {
            org_slug: self.org_slug()?,
            ticket_slug: slug,
            title: req.title,
            content: req.content,
            status: None,
            priority: req.priority.map(|p| {
                serde_json::to_value(agentsmesh_types::TicketPriority::from(p))
                    .ok()
                    .and_then(|v| v.as_str().map(String::from))
                    .unwrap_or_default()
            }),
            repository_id: req.repository_id,
            assignee_ids: vec![],
            labels: vec![],
            due_date: None,
        };
        let resp = self.api.update_ticket_connect(&proto_req).await?;
        Ok(ticket_from_proto(resp))
    }

    pub async fn delete_ticket(&self, slug: String) -> Result<(), CoreError> {
        let req = ticket_proto::DeleteTicketRequest {
            org_slug: self.org_slug()?,
            ticket_slug: slug,
        };
        self.api.delete_ticket_connect(&req).await?;
        Ok(())
    }

    pub async fn update_ticket_status(
        &self,
        slug: String,
        status: TicketStatusDto,
    ) -> Result<TicketDto, CoreError> {
        let req = ticket_proto::UpdateTicketStatusRequest {
            org_slug: self.org_slug()?,
            ticket_slug: slug.clone(),
            status: ticket_status_to_proto(status),
        };
        self.api.update_ticket_status_connect(&req).await?;
        // proto.ticket.v1 UpdateTicketStatus returns empty; refetch the ticket so
        // the existing FFI contract (returns updated TicketDto) holds.
        let get_req = ticket_proto::GetTicketRequest {
            org_slug: self.org_slug()?,
            ticket_slug: slug,
        };
        let ticket = self.api.get_ticket_connect(&get_req).await?;
        Ok(ticket_from_proto(ticket))
    }

    pub async fn get_active_tickets(
        &self,
        limit: Option<u32>,
    ) -> Result<TicketListResponseDto, CoreError> {
        let req = ticket_proto::GetActiveTicketsRequest {
            org_slug: self.org_slug()?,
            repository_id: None,
            limit: limit.map(|v| v as i32),
        };
        let resp = self.api.get_active_tickets_connect(&req).await?;
        Ok(TicketListResponseDto {
            tickets: resp.items.into_iter().map(ticket_from_proto).collect(),
            total: Some(resp.total),
            limit: Some(resp.limit as i64),
            offset: Some(resp.offset as i64),
        })
    }

    pub async fn get_ticket_board(
        &self,
        repository_id: Option<i64>,
    ) -> Result<BoardResponseDto, CoreError> {
        use crate::dto::BoardColumnDto;
        let req = ticket_proto::GetBoardRequest {
            org_slug: self.org_slug()?,
            repository_id,
            limit: None,
            priority: None,
            assignee_id: None,
            query: None,
        };
        let resp = self.api.get_board_connect(&req).await?;
        Ok(BoardResponseDto {
            columns: resp.columns.into_iter().map(|c| BoardColumnDto {
                status: serde_json::from_value::<agentsmesh_types::TicketStatus>(
                    serde_json::Value::String(c.status),
                )
                .unwrap_or_default()
                .into(),
                tickets: c.tickets.into_iter().map(ticket_from_proto).collect(),
                total_count: c.total_count,
            }).collect(),
            priority_counts_json: None,
        })
    }

    pub async fn get_sub_tickets(&self, slug: String) -> Result<TicketListResponseDto, CoreError> {
        let req = ticket_proto::GetSubTicketsRequest {
            org_slug: self.org_slug()?,
            ticket_slug: slug,
        };
        let resp = self.api.get_sub_tickets_connect(&req).await?;
        Ok(TicketListResponseDto {
            tickets: resp.items.into_iter().map(ticket_from_proto).collect(),
            total: Some(resp.total),
            limit: Some(resp.limit as i64),
            offset: Some(resp.offset as i64),
        })
    }

    pub async fn list_labels(
        &self,
        repository_id: Option<i64>,
    ) -> Result<LabelListResponseDto, CoreError> {
        let req = ticket_proto::ListLabelsRequest {
            org_slug: self.org_slug()?,
            repository_id,
        };
        let resp = self.api.list_labels_connect(&req).await?;
        Ok(LabelListResponseDto {
            labels: resp.items.into_iter().map(label_from_proto).collect(),
        })
    }

    pub async fn create_label(&self, req: CreateLabelRequestDto) -> Result<LabelDto, CoreError> {
        let proto_req = ticket_proto::CreateLabelRequest {
            org_slug: self.org_slug()?,
            name: req.name,
            color: req.color,
            repository_id: None,
        };
        let resp = self.api.create_label_connect(&proto_req).await?;
        Ok(label_from_proto(resp))
    }

    pub async fn update_label(
        &self,
        id: i64,
        req: UpdateLabelRequestDto,
    ) -> Result<LabelDto, CoreError> {
        let proto_req = ticket_proto::UpdateLabelRequest {
            org_slug: self.org_slug()?,
            id,
            name: req.name,
            color: req.color,
        };
        let resp = self.api.update_label_connect(&proto_req).await?;
        Ok(label_from_proto(resp))
    }

    pub async fn delete_label(&self, id: i64) -> Result<(), CoreError> {
        let req = ticket_proto::DeleteLabelRequest {
            org_slug: self.org_slug()?,
            id,
        };
        self.api.delete_label_connect(&req).await?;
        Ok(())
    }

    pub async fn add_ticket_assignee(&self, slug: String, user_id: i64) -> Result<(), CoreError> {
        let req = ticket_proto::AddAssigneeRequest {
            org_slug: self.org_slug()?,
            ticket_slug: slug,
            user_id,
        };
        self.api.add_assignee_connect(&req).await?;
        Ok(())
    }

    pub async fn remove_ticket_assignee(
        &self,
        slug: String,
        user_id: i64,
    ) -> Result<(), CoreError> {
        let req = ticket_proto::RemoveAssigneeRequest {
            org_slug: self.org_slug()?,
            ticket_slug: slug,
            user_id,
        };
        self.api.remove_assignee_connect(&req).await?;
        Ok(())
    }

    pub async fn get_ticket_pods(
        &self,
        slug: String,
        active_only: Option<bool>,
    ) -> Result<PodListResponseDto, CoreError> {
        // proto.ticket.v1 doesn't own ticket→pod lookup — that's MeshService.
        // The MeshNode projection carries the renderer-facing pod fields;
        // the unset PodDto fields stay None on this lookup path.
        use agentsmesh_services::parse_status;
        use agentsmesh_types::PodStatus;
        use agentsmesh_types::proto_mesh_v1 as mp;
        use crate::dto::PodDto;
        let req = mp::GetTicketPodsRequest {
            org_slug: self.org_slug()?,
            ticket_slug: slug,
            active_only,
        };
        let resp = self.api.get_ticket_pods_connect(&req).await?;
        let pods = resp
            .pods
            .into_iter()
            .map(|n| PodDto {
                key: n.pod_key,
                id: None,
                status: parse_status::<PodStatus>(&n.status).into(),
                agent_status: if n.agent_status.is_empty() {
                    None
                } else {
                    Some(n.agent_status)
                },
                alias: n.alias,
                title: n.title,
                agent_slug: n.agent_slug,
                runner_id: if n.runner_id == 0 { None } else { Some(n.runner_id) },
                runner_name: if n.runner_node_id.is_empty() {
                    None
                } else {
                    Some(n.runner_node_id)
                },
                user_id: if n.created_by_id == 0 {
                    None
                } else {
                    Some(n.created_by_id)
                },
                ticket_slug: n.ticket_slug,
                channel_id: None,
                runner: None,
                agent: None,
                repository: None,
                ticket: None,
                loop_info: None,
                created_by: None,
                prompt: None,
                branch_name: None,
                sandbox_path: None,
                started_at: n.started_at,
                finished_at: None,
                last_activity: None,
                created_at: None,
                updated_at: None,
                interaction_mode: None,
                perpetual: None,
                restart_count: None,
                last_restart_at: None,
                error_code: None,
                error_message: None,
            })
            .collect();
        Ok(PodListResponseDto { pods, total: None })
    }

    pub async fn add_ticket_label(&self, slug: String, label_id: i64) -> Result<(), CoreError> {
        let req = ticket_proto::AddLabelRequest {
            org_slug: self.org_slug()?,
            ticket_slug: slug,
            label_id,
        };
        self.api.add_label_connect(&req).await?;
        Ok(())
    }

    pub async fn remove_ticket_label(
        &self,
        slug: String,
        label_id: i64,
    ) -> Result<(), CoreError> {
        let req = ticket_proto::RemoveLabelRequest {
            org_slug: self.org_slug()?,
            ticket_slug: slug,
            label_id,
        };
        self.api.remove_label_connect(&req).await?;
        Ok(())
    }
}
