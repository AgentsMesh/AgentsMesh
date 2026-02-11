"use client";

import { useTranslations } from "next-intl";
import { DocNavigation } from "@/components/docs/DocNavigation";

export default function FAQPage() {
  const t = useTranslations();

  return (
    <div>
      <h1 className="text-4xl font-bold mb-8">
        {t("docs.faq.title")}
      </h1>

      <p className="text-muted-foreground leading-relaxed mb-8">
        {t("docs.faq.description")}
      </p>

      {/* Runner Issues */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.faq.categories.runner")}
        </h2>
        <div className="space-y-3">
          <details className="border border-border rounded-lg p-4">
            <summary className="font-medium cursor-pointer">
              {t("docs.faq.items.runnerConnection.question")}
            </summary>
            <p className="text-sm text-muted-foreground mt-3">
              {t("docs.faq.items.runnerConnection.answer")}
            </p>
          </details>
          <details className="border border-border rounded-lg p-4">
            <summary className="font-medium cursor-pointer">
              {t("docs.faq.items.runnerMultiple.question")}
            </summary>
            <p className="text-sm text-muted-foreground mt-3">
              {t("docs.faq.items.runnerMultiple.answer")}
            </p>
          </details>
        </div>
      </section>

      {/* Pod Issues */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.faq.categories.pod")}
        </h2>
        <div className="space-y-3">
          <details className="border border-border rounded-lg p-4">
            <summary className="font-medium cursor-pointer">
              {t("docs.faq.items.podCreationFail.question")}
            </summary>
            <p className="text-sm text-muted-foreground mt-3">
              {t("docs.faq.items.podCreationFail.answer")}
            </p>
          </details>
          <details className="border border-border rounded-lg p-4">
            <summary className="font-medium cursor-pointer">
              {t("docs.faq.items.podStuck.question")}
            </summary>
            <p className="text-sm text-muted-foreground mt-3">
              {t("docs.faq.items.podStuck.answer")}
            </p>
          </details>
        </div>
      </section>

      {/* API Key Configuration */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.faq.categories.apiKey")}
        </h2>
        <div className="space-y-3">
          <details className="border border-border rounded-lg p-4">
            <summary className="font-medium cursor-pointer">
              {t("docs.faq.items.apiKeyFormat.question")}
            </summary>
            <p className="text-sm text-muted-foreground mt-3">
              {t("docs.faq.items.apiKeyFormat.answer")}
            </p>
          </details>
          <details className="border border-border rounded-lg p-4">
            <summary className="font-medium cursor-pointer">
              {t("docs.faq.items.apiKeyMultiple.question")}
            </summary>
            <p className="text-sm text-muted-foreground mt-3">
              {t("docs.faq.items.apiKeyMultiple.answer")}
            </p>
          </details>
        </div>
      </section>

      {/* Git Integration */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.faq.categories.git")}
        </h2>
        <div className="space-y-3">
          <details className="border border-border rounded-lg p-4">
            <summary className="font-medium cursor-pointer">
              {t("docs.faq.items.gitCloneFail.question")}
            </summary>
            <p className="text-sm text-muted-foreground mt-3">
              {t("docs.faq.items.gitCloneFail.answer")}
            </p>
          </details>
          <details className="border border-border rounded-lg p-4">
            <summary className="font-medium cursor-pointer">
              {t("docs.faq.items.gitWorktreeConflict.question")}
            </summary>
            <p className="text-sm text-muted-foreground mt-3">
              {t("docs.faq.items.gitWorktreeConflict.answer")}
            </p>
          </details>
        </div>
      </section>

      {/* Billing & Plans */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.faq.categories.billing")}
        </h2>
        <div className="space-y-3">
          <details className="border border-border rounded-lg p-4">
            <summary className="font-medium cursor-pointer">
              {t("docs.faq.items.billingBYOK.question")}
            </summary>
            <p className="text-sm text-muted-foreground mt-3">
              {t("docs.faq.items.billingBYOK.answer")}
            </p>
          </details>
          <details className="border border-border rounded-lg p-4">
            <summary className="font-medium cursor-pointer">
              {t("docs.faq.items.billingFree.question")}
            </summary>
            <p className="text-sm text-muted-foreground mt-3">
              {t("docs.faq.items.billingFree.answer")}
            </p>
          </details>
        </div>
      </section>

      <DocNavigation />
    </div>
  );
}
