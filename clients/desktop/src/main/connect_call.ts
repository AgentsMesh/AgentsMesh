import { ipcMain } from "electron";
import { logEvent, type AppState } from "@agentsmesh/node-bridge";
import { connectFetch } from "./connect-fetch";
import { connectErrorFromResponse, connectNetworkError } from "./connect-error";

interface Deps {
  // Getters (not values): AppState + resolved API URL rebind on server switch,
  // so each call reads the current ones.
  getAppState: () => AppState;
  getApiUrl: () => string;
}

export interface ConnectCaller {
  callConnectJson(service: string, method: string, payload?: unknown): Promise<string>;
  orgSlug(): string;
}

// Main → backend Connect egress over the JSON wire. Used by the legacy IPC
// aliases (legacy_api_aliases.ts). Rewrites the `unauthenticated` envelope to
// the `auth_expired` token the renderer + e2e specs key off.
export function makeConnectCaller({ getAppState, getApiUrl }: Deps): ConnectCaller {
  const callConnectJson = async (service: string, method: string, payload: unknown = {}): Promise<string> => {
    const url = `${getApiUrl()}/${service}/${method}`;
    const token = (getAppState() as { authGetToken?: () => string | null }).authGetToken?.();
    const headers: Record<string, string> = {
      "Content-Type": "application/json",
      "Connect-Protocol-Version": "1",
    };
    if (token) headers.Authorization = `Bearer ${token}`;
    const res = await connectFetch(url, { method: "POST", headers, body: JSON.stringify(payload) });
    if (!res.ok) {
      const body = await res.text().catch(() => "");
      logEvent("warn", "ipc-rpc", `${service}/${method} → ${res.status}`);
      const message = body.includes("unauthenticated")
        ? `auth_expired ${res.status} ${url} ${body}`
        : `${res.status} ${res.statusText} ${url} ${body}`;
      throw new Error(message.trim());
    }
    return await res.text();
  };

  const orgSlug = () => {
    const raw = (getAppState() as { authGetCurrentOrgJson?: () => string | null }).authGetCurrentOrgJson?.();
    if (!raw) return "";
    try { return (JSON.parse(raw) as { slug?: string }).slug ?? ""; }
    catch { return ""; }
  };

  return { callConnectJson, orgSlug };
}

// Generic binary Connect-RPC proxy — NOT a legacy alias. Web's wasm services
// expose `<method>Connect(Uint8Array)`; ElectronXxxService adapters without a
// hand-written `_connect` handler route through this. Protobuf encode/decode
// stays on the renderer; main only ferries bytes (number[]) to the backend —
// the renderer has no wasm, so this is the sole egress point for those RPCs.
export function registerConnectCall(
  { getAppState, getApiUrl }: Deps,
  tracked: Set<string>,
): void {
  // Remove before re-register so rebind (which runs bindAppStateHandlers first,
  // clearing tracked) can re-register with fresh closures. Safe at boot too.
  ipcMain.removeHandler("connectCall");
  tracked.add("connectCall");
  ipcMain.handle("connectCall", async (_e, service: string, method: string, bodyArr: number[]) => {
    const url = `${getApiUrl()}/${service}/${method}`;
    const token = (getAppState() as { authGetToken?: () => string | null }).authGetToken?.();
    const headers: Record<string, string> = {
      "Content-Type": "application/proto",
      "Connect-Protocol-Version": "1",
    };
    if (token) headers.Authorization = `Bearer ${token}`;
    logEvent("debug", "ipc-rpc", `${service}/${method} (${bodyArr.length}B)`);
    let res: Response;
    try {
      res = await connectFetch(url, { method: "POST", headers, body: Uint8Array.from(bodyArr) });
    } catch (err) {
      logEvent("warn", "ipc-rpc", `${service}/${method} → network error`);
      throw connectNetworkError(err);
    }
    if (!res.ok) {
      logEvent("warn", "ipc-rpc", `${service}/${method} → ${res.status}`);
      throw await connectErrorFromResponse(res);
    }
    return Array.from(new Uint8Array(await res.arrayBuffer()));
  });
}
