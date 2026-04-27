import { invoke } from "./invoke";
import type { IAutopilotService } from "@agentsmesh/service-interface";

export class ElectronAutopilotService implements IAutopilotService {
  private _controllersCache = "[]";
  private _currentControllerCache: string | null = null;
  private _iterationsCache = new Map<string, string>();
  private _thinkingCache = new Map<string, string>();
  private _thinkingHistoryCache = new Map<string, string>();

  controllers_json(): string { return this._controllersCache; }
  current_controller_json(): unknown { return this._currentControllerCache; }

  get_controller_by_pod_key_json(podKey: string): unknown {
    const ctrls = JSON.parse(this._controllersCache) as { pod_key: string }[];
    const c = ctrls.find(x => x.pod_key === podKey);
    return c ? JSON.stringify(c) : null;
  }

  get_iterations_json(key: string): unknown {
    return this._iterationsCache.get(key) ?? null;
  }

  get_thinking_json(key: string): unknown {
    return this._thinkingCache.get(key) ?? null;
  }

  get_thinking_history_json(key: string): unknown {
    return this._thinkingHistoryCache.get(key) ?? null;
  }

  set_controllers(json: string): void { this._controllersCache = json; }
  set_current_controller(json: string): void { this._currentControllerCache = json || null; }
  set_iterations(key: string, json: string): void { this._iterationsCache.set(key, json); }

  add_controller(json: string): void {
    const ctrls = JSON.parse(this._controllersCache) as unknown[];
    ctrls.push(JSON.parse(json));
    this._controllersCache = JSON.stringify(ctrls);
  }

  add_iteration(key: string, json: string): void {
    const iters = JSON.parse(this._iterationsCache.get(key) ?? "[]") as unknown[];
    iters.push(JSON.parse(json));
    this._iterationsCache.set(key, JSON.stringify(iters));
  }

  remove_controller(key: string): void {
    const ctrls = JSON.parse(this._controllersCache) as { key: string }[];
    this._controllersCache = JSON.stringify(ctrls.filter(x => x.key !== key));
  }

  update_controller(key: string, json: string): void {
    const ctrls = JSON.parse(this._controllersCache) as { key: string }[];
    const idx = ctrls.findIndex(x => x.key === key);
    if (idx >= 0) ctrls[idx] = { ...ctrls[idx], ...JSON.parse(json) };
    this._controllersCache = JSON.stringify(ctrls);
  }

  update_thinking(key: string, json: string): void {
    this._thinkingCache.set(key, json);
  }

  async fetch_controllers(): Promise<string> {
    const result = await invoke<string>("autopilotFetchControllers");
    const parsed = JSON.parse(result);
    this._controllersCache = JSON.stringify(parsed.controllers ?? parsed);
    return result;
  }

  async fetch_controller(key: string): Promise<string> {
    const result = await invoke<string>("autopilotFetchController", key);
    this._currentControllerCache = result;
    this.update_controller(key, result);
    return result;
  }

  async fetch_iterations(key: string): Promise<string> {
    const result = await invoke<string>("autopilotFetchIterations", key);
    const parsed = JSON.parse(result);
    this._iterationsCache.set(key, JSON.stringify(parsed.iterations ?? parsed));
    return result;
  }

  async create_controller(json: string): Promise<string> {
    const result = await invoke<string>("autopilotCreateController", json);
    this.add_controller(result);
    this._currentControllerCache = result;
    return result;
  }

  async approve_controller(key: string, json: string): Promise<void> {
    await invoke<void>("autopilotApproveController", key, json);
  }

  async pause_controller(key: string): Promise<void> {
    await invoke<void>("autopilotPauseController", key);
  }

  async resume_controller(key: string): Promise<void> {
    await invoke<void>("autopilotResumeController", key);
  }

  async stop_controller(key: string): Promise<void> {
    await invoke<void>("autopilotStopController", key);
  }

  async takeover_controller(key: string): Promise<void> {
    await invoke<void>("autopilotTakeoverController", key);
  }

  async handback_controller(key: string): Promise<void> {
    await invoke<void>("autopilotHandbackController", key);
  }
}
