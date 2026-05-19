import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import {
  lightGetRunnerAuthStatus,
  lightAuthorizeRunner,
  lightCreateRunnerToken,
  lightListRunners,
} from "./runners";
import { writeLightSession, resolveLightBaseUrl } from "@/lib/light-session";

const ORIGIN = resolveLightBaseUrl();

function primeAuth() {
  writeLightSession({
    accessToken: "tok",
    refreshToken: "r",
    expiresAt: Math.floor(Date.now() / 1000) + 3600,
    baseUrl: ORIGIN,
  });
}

describe("lightGetRunnerAuthStatus", () => {
  let originalFetch: typeof fetch;

  beforeEach(() => {
    originalFetch = globalThis.fetch;
    window.localStorage.clear();
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
    window.localStorage.clear();
  });

  it("encodes key as query param and returns the status payload", async () => {
    const fetchSpy = vi.fn(async () =>
      new Response(
        JSON.stringify({
          status: "authorized",
          node_id: "node-1",
          expires_at: "2026-12-31T00:00:00Z",
        }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      ),
    );
    globalThis.fetch = fetchSpy as typeof fetch;

    const status = await lightGetRunnerAuthStatus("abc+key/with space");

    expect(status.status).toBe("authorized");
    expect(status.node_id).toBe("node-1");
    const [url, init] = fetchSpy.mock.calls[0];
    expect(String(url)).toBe(
      `${ORIGIN}/api/v1/runners/grpc/auth-status?key=abc%2Bkey%2Fwith+space`,
    );
    expect((init as RequestInit).method ?? "GET").toBe("GET");
  });
});

describe("lightAuthorizeRunner", () => {
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

  it("POSTs to /api/v1/orgs/:slug/runners/grpc/authorize with auth_key + node_id", async () => {
    const fetchSpy = vi.fn(async () =>
      new Response(
        JSON.stringify({ runner_id: 42, node_id: "node-42", message: "ok" }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      ),
    );
    globalThis.fetch = fetchSpy as typeof fetch;

    const resp = await lightAuthorizeRunner({
      organizationSlug: "dev-org",
      authKey: "key-1",
      nodeId: "node-42",
    });

    expect(resp.runner_id).toBe(42);
    const [url, init] = fetchSpy.mock.calls[0];
    expect(String(url)).toBe(
      `${ORIGIN}/api/v1/orgs/dev-org/runners/grpc/authorize`,
    );
    expect((init as RequestInit).method).toBe("POST");
    expect((init as RequestInit).body).toBe(
      JSON.stringify({ auth_key: "key-1", node_id: "node-42" }),
    );
    const headers = (init as RequestInit).headers as Record<string, string>;
    expect(headers.Authorization).toBe("Bearer tok");
  });

  it("encodes org slug containing special chars", async () => {
    const fetchSpy = vi.fn(async () =>
      new Response(JSON.stringify({ runner_id: 1 }), { status: 200 }),
    );
    globalThis.fetch = fetchSpy as typeof fetch;

    await lightAuthorizeRunner({
      organizationSlug: "ns/dev",
      authKey: "k",
    });

    const [url] = fetchSpy.mock.calls[0];
    expect(String(url)).toBe(
      `${ORIGIN}/api/v1/orgs/ns%2Fdev/runners/grpc/authorize`,
    );
  });

  it("omits node_id from payload when not provided", async () => {
    const fetchSpy = vi.fn(async () =>
      new Response(JSON.stringify({ runner_id: 1 }), { status: 200 }),
    );
    globalThis.fetch = fetchSpy as typeof fetch;

    await lightAuthorizeRunner({ organizationSlug: "dev", authKey: "k" });

    const [, init] = fetchSpy.mock.calls[0];
    expect((init as RequestInit).body).toBe(
      JSON.stringify({ auth_key: "k" }),
    );
  });
});

describe("lightCreateRunnerToken", () => {
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

  it("POSTs to /api/v1/orgs/:slug/runners/grpc/tokens and returns the token", async () => {
    const fetchSpy = vi.fn(async () =>
      new Response(
        JSON.stringify({ token: "reg-token-xyz", expires_at: "..." }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      ),
    );
    globalThis.fetch = fetchSpy as typeof fetch;

    const tok = await lightCreateRunnerToken("dev-org");

    expect(tok).toBe("reg-token-xyz");
    const [url, init] = fetchSpy.mock.calls[0];
    expect(String(url)).toBe(`${ORIGIN}/api/v1/orgs/dev-org/runners/grpc/tokens`);
    expect((init as RequestInit).method).toBe("POST");
    expect((init as RequestInit).body).toBe("{}");
  });

  it("returns null when token field is missing", async () => {
    globalThis.fetch = vi.fn(async () =>
      new Response("{}", { status: 200 }),
    ) as typeof fetch;
    const tok = await lightCreateRunnerToken("dev-org");
    expect(tok).toBeNull();
  });
});

describe("lightListRunners", () => {
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

  it("returns the runners array from /api/v1/orgs/:slug/runners", async () => {
    const fetchSpy = vi.fn(async () =>
      new Response(
        JSON.stringify({
          runners: [
            {
              id: 1,
              node_id: "n1",
              status: "online",
              current_pods: 0,
              max_concurrent_pods: 4,
              is_enabled: true,
            },
          ],
        }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      ),
    );
    globalThis.fetch = fetchSpy as typeof fetch;

    const runners = await lightListRunners("dev-org");

    expect(runners).toHaveLength(1);
    expect(runners[0].node_id).toBe("n1");
    const [url, init] = fetchSpy.mock.calls[0];
    expect(String(url)).toBe(`${ORIGIN}/api/v1/orgs/dev-org/runners`);
    const headers = (init as RequestInit).headers as Record<string, string>;
    expect(headers.Authorization).toBe("Bearer tok");
  });

  it("returns empty array when runners field is missing", async () => {
    globalThis.fetch = vi.fn(async () =>
      new Response("{}", { status: 200 }),
    ) as typeof fetch;
    const runners = await lightListRunners("dev-org");
    expect(runners).toEqual([]);
  });
});
