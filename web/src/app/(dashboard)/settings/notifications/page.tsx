"use client";

import React from "react";
import { useTranslations } from "next-intl";
import { NotificationSettings } from "@/components/settings/NotificationSettings";

export default function PersonalNotificationsPage() {
  const t = useTranslations();

  return (
    <div className="p-6 max-w-2xl">
      <div className="mb-6">
        <h2 className="text-lg font-semibold">{t("settings.notifications.title")}</h2>
        <p className="text-sm text-muted-foreground mt-1">
          {t("settings.notifications.description")}
        </p>
      </div>

      <NotificationSettings />
    </div>
  );
}
