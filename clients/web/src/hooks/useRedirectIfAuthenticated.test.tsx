import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { useRedirectIfAuthenticated } from "./useRedirectIfAuthenticated";
import { writeLightSession, clearLightSession, resolveLightBaseUrl, updateLightSessionOrgSlug } from "@/lib/light-session";

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

function persistSession(orgSlug: string | null) {
  writeLightSession({
    accessToken: "tok",
    refreshToken: "ref",
    expiresAt: Math.floor(Date.now() / 1000) + 3600,
    currentOrgSlug: orgSlug,
    baseUrl: ORIGIN,
  });
}

describe("useRedirectIfAuthenticated", () => {
  let originalFetch: typeof fetch;

  beforeEach(() => {
    originalFetch = globalThis.fetch;
    window.localStorage.clear();
    mockReplace.mockReset();
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
    window.localStorage.clear();
  });

  it("does NOT redirect when session is null", async () => {
    clearLightSession(ORIGIN);
    const { result } = renderHook(() => useRedirectIfAuthenticated());

    // Give the effect a chance to fire
    await new Promise((r) => setTimeout(r, 0));

    expect(mockReplace).not.toHaveBeenCalled();
    expect(result.current.redirecting).toBe(false);
    expect(result.current.hydrated).toBe(true);
  });

  it("redirects to getDefaultRoute(slug) when authenticated + currentOrgSlug present", async () => {
    persistSession(null);
    updateLightSessionOrgSlug("my-org", ORIGIN);

    const { result } = renderHook(() => useRedirectIfAuthenticated());

    await waitFor(() => expect(mockReplace).toHaveBeenCalled());
    // getDefaultRoute returns `/<slug>/workspace` (desktop) or `/<slug>/channels` (mobile)
    expect(mockReplace.mock.calls[0][0]).toMatch(/^\/my-org\//);
    expect(result.current.redirecting).toBe(true);
  });

  it("authenticated + no currentOrgSlug → fetches /orgs and redirects to first org", async () => {
    persistSession(null);
    globalThis.fetch = vi.fn(async () =>
      new Response(JSON.stringify({
        organizations: [
          { id: 1, slug: "fetched-org", name: "Fetched" },
        ],
      }), { status: 200 }),
    ) as typeof fetch;

    renderHook(() => useRedirectIfAuthenticated());

    await waitFor(() => expect(mockReplace).toHaveBeenCalled());
    expect(mockReplace.mock.calls[0][0]).toMatch(/^\/fetched-org\//);
  });

  it("authenticated + fetch failure → redirects to /onboarding", async () => {
    persistSession(null);
    globalThis.fetch = vi.fn(async () =>
      new Response("err", { status: 500 }),
    ) as typeof fetch;

    renderHook(() => useRedirectIfAuthenticated());

    await waitFor(() => expect(mockReplace).toHaveBeenCalled());
    expect(mockReplace).toHaveBeenCalledWith("/onboarding");
  });

  it("authenticated + empty orgs list → redirects to /onboarding", async () => {
    persistSession(null);
    globalThis.fetch = vi.fn(async () =>
      new Response(JSON.stringify({ organizations: [] }), { status: 200 }),
    ) as typeof fetch;

    renderHook(() => useRedirectIfAuthenticated());

    await waitFor(() => expect(mockReplace).toHaveBeenCalled());
    expect(mockReplace).toHaveBeenCalledWith("/onboarding");
  });

  it("cancellation guard: unmount before fetch resolves → no redirect", async () => {
    persistSession(null);
    let resolveFetch: (r: Response) => void = () => {};
    globalThis.fetch = vi.fn(
      () => new Promise<Response>((r) => { resolveFetch = r; }),
    ) as typeof fetch;

    const { unmount } = renderHook(() => useRedirectIfAuthenticated());
    unmount();

    // Resolve fetch AFTER unmount — cancellation flag must suppress the replace.
    resolveFetch(new Response(JSON.stringify({ organizations: [{ id: 1, slug: "x", name: "X" }] }), { status: 200 }));
    await new Promise((r) => setTimeout(r, 10));

    expect(mockReplace).not.toHaveBeenCalled();
  });
});
