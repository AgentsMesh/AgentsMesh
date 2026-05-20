import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import {
  lightListOrganizations,
  lightCreateOrganization,
  lightCreatePersonalOrganization,
} from "./organizations";
import { ApiError } from "@/lib/api/api-types";
import {
  writeLightSession,
  sessionStorageKey,
  resolveLightBaseUrl,
} from "@/lib/light-session";

const ORIGIN = resolveLightBaseUrl();

function readBlob() {
  const raw = window.localStorage.getItem(sessionStorageKey(ORIGIN));
  return raw ? JSON.parse(raw) : null;
}

function primeAuth() {
  writeLightSession({
    accessToken: "tok",
    refreshToken: "r",
    expiresAt: Math.floor(Date.now() / 1000) + 3600,
    baseUrl: ORIGIN,
  });
}

describe("lightListOrganizations", () => {
  let originalFetch: typeof fetch;

  beforeEach(() => {
    originalFetch = globalThis.fetch;
    window.localStorage.clear();
    primeAuth();
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
    window.localStorage.clear();
  });

  it("returns the organizations array on 200", async () => {
    const fetchSpy = vi.fn(async () =>
      new Response(
        JSON.stringify({
          organizations: [
            { id: 1, name: "Alpha", slug: "alpha" },
            { id: 2, name: "Beta", slug: "beta" },
          ],
        }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      ),
    );
    globalThis.fetch = fetchSpy as typeof fetch;

    const orgs = await lightListOrganizations();

    expect(orgs).toHaveLength(2);
    expect(orgs[0].slug).toBe("alpha");
    const [url, init] = fetchSpy.mock.calls[0];
    expect(String(url)).toBe(`${ORIGIN}/api/v1/orgs`);
    const headers = (init as RequestInit).headers as Record<string, string>;
    expect(headers.Authorization).toBe("Bearer tok");
  });

  it("returns empty array when organizations field is missing", async () => {
    globalThis.fetch = vi.fn(async () =>
      new Response("{}", { status: 200 }),
    ) as typeof fetch;
    const orgs = await lightListOrganizations();
    expect(orgs).toEqual([]);
  });

  it("throws ApiError on 401 unauthorized", async () => {
    globalThis.fetch = vi.fn(async () =>
      new Response(JSON.stringify({ code: "UNAUTHORIZED" }), {
        status: 401,
        headers: { "Content-Type": "application/json" },
      }),
    ) as typeof fetch;

    let caught: unknown = null;
    try {
      await lightListOrganizations();
    } catch (e) {
      caught = e;
    }
    expect(caught).toBeInstanceOf(ApiError);
    expect((caught as ApiError).status).toBe(401);
  });
});

describe("lightCreateOrganization", () => {
  let originalFetch: typeof fetch;

  beforeEach(() => {
    originalFetch = globalThis.fetch;
    window.localStorage.clear();
    primeAuth();
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
    window.localStorage.clear();
  });

  it("POSTs and updates current_org_slug to the new org's slug", async () => {
    const fetchSpy = vi.fn(async () =>
      new Response(
        JSON.stringify({
          organization: { id: 7, name: "Gamma", slug: "gamma-co" },
        }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      ),
    );
    globalThis.fetch = fetchSpy as typeof fetch;

    const org = await lightCreateOrganization({ name: "Gamma", slug: "gamma-co" });

    expect(org.slug).toBe("gamma-co");
    const [url, init] = fetchSpy.mock.calls[0];
    expect(String(url)).toBe(`${ORIGIN}/api/v1/orgs`);
    expect((init as RequestInit).method).toBe("POST");
    expect((init as RequestInit).body).toBe(
      JSON.stringify({ name: "Gamma", slug: "gamma-co" }),
    );
    expect(readBlob().current_org_slug).toBe("gamma-co");
  });

  it("throws when 200 response has no organization payload", async () => {
    globalThis.fetch = vi.fn(async () =>
      new Response("{}", { status: 200 }),
    ) as typeof fetch;

    await expect(
      lightCreateOrganization({ name: "X", slug: "x" }),
    ).rejects.toThrow(
      "organizations.create returned 200 with no organization payload",
    );
  });

  it("propagates ApiError on 409 slug conflict", async () => {
    globalThis.fetch = vi.fn(async () =>
      new Response(JSON.stringify({ code: "SLUG_TAKEN" }), {
        status: 409,
        headers: { "Content-Type": "application/json" },
      }),
    ) as typeof fetch;

    let caught: unknown = null;
    try {
      await lightCreateOrganization({ name: "Dup", slug: "dup" });
    } catch (e) {
      caught = e;
    }
    expect(caught).toBeInstanceOf(ApiError);
    expect((caught as ApiError).status).toBe(409);
    expect(readBlob().current_org_slug).toBeNull();
  });
});

describe("lightCreatePersonalOrganization", () => {
  let originalFetch: typeof fetch;

  beforeEach(() => {
    originalFetch = globalThis.fetch;
    window.localStorage.clear();
    primeAuth();
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
    window.localStorage.clear();
  });

  it("POSTs to /api/v1/orgs/personal with empty body and no slug", async () => {
    const fetchSpy = vi.fn(async () =>
      new Response(
        JSON.stringify({
          organization: { id: 1, name: "kudin-private's Workspace", slug: "kudin-private-workspace" },
        }),
        { status: 201, headers: { "Content-Type": "application/json" } },
      ),
    );
    globalThis.fetch = fetchSpy as typeof fetch;

    const org = await lightCreatePersonalOrganization();

    expect(org.slug).toBe("kudin-private-workspace");
    const [url, init] = fetchSpy.mock.calls[0];
    expect(String(url)).toBe(`${ORIGIN}/api/v1/orgs/personal`);
    expect((init as RequestInit).method).toBe("POST");
    // Body is empty {} — caller does NOT send slug; server derives it.
    expect((init as RequestInit).body).toBe("{}");
    expect(readBlob().current_org_slug).toBe("kudin-private-workspace");
  });

  it("throws when 200 response has no organization payload", async () => {
    globalThis.fetch = vi.fn(async () =>
      new Response("{}", { status: 200 }),
    ) as typeof fetch;
    await expect(lightCreatePersonalOrganization()).rejects.toThrow(
      "organizations.createPersonal returned 200 with no organization payload",
    );
  });

  it("propagates ApiError on rate limit (429)", async () => {
    globalThis.fetch = vi.fn(async () =>
      new Response(JSON.stringify({ code: "RATE_LIMITED" }), {
        status: 429,
        headers: { "Content-Type": "application/json" },
      }),
    ) as typeof fetch;

    let caught: unknown = null;
    try {
      await lightCreatePersonalOrganization();
    } catch (e) {
      caught = e;
    }
    expect(caught).toBeInstanceOf(ApiError);
    expect((caught as ApiError).status).toBe(429);
  });
});
