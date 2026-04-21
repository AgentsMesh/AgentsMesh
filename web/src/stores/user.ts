import { create } from "zustand";
import { useMemo } from "react";
import { getUserState } from "@/lib/wasm-core";

export interface UserIdentity {
  id: number;
  provider: "github" | "google" | "gitlab" | "gitee";
  provider_user_id: string;
  provider_username?: string;
  created_at: string;
}

export interface User {
  id: number;
  email: string;
  username: string;
  name?: string;
  avatar_url?: string;
  is_active: boolean;
  last_login_at?: string;
  created_at: string;
  updated_at: string;
}

export interface UserProfile extends User {
  identities: UserIdentity[];
  organizations: Array<{
    id: number;
    name: string;
    slug: string;
    role: string;
  }>;
}

export function readProfile(): UserProfile | null {
  const val = getUserState().profile_json();
  if (!val) return null;
  return typeof val === "string" ? JSON.parse(val) : val;
}

interface UserState {
  _tick: number;
  isLoading: boolean;
  error: string | null;
  setProfile: (profile: UserProfile | null) => void;
  updateProfile: (updates: Partial<UserProfile>) => void;
  addIdentity: (identity: UserIdentity) => void;
  removeIdentity: (provider: string) => void;
  setLoading: (loading: boolean) => void;
  setError: (error: string | null) => void;
  reset: () => void;
}

const bump = () => useUserStore.setState((s) => ({ _tick: s._tick + 1 }));

export function useUserProfile(): UserProfile | null {
  const tick = useUserStore((s) => s._tick);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  return useMemo(() => readProfile(), [tick]);
}

export const useUserStore = create<UserState>((set) => ({
  _tick: 0, isLoading: false, error: null,

  setProfile: (profile) => {
    getUserState().set_profile(profile ? JSON.stringify(profile) : "");
    bump();
  },

  updateProfile: (updates) => {
    const cur = readProfile();
    if (!cur) return;
    getUserState().set_profile(JSON.stringify({ ...cur, ...updates }));
    bump();
  },

  addIdentity: (identity) => {
    getUserState().add_identity(JSON.stringify(identity));
    bump();
  },

  removeIdentity: (provider) => {
    const cur = readProfile();
    if (!cur) return;
    getUserState().set_profile(JSON.stringify({
      ...cur,
      identities: cur.identities.filter((i) => i.provider !== provider),
    }));
    bump();
  },

  setLoading: (isLoading) => set({ isLoading }),
  setError: (error) => set({ error }),

  reset: () => {
    getUserState().set_profile("");
    set({ _tick: 0, isLoading: false, error: null });
  },
}));
