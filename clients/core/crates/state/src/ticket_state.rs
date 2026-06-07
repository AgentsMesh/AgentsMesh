use std::collections::HashMap;
use std::sync::Arc;

use agentsmesh_persistence::{StorageBackend, TicketRepo};
use agentsmesh_types::proto_pod_v1::Pod;
use agentsmesh_types::proto_ticket_v1::{BoardColumn, Label, Ticket};

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ViewMode {
    List,
    Board,
}

pub mod ticket_status {
    pub const BACKLOG: &str = "backlog";
    pub const TODO: &str = "todo";
    pub const IN_PROGRESS: &str = "in_progress";
    pub const IN_REVIEW: &str = "in_review";
    pub const DONE: &str = "done";
}

pub mod ticket_priority {
    pub const NONE: &str = "none";
    pub const LOW: &str = "low";
    pub const MEDIUM: &str = "medium";
    pub const HIGH: &str = "high";
    pub const URGENT: &str = "urgent";
}

pub struct TicketState {
    tickets: Vec<Ticket>,
    current_ticket: Option<Ticket>,
    labels: Vec<Label>,
    board_columns: Vec<BoardColumn>,
    view_mode: ViewMode,
    pods_by_ticket_slug: HashMap<String, Vec<Pod>>,
    repo: Option<TicketRepo>,
}

fn save_ticket(repo: &TicketRepo, ticket: &Ticket) {
    let _ = repo.save_ticket(ticket);
}

impl TicketState {
    pub fn new() -> Self {
        Self {
            tickets: Vec::new(), current_ticket: None, labels: Vec::new(),
            board_columns: Vec::new(), view_mode: ViewMode::List,
            pods_by_ticket_slug: HashMap::new(), repo: None,
        }
    }

    pub fn with_storage(backend: Arc<dyn StorageBackend>) -> Self {
        let repo = TicketRepo::new(backend);
        let tickets = repo.list_tickets().unwrap_or_default();
        Self {
            tickets, current_ticket: None, labels: Vec::new(),
            board_columns: Vec::new(), view_mode: ViewMode::List,
            pods_by_ticket_slug: HashMap::new(), repo: Some(repo),
        }
    }

    // --- Ticket CRUD ---

    pub fn get_tickets(&self) -> &[Ticket] { &self.tickets }

    pub fn get_ticket_by_slug(&self, slug: &str) -> Option<&Ticket> {
        self.tickets.iter().find(|t| t.slug == slug)
    }

    pub fn set_tickets(&mut self, tickets: Vec<Ticket>) {
        tracing::debug!(target: "ticket", count = tickets.len(), "set tickets (baseline)");
        self.tickets = tickets;
        if let Some(repo) = &self.repo {
            let _ = repo.clear();
            for t in &self.tickets { save_ticket(repo, t); }
        }
    }

    pub fn add_ticket(&mut self, ticket: Ticket) {
        tracing::info!(target: "ticket", slug = %ticket.slug, status = %ticket.status, "add ticket");
        if let Some(repo) = &self.repo { save_ticket(repo, &ticket); }
        self.tickets.push(ticket);
    }

    pub fn update_ticket(&mut self, slug: &str, updated: Ticket) {
        tracing::info!(target: "ticket", slug, "update ticket");
        if let Some(t) = self.tickets.iter_mut().find(|t| t.slug == slug) {
            *t = updated.clone();
            if let Some(repo) = &self.repo { save_ticket(repo, t); }
        }
        for col in &mut self.board_columns {
            if let Some(t) = col.tickets.iter_mut().find(|t| t.slug == slug) {
                *t = updated.clone();
            }
        }
        if self.current_ticket.as_ref().is_some_and(|ct| ct.slug == slug) {
            self.current_ticket = Some(updated);
        }
    }

    pub fn update_ticket_status(&mut self, slug: &str, status: &str) {
        tracing::info!(target: "ticket", slug, status, "status changed");
        if let Some(t) = self.tickets.iter_mut().find(|t| t.slug == slug) {
            t.status = status.to_string();
            if let Some(repo) = &self.repo { save_ticket(repo, t); }
        }
        if self.current_ticket.as_ref().is_some_and(|ct| ct.slug == slug) {
            if let Some(ct) = &mut self.current_ticket {
                ct.status = status.to_string();
            }
        }
    }

