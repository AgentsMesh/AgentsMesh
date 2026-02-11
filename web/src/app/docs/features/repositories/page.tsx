"use client";

import { useTranslations } from "next-intl";
import { DocNavigation } from "@/components/docs/DocNavigation";

export default function RepositoriesPage() {
  const t = useTranslations();

  return (
    <div>
      <h1 className="text-4xl font-bold mb-8">
        {t("docs.features.repositories.title")}
      </h1>

      <p className="text-muted-foreground leading-relaxed mb-8">
        {t("docs.features.repositories.description")}
      </p>

      {/* Overview */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.features.repositories.overview.title")}
        </h2>
        <p className="text-muted-foreground leading-relaxed">
          {t("docs.features.repositories.overview.description")}
        </p>
      </section>

      {/* Supported Git Providers */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.features.repositories.providers.title")}
        </h2>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">
              {t("docs.features.repositories.providers.github")}
            </h3>
            <p className="text-sm text-muted-foreground">
              {t("docs.features.repositories.providers.githubDesc")}
            </p>
          </div>
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">
              {t("docs.features.repositories.providers.gitlab")}
            </h3>
            <p className="text-sm text-muted-foreground">
              {t("docs.features.repositories.providers.gitlabDesc")}
            </p>
          </div>
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">
              {t("docs.features.repositories.providers.gitee")}
            </h3>
            <p className="text-sm text-muted-foreground">
              {t("docs.features.repositories.providers.giteeDesc")}
            </p>
          </div>
        </div>
      </section>

      {/* Authentication Methods */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.features.repositories.auth.title")}
        </h2>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">
              {t("docs.features.repositories.auth.oauth")}
            </h3>
            <p className="text-sm text-muted-foreground">
              {t("docs.features.repositories.auth.oauthDesc")}
            </p>
          </div>
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">
              {t("docs.features.repositories.auth.pat")}
            </h3>
            <p className="text-sm text-muted-foreground">
              {t("docs.features.repositories.auth.patDesc")}
            </p>
          </div>
        </div>
      </section>

      {/* Repository & Pod Association */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.features.repositories.podAssociation.title")}
        </h2>
        <p className="text-muted-foreground leading-relaxed mb-4">
          {t("docs.features.repositories.podAssociation.description")}
        </p>
        <ol className="list-decimal list-inside text-muted-foreground space-y-2">
          <li>{t("docs.features.repositories.podAssociation.step1")}</li>
          <li>{t("docs.features.repositories.podAssociation.step2")}</li>
          <li>{t("docs.features.repositories.podAssociation.step3")}</li>
          <li>{t("docs.features.repositories.podAssociation.step4")}</li>
          <li>{t("docs.features.repositories.podAssociation.step5")}</li>
        </ol>
      </section>

      {/* Git Worktree Isolation */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.features.repositories.worktree.title")}
        </h2>
        <p className="text-muted-foreground leading-relaxed">
          {t("docs.features.repositories.worktree.description")}
        </p>
      </section>

      {/* Managing Repositories */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.features.repositories.management.title")}
        </h2>
        <div className="space-y-4">
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">
              {t("docs.features.repositories.management.addRepo")}
            </h3>
            <p className="text-sm text-muted-foreground">
              {t("docs.features.repositories.management.addRepoSteps")}
            </p>
          </div>
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">
              {t("docs.features.repositories.management.removeRepo")}
            </h3>
            <p className="text-sm text-muted-foreground">
              {t("docs.features.repositories.management.removeRepoSteps")}
            </p>
          </div>
        </div>
      </section>

      <DocNavigation />
    </div>
  );
}
