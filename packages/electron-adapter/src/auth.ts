import { invoke } from "./invoke";
import type { IAuthManager } from "@agentsmesh/service-interface";

// Mirrors clients/core/crates/auth/src/token_store.rs `REFRESH_FALLBACK_TTL_SECS`.
// When the backend omits `expires_in`, both sides use the same default so the
// TS adapter and the Rust manager agree on `is_authenticated()` outcomes.
const REFRESH_FALLBACK_TTL_SECS = 3600;

function ttlFromResponse(expiresIn: number | undefined): number {
  if (expiresIn != null) return expiresIn;
  // eslint-disable-next-line no-console
  console.warn(
    `auth: server did not return expires_in; falling back to ${REFRESH_FALLBACK_TTL_SECS}s`,
  );
  return REFRESH_FALLBACK_TTL_SECS;
}

export class ElectronAuthService implements IAuthManager {
  private _token: string | undefined;
  private _refreshToken: string | undefined;
  private _expiresAt: number | undefined;
  private _organizationsCache = "[]";
  private _currentUserCache: string | null = null;
  private _currentOrgCache: string | null = null;

  readonly base_url: string;

  constructor(baseUrl: string) {
    this.base_url = baseUrl;
  }

  get_token(): string | undefined { return this._token; }
  get_refresh_token(): string | undefined { return this._refreshToken; }

  // Mirrors `AuthManager::is_authenticated` in Rust: token is "authenticated"
  // only if it's both present AND not expired. Without the expires_at check,
  // RootRedirect sends users to dashboard with an expired token → 401 → loop.
  is_authenticated(): boolean {
    if (!this._token) return false;
    if (this._expiresAt == null) return false;
    return this._expiresAt > Math.floor(Date.now() / 1000);
  }

  get_current_user_json(): unknown { return this._currentUserCache; }
  get_current_org_json(): unknown { return this._currentOrgCache; }
  get_organizations_json(): string { return this._organizationsCache; }

  // Mirrors AuthTokenStore::get_current_org_slug on the Rust side. Used by
  // ElectronApiClientProxy.org_path() so URL prefix tracks current org
  // without renderer-side `set_org_slug()`.
  get_current_org_slug(): string | undefined {
    if (!this._currentOrgCache) return undefined;
    try {
      const org = JSON.parse(this._currentOrgCache) as { slug?: string };
      return org.slug;
    } catch {
      return undefined;
    }
  }

  switch_org(slug: string): void {
    invoke("authSwitchOrg", slug).catch(() => {});
    const orgs = JSON.parse(this._organizationsCache) as { slug: string }[];
    const org = orgs.find(o => o.slug === slug);
    if (org) this._currentOrgCache = JSON.stringify(org);
  }

  async bootstrap(): Promise<string> {
    const result = await invoke<string>("authBootstrap");
    try {
      const parsed = JSON.parse(result) as {
        kind: "anonymous" | "authenticated" | "anonymous_after_cleanup";
        user?: unknown;
        current_org?: unknown;
      };
      if (parsed.kind === "authenticated") {
        this._token = await invoke<string | undefined>("authGetToken");
        this._expiresAt = (await invoke<number | undefined>("authGetExpiresAt")) ?? undefined;
        this._currentUserCache = parsed.user != null ? JSON.stringify(parsed.user) : null;
        this._currentOrgCache = parsed.current_org != null ? JSON.stringify(parsed.current_org) : null;
        this._organizationsCache = (await invoke<string>("authGetOrganizationsJson")) ?? "[]";
      } else {
        // Bootstrap reports anonymous → main already has no session; only
        // wipe renderer cache. Skip the authClearSession IPC that
        // clear_session() would fan out (redundant + an extra round-trip
        // on every cold start).
        this.resetLocalCache();
      }
    } catch {
      // Parse failure: leave caches as-is, caller treats as anonymous via JSON
    }
    return result;
  }

  async login(email: string, password: string): Promise<string> {
    const result = await invoke<string>("authLogin", email, password);
    this.applySessionPayload(result);
    return result;
  }

  async logout(): Promise<void> {
    await invoke<void>("authLogout");
    // authLogout already cleared the Rust manager's session; only the
    // renderer cache needs wiping here. clear_session() would re-fan-out
    // authClearSession — correct but wasteful.
    this.resetLocalCache();
  }

  async refresh_token(): Promise<string> {
    const result = await invoke<string>("authRefreshToken");
    const parsed = JSON.parse(result) as { token?: string; refresh_token?: string; expires_in?: number };
    this._token = parsed.token;
    this._refreshToken = parsed.refresh_token;
    this._expiresAt = Math.floor(Date.now() / 1000) + ttlFromResponse(parsed.expires_in);
    return result;
  }

  async fetch_organizations(): Promise<string> {
    const result = await invoke<string>("authFetchOrganizations");
    this._organizationsCache = result;
    // also refresh current org
    this._currentOrgCache = (await invoke<string | null>("authGetCurrentOrgJson")) ?? null;
    return result;
  }

  // Mirror state to BOTH ends: renderer cache (sync) AND main-process Rust
  // AuthManager (via IPC). The renderer-only path was the v0.31.x OAuth
  // deep-link bug — token landed in ElectronAuthService but never made it
  // to ApiClient (which reads from the Rust manager), so the next
  // `userGetMe` IPC 401'd and surfaced as `{"kind":"auth_expired"}`.
  // Callers MUST `await` because the IAuthManager signature now returns
  // Promise<void> | void; on the wasm side this is a no-op `await`.
  async apply_session(sessionJson: string): Promise<void> {
    // Push to Rust first so any IPC issued after `apply_session` returns
    // (e.g. userGetMe in OAuthCallbackPage) sees the token on the main side.
    await invoke("authApplySession", sessionJson);
    this.applySessionPayload(sessionJson);
  }

  async set_organizations(orgsJson: string): Promise<void> {
    await invoke("authSetOrganizations", orgsJson);
    this._organizationsCache = orgsJson;
  }

  async set_current_org(orgJson: string): Promise<void> {
    this._currentOrgCache = orgJson;
    await invoke("authSetCurrentOrg", orgJson);
  }

  async clear_session(): Promise<void> {
    // Same reasoning as apply_session: the Rust AuthManager is the SSOT
    // for ApiClient's token, so renderer-only clears leave a logged-in
    // main process. logout() already covers this for the API-logout path;
    // clear_session is the local-reset used by Zustand on OAuth-failure
    // cleanup, where the IPC fan-out is still required.
    await invoke("authClearSession");
    this.resetLocalCache();
  }

  static new_with_storage(baseUrl: string, _storage: unknown): IAuthManager {
    return new ElectronAuthService(baseUrl);
  }

  private resetLocalCache(): void {
    this._token = undefined;
    this._refreshToken = undefined;
    this._expiresAt = undefined;
    this._currentUserCache = null;
    this._currentOrgCache = null;
    this._organizationsCache = "[]";
  }

  // Single source of truth for "I just got a session payload from server,
  // hydrate my caches" — used by both `login()` and `apply_session()` so the
  // expires_in fallback / user-cache write logic can't drift between them.
  private applySessionPayload(json: string): void {
    const parsed = JSON.parse(json) as {
      token?: string; refresh_token?: string; expires_in?: number; user?: unknown;
    };
    if (parsed.token) this._token = parsed.token;
    if (parsed.refresh_token) this._refreshToken = parsed.refresh_token;
    this._expiresAt = Math.floor(Date.now() / 1000) + ttlFromResponse(parsed.expires_in);
    if (parsed.user != null) this._currentUserCache = JSON.stringify(parsed.user);
  }
}
