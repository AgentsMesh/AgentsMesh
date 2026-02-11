"use client";

import { useTranslations } from "next-intl";
import { DocNavigation } from "@/components/docs/DocNavigation";

export default function MultiAgentWorkflowsPage() {
  const t = useTranslations();

  return (
    <div>
      <h1 className="text-4xl font-bold mb-8">
        {t("docs.guides.multiAgent.title")}
      </h1>

      <p className="text-muted-foreground leading-relaxed mb-8">
        {t("docs.guides.multiAgent.description")}
      </p>

      {/* Overview */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.guides.multiAgent.overview.title")}
        </h2>
        <p className="text-muted-foreground leading-relaxed">
          {t("docs.guides.multiAgent.overview.description")}
        </p>
      </section>

      {/* Setting Up a Multi-Agent Workflow */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.guides.multiAgent.setup.title")}
        </h2>
        <div className="border border-border rounded-lg p-6">
          <ol className="list-decimal list-inside text-muted-foreground space-y-3">
            <li>{t("docs.guides.multiAgent.setup.step1")}</li>
            <li>{t("docs.guides.multiAgent.setup.step2")}</li>
            <li>{t("docs.guides.multiAgent.setup.step3")}</li>
            <li>{t("docs.guides.multiAgent.setup.step4")}</li>
            <li>{t("docs.guides.multiAgent.setup.step5")}</li>
          </ol>
        </div>
      </section>

      {/* Scenario 1: Code Review Workflow */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.guides.multiAgent.scenario1.title")}
        </h2>
        <p className="text-muted-foreground leading-relaxed mb-4">
          {t("docs.guides.multiAgent.scenario1.description")}
        </p>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">
              {t("docs.guides.multiAgent.scenario1.developer").split(" — ")[0]}
            </h3>
            <p className="text-sm text-muted-foreground">
              {t("docs.guides.multiAgent.scenario1.developer").split(" — ")[1]}
            </p>
          </div>
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">
              {t("docs.guides.multiAgent.scenario1.reviewer").split(" — ")[0]}
            </h3>
            <p className="text-sm text-muted-foreground">
              {t("docs.guides.multiAgent.scenario1.reviewer").split(" — ")[1]}
            </p>
          </div>
        </div>
        <div className="bg-muted rounded-lg p-4">
          <p className="text-sm text-muted-foreground">
            {t("docs.guides.multiAgent.scenario1.flow")}
          </p>
        </div>
      </section>

      {/* Scenario 2: Frontend + Backend Split */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.guides.multiAgent.scenario2.title")}
        </h2>
        <p className="text-muted-foreground leading-relaxed mb-4">
          {t("docs.guides.multiAgent.scenario2.description")}
        </p>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">
              {t("docs.guides.multiAgent.scenario2.frontend").split(" — ")[0]}
            </h3>
            <p className="text-sm text-muted-foreground">
              {t("docs.guides.multiAgent.scenario2.frontend").split(" — ")[1]}
            </p>
          </div>
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">
              {t("docs.guides.multiAgent.scenario2.backend").split(" — ")[0]}
            </h3>
            <p className="text-sm text-muted-foreground">
              {t("docs.guides.multiAgent.scenario2.backend").split(" — ")[1]}
            </p>
          </div>
        </div>
        <div className="bg-muted rounded-lg p-4">
          <p className="text-sm text-muted-foreground">
            {t("docs.guides.multiAgent.scenario2.coordination")}
          </p>
        </div>
      </section>

      {/* Scenario 3: Test & Development in Parallel */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.guides.multiAgent.scenario3.title")}
        </h2>
        <p className="text-muted-foreground leading-relaxed mb-4">
          {t("docs.guides.multiAgent.scenario3.description")}
        </p>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">
              {t("docs.guides.multiAgent.scenario3.devPod").split(" — ")[0]}
            </h3>
            <p className="text-sm text-muted-foreground">
              {t("docs.guides.multiAgent.scenario3.devPod").split(" — ")[1]}
            </p>
          </div>
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">
              {t("docs.guides.multiAgent.scenario3.testPod").split(" — ")[0]}
            </h3>
            <p className="text-sm text-muted-foreground">
              {t("docs.guides.multiAgent.scenario3.testPod").split(" — ")[1]}
            </p>
          </div>
        </div>
        <div className="bg-muted rounded-lg p-4">
          <p className="text-sm text-muted-foreground">
            {t("docs.guides.multiAgent.scenario3.process")}
          </p>
        </div>
      </section>

      {/* Best Practices */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.guides.multiAgent.bestPractices.title")}
        </h2>
        <div className="space-y-3">
          <div className="border border-border rounded-lg p-4">
            <p className="text-sm text-muted-foreground">
              {t("docs.guides.multiAgent.bestPractices.tip1")}
            </p>
          </div>
          <div className="border border-border rounded-lg p-4">
            <p className="text-sm text-muted-foreground">
              {t("docs.guides.multiAgent.bestPractices.tip2")}
            </p>
          </div>
          <div className="border border-border rounded-lg p-4">
            <p className="text-sm text-muted-foreground">
              {t("docs.guides.multiAgent.bestPractices.tip3")}
            </p>
          </div>
          <div className="border border-border rounded-lg p-4">
            <p className="text-sm text-muted-foreground">
              {t("docs.guides.multiAgent.bestPractices.tip4")}
            </p>
          </div>
          <div className="border border-border rounded-lg p-4">
            <p className="text-sm text-muted-foreground">
              {t("docs.guides.multiAgent.bestPractices.tip5")}
            </p>
          </div>
        </div>
      </section>

      <DocNavigation />
    </div>
  );
}
