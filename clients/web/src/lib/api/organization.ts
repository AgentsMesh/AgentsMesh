// Legacy shape preserved as a thin wrapper over the Connect adapter
// (lib/api/org.ts). The old wasm-bridge JSON methods (getOrgApiService().list
// etc.) still exist for the dual-track window; this facade routes new
// callers through the binary lane while keeping the snake_case OrganizationData
// surface unchanged.

import {
  listMyOrgs,
  createOrg,
  getOrg,
  updateOrg,
  deleteOrg,
  listMembers,
  inviteMember,
  removeMember,
  updateMemberRole,
  createPersonalOrg,
} from "./org";
export type { OrganizationData, OrganizationMember } from "./organizationTypes";

export const organizationApi = {
  list: async () => {
    const resp = await listMyOrgs();
    return { organizations: resp.items };
  },
  create: async (data: { name: string; slug: string }) => {
    const org = await createOrg({ name: data.name, slug: data.slug });
    return { organization: org };
  },
  createPersonal: async () => {
    const org = await createPersonalOrg();
    return { organization: org };
  },
  get: async (slug: string) => {
    const org = await getOrg(slug);
    return { organization: org };
  },
  update: async (slug: string, data: { name?: string; logoUrl?: string }) => {
    const org = await updateOrg(slug, data);
    return { organization: org };
  },
  delete: async (slug: string) => {
    await deleteOrg(slug);
  },
  listMembers: async (slug: string) => {
    const resp = await listMembers(slug);
    return { members: resp.items };
  },
  inviteMember: async (slug: string, data: { email?: string; userId?: number; role: string }) =>
    inviteMember(slug, data),
  removeMember: async (slug: string, userId: number) => removeMember(slug, userId),
  updateMemberRole: async (slug: string, userId: number, role: string) =>
    updateMemberRole(slug, userId, role),
};
