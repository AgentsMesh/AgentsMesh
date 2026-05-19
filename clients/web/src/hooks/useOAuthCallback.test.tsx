import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { StrictMode } from "react";
import { useOAuthCallback } from "./useOAuthCallback";
import {
  sessionStorageKey,
  resolveLightBaseUrl,
} from "@/lib/light-session";

const mockRouterPush = vi.fn();

vi.mock("next/navigation", () => ({
  useRouter: () => ({
    push: mockRouterPush,
    replace: vi.fn(),
    back: vi.fn(),
    forward: vi.fn(),
    prefetch: vi.fn(),
    refresh: vi.fn(),
  }),
}));

function makeSearchParams(map: Record<string, string | undefined>) {
  return {
    get(key: string): string | null {
      const v = map[key];
      return v === undefined ? null : v;
    },
  };
}

const ORIGIN = resolveLightBaseUrl();

function mockOrgsFetch(orgs: { id: number; slug: string; name: string }[]) {
  const spy = vi.fn(async () =>
    new Response(JSON.stringify({ organizations: orgs }), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    }),
  );
  globalThis.fetch = spy as typeof fetch;
  return spy;
}

// Yields control so React effects + microtask fetch chain can run, while
// using fake timers (we manually advance the 1500ms redirect setTimeout).
async function flushAsyncEffects() {
  await act(async () => {
    await Promise.resolve();
    await Promise.resolve();
    await Promise.resolve();
  });
}

describe("useOAuthCallback", () => {
  let originalFetch: typeof fetch;
  let originalReplaceState: typeof window.history.replaceState;

  beforeEach(() => {
    vi.useFakeTimers({ toFake: ["setTimeout", "clearTimeout"] });
    originalFetch = globalThis.fetch;
    originalReplaceState = window.history.replaceState.bind(window.history);
    window.localStorage.clear();
    mockRouterPush.mockReset();
    window.history.replaceState({}, "", "/auth/callback?token=t&refresh_token=r");
  });

  afterEach(() => {
    vi.useRealTimers();
    globalThis.fetch = originalFetch;
    window.history.replaceState = originalReplaceState;
    window.localStorage.clear();
  });

  it("processedRef gates single execution under strict-mode double mount", async () => {
    mockOrgsFetch([{ id: 1, slug: "dev-org", name: "Dev" }]);
    const params = makeSearchParams({ token: "abc", refresh_token: "xyz" });

    const { result } = renderHook(() => useOAuthCallback(params), { wrapper: StrictMode });

    await flushAsyncEffects();
    expect(result.current.status).toBe("success");

    // Redirect timer must still be in-flight (cleanup did NOT clear it
    // under strict-mode double effect — that's the load-bearing invariant).
    expect(mockRouterPush).not.toHaveBeenCalled();
    await act(async () => {
      vi.advanceTimersByTime(1600);
    });
    // Critical: fires exactly once (strict-mode double mount must not double-schedule).
    expect(mockRouterPush).toHaveBeenCalledTimes(1);
    expect(mockRouterPush).toHaveBeenCalledWith(expect.stringMatching(/^\/dev-org\//));
  });

  it("token + refresh_token → status=success, writes session, router.push fires after 1500ms", async () => {
    mockOrgsFetch([{ id: 1, slug: "alpha", name: "Alpha" }]);
    const params = makeSearchParams({ token: "T", refresh_token: "R" });

    const { result } = renderHook(() => useOAuthCallback(params));

    await flushAsyncEffects();
    expect(result.current.status).toBe("success");

    const raw = window.localStorage.getItem(sessionStorageKey(ORIGIN));
    expect(raw).not.toBeNull();
    const blob = JSON.parse(raw!);
    expect(blob.access_token).toBe("T");
    expect(blob.refresh_token).toBe("R");

    expect(mockRouterPush).not.toHaveBeenCalled();
    await act(async () => {
      vi.advanceTimersByTime(1499);
    });
    expect(mockRouterPush).not.toHaveBeenCalled();
    await act(async () => {
      vi.advanceTimersByTime(2);
    });
    expect(mockRouterPush).toHaveBeenCalledTimes(1);
  });

  it("error param surfaces as status=error + errorReason", async () => {
    const params = makeSearchParams({ error: "access_denied" });
    const { result } = renderHook(() => useOAuthCallback(params));

    await flushAsyncEffects();
    expect(result.current.status).toBe("error");
    expect(result.current.errorReason).toBe("access_denied");
    expect(mockRouterPush).not.toHaveBeenCalled();
  });

  it("no token + no error → status=error + reason=missing_token", async () => {
    const params = makeSearchParams({});
    const { result } = renderHook(() => useOAuthCallback(params));

    await flushAsyncEffects();
    expect(result.current.status).toBe("error");
    expect(result.current.errorReason).toBe("missing_token");
  });

  it("redirect param is forwarded to resolvePostLoginUrlLight (safe path used directly)", async () => {
    globalThis.fetch = vi.fn(async () => new Response("{}", { status: 200 })) as typeof fetch;
    const params = makeSearchParams({
      token: "T",
      refresh_token: "R",
      redirect: "/back/here",
    });

    const { result } = renderHook(() => useOAuthCallback(params));
    await flushAsyncEffects();
    expect(result.current.status).toBe("success");

    await act(async () => {
      vi.advanceTimersByTime(1600);
    });
    expect(mockRouterPush).toHaveBeenCalledWith("/back/here");
  });

  it("strips sensitive params from URL via history.replaceState", async () => {
    mockOrgsFetch([]);
    const replaceSpy = vi.spyOn(window.history, "replaceState");
    const params = makeSearchParams({ token: "T", refresh_token: "R" });

    renderHook(() => useOAuthCallback(params));
    expect(replaceSpy).toHaveBeenCalledWith({}, "", window.location.pathname);
    await flushAsyncEffects();
    replaceSpy.mockRestore();
  });

  it("does NOT call replaceState when no sensitive params present", async () => {
    const replaceSpy = vi.spyOn(window.history, "replaceState");
    const params = makeSearchParams({ redirect: "/foo" });

    renderHook(() => useOAuthCallback(params));
    expect(replaceSpy).not.toHaveBeenCalled();
    await flushAsyncEffects();
    replaceSpy.mockRestore();
  });

  it("falls back to /onboarding when no orgs and no redirect", async () => {
    mockOrgsFetch([]);
    const params = makeSearchParams({ token: "T", refresh_token: "R" });
    const { result } = renderHook(() => useOAuthCallback(params));

    await flushAsyncEffects();
    expect(result.current.status).toBe("success");

    await act(async () => {
      vi.advanceTimersByTime(1600);
    });
    expect(mockRouterPush).toHaveBeenCalledWith("/onboarding");
  });
});
