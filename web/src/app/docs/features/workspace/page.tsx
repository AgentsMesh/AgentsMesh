"use client";

import { useTranslations } from "next-intl";
import { DocNavigation } from "@/components/docs/DocNavigation";

export default function WorkspacePage() {
  const t = useTranslations();

  return (
    <div>
      <h1 className="text-4xl font-bold mb-8">
        {t("docs.features.workspace.title")}
      </h1>

      <p className="text-muted-foreground leading-relaxed mb-8">
        {t("docs.features.workspace.description")}
      </p>

      {/* Overview */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.features.workspace.overview.title")}
        </h2>
        <p className="text-muted-foreground leading-relaxed">
          {t("docs.features.workspace.overview.description")}
        </p>
      </section>

      {/* Multi-Terminal View */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.features.workspace.multiTerminal.title")}
        </h2>
        <p className="text-muted-foreground leading-relaxed mb-6">
          {t("docs.features.workspace.multiTerminal.description")}
        </p>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">
              {t("docs.features.workspace.multiTerminal.splitView")}
            </h3>
            <p className="text-sm text-muted-foreground">
              {t("docs.features.workspace.multiTerminal.splitViewDesc")}
            </p>
          </div>
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">
              {t("docs.features.workspace.multiTerminal.fullscreen")}
            </h3>
            <p className="text-sm text-muted-foreground">
              {t("docs.features.workspace.multiTerminal.fullscreenDesc")}
            </p>
          </div>
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">
              {t("docs.features.workspace.multiTerminal.tabbing")}
            </h3>
            <p className="text-sm text-muted-foreground">
              {t("docs.features.workspace.multiTerminal.tabbingDesc")}
            </p>
          </div>
        </div>
      </section>

      {/* Pod Management */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.features.workspace.podList.title")}
        </h2>
        <p className="text-muted-foreground leading-relaxed mb-6">
          {t("docs.features.workspace.podList.description")}
        </p>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">
              {t("docs.features.workspace.podList.statusFilters")}
            </h3>
            <p className="text-sm text-muted-foreground">
              {t("docs.features.workspace.podList.statusFiltersDesc")}
            </p>
          </div>
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">
              {t("docs.features.workspace.podList.quickActions")}
            </h3>
            <p className="text-sm text-muted-foreground">
              {t("docs.features.workspace.podList.quickActionsDesc")}
            </p>
          </div>
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">
              {t("docs.features.workspace.podList.details")}
            </h3>
            <p className="text-sm text-muted-foreground">
              {t("docs.features.workspace.podList.detailsDesc")}
            </p>
          </div>
        </div>
      </section>

      {/* Quick Create */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.features.workspace.quickCreate.title")}
        </h2>
        <p className="text-muted-foreground leading-relaxed mb-4">
          {t("docs.features.workspace.quickCreate.description")}
        </p>
        <div className="bg-muted rounded-lg p-4">
          <p className="text-sm text-muted-foreground">
            {t("docs.features.workspace.quickCreate.steps")}
          </p>
        </div>
      </section>

      {/* Real-Time Interaction */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.features.workspace.realTime.title")}
        </h2>
        <p className="text-muted-foreground leading-relaxed">
          {t("docs.features.workspace.realTime.description")}
        </p>
      </section>

      <DocNavigation />
    </div>
  );
}
