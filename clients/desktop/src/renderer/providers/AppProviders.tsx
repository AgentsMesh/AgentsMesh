import React, { useEffect, useState } from "react";
// MUST import via `next-themes` alias — direct `./ThemeProvider` produces a separate
// bundle module, causing two ThemeContext instances + "useTheme must be used within ThemeProvider".
import { ThemeProvider } from "next-themes";
import { DesktopIntlProvider } from "./IntlProvider";
import { RealtimeProvider } from "./RealtimeProvider";
import { Toaster } from "sonner";
import { ensurePlatformReady } from "@agentsmesh/service-runtime";
import { useAuthStore } from "@/stores/auth";

async function bootstrapAuth() {
  await ensurePlatformReady();
  await useAuthStore.getState().bootstrap();
  useAuthStore.getState().setHasHydrated(true);
}

function PlatformGate({ children }: { children: React.ReactNode }) {
  const [ready, setReady] = useState(false);

  useEffect(() => {
    bootstrapAuth().then(() => setReady(true));
  }, []);

  if (!ready) return null;
  return <>{children}</>;
}

export function AppProviders({ children }: { children: React.ReactNode }) {
  return (
    <ThemeProvider defaultTheme="system" attribute="class">
      <DesktopIntlProvider>
        <PlatformGate>
          <RealtimeProvider>
            {children}
          </RealtimeProvider>
        </PlatformGate>
        <Toaster richColors position="top-right" />
      </DesktopIntlProvider>
    </ThemeProvider>
  );
}
