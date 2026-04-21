use std::collections::HashMap;
use std::sync::Arc;

use agentsmesh_persistence::{StorageBackend, TicketRepo};
use agentsmesh_types::{BoardColumn, Label, Pod, Ticket, TicketPriority, TicketStatus};

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ViewMode {
    List,
    Board,
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
        self.tickets = tickets;
        if let Some(repo) = &self.repo {
            let _ = repo.clear();
            for t in &self.tickets { save_ticket(repo, t); }
        }
    }

    pub fn add_ticket(&mut self, ticket: Ticket) {
        if let Some(repo) = &self.repo { save_ticket(repo, &ticket); }
        self.tickets.push(ticket);
    }

    pub fn update_ticket(&mut self, slug: &str, updated: Ticket) {
        if let Some(t) = self.tickets.iter_mut().find(|t| t.slug == slug) {
            *t = updated.clone();
            if let Some(repo) = &self.repo { save_ticket(repo, t); }
        }
        // Also update in board columns if present
        for col in &mut self.board_columns {
            if let Some(t) = col.tickets.iter_mut().find(|t| t.slug == slug) {
                *t = updated.clone();
            }
        }
        // Update current ticket if it matches
        if self.current_ticket.as_ref().is_some_and(|ct| ct.slug == slug) {
            self.current_ticket = Some(updated);
        }
    }

    pub fn update_ticket_status(&mut self, slug: &str, status: TicketStatus) {
        if let Some(t) = self.tickets.iter_mut().find(|t| t.slug == slug) {
            t.status = status;
            if let Some(repo) = &self.repo { save_ticket(repo, t); }
        }
        if self.current_ticket.as_ref().is_some_and(|ct| ct.slug == slug) {
            if let Some(ct) = &mut self.current_ticket {
                ct.status = status;
            }
        }
    }

    pub fn remove_ticket(&mut self, slug: &str) {
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
        statuses: &[TicketStatus],
        priorities: &[TicketPriority],
        repository_ids: &[i64],
    ) -> Vec<&Ticket> {
        let search_lower = search.map(|s| s.to_lowercase());
        self.tickets.iter().filter(|t| {
            if let Some(ref q) = search_lower {
                if !t.title.to_lowercase().contains(q) && !t.slug.to_lowercase().contains(q) {
                    return false;
                }
            }
            if !statuses.is_empty() && !statuses.contains(&t.status) { return false; }
            if !priorities.is_empty() && !priorities.contains(&t.priority) { return false; }
            if !repository_ids.is_empty() {
                let rid = t.repository_id.unwrap_or(0);
                if !repository_ids.contains(&rid) { return false; }
            }
            true
        }).collect()
    }

    // --- Board columns ---

    pub fn set_board_columns(&mut self, columns: Vec<BoardColumn>) {
        // Sync flat ticket list from board columns
        self.tickets = columns.iter().flat_map(|c| c.tickets.clone()).collect();
        if let Some(repo) = &self.repo {
            let _ = repo.clear();
            for t in &self.tickets { save_ticket(repo, t); }
        }
        self.board_columns = columns;
    }

    pub fn get_board_columns(&self) -> &[BoardColumn] { &self.board_columns }

    pub fn append_column_tickets(&mut self, status: TicketStatus, tickets: Vec<Ticket>) {
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
    pub fn add_label(&mut self, label: Label) { self.labels.push(label); }

    pub fn update_label(&mut self, updated: Label) {
        if let Some(l) = self.labels.iter_mut().find(|l| l.id == updated.id) {
            *l = updated;
        }
    }

    pub fn remove_label(&mut self, id: i64) { self.labels.retain(|l| l.id != id); }

    // --- Current ticket ---

    pub fn set_current_ticket(&mut self, ticket: Option<Ticket>) { self.current_ticket = ticket; }
    pub fn get_current_ticket(&self) -> Option<&Ticket> { self.current_ticket.as_ref() }

    // --- Pods per ticket cache ---

    pub fn set_ticket_pods(&mut self, slug: &str, pods: Vec<Pod>) {
        self.pods_by_ticket_slug.insert(slug.to_string(), pods);
    }

    pub fn get_ticket_pods(&self, slug: &str) -> Vec<Pod> {
        self.pods_by_ticket_slug.get(slug).cloned().unwrap_or_default()
    }

    pub fn clear_ticket_pods(&mut self, slug: &str) {
        self.pods_by_ticket_slug.remove(slug);
    }
}

impl Default for TicketState {
    fn default() -> Self { Self::new() }
}

#[cfg(test)]
mod tests {
    use super::*;
    use agentsmesh_persistence::InMemoryBackend;

