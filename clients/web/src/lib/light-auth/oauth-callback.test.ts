import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { consumeOAuthCallbackParams } from "./oauth-callback";
import { sessionStorageKey, resolveLightBaseUrl } from "@/lib/light-session";

const ORIGIN = resolveLightBaseUrl();

function readBlob() {
  const raw = window.localStorage.getItem(sessionStorageKey(ORIGIN));
  return raw ? JSON.parse(raw) : null;
}

describe("consumeOAuthCallbackParams", () => {
  beforeEach(() => window.localStorage.clear());
  afterEach(() => window.localStorage.clear());

  it("returns error when params.error is set, without writing localStorage", () => {
    const params = new URLSearchParams({ error: "access_denied" });
    const result = consumeOAuthCallbackParams(params);
    expect(result).toEqual({ status: "error", reason: "access_denied" });
    expect(readBlob()).toBeNull();
  });

  it("returns missing_token error when neither token nor error present", () => {
    const params = new URLSearchParams();
    const result = consumeOAuthCallbackParams(params);
    expect(result).toEqual({ status: "error", reason: "missing_token" });
    expect(readBlob()).toBeNull();
  });

  it("persists session and returns ok when token + refresh_token present", () => {
    const now = Math.floor(Date.now() / 1000);
    const params = new URLSearchParams({
      token: "oauth-access",
      refresh_token: "oauth-refresh",
    });
    const result = consumeOAuthCallbackParams(params);
    expect(result).toEqual({
      status: "ok",
      token: "oauth-access",
      refreshToken: "oauth-refresh",
    });
    const blob = readBlob();
    expect(blob.access_token).toBe("oauth-access");
    expect(blob.refresh_token).toBe("oauth-refresh");
    expect(blob.schema_version).toBe(1);
    expect(blob.current_org_slug).toBeNull();
    expect(blob.expires_at).toBeGreaterThanOrEqual(now + 3590);
    expect(blob.expires_at).toBeLessThanOrEqual(now + 3610);
  });

  it("defaults refresh_token to empty string when absent", () => {
    const params = new URLSearchParams({ token: "tok-only" });
    const result = consumeOAuthCallbackParams(params);
    expect(result).toEqual({ status: "ok", token: "tok-only", refreshToken: "" });
    expect(readBlob().refresh_token).toBe("");
  });

  it("accepts duck-typed params object with a get() method", () => {
    const mock = { get: (k: string): string | null => (k === "token" ? "ducked" : k === "refresh_token" ? "duck-ref" : null) };
    const result = consumeOAuthCallbackParams(mock);
    expect(result.status).toBe("ok");
    if (result.status === "ok") {
      expect(result.token).toBe("ducked");
      expect(result.refreshToken).toBe("duck-ref");
    }
  });

  it("prefers error over token when both are present", () => {
    const params = new URLSearchParams({ error: "server_error", token: "ignored" });
    const result = consumeOAuthCallbackParams(params);
    expect(result).toEqual({ status: "error", reason: "server_error" });
    expect(readBlob()).toBeNull();
  });
});
