import { useState } from "react";
import { useTranslations } from "next-intl";
import { HeroPanel } from "./HeroPanel";
import { ServerSettingsModal } from "./ServerSettingsModal";
// Relative path required: `@/` resolves to clients/web/src via electron-vite alias,
// not the desktop renderer. server-config is desktop-only state.
import { getConfig, getActiveLabel } from "../../../lib/server-config";

export function AuthShell({ children }: { children: React.ReactNode }) {
  const t = useTranslations();
  const [open, setOpen] = useState(false);
  const cfg = getConfig();
  const activeLabel = getActiveLabel(cfg);

  return (
    <div className="flex min-h-screen bg-background">
      <div className="relative hidden flex-1 lg:flex lg:p-12">
        <HeroPanel />
        <div className="absolute bottom-6 left-6 flex items-center gap-2 text-xs text-muted-foreground">
          <button
            type="button"
            onClick={() => setOpen(true)}
            className="inline-flex items-center gap-1.5 rounded-md px-2 py-1 hover:bg-muted/60 hover:text-foreground transition-colors"
            aria-label={t("auth.loginPage.serverSettings")}
          >
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor"
              strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden>
              <circle cx="12" cy="12" r="3" />
              <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z" />
            </svg>
            <span>{t("auth.loginPage.serverSettings")}</span>
            <span className="text-muted-foreground/70">· {activeLabel}</span>
          </button>
        </div>
      </div>
      <div className="flex w-full items-center justify-center p-6 lg:w-[480px] lg:border-l lg:border-border">
        {children}
      </div>
      <ServerSettingsModal open={open} onOpenChange={setOpen} />
    </div>
  );
}
