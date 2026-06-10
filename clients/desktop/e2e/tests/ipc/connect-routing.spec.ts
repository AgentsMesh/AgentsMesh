import { test, expect } from "../../fixtures/electron-shared.fixture";

// Proxy-routed Connect services have no napi channel, so they're absent from the
// _generated/*.api.spec.ts contract — a wrong withConnectFallback derivation 404'd
// on desktop undetected (#434). Unregistered route → ServeMux "page not found";
// a reached handler → 200 or a connect-framed typed error.
test.describe.configure({ mode: "serial" });

// Routes mirror the override maps in packages/electron-adapter/src/connect-fallback.ts.
const ROUTES = [
  { label: "autopilot — reported 404", service: "proto.autopilot.v1.AutopilotControllerService", method: "ListAutopilotControllers" },
  { label: "billing — Subscription-suffix verb", service: "proto.billing.v1.BillingService", method: "ReactivateSubscription" },
  { label: "billing — public sibling service", service: "proto.billing.v1.BillingPublicService", method: "GetPublicPricing" },
  { label: "grant — newly wired facade", service: "proto.grant.v1.GrantService", method: "ListGrants" },
  { label: "pod — default 1:1 derivation", service: "proto.pod.v1.PodService", method: "ListPods" },
] as const;

test.describe("IPC · connect routing reaches the backend", () => {
  for (const { label, service, method } of ROUTES) {
    test(`${label}: ${service.split(".").pop()}/${method}`, async ({ sharedPage }) => {
      const err = await sharedPage.evaluate(async ({ s, m }) => {
        const api = (window as unknown as { electronAPI: { invoke: (c: string, ...a: unknown[]) => Promise<unknown> } }).electronAPI;
        try { await api.invoke("connectCall", s, m, []); return null; }
        catch (e) { return e instanceof Error ? e.message : String(e); }
      }, { s: service, m: method });

      if (err !== null) {
        expect(err, `${service}/${method} is unregistered — desktop-only 404`).not.toMatch(/page not found/i);
      }
    });
  }
});
