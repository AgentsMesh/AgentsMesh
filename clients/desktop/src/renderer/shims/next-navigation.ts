import {
  useNavigate,
  useLocation,
  useSearchParams as useRRSearchParams,
  useParams as useRRParams,
} from "react-router-dom";
import { useMemo } from "react";

interface NavigationOptions {
  scroll?: boolean;
}

// Workaround Electron file:// production renderer bug: navigate(url) occasionally fails
// to mutate window.location.hash; force assignment so HashRouter sees the change.
function pushWithHashFallback(navigate: ReturnType<typeof useNavigate>) {
  return (url: string) => {
    navigate(url);
    if (typeof window !== "undefined") {
      const target = url.startsWith("#") ? url : `#${url.startsWith("/") ? url : `/${url}`}`;
      if (window.location.hash !== target) window.location.hash = target;
    }
  };
}

function replaceWithHashFallback(navigate: ReturnType<typeof useNavigate>) {
  return (url: string) => {
    navigate(url, { replace: true });
    if (typeof window !== "undefined") {
      const target = url.startsWith("#") ? url : `#${url.startsWith("/") ? url : `/${url}`}`;
      if (window.location.hash !== target) {
        const base = window.location.href.split("#")[0];
        window.history.replaceState(null, "", base + target);
        window.dispatchEvent(new HashChangeEvent("hashchange"));
      }
    }
  };
}

export function useRouter() {
  const navigate = useNavigate();

  return useMemo(
    () => ({
      push: (url: string, _options?: NavigationOptions) => pushWithHashFallback(navigate)(url),
      replace: (url: string, _options?: NavigationOptions) => replaceWithHashFallback(navigate)(url),
      back: () => navigate(-1),
      forward: () => navigate(1),
      refresh: () => navigate(0),
      prefetch: (_url: string) => {},
    }),
    [navigate],
  );
}

export function usePathname(): string {
  const location = useLocation();
  return location.pathname;
}

export function useSearchParams(): URLSearchParams {
  const [searchParams] = useRRSearchParams();
  return searchParams;
}

export function useParams<T extends Record<string, string> = Record<string, string>>(): T {
  return useRRParams() as T;
}

export { useLocation } from "react-router-dom";

export function redirect(url: string): never {
  window.location.href = url;
  throw new Error("redirect");
}

export function useSelectedLayoutSegment(): string | null {
  const location = useLocation();
  const segments = location.pathname.split("/").filter(Boolean);
  return segments[0] ?? null;
}
