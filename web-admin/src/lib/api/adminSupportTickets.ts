import { apiClient, PaginatedResponse } from "./base";
import type {
  SupportTicket,
  SupportTicketMessage,
  SupportTicketStats,
  SupportTicketListParams,
  SupportTicketDetail,
} from "./adminTypesExtended";

export async function listSupportTickets(params?: SupportTicketListParams): Promise<PaginatedResponse<SupportTicket>> {
  return apiClient.get<PaginatedResponse<SupportTicket>>("/support-tickets", params as Record<string, string | number | undefined>);
}

export async function getSupportTicketStats(): Promise<SupportTicketStats> {
  return apiClient.get<SupportTicketStats>("/support-tickets/stats");
}

export async function getSupportTicketDetail(id: number): Promise<SupportTicketDetail> {
  return apiClient.get<SupportTicketDetail>(`/support-tickets/${id}`);
}

export async function getSupportTicketMessages(id: number): Promise<{ messages: SupportTicketMessage[] }> {
  return apiClient.get<{ messages: SupportTicketMessage[] }>(`/support-tickets/${id}/messages`);
}

export async function replySupportTicket(id: number, content: string, files?: File[]): Promise<SupportTicketMessage> {
  const formData = new FormData();
  formData.append("content", content);
  if (files) {
    files.forEach((file) => {
      formData.append("files[]", file);
    });
  }
  return apiClient.postFormData<SupportTicketMessage>(`/support-tickets/${id}/reply`, formData);
}

export async function updateSupportTicketStatus(id: number, status: string): Promise<{ message: string }> {
  return apiClient.patch<{ message: string }>(`/support-tickets/${id}/status`, { status });
}

export async function assignSupportTicket(id: number, adminId: number): Promise<{ message: string }> {
  return apiClient.post<{ message: string }>(`/support-tickets/${id}/assign`, { admin_id: adminId });
}

export async function getSupportTicketAttachmentUrl(attachmentId: number): Promise<{ url: string }> {
  return apiClient.get<{ url: string }>(`/support-tickets/attachments/${attachmentId}/url`);
}
