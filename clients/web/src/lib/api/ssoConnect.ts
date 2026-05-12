// Connect-RPC adapter for proto.sso.v1.SSOService (public — no auth).
//
// Encodes requests via @bufbuild/protobuf .toBinary(), passes the Uint8Array
// to the wasm bridge (binary in / binary out per conventions §2.5), decodes
// responses via .fromBinary(). No JSON intermediate.
//
// Returns the existing snake_case web shapes (SSOConfig, LDAPAuthResponse)
// so the 4 login-page call sites don't have to flip off camelCase + BigInt
// — same dual-track pattern as podConnect.ts and ticketConnect.ts during
// the migration window. The legacy `ssoApi` JSON methods stay available
// until call sites finish flipping over.

import {
  DiscoverRequestSchema,
  DiscoverResponseSchema,
  LdapAuthRequestSchema,
  LdapAuthResponseSchema,
  type LdapAuthUser as ProtoLdapAuthUser,
  type LdapAuthResponse as ProtoLdapAuthResponse,
  type SSODiscoverConfig as ProtoSSODiscoverConfig,
} from "@proto/sso/v1/sso_pb";
import { create, toBinary, fromBinary } from "@bufbuild/protobuf";
import { getSSOService } from "@/lib/wasm-core";
import type { SSOConfig } from "@/lib/api/ssoTypes";

// ============== Wire conversion (proto -> snake_case web shape) ==============

function fromProtoSSOConfig(p: ProtoSSODiscoverConfig): SSOConfig {
  return {
    domain: p.domain,
    name: p.name,
    protocol: p.protocol as SSOConfig["protocol"],
    enforce_sso: p.enforceSso,
  };
}

// LDAPAuthUser mirrors `user` sub-object in the legacy REST response —
// the web `LDAPAuthResponse` type defines `user.name` as `string |
// undefined`; map proto's `name?: string` accordingly. Numeric ID
// stays a number (proto int64 → bigint, but the web shape uses number
// for login flow — the IDs fit in 2^53).
function fromProtoLdapAuthUser(u: ProtoLdapAuthUser): LDAPAuthUserShape {
  return {
    id: Number(u.id),
    email: u.email,
    username: u.username,
    name: u.name,
  };
}

export interface LDAPAuthUserShape {
  id: number;
  email: string;
  username: string;
  name?: string;
}

export interface LDAPAuthResponseShape {
  token: string;
  refresh_token: string;
  expires_at: string;
  token_type: string;
  user: LDAPAuthUserShape;
}

function fromProtoLdapAuthResponse(p: ProtoLdapAuthResponse): LDAPAuthResponseShape {
  return {
    token: p.token,
    refresh_token: p.refreshToken,
    expires_at: p.expiresAt,
    token_type: p.tokenType,
    user: p.user
      ? fromProtoLdapAuthUser(p.user)
      : { id: 0, email: "", username: "" },
  };
}

// ============== SSOService — PUBLIC (no auth) ==============

export async function discover(email: string): Promise<{ configs: SSOConfig[] }> {
  const req = create(DiscoverRequestSchema, { email });
  const bytes = toBinary(DiscoverRequestSchema, req);
  const respBytes = await getSSOService().discoverConnect(bytes);
  const resp = fromBinary(DiscoverResponseSchema, new Uint8Array(respBytes));
  return { configs: resp.items.map(fromProtoSSOConfig) };
}

export async function ldapAuth(
  domain: string,
  username: string,
  password: string,
): Promise<LDAPAuthResponseShape> {
  const req = create(LdapAuthRequestSchema, { domain, username, password });
  const bytes = toBinary(LdapAuthRequestSchema, req);
  const respBytes = await getSSOService().ldapAuthConnect(bytes);
  return fromProtoLdapAuthResponse(
    fromBinary(LdapAuthResponseSchema, new Uint8Array(respBytes)),
  );
}
