import { describe, it, expect, beforeEach, afterEach } from "vitest";
import {
  urlSlug,
  sessionStorageKey,
  readLightSession,
  readLightAuthToken,
  writeLightSession,
  clearLightSession,
  updateLightSessionOrgSlug,
} from "./light-session";

describe("urlSlug", () => {
  it("normalizes scheme + host to lowercase + alphanumeric_underscore", () => {
    expect(urlSlug("https://AGENTSMESH.AI")).toBe("https_agentsmesh_ai");
  });

  it("includes port when present", () => {
    expect(urlSlug("http://localhost:10007")).toBe("http_localhost_10007");
  });

  it("strips trailing path / query / hash", () => {
    expect(urlSlug("https://agentsmesh.ai/path?q=1#h")).toBe("https_agentsmesh_ai");
  });

  it("caps at 64 chars", () => {
    const long = "https://" + "a".repeat(200) + ".com";
    expect(urlSlug(long).length).toBeLessThanOrEqual(64);
  });

  it("falls back gracefully on garbage input", () => {
    const slug = urlSlug("not a url");
    expect(slug).toMatch(/^[a-z0-9_]+$/);
  });
});

describe("sessionStorageKey", () => {
  it("formats as agentsmesh-auth/<slug>/session", () => {
    expect(sessionStorageKey("https://agentsmesh.ai"))
      .toBe("agentsmesh-auth/https_agentsmesh_ai/session");
  });
});

describe("readLightSession", () => {
  const ORIGIN = "http://localhost:10007";
  const KEY = sessionStorageKey(ORIGIN);

  beforeEach(() => {
    window.localStorage.clear();
  });

  afterEach(() => {
    window.localStorage.clear();
  });

  it("returns null when no session is stored", () => {
    expect(readLightSession(ORIGIN)).toBeNull();
  });

  it("parses a fresh session as authenticated", () => {
    const future = Math.floor(Date.now() / 1000) + 3600;
    window.localStorage.setItem(
      KEY,
      JSON.stringify({
        access_token: "tok",
        refresh_token: "r",
        expires_at: future,
        base_url: ORIGIN,
        current_org_slug: "dev-org",
        schema_version: 1,
      }),
    );
    const s = readLightSession(ORIGIN);
    expect(s).toEqual({
      currentOrgSlug: "dev-org",
      expiresAt: future,
      isAuthenticated: true,
    });
  });

  it("treats expired session as unauthenticated", () => {
    const past = Math.floor(Date.now() / 1000) - 60;
    window.localStorage.setItem(
      KEY,
      JSON.stringify({
        access_token: "tok",
        expires_at: past,
        current_org_slug: "dev-org",
      }),
    );
    const s = readLightSession(ORIGIN);
    expect(s?.isAuthenticated).toBe(false);
    expect(s?.currentOrgSlug).toBe("dev-org");
  });

  it("treats missing access_token as unauthenticated", () => {
    window.localStorage.setItem(
      KEY,
      JSON.stringify({ expires_at: 9_999_999_999 }),
    );
    const s = readLightSession(ORIGIN);
    expect(s?.isAuthenticated).toBe(false);
  });

  it("returns null on corrupt JSON", () => {
    window.localStorage.setItem(KEY, "{not json");
    expect(readLightSession(ORIGIN)).toBeNull();
  });

  it("isolates sessions across base_url namespaces", () => {
    const dev = "http://localhost:10007";
    const prod = "https://agentsmesh.ai";
    window.localStorage.setItem(
      sessionStorageKey(dev),
      JSON.stringify({ access_token: "dev", expires_at: 9_999_999_999, current_org_slug: "d" }),
    );
    window.localStorage.setItem(
      sessionStorageKey(prod),
      JSON.stringify({ access_token: "prod", expires_at: 9_999_999_999, current_org_slug: "p" }),
    );
    expect(readLightSession(dev)?.currentOrgSlug).toBe("d");
    expect(readLightSession(prod)?.currentOrgSlug).toBe("p");
  });
});

