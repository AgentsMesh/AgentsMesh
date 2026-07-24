"use client";

import { useEffect, useState } from "react";
import { initWasmCore } from "@/lib/wasm-core";

function isEmbedRoute(): boolean {
  if (typeof window === "undefined") return false;
  return window.location.pathname.startsWith("/blocks-embed");
}

// Loads the wasm core. Does NOT touch auth state — that's AuthBootstrap's
// job (see components/auth/AuthBootstrap.tsx). Embed routes (iOS WKWebView)
// skip wasm because the embed page wires its own platform init via the
// ios-bridge RPC; running WASM there would spin up a second Rust core.
export function WasmProvider({ children }: { children: React.ReactNode }) {
  const [ready, setReady] = useState(false);

  useEffect(() => {
    if (isEmbedRoute()) {
      Promise.resolve().then(() => setReady(true));
      return;
    }
    initWasmCore().then(async () => {
      if (process.env.NEXT_PUBLIC_E2E === "true") {
        const { installRelayReadinessProbe } = await import("@/lib/e2e/relayReadinessProbe");
        installRelayReadinessProbe(window);
      }
      setReady(true);
    });
  }, []);

  if (!ready) return null;
  return <>{children}</>;
}
