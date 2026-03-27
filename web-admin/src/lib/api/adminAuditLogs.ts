import { apiClient, PaginatedResponse } from "./base";
import type { AuditLog, AuditLogListParams } from "./adminTypes";

export async function listAuditLogs(params?: AuditLogListParams): Promise<PaginatedResponse<AuditLog>> {
  return apiClient.get<PaginatedResponse<AuditLog>>("/audit-logs", params as Record<string, string | number | undefined>);
}
