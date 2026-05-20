// Authenticated REST against /api/v1/orgs — the same endpoint the
// wasm OrgApiService wraps. Used by post-login redirect resolution and the
// onboarding flow's "create my first workspace" call.

import { lightFetch } from "./api-fetch";
import { updateLightSessionOrgSlug } from "@/lib/light-session";

export interface LightOrganization {
  id: number;
  name: string;
  slug: string;
  role?: string;
  logo_url?: string;
  subscription_plan?: string;
  subscription_status?: string;
}

interface ListOrgsResponse {
  organizations?: LightOrganization[];
}

interface CreateOrgResponse {
  organization?: LightOrganization;
}

export async function lightListOrganizations(): Promise<LightOrganization[]> {
  const resp = await lightFetch<ListOrgsResponse>("/api/v1/orgs", {
    authenticated: true,
  });
  return resp?.organizations ?? [];
}

export interface LightCreateOrgInput {
  name: string;
  slug: string;
  logo_url?: string;
}

export async function lightCreateOrganization(
  input: LightCreateOrgInput,
): Promise<LightOrganization> {
  const resp = await lightFetch<CreateOrgResponse>("/api/v1/orgs", {
    method: "POST",
    body: input,
    authenticated: true,
  });
  const org = resp?.organization;
  if (!org) throw new Error("organizations.create returned 200 with no organization payload");
  updateLightSessionOrgSlug(org.slug);
  return org;
}

// Server derives slug from users.username via slugkit.Sanitize — caller passes
// no body. Use this for onboarding "Quick Start"; never construct the slug
// client-side.
export async function lightCreatePersonalOrganization(): Promise<LightOrganization> {
  const resp = await lightFetch<CreateOrgResponse>("/api/v1/orgs/personal", {
    method: "POST",
    body: {},
    authenticated: true,
  });
  const org = resp?.organization;
  if (!org) throw new Error("organizations.createPersonal returned 200 with no organization payload");
  updateLightSessionOrgSlug(org.slug);
  return org;
}
