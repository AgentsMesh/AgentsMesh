"use client";

import Link from "next/link";
import { useTranslations } from "next-intl";
import { ChevronLeft } from "lucide-react";
import { GitSettingsContent } from "@/components/settings";

/**
 * GitSettingsPage - User settings page for Git configuration
 *
 * This page wraps the shared GitSettingsContent component with
 * a page header including navigation back to settings.
 */
export default function GitSettingsPage() {
  const t = useTranslations();

  return (
    <div className="p-6 max-w-4xl mx-auto">
      {/* Page Header */}
      <div className="mb-6">
        <Link
          href="/settings"
          className="flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground mb-4"
        >
          <ChevronLeft className="w-4 h-4" />
          {t("settings.backToSettings")}
        </Link>
        <h1 className="text-2xl font-bold">{t("settings.gitSettings.title")}</h1>
        <p className="text-muted-foreground">
          {t("settings.gitSettings.description")}
        </p>
      </div>

      {/* Git Settings Content */}
      <GitSettingsContent />
    </div>
  );
}
