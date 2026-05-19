// Pure fetch wrapper for the (auth) route group — no wasm, no agentsmesh-wasm.
// Throws @/lib/api/api-types::ApiError on 4xx/5xx so existing UI code that
// matches `err instanceof ApiError && err.hasCode(...)` keeps working
// unchanged. Bearer tokens are pulled lazily from localStorage via
// readLightAuthToken so callers don't need to thread them through.

import { ApiError } from "@/lib/api/api-types";
import { readLightAuthToken, resolveLightBaseUrl } from "@/lib/light-session";

type Method = "GET" | "POST" | "PUT" | "DELETE" | "PATCH";

export interface LightFetchOptions {
  method?: Method;
  body?: unknown;
  authenticated?: boolean;
  signal?: AbortSignal;
  query?: Record<string, string | number | boolean | undefined | null>;
  baseUrl?: string;
}

function appendQuery(path: string, query?: LightFetchOptions["query"]): string {
  if (!query) return path;
  const params = new URLSearchParams();
  for (const [k, v] of Object.entries(query)) {
    if (v === undefined || v === null) continue;
    params.set(k, String(v));
  }
  const qs = params.toString();
  if (!qs) return path;
  return path.includes("?") ? `${path}&${qs}` : `${path}?${qs}`;
}

export async function lightFetch<T = unknown>(
  path: string,
  options: LightFetchOptions = {},
): Promise<T> {
  const baseUrl = options.baseUrl ?? resolveLightBaseUrl();
  const url = `${baseUrl}${appendQuery(path, options.query)}`;
  const headers: Record<string, string> = { Accept: "application/json" };
  if (options.body !== undefined) headers["Content-Type"] = "application/json";
  if (options.authenticated) {
    const token = readLightAuthToken(baseUrl);
    if (token) headers["Authorization"] = `Bearer ${token}`;
  }

  const resp = await fetch(url, {
    method: options.method ?? "GET",
    headers,
    body: options.body !== undefined ? JSON.stringify(options.body) : undefined,
    signal: options.signal,
    credentials: "include",
  });

  if (!resp.ok) {
    let data: unknown = null;
    try { data = await resp.json(); } catch { /* ignore */ }
    throw new ApiError(resp.status, resp.statusText, data);
  }

  if (resp.status === 204) return undefined as T;
  const text = await resp.text();
  if (!text) return undefined as T;
  try { return JSON.parse(text) as T; } catch { return undefined as T; }
}
