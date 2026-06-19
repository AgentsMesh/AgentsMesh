// Ticket ViewModels — UI-side projection of `proto.ticket.v1` types. Owned in
// this zero-dep contract layer so the web (fromProtoTicket) and desktop
// (electron-adapter projection) share one definition.
//
// Snake_case fields are kept because (a) the UI consumes denormalized
// reporter/assignees/labels/repository/parent_ticket joins not in proto,
// (b) many consumers (TicketCreateDialog, TicketDetailSidebar, KanbanBoard,
// RelationsList, CommentsList, CommitsList, TicketIcons) reference these
// directly. Adapters in lib/api/ticket*.ts map the camelCase proto types on.
export type TicketStatus = "backlog" | "todo" | "in_progress" | "in_review" | "done";
export type TicketPriority = "none" | "low" | "medium" | "high" | "urgent";

export interface TicketData {
  id: number;
  number: number;
  slug: string;
  title: string;
  content?: string;
  status: TicketStatus;
  priority: TicketPriority;
  severity?: string;
  estimate?: number;
  due_date?: string;
  started_at?: string;
  completed_at?: string;
  created_at?: string;
  updated_at?: string;
  reporter?: { id: number; username: string; name?: string; avatar_url?: string };
  assignees?: Array<{
    ticket_id: number;
    user_id: number;
    user?: { id: number; username: string; name?: string; avatar_url?: string };
  }>;
  labels?: Array<{ id: number; name: string; color: string }>;
  repository_id?: number;
  repository?: { id: number; name: string };
  parent_ticket?: { id: number; slug: string; title: string };
}

export interface TicketRelation {
  id: number;
  source_ticket_id: number;
  target_ticket_id: number;
  relation_type: string;
  source_ticket?: { id: number; slug: string; title: string };
  target_ticket?: { id: number; slug: string; title: string };
  created_at?: string;
}

export interface TicketCommit {
  id: number;
  ticket_id: number;
  commit_sha: string;
  commit_message?: string;
  commit_url?: string;
  author_name?: string;
  author_email?: string;
  committed_at?: string;
  created_at?: string;
}

export interface TicketComment {
  id: number;
  ticket_id: number;
  user_id: number;
  content: string;
  parent_id?: number;
  mentions?: Array<{ user_id: number; username: string }>;
  created_at?: string;
  updated_at?: string;
  user?: { id: number; username: string; name?: string; avatar_url?: string };
  replies?: TicketComment[];
}

export interface BoardColumn {
  status: string;
  tickets: TicketData[];
  count: number;
}
