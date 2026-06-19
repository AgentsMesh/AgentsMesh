import type { Runner as ProtoRunner } from "@agentsmesh/proto/runner_api/v1/runner_pb";
import type { RunnerData } from "@agentsmesh/service-interface";
import { parseJSON } from "./parse-json";

// Single source of truth for the proto.runner_api.v1.Runner → RunnerData
// projection, shared by web (fromProtoRunner in runnerConnect.ts) and desktop
// (ElectronRunnerService). host_info ships as a JSON string.
export function runnerToCache(r: ProtoRunner): RunnerData {
  return {
    id: Number(r.id),
    node_id: r.nodeId,
    description: r.description || undefined,
    status: r.status as RunnerData["status"],
    last_heartbeat: r.lastHeartbeat,
    current_pods: r.currentPods,
    max_concurrent_pods: r.maxConcurrentPods,
    runner_version: r.runnerVersion,
    is_enabled: r.isEnabled,
    visibility: r.visibility as RunnerData["visibility"],
    registered_by_user_id:
      r.registeredByUserId === undefined ? undefined : Number(r.registeredByUserId),
    host_info: parseJSON<RunnerData["host_info"]>(r.hostInfoJson),
    available_agents: r.availableAgents?.length ? r.availableAgents : undefined,
    tags: r.tags?.length ? r.tags : undefined,
    created_at: r.createdAt,
    updated_at: r.updatedAt,
  };
}
