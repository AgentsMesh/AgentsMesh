import React from "react";
import ReactDOM from "react-dom/client";
import { RouterProvider } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { AppProviders } from "./providers/AppProviders";
import { router } from "./router";
import "./lib/platform-init";
import "./globals.css";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 1000 * 60,
      retry: 1,
    },
  },
});

// Dev-only: detect store update bursts (>30/s) — fingerprint of React #185
// render loops. Lazy-imported so production bundles don't pull in the
// subscribe wiring or pin extra stores into the entry chunk.
if (import.meta.env.DEV) {
  import("@/lib/debug/storeBurstDetector")
    .then(({ installStoreBurstDetector }) => installStoreBurstDetector(30))
    .catch((err) => console.warn("[storeBurst] install failed:", err));
}

// Deep-link OAuth callback. Main process forwards `agentsmesh://oauth/callback?token=...`
// after the system browser completes GitHub/Google OAuth and the backend 302-redirects
// to our custom scheme. We translate the deep-link URL into an in-app navigation to
// `/auth/callback?token=...`, which is the existing OAuthCallbackPage that already knows
// how to capture token + refresh_token from query params and finish login.
//
// `router.navigate` is the imperative API on the createHashRouter instance — works
// outside any component, before RouterProvider mounts, and is idempotent under React
// StrictMode double-render (the listener is registered once at module load).
if (typeof window !== "undefined" && window.electronAPI?.onOAuthCallback) {
  window.electronAPI.onOAuthCallback((deepLink) => {
    try {
      const u = new URL(deepLink);
      router.navigate(`/auth/callback${u.search}`);
    } catch (err) {
      console.error("[oauth] invalid deep link:", deepLink, err);
    }
  });
}

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <AppProviders>
        <RouterProvider router={router} />
      </AppProviders>
    </QueryClientProvider>
  </React.StrictMode>,
);
