import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { lightVerifyEmail, lightResendVerification } from "./verify-email";
import { ApiError } from "@/lib/api/api-types";
import { sessionStorageKey, resolveLightBaseUrl } from "@/lib/light-session";

const ORIGIN = resolveLightBaseUrl();

function readBlob() {
  const raw = window.localStorage.getItem(sessionStorageKey(ORIGIN));
  return raw ? JSON.parse(raw) : null;
}

describe("lightVerifyEmail", () => {
  let originalFetch: typeof fetch;

  beforeEach(() => {
    originalFetch = globalThis.fetch;
    window.localStorage.clear();
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
    window.localStorage.clear();
  });

  it("POSTs token and persists fresh session on 200", async () => {
    const fetchSpy = vi.fn(async () =>
      new Response(
        JSON.stringify({
          token: "verified-tok",
          refresh_token: "verified-ref",
          expires_in: 3600,
          user: { id: 3, email: "v@b.c", username: "verified", is_email_verified: true },
        }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      ),
    );
    globalThis.fetch = fetchSpy as typeof fetch;

    const resp = await lightVerifyEmail("magic-link-token");

    expect(resp.token).toBe("verified-tok");
    const [url, init] = fetchSpy.mock.calls[0];
    expect(String(url)).toBe(`${ORIGIN}/api/v1/auth/verify-email`);
    expect((init as RequestInit).method).toBe("POST");
    expect((init as RequestInit).body).toBe(JSON.stringify({ token: "magic-link-token" }));
    expect(readBlob().access_token).toBe("verified-tok");
  });

  it("throws ApiError on 400 invalid token without persisting", async () => {
    globalThis.fetch = vi.fn(async () =>
      new Response(
        JSON.stringify({ code: "INVALID_TOKEN", error: "token expired or invalid" }),
        { status: 400, headers: { "Content-Type": "application/json" } },
      ),
    ) as typeof fetch;

    let caught: unknown = null;
    try {
      await lightVerifyEmail("bad-token");
    } catch (e) {
      caught = e;
    }
    expect(caught).toBeInstanceOf(ApiError);
    expect((caught as ApiError).status).toBe(400);
    expect((caught as ApiError).hasCode("INVALID_TOKEN")).toBe(true);
    expect(readBlob()).toBeNull();
  });
});

describe("lightResendVerification", () => {
  let originalFetch: typeof fetch;

  beforeEach(() => {
    originalFetch = globalThis.fetch;
    window.localStorage.clear();
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
    window.localStorage.clear();
  });

  it("POSTs to /resend-verification and does not write localStorage", async () => {
    const fetchSpy = vi.fn(async () =>
      new Response(JSON.stringify({ message: "sent" }), { status: 200 }),
    );
    globalThis.fetch = fetchSpy as typeof fetch;

    await lightResendVerification("v@b.c");

    const [url, init] = fetchSpy.mock.calls[0];
    expect(String(url)).toBe(`${ORIGIN}/api/v1/auth/resend-verification`);
    expect((init as RequestInit).method).toBe("POST");
    expect((init as RequestInit).body).toBe(JSON.stringify({ email: "v@b.c" }));
    expect(readBlob()).toBeNull();
  });

  it("propagates server errors as ApiError", async () => {
    globalThis.fetch = vi.fn(async () =>
      new Response(JSON.stringify({ code: "RATE_LIMITED" }), {
        status: 429,
        headers: { "Content-Type": "application/json" },
      }),
    ) as typeof fetch;

    let caught: unknown = null;
    try {
      await lightResendVerification("v@b.c");
    } catch (e) {
      caught = e;
    }
    expect(caught).toBeInstanceOf(ApiError);
    expect((caught as ApiError).status).toBe(429);
  });
});
