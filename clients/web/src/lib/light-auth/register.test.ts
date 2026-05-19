import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { lightRegister } from "./register";
import { ApiError } from "@/lib/api/api-types";
import { sessionStorageKey, resolveLightBaseUrl } from "@/lib/light-session";

const ORIGIN = resolveLightBaseUrl();

function readBlob() {
  const raw = window.localStorage.getItem(sessionStorageKey(ORIGIN));
  return raw ? JSON.parse(raw) : null;
}

describe("lightRegister", () => {
  let originalFetch: typeof fetch;

  beforeEach(() => {
    originalFetch = globalThis.fetch;
    window.localStorage.clear();
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
    window.localStorage.clear();
  });

  it("POSTs to /api/v1/auth/register with full payload and persists session", async () => {
    const fetchSpy = vi.fn(async () =>
      new Response(
        JSON.stringify({
          token: "reg-tok",
          refresh_token: "reg-ref",
          expires_in: 3600,
          user: {
            id: 2,
            email: "new@b.c",
            username: "newbie",
            is_email_verified: false,
          },
        }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      ),
    );
    globalThis.fetch = fetchSpy as typeof fetch;

    const resp = await lightRegister({
      email: "new@b.c",
      username: "newbie",
      password: "StrongP@ss1",
      name: "New User",
    });

    expect(resp.token).toBe("reg-tok");
    const [url, init] = fetchSpy.mock.calls[0];
    expect(String(url)).toBe(`${ORIGIN}/api/v1/auth/register`);
    expect((init as RequestInit).method).toBe("POST");
    expect((init as RequestInit).body).toBe(
      JSON.stringify({
        email: "new@b.c",
        username: "newbie",
        password: "StrongP@ss1",
        name: "New User",
      }),
    );
    expect(readBlob().access_token).toBe("reg-tok");
  });

  it("throws ApiError on 409 duplicate email without persisting", async () => {
    globalThis.fetch = vi.fn(async () =>
      new Response(
        JSON.stringify({ code: "EMAIL_TAKEN", error: "email already registered" }),
        { status: 409, headers: { "Content-Type": "application/json" } },
      ),
    ) as typeof fetch;

    let caught: unknown = null;
    try {
      await lightRegister({
        email: "dup@b.c",
        username: "dup",
        password: "x",
        name: "Dup",
      });
    } catch (e) {
      caught = e;
    }
    expect(caught).toBeInstanceOf(ApiError);
    expect((caught as ApiError).status).toBe(409);
    expect((caught as ApiError).hasCode("EMAIL_TAKEN")).toBe(true);
    expect(readBlob()).toBeNull();
  });

  it("throws ApiError on 400 weak password without persisting", async () => {
    globalThis.fetch = vi.fn(async () =>
      new Response(
        JSON.stringify({ code: "WEAK_PASSWORD", error: "password too short" }),
        { status: 400, headers: { "Content-Type": "application/json" } },
      ),
    ) as typeof fetch;

    let caught: unknown = null;
    try {
      await lightRegister({
        email: "weak@b.c",
        username: "weak",
        password: "1",
        name: "Weak",
      });
    } catch (e) {
      caught = e;
    }
    expect(caught).toBeInstanceOf(ApiError);
    expect((caught as ApiError).status).toBe(400);
    expect((caught as ApiError).hasCode("WEAK_PASSWORD")).toBe(true);
    expect(readBlob()).toBeNull();
  });
});
