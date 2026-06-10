import { invoke } from "./invoke";
import type { IMeshService } from "@agentsmesh/service-interface";
import { fromBinary } from "@bufbuild/protobuf";
import { ReplaceTopologyRequestSchema } from "@agentsmesh/proto/mesh_state/v1/mesh_state_pb";
import type { MeshTopology as ProtoMeshTopology } from "@agentsmesh/proto/mesh/v1/mesh_pb";

// proto.mesh.v1.MeshTopology (camelCase + BigInt) → renderer cache shape
// (snake_case + number). Mirrors the legacy serde-skip output so existing
// topology_json() consumers keep parsing the same fields.
function meshTopologyToCache(t: ProtoMeshTopology): Record<string, unknown> {
  return {
    nodes: t.nodes.map((n) => ({
      pod_key: n.podKey, alias: n.alias, status: n.status,
      agent_status: n.agentStatus, agent_slug: n.agentSlug,
      runner_id: n.runnerId !== undefined && n.runnerId !== BigInt(0) ? Number(n.runnerId) : undefined,
      model: n.model, title: n.title,
      ticket_id: n.ticketId !== undefined && n.ticketId !== BigInt(0) ? Number(n.ticketId) : undefined,
      ticket_slug: n.ticketSlug, ticket_title: n.ticketTitle,
      repository_id: n.repositoryId !== undefined && n.repositoryId !== BigInt(0) ? Number(n.repositoryId) : undefined,
      created_by_id: n.createdById !== undefined && n.createdById !== BigInt(0) ? Number(n.createdById) : undefined,
      runner_node_id: n.runnerNodeId, runner_status: n.runnerStatus,
      started_at: n.startedAt,
    })),
    edges: t.edges.map((e) => ({
      id: e.id !== undefined && e.id !== BigInt(0) ? Number(e.id) : undefined,
      source: e.source, target: e.target,
      status: e.status,
      granted_scopes: e.grantedScopes, pending_scopes: e.pendingScopes,
    })),
    channels: t.channels.map((c) => ({
      id: Number(c.id), name: c.name, description: c.description,
      pod_keys: c.podKeys, message_count: c.messageCount,
      is_archived: c.isArchived,
    })),
    runners: t.runners.map((r) => ({
      id: Number(r.id), status: r.status,
      node_id: r.nodeId, max_concurrent_pods: r.maxConcurrentPods,
      current_pods: r.currentPods,
    })),
  };
}

export class ElectronMeshService implements IMeshService {
  private _topologyCache: string | null = null;
  private _selectedNode: string | null = null;

  topology_json(): unknown { return this._topologyCache; }
  selected_node(): unknown { return this._selectedNode; }

  get_node_json(podKey: string): unknown {
    if (!this._topologyCache) return null;
    const topo = JSON.parse(this._topologyCache) as { nodes?: { pod_key: string }[] };
    const n = topo.nodes?.find(x => x.pod_key === podKey);
    return n ? JSON.stringify(n) : null;
  }

  get_active_nodes_json(): string {
    if (!this._topologyCache) return "[]";
    const topo = JSON.parse(this._topologyCache) as { nodes?: { status?: string }[] };
    // Match Rust MeshState::get_active_nodes (running || creating), not "active".
    return JSON.stringify((topo.nodes ?? []).filter(n => n.status === "running" || n.status === "creating"));
  }

  get_edges_for_node_json(podKey: string): string {
    if (!this._topologyCache) return "[]";
    const topo = JSON.parse(this._topologyCache) as { edges?: { source: string; target: string }[] };
    return JSON.stringify((topo.edges ?? []).filter(e => e.source === podKey || e.target === podKey));
  }

  get_channels_for_node_json(podKey: string): string {
    if (!this._topologyCache) return "[]";
    const topo = JSON.parse(this._topologyCache) as { channels?: { pod_keys?: string[] }[] };
    return JSON.stringify((topo.channels ?? []).filter(c => c.pod_keys?.includes(podKey)));
  }

  get_nodes_by_runner_json(runnerId: bigint): string {
    if (!this._topologyCache) return "[]";
    const topo = JSON.parse(this._topologyCache) as { nodes?: { runner_id?: number }[] };
    return JSON.stringify((topo.nodes ?? []).filter(n => n.runner_id === Number(runnerId)));
  }

  get_runner_info_json(runnerId: bigint): unknown {
    if (!this._topologyCache) return null;
    const topo = JSON.parse(this._topologyCache) as { runners?: { id: number }[] };
    const r = topo.runners?.find(x => x.id === Number(runnerId));
    return r ? JSON.stringify(r) : null;
  }

  replace_topology(reqBytes: Uint8Array): void {
    const req = fromBinary(ReplaceTopologyRequestSchema, reqBytes);
    // ReplaceTopologyRequest now carries the typed proto.mesh.v1.MeshTopology
    // directly (PASS 3 SSOT alignment) — serialize the cached projection back
    // to JSON for the renderer's existing topology_json() consumers.
    this._topologyCache = req.topology ? JSON.stringify(meshTopologyToCache(req.topology)) : null;
    void invoke<void>("appMeshReplaceTopology", Array.from(reqBytes)).catch(() => undefined);
  }

  clear_topology(): void { this._topologyCache = null; }

  // status/agent_status only: a full-node Object.assign would revert
  // meshTopologyToCache's 0→undefined projection and drift this node's shape.
  update_node(json: string): void {
    if (!this._topologyCache) return;
    let patch: { pod_key?: string; status?: string; agent_status?: string };
    try {
      patch = JSON.parse(json) as { pod_key?: string; status?: string; agent_status?: string };
    } catch {
      return;
    }
    if (!patch.pod_key) return;
    const topo = JSON.parse(this._topologyCache) as { nodes?: Record<string, unknown>[] };
    const node = topo.nodes?.find((n) => (n as { pod_key?: string }).pod_key === patch.pod_key);
    if (!node) return;
    if (patch.status !== undefined) node.status = patch.status;
    if (patch.agent_status !== undefined) node.agent_status = patch.agent_status;
    this._topologyCache = JSON.stringify(topo);
  }

  select_node(podKey?: string | null): void {
    this._selectedNode = podKey ?? null;
  }

  async fetch_topology(): Promise<Uint8Array> {
    // Networking-only: returns ReplaceTopologyRequest bytes. The store feeds
    // them to replace_topology (cache + runtime.state sync) — no local cache
    // write here, else the projection would run twice.
    return invoke<Uint8Array>("meshFetchTopology");
  }

  // Connect-RPC: proto.mesh.v1.MeshService. Binary wire — every method
  // forwards a Uint8Array request to the matching napi handler
  // (commands/mesh.rs) and gets a Uint8Array response back. Callers wrap
  // with @bufbuild/protobuf .toBinary() / .fromBinary().

  async getMeshTopologyConnect(request: Uint8Array): Promise<Uint8Array> {
    return invoke<Uint8Array>("meshGetMeshTopologyConnect", request);
  }

  async getTicketPodsConnect(request: Uint8Array): Promise<Uint8Array> {
    return invoke<Uint8Array>("meshGetTicketPodsConnect", request);
  }

  async batchGetTicketPodsConnect(request: Uint8Array): Promise<Uint8Array> {
    return invoke<Uint8Array>("meshBatchGetTicketPodsConnect", request);
  }

  async createPodForTicketConnect(request: Uint8Array): Promise<Uint8Array> {
    return invoke<Uint8Array>("meshCreatePodForTicketConnect", request);
  }
}
