import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { lightLogin } from "./login";
import { ApiError } from "@/lib/api/api-types";
import { sessionStorageKey, resolveLightBaseUrl } from "@/lib/light-session";

const ORIGIN = resolveLightBaseUrl();

function readBlob() {
  const raw = window.localStorage.getItem(sessionStorageKey(ORIGIN));
  return raw ? JSON.parse(raw) : null;
}

describe("lightLogin", () => {
  let originalFetch: typeof fetch;

  beforeEach(() => {
    originalFetch = globalThis.fetch;
    window.localStorage.clear();
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
    window.localStorage.clear();
  });

  it("POSTs to /api/v1/auth/login and persists session on 200", async () => {
    const now = Math.floor(Date.now() / 1000);
    const fetchSpy = vi.fn(async () =>
      new Response(
        JSON.stringify({
          token: "access-1",
          refresh_token: "refresh-1",
          expires_in: 3600,
          user: { id: 1, email: "a@b.c", username: "alice" },
        }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      ),
    );
    globalThis.fetch = fetchSpy as typeof fetch;

    const resp = await lightLogin({ email: "a@b.c", password: "secret" });

    expect(resp.token).toBe("access-1");
    expect(resp.refresh_token).toBe("refresh-1");
    const [url, init] = fetchSpy.mock.calls[0];
    expect(String(url)).toBe(`${ORIGIN}/api/v1/auth/login`);
    expect((init as RequestInit).method).toBe("POST");
    expect((init as RequestInit).body).toBe(
      JSON.stringify({ email: "a@b.c", password: "secret" }),
    );
    const blob = readBlob();
    expect(blob.access_token).toBe("access-1");
    expect(blob.refresh_token).toBe("refresh-1");
    expect(blob.schema_version).toBe(1);
    expect(blob.expires_at).toBeGreaterThanOrEqual(now + 3590);
  });

  it("throws ApiError with 401 code on invalid credentials and does not persist", async () => {
    globalThis.fetch = vi.fn(async () =>
      new Response(
        JSON.stringify({ code: "INVALID_CREDENTIALS", error: "wrong password" }),
        { status: 401, headers: { "Content-Type": "application/json" } },
      ),
    ) as typeof fetch;

    let caught: unknown = null;
    try {
      await lightLogin({ email: "a@b.c", password: "bad" });
    } catch (e) {
      caught = e;
    }
    expect(caught).toBeInstanceOf(ApiError);
    expect((caught as ApiError).status).toBe(401);
    expect((caught as ApiError).hasCode("INVALID_CREDENTIALS")).toBe(true);
    expect(readBlob()).toBeNull();
  });

  it("propagates raw network errors", async () => {
    globalThis.fetch = vi.fn(async () => {
      throw new TypeError("network down");
    }) as typeof fetch;
    await expect(lightLogin({ email: "a@b.c", password: "x" })).rejects.toThrow(
      "network down",
    );
    expect(readBlob()).toBeNull();
  });
});
