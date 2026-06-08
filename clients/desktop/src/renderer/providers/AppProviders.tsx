import React, { useEffect, useState } from "react";
// MUST import via `next-themes` alias — direct `./ThemeProvider` produces a separate
// bundle module, causing two ThemeContext instances + "useTheme must be used within ThemeProvider".
import { ThemeProvider } from "next-themes";
import { DesktopIntlProvider } from "./IntlProvider";
import { Toaster } from "sonner";
import { UpdaterProvider } from "@updater";
import { UpdateReminder } from "../components/UpdateReminder";
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
  // RealtimeProvider is mounted inside DashboardShell (and PopoutTerminalPage)
  // — both rely on `currentOrg` being known to filter events. Mounting it
  // here as well would cause double `subscribeAll` registration, with each
  // event dispatched twice (channel:message duplicates, etc).
  return (
    <ThemeProvider defaultTheme="system" attribute="class">
      <DesktopIntlProvider>
        <UpdaterProvider>
          <PlatformGate>
            {children}
          </PlatformGate>
          <Toaster richColors position="top-right" />
          <UpdateReminder />
        </UpdaterProvider>
      </DesktopIntlProvider>
    </ThemeProvider>
  );
}
