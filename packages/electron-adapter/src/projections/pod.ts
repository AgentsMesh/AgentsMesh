import type { Pod as ProtoPod } from "@agentsmesh/proto/pod/v1/pod_pb";
import type { PodData, PodMode } from "@agentsmesh/service-interface";

// Single source of truth for the proto.pod.v1.Pod → PodData projection, shared
// by web (fromProtoPod in podConnect.ts) and desktop (ElectronPodService).
// Optional nested objects map cleanly because proto3 emits them as `| undefined`.
export function podToCache(p: ProtoPod): PodData {
  return {
    id: Number(p.id),
    pod_key: p.podKey,
    status: p.status as PodData["status"],
    agent_status: p.agentStatus,
    alias: p.alias,
    title: p.title,
    runner: p.runner
      ? {
          id: p.runner.id === undefined ? undefined : Number(p.runner.id),
          node_id: p.runner.nodeId,
          status: p.runner.status,
        }
      : undefined,
    agent: p.agent ? { name: p.agent.name, slug: p.agent.slug } : undefined,
    repository: p.repository
      ? {
          id: p.repository.id === undefined ? undefined : Number(p.repository.id),
          name: p.repository.name,
          slug: p.repository.slug,
          provider_type: p.repository.providerType,
        }
      : undefined,
    ticket: p.ticket
      ? {
          id: p.ticket.id === undefined ? undefined : Number(p.ticket.id),
          slug: p.ticket.slug,
          title: p.ticket.title,
        }
      : undefined,
    loop: p.loop
      ? {
          id: p.loop.id === undefined ? undefined : Number(p.loop.id),
          name: p.loop.name,
          slug: p.loop.slug,
        }
      : undefined,
    created_by: p.createdBy
      ? {
          id: p.createdBy.id === undefined ? undefined : Number(p.createdBy.id),
          username: p.createdBy.username,
          name: p.createdBy.name,
        }
      : undefined,
    prompt: p.prompt,
    branch_name: p.branchName,
    sandbox_path: p.sandboxPath,
    interaction_mode: p.interactionMode as PodMode,
    perpetual: p.perpetual,
    restart_count: p.restartCount,
    last_restart_at: p.lastRestartAt,
    started_at: p.startedAt,
    finished_at: p.finishedAt,
    last_activity: p.lastActivity,
    created_at: p.createdAt,
    error_code: p.errorCode,
    error_message: p.errorMessage,
    resumed_by_pod_key: p.resumedByPodKey,
  };
}
