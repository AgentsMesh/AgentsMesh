import { create } from "zustand";
import { useMemo } from "react";
import { initWasmCore, getAuthManager, getApiClient } from "@/lib/wasm-core";
import { getErrorMessage } from "@/lib/utils";

interface User {
  id: number;
  email: string;
  username: string;
  name?: string;
  avatar_url?: string;
}

interface Organization {
  id: number;
  name: string;
  slug: string;
  role?: string;
  logo_url?: string;
  subscription_plan?: string;
  subscription_status?: string;
  created_at?: string;
  updated_at?: string;
}

interface AuthState {
  _tick: number;
  _hasHydrated: boolean;
  error: string | null;

  login: (email: string, password: string) => Promise<void>;
  fetchOrganizations: () => Promise<void>;
  switchOrg: (slug: string) => void;
  refreshSession: () => Promise<void>;

  setAuth: (token: string, user: User, refreshToken?: string) => void;
  setOrganizations: (orgs: Organization[]) => void;
  setCurrentOrg: (org: Organization) => void;
  logout: () => void;
  isAuthenticated: () => boolean;
  setHasHydrated: (state: boolean) => void;
  clearError: () => void;
}

const mgr = () => getAuthManager();
const client = () => getApiClient();
const bump = () => useAuthStore.setState((s) => ({ _tick: s._tick + 1 }));

// Selector helpers: Rust is SSOT — these read from AuthManager on every tick.

function parseJson<T>(raw: unknown): T | null {
  if (raw == null) return null;
  if (typeof raw === "string") {
    if (raw === "") return null;
    try { return JSON.parse(raw) as T; } catch { return null; }
  }
  return raw as T;
}

export function readCurrentUser(): User | null {
  try { return parseJson<User>(mgr().get_current_user_json()); } catch { return null; }
}

export function readCurrentOrg(): Organization | null {
  try { return parseJson<Organization>(mgr().get_current_org_json()); } catch { return null; }
}

export function readOrganizations(): Organization[] {
  try {
    const raw = mgr().get_organizations_json();
    if (!raw) return [];
    return JSON.parse(raw) as Organization[];
  } catch { return []; }
}

export function useCurrentUser(): User | null {
  const tick = useAuthStore((s) => s._tick);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  return useMemo(() => readCurrentUser(), [tick]);
}

export function useCurrentOrg(): Organization | null {
  const tick = useAuthStore((s) => s._tick);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  return useMemo(() => readCurrentOrg(), [tick]);
}

export function useAuthOrganizations(): Organization[] {
  const tick = useAuthStore((s) => s._tick);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  return useMemo(() => readOrganizations(), [tick]);
}

export const useAuthStore = create<AuthState>((set) => ({
  _tick: 0,
  _hasHydrated: false,
  error: null,

  login: async (email, password) => {
    await initWasmCore();
    try {
      const sessionJson = await mgr().login(email, password);
      const session = JSON.parse(sessionJson);

      client().set_token(session.token, session.refresh_token);

      const orgsJson = await mgr().fetch_organizations();
      const orgs: Organization[] = JSON.parse(orgsJson);
      if (orgs[0]) client().set_org_slug(orgs[0].slug);
      bump();
    } catch (e) {
      set({ error: getErrorMessage(e, "Login failed") });
      throw e;
    }
  },

  fetchOrganizations: async () => {
    await initWasmCore();
    try {
      await mgr().fetch_organizations();
      const current = readCurrentOrg();
      if (current) client().set_org_slug(current.slug);
      bump();
    } catch (e) {
      set({ error: getErrorMessage(e, "Failed to fetch organizations") });
    }
  },

  switchOrg: (slug) => {
    mgr().switch_org(slug);
    client().set_org_slug(slug);
    bump();
  },

  refreshSession: async () => {
    await initWasmCore();
    try {
      const tokenJson = await mgr().refresh_token();
      const { token, refresh_token } = JSON.parse(tokenJson);
      client().set_token(token, refresh_token);
    } catch (e) {
      set({ error: getErrorMessage(e, "Session refresh failed") });
      throw e;
    }
  },

  setAuth: (token, user, refreshToken) => {
    try {
      client().set_token(token, refreshToken || "");
      // Persist user into Rust AuthManager so selectors see it immediately.
      mgr().apply_session(JSON.stringify({
        token,
        refresh_token: refreshToken || "",
        user,
      }));
    } catch { /* WASM not ready yet */ }
    set({ error: null });
    bump();
  },

  setOrganizations: (organizations) => {
    try { mgr().set_organizations(JSON.stringify(organizations)); } catch { /* noop */ }
    const current = readCurrentOrg();
    if (current) {
      try { client().set_org_slug(current.slug); } catch { /* noop */ }
    }
    bump();
  },

  setCurrentOrg: (org) => {
    try { mgr().set_current_org(JSON.stringify(org)); } catch { /* noop */ }
    try { client().set_org_slug(org.slug); } catch { /* noop */ }
    bump();
  },

  logout: () => {
    try { mgr().logout().catch(() => {}); } catch { /* noop */ }
    try { mgr().clear_session(); } catch { /* noop */ }
    try { client().clear_auth(); } catch { /* noop */ }
    set({ error: null });
    bump();
  },

  isAuthenticated: () => readCurrentUser() !== null,
  setHasHydrated: (state) => set({ _hasHydrated: state }),
  clearError: () => set({ error: null }),
}));

export type { User, Organization };