describe("writeLightSession", () => {
  const ORIGIN = "http://localhost:10007";
  const KEY = sessionStorageKey(ORIGIN);

  beforeEach(() => window.localStorage.clear());
  afterEach(() => window.localStorage.clear());

  it("writes a Rust-compatible PersistedSession blob", () => {
    const expiresAt = Math.floor(Date.now() / 1000) + 3600;
    writeLightSession({
      accessToken: "tok",
      refreshToken: "r",
      expiresAt,
      currentOrgSlug: "dev-org",
      baseUrl: ORIGIN,
    });
    const raw = window.localStorage.getItem(KEY);
    expect(raw).not.toBeNull();
    const parsed = JSON.parse(raw!);
    expect(parsed).toEqual({
      access_token: "tok",
      refresh_token: "r",
      expires_at: expiresAt,
      base_url: ORIGIN,
      current_org_slug: "dev-org",
      schema_version: 1,
    });
  });

  it("defaults currentOrgSlug to null", () => {
    writeLightSession({
      accessToken: "tok",
      refreshToken: "r",
      expiresAt: 9_999_999_999,
      baseUrl: ORIGIN,
    });
    const parsed = JSON.parse(window.localStorage.getItem(KEY)!);
    expect(parsed.current_org_slug).toBeNull();
  });

  it("round-trips with readLightSession", () => {
    const expiresAt = Math.floor(Date.now() / 1000) + 3600;
    writeLightSession({
      accessToken: "tok",
      refreshToken: "r",
      expiresAt,
      currentOrgSlug: "dev-org",
      baseUrl: ORIGIN,
    });
    const session = readLightSession(ORIGIN);
    expect(session?.isAuthenticated).toBe(true);
    expect(session?.currentOrgSlug).toBe("dev-org");
    expect(session?.expiresAt).toBe(expiresAt);
  });
});

describe("readLightAuthToken", () => {
  const ORIGIN = "http://localhost:10007";

  beforeEach(() => window.localStorage.clear());
  afterEach(() => window.localStorage.clear());

  it("returns the access_token when session is fresh", () => {
    writeLightSession({
      accessToken: "fresh-tok",
      refreshToken: "r",
      expiresAt: Math.floor(Date.now() / 1000) + 3600,
      baseUrl: ORIGIN,
    });
    expect(readLightAuthToken(ORIGIN)).toBe("fresh-tok");
  });

  it("returns null when session is expired", () => {
    writeLightSession({
      accessToken: "stale-tok",
      refreshToken: "r",
      expiresAt: Math.floor(Date.now() / 1000) - 60,
      baseUrl: ORIGIN,
    });
    expect(readLightAuthToken(ORIGIN)).toBeNull();
  });

  it("returns null when no session is stored", () => {
    expect(readLightAuthToken(ORIGIN)).toBeNull();
  });
});

describe("clearLightSession", () => {
  const ORIGIN = "http://localhost:10007";

  beforeEach(() => window.localStorage.clear());
  afterEach(() => window.localStorage.clear());

  it("removes the stored session", () => {
    writeLightSession({
      accessToken: "tok",
      refreshToken: "r",
      expiresAt: 9_999_999_999,
      baseUrl: ORIGIN,
    });
    expect(readLightSession(ORIGIN)).not.toBeNull();
    clearLightSession(ORIGIN);
    expect(readLightSession(ORIGIN)).toBeNull();
  });

  it("is a no-op when no session exists", () => {
    expect(() => clearLightSession(ORIGIN)).not.toThrow();
  });
});

describe("updateLightSessionOrgSlug", () => {
  const ORIGIN = "http://localhost:10007";

  beforeEach(() => window.localStorage.clear());
  afterEach(() => window.localStorage.clear());

  it("updates current_org_slug while preserving the rest of the session", () => {
    const expiresAt = Math.floor(Date.now() / 1000) + 3600;
    writeLightSession({
      accessToken: "tok",
      refreshToken: "r",
      expiresAt,
      currentOrgSlug: null,
      baseUrl: ORIGIN,
    });
    updateLightSessionOrgSlug("new-org", ORIGIN);
    const parsed = JSON.parse(window.localStorage.getItem(sessionStorageKey(ORIGIN))!);
    expect(parsed.current_org_slug).toBe("new-org");
    expect(parsed.access_token).toBe("tok");
    expect(parsed.expires_at).toBe(expiresAt);
    expect(parsed.schema_version).toBe(1);
  });

  it("does nothing when no session exists", () => {
    updateLightSessionOrgSlug("new-org", ORIGIN);
    expect(window.localStorage.getItem(sessionStorageKey(ORIGIN))).toBeNull();
  });
});
