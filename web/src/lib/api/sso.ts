import { publicRequest, publicPost } from "./base";
import { getOAuthBaseUrl } from "@/lib/env";

// SSO API

export interface SSOConfig {
  domain: string;
  name: string;
  protocol: "oidc" | "saml" | "ldap";
  enforce_sso: boolean;
}

export interface SSODiscoverResponse {
  configs: SSOConfig[];
}

export interface LDAPAuthResponse {
  token: string;
  refresh_token: string;
  expires_at: string;
  token_type: string;
  user: {
    id: number;
    email: string;
    username: string;
    name?: string;
  };
}

export const ssoApi = {
  discover: (email: string) =>
    publicRequest<SSODiscoverResponse>(
      `/api/v1/auth/sso/discover?email=${encodeURIComponent(email)}`
    ).catch(() => ({ configs: [] }) as SSODiscoverResponse),

  ldapAuth: (domain: string, username: string, password: string) =>
    publicPost<LDAPAuthResponse>(
      `/api/v1/auth/sso/${encodeURIComponent(domain)}/ldap`,
      { username, password }
    ),
};

export function getSSOAuthURL(domain: string, protocol: string, redirect?: string): string {
  const base = getOAuthBaseUrl();
  const params = redirect ? `?redirect=${encodeURIComponent(redirect)}` : "";
  return `${base}/api/v1/auth/sso/${encodeURIComponent(domain)}/${encodeURIComponent(protocol)}${params}`;
}
