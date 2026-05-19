// /api/v1/users/me — the only authenticated user-info endpoint that the
// (auth) route group needs. Three pages (invite acceptance, onboarding,
// runners/authorize) read it just to display "Signed in as <email>" /
// derive a default workspace name without booting wasm. Centralising it
// here avoids three slightly-different inline `MeResponse` interfaces.
// The function returns `null` on any failure (network, 401, parse error)
// — these pages use it as best-effort UI sugar and must keep working
// when the call fails.

import { lightFetch } from "./api-fetch";

export interface LightUser {
  id: number;
  email: string;
  username: string;
  name?: string;
  avatar_url?: string;
}

interface MeResponse {
  user?: LightUser;
}

export async function lightFetchMe(): Promise<LightUser | null> {
  try {
    const resp = await lightFetch<MeResponse>("/api/v1/users/me", { authenticated: true });
    return resp?.user ?? null;
  } catch {
    return null;
  }
}
