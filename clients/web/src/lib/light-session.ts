// Pure-TS reader/writer for the PersistedSession blob.
// Marketing pages and the (auth) route group read & write auth state through
// this module instead of pulling 40MB of wasm just to render Sign In / handle
// /login. The schema and url_slug algorithm mirror Rust SSOT at:
//   clients/core/crates/auth/src/state.rs (PersistedSession + url_slug)
// MUST NOT import from @/lib/wasm-core, @agentsmesh/service-runtime, or
// agentsmesh-wasm — that would defeat the whole purpose.

import { getApiBaseUrl } from "@/lib/env";

export interface LightSession {
  currentOrgSlug: string | null;
  isAuthenticated: boolean;
  expiresAt: number;
}

// Cross-language SSOT pin. MUST stay equal to:
//   clients/core/crates/auth/src/state.rs::SCHEMA_VERSION
// If you bump this number on either side, you also need to:
//   1. Decide what to do with blobs written by older clients still in the
//      wild (forward-compat? cleanup? force re-login?).
//   2. Confirm Rust bootstrap doesn't reject newer/unknown versions
//      (today it doesn't — `#[serde(default)]` is tolerant — but a
//      future version gate would silently break light writes here).
// In short: don't touch in isolation. Cross-language change requires
// coordinating both files in one commit + a migration plan.
const SCHEMA_VERSION = 1;
const NAMESPACE_PREFIX = "agentsmesh-auth";

// Mirrors Rust state.rs::url_slug — keep in sync. Same algorithm runs in
// e2e-playwright/fixtures/blockstore.fixture.ts (live cross-check).
export function urlSlug(baseUrl: string): string {
  try {
    const u = new URL(baseUrl);
    const port = u.port ? `_${u.port}` : "";
    const raw = `${u.protocol.replace(":", "")}_${u.hostname.toLowerCase()}${port}`;
    return raw.replace(/[^a-zA-Z0-9]/g, "_").slice(0, 64);
  } catch {
    return baseUrl.toLowerCase().replace(/[^a-zA-Z0-9]/g, "_").slice(0, 64);
  }
}

export function sessionStorageKey(baseUrl: string): string {
  return `${NAMESPACE_PREFIX}/${urlSlug(baseUrl)}/session`;
}

// Resolve the canonical base_url light writers use. MUST stay byte-equal with
// the value wasm-core.ts feeds to WasmAuthManager — bootstrap clears the
// session if base_url disagrees (see bootstrap.rs::BaseUrlMismatch).
export function resolveLightBaseUrl(): string {
  const explicit = getApiBaseUrl();
  if (explicit) return explicit;
  // wasm-core falls back to window.location.origin when getApiBaseUrl() is
  // an empty string (local dev proxy mode); do the same here.
  if (typeof window !== "undefined") return window.location.origin;
  return "";
}

interface PersistedSessionWire {
  access_token?: string;
  refresh_token?: string;
  expires_at?: number;
  base_url?: string;
  current_org_slug?: string | null;
  schema_version?: number;
}

function readWire(baseUrl: string): PersistedSessionWire | null {
  if (typeof window === "undefined") return null;
  const raw = window.localStorage.getItem(sessionStorageKey(baseUrl));
  if (!raw) return null;
  try {
    return JSON.parse(raw) as PersistedSessionWire;
  } catch {
    return null;
  }
}

export function readLightSession(baseUrl?: string): LightSession | null {
  const url = baseUrl ?? resolveLightBaseUrl();
  const s = readWire(url);
  if (!s) return null;
  const now = Math.floor(Date.now() / 1000);
  return {
    currentOrgSlug: s.current_org_slug ?? null,
    expiresAt: s.expires_at ?? 0,
    isAuthenticated: !!s.access_token && (s.expires_at ?? 0) > now,
  };
}

export function readLightAuthToken(baseUrl?: string): string | null {
  const url = baseUrl ?? resolveLightBaseUrl();
  const s = readWire(url);
  if (!s?.access_token) return null;
  const now = Math.floor(Date.now() / 1000);
  if ((s.expires_at ?? 0) <= now) return null;
  return s.access_token;
}

export interface PersistedSessionWriteInput {
  accessToken: string;
  refreshToken: string;
  expiresAt: number;
  currentOrgSlug?: string | null;
  baseUrl?: string;
}

export function writeLightSession(input: PersistedSessionWriteInput): void {
  if (typeof window === "undefined") return;
  const baseUrl = input.baseUrl ?? resolveLightBaseUrl();
  const blob: PersistedSessionWire = {
    access_token: input.accessToken,
    refresh_token: input.refreshToken,
    expires_at: input.expiresAt,
    base_url: baseUrl,
    current_org_slug: input.currentOrgSlug ?? null,
    schema_version: SCHEMA_VERSION,
  };
  window.localStorage.setItem(sessionStorageKey(baseUrl), JSON.stringify(blob));
}

export function updateLightSessionOrgSlug(orgSlug: string | null, baseUrl?: string): void {
  if (typeof window === "undefined") return;
  const url = baseUrl ?? resolveLightBaseUrl();
  const existing = readWire(url);
  if (!existing) return;
  existing.current_org_slug = orgSlug;
  window.localStorage.setItem(sessionStorageKey(url), JSON.stringify(existing));
}

export function clearLightSession(baseUrl?: string): void {
  if (typeof window === "undefined") return;
  const url = baseUrl ?? resolveLightBaseUrl();
  window.localStorage.removeItem(sessionStorageKey(url));
}
