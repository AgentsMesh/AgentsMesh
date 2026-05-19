// Turns backend auth responses into a PersistedSession blob and writes it
// via writeLightSession. Dashboard's wasm bootstrap later picks the blob up,
// validates the token, and fetches /users/me + /users/me/organizations to
// hydrate full state — so we deliberately leave user and current_org_slug
// out of the blob (Rust SSOT only persists tokens + base_url + org_slug).

import { writeLightSession } from "@/lib/light-session";

export interface AuthLoginResponse {
  token: string;
  refresh_token: string;
  expires_in: number;
  user?: {
    id: number;
    email: string;
    username: string;
    name?: string;
    avatar_url?: string;
    is_email_verified?: boolean;
  };
}

const DEFAULT_EXPIRES_IN = 3600;

function computeExpiresAt(expiresIn?: number): number {
  return Math.floor(Date.now() / 1000) + (expiresIn && expiresIn > 0 ? expiresIn : DEFAULT_EXPIRES_IN);
}

export function persistLoginResponse(resp: AuthLoginResponse): void {
  writeLightSession({
    accessToken: resp.token,
    refreshToken: resp.refresh_token,
    expiresAt: computeExpiresAt(resp.expires_in),
    currentOrgSlug: null,
  });
}

export interface OAuthCallbackTokens {
  token: string;
  refreshToken: string;
  expiresIn?: number;
}

// OAuth callback redirect from the backend currently omits expires_in
// (auth_oauth.go:135-142 only forwards token + refresh_token). We fall back
// to a 1h window — bootstrap will refresh via /auth/refresh well before the
// real server-side expiry.
export function persistOAuthTokens(tokens: OAuthCallbackTokens): void {
  writeLightSession({
    accessToken: tokens.token,
    refreshToken: tokens.refreshToken,
    expiresAt: computeExpiresAt(tokens.expiresIn),
    currentOrgSlug: null,
  });
}
