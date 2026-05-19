import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { lightFetchInvitation, lightAcceptInvitation } from "./invitations";
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

describe("lightFetchInvitation", () => {
  let originalFetch: typeof fetch;

  beforeEach(() => {
    originalFetch = globalThis.fetch;
    window.localStorage.clear();
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
    window.localStorage.clear();
  });

  it("returns InvitationInfo on 200", async () => {
    const fetchSpy = vi.fn(async () =>
      new Response(
        JSON.stringify({
          invitation: {
            id: 1,
            email: "invitee@b.c",
            role: "member",
            organization_id: 5,
            organization_name: "Dev",
            organization_slug: "dev-org",
            inviter_name: "alice",
            expires_at: "2026-12-31T00:00:00Z",
            is_expired: false,
          },
        }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      ),
    );
    globalThis.fetch = fetchSpy as typeof fetch;

    const info = await lightFetchInvitation("inv-token-123");

    expect(info?.organization_slug).toBe("dev-org");
    expect(info?.inviter_name).toBe("alice");
    const [url] = fetchSpy.mock.calls[0];
    expect(String(url)).toBe(`${ORIGIN}/api/v1/invitations/inv-token-123`);
  });

  it("URL-encodes the token", async () => {
    const fetchSpy = vi.fn(async () =>
      new Response("{}", { status: 200 }),
    );
    globalThis.fetch = fetchSpy as typeof fetch;

    await lightFetchInvitation("a/b c+d");

    const [url] = fetchSpy.mock.calls[0];
    expect(String(url)).toBe(`${ORIGIN}/api/v1/invitations/a%2Fb%20c%2Bd`);
  });

  it("returns null when invitation field is missing", async () => {
    globalThis.fetch = vi.fn(async () =>
      new Response("{}", { status: 200 }),
    ) as typeof fetch;
    const info = await lightFetchInvitation("inv-x");
    expect(info).toBeNull();
  });

  it("throws ApiError on 404 not found", async () => {
    globalThis.fetch = vi.fn(async () =>
      new Response(JSON.stringify({ code: "NOT_FOUND" }), {
        status: 404,
        headers: { "Content-Type": "application/json" },
      }),
    ) as typeof fetch;

    let caught: unknown = null;
    try {
      await lightFetchInvitation("gone");
    } catch (e) {
      caught = e;
    }
    expect(caught).toBeInstanceOf(ApiError);
    expect((caught as ApiError).status).toBe(404);
  });
});

describe("lightAcceptInvitation", () => {
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

  it("POSTs to /:token/accept and updates current_org_slug eagerly", async () => {
    const fetchSpy = vi.fn(async () =>
      new Response("{}", { status: 200 }),
    );
    globalThis.fetch = fetchSpy as typeof fetch;

    await lightAcceptInvitation("inv-token-9", "joined-org");

    const [url, init] = fetchSpy.mock.calls[0];
    expect(String(url)).toBe(`${ORIGIN}/api/v1/invitations/inv-token-9/accept`);
    expect((init as RequestInit).method).toBe("POST");
    const headers = (init as RequestInit).headers as Record<string, string>;
    expect(headers.Authorization).toBe("Bearer tok");
    expect(readBlob().current_org_slug).toBe("joined-org");
  });

  it("propagates ApiError on 409 already member", async () => {
    globalThis.fetch = vi.fn(async () =>
      new Response(JSON.stringify({ code: "ALREADY_MEMBER" }), {
        status: 409,
        headers: { "Content-Type": "application/json" },
      }),
    ) as typeof fetch;

    let caught: unknown = null;
    try {
      await lightAcceptInvitation("inv-x", "some-org");
    } catch (e) {
      caught = e;
    }
    expect(caught).toBeInstanceOf(ApiError);
    expect((caught as ApiError).status).toBe(409);
    expect(readBlob().current_org_slug).toBeNull();
  });
});
