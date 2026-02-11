"use client";

import { useTranslations } from "next-intl";
import { DocNavigation } from "@/components/docs/DocNavigation";

export default function ConceptsPage() {
  const t = useTranslations();

  return (
    <div>
      <h1 className="text-4xl font-bold mb-8">{t("docs.concepts.title")}</h1>

      <p className="text-muted-foreground leading-relaxed mb-8">
        {t("docs.concepts.description")}
      </p>

      {/* Architecture Overview */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.concepts.architecture.title")}
        </h2>
        <p className="text-muted-foreground leading-relaxed mb-6">
          {t("docs.concepts.architecture.description")}
        </p>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">
              {t("docs.concepts.architecture.orgTitle")}
            </h3>
            <p className="text-sm text-muted-foreground">
              {t("docs.concepts.architecture.orgDesc")}
            </p>
          </div>
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">
              {t("docs.concepts.architecture.runnerTitle")}
            </h3>
            <p className="text-sm text-muted-foreground">
              {t("docs.concepts.architecture.runnerDesc")}
            </p>
          </div>
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">
              {t("docs.concepts.architecture.podTitle")}
            </h3>
            <p className="text-sm text-muted-foreground">
              {t("docs.concepts.architecture.podDesc")}
            </p>
          </div>
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">
              {t("docs.concepts.architecture.channelTitle")}
            </h3>
            <p className="text-sm text-muted-foreground">
              {t("docs.concepts.architecture.channelDesc")}
            </p>
          </div>
          <div className="border border-border rounded-lg p-4 md:col-span-2">
            <h3 className="font-medium mb-2">
              {t("docs.concepts.architecture.ticketTitle")}
            </h3>
            <p className="text-sm text-muted-foreground">
              {t("docs.concepts.architecture.ticketDesc")}
            </p>
          </div>
        </div>
      </section>

      {/* BYOK */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.concepts.byok.title")}
        </h2>
        <p className="text-muted-foreground leading-relaxed">
          {t("docs.concepts.byok.description")}
        </p>
      </section>

      {/* mTLS Security */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.concepts.mtls.title")}
        </h2>
        <p className="text-muted-foreground leading-relaxed">
          {t("docs.concepts.mtls.description")}
        </p>
      </section>

      {/* Sandbox & Git Worktree Isolation */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.concepts.sandbox.title")}
        </h2>
        <p className="text-muted-foreground leading-relaxed">
          {t("docs.concepts.sandbox.description")}
        </p>
      </section>

      {/* MCP */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.concepts.mcp.title")}
        </h2>
        <p className="text-muted-foreground leading-relaxed">
          {t("docs.concepts.mcp.description")}
        </p>
      </section>

      <DocNavigation />
    </div>
  );
}