    pub fn remove_ticket(&mut self, slug: &str) {
        tracing::info!(target: "ticket", slug, "remove ticket");
        self.tickets.retain(|t| t.slug != slug);
        for col in &mut self.board_columns {
            col.tickets.retain(|t| t.slug != slug);
        }
        if self.current_ticket.as_ref().is_some_and(|ct| ct.slug == slug) {
            self.current_ticket = None;
        }
        if let Some(repo) = &self.repo { let _ = repo.delete_ticket(slug); }
    }

    // --- Filtering ---

    pub fn filter_tickets(
        &self,
        search: Option<&str>,
        statuses: &[String],
        priorities: &[String],
        repository_ids: &[i64],
    ) -> Vec<&Ticket> {
        let search_lower = search.map(|s| s.to_lowercase());
        self.tickets.iter().filter(|t| {
            if let Some(ref q) = search_lower {
                if !t.title.to_lowercase().contains(q) && !t.slug.to_lowercase().contains(q) {
                    return false;
                }
            }
            if !statuses.is_empty() && !statuses.iter().any(|s| s == &t.status) { return false; }
            if !priorities.is_empty() && !priorities.iter().any(|p| p == &t.priority) { return false; }
            if !repository_ids.is_empty() {
                let rid = t.repository_id.unwrap_or(0);
                if !repository_ids.contains(&rid) { return false; }
            }
            true
        }).collect()
    }

    // --- Board columns ---

    pub fn set_board_columns(&mut self, columns: Vec<BoardColumn>) {
        tracing::debug!(target: "ticket", count = columns.len(), "set board columns (baseline)");
        self.tickets = columns.iter().flat_map(|c| c.tickets.clone()).collect();
        if let Some(repo) = &self.repo {
            let _ = repo.clear();
            for t in &self.tickets { save_ticket(repo, t); }
        }
        self.board_columns = columns;
    }

    pub fn get_board_columns(&self) -> &[BoardColumn] { &self.board_columns }

    pub fn append_column_tickets(&mut self, status: &str, tickets: Vec<Ticket>) {
        tracing::debug!(target: "ticket", status, count = tickets.len(), "append column tickets");
        if let Some(col) = self.board_columns.iter_mut().find(|c| c.status == status) {
            for t in &tickets {
                if let Some(repo) = &self.repo { save_ticket(repo, t); }
                self.tickets.push(t.clone());
            }
            col.tickets.extend(tickets);
        }
    }

    // --- View mode ---

    pub fn set_view_mode(&mut self, mode: ViewMode) { self.view_mode = mode; }
    pub fn get_view_mode(&self) -> ViewMode { self.view_mode }

    // --- Labels ---

    pub fn get_labels(&self) -> &[Label] { &self.labels }
    pub fn set_labels(&mut self, labels: Vec<Label>) { self.labels = labels; }
    pub fn add_label(&mut self, label: Label) {
        tracing::info!(target: "ticket", label_id = label.id, "add label");
        self.labels.push(label);
    }

    pub fn update_label(&mut self, updated: Label) {
        tracing::info!(target: "ticket", label_id = updated.id, "update label");
        if let Some(l) = self.labels.iter_mut().find(|l| l.id == updated.id) {
            *l = updated;
        }
    }

    pub fn remove_label(&mut self, id: i64) {
        tracing::info!(target: "ticket", label_id = id, "remove label");
        self.labels.retain(|l| l.id != id);
    }

    // --- Current ticket ---

    pub fn set_current_ticket(&mut self, ticket: Option<Ticket>) { self.current_ticket = ticket; }
    pub fn get_current_ticket(&self) -> Option<&Ticket> { self.current_ticket.as_ref() }

    // --- Pods per ticket cache ---

    pub fn set_ticket_pods(&mut self, slug: &str, pods: Vec<Pod>) {
        tracing::debug!(target: "ticket", slug, count = pods.len(), "set ticket pods");
        self.pods_by_ticket_slug.insert(slug.to_string(), pods);
    }

    pub fn get_ticket_pods(&self, slug: &str) -> Vec<Pod> {
        self.pods_by_ticket_slug.get(slug).cloned().unwrap_or_default()
    }

    pub fn clear_ticket_pods(&mut self, slug: &str) {
        tracing::debug!(target: "ticket", slug, "clear ticket pods");
        self.pods_by_ticket_slug.remove(slug);
    }
}

impl Default for TicketState {
    fn default() -> Self { Self::new() }
}
