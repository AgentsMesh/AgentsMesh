import React from "react";
import ReactDOM from "react-dom/client";
import { RouterProvider } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { AppProviders } from "./providers/AppProviders";
import { RootErrorBoundary } from "./components/RootErrorBoundary";
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

if (import.meta.env.DEV) {
  import("@/lib/debug/storeBurstDetector")
    .then(({ installStoreBurstDetector }) => installStoreBurstDetector(30))
    .catch((err) => console.warn("[storeBurst] install failed:", err));
}

// IPC: main forwards `agentsmesh://oauth/callback?...` deep link; router.navigate
// is idempotent under React StrictMode double-render (registered once at module load).
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
    <RootErrorBoundary>
      <QueryClientProvider client={queryClient}>
        <AppProviders>
          <RouterProvider router={router} />
        </AppProviders>
      </QueryClientProvider>
    </RootErrorBoundary>
  </React.StrictMode>,
);
