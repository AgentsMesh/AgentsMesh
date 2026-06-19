// renderer topology cache (snake_case JSON) → state proto bytes. Inverse of
// mesh.meshTopologyToCache; mirrors the wasm topology_bytes() reader so the
// shared web selectors (useTopology + derived queries) decode desktop and web
// identically. update_node mutates the cache, so we re-encode on each read.
import { create, toBinary } from "@bufbuild/protobuf";
import { MeshTopologySchema } from "@agentsmesh/proto/mesh/v1/mesh_pb";

type Obj = Record<string, unknown>;
const s = (v: unknown): string => (v as string) ?? "";
const n = (v: unknown): number => (v as number) ?? 0;

export function topologyBytes(cacheJson: string | null): Uint8Array {
  if (!cacheJson) return new Uint8Array();
  const t = JSON.parse(cacheJson) as { nodes?: Obj[]; edges?: Obj[]; channels?: Obj[]; runners?: Obj[] };
  return toBinary(MeshTopologySchema, create(MeshTopologySchema, {
    nodes: (t.nodes ?? []).map((node) => ({
      podKey: s(node.pod_key), status: s(node.status), agentStatus: s(node.agent_status),
      agentSlug: s(node.agent_slug), alias: (node.alias as string) ?? undefined,
      model: (node.model as string) ?? undefined, title: (node.title as string) ?? undefined,
      runnerId: BigInt(n(node.runner_id)), runnerNodeId: s(node.runner_node_id),
      runnerStatus: s(node.runner_status), createdById: BigInt(n(node.created_by_id)),
      ticketId: node.ticket_id != null ? BigInt(n(node.ticket_id)) : undefined,
      ticketSlug: (node.ticket_slug as string) ?? undefined,
      ticketTitle: (node.ticket_title as string) ?? undefined,
      repositoryId: node.repository_id != null ? BigInt(n(node.repository_id)) : undefined,
      startedAt: (node.started_at as string) ?? undefined,
    })),
    edges: (t.edges ?? []).map((e) => ({
      id: BigInt(n(e.id)), source: s(e.source), target: s(e.target), status: s(e.status),
      grantedScopes: (e.granted_scopes as string[]) ?? [], pendingScopes: (e.pending_scopes as string[]) ?? [],
    })),
    channels: (t.channels ?? []).map((c) => ({
      id: BigInt(n(c.id)), name: s(c.name), description: (c.description as string) ?? undefined,
      podKeys: (c.pod_keys as string[]) ?? [], messageCount: n(c.message_count), isArchived: !!c.is_archived,
    })),
    runners: (t.runners ?? []).map((r) => ({
      id: BigInt(n(r.id)), status: s(r.status), nodeId: s(r.node_id),
      maxConcurrentPods: n(r.max_concurrent_pods), currentPods: n(r.current_pods),
    })),
  }));
}
