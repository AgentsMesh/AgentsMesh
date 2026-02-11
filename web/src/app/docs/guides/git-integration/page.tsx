"use client";

import { useTranslations } from "next-intl";
import { DocNavigation } from "@/components/docs/DocNavigation";

export default function GitIntegrationPage() {
  const t = useTranslations();

  return (
    <div>
      <h1 className="text-4xl font-bold mb-8">
        {t("docs.guides.gitIntegration.title")}
      </h1>

      <p className="text-muted-foreground leading-relaxed mb-8">
        {t("docs.guides.gitIntegration.description")}
      </p>

      {/* Overview */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.guides.gitIntegration.overview.title")}
        </h2>
        <p className="text-muted-foreground leading-relaxed">
          {t("docs.guides.gitIntegration.overview.description")}
        </p>
      </section>

      {/* GitHub */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.guides.gitIntegration.github.title")}
        </h2>
        <p className="text-muted-foreground leading-relaxed mb-4">
          {t("docs.guides.gitIntegration.github.description")}
        </p>
        <div className="border border-border rounded-lg p-6">
          <ol className="list-decimal list-inside text-muted-foreground space-y-3">
            <li>{t("docs.guides.gitIntegration.github.step1")}</li>
            <li>{t("docs.guides.gitIntegration.github.step2")}</li>
            <li>{t("docs.guides.gitIntegration.github.step3")}</li>
            <li>{t("docs.guides.gitIntegration.github.step4")}</li>
            <li>{t("docs.guides.gitIntegration.github.step5")}</li>
          </ol>
        </div>
      </section>

      {/* GitLab */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.guides.gitIntegration.gitlab.title")}
        </h2>
        <p className="text-muted-foreground leading-relaxed mb-4">
          {t("docs.guides.gitIntegration.gitlab.description")}
        </p>
        <div className="border border-border rounded-lg p-6">
          <ol className="list-decimal list-inside text-muted-foreground space-y-3">
            <li>{t("docs.guides.gitIntegration.gitlab.step1")}</li>
            <li>{t("docs.guides.gitIntegration.gitlab.step2")}</li>
            <li>{t("docs.guides.gitIntegration.gitlab.step3")}</li>
            <li>{t("docs.guides.gitIntegration.gitlab.step4")}</li>
            <li>{t("docs.guides.gitIntegration.gitlab.step5")}</li>
          </ol>
          <div className="bg-muted rounded-lg p-4 mt-4">
            <p className="text-sm text-muted-foreground">
              {t("docs.guides.gitIntegration.gitlab.selfHosted")}
            </p>
          </div>
        </div>
      </section>

      {/* Gitee */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.guides.gitIntegration.gitee.title")}
        </h2>
        <p className="text-muted-foreground leading-relaxed mb-4">
          {t("docs.guides.gitIntegration.gitee.description")}
        </p>
        <div className="border border-border rounded-lg p-6">
          <ol className="list-decimal list-inside text-muted-foreground space-y-3">
            <li>{t("docs.guides.gitIntegration.gitee.step1")}</li>
            <li>{t("docs.guides.gitIntegration.gitee.step2")}</li>
            <li>{t("docs.guides.gitIntegration.gitee.step3")}</li>
            <li>{t("docs.guides.gitIntegration.gitee.step4")}</li>
          </ol>
        </div>
      </section>

      {/* SSH Keys */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.guides.gitIntegration.sshKeys.title")}
        </h2>
        <p className="text-muted-foreground leading-relaxed mb-4">
          {t("docs.guides.gitIntegration.sshKeys.description")}
        </p>
        <div className="border border-border rounded-lg p-6">
          <ol className="list-decimal list-inside text-muted-foreground space-y-3">
            <li>{t("docs.guides.gitIntegration.sshKeys.step1")}</li>
            <li>{t("docs.guides.gitIntegration.sshKeys.step2")}</li>
            <li>{t("docs.guides.gitIntegration.sshKeys.step3")}</li>
          </ol>
          <div className="bg-muted rounded-lg p-4 font-mono text-sm overflow-x-auto mt-4">
            <p className="text-sm text-muted-foreground font-sans">
              {t("docs.guides.gitIntegration.sshKeys.dockerNote")}
            </p>
          </div>
        </div>
      </section>

      {/* Using Repositories in Pods */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.guides.gitIntegration.podUsage.title")}
        </h2>
        <p className="text-muted-foreground leading-relaxed mb-4">
          {t("docs.guides.gitIntegration.podUsage.description")}
        </p>
        <div className="border border-border rounded-lg p-6">
          <ol className="list-decimal list-inside text-muted-foreground space-y-3">
            <li>{t("docs.guides.gitIntegration.podUsage.step1")}</li>
            <li>{t("docs.guides.gitIntegration.podUsage.step2")}</li>
            <li>{t("docs.guides.gitIntegration.podUsage.step3")}</li>
            <li>{t("docs.guides.gitIntegration.podUsage.step4")}</li>
          </ol>
        </div>
      </section>

      <DocNavigation />
    </div>
  );
}
