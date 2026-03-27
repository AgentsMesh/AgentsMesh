import { apiClient } from "./base";
import type {
  RelayStats,
  RelayListResponse,
  RelayDetailResponse,
  SessionListResponse,
} from "./adminTypesExtended";

export async function listRelays(): Promise<RelayListResponse> {
  return apiClient.get<RelayListResponse>("/relays");
}

export async function getRelayStats(): Promise<RelayStats> {
  return apiClient.get<RelayStats>("/relays/stats");
}

export async function getRelay(id: string): Promise<RelayDetailResponse> {
  return apiClient.get<RelayDetailResponse>(`/relays/${encodeURIComponent(id)}`);
}

export async function forceUnregisterRelay(id: string, migrateSessions: boolean = false): Promise<{ status: string; relay_id: string; affected_sessions: number }> {
  return apiClient.delete<{ status: string; relay_id: string; affected_sessions: number }>(`/relays/${encodeURIComponent(id)}`, { migrate_sessions: migrateSessions });
}

export async function listSessions(relayId?: string): Promise<SessionListResponse> {
  const params = relayId ? { relay_id: relayId } : undefined;
  return apiClient.get<SessionListResponse>("/relays/sessions", params);
}

export async function migrateSession(podKey: string, targetRelay: string): Promise<{ status: string; from_relay: string; to_relay: string }> {
  return apiClient.post<{ status: string; from_relay: string; to_relay: string }>("/relays/sessions/migrate", { pod_key: podKey, target_relay: targetRelay });
}

export async function bulkMigrateSessions(sourceRelay: string, targetRelay: string): Promise<{ status: string; total: number; migrated: number; failed: number }> {
  return apiClient.post<{ status: string; total: number; migrated: number; failed: number }>("/relays/sessions/bulk-migrate", { source_relay: sourceRelay, target_relay: targetRelay });
}
