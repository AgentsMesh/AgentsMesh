// SSO discovery (find configured providers for an email domain) and the
// LDAP password flow. Discovery returns a list of provider configs that the
// login page renders as either redirect buttons (OIDC/SAML) or an LDAP
// username/password form. The login page already wires the redirect
// buttons against getOAuthBaseUrl()/api/v1/auth/sso/<domain>/<protocol> —
// no wasm needed.

import { lightFetch } from "./api-fetch";
import { persistLoginResponse, type AuthLoginResponse } from "./persist";
import type { SSOConfig } from "@/lib/api/ssoTypes";

interface DiscoverResponse {
  configs?: SSOConfig[];
}

export async function lightDiscoverSSO(email: string): Promise<SSOConfig[]> {
  if (!email || !email.includes("@")) return [];
  const resp = await lightFetch<DiscoverResponse>("/api/v1/auth/sso/discover", {
    query: { email },
  });
  return resp?.configs ?? [];
}

export interface LightLdapAuthInput {
  domain: string;
  username: string;
  password: string;
}

export async function lightLdapAuth(input: LightLdapAuthInput): Promise<AuthLoginResponse> {
  const path = `/api/v1/auth/sso/${encodeURIComponent(input.domain)}/ldap`;
  const resp = await lightFetch<AuthLoginResponse>(path, {
    method: "POST",
    body: { username: input.username, password: input.password },
  });
  persistLoginResponse(resp);
  return resp;
}
