import { getOrgApiService } from "@/lib/wasm-core";
export type { OrganizationData, OrganizationMember } from "./organizationTypes";

export const organizationApi = {
  list: async () => {
    const json = await getOrgApiService().list();
    return JSON.parse(json);
  },
  create: async (data: { name: string; slug?: string }) => {
    const json = await getOrgApiService().create(JSON.stringify(data));
    return JSON.parse(json);
  },
  // Server derives the slug from users.username via slugkit.Sanitize.
  // Use this for onboarding "Quick Start" instead of constructing a slug
  // client-side (Phase 5 / onboarding bug fix).
  createPersonal: async () => {
    const json = await getOrgApiService().create_personal();
    return JSON.parse(json);
  },
  get: async (slug: string) => {
    const json = await getOrgApiService().get(slug);
    return JSON.parse(json);
  },
  update: async (slug: string, data: { name?: string }) => {
    const json = await getOrgApiService().update(slug, JSON.stringify(data));
    return JSON.parse(json);
  },
  delete: async (slug: string) => {
    await getOrgApiService().delete(slug);
  },
  listMembers: async (slug: string) => {
    const json = await getOrgApiService().list_members(slug);
    return JSON.parse(json);
  },
  removeMember: async (slug: string, userId: number) => {
    await getOrgApiService().remove_member(slug, BigInt(userId));
  },
  updateMemberRole: async (slug: string, userId: number, role: string) => {
    const json = await getOrgApiService().update_member_role(slug, BigInt(userId), JSON.stringify({ role }));
    return JSON.parse(json);
  },
};
