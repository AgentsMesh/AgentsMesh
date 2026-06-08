import { useTranslations } from "next-intl";
import { useUpdater } from "@updater";

// Shown only once an update is staged; clicking restarts to apply it.
export function UpdateReadyBanner() {
  const t = useTranslations("settings");
  const { state, availableVersion, quitAndInstall } = useUpdater();

  if (state !== "ready") return null;

  return (
    <button
      type="button"
      onClick={quitAndInstall}
      data-testid="update-ready-banner"
      className="fixed inset-x-0 top-0 z-[60] flex items-center justify-center gap-2 bg-emerald-600/95 px-4 py-1.5 text-center text-sm font-medium text-white shadow transition-colors hover:bg-emerald-600"
    >
      <span className="inline-block h-2 w-2 animate-pulse rounded-full bg-white/90" />
      {t("updater.bannerReady", { version: availableVersion ?? "" })}
    </button>
  );
}
