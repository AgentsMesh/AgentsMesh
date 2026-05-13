// Web-side ResourceGrant type — snake_case interface mirroring the
// proto.grant.v1 wire shape. The grantApi facade was removed in
// cleanup(grant); call sites use grantConnect.ts directly.
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
