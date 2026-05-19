import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { lightDiscoverSSO, lightLdapAuth } from "./sso-discover";
import { ApiError } from "@/lib/api/api-types";
import { sessionStorageKey, resolveLightBaseUrl } from "@/lib/light-session";

const ORIGIN = resolveLightBaseUrl();

function readBlob() {
  const raw = window.localStorage.getItem(sessionStorageKey(ORIGIN));
  return raw ? JSON.parse(raw) : null;
}

describe("lightDiscoverSSO", () => {
  let originalFetch: typeof fetch;

  beforeEach(() => {
    originalFetch = globalThis.fetch;
    window.localStorage.clear();
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
    window.localStorage.clear();
  });

  it("returns [] without firing fetch for empty email", async () => {
    const fetchSpy = vi.fn();
    globalThis.fetch = fetchSpy as unknown as typeof fetch;

    const configs = await lightDiscoverSSO("");

    expect(configs).toEqual([]);
    expect(fetchSpy).not.toHaveBeenCalled();
  });

  it("returns [] without firing fetch when email lacks @", async () => {
    const fetchSpy = vi.fn();
    globalThis.fetch = fetchSpy as unknown as typeof fetch;

    const configs = await lightDiscoverSSO("not-an-email");

    expect(configs).toEqual([]);
    expect(fetchSpy).not.toHaveBeenCalled();
  });

  it("returns the configs array when SSO is configured for the domain", async () => {
    const fetchSpy = vi.fn(async () =>
      new Response(
        JSON.stringify({
          configs: [
            { domain: "b.c", name: "Corp OIDC", protocol: "oidc", enforce_sso: true },
            { domain: "b.c", name: "Corp LDAP", protocol: "ldap", enforce_sso: false },
          ],
        }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      ),
    );
    globalThis.fetch = fetchSpy as typeof fetch;

    const configs = await lightDiscoverSSO("user@b.c");

    expect(configs).toHaveLength(2);
    expect(configs[0].protocol).toBe("oidc");
    const [url] = fetchSpy.mock.calls[0];
    expect(String(url)).toBe(
      `${ORIGIN}/api/v1/auth/sso/discover?email=user%40b.c`,
    );
  });

  it("returns [] when 200 response has no configs field", async () => {
    globalThis.fetch = vi.fn(async () =>
      new Response("{}", { status: 200 }),
    ) as typeof fetch;
    const configs = await lightDiscoverSSO("user@b.c");
    expect(configs).toEqual([]);
  });

  it("propagates ApiError on server failure", async () => {
    globalThis.fetch = vi.fn(async () =>
      new Response(JSON.stringify({ code: "INTERNAL" }), {
        status: 500,
        headers: { "Content-Type": "application/json" },
      }),
    ) as typeof fetch;

    let caught: unknown = null;
    try {
      await lightDiscoverSSO("user@b.c");
    } catch (e) {
      caught = e;
    }
    expect(caught).toBeInstanceOf(ApiError);
    expect((caught as ApiError).status).toBe(500);
  });
});

describe("lightLdapAuth", () => {
  let originalFetch: typeof fetch;

  beforeEach(() => {
    originalFetch = globalThis.fetch;
    window.localStorage.clear();
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
    window.localStorage.clear();
  });

  it("POSTs to /sso/:domain/ldap and persists session on 200", async () => {
    const fetchSpy = vi.fn(async () =>
      new Response(
        JSON.stringify({
          token: "ldap-tok",
          refresh_token: "ldap-ref",
          expires_in: 3600,
          user: { id: 11, email: "u@b.c", username: "u" },
        }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      ),
    );
    globalThis.fetch = fetchSpy as typeof fetch;

    const resp = await lightLdapAuth({
      domain: "corp.example",
      username: "alice",
      password: "p@ss",
    });

    expect(resp.token).toBe("ldap-tok");
    const [url, init] = fetchSpy.mock.calls[0];
    expect(String(url)).toBe(
      `${ORIGIN}/api/v1/auth/sso/corp.example/ldap`,
    );
    expect((init as RequestInit).method).toBe("POST");
    expect((init as RequestInit).body).toBe(
      JSON.stringify({ username: "alice", password: "p@ss" }),
    );
    expect(readBlob().access_token).toBe("ldap-tok");
  });

  it("URL-encodes domain segment with special chars", async () => {
    const fetchSpy = vi.fn(async () =>
      new Response(
        JSON.stringify({ token: "t", refresh_token: "r", expires_in: 3600 }),
        { status: 200 },
      ),
    );
    globalThis.fetch = fetchSpy as typeof fetch;

    await lightLdapAuth({ domain: "ns/corp", username: "u", password: "p" });

    const [url] = fetchSpy.mock.calls[0];
    expect(String(url)).toBe(`${ORIGIN}/api/v1/auth/sso/ns%2Fcorp/ldap`);
  });

  it("throws ApiError on bad credentials without persisting", async () => {
    globalThis.fetch = vi.fn(async () =>
      new Response(JSON.stringify({ code: "BAD_CREDENTIALS" }), {
        status: 401,
        headers: { "Content-Type": "application/json" },
      }),
    ) as typeof fetch;

    let caught: unknown = null;
    try {
      await lightLdapAuth({ domain: "corp.example", username: "u", password: "wrong" });
    } catch (e) {
      caught = e;
    }
    expect(caught).toBeInstanceOf(ApiError);
    expect((caught as ApiError).status).toBe(401);
    expect(readBlob()).toBeNull();
  });
});
