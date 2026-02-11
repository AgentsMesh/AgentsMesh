"use client";

import { useTranslations } from "next-intl";
import { LanguageSwitcher } from "@/components/i18n";

export function LanguageSettings() {
  const t = useTranslations();

  return (
    <div className="border border-border rounded-lg p-6">
      <h2 className="text-lg font-semibold mb-4">{t("settings.language.title")}</h2>
      <p className="text-sm text-muted-foreground mb-4">
        {t("settings.language.description")}
      </p>
      <div className="flex items-center justify-between">
        <div>
          <label className="block text-sm font-medium mb-1">
            {t("settings.language.selectLabel")}
          </label>
          <p className="text-xs text-muted-foreground">
            {t("settings.language.selectHint")}
          </p>
        </div>
        <LanguageSwitcher variant="select" className="w-40" />
      </div>
    </div>
  );
}
