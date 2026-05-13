import { invoke } from "./invoke";
import type { IAgentService } from "@agentsmesh/service-interface";

export class ElectronAgentService implements IAgentService {
  async list_providers(): Promise<string> {
    return invoke<string>("agentListProviders");
  }

  async create_provider(json: string): Promise<string> {
    return invoke<string>("agentCreateProvider", json);
  }

  async update_provider(id: bigint, json: string): Promise<string> {
    return invoke<string>("agentUpdateProvider", Number(id), json);
  }

  async delete_provider(id: bigint): Promise<void> {
    await invoke<void>("agentDeleteProvider", Number(id));
  }

  async set_default_provider(id: bigint): Promise<void> {
    await invoke<void>("agentSetDefaultProvider", Number(id));
  }

  async get_agentpod_settings(): Promise<string> {
    return invoke<string>("agentGetAgentpodSettings");
  }

  async update_agentpod_settings(json: string): Promise<string> {
    return invoke<string>("agentUpdateAgentpodSettings", json);
  }
}
