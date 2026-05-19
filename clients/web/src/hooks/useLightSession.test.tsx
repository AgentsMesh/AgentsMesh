import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useLightSession } from "./useLightSession";
import { writeLightSession, clearLightSession, sessionStorageKey, resolveLightBaseUrl } from "@/lib/light-session";

const ORIGIN = resolveLightBaseUrl();

describe("useLightSession", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  afterEach(() => {
    window.localStorage.clear();
  });

  it("returns null session when localStorage is empty", () => {
    const { result } = renderHook(() => useLightSession());
    expect(result.current.session).toBeNull();
    expect(result.current.hydrated).toBe(true);
  });

  it("returns LightSession derived from the persisted blob", () => {
    writeLightSession({
      accessToken: "tok",
      refreshToken: "ref",
      expiresAt: Math.floor(Date.now() / 1000) + 3600,
      currentOrgSlug: "alpha",
      baseUrl: ORIGIN,
    });
    const { result } = renderHook(() => useLightSession());
    expect(result.current.session).not.toBeNull();
    expect(result.current.session?.isAuthenticated).toBe(true);
    expect(result.current.session?.currentOrgSlug).toBe("alpha");
  });

  it("re-reads on `storage` event from another tab", () => {
    clearLightSession(ORIGIN);
    const { result } = renderHook(() => useLightSession());
    expect(result.current.session).toBeNull();

    act(() => {
      writeLightSession({
        accessToken: "tok",
        refreshToken: "ref",
        expiresAt: Math.floor(Date.now() / 1000) + 3600,
        currentOrgSlug: "beta",
        baseUrl: ORIGIN,
      });
      // Simulate the cross-tab storage event the hook listens for
      window.dispatchEvent(new StorageEvent("storage", {
        key: sessionStorageKey(ORIGIN),
        newValue: window.localStorage.getItem(sessionStorageKey(ORIGIN)),
      }));
    });

    expect(result.current.session?.currentOrgSlug).toBe("beta");
    expect(result.current.session?.isAuthenticated).toBe(true);
  });

  it("ignores storage events with unrelated keys", () => {
    writeLightSession({
      accessToken: "tok",
      refreshToken: "ref",
      expiresAt: Math.floor(Date.now() / 1000) + 3600,
      currentOrgSlug: "alpha",
      baseUrl: ORIGIN,
    });
    const { result } = renderHook(() => useLightSession());
    const before = result.current.session;

    act(() => {
      window.dispatchEvent(new StorageEvent("storage", {
        key: "unrelated/key",
        newValue: "x",
      }));
    });

    // Reference may change but content must remain equivalent
    expect(result.current.session?.currentOrgSlug).toBe(before?.currentOrgSlug);
    expect(result.current.session?.isAuthenticated).toBe(true);
  });

  it("detects logout via storage event clearing the blob", () => {
    writeLightSession({
      accessToken: "tok",
      refreshToken: "ref",
      expiresAt: Math.floor(Date.now() / 1000) + 3600,
      currentOrgSlug: "alpha",
      baseUrl: ORIGIN,
    });
    const { result } = renderHook(() => useLightSession());
    expect(result.current.session).not.toBeNull();

    act(() => {
      clearLightSession(ORIGIN);
      window.dispatchEvent(new StorageEvent("storage", {
        key: sessionStorageKey(ORIGIN),
        newValue: null,
      }));
    });

    expect(result.current.session).toBeNull();
  });
});
