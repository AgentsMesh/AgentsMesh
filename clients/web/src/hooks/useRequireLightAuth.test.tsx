import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { useRequireLightAuth } from "./useRequireLightAuth";
import { writeLightSession, clearLightSession, resolveLightBaseUrl } from "@/lib/light-session";

const mockReplace = vi.fn();

vi.mock("next/navigation", () => ({
  useRouter: () => ({
    push: vi.fn(),
    replace: mockReplace,
    back: vi.fn(),
    forward: vi.fn(),
    prefetch: vi.fn(),
    refresh: vi.fn(),
  }),
}));

const ORIGIN = resolveLightBaseUrl();

function persistAuthenticated() {
  writeLightSession({
    accessToken: "tok",
    refreshToken: "ref",
    expiresAt: Math.floor(Date.now() / 1000) + 3600,
    currentOrgSlug: "org",
    baseUrl: ORIGIN,
  });
}

function persistExpired() {
  writeLightSession({
    accessToken: "tok",
    refreshToken: "ref",
    expiresAt: Math.floor(Date.now() / 1000) - 60,
    currentOrgSlug: "org",
    baseUrl: ORIGIN,
  });
}

describe("useRequireLightAuth", () => {
  beforeEach(() => {
    window.localStorage.clear();
    mockReplace.mockReset();
    window.history.replaceState({}, "", "/dashboard/foo?bar=baz");
  });

  afterEach(() => {
    window.localStorage.clear();
  });

  it("redirects to /login?redirect=<pathname+search> when session is null", async () => {
    clearLightSession(ORIGIN);
    const { result } = renderHook(() => useRequireLightAuth());

    await waitFor(() => expect(mockReplace).toHaveBeenCalled());
    const arg = mockReplace.mock.calls[0][0] as string;
    expect(arg).toBe(`/login?redirect=${encodeURIComponent("/dashboard/foo?bar=baz")}`);
    expect(result.current.authenticated).toBe(false);
  });

  it("redirects when session exists but isAuthenticated=false (expired)", async () => {
    persistExpired();
    renderHook(() => useRequireLightAuth());

    await waitFor(() => expect(mockReplace).toHaveBeenCalled());
    expect((mockReplace.mock.calls[0][0] as string).startsWith("/login")).toBe(true);
  });

  it("does NOT redirect when authenticated", async () => {
    persistAuthenticated();
    const { result } = renderHook(() => useRequireLightAuth());

    // Wait a tick for the effect to fire
    await new Promise((r) => setTimeout(r, 0));

    expect(mockReplace).not.toHaveBeenCalled();
    expect(result.current.authenticated).toBe(true);
    expect(result.current.hydrated).toBe(true);
  });

  it("redirects to bare /login when pathname is falsy ('/')", async () => {
    clearLightSession(ORIGIN);
    window.history.replaceState({}, "", "/");
    renderHook(() => useRequireLightAuth());

    await waitFor(() => expect(mockReplace).toHaveBeenCalled());
    const arg = mockReplace.mock.calls[0][0] as string;
    // safeRedirectPath("/") returns "/" since it starts with "/" and length=1.
    // loginUrlWithRedirect("/") → "/login?redirect=%2F"
    expect(arg).toBe("/login?redirect=%2F");
  });
});
