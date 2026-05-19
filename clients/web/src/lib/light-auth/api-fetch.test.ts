import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { lightFetch } from "./api-fetch";
import { ApiError } from "@/lib/api/api-types";
import { writeLightSession, clearLightSession } from "@/lib/light-session";

const ORIGIN = "http://localhost:10000";

function mockFetch(impl: typeof fetch) {
  globalThis.fetch = impl as typeof fetch;
}

describe("lightFetch", () => {
  let originalFetch: typeof fetch;

  beforeEach(() => {
    originalFetch = globalThis.fetch;
    window.localStorage.clear();
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
    window.localStorage.clear();
  });

  it("sends a GET with Accept header and parses JSON", async () => {
    const fetchSpy = vi.fn(async () =>
      new Response(JSON.stringify({ ok: true }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );
    mockFetch(fetchSpy);

    const result = await lightFetch<{ ok: boolean }>("/api/v1/ping", { baseUrl: ORIGIN });

    expect(result).toEqual({ ok: true });
    const [url, init] = fetchSpy.mock.calls[0];
    expect(url).toBe(`${ORIGIN}/api/v1/ping`);
    expect((init as RequestInit).method).toBe("GET");
    const headers = (init as RequestInit).headers as Record<string, string>;
    expect(headers.Accept).toBe("application/json");
    expect(headers["Content-Type"]).toBeUndefined();
  });

  it("serializes body and sets Content-Type for POST", async () => {
    const fetchSpy = vi.fn(async () => new Response("{}", { status: 200 }));
    mockFetch(fetchSpy);

    await lightFetch("/api/v1/auth/login", {
      method: "POST",
      body: { email: "a@b.c", password: "x" },
      baseUrl: ORIGIN,
    });

    const [, init] = fetchSpy.mock.calls[0];
    expect((init as RequestInit).method).toBe("POST");
    expect((init as RequestInit).body).toBe(JSON.stringify({ email: "a@b.c", password: "x" }));
    const headers = (init as RequestInit).headers as Record<string, string>;
    expect(headers["Content-Type"]).toBe("application/json");
  });

  it("injects Bearer token when authenticated=true and session is fresh", async () => {
    writeLightSession({
      accessToken: "live-tok",
      refreshToken: "r",
      expiresAt: Math.floor(Date.now() / 1000) + 3600,
      baseUrl: ORIGIN,
    });
    const fetchSpy = vi.fn(async () => new Response("{}", { status: 200 }));
    mockFetch(fetchSpy);

    await lightFetch("/api/v1/orgs", { authenticated: true, baseUrl: ORIGIN });

    const [, init] = fetchSpy.mock.calls[0];
    const headers = (init as RequestInit).headers as Record<string, string>;
    expect(headers.Authorization).toBe("Bearer live-tok");
  });

  it("omits Authorization header when no token is stored", async () => {
    clearLightSession(ORIGIN);
    const fetchSpy = vi.fn(async () => new Response("{}", { status: 200 }));
    mockFetch(fetchSpy);

    await lightFetch("/api/v1/anything", { authenticated: true, baseUrl: ORIGIN });

    const headers = (fetchSpy.mock.calls[0][1] as RequestInit).headers as Record<string, string>;
    expect(headers.Authorization).toBeUndefined();
  });

  it("encodes query params", async () => {
    const fetchSpy = vi.fn(async () => new Response("{}", { status: 200 }));
    mockFetch(fetchSpy);

    await lightFetch("/api/v1/auth/sso/discover", {
      query: { email: "user@example.com", limit: 5 },
      baseUrl: ORIGIN,
    });

    const [url] = fetchSpy.mock.calls[0];
    expect(url).toBe(`${ORIGIN}/api/v1/auth/sso/discover?email=user%40example.com&limit=5`);
  });

  it("throws ApiError on 4xx and preserves data + code", async () => {
    mockFetch(vi.fn(async () =>
      new Response(JSON.stringify({ code: "INVALID_CREDENTIALS", error: "wrong password" }), {
        status: 401,
        headers: { "Content-Type": "application/json" },
      }),
    ));

    await expect(
      lightFetch("/api/v1/auth/login", { method: "POST", body: {}, baseUrl: ORIGIN }),
    ).rejects.toMatchObject({
      status: 401,
    });

    try {
      await lightFetch("/api/v1/auth/login", { method: "POST", body: {}, baseUrl: ORIGIN });
    } catch (e) {
      expect(e).toBeInstanceOf(ApiError);
      const err = e as ApiError;
      expect(err.code).toBe("INVALID_CREDENTIALS");
      expect(err.serverMessage).toBe("wrong password");
      expect(err.hasCode("INVALID_CREDENTIALS")).toBe(true);
    }
  });

  it("returns undefined for 204 responses", async () => {
    mockFetch(vi.fn(async () => new Response(null, { status: 204 })));
    const result = await lightFetch("/api/v1/auth/logout", { method: "POST", baseUrl: ORIGIN });
    expect(result).toBeUndefined();
  });

  it("handles empty body without throwing", async () => {
    mockFetch(vi.fn(async () => new Response("", { status: 200 })));
    const result = await lightFetch("/api/v1/empty", { baseUrl: ORIGIN });
    expect(result).toBeUndefined();
  });
});
