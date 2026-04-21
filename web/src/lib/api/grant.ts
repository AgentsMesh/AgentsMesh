import { getGrantService } from "@/lib/wasm-core";

export interface ResourceGrant {
  id: number;
  resource_type: string;
  resource_id: string;
  user_id: number;
  granted_by: number;
  created_at: string;
  user?: { id: number; email: string; username: string; name?: string };
  granted_by_user?: { id: number; email: string; username: string; name?: string };
}

export const grantApi = {
  list: async (resourceType: string, resourceId: string) => {
    const json = await getGrantService().list(resourceType, resourceId);
    return JSON.parse(json) as { grants: ResourceGrant[] };
  },

  grant: async (resourceType: string, resourceId: string, userId: number) => {
    const json = await getGrantService().grant(resourceType, resourceId, BigInt(userId));
    return JSON.parse(json) as { grant: ResourceGrant };
  },

  revoke: async (resourceType: string, resourceId: string, grantId: number) => {
    await getGrantService().revoke(resourceType, resourceId, BigInt(grantId));
    return { message: "ok" };
  },
};
