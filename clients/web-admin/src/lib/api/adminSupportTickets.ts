// Connect-RPC adapter for proto.support_ticket.v1.SupportTicketAdminService.
//
// Migrated from REST `/api/v1/admin/support-tickets/*`. Keeps the existing
// snake_case + number TS surface (`SupportTicket`, `SupportTicketMessage`,
// `SupportTicketDetail`) so the page + table + detail view don't need to
// change. Proto types are camelCase + bigint; conversion lives in
// supportTicketsConvert.ts to keep this file under the 200-line cap.
//
// Reply (multipart): when files are attached, falls back to the REST
// POST /support-tickets/:id/reply endpoint (multipart/form-data). Connect
// has no multipart story; the JSON-only path uses Connect.
import {
  AdminAssignSupportTicketRequestSchema,
  AdminAssignSupportTicketResponseSchema,
  AdminGetSupportTicketAttachmentUrlRequestSchema,
  AdminGetSupportTicketAttachmentUrlResponseSchema,
  AdminGetSupportTicketRequestSchema,
  AdminListSupportTicketMessagesRequestSchema,
  AdminListSupportTicketMessagesResponseSchema,
  AdminListSupportTicketsRequestSchema,
  AdminListSupportTicketsResponseSchema,
  AdminReplySupportTicketRequestSchema,
  AdminSupportTicketDetailSchema,
  AdminSupportTicketMessageSchema,
  AdminUpdateSupportTicketStatusRequestSchema,
  AdminUpdateSupportTicketStatusResponseSchema,
  GetSupportTicketStatsRequestSchema,
  SupportTicketAdminService,
  SupportTicketStatsSchema,
} from "@proto/support_ticket/v1/support_ticket_admin_pb";

import { callConnect } from "@/lib/connect/transport";
import { apiClient, PaginatedResponse } from "./base";
import {
  fromProtoMessage,
  fromProtoTicket,
} from "./supportTicketsConvert";
import type {
  SupportTicket,
  SupportTicketDetail,
  SupportTicketListParams,
  SupportTicketMessage,
  SupportTicketStats,
} from "./adminTypesExtended";

const SERVICE = "proto.support_ticket.v1.SupportTicketAdminService";
void SupportTicketAdminService;

export async function listSupportTickets(
  params?: SupportTicketListParams,
): Promise<PaginatedResponse<SupportTicket>> {
  const resp = await callConnect(
    SERVICE,
    "ListSupportTickets",
    AdminListSupportTicketsRequestSchema,
    AdminListSupportTicketsResponseSchema,
    {
      search: params?.search ?? "",
      status: params?.status ?? "",
      category: params?.category ?? "",
      priority: params?.priority ?? "",
      page: params?.page ?? 0,
      pageSize: params?.page_size ?? 0,
    },
  );
  return {
    data: resp.data.map(fromProtoTicket),
    total: Number(resp.total),
    page: resp.page,
    page_size: resp.pageSize,
    total_pages: resp.totalPages,
  };
}

export async function getSupportTicketStats(): Promise<SupportTicketStats> {
  const resp = await callConnect(
    SERVICE,
    "GetSupportTicketStats",
    GetSupportTicketStatsRequestSchema,
    SupportTicketStatsSchema,
    {},
  );
  return {
    total: Number(resp.total),
    open: Number(resp.open),
    in_progress: Number(resp.inProgress),
    resolved: Number(resp.resolved),
    closed: Number(resp.closed),
  };
}

export async function getSupportTicketDetail(id: number): Promise<SupportTicketDetail> {
  const resp = await callConnect(
    SERVICE,
    "GetSupportTicket",
    AdminGetSupportTicketRequestSchema,
    AdminSupportTicketDetailSchema,
    { id: BigInt(id) },
  );
  if (!resp.ticket) {
    throw new Error("Support ticket not found");
  }
  return {
    ticket: fromProtoTicket(resp.ticket),
    messages: resp.messages.map(fromProtoMessage),
  };
}

export async function getSupportTicketMessages(
  id: number,
): Promise<{ messages: SupportTicketMessage[] }> {
  const resp = await callConnect(
    SERVICE,
    "ListSupportTicketMessages",
    AdminListSupportTicketMessagesRequestSchema,
    AdminListSupportTicketMessagesResponseSchema,
    { id: BigInt(id) },
  );
  return { messages: resp.data.map(fromProtoMessage) };
}

export async function replySupportTicket(
  id: number,
  content: string,
  files?: File[],
): Promise<SupportTicketMessage> {
  // Multipart path stays on REST — Connect has no multipart story.
  if (files && files.length > 0) {
    const formData = new FormData();
    formData.append("content", content);
    files.forEach((file) => {
      formData.append("files[]", file);
    });
    return apiClient.postFormData<SupportTicketMessage>(
      `/support-tickets/${id}/reply`,
      formData,
    );
  }
  const resp = await callConnect(
    SERVICE,
    "ReplySupportTicket",
    AdminReplySupportTicketRequestSchema,
    AdminSupportTicketMessageSchema,
    { id: BigInt(id), content },
  );
  return fromProtoMessage(resp);
}

export async function updateSupportTicketStatus(
  id: number,
  status: string,
): Promise<{ message: string }> {
  const resp = await callConnect(
    SERVICE,
    "UpdateSupportTicketStatus",
    AdminUpdateSupportTicketStatusRequestSchema,
    AdminUpdateSupportTicketStatusResponseSchema,
    { id: BigInt(id), status },
  );
  return { message: resp.message };
}

export async function assignSupportTicket(
  id: number,
  adminId: number,
): Promise<{ message: string }> {
  const resp = await callConnect(
    SERVICE,
    "AssignSupportTicket",
    AdminAssignSupportTicketRequestSchema,
    AdminAssignSupportTicketResponseSchema,
    { id: BigInt(id), adminId: BigInt(adminId) },
  );
  return { message: resp.message };
}

export async function getSupportTicketAttachmentUrl(
  attachmentId: number,
): Promise<{ url: string }> {
  const resp = await callConnect(
    SERVICE,
    "GetSupportTicketAttachmentUrl",
    AdminGetSupportTicketAttachmentUrlRequestSchema,
    AdminGetSupportTicketAttachmentUrlResponseSchema,
    { attachmentId: BigInt(attachmentId) },
  );
  return { url: resp.url };
}
