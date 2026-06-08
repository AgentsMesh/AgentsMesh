import { useTranslations } from "next-intl";
import { RefreshCw } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useUpdater } from "@updater";
import { type UpdaterState } from "../../../../shared/updater-reducer";

const STATUS_KEY: Record<UpdaterState, string> = {
  idle: "updater.upToDate",
  checking: "updater.checking",
  downloading: "updater.downloading",
  ready: "updater.ready",
  error: "updater.error",
};

export function AppUpdateSection() {
  const t = useTranslations("settings");
  const { state, percent, currentVersion, check, quitAndInstall } = useUpdater();

  const statusText =
    state === "downloading"
      ? t("updater.downloading", { percent })
      : t(STATUS_KEY[state]);

  const busy = state === "checking" || state === "downloading";

  return (
    <div className="border border-border rounded-lg p-6">
      <h2 className="text-lg font-semibold mb-4">{t("updater.sectionTitle")}</h2>
      <div className="flex items-center justify-between gap-4">
        <div>
          <p className="text-sm text-muted-foreground">{t("updater.currentVersion")}</p>
          <p className="font-medium">{currentVersion ?? "-"}</p>
          <p className="text-sm text-muted-foreground mt-1">{statusText}</p>
        </div>
        {state === "ready" ? (
          <Button onClick={quitAndInstall}>{t("updater.restart")}</Button>
        ) : (
          <Button
            variant="outline"
            onClick={check}
            disabled={busy}
            className="flex items-center gap-2"
          >
            <RefreshCw className={`w-4 h-4 ${state === "checking" ? "animate-spin" : ""}`} />
            {t("updater.checkButton")}
          </Button>
        )}
      </div>
    </div>
  );
}
