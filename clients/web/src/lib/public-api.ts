// Public (auth-free) API fetcher for marketing pages.
// Marketing pages (/, /docs, ...) MUST NOT load wasm — they use this module
// instead of @/lib/wasm-core to read public endpoints over plain fetch. The
// pricing endpoint is the canonical example; add new public endpoints here.
// Backend route: backend/internal/api/rest/v1/billing_routes.go
//   RegisterPublicConfigRoutes mounts GET /api/v1/config/pricing (no auth).

import { getApiBaseUrl } from "./env";
import type { PublicPricingResponse, DeploymentInfo } from "@/lib/api/billing-types";

function resolveBase(): string {
  const cfg = getApiBaseUrl();
  if (cfg) return cfg;
  if (typeof window !== "undefined") return window.location.origin;
  return "";
}

async function getJson<T>(path: string): Promise<T> {
  const res = await fetch(`${resolveBase()}${path}`, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
  });
  if (!res.ok) throw new Error(`public api ${path} http ${res.status}`);
  return (await res.json()) as T;
}

export async function fetchPublicPricing(): Promise<PublicPricingResponse> {
  return getJson<PublicPricingResponse>("/api/v1/config/pricing");
}

export async function fetchPublicDeploymentInfo(): Promise<DeploymentInfo> {
  return getJson<DeploymentInfo>("/api/v1/config/deployment");
}
