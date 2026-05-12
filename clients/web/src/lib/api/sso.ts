// SSO API surface — migrated to Connect-RPC (proto.sso.v1.SSOService).
//
// The implementation now goes through the Connect adapter
// (`ssoConnect.ts`) which speaks binary protobuf over the
// /proto.sso.v1.SSOService/* endpoints. The public export surface
// stays unchanged so existing call sites (login page x2, SSOSection,
// desktop login page) compile without edits.

import { discover, ldapAuth } from "./ssoConnect";
export type { SSOConfig } from "./ssoTypes";

export const ssoApi = {
  discover: async (email: string) => discover(email),
  ldapAuth: async (
    domain: string,
    data: { username: string; password: string },
  ) => ldapAuth(domain, data.username, data.password),
};

export function getSSOAuthURL(config: { protocol: string; domain: string; provider_url?: string }, redirectUrl?: string): string {
  const base = config.provider_url || "";
  return redirectUrl ? `${base}?redirect=${encodeURIComponent(redirectUrl)}` : base;
}
