import { create } from "zustand";
import { useMemo } from "react";
import { initWasmCore, getAuthManager } from "@/lib/wasm-core";
import { getErrorMessage } from "@/lib/utils";
import { useWorkspaceStore } from "./workspace";
import { resetOrgScopedServices } from "@/lib/org-scope/registry";

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

interface BootstrapResult {
  kind: "anonymous" | "authenticated" | "anonymous_after_cleanup";
  user?: User;
  current_org?: Organization;
  reason?: string;
}

interface AuthState {
  _tick: number;
  _hasHydrated: boolean;
  error: string | null;

  bootstrap: () => Promise<BootstrapResult>;
  login: (email: string, password: string) => Promise<void>;
  fetchOrganizations: () => Promise<void>;
  switchOrg: (slug: string) => void;
  refreshSession: () => Promise<void>;

  // setAuth / setOrganizations / logout return Promise: on Electron the
  // underlying adapter awaits an IPC round-trip to the Rust SSOT. Callers
  // MUST await — fire-and-forget leaves the main process without the token
  // (the v0.31.x OAuth deep-link bug). Wasm path resolves synchronously.
  setAuth: (token: string, user: User, refreshToken?: string) => Promise<void>;
  setOrganizations: (orgs: Organization[]) => Promise<void>;
  setCurrentOrg: (org: Organization) => Promise<void>;
  logout: () => Promise<void>;
  isAuthenticated: () => boolean;
  setHasHydrated: (state: boolean) => void;
  clearError: () => void;
}

const mgr = () => getAuthManager();
const bump = () => useAuthStore.setState((s) => ({ _tick: s._tick + 1 }));

// Selector helpers: Rust is SSOT — these read from AuthManager on every tick.
// ApiClient shares AuthManager's token store (Plan I6), so token writes
// propagate without TS-side `client.set_token()` synchronization.

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

export function useIsAuthenticated(): boolean {
  const tick = useAuthStore((s) => s._tick);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  return useMemo(() => {
    try {
      return mgr().is_authenticated();
    } catch {
      return false;
    }
  }, [tick]);
}

export const useAuthStore = create<AuthState>((set) => ({
  _tick: 0,
  _hasHydrated: false,
  error: null,

  bootstrap: async () => {
    await initWasmCore();
    let result: BootstrapResult;
    try {
      const json = await mgr().bootstrap();
      result = JSON.parse(json) as BootstrapResult;
    } catch (e) {
      // Bootstrap call itself failed (network down before any storage hit).
      // Treat as anonymous; storage is untouched so retry on reload is safe.
      console.warn("auth bootstrap failed:", getErrorMessage(e));
      result = { kind: "anonymous" };
    }
    if (result.kind === "anonymous_after_cleanup") {
      console.warn(`auth bootstrap cleanup: ${result.reason}`);
    }
    bump();
    return result;
  },

  login: async (email, password) => {
    await initWasmCore();
    try {
      await mgr().login(email, password);
      await mgr().fetch_organizations();
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
      bump();
    } catch (e) {
      set({ error: getErrorMessage(e, "Failed to fetch organizations") });
    }
  },

  switchOrg: (slug) => {
    mgr().switch_org(slug);
    bump();
  },

  refreshSession: async () => {
    await initWasmCore();
    try {
      await mgr().refresh_token();
    } catch (e) {
      set({ error: getErrorMessage(e, "Session refresh failed") });
      throw e;
    }
  },

  setAuth: async (token, user, refreshToken) => {
    try {
      await mgr().apply_session(JSON.stringify({
        token,
        refresh_token: refreshToken || "",
        user,
      }));
    } catch { /* WASM not ready yet */ }
    set({ error: null });
    bump();
  },

  setOrganizations: async (organizations) => {
    try { await mgr().set_organizations(JSON.stringify(organizations)); } catch { /* noop */ }
    bump();
  },

  setCurrentOrg: async (org) => {
    try { await mgr().set_current_org(JSON.stringify(org)); } catch { /* noop */ }
    try {
      useWorkspaceStore.getState().clearAllPanes();
      useWorkspaceStore.persist.clearStorage?.();
    } catch { /* noop */ }
    resetOrgScopedServices();
    bump();
  },

  logout: async () => {
    // Sync local cleanup first — guarantees post-call state is logged-out
    // even if the network POST below fails or hangs.
    try { await mgr().clear_session(); } catch { /* noop */ }
    // Best-effort API logout (informs server, doesn't block UI).
    try { mgr().logout().catch(() => {}); } catch { /* noop */ }
    set({ error: null });
    bump();
  },

  isAuthenticated: () => {
    try {
      return mgr().is_authenticated();
    } catch {
      return false;
    }
  },
  setHasHydrated: (state) => set({ _hasHydrated: state }),
  clearError: () => set({ error: null }),
}));

export type { User, Organization };
