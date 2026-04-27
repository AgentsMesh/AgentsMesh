import { invoke } from "./invoke";
import type { IPodService } from "@agentsmesh/service-interface";

export class ElectronPodService implements IPodService {
  private _podsCache = "[]";
  private _currentPodCache: string | null = null;

  pods_json(): string { return this._podsCache; }
  current_pod_json(): unknown { return this._currentPodCache; }
  get_pod_json(pod_key: string): unknown {
    const pods = JSON.parse(this._podsCache) as { pod_key: string }[];
    const p = pods.find(x => x.pod_key === pod_key);
    return p ? JSON.stringify(p) : null;
  }

  upsert_pod(json: string): void {
    const pod = JSON.parse(json) as { pod_key: string };
    const pods = JSON.parse(this._podsCache) as { pod_key: string }[];
    const idx = pods.findIndex(x => x.pod_key === pod.pod_key);
    if (idx >= 0) pods[idx] = pod; else pods.push(pod);
    this._podsCache = JSON.stringify(pods);
  }

  set_pods(json: string): void { this._podsCache = json; }
  set_current_pod(json: string): void { this._currentPodCache = json || null; }

  update_pod_status(key: string, status: string, agentStatus?: string | null, errorCode?: string | null, errorMessage?: string | null): void {
    const pods = JSON.parse(this._podsCache) as { pod_key: string; status: string; agent_status?: string; error_code?: string; error_message?: string }[];
    const p = pods.find(x => x.pod_key === key);
    if (p) {
      p.status = status;
      if (agentStatus !== undefined) p.agent_status = agentStatus ?? undefined;
      if (errorCode !== undefined) p.error_code = errorCode ?? undefined;
      if (errorMessage !== undefined) p.error_message = errorMessage ?? undefined;
    }
    this._podsCache = JSON.stringify(pods);
  }

  update_pod_title(key: string, title: string): void {
    const pods = JSON.parse(this._podsCache) as { pod_key: string; title?: string }[];
    const p = pods.find(x => x.pod_key === key);
    if (p) p.title = title;
    this._podsCache = JSON.stringify(pods);
  }

  update_pod_alias(key: string, alias: string): void {
    const pods = JSON.parse(this._podsCache) as { pod_key: string; alias?: string }[];
    const p = pods.find(x => x.pod_key === key);
    if (p) p.alias = alias;
    this._podsCache = JSON.stringify(pods);
  }

  update_agent_status(key: string, status: string): void {
    const pods = JSON.parse(this._podsCache) as { pod_key: string; agent_status?: string }[];
    const p = pods.find(x => x.pod_key === key);
    if (p) p.agent_status = status;
    this._podsCache = JSON.stringify(pods);
  }

  remove_pod(key: string): void {
    const pods = JSON.parse(this._podsCache) as { pod_key: string }[];
    this._podsCache = JSON.stringify(pods.filter(x => x.pod_key !== key));
  }

  async fetch_pods(status?: string | null, runnerId?: bigint | null, createdById?: bigint | null, limit?: bigint | null, offset?: bigint | null): Promise<string> {
    const result = await invoke<string>("podFetchPods", status, runnerId ? Number(runnerId) : null, createdById ? Number(createdById) : null, limit ? Number(limit) : null, offset ? Number(offset) : null);
    const parsed = JSON.parse(result);
    this._podsCache = JSON.stringify(parsed.pods || []);
    return result;
  }

  async fetch_sidebar_pods(filter: string, userId?: bigint | null): Promise<string> {
    const result = await invoke<string>("podFetchSidebarPods", filter, userId ? Number(userId) : null);
    const parsed = JSON.parse(result);
    this._podsCache = JSON.stringify(parsed.pods || []);
    return result;
  }

  async load_more_pods(filter: string, userId: bigint | null | undefined, offset: bigint): Promise<string> {
    const result = await invoke<string>("podLoadMorePods", filter, userId ? Number(userId) : null, Number(offset));
    const parsed = JSON.parse(result);
    for (const pod of (parsed.newPods || [])) this.upsert_pod(JSON.stringify(pod));
    return result;
  }

  async fetch_pod(key: string): Promise<string> {
    const result = await invoke<string>("podFetchPod", key);
    this.upsert_pod(result);
    this._currentPodCache = result;
    return result;
  }

  async create_pod(json: string): Promise<string> {
    const result = await invoke<string>("podCreatePod", json);
    this.upsert_pod(result);
    this._currentPodCache = result;
    return result;
  }

  async terminate_pod(key: string): Promise<void> {
    await invoke<void>("podTerminatePod", key);
    this.update_pod_status(key, "terminated");
  }

  async update_pod_alias_api(key: string, alias?: string | null): Promise<void> {
    await invoke<void>("podUpdatePodAlias", key, alias);
    this.update_pod_alias(key, alias || "");
  }

  async get_pod_connection(key: string): Promise<string> {
    return invoke<string>("podGetPodConnection", key);
  }
}
