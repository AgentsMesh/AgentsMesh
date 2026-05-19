import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { persistLoginResponse, persistOAuthTokens } from "./persist";
import { sessionStorageKey, readLightSession, resolveLightBaseUrl } from "@/lib/light-session";

// persist.ts pulls base_url from resolveLightBaseUrl() at call time, which
// in the jsdom test environment falls back to window.location.origin
// (http://localhost:3000 by default). Use the same source-of-truth here so
// the read-side and write-side namespaces match.
const ORIGIN = resolveLightBaseUrl();

function readBlob() {
  const raw = window.localStorage.getItem(sessionStorageKey(ORIGIN));
  return raw ? JSON.parse(raw) : null;
}

describe("persistLoginResponse", () => {
  beforeEach(() => window.localStorage.clear());
  afterEach(() => window.localStorage.clear());

  it("writes a Rust-compatible PersistedSession blob", () => {
    const now = Math.floor(Date.now() / 1000);
    persistLoginResponse({
      token: "tok",
      refresh_token: "ref",
      expires_in: 3600,
      user: { id: 42, email: "a@b.c", username: "alice" },
    });
    const blob = readBlob();
    expect(blob.access_token).toBe("tok");
    expect(blob.refresh_token).toBe("ref");
    expect(blob.schema_version).toBe(1);
    expect(blob.current_org_slug).toBeNull();
    expect(blob.base_url).toBe(ORIGIN);
    expect(blob.expires_at).toBeGreaterThanOrEqual(now + 3590);
    expect(blob.expires_at).toBeLessThanOrEqual(now + 3610);
  });

  it("falls back to 3600s when expires_in is missing or zero", () => {
    const now = Math.floor(Date.now() / 1000);
    persistLoginResponse({
      token: "tok",
      refresh_token: "ref",
      expires_in: 0,
    });
    const blob = readBlob();
    expect(blob.expires_at).toBeGreaterThanOrEqual(now + 3590);
  });

  it("makes the session readable through readLightSession", () => {
    persistLoginResponse({
      token: "tok",
      refresh_token: "ref",
      expires_in: 3600,
    });
    const session = readLightSession(ORIGIN);
    expect(session?.isAuthenticated).toBe(true);
  });
});

describe("persistOAuthTokens", () => {
  beforeEach(() => window.localStorage.clear());
  afterEach(() => window.localStorage.clear());

  it("writes session blob from OAuth callback tokens", () => {
    persistOAuthTokens({
      token: "oauth-tok",
      refreshToken: "oauth-ref",
    });
    const blob = readBlob();
    expect(blob.access_token).toBe("oauth-tok");
    expect(blob.refresh_token).toBe("oauth-ref");
    expect(blob.schema_version).toBe(1);
    expect(blob.current_org_slug).toBeNull();
  });

  it("uses provided expiresIn when present", () => {
    const now = Math.floor(Date.now() / 1000);
    persistOAuthTokens({
      token: "tok",
      refreshToken: "ref",
      expiresIn: 7200,
    });
    const blob = readBlob();
    expect(blob.expires_at).toBeGreaterThanOrEqual(now + 7190);
    expect(blob.expires_at).toBeLessThanOrEqual(now + 7210);
  });
});
