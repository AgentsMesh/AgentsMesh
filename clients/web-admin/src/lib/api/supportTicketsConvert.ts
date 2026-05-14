// Proto ↔ snake_case TS conversion for support tickets admin surface.
// Mirrors the REST JSON shape so the page + table + detail view consume
// the same SupportTicket / SupportTicketMessage types they always have.
//
// bigint → number coercion is intentional: ticket / user IDs in this
// product are well below 2^53. Number(BigInt) is the standard adapter
// pattern (see ssoConvert.ts and connect-admin relays adapter).
import type {
  AdminSupportTicket,
  AdminSupportTicketAttachment,
  AdminSupportTicketMessage,
  AdminSupportTicketUser,
} from "@proto/support_ticket/v1/support_ticket_admin_pb";

import type {
  SupportTicket,
  SupportTicketAttachment,
  SupportTicketMessage,
} from "./adminTypesExtended";

export function fromProtoTicket(t: AdminSupportTicket): SupportTicket {
  const out: SupportTicket = {
    id: Number(t.id),
    user_id: Number(t.userId),
    title: t.title,
    category: t.category,
    status: t.status,
    priority: t.priority,
    created_at: t.createdAt,
    updated_at: t.updatedAt,
  };
  if (t.resolvedAt !== undefined) out.resolved_at = t.resolvedAt;
  if (t.assignedAdminId !== undefined) {
    out.assigned_admin_id = Number(t.assignedAdminId);
  }
  return out;
}

export function fromProtoMessage(m: AdminSupportTicketMessage): SupportTicketMessage {
  return {
    id: Number(m.id),
    ticket_id: Number(m.ticketId),
    user_id: Number(m.userId),
    content: m.content,
    is_admin_reply: m.isAdminReply,
    created_at: m.createdAt,
    user: m.user ? fromProtoUser(m.user) : undefined,
    attachments: m.attachments.map(fromProtoAttachment),
  };
}

function fromProtoUser(u: AdminSupportTicketUser): {
  id: number;
  name: string;
  email: string;
  avatar_url?: string;
} {
  return {
    id: Number(u.id),
    name: u.name ?? "",
    email: u.email,
    avatar_url: u.avatarUrl,
  };
}

function fromProtoAttachment(a: AdminSupportTicketAttachment): SupportTicketAttachment {
  return {
    id: Number(a.id),
    original_name: a.originalName,
    mime_type: a.mimeType,
    size: Number(a.size),
  };
}
