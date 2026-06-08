import { useEffect } from "react";
import { useTranslations } from "next-intl";
import { useUpdater } from "@updater";
import { useReminderStore } from "@/stores/reminders";

// Headless registrar: mirrors the desktop-only updater state into the shared
// reminder store so it surfaces in the ActivityBar reminder area (web has no
// updater, so this lives in the desktop provider tree, not in @/components).
export function UpdateReminder() {
  const { state, availableVersion, quitAndInstall } = useUpdater();
  const setReminder = useReminderStore((s) => s.setReminder);
  const clearReminder = useReminderStore((s) => s.clearReminder);
  const t = useTranslations("settings");

  useEffect(() => {
    if (state !== "ready") {
      clearReminder("update-ready");
      return;
    }
    setReminder(
      {
        id: "update-ready",
        tone: "success",
        message: t("updater.bannerReady", { version: availableVersion ?? "" }),
        onAction: quitAndInstall,
      },
      availableVersion ?? "",
    );
  }, [state, availableVersion, quitAndInstall, setReminder, clearReminder, t]);

  return null;
}

export default UpdateReminder;
