"use client";

import { useEffect, useState } from "react";
import { useAuthStore } from "@/stores/auth";
import { WasmProvider } from "@/providers/WasmProvider";
import { CenteredSpinner } from "@/components/ui/spinner";

export function AuthBootstrap({ children }: { children: React.ReactNode }) {
  return (
    <WasmProvider>
      <BootstrapGate>{children}</BootstrapGate>
    </WasmProvider>
  );
}

function BootstrapGate({ children }: { children: React.ReactNode }) {
  const [done, setDone] = useState(false);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        await useAuthStore.getState().bootstrap();
      } finally {
        if (cancelled) return;
        useAuthStore.getState().setHasHydrated(true);
        setDone(true);
      }
    })();
    return () => { cancelled = true; };
  }, []);

  if (!done) return <CenteredSpinner />;
  return <>{children}</>;
}
