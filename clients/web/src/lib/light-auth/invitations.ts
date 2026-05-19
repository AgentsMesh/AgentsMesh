
import { lightFetch } from "./api-fetch";
import { updateLightSessionOrgSlug } from "@/lib/light-session";
import type { InvitationInfo } from "@/lib/api/invitationTypes";

interface GetInviteResponse {
  invitation?: InvitationInfo;
}

export async function lightFetchInvitation(token: string): Promise<InvitationInfo | null> {
  const resp = await lightFetch<GetInviteResponse>(
    `/api/v1/invitations/${encodeURIComponent(token)}`,
  );
  return resp?.invitation ?? null;
}

export async function lightAcceptInvitation(
  token: string,
  organizationSlug: string,
): Promise<void> {
  await lightFetch<unknown>(
    `/api/v1/invitations/${encodeURIComponent(token)}/accept`,
    { method: "POST", authenticated: true },
  );
  // Make the just-joined org the current one so the post-accept redirect
  // lands on its workspace. Dashboard bootstrap may overwrite this once
  // wasm reads /users/me/organizations, but the eager update prevents a
  // flicker / wrong-org render during the transition.
  updateLightSessionOrgSlug(organizationSlug);
}
