import { create } from "@bufbuild/protobuf";
import {
  TicketSchema,
  type Ticket as ProtoTicket,
  type Label as ProtoLabel,
  type BoardColumn as ProtoBoardColumn,
} from "@agentsmesh/proto/ticket/v1/ticket_pb";
import type {
  TicketData, TicketStatus, TicketPriority,
} from "@agentsmesh/service-interface";

// Single source of truth for the proto.ticket.v1 → TicketData projection,
// shared by web (fromProtoTicket in ticketConnect.ts) and desktop
// (ElectronTicketState). The cache holds the SSOT-shape ticket; the UI re-reads
// joined fields (reporter/assignees/labels) from the API payload.

export function ticketToCache(t: ProtoTicket): TicketData {
  return {
    id: Number(t.id),
    number: t.number,
    slug: t.slug,
    title: t.title,
    content: t.content,
    status: t.status as TicketStatus,
    priority: t.priority as TicketPriority,
    severity: t.severity,
    estimate: t.estimate,
    due_date: t.dueDate,
    started_at: t.startedAt,
    completed_at: t.completedAt,
    created_at: t.createdAt,
    updated_at: t.updatedAt,
    repository_id: t.repositoryId !== undefined ? Number(t.repositoryId) : undefined,
  };
}

// Desktop board cache mirrors wasm board_columns_json (serde of proto
// BoardColumn → `total_count` key), NOT the web getBoard facade shape (`count`).
// KanbanColumn reads `c.count` (absent here) → falls back to tickets.length,
// matching the web/wasm path. Web getBoard maps to the `count` BoardColumn
// facade inline — do NOT unify these two shapes.
export interface CachedBoardColumn {
  status: string;
  tickets: TicketData[];
  total_count: number;
}

export function boardColumnToCache(c: ProtoBoardColumn): CachedBoardColumn {
  return { status: c.status, tickets: c.tickets.map(ticketToCache), total_count: Number(c.totalCount) };
}

export function labelToCache(l: ProtoLabel): { id: number; name: string; color: string } {
  return { id: Number(l.id), name: l.name, color: l.color };
}

// Inverse of ticketToCache: filter_tickets returns a FilterTicketsResponse of
// proto Tickets (the store decodes it back via ticketToCache).
export function cacheTicketToProto(t: TicketData): ProtoTicket {
  return create(TicketSchema, {
    id: BigInt(t.id), number: t.number, slug: t.slug, title: t.title,
    content: t.content, status: t.status, priority: t.priority, severity: t.severity,
    estimate: t.estimate, dueDate: t.due_date, startedAt: t.started_at, completedAt: t.completed_at,
    repositoryId: t.repository_id !== undefined ? BigInt(t.repository_id) : undefined,
    createdAt: t.created_at ?? "", updatedAt: t.updated_at ?? "",
  });
}
