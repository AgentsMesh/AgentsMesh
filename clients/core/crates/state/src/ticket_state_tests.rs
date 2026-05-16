use crate::ticket_state::{TicketState, ViewMode};
use agentsmesh_types::*;

fn tk(slug: &str, title: &str) -> Ticket { Ticket { id: 0, slug: slug.into(), title: title.into(), content: None, status: TicketStatus::Todo, priority: TicketPriority::Medium, repository_id: None, parent_slug: None, created_at: None, updated_at: None } }
fn lbl(id: i64, name: &str) -> Label { Label { id, name: name.into(), color: "#000".into() } }

#[test] fn new_state() { let s = TicketState::new(); assert!(s.get_tickets().is_empty()); assert!(s.get_labels().is_empty()); assert_eq!(s.get_view_mode(), ViewMode::List); }
#[test] fn set_tickets() { let mut s = TicketState::new(); s.set_tickets(vec![tk("T-1","a")]); assert_eq!(s.get_tickets().len(), 1); }
#[test] fn add_ticket() { let mut s = TicketState::new(); s.add_ticket(tk("T-1","a")); s.add_ticket(tk("T-2","b")); assert_eq!(s.get_tickets().len(), 2); }
#[test] fn update_ticket() { let mut s = TicketState::new(); s.add_ticket(tk("T-1","old")); s.update_ticket("T-1", tk("T-1","new")); assert_eq!(s.get_ticket_by_slug("T-1").unwrap().title, "new"); }
#[test] fn update_nonexistent() { let mut s = TicketState::new(); s.update_ticket("x", tk("x","y")); assert!(s.get_tickets().is_empty()); }
#[test] fn remove_ticket() { let mut s = TicketState::new(); s.add_ticket(tk("T-1","a")); s.remove_ticket("T-1"); assert!(s.get_tickets().is_empty()); }
#[test] fn get_by_slug() { let mut s = TicketState::new(); s.add_ticket(tk("T-1","a")); assert!(s.get_ticket_by_slug("T-1").is_some()); assert!(s.get_ticket_by_slug("X").is_none()); }
#[test] fn view_mode() { let mut s = TicketState::new(); s.set_view_mode(ViewMode::Board); assert_eq!(s.get_view_mode(), ViewMode::Board); }
#[test] fn labels() { let mut s = TicketState::new(); s.add_label(lbl(1,"bug")); s.add_label(lbl(2,"feat")); assert_eq!(s.get_labels().len(), 2); s.update_label(Label{id:1,name:"hotfix".into(),color:"#f00".into()}); assert_eq!(s.get_labels().iter().find(|l|l.id==1).unwrap().name,"hotfix"); s.remove_label(2); assert_eq!(s.get_labels().len(), 1); }
#[test] fn current_ticket() { let mut s = TicketState::new(); s.set_current_ticket(Some(tk("T-1","a"))); assert_eq!(s.get_current_ticket().unwrap().slug, "T-1"); s.set_current_ticket(None); assert!(s.get_current_ticket().is_none()); }
#[test] fn board_columns() { let mut s = TicketState::new(); s.set_board_columns(vec![BoardColumn{status:TicketStatus::Todo,tickets:vec![],total_count:0}]); assert_eq!(s.get_board_columns().len(), 1); }
#[test] fn default_impl() { let s = TicketState::default(); assert!(s.get_tickets().is_empty()); }