    fn make_ticket(slug: &str, status: TicketStatus, priority: TicketPriority) -> Ticket {
        Ticket {
            slug: slug.into(), title: format!("Ticket {slug}"), content: None,
            status, priority, repository_id: Some(1), parent_slug: None,
            created_at: None, updated_at: None,
        }
    }

    fn state_with_tickets() -> TicketState {
        let mut s = TicketState::with_storage(Arc::new(InMemoryBackend::new()));
        s.set_tickets(vec![
            make_ticket("T-1", TicketStatus::Todo, TicketPriority::High),
            make_ticket("T-2", TicketStatus::InProgress, TicketPriority::Low),
            make_ticket("T-3", TicketStatus::Done, TicketPriority::Medium),
        ]);
        s
    }

    #[test]
    fn crud_basics() {
        let mut s = state_with_tickets();
        assert_eq!(s.get_tickets().len(), 3);
        assert!(s.get_ticket_by_slug("T-1").is_some());

        s.add_ticket(make_ticket("T-4", TicketStatus::Backlog, TicketPriority::None));
        assert_eq!(s.get_tickets().len(), 4);

        s.remove_ticket("T-2");
        assert_eq!(s.get_tickets().len(), 3);
        assert!(s.get_ticket_by_slug("T-2").is_none());
    }

    #[test]
    fn update_ticket_propagates() {
        let mut s = state_with_tickets();
        s.set_current_ticket(Some(make_ticket("T-1", TicketStatus::Todo, TicketPriority::High)));

        let mut updated = make_ticket("T-1", TicketStatus::InProgress, TicketPriority::Urgent);
        updated.title = "Updated title".into();
        s.update_ticket("T-1", updated);

        assert_eq!(s.get_ticket_by_slug("T-1").unwrap().title, "Updated title");
        assert_eq!(s.get_current_ticket().unwrap().title, "Updated title");
    }

    #[test]
    fn update_ticket_status() {
        let mut s = state_with_tickets();
        s.set_current_ticket(Some(make_ticket("T-1", TicketStatus::Todo, TicketPriority::High)));
        s.update_ticket_status("T-1", TicketStatus::Done);
        assert_eq!(s.get_ticket_by_slug("T-1").unwrap().status, TicketStatus::Done);
        assert_eq!(s.get_current_ticket().unwrap().status, TicketStatus::Done);
    }

    #[test]
    fn remove_clears_current() {
        let mut s = state_with_tickets();
        s.set_current_ticket(Some(make_ticket("T-1", TicketStatus::Todo, TicketPriority::High)));
        s.remove_ticket("T-1");
        assert!(s.get_current_ticket().is_none());
    }

    #[test]
    fn filter_tickets() {
        let s = state_with_tickets();
        let r = s.filter_tickets(None, &[TicketStatus::Todo], &[], &[]);
        assert_eq!(r.len(), 1);
        assert_eq!(r[0].slug, "T-1");

        let r = s.filter_tickets(Some("ticket t-2"), &[], &[], &[]);
        assert_eq!(r.len(), 1);
        assert_eq!(r[0].slug, "T-2");

        let r = s.filter_tickets(None, &[], &[TicketPriority::High, TicketPriority::Medium], &[]);
        assert_eq!(r.len(), 2);
    }

    #[test]
    fn board_columns_sync() {
        let mut s = TicketState::with_storage(Arc::new(InMemoryBackend::new()));
        let cols = vec![
            BoardColumn { status: TicketStatus::Todo, tickets: vec![make_ticket("T-1", TicketStatus::Todo, TicketPriority::High)], total_count: 5 },
            BoardColumn { status: TicketStatus::Done, tickets: vec![], total_count: 0 },
        ];
        s.set_board_columns(cols);
        assert_eq!(s.get_tickets().len(), 1);
        assert_eq!(s.get_board_columns().len(), 2);

        s.append_column_tickets(TicketStatus::Todo, vec![
            make_ticket("T-2", TicketStatus::Todo, TicketPriority::Low),
        ]);
        assert_eq!(s.get_tickets().len(), 2);
        assert_eq!(s.get_board_columns()[0].tickets.len(), 2);
    }

    #[test]
    fn label_crud() {
        let mut s = TicketState::new();
        s.add_label(Label { id: 1, name: "bug".into(), color: "#f00".into() });
        s.add_label(Label { id: 2, name: "feat".into(), color: "#0f0".into() });
        assert_eq!(s.get_labels().len(), 2);

        s.update_label(Label { id: 1, name: "bugfix".into(), color: "#f00".into() });
        assert_eq!(s.get_labels()[0].name, "bugfix");

        s.remove_label(1);
        assert_eq!(s.get_labels().len(), 1);
        assert_eq!(s.get_labels()[0].id, 2);
    }
}
