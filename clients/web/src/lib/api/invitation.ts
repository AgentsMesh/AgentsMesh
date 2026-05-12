// Legacy `invitationApi` adapter. After the proto migration this thin wrapper
// delegates to invitationConnect.ts (binary-wire Connect-RPC) so existing
// call sites keep working unchanged while the wire is now prost-encoded.
//
// New call sites should import directly from `./invitationConnect`; this
// module stays as the dual-track shim until every consumer flips.

import {
  acceptInvitation,
  createInvitation,
  getInvitationByToken,
  listInvitations,
  listPendingInvitations,
  resendInvitation,
  revokeInvitation,
} from "./invitationConnect";

export type { Invitation, InvitationInfo, PendingInvitation } from "./invitationTypes";

// Most legacy callers don't pass orgSlug — they let the wasm session carry
// it. With Connect every org-scoped RPC needs the slug on the request body.
// The default empty string is a deliberate ResolveOrgScope-fail at the
// boundary so the call surfaces the missing context instead of silently
// hitting the wrong org. Tenant-aware components must pass it through.
export const invitationApi = {
  getByToken: async (token: string) => {
    // Legacy callers expect `{ invitation: ... }` wrapper from the old REST
    // shape; the migration unwraps the wrapper at the wire and the adapter
    // re-wraps for backward compatibility.
    const info = await getInvitationByToken(token);
    return { invitation: info };
  },
  accept: async (token: string) => {
    await acceptInvitation(token);
  },
  list: async (orgSlug = "") => {
    const resp = await listInvitations(orgSlug);
    return { invitations: resp.items };
  },
  create: async (orgSlug: string, email: string, role: string) =>
    createInvitation(orgSlug, email, role),
  revoke: async (orgSlug: string, id: number) => {
    await revokeInvitation(orgSlug, id);
  },
  resend: async (orgSlug: string, id: number) => {
    await resendInvitation(orgSlug, id);
  },
  listPending: async () => {
    const resp = await listPendingInvitations();
    return { invitations: resp.items };
  },
};
