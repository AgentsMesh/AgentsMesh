import { invoke } from "./invoke";
import type { IMeshService } from "@agentsmesh/service-interface";

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
    return JSON.stringify((topo.nodes ?? []).filter(n => n.status === "active"));
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

  set_topology(json: string): void { this._topologyCache = json; }
  clear_topology(): void { this._topologyCache = null; }

  select_node(podKey?: string | null): void {
    this._selectedNode = podKey ?? null;
  }

  async fetch_topology(): Promise<string> {
    const result = await invoke<string>("meshFetchTopology");
    this._topologyCache = result;
    return result;
  }
}
