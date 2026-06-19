import type { MeshTopology as ProtoTopology } from "@proto/mesh/v1/mesh_pb";
import type { MeshTopology, MeshNode, MeshEdge, ChannelInfo, RunnerInfo } from "./mesh";

// state proto MeshTopology (camelCase) → view MeshTopology (snake_case).
// Replaces the topology_json serde read — UI is now a projection of the
// prost-encoded state. Maps only fields the proto carries (the view's phantom
// fields stayed undefined under the old serde path too).
type N = ProtoTopology["nodes"][number];
type E = ProtoTopology["edges"][number];
type C = ProtoTopology["channels"][number];
type R = ProtoTopology["runners"][number];

function nodeToCache(n: N): MeshNode {
  return {
    pod_key: n.podKey, status: n.status,
    alias: n.alias || undefined, agent_status: n.agentStatus || undefined,
    agent_slug: n.agentSlug || undefined, model: n.model || undefined, title: n.title || undefined,
    runner_id: Number(n.runnerId), runner_node_id: n.runnerNodeId || undefined,
    runner_status: n.runnerStatus || undefined,
    ticket_id: n.ticketId !== undefined ? Number(n.ticketId) : undefined,
    ticket_slug: n.ticketSlug || undefined, ticket_title: n.ticketTitle || undefined,
    repository_id: n.repositoryId !== undefined ? Number(n.repositoryId) : undefined,
    created_by_id: Number(n.createdById), started_at: n.startedAt || undefined,
  };
}

function edgeToCache(e: E): MeshEdge {
  return {
    id: Number(e.id), source: e.source, target: e.target, status: e.status || undefined,
    granted_scopes: e.grantedScopes.length ? e.grantedScopes : undefined,
    pending_scopes: e.pendingScopes.length ? e.pendingScopes : undefined,
  };
}

function channelToCache(c: C): ChannelInfo {
  return {
    id: Number(c.id), name: c.name, description: c.description || undefined,
    pod_keys: c.podKeys, message_count: c.messageCount, is_archived: c.isArchived,
  };
}

function runnerToCache(r: R): RunnerInfo {
  return {
    id: Number(r.id), name: "", status: r.status, node_id: r.nodeId || undefined,
    max_concurrent_pods: r.maxConcurrentPods, current_pods: r.currentPods,
  };
}

export function topologyToCache(t: ProtoTopology): MeshTopology {
  return {
    nodes: t.nodes.map(nodeToCache), edges: t.edges.map(edgeToCache),
    channels: t.channels.map(channelToCache), runners: t.runners.map(runnerToCache),
  };
}
