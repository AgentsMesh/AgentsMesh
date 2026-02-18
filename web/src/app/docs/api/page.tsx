"use client";

import { useTranslations } from "next-intl";
import { DocNavigation } from "@/components/docs/DocNavigation";

export default function ApiOverviewPage() {
  const t = useTranslations();

  return (
    <div>
      <h1 className="text-4xl font-bold mb-8">
        {t("docs.api.overview.title")}
      </h1>

      <p className="text-muted-foreground leading-relaxed mb-8">
        {t("docs.api.overview.description")}
      </p>

      {/* Overview */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.api.overview.overview.title")}
        </h2>
        <p className="text-muted-foreground leading-relaxed mb-4">
          {t("docs.api.overview.overview.description")}
        </p>
        <ul className="list-disc list-inside text-muted-foreground space-y-2">
          <li>{t("docs.api.overview.overview.item1")}</li>
          <li>{t("docs.api.overview.overview.item2")}</li>
          <li>{t("docs.api.overview.overview.item3")}</li>
          <li>{t("docs.api.overview.overview.item4")}</li>
        </ul>
      </section>

      {/* Getting an API Key */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.api.overview.gettingKey.title")}
        </h2>
        <p className="text-muted-foreground leading-relaxed mb-4">
          {t("docs.api.overview.gettingKey.description")}
        </p>
        <ol className="list-decimal list-inside text-muted-foreground space-y-2 mb-4">
          <li>{t("docs.api.overview.gettingKey.step1")}</li>
          <li>{t("docs.api.overview.gettingKey.step2")}</li>
          <li>{t("docs.api.overview.gettingKey.step3")}</li>
          <li>{t("docs.api.overview.gettingKey.step4")}</li>
        </ol>
        <div className="border border-yellow-500/50 bg-yellow-500/10 rounded-lg p-4">
          <p className="text-sm text-yellow-700 dark:text-yellow-400">
            {t("docs.api.overview.gettingKey.warning")}
          </p>
        </div>
      </section>

      {/* Base URL & Path */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.api.overview.basePath.title")}
        </h2>
        <p className="text-muted-foreground leading-relaxed mb-4">
          {t("docs.api.overview.basePath.description")}
        </p>
        <div className="bg-muted rounded-lg p-4 font-mono text-sm mb-4">
          <pre className="text-green-500 dark:text-green-400">{`{BASE_URL}/api/v1/ext/orgs/{slug}/...`}</pre>
        </div>
        <p className="text-sm text-muted-foreground">
          {t("docs.api.overview.basePath.slugNote")}
        </p>
      </section>

      <DocNavigation />
    </div>
  );
}
