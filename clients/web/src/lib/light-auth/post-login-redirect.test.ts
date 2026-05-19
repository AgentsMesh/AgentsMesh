import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { fetchFirstOrgSlug, resolvePostLoginUrlLight } from "./post-login-redirect";
import { writeLightSession } from "@/lib/light-session";

const ORIGIN = "http://localhost:10000";

describe("resolvePostLoginUrlLight", () => {
  let originalFetch: typeof fetch;

  beforeEach(() => {
    originalFetch = globalThis.fetch;
    window.localStorage.clear();
    writeLightSession({
      accessToken: "tok",
      refreshToken: "r",
      expiresAt: Math.floor(Date.now() / 1000) + 3600,
      baseUrl: ORIGIN,
    });
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
    window.localStorage.clear();
  });

  it("honours a safe ?redirect= param", async () => {
    globalThis.fetch = vi.fn(async () => new Response("{}", { status: 200 })) as typeof fetch;
    const url = await resolvePostLoginUrlLight({ redirectParam: "/foo/bar" });
    expect(url).toBe("/foo/bar");
  });

  it("rejects unsafe redirect and falls back to first-org default route", async () => {
    globalThis.fetch = vi.fn(async () =>
      new Response(JSON.stringify({ organizations: [{ id: 1, name: "Dev", slug: "dev-org" }] }), {
        status: 200,
      }),
    ) as typeof fetch;
    const url = await resolvePostLoginUrlLight({ redirectParam: "//evil.com" });
    // getDefaultRoute returns `/<slug>/workspace` (desktop) or `/<slug>/channels` (mobile)
    expect(url.startsWith("/dev-org/")).toBe(true);
  });

  it("falls back to /onboarding when no orgs", async () => {
    globalThis.fetch = vi.fn(async () =>
      new Response(JSON.stringify({ organizations: [] }), { status: 200 }),
    ) as typeof fetch;
    const url = await resolvePostLoginUrlLight({ redirectParam: null });
    expect(url).toBe("/onboarding");
  });

  it("falls back to /onboarding when the org list fetch fails", async () => {
    globalThis.fetch = vi.fn(async () => new Response("server error", { status: 500 })) as typeof fetch;
    const url = await resolvePostLoginUrlLight({ redirectParam: null });
    expect(url).toBe("/onboarding");
  });
});

describe("fetchFirstOrgSlug", () => {
  let originalFetch: typeof fetch;

  beforeEach(() => {
    originalFetch = globalThis.fetch;
    window.localStorage.clear();
    writeLightSession({
      accessToken: "tok",
      refreshToken: "r",
      expiresAt: Math.floor(Date.now() / 1000) + 3600,
      baseUrl: ORIGIN,
    });
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
    window.localStorage.clear();
  });

  it("returns the first org's slug", async () => {
    globalThis.fetch = vi.fn(async () =>
      new Response(JSON.stringify({
        organizations: [
          { id: 1, name: "Alpha", slug: "alpha" },
          { id: 2, name: "Beta", slug: "beta" },
        ],
      }), { status: 200 }),
    ) as typeof fetch;
    expect(await fetchFirstOrgSlug()).toBe("alpha");
  });

  it("returns null when list is empty", async () => {
    globalThis.fetch = vi.fn(async () =>
      new Response(JSON.stringify({ organizations: [] }), { status: 200 }),
    ) as typeof fetch;
    expect(await fetchFirstOrgSlug()).toBeNull();
  });
});
