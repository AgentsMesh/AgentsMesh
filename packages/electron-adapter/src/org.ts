import { invoke } from "./invoke";
import type { IOrgApiService } from "@agentsmesh/service-interface";

export class ElectronOrgService implements IOrgApiService {
  async list(): Promise<string> {
    return invoke<string>("orgList");
  }

  async get(slug: string): Promise<string> {
    return invoke<string>("orgGet", slug);
  }

  async create(json: string): Promise<string> {
    return invoke<string>("orgCreate", json);
  }

  async create_personal(): Promise<string> {
    return invoke<string>("orgCreatePersonal");
  }

  async update(slug: string, json: string): Promise<string> {
    return invoke<string>("orgUpdate", slug, json);
  }

  async delete(slug: string): Promise<void> {
    await invoke<void>("orgDelete", slug);
  }

  async list_members(slug: string): Promise<string> {
    return invoke<string>("orgListMembers", slug);
  }

  async invite_member(slug: string, json: string): Promise<string> {
    return invoke<string>("orgInviteMember", slug, json);
  }

  async remove_member(slug: string, userId: bigint): Promise<void> {
    await invoke<void>("orgRemoveMember", slug, Number(userId));
  }

  async update_member_role(slug: string, userId: bigint, json: string): Promise<string> {
    return invoke<string>("orgUpdateMemberRole", slug, Number(userId), json);
  }
}
