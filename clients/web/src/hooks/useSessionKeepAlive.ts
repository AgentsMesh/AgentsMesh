import { useEffect } from "react";
import { useAuthStore } from "@/stores/auth";

// Access tokens expire (backend JWT TTL is 24h) and nothing rotates them while
// the app stays open, so once the TTL elapses the session reads as expired and
// the dashboard blanks. Proactively refresh: on regaining foreground (covers
// the OS-sleep / window-switch case, the dominant 24h trigger) and on a coarse
// interval for an always-foreground window.
const REFRESH_INTERVAL_MS = 30 * 60 * 1000;

export function useSessionKeepAlive(): void {
  const refreshSession = useAuthStore((s) => s.refreshSession);

  useEffect(() => {
    let disposed = false;
    // Transient failures are swallowed — the next focus/interval retries. A
    // hard failure (refresh token also expired) flips is_authenticated to
    // false, which DashboardShell turns into a redirect to /login.
    const refresh = () => {
      if (!disposed) void refreshSession().catch(() => {});
    };
    const onVisible = () => {
      if (document.visibilityState === "visible") refresh();
    };

    document.addEventListener("visibilitychange", onVisible);
    window.addEventListener("focus", onVisible);
    const timer = window.setInterval(refresh, REFRESH_INTERVAL_MS);

    return () => {
      disposed = true;
      document.removeEventListener("visibilitychange", onVisible);
      window.removeEventListener("focus", onVisible);
      window.clearInterval(timer);
    };
  }, [refreshSession]);
}
