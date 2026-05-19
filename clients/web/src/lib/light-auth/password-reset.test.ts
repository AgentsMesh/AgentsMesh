import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { lightForgotPassword, lightResetPassword } from "./password-reset";
import { ApiError } from "@/lib/api/api-types";
import { sessionStorageKey, resolveLightBaseUrl } from "@/lib/light-session";

const ORIGIN = resolveLightBaseUrl();

function readBlob() {
  const raw = window.localStorage.getItem(sessionStorageKey(ORIGIN));
  return raw ? JSON.parse(raw) : null;
}

describe("lightForgotPassword", () => {
  let originalFetch: typeof fetch;

  beforeEach(() => {
    originalFetch = globalThis.fetch;
    window.localStorage.clear();
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
    window.localStorage.clear();
  });

  it("POSTs to /forgot-password with email payload", async () => {
    const fetchSpy = vi.fn(async () =>
      new Response(JSON.stringify({ message: "if-account-exists" }), { status: 200 }),
    );
    globalThis.fetch = fetchSpy as typeof fetch;

    await lightForgotPassword("user@example.com");

    const [url, init] = fetchSpy.mock.calls[0];
    expect(String(url)).toBe(`${ORIGIN}/api/v1/auth/forgot-password`);
    expect((init as RequestInit).method).toBe("POST");
    expect((init as RequestInit).body).toBe(
      JSON.stringify({ email: "user@example.com" }),
    );
    expect(readBlob()).toBeNull();
  });

  it("throws ApiError on 5xx server error", async () => {
    globalThis.fetch = vi.fn(async () =>
      new Response(JSON.stringify({ code: "INTERNAL", error: "boom" }), {
        status: 500,
        headers: { "Content-Type": "application/json" },
      }),
    ) as typeof fetch;

    let caught: unknown = null;
    try {
      await lightForgotPassword("user@example.com");
    } catch (e) {
      caught = e;
    }
    expect(caught).toBeInstanceOf(ApiError);
    expect((caught as ApiError).status).toBe(500);
  });
});

describe("lightResetPassword", () => {
  let originalFetch: typeof fetch;

  beforeEach(() => {
    originalFetch = globalThis.fetch;
    window.localStorage.clear();
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
    window.localStorage.clear();
  });

  it("POSTs token + new_password (snake_case) to /reset-password", async () => {
    const fetchSpy = vi.fn(async () =>
      new Response(JSON.stringify({ message: "ok" }), { status: 200 }),
    );
    globalThis.fetch = fetchSpy as typeof fetch;

    await lightResetPassword({ token: "reset-tok", newPassword: "FreshP@ss1" });

    const [url, init] = fetchSpy.mock.calls[0];
    expect(String(url)).toBe(`${ORIGIN}/api/v1/auth/reset-password`);
    expect((init as RequestInit).method).toBe("POST");
    expect((init as RequestInit).body).toBe(
      JSON.stringify({ token: "reset-tok", new_password: "FreshP@ss1" }),
    );
    expect(readBlob()).toBeNull();
  });

  it("throws ApiError when token is invalid or expired", async () => {
    globalThis.fetch = vi.fn(async () =>
      new Response(
        JSON.stringify({ code: "TOKEN_EXPIRED", error: "expired" }),
        { status: 400, headers: { "Content-Type": "application/json" } },
      ),
    ) as typeof fetch;

    let caught: unknown = null;
    try {
      await lightResetPassword({ token: "old", newPassword: "FreshP@ss1" });
    } catch (e) {
      caught = e;
    }
    expect(caught).toBeInstanceOf(ApiError);
    expect((caught as ApiError).status).toBe(400);
    expect((caught as ApiError).hasCode("TOKEN_EXPIRED")).toBe(true);
  });
});
